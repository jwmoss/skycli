package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jwmoss/skycli/internal/config"
	"github.com/jwmoss/skycli/internal/skylight"
)

func auditRun(t *testing.T, url string, args ...string) (int, string, string) {
	t.Helper()
	cfg := writeTestConfig(t, config.Config{BaseURL: url, AccessToken: "fixture", DefaultFrameID: 123})
	var out, errs bytes.Buffer
	code := Run(context.Background(), append([]string{"--config", cfg}, args...), strings.NewReader(""), &out, &errs)
	return code, out.String(), errs.String()
}

func TestRawRejectsTrailingArgumentsAndPreservesNumbers(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"id":9007199254740993}` {
			t.Errorf("body changed integer precision: %s", body)
		}
		_, _ = fmt.Fprint(w, `{"id":9007199254740993}`)
	}))
	defer srv.Close()
	code, _, _ := auditRun(t, srv.URL, "raw", "--method", "POST", "/api/example", "--body", `{"id":1}`)
	if code != exitUsage || calls != 0 {
		t.Errorf("trailing flags must fail before HTTP: exit=%d calls=%d", code, calls)
	}
	code, out, errs := auditRun(t, srv.URL, "raw", "--method", "POST", "--body", `{"id":9007199254740993}`, "/api/example", "--json")
	if code != exitOK || !strings.Contains(out, "9007199254740993") {
		t.Errorf("raw response lost precision: exit=%d stdout=%s stderr=%s", code, out, errs)
	}
}

func TestImportRemapsRecipesAndRejectsInvalidReferences(t *testing.T) {
	var recipeID string
	mutations := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			mutations++
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/recipes"):
			_, _ = fmt.Fprint(w, `{"data":{"id":"new"}}`)
		case strings.HasSuffix(r.URL.Path, "/sittings"):
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			recipeID, _ = body["meal_recipe_id"].(string)
			_, _ = fmt.Fprint(w, `{"data":{"id":"sitting"}}`)
		default:
			_, _ = fmt.Fprint(w, `{"data":[]}`)
		}
	}))
	defer srv.Close()
	file := filepath.Join(t.TempDir(), "export.json")
	data := `{"frame_id":123,"recipes":[{"id":"old","summary":"Soup"}],"meal_sittings":[{"summary":"Dinner","meal_recipe_id":"old"}]}`
	if err := os.WriteFile(file, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errs := auditRun(t, srv.URL, "import", "--file", file, "--resources", "recipes,sittings", "--json")
	if code != exitOK || recipeID != "new" {
		t.Errorf("recipe reference: exit=%d id=%q out=%s errs=%s", code, recipeID, out, errs)
	}
	for _, input := range []string{
		`{"frame_id":456,"recipes":[{"id":"old","summary":"Soup"}]}`,
		`{"frame_id":123,"meal_sittings":[{"summary":"Dinner","meal_recipe_id":"missing"}]}`,
	} {
		for _, dry := range []bool{false, true} {
			if err := os.WriteFile(file, []byte(input), 0o600); err != nil {
				t.Fatal(err)
			}
			before := mutations
			args := []string{"import", "--file", file, "--resources", "recipes,sittings", "--json"}
			if dry {
				args = append(args, "--dry-run")
			}
			code, _, _ = auditRun(t, srv.URL, args...)
			if code == exitOK || mutations != before {
				t.Errorf("invalid reference must fail before mutation; dry=%v exit=%d", dry, code)
			}
		}
	}
}

func TestCalendarWeekUsesFrameTimezone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/frames/123" {
			_, _ = fmt.Fprint(w, `{"data":{"id":"123","attributes":{"timezone":"America/New_York"}}}`)
			return
		}
		_, _ = fmt.Fprint(w, `{"data":[{"id":"evening","attributes":{"starts_at":"2026-09-16T01:00:00Z","summary":"Dinner"}},{"id":"all-day","attributes":{"starts_at":"2026-09-16","all_day":true}}]}`)
	}))
	defer srv.Close()
	code, out, errs := auditRun(t, srv.URL, "calendar", "week", "--date", "2026-09-15", "--json")
	var days []weeklyCalendarDay
	if code != exitOK || json.Unmarshal([]byte(out), &days) != nil {
		t.Fatalf("exit=%d out=%s errs=%s", code, out, errs)
	}
	if len(days[1].Events) != 1 || days[1].Events[0].ID != "evening" || len(days[2].Events) != 1 {
		t.Fatalf("wrong local dates: %s", out)
	}
}

func TestWatchOnceReportsFailedPoll(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"unavailable"}`, http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	code, out, _ := auditRun(t, srv.URL, "watch", "--once", "--resources", "rewards", "--json")
	if code != exitErr || strings.Contains(out, `"seeded": true`) {
		t.Fatalf("false success: exit=%d out=%s", code, out)
	}
}

type auditTransport func(*http.Request) (*http.Response, error)

func (f auditTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPhotoDownloadHonorsTimeoutAndPreservesExistingFile(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	http.DefaultTransport = auditTransport(func(r *http.Request) (*http.Response, error) {
		deadline, ok := r.Context().Deadline()
		if !ok || time.Until(deadline) > time.Second {
			t.Error("photo request has no configured timeout")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("image")), Header: make(http.Header)}, nil
	})
	file := filepath.Join(t.TempDir(), "photo.jpg")
	code, _, _ := auditRun(t, "http://unused.invalid", "--timeout", "100ms", "photos", "download", "--asset-url", "http://asset.invalid/photo", "--out", file)
	if code != exitOK {
		t.Fatalf("download exit=%d", code)
	}
	http.DefaultTransport = old
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = fmt.Fprint(w, "partial")
	}))
	defer srv.Close()
	code, _, _ = auditRun(t, srv.URL, "photos", "download", "--asset-url", srv.URL, "--out", file)
	data, err := os.ReadFile(file)
	if code != exitErr || err != nil || string(data) != "image" {
		t.Fatalf("failed download replaced file: exit=%d content=%q error=%v", code, data, err)
	}
}

func TestHelpAndReadOnlyDefaults(t *testing.T) {
	for _, args := range [][]string{{"chores", "--help"}, {"chores", "list", "--help"}} {
		code, _, errs := auditRun(t, "http://unused.invalid", args...)
		if code != exitOK {
			t.Errorf("help %v exit=%d stderr=%s", args, code, errs)
		}
	}
	if !isReadOnlyInvocation([]string{"chores"}) || !isReadOnlyInvocation([]string{"photos", "download"}) {
		t.Error("readonly rejects read defaults/download")
	}
}

func TestBountiesDoesNotInferRelationshipsFromPoints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/chores") {
			_, _ = fmt.Fprint(w, `{"data":[{"id":"1","attributes":{"summary":"Unrelated task","status":"pending","reward_points":10}}]}`)
			return
		}
		_, _ = fmt.Fprint(w, `{"data":[{"id":"2","attributes":{"name":"Unrelated reward","point_value":10}}]}`)
	}))
	defer srv.Close()
	code, out, _ := auditRun(t, srv.URL, "bounties", "list", "--json")
	var pairs []bountyResult
	if code != exitOK || json.Unmarshal([]byte(out), &pairs) != nil || len(pairs) != 0 {
		t.Fatalf("invented relationship: %s", out)
	}
}

func TestImportRejectsMissingCategoriesBeforeMutation(t *testing.T) {
	mutations := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			mutations++
		}
		_, _ = fmt.Fprint(w, `{"data":[]}`)
	}))
	defer srv.Close()
	file := filepath.Join(t.TempDir(), "export.json")
	if err := os.WriteFile(file, []byte(`{"frame_id":123,"calendar_events":[{"summary":"Event","starts_at":"2026-09-20","category_id":"missing"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"--dry-run", "--json"} {
		code, _, _ := auditRun(t, srv.URL, "import", "--file", file, "--resources", "calendar", mode)
		if code == exitOK || mutations != 0 {
			t.Errorf("missing category accepted: exit=%d mutations=%d", code, mutations)
		}
	}
}

func TestWatchRejectsUnknownResource(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = fmt.Fprint(w, `{"data":[]}`) }))
	defer srv.Close()
	code, _, _ := auditRun(t, srv.URL, "watch", "--once", "--resources", "rewards,typo", "--json")
	if code != exitUsage {
		t.Fatalf("unknown resource must be a usage error: %d", code)
	}
}

func TestWatchRefreshesBetweenPollsAndScopesState(t *testing.T) {
	refreshes, reads := 0, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			refreshes++
			_, _ = fmt.Fprint(w, `{"access_token":"fresh-fixture","refresh_token":"refresh-fixture","expires_in":3600}`)
			return
		}
		reads++
		if reads == 2 && r.Header.Get("Authorization") != "Bearer fresh-fixture" {
			t.Error("poll retained expired credentials")
		}
		_, _ = fmt.Fprint(w, `{"data":[]}`)
	}))
	defer srv.Close()
	cfg := config.Config{BaseURL: srv.URL, AccessToken: "old-fixture", RefreshToken: "refresh-fixture", AccessTokenExpAt: time.Now().Add(time.Hour), DeviceFingerprint: "device", SecretsBackend: secretsBackendConfig}
	cfgPath := writeTestConfig(t, cfg)
	rc := &runCtx{ctx: context.Background(), g: &globals{configPath: cfgPath, timeout: time.Second}, cfg: &cfg, stderr: io.Discard, out: newPrinter(io.Discard, true, false)}
	state := newWatchState()
	if err := pollWatch(rc.ctx, rc, 123, state, []string{"rewards"}, time.UTC); err != nil {
		t.Fatal(err)
	}
	cfg.AccessTokenExpAt = time.Unix(1, 0)
	if err := config.Save(cfgPath, &cfg); err != nil {
		t.Fatal(err)
	}
	if err := pollWatch(rc.ctx, rc, 123, state, []string{"rewards"}, time.UTC); err != nil {
		t.Fatal(err)
	}
	if refreshes != 1 {
		t.Fatalf("refresh calls=%d", refreshes)
	}
	a, err := rc.watchStatePath(123, "account-a")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := rc.watchStatePath(123, "account-b")
	c, _ := rc.watchStatePath(456, "account-a")
	if a == b || a == c || b == c {
		t.Fatal("watch state is shared across accounts or frames")
	}
}

func TestFailedRecipeCreationDoesNotReuseOldRecipe(t *testing.T) {
	sittings := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/sittings") {
			sittings++
		}
		http.Error(w, `{"message":"recipe failed"}`, http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	file := filepath.Join(t.TempDir(), "export.json")
	if err := os.WriteFile(file, []byte(`{"frame_id":123,"recipes":[{"id":"old","summary":"Soup"}],"meal_sittings":[{"summary":"Dinner","meal_recipe_id":"old"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, _ := auditRun(t, srv.URL, "import", "--file", file, "--resources", "recipes,sittings", "--json")
	if code != exitErr || sittings != 0 || !strings.Contains(out, "referenced recipe was not created") {
		t.Fatalf("unsafe partial import: code=%d sittings=%d out=%s", code, sittings, out)
	}
}

func TestSkippedChoresDoNotBreakLocalStreak(t *testing.T) {
	var chores []skylight.Chore
	if err := json.Unmarshal([]byte(`[
	 {"attributes":{"start":"2026-09-18","status":"complete"},"relationships":{"category":{"data":{"id":"1"}}}},
	 {"attributes":{"start":"2026-09-19","status":"skipped"},"relationships":{"category":{"data":{"id":"1"}}}},
	 {"attributes":{"start":"2026-09-20","status":"complete"},"relationships":{"category":{"data":{"id":"1"}}}}
	]`), &chores); err != nil {
		t.Fatal(err)
	}
	stats := computeChoreStreaks(chores, []string{"2026-09-18", "2026-09-19", "2026-09-20"}, map[string]string{"1": "Person"})
	if len(stats) != 1 || stats[0].CurrentStreak != 2 || stats[0].TotalChores != 2 {
		t.Fatalf("skip broke local streak: %+v", stats)
	}
}

type failedOutput struct{}

func (failedOutput) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestCommandFailsWhenOutputCannotBeWritten(t *testing.T) {
	cfg := writeTestConfig(t, config.Config{})
	code := Run(context.Background(), []string{"--config", cfg, "version", "--json"}, strings.NewReader(""), failedOutput{}, io.Discard)
	if code != exitErr {
		t.Fatalf("output failure returned %d", code)
	}
}
