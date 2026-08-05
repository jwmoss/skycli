package skylight

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type CalendarEvent struct {
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

type CalendarEventFilter struct {
	StartDate string
	EndDate   string
}

type CalendarSearchFilter struct {
	Query    string
	Timezone string
	Include  string
}

func (c *Client) ListCalendarEvents(ctx context.Context, frameID int64, filter CalendarEventFilter) (*Collection[CalendarEvent], error) {
	q := url.Values{}
	if filter.StartDate != "" {
		q.Set("date_min", filter.StartDate)
	}
	if filter.EndDate != "" {
		q.Set("date_max", filter.EndDate)
	}
	raw, err := c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/calendar_events", frameID), q, nil)
	if err != nil {
		return nil, err
	}
	return decodeCollection[CalendarEvent](raw)
}

func (c *Client) ListSourceCalendars(ctx context.Context, frameID int64) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/source_calendars", frameID), nil, nil)
}

func (c *Client) SearchCalendarEvents(ctx context.Context, frameID int64, filter CalendarSearchFilter) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("search_query", filter.Query)
	q.Set("timezone", filter.Timezone)
	q.Set("include", filter.Include)
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/calendar_events/search", frameID), q, nil)
}

func (c *Client) ListCountdownEvents(ctx context.Context, frameID int64, filter CalendarEventFilter, timezone, include string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("date_min", filter.StartDate)
	q.Set("date_max", filter.EndDate)
	q.Set("timezone", timezone)
	q.Set("include", include)
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/calendar_events/countdowns", frameID), q, nil)
}

func (c *Client) ListRecentInvitedEmails(ctx context.Context, frameID int64) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/calendar_events/recent_invited_emails", frameID), nil, nil)
}

func (c *Client) CreateCalendarEvent(ctx context.Context, frameID int64, payload map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/calendar_events", frameID), nil, payload)
}

func (c *Client) UpdateCalendarEvent(ctx context.Context, frameID int64, eventID string, payload map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, fmt.Sprintf("/api/frames/%d/calendar_events/%s", frameID, eventID), nil, payload)
}

func (c *Client) DeleteCalendarEvent(ctx context.Context, frameID int64, eventID string) error {
	_, err := c.Do(ctx, http.MethodDelete, fmt.Sprintf("/api/frames/%d/calendar_events/%s", frameID, eventID), nil, nil)
	return err
}
