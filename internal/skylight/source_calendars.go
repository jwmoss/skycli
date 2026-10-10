package skylight

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (c *Client) CreateSourceCalendar(ctx context.Context, frameID int64, attributes map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/source_calendars", frameID), nil, map[string]any{"attributes": attributes})
}

func (c *Client) UpdateSourceCalendar(ctx context.Context, frameID int64, id string, attributes map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, fmt.Sprintf("/api/frames/%d/source_calendars/%s", frameID, url.PathEscape(id)), nil, attributes)
}

func (c *Client) DeleteSourceCalendar(ctx context.Context, frameID int64, id string) error {
	_, err := c.Do(ctx, http.MethodDelete, fmt.Sprintf("/api/frames/%d/source_calendars/%s", frameID, url.PathEscape(id)), nil, nil)
	return err
}

func (c *Client) SetDefaultSourceCalendar(ctx context.Context, frameID int64, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/source_calendars/set_default_for_new_events", frameID), nil, map[string]any{"id": id})
}

func (c *Client) SetSourceCalendarCategorizations(ctx context.Context, frameID int64, id string, payload map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, fmt.Sprintf("/api/frames/%d/source_calendars/%s/source_calendar_categorizations", frameID, url.PathEscape(id)), nil, payload)
}

func (c *Client) UpdateCalendarAccount(ctx context.Context, frameID int64, id string, payload map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, fmt.Sprintf("/api/frames/%d/calendars/%s", frameID, url.PathEscape(id)), nil, payload)
}
