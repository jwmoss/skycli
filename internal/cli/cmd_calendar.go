package cli

import (
	"flag"
	"sort"
	"time"

	"github.com/jwmoss/skycli/internal/skylight"
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
	case "search":
		return calendarSearch(rc, args[1:])
	case "countdowns":
		return calendarCountdowns(rc, args[1:])
	case "recent-invites":
		return calendarRecentInvites(rc, args[1:])
	default:
		return usage(rc, "unknown calendar subcommand: "+args[0])
	}
}

func calendarSearch(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("calendar search", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	query := fs.String("query", "", "event search text")
	timezone := fs.String("timezone", "UTC", "IANA timezone")
	include := fs.String("include", "categories", "related resources to include")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*query, "query"); err != nil {
		return usage(rc, err.Error())
	}
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.SearchCalendarEvents(rc.ctx, frameID, skylight.CalendarSearchFilter{Query: *query, Timezone: *timezone, Include: *include})
	})
}

func calendarCountdowns(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("calendar countdowns", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	start := fs.String("start-date", "", "start date filter YYYY-MM-DD")
	end := fs.String("end-date", "", "end date filter YYYY-MM-DD")
	timezone := fs.String("timezone", "UTC", "IANA timezone")
	include := fs.String("include", "categories", "related resources to include")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*start, "start-date"); err != nil {
		return usage(rc, err.Error())
	}
	if err := requireFlagValue(*end, "end-date"); err != nil {
		return usage(rc, err.Error())
	}
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.ListCountdownEvents(rc.ctx, frameID, skylight.CalendarEventFilter{StartDate: *start, EndDate: *end}, *timezone, *include)
	})
}

func calendarRecentInvites(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("calendar recent-invites", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.ListRecentInvitedEmails(rc.ctx, frameID)
	})
}

type calendarEventEntry = skylight.CalendarEvent

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
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.ListCalendarEvents(rc.ctx, frameID, skylight.CalendarEventFilter{StartDate: *start, EndDate: *end})
	})
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
	events, err := c.ListCalendarEvents(rc.ctx, frameID, skylight.CalendarEventFilter{StartDate: start, EndDate: end})
	if err != nil {
		return nil, err
	}
	return events.Data, nil
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
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.ListSourceCalendars(rc.ctx, frameID)
	})
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
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.CreateCalendarEvent(rc.ctx, frameID, payload)
	})
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
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.UpdateCalendarEvent(rc.ctx, frameID, *eventID, payload)
	})
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
	return runFrameResourceOK(rc, *frameStr, map[string]any{"deleted": *eventID}, func(c *skylight.Client, frameID int64) error {
		return c.DeleteCalendarEvent(rc.ctx, frameID, *eventID)
	})
}
