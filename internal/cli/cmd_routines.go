package cli

import (
	"flag"
	"fmt"
	"strings"

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
		return routinesWrite(rc, args[1:], false)
	case "update":
		return routinesWrite(rc, args[1:], true)
	case "delete":
		return choresDelete(rc, args[1:])
	case "reorder":
		return routinesReorder(rc, args[1:])
	case "move":
		return choresMove(rc, args[1:])
	case "complete":
		return choresSetCompletion(rc, args[1:], "complete")
	case "skip":
		return choresSetCompletion(rc, args[1:], "skipped")
	case "undo", "unskip":
		return choresSetCompletion(rc, args[1:], "pending")
	default:
		return usage(rc, "unknown routines subcommand: "+args[0])
	}
}

func runRoutine(rc *runCtx, args []string) int { return runRoutines(rc, args) }

func routinesList(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("routines list", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frame := fs.String("frame", "", "frame ID")
	date := fs.String("date", "", "single date YYYY-MM-DD (default: today)")
	after := fs.String("after", "", "first date YYYY-MM-DD")
	before := fs.String("before", "", "last date YYYY-MM-DD")
	assignee := fs.String("assignee-id", "", "assignee category ID")
	status := fs.String("status", "", "pending | complete | skipped")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if *date == "" && *after == "" && *before == "" {
		*date = today()
	}
	if *date != "" {
		if *after == "" {
			*after = *date
		}
		if *before == "" {
			*before = *date
		}
	}
	return runFrameResourceJSON(rc, *frame, func(c *skylight.Client, frameID int64) (any, error) {
		return c.ListRoutines(rc.ctx, frameID, skylight.ChoreFilter{Date: *date, After: *after, Before: *before, AssigneeID: *assignee, Status: *status, IncludeLate: true})
	})
}

func routinesWrite(rc *runCtx, args []string, update bool) int {
	name := "routines create"
	if update {
		name = "routines update"
	}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frame := fs.String("frame", "", "frame ID")
	var id, summary, category string
	fs.StringVar(&id, "id", "", "routine ID for update")
	fs.StringVar(&id, "routine-id", "", "alias for --id")
	fs.StringVar(&summary, "summary", "", "routine task summary")
	fs.StringVar(&summary, "title", "", "alias for --summary")
	fs.StringVar(&category, "category", "", "assignee category ID")
	fs.StringVar(&category, "assignee-id", "", "alias for --category")
	categories := fs.String("categories", "", "comma-separated category IDs")
	desc := fs.String("description", "", "task description")
	emoji := fs.String("emoji", "", "emoji icon")
	start := fs.String("start", "", "first date YYYY-MM-DD (default: today)")
	recur := fs.String("recurrence", "", "daily, weekly:MO,FR, or raw RRULE")
	segment := fs.String("time-of-day", "", "morning | afternoon | evening (default: morning for new routines)")
	points := fs.Int("points", 0, "reward points")
	trackHabit := fs.Bool("track-habit", false, "track this routine as a habit")
	applyTo := fs.String("apply-to", "", "all | one | future")
	applyProfiles := fs.String("apply-to-profiles", "", "one | all")
	fs.String("steps", "", "obsolete: create one routine task per step")
	body, bodyFile := bodyFlags(fs, rc)
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if flagChanged(fs, "steps") {
		return usage(rc, "--steps is unsupported: create one routine task per step")
	}
	if update && id == "" {
		return usage(rc, "--routine-id or --id is required")
	}
	payload, err := readPayload(rc, *body, *bodyFile)
	if err != nil {
		return fail(rc, err)
	}
	if _, ok := payload["steps"]; ok {
		return usage(rc, "steps are unsupported: create one routine task per step")
	}
	if summary != "" {
		payload["summary"] = summary
	}
	if old, ok := payload["title"]; ok {
		if _, set := payload["summary"]; !set {
			payload["summary"] = old
		}
		delete(payload, "title")
	}
	if old, ok := payload["assignee_id"]; ok {
		if _, set := payload["category_ids"]; !set {
			payload["category_ids"] = []any{old}
		}
		delete(payload, "assignee_id")
	}
	if category != "" && *categories != "" {
		return usage(rc, "choose only one of --category or --categories")
	}
	if category != "" {
		*categories = category
	}
	if *categories != "" {
		ids, err := parseTaskCategories(*categories)
		if err != nil {
			return usage(rc, err.Error())
		}
		payload["category_ids"] = ids
	}
	addStringIfSet(fs, payload, "description", "description", *desc)
	addStringIfSet(fs, payload, "emoji", "emoji_icon", *emoji)
	addStringIfSet(fs, payload, "start", "start", *start)
	if flagChanged(fs, "apply-to") {
		scope, err := normalizeTaskScope(*applyTo)
		if err != nil {
			return usage(rc, err.Error())
		}
		payload["apply_to"] = scope
	}
	if *applyProfiles != "" && *applyProfiles != "one" && *applyProfiles != "all" {
		return usage(rc, "--apply-to-profiles must be one or all")
	}
	addStringIfSet(fs, payload, "apply-to-profiles", "apply_to_profiles", *applyProfiles)
	if *points < 0 {
		return usage(rc, "--points must be zero or greater")
	}
	addIntIfSet(fs, payload, "points", "reward_points", *points)
	addBoolIfSet(fs, payload, "track-habit", "track_habit", *trackHabit)
	if update && len(payload) == 0 && *recur == "" && *segment == "" {
		return usage(rc, "provide at least one update field")
	}
	if !update {
		if text, ok := payload["summary"].(string); !ok || strings.TrimSpace(text) == "" {
			return usage(rc, "--summary is required")
		}
		if _, ok := payload["category_ids"]; !ok {
			return usage(rc, "--category or --categories is required")
		}
		if _, ok := payload["start"]; !ok {
			payload["start"] = today()
		}
	}
	if !update || *recur != "" || *segment != "" {
		if *recur == "" {
			if rules, ok := payload["recurrence_set"].([]any); ok && len(rules) > 0 {
				*recur, _ = rules[0].(string)
			}
			if *recur == "" {
				*recur = "daily"
			}
		}
		if update && *segment != "" && *recur == "daily" && !flagChanged(fs, "recurrence") {
			return usage(rc, "provide --recurrence with --time-of-day to preserve the schedule")
		}
		if update && *segment == "" && !strings.Contains(*recur, "BYHOUR=") {
			return usage(rc, "provide --time-of-day with --recurrence for routine updates")
		}
		rule, err := routineRRULE(*recur, *segment)
		if err != nil {
			return usage(rc, err.Error())
		}
		payload["recurrence_set"] = []string{rule}
	}
	payload["routine"] = true
	payload["start_time"] = nil
	payload["recurring_until"] = nil
	payload["up_for_grabs"] = false
	payload["renewal_interval"] = nil
	payload["renewal_unit"] = nil
	return runFrameResourceJSON(rc, *frame, func(c *skylight.Client, frameID int64) (any, error) {
		if update {
			return c.UpdateRoutine(rc.ctx, frameID, id, payload)
		}
		return c.CreateRoutine(rc.ctx, frameID, payload)
	})
}

func routineRRULE(recurrence, segment string) (string, error) {
	rule, err := normalizeRRULE(recurrence)
	if err != nil {
		return "", err
	}
	hour := "6"
	switch strings.ToLower(segment) {
	case "", "morning":
	case "afternoon":
		hour = "14"
	case "evening":
		hour = "20"
	default:
		return "", fmt.Errorf("--time-of-day must be morning, afternoon, or evening")
	}
	parts := strings.Split(strings.TrimPrefix(rule, "RRULE:"), ";")
	kept := make([]string, 0, len(parts)+1)
	for _, part := range parts {
		if strings.HasPrefix(part, "BYHOUR=") {
			if segment == "" {
				hour = strings.TrimPrefix(part, "BYHOUR=")
			}
			continue
		}
		if strings.HasPrefix(part, "BYMINUTE=") || strings.HasPrefix(part, "BYSECOND=") || strings.HasPrefix(part, "UNTIL=") {
			continue
		}
		kept = append(kept, part)
	}
	return "RRULE:" + strings.Join(append(kept, "BYHOUR="+hour), ";"), nil
}

func parseTaskCategories(raw string) ([]int, error) {
	ids, err := parseCSVInts(raw, "categories")
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("at least one category ID is required")
	}
	for _, id := range ids {
		if id <= 0 {
			return nil, fmt.Errorf("category IDs must be positive integers")
		}
	}
	return ids, nil
}

func routinesReorder(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("routines reorder", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frame := fs.String("frame", "", "frame ID")
	ids := fs.String("routine-ids", "", "comma-separated routine series IDs in desired order")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	values := parseCSVStrings(*ids)
	if len(values) < 2 {
		return usage(rc, "--routine-ids needs at least two IDs")
	}
	seen := map[string]bool{}
	for _, id := range values {
		if _, err := parseInt64Flag(id, "routine-ids"); err != nil {
			return usage(rc, err.Error())
		}
		if seen[id] {
			return usage(rc, "--routine-ids must not contain duplicates")
		}
		seen[id] = true
	}
	return runFrameResourceJSON(rc, *frame, func(c *skylight.Client, frameID int64) (any, error) {
		return c.ReorderRoutines(rc.ctx, frameID, values)
	})
}
