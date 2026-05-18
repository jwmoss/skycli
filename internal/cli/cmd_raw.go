package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
)

func runRaw(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("raw", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	method := fs.String("method", "GET", "HTTP method")
	body := fs.String("body", "", "request body (JSON string); use --body-file for stdin or file")
	bodyFile := fs.String("body-file", "", "path or - for stdin")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	rest := fs.Args()
	if len(rest) == 0 {
		return usage(rc, "skycli raw [--method M] [--body JSON] <path-or-url>")
	}
	path := rest[0]
	if !strings.HasPrefix(path, "/") && !isHTTPURL(path) {
		path = "/" + path
	}
	// Split path and query.
	var query url.Values
	if i := strings.IndexByte(path, '?'); i >= 0 {
		q, err := url.ParseQuery(path[i+1:])
		if err != nil {
			return fail(rc, err)
		}
		query = q
		path = path[:i]
	}
	var payload any
	switch {
	case *bodyFile != "":
		var rdr io.Reader
		if *bodyFile == "-" {
			rdr = rc.stdin
		} else {
			f, err := os.Open(*bodyFile)
			if err != nil {
				return fail(rc, err)
			}
			defer f.Close()
			rdr = f
		}
		raw, err := io.ReadAll(rdr)
		if err != nil {
			return fail(rc, err)
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			return fail(rc, fmt.Errorf("parse body JSON: %w", err))
		}
	case *body != "":
		if err := json.Unmarshal([]byte(*body), &payload); err != nil {
			return fail(rc, fmt.Errorf("parse --body JSON: %w", err))
		}
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	data, err := c.Do(rc.ctx, strings.ToUpper(*method), path, query, payload)
	if err != nil {
		return fail(rc, err)
	}
	// Pretty-print if JSON, else raw.
	var pretty any
	if json.Unmarshal(data, &pretty) == nil {
		_ = rc.out.JSON(pretty)
		return exitOK
	}
	fmt.Fprintln(rc.stdout, string(data))
	return exitOK
}

func isHTTPURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
