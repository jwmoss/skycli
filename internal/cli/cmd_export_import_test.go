package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwmoss/skycli/internal/config"
)

// An unknown --resources token must be a usage error, never a silently empty
// export/import, because backups depend on exactly the requested resources.
func TestParseResourceSelectionRejectsUnknown(t *testing.T) {
	if _, err := parseResourceSelection("chores,bogus", allPortableResources); err == nil {
		t.Fatal("expected error for unknown resource token")
	}
	if _, err := parseResourceSelection("reward", allPortableResources); err == nil {
		t.Fatal("expected error for misspelled resource (reward vs rewards)")
	}
	sel, err := parseResourceSelection("chores,rewards", allPortableResources)
	if err != nil {
		t.Fatalf("valid selection errored: %v", err)
	}
	if !sel["chores"] || !sel["rewards"] || len(sel) != 2 {
		t.Fatalf("selection = %#v", sel)
	}
}

// A failed per-list item fetch must fail the export so a backup can never be
// written with lists missing their contents.
func TestExportFailsWhenListItemFetchFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/frames/123/lists" {
			fmt.Fprint(w, `{"data":[{"id":"77","attributes":{"label":"Groceries"}}]}`)
			return
		}
		http.Error(w, `{"message":"boom"}`, http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfgPath := writeTestConfig(t, config.Config{
		BaseURL:        srv.URL,
		AuthScheme:     config.DefaultAuthScheme,
		AccessToken:    "tok",
		DefaultFrameID: 123,
	})

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(),
		[]string{"--config", cfgPath, "export", "--frame", "123", "--resources", "lists"},
		strings.NewReader(""), &stdout, &stderr)
	if code == exitOK {
		t.Fatalf("expected non-zero exit when list items fail\nstdout=%s", stdout.String())
	}
	if strings.Contains(stdout.String(), "Groceries") {
		t.Fatalf("incomplete list export was emitted: %s", stdout.String())
	}
}

func TestCalendarExportPreservesDescriptionAndCategory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/frames/123/calendar_events" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		fmt.Fprint(w, `{"data":[{"id":"evt-1","attributes":{"summary":"Dentist","starts_at":"2026-06-10T14:00:00.000Z","ends_at":"2026-06-10T15:00:00.000Z","all_day":false,"color":"#00526D","description":"Bring forms"},"relationships":{"category":{"data":{"id":"20431189","type":"category"}}}}]}`)
	}))
	defer srv.Close()

	cfgPath := writeTestConfig(t, config.Config{
		BaseURL:        srv.URL,
		AuthScheme:     config.DefaultAuthScheme,
		AccessToken:    "tok",
		DefaultFrameID: 123,
	})

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(),
		[]string{"--config", cfgPath, "export", "--frame", "123", "--resources", "calendar", "--days", "1"},
		strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	var exported portableExport
	if err := json.Unmarshal(stdout.Bytes(), &exported); err != nil {
		t.Fatalf("parse export: %v\n%s", err, stdout.String())
	}
	if len(exported.CalendarEvents) != 1 {
		t.Fatalf("calendar events = %#v", exported.CalendarEvents)
	}
	got := exported.CalendarEvents[0]
	if got.CategoryID != "20431189" || got.Description != "Bring forms" {
		t.Fatalf("calendar event = %#v", got)
	}
}

func TestCalendarImportPostsDescriptionAndCategory(t *testing.T) {
	var payload map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/frames/123/calendar_events" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		fmt.Fprint(w, `{"data":{"id":"evt-1"}}`)
	}))
	defer srv.Close()

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "export.json")
	data, err := json.Marshal(portableExport{CalendarEvents: []portableCalendarEvent{{
		Summary:     "Dentist",
		StartsAt:    "2026-06-10T14:00:00.000Z",
		EndsAt:      "2026-06-10T15:00:00.000Z",
		CategoryID:  "20431189",
		Description: "Bring forms",
	}}})
	if err != nil {
		t.Fatalf("marshal export: %v", err)
	}
	if err := os.WriteFile(inputPath, data, 0o600); err != nil {
		t.Fatalf("write export: %v", err)
	}
	cfgPath := writeTestConfig(t, config.Config{
		BaseURL:        srv.URL,
		AuthScheme:     config.DefaultAuthScheme,
		AccessToken:    "tok",
		DefaultFrameID: 123,
	})

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(),
		[]string{"--config", cfgPath, "import", "--frame", "123", "--file", inputPath, "--resources", "calendar"},
		strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	if payload["category_id"] != "20431189" || payload["description"] != "Bring forms" {
		t.Fatalf("payload = %#v", payload)
	}
}
