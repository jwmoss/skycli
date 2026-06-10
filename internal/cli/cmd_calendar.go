package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"
)

func runCalendar(rc *runCtx, args []string) int {
	if len(args) == 0 {
		return calendarList(rc, nil)
	}
	switch args[0] {
	case "list":
		return calendarList(rc, args[1:])
	case "create":
		return calendarCreate(rc, args[1:], false)
	case "create-countdown":
		return calendarCreate(rc, args[1:], true)
	case "week":
		return calendarWeek(rc, args[1:])
	case "update":
		return calendarUpdate(rc, args[1:])
	case "delete":
		return calendarDelete(rc, args[1:])
	case "sources":
		return calendarSources(rc, args[1:])
	default:
		return usage(rc, "unknown calendar subcommand: "+args[0])
	}
}

type calendarEventEntry struct {
	ID         string `json:"id"`
	Attributes struct {
		Summary     string `json:"summary"`
		StartsAt    string `json:"starts_at"`
		EndsAt      string `json:"ends_at"`
		AllDay      bool   `json:"all_day"`
		Color       string `json:"color"`
		Description string `json:"description"`
	} `json:"attributes"`
	Relationships struct {
		Category struct {
			Data *struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"category"`
	} `json:"relationships"`
}

type weeklyCalendarDay struct {
	Day    string               `json:"day"`
	Date   string               `json:"date"`
	Events []calendarEventEntry `json:"events"`
}

func calendarList(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("calendar list", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	start := fs.String("start-date", "", "start date filter YYYY-MM-DD")
	end := fs.String("end-date", "", "end date filter YYYY-MM-DD")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	q := url.Values{}
	if *start != "" {
		q.Set("date_min", *start)
	}
	if *end != "" {
		q.Set("date_max", *end)
	}
	return doFrameJSON(rc, *frameStr, http.MethodGet, "/api/frames/%d/calendar_events", q, nil)
}

func calendarWeek(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("calendar week", flag.ContinueOnError)
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
	events, err := fetchCalendarEvents(rc, frameID, monday.Format(dateLayout), sunday.Format(dateLayout))
	if err != nil {
		return fail(rc, err)
	}
	days := buildWeeklyCalendarDays(events, monday)
	if rc.g.asJSON {
		_ = rc.out.JSON(days)
		return exitOK
	}
	rows := [][]string{}
	for _, d := range days {
		if len(d.Events) == 0 {
			rows = append(rows, []string{d.Day, d.Date, "(none)", "-", "-"})
			continue
		}
		for i, ev := range d.Events {
			day, date := d.Day, d.Date
			if i > 0 {
				day, date = "", ""
			}
			timeCol := "all day"
			if !ev.Attributes.AllDay {
				timeCol = "-"
				if len(ev.Attributes.StartsAt) >= 16 {
					timeCol = ev.Attributes.StartsAt[11:16]
				}
			}
			rows = append(rows, []string{day, date, truncate(ev.Attributes.Summary, 36), timeCol, boolYN(ev.Attributes.AllDay)})
		}
	}
	rc.out.Table([]string{"DAY", "DATE", "SUMMARY", "TIME", "ALL-DAY"}, rows)
	return exitOK
}

func fetchCalendarEvents(rc *runCtx, frameID int64, start, end string) ([]calendarEventEntry, error) {
	c, err := rc.client()
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	if start != "" {
		q.Set("date_min", start)
	}
	if end != "" {
		q.Set("date_max", end)
	}
	raw, err := c.Do(rc.ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/calendar_events", frameID), q, nil)
	if err != nil {
		return nil, err
	}
	var env struct {
		Data []calendarEventEntry `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

func buildWeeklyCalendarDays(events []calendarEventEntry, monday time.Time) []weeklyCalendarDay {
	byDate := map[string][]calendarEventEntry{}
	for _, ev := range events {
		d := ev.Attributes.StartsAt
		if len(d) >= 10 {
			d = d[:10]
		}
		byDate[d] = append(byDate[d], ev)
	}
	days := make([]weeklyCalendarDay, 7)
	for i := range days {
		d := monday.AddDate(0, 0, i)
		key := d.Format(dateLayout)
		items := byDate[key]
		sort.Slice(items, func(a, b int) bool {
			return items[a].Attributes.StartsAt < items[b].Attributes.StartsAt
		})
		days[i] = weeklyCalendarDay{Day: d.Format("Mon"), Date: key, Events: items}
	}
	return days
}

func calendarSources(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("calendar sources", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	return doFrameJSON(rc, *frameStr, http.MethodGet, "/api/frames/%d/source_calendars", nil, nil)
}

func calendarCreate(rc *runCtx, args []string, countdown bool) int {
	fs := flag.NewFlagSet("calendar create", flag.ContinueOnError)
	if countdown {
		fs = flag.NewFlagSet("calendar create-countdown", flag.ContinueOnError)
	}
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	title := fs.String("title", "", "event title")
	startAt := fs.String("start-at", "", "event start time/date")
	date := fs.String("date", "", "alias for --start-at on countdown events")
	endAt := fs.String("end-at", "", "event end time/date")
	allDay := fs.Bool("all-day", false, "all day event")
	color := fs.String("color", "", "event color")
	category := fs.String("category", "", "category ID")
	defaultEventType := ""
	if countdown {
		defaultEventType = "countdown"
	}
	eventType := fs.String("event-type", defaultEventType, "event type")
	body, bodyFile := bodyFlags(fs, rc)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	payload, err := readPayload(rc, *body, *bodyFile)
	if err != nil {
		return fail(rc, err)
	}
	if *title != "" {
		payload["summary"] = *title
	}
	if countdown && *startAt == "" && *date != "" {
		*startAt = *date
	}
	if *startAt != "" {
		payload["starts_at"] = *startAt
	}
	if *endAt != "" {
		payload["ends_at"] = *endAt
	}
	if flagChanged(fs, "all-day") || countdown {
		payload["all_day"] = *allDay || countdown
	}
	if *color != "" {
		payload["color"] = *color
	}
	if *category != "" {
		payload["category_id"] = *category
	}
	if *eventType != "" {
		payload["event_type"] = *eventType
	}
	return doFrameJSON(rc, *frameStr, http.MethodPost, "/api/frames/%d/calendar_events", nil, payload)
}

func calendarUpdate(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("calendar update", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	eventID := fs.String("event-id", "", "event ID")
	title := fs.String("title", "", "event title")
	startAt := fs.String("start-at", "", "event start time/date")
	endAt := fs.String("end-at", "", "event end time/date")
	allDay := fs.Bool("all-day", false, "all day event")
	color := fs.String("color", "", "event color")
	category := fs.String("category", "", "category ID")
	eventType := fs.String("event-type", "", "event type")
	body, bodyFile := bodyFlags(fs, rc)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*eventID, "event-id"); err != nil {
		return usage(rc, err.Error())
	}
	payload, err := readPayload(rc, *body, *bodyFile)
	if err != nil {
		return fail(rc, err)
	}
	addStringIfSet(fs, payload, "title", "summary", *title)
	addStringIfSet(fs, payload, "start-at", "starts_at", *startAt)
	addStringIfSet(fs, payload, "end-at", "ends_at", *endAt)
	addBoolIfSet(fs, payload, "all-day", "all_day", *allDay)
	addStringIfSet(fs, payload, "color", "color", *color)
	addStringIfSet(fs, payload, "category", "category_id", *category)
	addStringIfSet(fs, payload, "event-type", "event_type", *eventType)
	return doFrameJSON(rc, *frameStr, http.MethodPut, "/api/frames/%d/calendar_events/%s", nil, payload, *eventID)
}

func calendarDelete(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("calendar delete", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	eventID := fs.String("event-id", "", "event ID")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*eventID, "event-id"); err != nil {
		return usage(rc, err.Error())
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	return doNoContent(rc, methodDelete(), "/api/frames/"+formatID(frameID)+"/calendar_events/"+*eventID, nil, nil, map[string]any{"deleted": *eventID})
}
