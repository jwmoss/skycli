package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/jwmoss/skycli/internal/skylight"
)

func printJSONBytes(rc *runCtx, data []byte) int {
	var pretty any
	if json.Unmarshal(data, &pretty) == nil {
		_ = rc.out.JSON(pretty)
		return exitOK
	}
	fmt.Fprintln(rc.stdout, string(data))
	return exitOK
}

func readPayload(rc *runCtx, body, bodyFile string) (map[string]any, error) {
	payload := map[string]any{}
	if body == "" && bodyFile == "" {
		return payload, nil
	}
	var raw []byte
	var err error
	if bodyFile != "" {
		var rdr io.Reader
		if bodyFile == "-" {
			rdr = rc.stdin
		} else {
			f, err := os.Open(bodyFile)
			if err != nil {
				return nil, err
			}
			defer f.Close()
			rdr = f
		}
		raw, err = io.ReadAll(rdr)
	} else {
		raw = []byte(body)
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("parse body JSON: %w", err)
	}
	return payload, nil
}

func runResourceJSON(rc *runCtx, call func(*skylight.Client) (any, error)) int {
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	data, err := call(c)
	if err != nil {
		return fail(rc, err)
	}
	if err := rc.out.JSON(data); err != nil {
		return fail(rc, err)
	}
	return exitOK
}

func runFrameResourceJSON(rc *runCtx, frameStr string, call func(*skylight.Client, int64) (any, error)) int {
	frameID, err := resolveFrame(rc, frameStr)
	if err != nil {
		return fail(rc, err)
	}
	return runResourceJSON(rc, func(c *skylight.Client) (any, error) {
		return call(c, frameID)
	})
}

func runFrameResourceOK(rc *runCtx, frameStr string, ok map[string]any, call func(*skylight.Client, int64) error) int {
	frameID, err := resolveFrame(rc, frameStr)
	if err != nil {
		return fail(rc, err)
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	if err := call(c, frameID); err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(ok)
	} else {
		rc.out.Line("ok")
	}
	return exitOK
}

func flagChanged(fs *flag.FlagSet, name string) bool {
	changed := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			changed = true
		}
	})
	return changed
}

func addStringIfSet(fs *flag.FlagSet, payload map[string]any, flagName, fieldName, value string) {
	if flagChanged(fs, flagName) {
		payload[fieldName] = value
	}
}

func addBoolIfSet(fs *flag.FlagSet, payload map[string]any, flagName, fieldName string, value bool) {
	if flagChanged(fs, flagName) {
		payload[fieldName] = value
	}
}

func addIntIfSet(fs *flag.FlagSet, payload map[string]any, flagName, fieldName string, value int) {
	if flagChanged(fs, flagName) {
		payload[fieldName] = value
	}
}

func parseCSVStrings(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseCSVInts(s, name string) ([]int, error) {
	parts := parseCSVStrings(s)
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		id, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("bad %s value %q: %w", name, p, err)
		}
		out = append(out, id)
	}
	return out, nil
}

func requireFlagValue(value, name string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("--%s is required", name)
	}
	return nil
}

func bodyFlags(fs *flag.FlagSet, rc *runCtx) (*string, *string) {
	body := fs.String("body", "", "raw JSON body to merge with flags")
	bodyFile := fs.String("body-file", "", "path or - for raw JSON body to merge with flags")
	return body, bodyFile
}

func formatID(id int64) string {
	return strconv.FormatInt(id, 10)
}
