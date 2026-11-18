package cli

import (
	"context"
	"flag"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/jwmoss/skycli/internal/skylight"
)

type analyticsStats struct {
	PeriodDays int                     `json:"period_days"`
	StartDate  string                  `json:"start_date"`
	EndDate    string                  `json:"end_date"`
	Assignees  []analyticsAssigneeStat `json:"assignees"`
	TopChores  []analyticsChoreStat    `json:"top_chores"`
	Rewards    analyticsRewardStat     `json:"rewards"`
	Calendar   analyticsCalendarStat   `json:"calendar"`
}

type analyticsAssigneeStat struct {
	Name            string  `json:"name"`
	CategoryID      string  `json:"category_id"`
	TotalChores     int     `json:"total_chores"`
	CompletedChores int     `json:"completed_chores"`
	CompletionRate  float64 `json:"completion_rate"`
	PointBalance    int     `json:"point_balance"`
}

type analyticsChoreStat struct {
	Title     string `json:"title"`
	Count     int    `json:"count"`
	Completed int    `json:"completed"`
}

type analyticsRewardStat struct {
	Total    int `json:"total"`
	Redeemed int `json:"redeemed"`
}

type analyticsCalendarStat struct {
	TotalEvents  int     `json:"total_events"`
	EventsPerDay float64 `json:"events_per_day"`
}

type countPair struct {
	total     int
	completed int
}

type reportResources struct {
	frame      *skylight.Frame
	chores     []skylight.Chore
	events     *skylight.Collection[skylight.CalendarEvent]
	categories []skylight.Category
	points     []skylight.RewardPoint
	rewards    []skylight.Reward
}

func reportRead[T any](name string, dst *T, read func() (T, error)) func() error {
	return func() error {
		value, err := read()
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		*dst = value
		return nil
	}
}

// Each batch uses a single authenticated client. Check errors in command order.
func readReport(ctx context.Context, c *skylight.Client, id int64, chores skylight.ChoreFilter, statusReport bool) (reportResources, error) {
	var data reportResources
	categories := reportRead("list categories", &data.categories, func() ([]skylight.Category, error) { return c.ListCategories(ctx, id) })
	listChores := reportRead("list chores", &data.chores, func() ([]skylight.Chore, error) { return c.ListChores(ctx, id, chores) })
	events := reportRead("list calendar events", &data.events, func() (*skylight.Collection[skylight.CalendarEvent], error) {
		return c.ListCalendarEvents(ctx, id, skylight.CalendarEventFilter{StartDate: chores.After, EndDate: chores.Before})
	})
	points := reportRead("list reward points", &data.points, func() ([]skylight.RewardPoint, error) { return c.ListRewardPoints(ctx, id) })
	var reads []func() error
	if statusReport {
		frame := reportRead("get frame", &data.frame, func() (*skylight.Frame, error) { return c.GetFrame(ctx, id) })
		reads = []func() error{frame, listChores, events, categories, points}
	} else {
		rewards := reportRead("list rewards", &data.rewards, func() ([]skylight.Reward, error) { return c.ListRewards(ctx, id) })
		reads = []func() error{categories, listChores, rewards, points, events}
	}
	for batch := range slices.Chunk(reads, 3) {
		if err := ctx.Err(); err != nil {
			return data, err
		}
		errs := make([]error, len(batch))
		var wg sync.WaitGroup
		for i, read := range batch {
			wg.Go(func() { errs[i] = read() })
		}
		wg.Wait()
		for _, err := range errs {
			if err != nil {
				return data, err
			}
		}
	}
	return data, nil
}

func runStatus(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	todayDate := today()
	data, err := readReport(rc.ctx, c, frameID, skylight.ChoreFilter{Date: todayDate, After: todayDate, Before: todayDate, Status: "pending", IncludeLate: true, IncludeUpForGrabs: true}, true)
	if err != nil {
		return fail(rc, err)
	}
	catNames := categoryNameMap(data.categories)
	pointRows := make([]map[string]any, 0, len(data.points))
	for _, p := range data.points {
		key := strconv.FormatInt(p.CategoryID, 10)
		name := catNames[key]
		if name == "" {
			name = key
		}
		pointRows = append(pointRows, map[string]any{
			"category_id": key,
			"name":        name,
			"balance":     p.CurrentPointBalance,
			"lifetime":    p.LifetimePointsEarned,
		})
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(map[string]any{
			"frame":          data.frame,
			"pending_chores": len(data.chores),
			"events_today":   len(data.events.Data),
			"points":         pointRows,
		})
		return exitOK
	}
	rc.out.Line("Frame:   %s", data.frame.Attributes.Name)
	rc.out.Line("Chores:  %d pending today", len(data.chores))
	rc.out.Line("Events:  %d today", len(data.events.Data))
	for _, row := range pointRows {
		rc.out.Line("Points:  %s: %d", row["name"], row["balance"])
	}
	return exitOK
}

func runAnalytics(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("analytics", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	days := fs.Int("days", 30, "number of days to include")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if *days <= 0 {
		return usage(rc, "--days must be greater than 0")
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	now := time.Now()
	start := now.AddDate(0, 0, -*days)
	startStr := start.Format(dateLayout)
	endStr := now.Format(dateLayout)
	data, err := readReport(rc.ctx, c, frameID, skylight.ChoreFilter{After: startStr, Before: endStr, IncludeLate: true, IncludeUpForGrabs: true}, false)
	if err != nil {
		return fail(rc, err)
	}
	stats := computeAnalyticsStats(data.chores, data.rewards, data.points, data.events.Data, categoryNameMap(data.categories), start, now)
	if rc.g.asJSON {
		_ = rc.out.JSON(stats)
		return exitOK
	}
	rc.out.Line("Analytics: %s to %s (%d days)", stats.StartDate, stats.EndDate, stats.PeriodDays)
	rc.out.Line("")
	rc.out.Line("Family Members:")
	if len(stats.Assignees) == 0 {
		rc.out.Line("  (none)")
	} else {
		for _, a := range stats.Assignees {
			rc.out.Line("  %-20s %d/%d chores (%.1f%%) points: %d", a.Name, a.CompletedChores, a.TotalChores, a.CompletionRate, a.PointBalance)
		}
	}
	rc.out.Line("")
	rc.out.Line("Top Chores:")
	for _, ch := range stats.TopChores {
		rc.out.Line("  %-30s %d times (%d completed)", truncate(ch.Title, 30), ch.Count, ch.Completed)
	}
	rc.out.Line("")
	rc.out.Line("Rewards:  %d total, %d redeemed", stats.Rewards.Total, stats.Rewards.Redeemed)
	rc.out.Line("Calendar: %d events (%.1f/day)", stats.Calendar.TotalEvents, stats.Calendar.EventsPerDay)
	return exitOK
}

func computeAnalyticsStats(
	chores []skylight.Chore,
	rewards []skylight.Reward,
	points []skylight.RewardPoint,
	events []calendarEventEntry,
	catNames map[string]string,
	start, end time.Time,
) analyticsStats {
	assigneeCounts := map[string]*countPair{}
	choreCounts := map[string]*countPair{}
	for _, ch := range chores {
		catID := ""
		if ch.Relationships.Category.Data != nil {
			catID = ch.Relationships.Category.Data.ID
		}
		if catID != "" {
			incrCount(assigneeCounts, catID, ch.Attributes.Status)
		}
		incrCount(choreCounts, ch.Attributes.Summary, ch.Attributes.Status)
	}

	pointsByCategory := map[string]int{}
	for _, p := range points {
		pointsByCategory[strconv.FormatInt(p.CategoryID, 10)] = p.CurrentPointBalance
	}
	assignees := make([]analyticsAssigneeStat, 0, len(assigneeCounts))
	for catID, c := range assigneeCounts {
		name := catNames[catID]
		if name == "" {
			name = catID
		}
		rate := 0.0
		if c.total > 0 {
			rate = float64(c.completed) / float64(c.total) * 100
		}
		assignees = append(assignees, analyticsAssigneeStat{
			Name:            name,
			CategoryID:      catID,
			TotalChores:     c.total,
			CompletedChores: c.completed,
			CompletionRate:  rate,
			PointBalance:    pointsByCategory[catID],
		})
	}
	sort.Slice(assignees, func(i, j int) bool {
		if assignees[i].Name == assignees[j].Name {
			return assignees[i].CategoryID < assignees[j].CategoryID
		}
		return assignees[i].Name < assignees[j].Name
	})

	top := make([]analyticsChoreStat, 0, len(choreCounts))
	for title, c := range choreCounts {
		top = append(top, analyticsChoreStat{Title: title, Count: c.total, Completed: c.completed})
	}
	sort.Slice(top, func(i, j int) bool {
		if top[i].Count == top[j].Count {
			return top[i].Title < top[j].Title
		}
		return top[i].Count > top[j].Count
	})
	if len(top) > 5 {
		top = top[:5]
	}

	redeemed := 0
	for _, r := range rewards {
		if r.Attributes.RedeemedAt != nil {
			redeemed++
		}
	}
	periodDays := int(end.Sub(start).Hours()/24) + 1
	eventsPerDay := 0.0
	if periodDays > 0 {
		eventsPerDay = float64(len(events)) / float64(periodDays)
	}
	return analyticsStats{
		PeriodDays: periodDays,
		StartDate:  start.Format(dateLayout),
		EndDate:    end.Format(dateLayout),
		Assignees:  assignees,
		TopChores:  top,
		Rewards:    analyticsRewardStat{Total: len(rewards), Redeemed: redeemed},
		Calendar:   analyticsCalendarStat{TotalEvents: len(events), EventsPerDay: eventsPerDay},
	}
}

func incrCount(m map[string]*countPair, key, status string) {
	if key == "" {
		return
	}
	if m[key] == nil {
		m[key] = &countPair{}
	}
	m[key].total++
	if status == "complete" {
		m[key].completed++
	}
}

func runHome(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("home", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	noTasks := fs.Bool("no-tasks", false, "exclude pending tasks")
	noLists := fs.Bool("no-lists", false, "exclude lists")
	date := fs.String("date", "", "week containing this date, YYYY-MM-DD")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	loc, err := rc.frameLocation(frameID)
	if err != nil {
		return fail(rc, err)
	}
	monday, err := weekStartIn(*date, loc)
	if err != nil {
		return fail(rc, err)
	}
	sunday := monday.AddDate(0, 0, 6)
	events, err := fetchCalendarEvents(rc, frameID, monday.Format(dateLayout), sunday.Format(dateLayout))
	if err != nil {
		return fail(rc, fmt.Errorf("list calendar events: %w", err))
	}
	var chores []skylight.Chore
	if !*noTasks {
		todayDate := time.Now().In(loc).Format(dateLayout)
		chores, err = c.ListChores(rc.ctx, frameID, skylight.ChoreFilter{
			Date:              todayDate,
			After:             todayDate,
			Before:            todayDate,
			Status:            "pending",
			IncludeLate:       true,
			IncludeUpForGrabs: true,
		})
		if err != nil {
			return fail(rc, fmt.Errorf("list chores: %w", err))
		}
	}
	var lists []listEntry
	if !*noLists {
		lists, err = fetchLists(rc, frameID)
		if err != nil {
			return fail(rc, fmt.Errorf("list lists: %w", err))
		}
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(map[string]any{
			"week_start": monday.Format(dateLayout),
			"week_end":   sunday.Format(dateLayout),
			"events":     events,
			"tasks":      chores,
			"lists":      lists,
		})
		return exitOK
	}
	rc.out.Line("Events This Week")
	calendarRows := [][]string{}
	for _, d := range buildWeeklyCalendarDays(events, monday) {
		if len(d.Events) == 0 {
			calendarRows = append(calendarRows, []string{d.Day, d.Date, "(none)", "-"})
			continue
		}
		for i, ev := range d.Events {
			day, date := d.Day, d.Date
			if i > 0 {
				day, date = "", ""
			}
			calendarRows = append(calendarRows, []string{day, date, truncate(ev.Attributes.Summary, 36), ev.Attributes.StartsAt})
		}
	}
	rc.out.Table([]string{"DAY", "DATE", "EVENT", "START"}, calendarRows)
	if len(chores) > 0 {
		rc.out.Line("")
		rc.out.Line("Pending Tasks Today")
		rows := make([][]string, 0, len(chores))
		for _, ch := range chores {
			rows = append(rows, []string{ch.ID, truncate(ch.Attributes.Summary, 36), ptrIntStr(ch.Attributes.RewardPoints)})
		}
		rc.out.Table([]string{"ID", "SUMMARY", "PTS"}, rows)
	}
	if len(lists) > 0 {
		rc.out.Line("")
		rc.out.Line("Lists")
		rows := make([][]string, 0, len(lists))
		for _, l := range lists {
			rows = append(rows, []string{l.ID, l.Attributes.Label, l.Attributes.Kind})
		}
		rc.out.Table([]string{"ID", "LABEL", "KIND"}, rows)
	}
	return exitOK
}
