package cli

import (
	"flag"
	"fmt"
	"sort"
	"strconv"
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

func runStatus(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	if err := fs.Parse(args); err != nil {
		return exitUsage
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
	frame, err := c.GetFrame(rc.ctx, frameID)
	if err != nil {
		return fail(rc, fmt.Errorf("get frame: %w", err))
	}
	chores, err := c.ListChores(rc.ctx, frameID, skylight.ChoreFilter{
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
	events, err := fetchCalendarEvents(rc, frameID, todayDate, todayDate)
	if err != nil {
		return fail(rc, fmt.Errorf("list calendar events: %w", err))
	}
	categories, err := c.ListCategories(rc.ctx, frameID)
	if err != nil {
		return fail(rc, fmt.Errorf("list categories: %w", err))
	}
	points, err := c.ListRewardPoints(rc.ctx, frameID)
	if err != nil {
		return fail(rc, fmt.Errorf("list reward points: %w", err))
	}
	catNames := categoryNameMap(categories)
	pointRows := make([]map[string]any, 0, len(points))
	for _, p := range points {
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
			"frame":          frame,
			"pending_chores": len(chores),
			"events_today":   len(events),
			"points":         pointRows,
		})
		return exitOK
	}
	rc.out.Line("Frame:   %s", frame.Attributes.Name)
	rc.out.Line("Chores:  %d pending today", len(chores))
	rc.out.Line("Events:  %d today", len(events))
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
		return exitUsage
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
	categories, err := c.ListCategories(rc.ctx, frameID)
	if err != nil {
		return fail(rc, fmt.Errorf("list categories: %w", err))
	}
	chores, err := c.ListChores(rc.ctx, frameID, skylight.ChoreFilter{After: startStr, Before: endStr, IncludeLate: true, IncludeUpForGrabs: true})
	if err != nil {
		return fail(rc, fmt.Errorf("list chores: %w", err))
	}
	rewards, err := c.ListRewards(rc.ctx, frameID)
	if err != nil {
		return fail(rc, fmt.Errorf("list rewards: %w", err))
	}
	points, err := c.ListRewardPoints(rc.ctx, frameID)
	if err != nil {
		return fail(rc, fmt.Errorf("list reward points: %w", err))
	}
	events, err := fetchCalendarEvents(rc, frameID, startStr, endStr)
	if err != nil {
		return fail(rc, fmt.Errorf("list calendar events: %w", err))
	}
	stats := computeAnalyticsStats(chores, rewards, points, events, categoryNameMap(categories), start, now)
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
		return assignees[i].Name < assignees[j].Name
	})

	top := make([]analyticsChoreStat, 0, len(choreCounts))
	for title, c := range choreCounts {
		top = append(top, analyticsChoreStat{Title: title, Count: c.total, Completed: c.completed})
	}
	sort.Slice(top, func(i, j int) bool {
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
		return exitUsage
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	monday, err := weekStart(*date)
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
		todayDate := today()
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
