package cli

import (
	"flag"
	"net/http"
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
	return doFrameJSON(rc, *frameStr, http.MethodGet, "/api/frames/%d/routines", nil, nil)
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
	return doFrameJSON(rc, *frameStr, http.MethodPost, "/api/frames/%d/routines", nil, payload)
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
	return doFrameJSON(rc, *frameStr, http.MethodPut, "/api/frames/%d/routines/%s", nil, payload, *routineID)
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
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	return doNoContent(rc, http.MethodDelete, "/api/frames/"+formatID(frameID)+"/routines/"+*routineID, nil, nil, map[string]any{"deleted": *routineID})
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
	body := map[string]any{"ids": parseCSVStrings(*ids)}
	return doFrameJSON(rc, *frameStr, http.MethodPatch, "/api/frames/%d/routines/reorder", nil, body)
}
