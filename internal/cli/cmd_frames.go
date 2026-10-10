package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/jwmoss/skycli/internal/skylight"
)

func runFrames(rc *runCtx, args []string) int {
	if len(args) == 0 {
		return framesList(rc, nil)
	}
	switch args[0] {
	case "list":
		return framesList(rc, args[1:])
	case "show":
		return framesShow(rc, args[1:])
	case "devices":
		return framesDevices(rc, args[1:])
	case "users":
		return framesUsers(rc, args[1:])
	case "device-config":
		return framesDeviceConfig(rc, args[1:])
	case "update-device-config":
		return framesUpdateDeviceConfig(rc, args[1:])
	case "create-nudge":
		return framesWriteNudge(rc, args[1:], false)
	case "update-nudge":
		return framesWriteNudge(rc, args[1:], true)
	case "delete-nudge":
		return framesDeleteNudge(rc, args[1:])
	case "device":
		return framesDevice(rc, args[1:])
	case "update-device":
		return framesUpdateDevice(rc, args[1:])
	case "sleep-device", "wake-device":
		return framesDeviceSleep(rc, args[1:], args[0] == "sleep-device")
	case "household-config":
		return framesHouseholdConfig(rc, args[1:])
	case "update-household-config":
		return framesUpdateHouseholdConfig(rc, args[1:])
	case "alarms":
		return framesAlarms(rc, args[1:])
	case "notifications":
		return framesNotifications(rc, args[1:])
	case "update-notifications":
		return framesUpdateNotifications(rc, args[1:])
	case "month-reviews":
		return framesMonthReviews(rc, args[1:])
	case "reminder-profile":
		return framesReminderProfile(rc, args[1:])
	case "nudges":
		return framesNudges(rc, args[1:])
	case "avatars":
		return runResourceJSON(rc, func(c *skylight.Client) (any, error) {
			return c.ListAvatars(rc.ctx)
		})
	case "colors":
		return runResourceJSON(rc, func(c *skylight.Client) (any, error) {
			return c.ListColors(rc.ctx)
		})
	case "hats":
		return runResourceJSON(rc, func(c *skylight.Client) (any, error) {
			return c.ListHatPacks(rc.ctx)
		})
	case "set-default":
		return framesSetDefault(rc, args[1:])
	default:
		// allow `frames` to be aliased to show
		return framesShow(rc, args)
	}
}

func framesUpdateDevice(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("frames update-device", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frame := fs.String("frame", "", "frame ID")
	deviceID := fs.String("device-id", "", "device ID (required)")
	body, bodyFile := bodyFlags(fs, rc)
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
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
		return c.UpdateFrameDevice(rc.ctx, frameID, id, payload)
	})
}

func framesDeviceSleep(rc *runCtx, args []string, asleep bool) int {
	name := "frames wake-device"
	if asleep {
		name = "frames sleep-device"
	}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frame := fs.String("frame", "", "frame ID")
	deviceID := fs.String("device-id", "", "device ID (required)")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	id, err := parseInt64Flag(*deviceID, "device-id")
	if err != nil || id < 1 {
		return usage(rc, "--device-id requires a positive integer ID")
	}
	return runFrameResourceOK(rc, *frame, map[string]any{"device_id": id, "sleeping": asleep}, func(c *skylight.Client, frameID int64) error {
		_, err := c.SetDeviceSleep(rc.ctx, frameID, id, asleep)
		return err
	})
}

func framesUpdateHouseholdConfig(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("frames update-household-config", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frame := fs.String("frame", "", "frame ID")
	body, bodyFile := bodyFlags(fs, rc)
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	payload, err := settingsPayload(rc, *body, *bodyFile)
	if err != nil {
		return usage(rc, err.Error())
	}
	return runFrameResourceJSON(rc, *frame, func(c *skylight.Client, frameID int64) (any, error) {
		return c.UpdateHouseholdConfig(rc.ctx, frameID, payload)
	})
}

func settingsPayload(rc *runCtx, body, bodyFile string) (map[string]any, error) {
	payload, err := readPayload(rc, body, bodyFile)
	if err != nil {
		return nil, err
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("provide a non-empty JSON object with --body or --body-file")
	}
	return payload, nil
}

func framesUpdateNotifications(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("frames update-notifications", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frame := fs.String("frame", "", "frame ID")
	kind := fs.String("type", "", "notification type: event or task")
	onTime := fs.Bool("on-time", false, "event notification at the start time")
	early := fs.Bool("early", false, "event notification before the start time")
	minutes := fs.Int("early-minutes-before", 0, "minutes before an event")
	body, bodyFile := bodyFlags(fs, rc)
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if *kind != "event" && *kind != "task" {
		return usage(rc, "--type must be event or task")
	}
	if *kind == "task" && (flagChanged(fs, "on-time") || flagChanged(fs, "early") || flagChanged(fs, "early-minutes-before")) {
		return usage(rc, "event notification flags require --type event")
	}
	payload, err := readPayload(rc, *body, *bodyFile)
	if err != nil {
		return usage(rc, err.Error())
	}
	addBoolIfSet(fs, payload, "on-time", "on_time", *onTime)
	addBoolIfSet(fs, payload, "early", "early", *early)
	addIntIfSet(fs, payload, "early-minutes-before", "early_minutes_before", *minutes)
	if len(payload) == 0 {
		return usage(rc, "provide notification fields or a non-empty JSON body")
	}
	if *kind == "event" {
		for _, key := range []string{"on_time", "early"} {
			if value, ok := payload[key]; ok {
				if _, ok := value.(bool); !ok {
					return usage(rc, key+" must be a boolean")
				}
			}
		}
		if value, ok := payload["early_minutes_before"]; ok {
			var amount int64
			var err error
			switch value := value.(type) {
			case int:
				amount = int64(value)
			case json.Number:
				amount, err = value.Int64()
			default:
				err = fmt.Errorf("not an integer")
			}
			if err != nil || amount < 0 {
				return usage(rc, "early_minutes_before must be a non-negative integer")
			}
		}
	} else {
		for _, key := range []string{"task_due", "task_completed"} {
			if value, ok := payload[key]; ok {
				setting, ok := value.(map[string]any)
				if !ok {
					return usage(rc, key+" must be a JSON object")
				}
				if enabled, ok := setting["enabled"]; ok {
					if _, ok := enabled.(bool); !ok {
						return usage(rc, key+".enabled must be a boolean")
					}
				}
			}
		}
	}
	return runFrameResourceJSON(rc, *frame, func(c *skylight.Client, frameID int64) (any, error) {
		if *kind == "event" {
			return c.UpdateEventNotificationSettings(rc.ctx, frameID, payload)
		}
		return c.UpdateTaskNotificationSettings(rc.ctx, frameID, payload)
	})
}

func framesNotifications(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("frames notifications", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	kind := fs.String("type", "", "notification type: event or task")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if *kind != "event" && *kind != "task" {
		return usage(rc, "--type must be event or task")
	}
	return runFrameResourceJSON(rc, *frameStr, func(
		c *skylight.Client,
		frameID int64,
	) (any, error) {
		if *kind == "event" {
			return c.GetEventNotificationSettings(rc.ctx, frameID)
		}
		return c.GetTaskNotificationSettings(rc.ctx, frameID)
	})
}

func framesMonthReviews(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("frames month-reviews", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	return runResourceJSON(rc, func(c *skylight.Client) (any, error) {
		return c.ListMonthReviews(rc.ctx)
	})
}

func framesReminderProfile(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("frames reminder-profile", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	return runResourceJSON(rc, func(c *skylight.Client) (any, error) {
		return c.GetReminderProfile(rc.ctx)
	})
}

func framesNudges(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("frames nudges", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	after := fs.String("after", "", "start time in RFC3339 format")
	before := fs.String("before", "", "end time in RFC3339 format")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if err := validateNudgeRange(*after, *before); err != nil {
		return usage(rc, err.Error())
	}
	return runFrameResourceJSON(rc, *frameStr, func(
		c *skylight.Client,
		frameID int64,
	) (any, error) {
		return c.ListNudges(rc.ctx, frameID, *after, *before)
	})
}

func validateNudgeRange(after string, before string) error {
	if err := requireFlagValue(after, "after"); err != nil {
		return err
	}
	if err := requireFlagValue(before, "before"); err != nil {
		return err
	}
	start, err := time.Parse(time.RFC3339, after)
	if err != nil {
		return fmt.Errorf("parse --after as RFC3339: %w", err)
	}
	end, err := time.Parse(time.RFC3339, before)
	if err != nil {
		return fmt.Errorf("parse --before as RFC3339: %w", err)
	}
	if end.Before(start) {
		return fmt.Errorf("--before must not be earlier than --after")
	}
	return nil
}

func framesDevice(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("frames device", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	deviceID := fs.String("device-id", "", "device ID")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if err := requireFlagValue(*deviceID, "device-id"); err != nil {
		return usage(rc, err.Error())
	}
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.GetFrameDevice(rc.ctx, frameID, *deviceID)
	})
}

func framesHouseholdConfig(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("frames household-config", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.GetHouseholdConfig(rc.ctx, frameID)
	})
}

func framesAlarms(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("frames alarms", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	deviceID := fs.String("device-id", "", "device ID")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if err := requireFlagValue(*deviceID, "device-id"); err != nil {
		return usage(rc, err.Error())
	}
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.ListDeviceAlarms(rc.ctx, frameID, *deviceID)
	})
}

func framesList(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("frames list", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	frames, err := c.ListFrames(rc.ctx)
	if err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(frames)
		return exitOK
	}
	rows := make([][]string, 0, len(frames))
	for _, frame := range frames {
		rows = append(rows, []string{
			frame.ID,
			truncate(frame.Attributes.Name, 28),
			truncate(frame.Attributes.HouseholdName, 28),
			frame.Attributes.Timezone,
			boolYN(frame.Attributes.Mine),
			boolYN(frame.Attributes.Plus),
			boolYN(frame.Attributes.Activated),
		})
	}
	rc.out.Table([]string{"ID", "NAME", "HOUSEHOLD", "TIMEZONE", "MINE", "PLUS", "ACTIVATED"}, rows)
	return exitOK
}

func framesShow(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("frames show", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("id", "", "frame ID (default: --frame or config default)")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	var frameID int64
	if *frameStr != "" {
		id, err := parseInt64Flag(*frameStr, "id")
		if err != nil {
			return fail(rc, err)
		}
		frameID = id
	} else {
		id, err := rc.requireFrame()
		if err != nil {
			return fail(rc, err)
		}
		frameID = id
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	frame, err := c.GetFrame(rc.ctx, frameID)
	if err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(frame)
		return exitOK
	}
	rc.out.Line("id:        %s", frame.ID)
	rc.out.Line("name:      %s", frame.Attributes.Name)
	rc.out.Line("household: %s", dashIfEmpty(frame.Attributes.HouseholdName))
	rc.out.Line("hardware:  %s", frame.Attributes.HardwareModel)
	rc.out.Line("timezone:  %s", frame.Attributes.Timezone)
	rc.out.Line("plus:      %s", boolYN(frame.Attributes.Plus))
	rc.out.Line("activated: %s", boolYN(frame.Attributes.Activated))
	return exitOK
}

func framesDevices(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("frames devices", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.ListFrameDevices(rc.ctx, frameID)
	})
}

func framesSetDefault(rc *runCtx, args []string) int {
	if len(args) != 1 {
		return usage(rc, "skycli frames set-default <id>")
	}
	id, err := parseInt64Flag(args[0], "id")
	if err != nil {
		return fail(rc, err)
	}
	rc.cfg.DefaultFrameID = id
	if err := rc.saveConfig(); err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(map[string]any{"default_frame_id": id})
	} else {
		rc.out.Line("default frame set to %d", id)
	}
	return exitOK
}
