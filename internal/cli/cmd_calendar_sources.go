package cli

import (
	"flag"
	"strconv"

	"github.com/jwmoss/skycli/internal/skylight"
)

func calendarSourceWrite(rc *runCtx, action string, args []string) int {
	fs := flag.NewFlagSet("calendar "+action, flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frame := fs.String("frame", "", "frame ID")
	idFlag := "source-id"
	if action == "update-account" {
		idFlag = "account-id"
	}
	id := fs.String(idFlag, "", "calendar source or account ID")
	body, bodyFile := bodyFlags(fs, rc)
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if action != "create-source" {
		if n, err := strconv.ParseInt(*id, 10, 64); err != nil || n <= 0 {
			return usage(rc, "--"+idFlag+" must be a positive integer")
		}
	}
	if action == "delete-source" {
		return runFrameResourceOK(rc, *frame, map[string]any{"deleted": *id}, func(c *skylight.Client, frameID int64) error {
			return c.DeleteSourceCalendar(rc.ctx, frameID, *id)
		})
	}
	if action == "default-source" {
		return runFrameResourceJSON(rc, *frame, func(c *skylight.Client, frameID int64) (any, error) {
			return c.SetDefaultSourceCalendar(rc.ctx, frameID, *id)
		})
	}
	payload, err := readPayload(rc, *body, *bodyFile)
	if err != nil {
		return fail(rc, err)
	}
	if len(payload) == 0 {
		return usage(rc, "provide source attributes with --body or --body-file")
	}
	for command, field := range map[string]string{"map-source": "categorizations", "update-account": "active_calendars"} {
		if action == command {
			if value, ok := payload[field]; !ok || value == nil {
				return usage(rc, "body must contain "+field)
			}
		}
	}
	return runFrameResourceJSON(rc, *frame, func(c *skylight.Client, frameID int64) (any, error) {
		switch action {
		case "create-source":
			return c.CreateSourceCalendar(rc.ctx, frameID, payload)
		case "update-source":
			return c.UpdateSourceCalendar(rc.ctx, frameID, *id, payload)
		case "map-source":
			return c.SetSourceCalendarCategorizations(rc.ctx, frameID, *id, payload)
		default:
			return c.UpdateCalendarAccount(rc.ctx, frameID, *id, payload)
		}
	})
}
