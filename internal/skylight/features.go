package skylight

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// ChoreSearchFilter controls a chore text search.
type ChoreSearchFilter struct {
	Query                  string
	IncludeUpForGrabs      bool
	EndedChoreLookbackDays int
	Limit                  int
}

// SearchChores searches current and ended chores on a frame.
func (c *Client) SearchChores(
	ctx context.Context,
	frameID int64,
	filter ChoreSearchFilter,
) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("search_query", filter.Query)
	q.Set("include_up_for_grabs", strconv.FormatBool(filter.IncludeUpForGrabs))
	if filter.EndedChoreLookbackDays > 0 {
		q.Set("ended_chore_lookback_days", strconv.Itoa(filter.EndedChoreLookbackDays))
	}
	if filter.Limit > 0 {
		q.Set("limit", strconv.Itoa(filter.Limit))
	}
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/chores/search", frameID), q, nil)
}

// GetEventNotificationSettings returns the event notification settings for a frame.
func (c *Client) GetEventNotificationSettings(
	ctx context.Context,
	frameID int64,
) (json.RawMessage, error) {
	return c.Do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("/api/frames/%d/event_notification_settings", frameID),
		nil,
		nil,
	)
}

// GetTaskNotificationSettings returns the task notification settings for a frame.
func (c *Client) GetTaskNotificationSettings(
	ctx context.Context,
	frameID int64,
) (json.RawMessage, error) {
	return c.Do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("/api/frames/%d/task_notification_settings", frameID),
		nil,
		nil,
	)
}

// ListMonthReviews returns the available monthly activity reviews.
func (c *Client) ListMonthReviews(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/month_in_reviews", nil, nil)
}

// GetReminderProfile returns the account reminder profile.
func (c *Client) GetReminderProfile(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/reminder_profile", nil, nil)
}

// ListNudges returns recorded nudges in an RFC3339 time range.
func (c *Client) ListNudges(
	ctx context.Context,
	frameID int64,
	after string,
	before string,
) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("after", after)
	q.Set("before", before)
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/nudges", frameID), q, nil)
}
