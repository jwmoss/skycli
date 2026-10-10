package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/jwmoss/skycli/internal/skylight"
)

func framesUsers(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("frames users", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frame := fs.String("frame", "", "frame ID")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if fs.NArg() != 0 {
		return usage(rc, "unexpected arguments")
	}
	return runFrameResourceJSON(rc, *frame, func(c *skylight.Client, frameID int64) (any, error) {
		return c.ListFrameUsers(rc.ctx, frameID)
	})
}

func framesDeviceConfig(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("frames device-config", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frame := fs.String("frame", "", "frame ID")
	deviceID := fs.String("device-id", "", "device ID (required)")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if fs.NArg() != 0 {
		return usage(rc, "unexpected arguments")
	}
	id, err := parseInt64Flag(*deviceID, "device-id")
	if err != nil || id < 1 {
		return usage(rc, "--device-id requires a positive integer ID")
	}
	return runFrameResourceJSON(rc, *frame, func(c *skylight.Client, frameID int64) (any, error) {
		return c.GetDeviceConfig(rc.ctx, frameID, id)
	})
}

func framesUpdateDeviceConfig(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("frames update-device-config", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frame := fs.String("frame", "", "frame ID")
	deviceID := fs.String("device-id", "", "device ID (required)")
	body, bodyFile := bodyFlags(fs, rc)
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if fs.NArg() != 0 {
		return usage(rc, "unexpected arguments")
	}
	id, err := parseInt64Flag(*deviceID, "device-id")
	if err != nil || id < 1 {
		return usage(rc, "--device-id requires a positive integer ID")
	}
	payload, err := settingsPayload(rc, *body, *bodyFile)
	if err != nil {
		return usage(rc, err.Error())
	}
	return runFrameResourceJSON(rc, *frame, func(c *skylight.Client, frameID int64) (any, error) {
		return c.UpdateDeviceConfig(rc.ctx, frameID, id, payload)
	})
}

func framesWriteNudge(rc *runCtx, args []string, update bool) int {
	name := "frames create-nudge"
	if update {
		name = "frames update-nudge"
	}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frame := fs.String("frame", "", "frame ID")
	var nudgeID string
	if update {
		fs.StringVar(&nudgeID, "id", "", "nudge ID (required)")
	}
	body, bodyFile := bodyFlags(fs, rc)
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if fs.NArg() != 0 {
		return usage(rc, "unexpected arguments")
	}
	var id int64
	if update {
		var err error
		id, err = parseInt64Flag(nudgeID, "id")
		if err != nil || id < 1 {
			return usage(rc, "--id requires a positive integer ID")
		}
	}
	payload, err := settingsPayload(rc, *body, *bodyFile)
	if err != nil {
		return usage(rc, err.Error())
	}
	if err := validateNudgePayload(payload); err != nil {
		return usage(rc, err.Error())
	}
	return runFrameResourceJSON(rc, *frame, func(c *skylight.Client, frameID int64) (any, error) {
		if update {
			return c.UpdateNudge(rc.ctx, frameID, id, payload)
		}
		return c.CreateNudge(rc.ctx, frameID, payload)
	})
}

func validateNudgePayload(payload map[string]any) error {
	for key, value := range payload {
		switch key {
		case "body":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("body must be a string")
			}
		case "category_ids":
			ids, ok := value.([]any)
			if !ok {
				return fmt.Errorf("category_ids must be an array of positive integer IDs")
			}
			for _, id := range ids {
				number, ok := id.(json.Number)
				if !ok {
					return fmt.Errorf("category_ids must be an array of positive integer IDs")
				}
				n, err := number.Int64()
				if err != nil || n < 1 {
					return fmt.Errorf("category_ids must be an array of positive integer IDs")
				}
			}
		case "deliver_at":
			timestamp, ok := value.(string)
			if !ok {
				return fmt.Errorf("deliver_at must be an RFC3339 timestamp")
			}
			if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
				return fmt.Errorf("deliver_at must be an RFC3339 timestamp: %w", err)
			}
		case "voice_kind":
			if value == "parent_voice" {
				return fmt.Errorf("recorded audio requires multipart upload; use the Skylight app")
			}
			if value != "kirk_voice" && value != "silent" {
				return fmt.Errorf("voice_kind must be kirk_voice or silent")
			}
		case "recorded_audio":
			if value != nil {
				return fmt.Errorf("recorded audio requires multipart upload; use the Skylight app")
			}
		case "rrule":
			// The app passes this value unchanged; the server validates recurrence.
		default:
			return fmt.Errorf("unsupported nudge field %q", key)
		}
	}
	return nil
}

func framesDeleteNudge(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("frames delete-nudge", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frame := fs.String("frame", "", "frame ID")
	nudgeID := fs.String("id", "", "nudge ID (required)")
	deliverAt := fs.String("deliver-at", "", "RFC3339 timestamp of one occurrence to delete")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if fs.NArg() != 0 {
		return usage(rc, "unexpected arguments")
	}
	id, err := parseInt64Flag(*nudgeID, "id")
	if err != nil || id < 1 {
		return usage(rc, "--id requires a positive integer ID")
	}
	if *deliverAt != "" {
		if _, err := time.Parse(time.RFC3339, *deliverAt); err != nil {
			return usage(rc, "--deliver-at must be an RFC3339 timestamp")
		}
	}
	return runFrameResourceOK(rc, *frame, map[string]any{"deleted": id}, func(c *skylight.Client, frameID int64) error {
		_, err := c.DeleteNudge(rc.ctx, frameID, id, *deliverAt)
		return err
	})
}
