package cli

import (
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
	case "device":
		return framesDevice(rc, args[1:])
	case "household-config":
		return framesHouseholdConfig(rc, args[1:])
	case "alarms":
		return framesAlarms(rc, args[1:])
	case "notifications":
		return framesNotifications(rc, args[1:])
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
	case "set-default":
		return framesSetDefault(rc, args[1:])
	default:
		// allow `frames` to be aliased to show
		return framesShow(rc, args)
	}
}

func framesNotifications(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("frames notifications", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	kind := fs.String("type", "", "notification type: event or task")
	if err := fs.Parse(args); err != nil {
		return exitUsage
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
		return exitUsage
	}
	return runResourceJSON(rc, func(c *skylight.Client) (any, error) {
		return c.ListMonthReviews(rc.ctx)
	})
}

func framesReminderProfile(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("frames reminder-profile", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
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
		return exitUsage
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
		return exitUsage
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
		return exitUsage
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
		return exitUsage
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
		return exitUsage
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
		return exitUsage
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
		return exitUsage
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
		fmt.Fprintf(rc.stdout, "default frame set to %d\n", id)
	}
	return exitOK
}
