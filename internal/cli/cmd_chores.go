package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jwmoss/skycli/internal/skylight"
)

func runChores(rc *runCtx, args []string) int {
	if len(args) == 0 {
		return choresList(rc, nil)
	}
	switch args[0] {
	case "list":
		return choresList(rc, args[1:])
	case "search":
		return choresSearch(rc, args[1:])
	case "week":
		return choresWeek(rc, args[1:])
	case "streak":
		return choresStreak(rc, args[1:])
	case "create":
		return choresCreate(rc, args[1:])
	case "create-up-for-grabs":
		return choresCreateUpForGrabs(rc, args[1:])
	case "update":
		return choresUpdate(rc, args[1:])
	case "claim":
		return choresClaim(rc, args[1:])
	case "complete":
		return choresSetCompletion(rc, args[1:], "complete")
	case "skip":
		return choresSetCompletion(rc, args[1:], "skipped")
	case "delete":
		return choresDelete(rc, args[1:])
	case "bulk":
		return choresBulk(rc, args[1:])
	default:
		return usage(rc, fmt.Sprintf("unknown chores subcommand: %s", args[0]))
	}
}

// ---- list ----

func choresSearch(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("chores search", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID (default: config default)")
	query := fs.String("query", "", "chore search text")
	includeUFG := fs.Bool("include-up-for-grabs", true, "include up-for-grabs chores")
	lookback := fs.Int("ended-lookback-days", 30, "days of ended chores to search")
	limit := fs.Int("limit", 100, "maximum result count")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*query, "query"); err != nil {
		return usage(rc, err.Error())
	}
	if *lookback < 1 {
		return usage(rc, "--ended-lookback-days must be greater than zero")
	}
	if *limit < 1 {
		return usage(rc, "--limit must be greater than zero")
	}
	return runFrameResourceJSON(rc, *frameStr, func(
		c *skylight.Client,
		frameID int64,
	) (any, error) {
		return c.SearchChores(rc.ctx, frameID, skylight.ChoreSearchFilter{
			Query:                  *query,
			IncludeUpForGrabs:      *includeUFG,
			EndedChoreLookbackDays: *lookback,
			Limit:                  *limit,
		})
	})
}

func choresList(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("chores list", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID (default: config default)")
	date := fs.String("date", "", "single date (YYYY-MM-DD); shorthand for --after & --before; default: today")
	after := fs.String("after", "", "filter after date (YYYY-MM-DD)")
	before := fs.String("before", "", "filter before date (YYYY-MM-DD)")
	startDate := fs.String("start-date", "", "alias for --after")
	endDate := fs.String("end-date", "", "alias for --before")
	status := fs.String("status", "", "filter by status (pending | complete | skipped)")
	assignee := fs.String("assignee-id", "", "filter by assignee/category ID")
	includeLate := fs.Bool("include-late", true, "include overdue chores")
	includeUFG := fs.Bool("include-up-for-grabs", true, "include up-for-grabs chores")
	onlyUFG := fs.Bool("up-for-grabs", false, "only show up-for-grabs chores")
	linked := fs.Bool("linked-to-profile", false, "only chores linked to a profile category")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if *after != "" && *startDate != "" && *after != *startDate {
		return usage(rc, "choose only one of --after or --start-date")
	}
	if *before != "" && *endDate != "" && *before != *endDate {
		return usage(rc, "choose only one of --before or --end-date")
	}
	if *after == "" {
		*after = *startDate
	}
	if *before == "" {
		*before = *endDate
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	if *date == "" && *after == "" && *before == "" {
		*date = today()
	}
	if *after == "" && *date != "" {
		*after = *date
	}
	if *before == "" && *date != "" {
		*before = *date
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	chores, err := c.ListChores(rc.ctx, frameID, skylight.ChoreFilter{
		Date:              *date,
		Status:            *status,
		AssigneeID:        *assignee,
		After:             *after,
		Before:            *before,
		IncludeLate:       *includeLate,
		IncludeUpForGrabs: *includeUFG,
		OnlyUpForGrabs:    *onlyUFG,
		LinkedToProfile:   *linked,
	})
	if err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(chores)
		return exitOK
	}
	rows := make([][]string, 0, len(chores))
	for _, ch := range chores {
		catID := "-"
		if ch.Relationships.Category.Data != nil {
			catID = ch.Relationships.Category.Data.ID
		}
		rows = append(rows, []string{
			ch.ID,
			catID,
			truncate(ch.Attributes.Summary, 36),
			ch.Attributes.Status,
			ptrIntStr(ch.Attributes.RewardPoints),
			boolYN(ch.Attributes.UpForGrabs),
			rruleSummary(ch.Attributes.RecurrenceSet),
		})
	}
	rc.out.Table([]string{"ID", "CAT", "SUMMARY", "STATUS", "PTS", "UFG", "RECUR"}, rows)
	return exitOK
}

// ---- create ----

func choresCreate(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("chores create", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID (default: config default)")
	catStr := fs.String("category", "", "category ID (required)")
	assigneeStr := fs.String("assignee-id", "", "alias for --category")
	summary := fs.String("summary", "", "chore summary (required)")
	title := fs.String("title", "", "alias for --summary")
	start := fs.String("start", today(), "start date YYYY-MM-DD")
	date := fs.String("date", "", "alias for --start")
	recur := fs.String("recurrence", "daily", "shorthand (daily | weekly:MO,FR) or raw RRULE")
	ufg := fs.Bool("up-for-grabs", false, "mark as up-for-grabs (claimable bonus chore)")
	points := fs.Int("points", -1, "reward_points; omit to leave null")
	desc := fs.String("description", "", "optional description")
	emoji := fs.String("emoji", "", "optional emoji_icon")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if strings.TrimSpace(*summary) == "" && strings.TrimSpace(*title) != "" {
		*summary = *title
	}
	if strings.TrimSpace(*catStr) == "" && strings.TrimSpace(*assigneeStr) != "" {
		*catStr = *assigneeStr
	}
	if strings.TrimSpace(*date) != "" {
		*start = *date
	}
	if strings.TrimSpace(*summary) == "" {
		return usage(rc, "--summary is required")
	}
	if strings.TrimSpace(*catStr) == "" {
		return usage(rc, "--category is required (find IDs with `skycli categories`)")
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	catID, err := parseInt64Flag(*catStr, "category")
	if err != nil {
		return fail(rc, err)
	}
	rule, err := normalizeRRULE(*recur)
	if err != nil {
		return fail(rc, err)
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	in := skylight.ChoreCreate{
		Summary:       *summary,
		CategoryID:    catID,
		Start:         *start,
		RecurrenceSet: []string{rule},
		UpForGrabs:    *ufg,
		Description:   *desc,
		EmojiIcon:     *emoji,
	}
	if *points >= 0 {
		p := *points
		in.RewardPoints = &p
	}
	chore, err := c.CreateChore(rc.ctx, frameID, in)
	if err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(chore)
	} else {
		rc.out.Line("created chore id=%s summary=%q", chore.ID, chore.Attributes.Summary)
	}
	return exitOK
}

func choresCreateUpForGrabs(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("chores create-up-for-grabs", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID (default: config default)")
	summary := fs.String("summary", "", "chore summary (required)")
	start := fs.String("start", today(), "start date YYYY-MM-DD")
	recur := fs.String("recurrence", "daily", "shorthand (daily | weekly:MO,FR) or raw RRULE")
	points := fs.Int("points", -1, "reward_points; omit to leave null")
	desc := fs.String("description", "", "optional description")
	emoji := fs.String("emoji", "", "optional emoji_icon")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if strings.TrimSpace(*summary) == "" {
		return usage(rc, "--summary is required")
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	rule, err := normalizeRRULE(*recur)
	if err != nil {
		return fail(rc, err)
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	in := skylight.ChoreCreate{
		Summary:       *summary,
		Start:         *start,
		RecurrenceSet: []string{rule},
		UpForGrabs:    true,
		Description:   *desc,
		EmojiIcon:     *emoji,
	}
	if *points >= 0 {
		p := *points
		in.RewardPoints = &p
	}
	chore, err := c.CreateUpForGrabsChore(rc.ctx, frameID, in)
	if err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(chore)
	} else {
		rc.out.Line("created up-for-grabs chore id=%s summary=%q", chore.ID, chore.Attributes.Summary)
	}
	return exitOK
}

func choresUpdate(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("chores update", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID (default: config default)")
	idStr := fs.String("id", "", "chore ID to update (required)")
	choreIDStr := fs.String("chore-id", "", "alias for --id")
	summary := fs.String("summary", "", "new chore summary")
	title := fs.String("title", "", "alias for --summary")
	catStr := fs.String("category", "", "new category/assignee ID")
	assigneeStr := fs.String("assignee-id", "", "alias for --category")
	start := fs.String("start", "", "new start date YYYY-MM-DD")
	date := fs.String("date", "", "alias for --start")
	status := fs.String("status", "", "new status")
	points := fs.Int("points", -1, "new reward_points (0+)")
	ufg := fs.Bool("up-for-grabs", false, "set up_for_grabs=true")
	notUFG := fs.Bool("not-up-for-grabs", false, "set up_for_grabs=false")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if strings.TrimSpace(*idStr) == "" && strings.TrimSpace(*choreIDStr) != "" {
		*idStr = *choreIDStr
	}
	if strings.TrimSpace(*summary) == "" && strings.TrimSpace(*title) != "" {
		*summary = *title
	}
	if strings.TrimSpace(*catStr) == "" && strings.TrimSpace(*assigneeStr) != "" {
		*catStr = *assigneeStr
	}
	if strings.TrimSpace(*start) == "" && strings.TrimSpace(*date) != "" {
		*start = *date
	}
	if strings.TrimSpace(*idStr) == "" {
		return usage(rc, "--id is required")
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	in := skylight.ChoreUpdate{}
	var changed bool
	if strings.TrimSpace(*summary) != "" {
		s := *summary
		in.Summary = &s
		changed = true
	}
	if strings.TrimSpace(*catStr) != "" {
		id, err := parseInt64Flag(*catStr, "category")
		if err != nil {
			return fail(rc, err)
		}
		in.CategoryID = &id
		changed = true
	}
	if strings.TrimSpace(*start) != "" {
		s := *start
		in.Start = &s
		changed = true
	}
	if strings.TrimSpace(*status) != "" {
		s := *status
		in.Status = &s
		changed = true
	}
	if *points >= 0 {
		p := *points
		in.RewardPoints = &p
		changed = true
	}
	if *ufg && *notUFG {
		return usage(rc, "choose only one of --up-for-grabs or --not-up-for-grabs")
	}
	if *ufg || *notUFG {
		v := *ufg
		in.UpForGrabs = &v
		changed = true
	}
	if !changed {
		return usage(rc, "provide at least one update field")
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	chore, err := c.UpdateChore(rc.ctx, frameID, *idStr, in)
	if err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(chore)
	} else {
		rc.out.Line("updated chore id=%s summary=%q", chore.ID, chore.Attributes.Summary)
	}
	return exitOK
}

func choresClaim(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("chores claim", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID (default: config default)")
	idStr := fs.String("id", "", "chore ID to claim (required)")
	choreIDStr := fs.String("chore-id", "", "alias for --id")
	catStr := fs.String("category", "", "category/assignee ID to claim for (required)")
	assigneeStr := fs.String("assignee-id", "", "alias for --category")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if strings.TrimSpace(*idStr) == "" && strings.TrimSpace(*choreIDStr) != "" {
		*idStr = *choreIDStr
	}
	if strings.TrimSpace(*catStr) == "" && strings.TrimSpace(*assigneeStr) != "" {
		*catStr = *assigneeStr
	}
	if strings.TrimSpace(*idStr) == "" || strings.TrimSpace(*catStr) == "" {
		return usage(rc, "--id and --category are required")
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	catID, err := parseInt64Flag(*catStr, "category")
	if err != nil {
		return fail(rc, err)
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	chore, err := c.ClaimChore(rc.ctx, frameID, *idStr, catID)
	if err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(chore)
	} else {
		rc.out.Line("claimed chore id=%s category=%d", chore.ID, catID)
	}
	return exitOK
}

func choresSetCompletion(rc *runCtx, args []string, status string) int {
	fs := flag.NewFlagSet("chores "+status, flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID (default: config default)")
	idStr := fs.String("id", "", "chore ID to update (required; composite instance IDs are accepted)")
	choreIDStr := fs.String("chore-id", "", "alias for --id")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if strings.TrimSpace(*idStr) == "" && strings.TrimSpace(*choreIDStr) != "" {
		*idStr = *choreIDStr
	}
	if strings.TrimSpace(*idStr) == "" {
		return usage(rc, "--id is required")
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	chore, err := c.SetChoreCompletion(rc.ctx, frameID, *idStr, status)
	if err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(chore)
	} else {
		rc.out.Line("%s chore id=%s", status, chore.ID)
	}
	return exitOK
}

// ---- delete ----

func choresDelete(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("chores delete", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID (default: config default)")
	idStr := fs.String("id", "", "chore ID to delete (required)")
	choreIDStr := fs.String("chore-id", "", "alias for --id")
	applyTo := fs.String("apply-to", "all", "all | this_only | this_and_following")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if strings.TrimSpace(*idStr) == "" && strings.TrimSpace(*choreIDStr) != "" {
		*idStr = *choreIDStr
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	choreID, err := parseInt64Flag(*idStr, "id")
	if err != nil {
		return fail(rc, err)
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	if err := c.DeleteChore(rc.ctx, frameID, choreID, *applyTo); err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(map[string]any{"deleted": choreID, "apply_to": *applyTo})
	} else {
		rc.out.Line("deleted chore %d (apply_to=%s)", choreID, *applyTo)
	}
	return exitOK
}

// ---- bulk ----

type bulkItem struct {
	Summary      string `json:"summary"`
	CategoryID   int64  `json:"category_id"`
	Start        string `json:"start,omitempty"`
	Recurrence   string `json:"recurrence,omitempty"`
	UpForGrabs   bool   `json:"up_for_grabs,omitempty"`
	RewardPoints *int   `json:"reward_points,omitempty"`
	Description  string `json:"description,omitempty"`
	EmojiIcon    string `json:"emoji_icon,omitempty"`
}

func choresBulk(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("chores bulk", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID (default: config default)")
	file := fs.String("file", "-", "JSON array of chore specs; - for stdin")
	sleepDur := fs.Duration("sleep", 5*time.Second, "delay between POSTs")
	stopOnError := fs.Bool("stop-on-error", false, "abort on first failure (default: continue and report)")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	var rdr io.Reader
	if *file == "-" {
		rdr = rc.stdin
	} else {
		f, err := os.Open(*file)
		if err != nil {
			return fail(rc, err)
		}
		defer f.Close()
		rdr = f
	}
	var items []bulkItem
	if err := json.NewDecoder(rdr).Decode(&items); err != nil {
		return fail(rc, fmt.Errorf("parse bulk file: %w", err))
	}
	if len(items) == 0 {
		return fail(rc, fmt.Errorf("no chores in input"))
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	results := make([]map[string]any, 0, len(items))
	var failures int
	var successes int
	for i, it := range items {
		if strings.TrimSpace(it.Summary) == "" || it.CategoryID == 0 {
			results = append(results, map[string]any{"index": i, "ok": false, "error": "summary and category_id are required"})
			failures++
			if *stopOnError {
				break
			}
			continue
		}
		recur := it.Recurrence
		if recur == "" {
			recur = "daily"
		}
		rule, err := normalizeRRULE(recur)
		if err != nil {
			results = append(results, map[string]any{"index": i, "ok": false, "summary": it.Summary, "error": err.Error()})
			failures++
			if *stopOnError {
				break
			}
			continue
		}
		start := it.Start
		if start == "" {
			start = today()
		}
		in := skylight.ChoreCreate{
			Summary:       it.Summary,
			CategoryID:    it.CategoryID,
			Start:         start,
			RecurrenceSet: []string{rule},
			UpForGrabs:    it.UpForGrabs,
			RewardPoints:  it.RewardPoints,
			Description:   it.Description,
			EmojiIcon:     it.EmojiIcon,
		}
		ch, err := c.CreateChore(rc.ctx, frameID, in)
		if err != nil {
			results = append(results, map[string]any{"index": i, "ok": false, "summary": it.Summary, "error": err.Error()})
			failures++
			if !rc.g.asJSON {
				fmt.Fprintf(rc.stderr, "[%d/%d] FAIL %s: %v\n", i+1, len(items), it.Summary, err)
			}
			if *stopOnError {
				break
			}
		} else {
			results = append(results, map[string]any{"index": i, "ok": true, "id": ch.ID, "summary": ch.Attributes.Summary})
			successes++
			if !rc.g.asJSON {
				fmt.Fprintf(rc.stderr, "[%d/%d] OK   id=%s %s\n", i+1, len(items), ch.ID, ch.Attributes.Summary)
			}
		}
		if i < len(items)-1 {
			select {
			case <-rc.ctx.Done():
				return fail(rc, rc.ctx.Err())
			case <-time.After(*sleepDur):
			}
		}
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(map[string]any{
			"total":    len(items),
			"failures": failures,
			"results":  results,
		})
	} else {
		fmt.Fprintf(rc.stderr, "done: %d ok, %d failed\n", successes, failures)
	}
	if failures > 0 {
		return exitErr
	}
	return exitOK
}

// ---- helpers ----

func today() string {
	return time.Now().Format("2006-01-02")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func rruleSummary(rules []string) string {
	if len(rules) == 0 {
		return "-"
	}
	r := rules[0]
	r = strings.TrimPrefix(r, "RRULE:")
	return r
}

func normalizeRRULE(s string) (string, error) {
	t := strings.TrimSpace(s)
	if t == "" {
		return "", fmt.Errorf("empty recurrence")
	}
	if strings.HasPrefix(t, "RRULE:") {
		return t, nil
	}
	switch strings.ToLower(t) {
	case "daily":
		return "RRULE:FREQ=DAILY;INTERVAL=1", nil
	case "weekly":
		return "RRULE:FREQ=WEEKLY;INTERVAL=1", nil
	case "monthly":
		return "RRULE:FREQ=MONTHLY;INTERVAL=1", nil
	}
	if strings.HasPrefix(strings.ToLower(t), "weekly:") {
		days := strings.ToUpper(strings.TrimPrefix(t, "weekly:"))
		days = strings.ToUpper(strings.TrimPrefix(days, "WEEKLY:"))
		return "RRULE:FREQ=WEEKLY;BYDAY=" + days, nil
	}
	return "", fmt.Errorf("unrecognized recurrence %q (use daily, weekly:MO,FR, or a raw RRULE:...)", s)
}
