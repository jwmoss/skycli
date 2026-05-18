package cli

import (
	"flag"
	"fmt"
	"sort"
	"time"

	"github.com/jwmoss/skycli/internal/skylight"
)

const dateLayout = "2006-01-02"

type weeklyChoreDay struct {
	Day    string           `json:"day"`
	Date   string           `json:"date"`
	Chores []skylight.Chore `json:"chores"`
}

type choreStreakStats struct {
	AssigneeID       string  `json:"assignee_id"`
	AssigneeName     string  `json:"assignee_name"`
	CurrentStreak    int     `json:"current_streak"`
	LongestStreak    int     `json:"longest_streak"`
	TotalChores      int     `json:"total_chores"`
	CompletedChores  int     `json:"completed_chores"`
	CompletionRatePC float64 `json:"completion_rate_pct"`
}

func weekStart(date string) (time.Time, error) {
	var t time.Time
	if date == "" || date == "current" {
		t = time.Now()
	} else {
		parsed, err := time.Parse(dateLayout, date)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid date %q: use YYYY-MM-DD", date)
		}
		t = parsed
	}
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	monday := t.AddDate(0, 0, -(wd - 1))
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.Local), nil
}

func choresWeek(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("chores week", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	date := fs.String("date", "", "week containing this date, YYYY-MM-DD")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	monday, err := weekStart(*date)
	if err != nil {
		return fail(rc, err)
	}
	sunday := monday.AddDate(0, 0, 6)
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	chores, err := c.ListChores(rc.ctx, frameID, skylight.ChoreFilter{
		After:             monday.Format(dateLayout),
		Before:            sunday.Format(dateLayout),
		IncludeLate:       true,
		IncludeUpForGrabs: true,
	})
	if err != nil {
		return fail(rc, err)
	}
	days := buildWeeklyChoreDays(chores, monday)
	if rc.g.asJSON {
		_ = rc.out.JSON(days)
		return exitOK
	}
	rows := [][]string{}
	for _, d := range days {
		if len(d.Chores) == 0 {
			rows = append(rows, []string{d.Day, d.Date, "(none)", "-", "-"})
			continue
		}
		for i, ch := range d.Chores {
			day, date := d.Day, d.Date
			if i > 0 {
				day, date = "", ""
			}
			rows = append(rows, []string{day, date, truncate(ch.Attributes.Summary, 36), ch.Attributes.Status, ptrIntStr(ch.Attributes.RewardPoints)})
		}
	}
	rc.out.Table([]string{"DAY", "DATE", "SUMMARY", "STATUS", "PTS"}, rows)
	return exitOK
}

func buildWeeklyChoreDays(chores []skylight.Chore, monday time.Time) []weeklyChoreDay {
	byDate := map[string][]skylight.Chore{}
	for _, ch := range chores {
		d := ch.Attributes.Start
		if len(d) >= 10 {
			d = d[:10]
		}
		byDate[d] = append(byDate[d], ch)
	}
	days := make([]weeklyChoreDay, 7)
	for i := range days {
		d := monday.AddDate(0, 0, i)
		key := d.Format(dateLayout)
		items := byDate[key]
		sort.Slice(items, func(a, b int) bool {
			return items[a].Attributes.Summary < items[b].Attributes.Summary
		})
		days[i] = weeklyChoreDay{
			Day:    d.Format("Mon"),
			Date:   key,
			Chores: items,
		}
	}
	return days
}

func choresStreak(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("chores streak", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	days := fs.Int("days", 30, "number of days to analyze")
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
	end := now.Format(dateLayout)
	start := now.AddDate(0, 0, -(*days - 1)).Format(dateLayout)
	chores, err := c.ListChores(rc.ctx, frameID, skylight.ChoreFilter{
		After:             start,
		Before:            end,
		IncludeLate:       true,
		IncludeUpForGrabs: true,
	})
	if err != nil {
		return fail(rc, err)
	}
	categories, err := c.ListCategories(rc.ctx, frameID)
	if err != nil {
		return fail(rc, err)
	}
	catNames := categoryNameMap(categories)
	dates := make([]string, 0, *days)
	for i := -(*days - 1); i <= 0; i++ {
		dates = append(dates, now.AddDate(0, 0, i).Format(dateLayout))
	}
	stats := computeChoreStreaks(chores, dates, catNames)
	if rc.g.asJSON {
		_ = rc.out.JSON(stats)
		return exitOK
	}
	rows := make([][]string, 0, len(stats))
	for _, s := range stats {
		rows = append(rows, []string{
			s.AssigneeID,
			s.AssigneeName,
			fmt.Sprintf("%d", s.CurrentStreak),
			fmt.Sprintf("%d", s.LongestStreak),
			fmt.Sprintf("%d/%d", s.CompletedChores, s.TotalChores),
			fmt.Sprintf("%.1f", s.CompletionRatePC),
		})
	}
	rc.out.Table([]string{"ASSIGNEE", "NAME", "CURRENT", "LONGEST", "DONE", "RATE%"}, rows)
	return exitOK
}

func categoryNameMap(categories []skylight.Category) map[string]string {
	out := make(map[string]string, len(categories))
	for _, c := range categories {
		out[c.ID] = c.Attributes.Label
	}
	return out
}

type choreDayKey struct {
	date       string
	assigneeID string
}

func computeChoreStreaks(chores []skylight.Chore, dates []string, catNames map[string]string) []choreStreakStats {
	total := map[choreDayKey]int{}
	done := map[choreDayKey]int{}
	assigneeSet := map[string]bool{}

	for _, ch := range chores {
		if ch.Relationships.Category.Data == nil {
			continue
		}
		assigneeID := ch.Relationships.Category.Data.ID
		date := ch.Attributes.Start
		if len(date) >= 10 {
			date = date[:10]
		}
		key := choreDayKey{date: date, assigneeID: assigneeID}
		total[key]++
		if ch.Attributes.Status == "complete" {
			done[key]++
		}
		assigneeSet[assigneeID] = true
	}

	out := make([]choreStreakStats, 0, len(assigneeSet))
	for assigneeID := range assigneeSet {
		var totalChores, completedChores, longest, current int
		for _, date := range dates {
			key := choreDayKey{date: date, assigneeID: assigneeID}
			t := total[key]
			d := done[key]
			totalChores += t
			completedChores += d
			if t == 0 {
				continue
			}
			if d == t {
				current++
				if current > longest {
					longest = current
				}
			} else {
				current = 0
			}
		}
		current = 0
		for i := len(dates) - 1; i >= 0; i-- {
			key := choreDayKey{date: dates[i], assigneeID: assigneeID}
			if total[key] == 0 {
				continue
			}
			if done[key] == total[key] {
				current++
				continue
			}
			break
		}
		rate := 0.0
		if totalChores > 0 {
			rate = float64(completedChores) / float64(totalChores) * 100
		}
		name := catNames[assigneeID]
		if name == "" {
			name = assigneeID
		}
		out = append(out, choreStreakStats{
			AssigneeID:       assigneeID,
			AssigneeName:     name,
			CurrentStreak:    current,
			LongestStreak:    longest,
			TotalChores:      totalChores,
			CompletedChores:  completedChores,
			CompletionRatePC: rate,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].AssigneeName < out[j].AssigneeName
	})
	return out
}
