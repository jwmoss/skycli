package cli

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jwmoss/skycli/internal/config"
)

func TestVerifiedFeatureCommandsRunInReadonlyMode(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		switch r.URL.Path {
		case "/api/frames/123/chores/search":
			if r.URL.Query().Get("search_query") != "Camp" {
				t.Fatalf("search query = %q", r.URL.Query().Get("search_query"))
			}
		case "/api/frames/123/event_notification_settings",
			"/api/frames/123/task_notification_settings",
			"/api/month_in_reviews",
			"/api/reminder_profile":
		case "/api/frames/123/nudges":
			if r.URL.Query().Get("after") == "" || r.URL.Query().Get("before") == "" {
				t.Fatal("nudge time range is missing")
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{}`)
	}))
	defer api.Close()

	cfgPath := writeTestConfig(t, config.Config{
		BaseURL:        api.URL,
		AccessToken:    "token",
		DefaultFrameID: 123,
	})
	tests := [][]string{
		{"chores", "search", "--query", "Camp"},
		{"frames", "notifications", "--type", "event"},
		{"frames", "notifications", "--type", "task"},
		{"frames", "month-reviews"},
		{"frames", "reminder-profile"},
		{
			"frames",
			"nudges",
			"--after",
			"2026-08-18T00:00:00Z",
			"--before",
			"2026-08-19T00:00:00Z",
		},
	}
	for _, args := range tests {
		args = append([]string{"--config", cfgPath, "--readonly", "--json"}, args...)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := Run(
			context.Background(),
			args,
			strings.NewReader(""),
			&stdout,
			&stderr,
		)
		if code != exitOK {
			t.Fatalf("%v: code=%d stdout=%s stderr=%s", args, code, stdout.String(), stderr.String())
		}
	}
}

func TestValidateNudgeRange(t *testing.T) {
	tests := []struct {
		name   string
		after  string
		before string
	}{
		{"missing start", "", "2026-08-19T00:00:00Z"},
		{"bad start", "2026-08-18", "2026-08-19T00:00:00Z"},
		{"bad end", "2026-08-18T00:00:00Z", "tomorrow"},
		{"reversed", "2026-08-19T00:00:00Z", "2026-08-18T00:00:00Z"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateNudgeRange(test.after, test.before); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
