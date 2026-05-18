package cli

import (
	"flag"
	"fmt"
	"net/http"
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
	case "avatars":
		return doJSON(rc, http.MethodGet, "/api/avatars", nil, nil)
	case "colors":
		return doJSON(rc, http.MethodGet, "/api/colors", nil, nil)
	case "set-default":
		return framesSetDefault(rc, args[1:])
	default:
		// allow `frames` to be aliased to show
		return framesShow(rc, args)
	}
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
	return doFrameJSON(rc, *frameStr, http.MethodGet, "/api/frames/%d/devices", nil, nil)
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
