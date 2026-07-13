package cli

import (
	"flag"

	"github.com/jwmoss/skycli/internal/skylight"
)

func runRoutines(rc *runCtx, args []string) int {
	if len(args) == 0 {
		return routinesList(rc, nil)
	}
	switch args[0] {
	case "list":
		return routinesList(rc, args[1:])
	case "create":
		return routinesCreate(rc, args[1:])
	case "update":
		return routinesUpdate(rc, args[1:])
	case "delete":
		return routinesDelete(rc, args[1:])
	case "reorder":
		return routinesReorder(rc, args[1:])
	default:
		return usage(rc, "unknown routines subcommand: "+args[0])
	}
}

func runRoutine(rc *runCtx, args []string) int {
	return runRoutines(rc, args)
}

func routinesList(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("routines list", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.ListRoutines(rc.ctx, frameID)
	})
}

func routinePayload(rc *runCtx, fs *flag.FlagSet, body, bodyFile, title, assigneeID, steps string) (map[string]any, error) {
	payload, err := readPayload(rc, body, bodyFile)
	if err != nil {
		return nil, err
	}
	addStringIfSet(fs, payload, "title", "title", title)
	addStringIfSet(fs, payload, "assignee-id", "assignee_id", assigneeID)
	if flagChanged(fs, "steps") {
		payload["steps"] = parseCSVStrings(steps)
	}
	return payload, nil
}

func routinesCreate(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("routines create", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	title := fs.String("title", "", "routine title")
	assigneeID := fs.String("assignee-id", "", "assignee category ID")
	steps := fs.String("steps", "", "comma-separated step titles")
	body, bodyFile := bodyFlags(fs, rc)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	payload, err := routinePayload(rc, fs, *body, *bodyFile, *title, *assigneeID, *steps)
	if err != nil {
		return fail(rc, err)
	}
	if *title != "" {
		payload["title"] = *title
	}
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.CreateRoutine(rc.ctx, frameID, payload)
	})
}

func routinesUpdate(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("routines update", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	routineID := fs.String("routine-id", "", "routine ID")
	title := fs.String("title", "", "routine title")
	assigneeID := fs.String("assignee-id", "", "assignee category ID")
	steps := fs.String("steps", "", "comma-separated step titles")
	body, bodyFile := bodyFlags(fs, rc)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*routineID, "routine-id"); err != nil {
		return usage(rc, err.Error())
	}
	payload, err := routinePayload(rc, fs, *body, *bodyFile, *title, *assigneeID, *steps)
	if err != nil {
		return fail(rc, err)
	}
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.UpdateRoutine(rc.ctx, frameID, *routineID, payload)
	})
}

func routinesDelete(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("routines delete", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	routineID := fs.String("routine-id", "", "routine ID")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*routineID, "routine-id"); err != nil {
		return usage(rc, err.Error())
	}
	return runFrameResourceOK(rc, *frameStr, map[string]any{"deleted": *routineID}, func(c *skylight.Client, frameID int64) error {
		return c.DeleteRoutine(rc.ctx, frameID, *routineID)
	})
}

func routinesReorder(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("routines reorder", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	ids := fs.String("routine-ids", "", "comma-separated routine IDs in desired order")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*ids, "routine-ids"); err != nil {
		return usage(rc, err.Error())
	}
	routineIDs := parseCSVStrings(*ids)
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.ReorderRoutines(rc.ctx, frameID, routineIDs)
	})
}
