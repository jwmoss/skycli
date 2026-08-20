package skylight

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFeatureReadsUseVerifiedPrivateEndpoints(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		query string
		call  func(context.Context, *Client) error
	}{
		{
			name: "chore search",
			path: "/api/frames/123/chores/search",
			query: "ended_chore_lookback_days=30&include_up_for_grabs=true&" +
				"limit=25&search_query=Camp",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.SearchChores(ctx, 123, ChoreSearchFilter{
					Query:                  "Camp",
					IncludeUpForGrabs:      true,
					EndedChoreLookbackDays: 30,
					Limit:                  25,
				})
				return err
			},
		},
		{
			name:  "event notifications",
			path:  "/api/frames/123/event_notification_settings",
			query: "",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.GetEventNotificationSettings(ctx, 123)
				return err
			},
		},
		{
			name:  "task notifications",
			path:  "/api/frames/123/task_notification_settings",
			query: "",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.GetTaskNotificationSettings(ctx, 123)
				return err
			},
		},
		{
			name:  "month reviews",
			path:  "/api/month_in_reviews",
			query: "",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.ListMonthReviews(ctx)
				return err
			},
		},
		{
			name:  "reminder profile",
			path:  "/api/reminder_profile",
			query: "",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.GetReminderProfile(ctx)
				return err
			},
		},
		{
			name:  "nudges",
			path:  "/api/frames/123/nudges",
			query: "after=2026-08-18T00%3A00%3A00Z&before=2026-08-19T00%3A00%3A00Z",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.ListNudges(
					ctx,
					123,
					"2026-08-18T00:00:00Z",
					"2026-08-19T00:00:00Z",
				)
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != test.path ||
					r.URL.RawQuery != test.query {
					t.Fatalf(
						"request = %s %s?%s, want GET %s?%s",
						r.Method,
						r.URL.Path,
						r.URL.RawQuery,
						test.path,
						test.query,
					)
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{}`)
			}))
			defer srv.Close()

			if err := test.call(context.Background(), New(srv.URL, "token")); err != nil {
				t.Fatalf("call: %v", err)
			}
		})
	}
}
