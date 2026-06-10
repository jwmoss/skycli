package cli

import (
	"errors"
	"fmt"

	"github.com/jwmoss/skycli/internal/skylight"
)

func runDoctor(rc *runCtx, args []string) int {
	_ = args
	checks := []map[string]any{}

	_, _, tokenErr := rc.requireToken()
	tokenOK := tokenErr == nil
	tokenCheck := map[string]any{"check": "token_present", "ok": tokenOK}
	if tokenErr != nil {
		tokenCheck["error"] = tokenErr.Error()
	}
	checks = append(checks, tokenCheck)

	if tokenErr != nil {
		return finishDoctor(rc, checks, tokenErr)
	}

	c, err := rc.client()
	if err != nil {
		return finishDoctor(rc, checks, err)
	}
	user, err := c.GetUser(rc.ctx)
	if err != nil {
		checks = append(checks, map[string]any{"check": "GET /api/user", "ok": false, "error": err.Error()})
		return finishDoctor(rc, checks, err)
	}
	checks = append(checks, map[string]any{"check": "GET /api/user", "ok": true, "user_id": user.ID})

	frameID, ferr := rc.requireFrame()
	if ferr != nil {
		checks = append(checks, map[string]any{"check": "frame_default", "ok": false, "error": ferr.Error()})
		return finishDoctor(rc, checks, ferr)
	}
	frame, err := c.GetFrame(rc.ctx, frameID)
	if err != nil {
		checks = append(checks, map[string]any{"check": "GET /api/frames/<id>", "ok": false, "error": err.Error()})
		return finishDoctor(rc, checks, err)
	}
	checks = append(checks, map[string]any{"check": "GET /api/frames/<id>", "ok": true, "frame_id": frame.ID, "name": frame.Attributes.Name})

	cats, err := c.ListCategories(rc.ctx, frameID)
	if err != nil {
		var apiErr *skylight.APIError
		if errors.As(err, &apiErr) {
			checks = append(checks, map[string]any{"check": "GET /api/frames/<id>/categories", "ok": false, "status": apiErr.Status, "error": apiErr.Message})
		} else {
			checks = append(checks, map[string]any{"check": "GET /api/frames/<id>/categories", "ok": false, "error": err.Error()})
		}
		return finishDoctor(rc, checks, err)
	}
	checks = append(checks, map[string]any{"check": "GET /api/frames/<id>/categories", "ok": true, "count": len(cats)})

	return finishDoctor(rc, checks, nil)
}

func finishDoctor(rc *runCtx, checks []map[string]any, err error) int {
	ok := err == nil && doctorChecksOK(checks)
	if rc.g.asJSON {
		out := map[string]any{"checks": checks, "ok": ok}
		if err != nil {
			out["error"] = err.Error()
		}
		_ = rc.out.JSON(out)
		if !ok {
			return exitErr
		}
		return exitOK
	}
	rc.out.Table([]string{"CHECK", "OK", "INFO"}, doctorRows(checks))
	if !ok {
		rc.out.Line("FAIL: %s", err)
		return exitErr
	}
	rc.out.Line("OK")
	return exitOK
}

func doctorChecksOK(checks []map[string]any) bool {
	for _, check := range checks {
		ok, _ := check["ok"].(bool)
		if !ok {
			return false
		}
	}
	return true
}

func doctorRows(checks []map[string]any) [][]string {
	rows := make([][]string, 0, len(checks))
	for _, c := range checks {
		name, _ := c["check"].(string)
		ok, _ := c["ok"].(bool)
		info := ""
		for k, v := range c {
			if k == "check" || k == "ok" {
				continue
			}
			if info != "" {
				info += "; "
			}
			info += k + "=" + toString(v)
		}
		rows = append(rows, []string{name, boolYN(ok), info})
	}
	return rows
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}
