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
	"sync"
	"testing"
	"time"

	"github.com/jwmoss/skycli/internal/config"
)

func TestCommandSurfaceJSONModeWithPostCommandFlag(t *testing.T) {
	api := newFakeSkylightAPI(t)
	defer api.Close()

	tmp := t.TempDir()
	photoPath := filepath.Join(tmp, "photo.jpg")
	if err := os.WriteFile(photoPath, []byte("jpeg"), 0o600); err != nil {
		t.Fatalf("write photo: %v", err)
	}
	downloadPath := filepath.Join(tmp, "download.jpg")
	importPath := filepath.Join(tmp, "import.json")
	if err := os.WriteFile(importPath, []byte(fakePortableExport()), 0o600); err != nil {
		t.Fatalf("write import: %v", err)
	}

	cases := []struct {
		name  string
		args  []string
		stdin string
	}{
		{"commands catalog", []string{"commands", "--json"}, ""},
		{"root help catalog", []string{"--help", "--json"}, ""},
		{"version", []string{"version", "--json"}, ""},
		{"auth status", []string{"auth", "status", "--json"}, ""},
		{"auth refresh", []string{"auth", "refresh", "--json"}, ""},
		{"auth set-token", []string{"auth", "set-token", "--json"}, "Bearer replacement-token\n"},
		{"config show", []string{"config", "show", "--json"}, ""},
		{"config get", []string{"config", "get", "base_url", "--json"}, ""},
		{"config set", []string{"config", "set", "api_version", "2026-04-15", "--json"}, ""},
		{"config unset", []string{"config", "unset", "default_frame_id", "--json"}, ""},
		{"doctor flag", []string{"--doctor", "--json"}, ""},
		{"frames list", []string{"frames", "list", "--json"}, ""},
		{"frames show", []string{"frames", "show", "--json"}, ""},
		{"frames devices", []string{"frames", "devices", "--json"}, ""},
		{"frames avatars", []string{"frames", "avatars", "--json"}, ""},
		{"frames colors", []string{"frames", "colors", "--json"}, ""},
		{"frames set-default", []string{"frames", "set-default", "123", "--json"}, ""},
		{"categories", []string{"categories", "--json"}, ""},
		{"chores list", []string{"chores", "list", "--start-date", "2026-06-10", "--end-date", "2026-06-17", "--json"}, ""},
		{"chores week", []string{"chores", "week", "--date", "2026-06-10", "--json"}, ""},
		{"chores streak", []string{"chores", "streak", "--days", "7", "--json"}, ""},
		{"chores create", []string{"chores", "create", "--category", "1", "--summary", "Laundry", "--json"}, ""},
		{"chores create-up-for-grabs", []string{"chores", "create-up-for-grabs", "--summary", "Windows", "--points", "10", "--json"}, ""},
		{"chores update", []string{"chores", "update", "--id", "99", "--summary", "Laundry updated", "--json"}, ""},
		{"chores claim", []string{"chores", "claim", "--id", "99", "--category", "1", "--json"}, ""},
		{"chores complete", []string{"chores", "complete", "--id", "99-2026-06-10", "--json"}, ""},
		{"chores skip", []string{"chores", "skip", "--id", "99-2026-06-10", "--json"}, ""},
		{"chores delete", []string{"chores", "delete", "--id", "99", "--json"}, ""},
		{"chores bulk", []string{"chores", "bulk", "--file", "-", "--sleep", "0s", "--json"}, `[{"summary":"Bulk chore","category_id":1}]`},
		{"rewards list", []string{"rewards", "list", "--json"}, ""},
		{"rewards points", []string{"rewards", "points", "--json"}, ""},
		{"rewards create", []string{"rewards", "create", "--name", "TV", "--points", "10", "--categories", "1", "--json"}, ""},
		{"rewards update", []string{"rewards", "update", "--id", "55", "--name", "TV updated", "--json"}, ""},
		{"rewards redeem", []string{"rewards", "redeem", "--id", "55", "--json"}, ""},
		{"rewards unredeem", []string{"rewards", "unredeem", "--id", "55", "--json"}, ""},
		{"rewards delete", []string{"rewards", "delete", "--id", "55", "--json"}, ""},
		{"rewards bulk", []string{"rewards", "bulk", "--file", "-", "--json"}, `[{"name":"Bulk reward","point_value":5,"category_ids":[1]}]`},
		{"calendar list", []string{"calendar", "list", "--start-date", "2026-06-10", "--end-date", "2026-06-17", "--json"}, ""},
		{"calendar week", []string{"calendar", "week", "--date", "2026-06-10", "--json"}, ""},
		{"calendar sources", []string{"calendar", "sources", "--json"}, ""},
		{"calendar create", []string{"calendar", "create", "--title", "Dentist", "--start-at", "2026-06-10T14:00:00Z", "--json"}, ""},
		{"calendar create-countdown", []string{"calendar", "create-countdown", "--title", "Beach", "--date", "2026-07-01", "--json"}, ""},
		{"calendar update", []string{"calendar", "update", "--event-id", "evt-1", "--title", "Dentist updated", "--json"}, ""},
		{"calendar delete", []string{"calendar", "delete", "--event-id", "evt-1", "--json"}, ""},
		{"lists list", []string{"lists", "list", "--json"}, ""},
		{"lists show", []string{"lists", "show", "--list-id", "77", "--json"}, ""},
		{"lists create", []string{"lists", "create", "--title", "Errands", "--json"}, ""},
		{"lists update", []string{"lists", "update", "--list-id", "77", "--title", "Errands updated", "--json"}, ""},
		{"lists delete", []string{"lists", "delete", "--list-id", "77", "--json"}, ""},
		{"lists add-item", []string{"lists", "add-item", "--list-id", "77", "--title", "Milk", "--json"}, ""},
		{"lists update-item", []string{"lists", "update-item", "--list-id", "77", "--item-id", "88", "--completed", "--json"}, ""},
		{"lists delete-item", []string{"lists", "delete-item", "--list-id", "77", "--item-id", "88", "--json"}, ""},
		{"lists clear-completed", []string{"lists", "clear-completed", "--list-id", "77", "--json"}, ""},
		{"lists organize", []string{"lists", "organize", "--list-id", "77", "--json"}, ""},
		{"lists order", []string{"lists", "order", "--list-id", "77", "--retailer", "instacart", "--json"}, ""},
		{"lists task-box-items", []string{"lists", "task-box-items", "--json"}, ""},
		{"lists task-box-item", []string{"lists", "task-box-item", "--title", "Inbox", "--json"}, ""},
		{"grocery list", []string{"grocery", "list", "--json"}, ""},
		{"grocery show", []string{"grocery", "show", "--list-id", "77", "--json"}, ""},
		{"grocery create", []string{"grocery", "create", "--title", "Groceries", "--json"}, ""},
		{"grocery add", []string{"grocery", "add", "--list-id", "77", "--title", "Eggs", "--json"}, ""},
		{"grocery clear", []string{"grocery", "clear", "--list-id", "77", "--json"}, ""},
		{"grocery organize", []string{"grocery", "organize", "--list-id", "77", "--json"}, ""},
		{"grocery order", []string{"grocery", "order", "--list-id", "77", "--json"}, ""},
		{"grocery add-recipe", []string{"grocery", "add-recipe", "--recipe-id", "recipe-1", "--json"}, ""},
		{"meals categories", []string{"meals", "categories", "--json"}, ""},
		{"meals recipes", []string{"meals", "recipes", "--json"}, ""},
		{"meals recipe-info", []string{"meals", "recipe-info", "--recipe-id", "recipe-1", "--json"}, ""},
		{"meals create-recipe", []string{"meals", "create-recipe", "--title", "Tacos", "--ingredients", "beans,rice", "--json"}, ""},
		{"meals update-recipe", []string{"meals", "update-recipe", "--recipe-id", "recipe-1", "--title", "Tacos updated", "--json"}, ""},
		{"meals delete-recipe", []string{"meals", "delete-recipe", "--recipe-id", "recipe-1", "--json"}, ""},
		{"meals sittings", []string{"meals", "sittings", "--date-min", "2026-06-10", "--date-max", "2026-06-17", "--json"}, ""},
		{"meals create-sitting", []string{"meals", "create-sitting", "--recipe-id", "recipe-1", "--summary", "Dinner", "--date", "2026-06-10", "--json"}, ""},
		{"meals delete-sitting", []string{"meals", "delete-sitting", "--sitting-id", "sitting-1", "--date", "2026-06-10", "--json"}, ""},
		{"meals add-to-grocery", []string{"meals", "add-to-grocery", "--recipe-id", "recipe-1", "--json"}, ""},
		{"photos list", []string{"photos", "list", "--json"}, ""},
		{"photos upload", []string{"photos", "upload", "--file", photoPath, "--caption", "test", "--json"}, ""},
		{"photos download", []string{"photos", "download", "--asset-url", api.assetURL(), "--out", downloadPath, "--json"}, ""},
		{"photos delete", []string{"photos", "delete", "--message-ids", "10,11", "--json"}, ""},
		{"routines list", []string{"routines", "list", "--json"}, ""},
		{"routines create", []string{"routines", "create", "--title", "Morning", "--assignee-id", "1", "--steps", "Brush,Pack", "--json"}, ""},
		{"routines update", []string{"routines", "update", "--routine-id", "routine-1", "--title", "Evening", "--json"}, ""},
		{"routines reorder", []string{"routines", "reorder", "--routine-ids", "routine-1,routine-2", "--json"}, ""},
		{"routines delete", []string{"routines", "delete", "--routine-id", "routine-1", "--json"}, ""},
		{"sidekick status", []string{"sidekick", "status", "--json"}, ""},
		{"sidekick history", []string{"sidekick", "history", "--json"}, ""},
		{"bounties list", []string{"bounties", "list", "--json"}, ""},
		{"bounties create", []string{"bounties", "create", "--title", "Garage", "--points", "10", "--assignee-id", "1", "--reward-title", "Garage reward", "--json"}, ""},
		{"bounties update", []string{"bounties", "update", "--chore-id", "99", "--reward-id", "55", "--title", "Garage updated", "--json"}, ""},
		{"bounties delete", []string{"bounties", "delete", "--chore-id", "99", "--reward-id", "55", "--json"}, ""},
		{"rotations create", []string{"rotations", "create", "--chores", "Trash,Dishes", "--assignee-ids", "1,2", "--weeks", "1", "--json"}, ""},
		{"status", []string{"status", "--json"}, ""},
		{"analytics", []string{"analytics", "--days", "7", "--json"}, ""},
		{"home", []string{"home", "--date", "2026-06-10", "--json"}, ""},
		{"watch once", []string{"watch", "--once", "--json"}, ""},
		{"export", []string{"export", "--resources", "all", "--days", "1", "--json"}, ""},
		{"import dry-run", []string{"import", "--file", importPath, "--dry-run", "--json"}, ""},
		{"import", []string{"import", "--file", importPath, "--resources", "all", "--json"}, ""},
		{"raw", []string{"raw", "/api/frames/123", "--json"}, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfgPath := writeTestConfig(t, config.Config{
				BaseURL:           api.URL(),
				AuthScheme:        config.DefaultAuthScheme,
				AccessToken:       "test-token",
				RefreshToken:      "refresh-token",
				DeviceFingerprint: "fingerprint",
				DefaultFrameID:    123,
			})
			args := append([]string{"--config", cfgPath}, tc.args...)
			var stdout, stderr bytes.Buffer
			code := Run(context.Background(), args, strings.NewReader(tc.stdin), &stdout, &stderr)
			if code != exitOK {
				t.Fatalf("exit code: got %d\nargs=%v\nstdout=%s\nstderr=%s", code, args, stdout.String(), stderr.String())
			}
			assertJSONDocument(t, stdout.Bytes())
			if strings.Contains(stderr.String(), "Usage:") {
				t.Fatalf("unexpected usage text on stderr for JSON command:\n%s", stderr.String())
			}
		})
	}
}

func TestJSONUsageErrorsAreStructured(t *testing.T) {
	cfgPath := writeTestConfig(t, config.Config{BaseURL: "https://example.invalid", AccessToken: "test-token"})
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "chores", "create", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitUsage {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if got["kind"] != "usage" {
		t.Fatalf("kind = %v, want usage; stdout=%s", got["kind"], stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("usage error leaked to stderr in JSON mode: %s", stderr.String())
	}
}

func TestDoctorCommandIsNotSupported(t *testing.T) {
	cfgPath := writeTestConfig(t, config.Config{BaseURL: "https://example.invalid", AccessToken: "test-token"})
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "doctor", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitUsage {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if got["error"] != "unknown command: doctor" {
		t.Fatalf("error = %v, want unknown command: doctor", got["error"])
	}
	if stderr.Len() != 0 {
		t.Fatalf("unknown command leaked to stderr in JSON mode: %s", stderr.String())
	}
}

func assertJSONDocument(t *testing.T, data []byte) {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(bytes.TrimSpace(data)))
	var v any
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, string(data))
	}
	if dec.Decode(&v) != io.EOF {
		t.Fatalf("stdout has multiple JSON documents:\n%s", string(data))
	}
}

type fakeSkylightAPI struct {
	t      *testing.T
	api    *httptest.Server
	upload *httptest.Server
	mu     sync.Mutex
	hits   []string
}

func newFakeSkylightAPI(t *testing.T) *fakeSkylightAPI {
	t.Helper()
	f := &fakeSkylightAPI{t: t}
	f.upload = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.record(r)
		switch r.Method {
		case http.MethodPut:
			_, _ = io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodGet:
			_, _ = w.Write([]byte("asset"))
		default:
			http.Error(w, "unexpected upload method", http.StatusMethodNotAllowed)
		}
	}))
	f.api = httptest.NewServer(http.HandlerFunc(f.handle))
	return f
}

func (f *fakeSkylightAPI) Close() {
	f.api.Close()
	f.upload.Close()
}

func (f *fakeSkylightAPI) URL() string {
	return f.api.URL
}

func (f *fakeSkylightAPI) assetURL() string {
	return f.upload.URL + "/asset.jpg"
}

func (f *fakeSkylightAPI) record(r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.hits = append(f.hits, r.Method+" "+r.URL.RequestURI())
}

func (f *fakeSkylightAPI) handle(w http.ResponseWriter, r *http.Request) {
	f.record(r)
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.Copy(io.Discard, r.Body)
	path := r.URL.Path
	method := r.Method

	switch {
	case method == http.MethodGet && path == "/api/user":
		writeJSON(w, map[string]any{"data": map[string]any{"id": "user-1", "attributes": map[string]any{"email": "test@example.com"}}})
	case method == http.MethodPost && path == "/oauth/token":
		writeJSON(w, map[string]any{"access_token": "new-token", "refresh_token": "new-refresh", "expires_in": 3600})
	case method == http.MethodGet && path == "/api/frames":
		writeJSON(w, map[string]any{"data": []any{fakeFrame()}})
	case method == http.MethodGet && path == "/api/frames/123":
		writeJSON(w, map[string]any{"data": fakeFrame()})
	case method == http.MethodGet && path == "/api/frames/123/devices":
		writeJSON(w, map[string]any{"data": []any{map[string]any{"id": "device-1"}}})
	case method == http.MethodGet && path == "/api/avatars":
		writeJSON(w, map[string]any{"data": []any{map[string]any{"id": "avatar-1"}}})
	case method == http.MethodGet && path == "/api/colors":
		writeJSON(w, map[string]any{"data": []string{"#2178AF"}})
	case method == http.MethodGet && path == "/api/frames/123/categories":
		writeJSON(w, map[string]any{"data": []any{fakeCategory("1", "Kid"), fakeCategory("2", "Family")}})
	case method == http.MethodGet && path == "/api/frames/123/chores":
		writeJSON(w, map[string]any{"data": []any{fakeChore("99", "Laundry", "complete", false), fakeChore("100", "Windows", "pending", true)}})
	case method == http.MethodPost && path == "/api/frames/123/chores":
		writeJSON(w, map[string]any{"data": fakeChore("99", "Laundry", "pending", false)})
	case method == http.MethodPost && path == "/api/frames/123/chores/create_multiple":
		writeJSON(w, map[string]any{"data": []any{fakeChore("100", "Windows", "pending", true)}})
	case method == http.MethodPut && path == "/api/frames/123/chores/99":
		writeJSON(w, map[string]any{"data": fakeChore("99", "Laundry updated", "pending", false)})
	case method == http.MethodPut && path == "/api/frames/123/chores/99/completions":
		writeJSON(w, map[string]any{"data": fakeChore("99", "Laundry", "complete", false)})
	case method == http.MethodDelete && path == "/api/frames/123/chores/99":
		writeJSON(w, map[string]any{"ok": true})
	case method == http.MethodGet && path == "/api/frames/123/rewards":
		writeJSON(w, map[string]any{"data": []any{fakeReward("55", "TV", false), fakeReward("56", "Redeemed", true)}})
	case method == http.MethodPost && path == "/api/frames/123/rewards":
		writeJSON(w, map[string]any{"data": []any{fakeReward("55", "TV", false)}})
	case method == http.MethodPatch && path == "/api/frames/123/rewards/55":
		writeJSON(w, map[string]any{"data": fakeReward("55", "TV updated", false)})
	case method == http.MethodDelete && path == "/api/frames/123/rewards/55":
		writeJSON(w, map[string]any{"ok": true})
	case method == http.MethodPost && (path == "/api/frames/123/rewards/55/redeem" || path == "/api/frames/123/rewards/55/unredeem"):
		writeJSON(w, map[string]any{"ok": true})
	case method == http.MethodGet && path == "/api/frames/123/reward_points":
		writeJSON(w, []any{map[string]any{"category_id": 1, "current_point_balance": 10, "lifetime_points_earned": 25}})
	case method == http.MethodGet && path == "/api/frames/123/calendar_events":
		if r.URL.Query().Get("start_date") != "" || r.URL.Query().Get("end_date") != "" {
			http.Error(w, "legacy calendar query keys are not allowed", http.StatusBadRequest)
			return
		}
		writeJSON(w, map[string]any{"data": []any{fakeCalendarEvent()}})
	case method == http.MethodPost && path == "/api/frames/123/calendar_events":
		writeJSON(w, map[string]any{"data": fakeCalendarEvent()})
	case method == http.MethodPut && path == "/api/frames/123/calendar_events/evt-1":
		writeJSON(w, map[string]any{"data": fakeCalendarEvent()})
	case method == http.MethodDelete && path == "/api/frames/123/calendar_events/evt-1":
		writeJSON(w, map[string]any{"ok": true})
	case method == http.MethodGet && path == "/api/frames/123/source_calendars":
		writeJSON(w, map[string]any{"data": []any{map[string]any{"id": "source-1"}}})
	case method == http.MethodGet && path == "/api/frames/123/lists":
		writeJSON(w, map[string]any{"data": []any{fakeList("77", "Groceries", "grocery"), fakeList("78", "Errands", "to_do")}})
	case method == http.MethodPost && path == "/api/frames/123/lists":
		writeJSON(w, map[string]any{"data": fakeList("77", "Groceries", "grocery")})
	case method == http.MethodGet && path == "/api/frames/123/lists/77":
		writeJSON(w, map[string]any{"data": fakeList("77", "Groceries", "grocery"), "included": []any{fakeListItem("88", "Milk", "completed")}})
	case method == http.MethodGet && path == "/api/frames/123/lists/78":
		writeJSON(w, map[string]any{"data": fakeList("78", "Errands", "to_do"), "included": []any{fakeListItem("89", "Return books", "pending")}})
	case method == http.MethodPut && path == "/api/frames/123/lists/77":
		writeJSON(w, map[string]any{"data": fakeList("77", "Errands updated", "to_do")})
	case method == http.MethodDelete && path == "/api/frames/123/lists/77":
		writeJSON(w, map[string]any{"ok": true})
	case method == http.MethodPost && path == "/api/frames/123/lists/77/list_items":
		writeJSON(w, map[string]any{"data": fakeListItem("88", "Milk", "pending")})
	case method == http.MethodPut && path == "/api/frames/123/lists/77/list_items/88":
		writeJSON(w, map[string]any{"data": fakeListItem("88", "Milk", "completed")})
	case method == http.MethodDelete && path == "/api/frames/123/lists/77/list_items/88":
		writeJSON(w, map[string]any{"ok": true})
	case method == http.MethodPost && (path == "/api/frames/123/lists/77/organize" || path == "/api/frames/123/lists/77/order"):
		writeJSON(w, map[string]any{"ok": true})
	case method == http.MethodGet && path == "/api/frames/123/task_box/items":
		writeJSON(w, map[string]any{"data": []any{map[string]any{"id": "tb-1", "type": "task_box_item", "attributes": map[string]any{"summary": "Inbox"}}}})
	case method == http.MethodPost && path == "/api/frames/123/task_box/items":
		writeJSON(w, map[string]any{"data": map[string]any{"id": "tb-1", "type": "task_box_item", "attributes": map[string]any{"summary": "Inbox"}}})
	case method == http.MethodGet && path == "/api/frames/123/meals/categories":
		writeJSON(w, map[string]any{"data": []any{map[string]any{"id": "meal-cat-1", "attributes": map[string]any{"name": "Dinner"}}}})
	case method == http.MethodGet && path == "/api/frames/123/meals/recipes":
		writeJSON(w, map[string]any{"data": []any{fakeRecipe()}})
	case method == http.MethodGet && path == "/api/frames/123/meals/recipes/recipe-1":
		writeJSON(w, map[string]any{"data": fakeRecipe()})
	case method == http.MethodPost && path == "/api/frames/123/meals/recipes":
		writeJSON(w, map[string]any{"data": fakeRecipe()})
	case method == http.MethodPatch && path == "/api/frames/123/meals/recipes/recipe-1":
		writeJSON(w, map[string]any{"data": fakeRecipe()})
	case method == http.MethodDelete && path == "/api/frames/123/meals/recipes/recipe-1":
		writeJSON(w, map[string]any{"ok": true})
	case method == http.MethodGet && path == "/api/frames/123/meals/sittings":
		writeJSON(w, map[string]any{"data": []any{fakeSitting()}})
	case method == http.MethodPost && path == "/api/frames/123/meals/sittings":
		writeJSON(w, map[string]any{"data": fakeSitting()})
	case method == http.MethodDelete && path == "/api/frames/123/meals/sittings/sitting-1/instances/2026-06-10":
		writeJSON(w, map[string]any{"ok": true})
	case method == http.MethodPost && path == "/api/frames/123/meals/recipes/recipe-1/add_to_grocery_list":
		writeJSON(w, map[string]any{"ok": true})
	case method == http.MethodGet && path == "/api/frames/123/messages":
		writeJSON(w, map[string]any{"data": []any{map[string]any{"id": "10", "attributes": map[string]any{"caption": "hello"}}}})
	case method == http.MethodPost && path == "/api/upload_url":
		writeJSON(w, map[string]any{"data": map[string]any{"url": f.upload.URL + "/upload", "key": "photo-key", "get_url": f.assetURL(), "message_ids": []int{10}, "frame_names": []string{"Kitchen"}}})
	case method == http.MethodDelete && path == "/api/frames/123/messages/destroy_multiple":
		writeJSON(w, map[string]any{"ok": true})
	case method == http.MethodGet && path == "/api/frames/123/routines":
		writeJSON(w, map[string]any{"data": []any{fakeRoutine()}})
	case method == http.MethodPost && path == "/api/frames/123/routines":
		writeJSON(w, map[string]any{"data": fakeRoutine()})
	case method == http.MethodPut && path == "/api/frames/123/routines/routine-1":
		writeJSON(w, map[string]any{"data": fakeRoutine()})
	case method == http.MethodPatch && path == "/api/frames/123/routines/reorder":
		writeJSON(w, map[string]any{"ok": true})
	case method == http.MethodDelete && path == "/api/frames/123/routines/routine-1":
		writeJSON(w, map[string]any{"ok": true})
	case method == http.MethodGet && path == "/api/plus_access":
		writeJSON(w, map[string]any{"data": map[string]any{
			"bundle_entitlement":           map[string]any{"available": false},
			"self_serve_trial_eligibility": map[string]any{"assistant": true},
			"subscriptions":                []any{map[string]any{"attributes": map[string]any{"plus_type": "cal_plus", "status": "active"}}},
		}})
	case method == http.MethodGet && path == "/api/frames/123/auto_creation_intents":
		writeJSON(w, map[string]any{"data": []any{map[string]any{"id": "intent-1", "type": "auto_creation_intent"}}})
	default:
		f.t.Fatalf("unexpected fake Skylight request: %s %s", method, r.URL.RequestURI())
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		panic(err)
	}
}

func fakeFrame() map[string]any {
	return map[string]any{
		"id": "123",
		"attributes": map[string]any{
			"name":           "Kitchen",
			"household_name": "Moss",
			"timezone":       "America/New_York",
			"hardware_model": "skylight",
			"mine":           true,
			"plus":           true,
			"activated":      true,
		},
	}
}

func fakeCategory(id, label string) map[string]any {
	return map[string]any{
		"id": id,
		"attributes": map[string]any{
			"label":                    label,
			"color":                    "#2178AF",
			"selected_for_chore_chart": true,
			"linked_to_profile":        true,
		},
	}
}

func fakeChore(id, summary, status string, upForGrabs bool) map[string]any {
	points := 5
	return map[string]any{
		"id": id,
		"attributes": map[string]any{
			"summary":         summary,
			"description":     "desc",
			"status":          status,
			"start":           "2026-06-10",
			"recurrence_set":  []string{"RRULE:FREQ=DAILY;INTERVAL=1"},
			"reward_points":   points,
			"emoji_icon":      "star",
			"up_for_grabs":    upForGrabs,
			"recurring":       true,
			"position":        1,
			"completed_on":    "2026-06-10",
			"recurring_until": nil,
		},
		"relationships": map[string]any{
			"category": map[string]any{"data": map[string]any{"id": "1"}},
		},
	}
}

func fakeReward(id, name string, redeemed bool) map[string]any {
	var redeemedAt any
	if redeemed {
		redeemedAt = "2026-06-10T12:00:00Z"
	}
	return map[string]any{
		"id": id,
		"attributes": map[string]any{
			"name":                  name,
			"emoji_icon":            "tv",
			"description":           "desc",
			"point_value":           10,
			"respawn_on_redemption": true,
			"redeemed_at":           redeemedAt,
		},
		"relationships": map[string]any{
			"category": map[string]any{"data": map[string]any{"id": "1"}},
		},
	}
}

func fakeCalendarEvent() map[string]any {
	return map[string]any{
		"id": "evt-1",
		"attributes": map[string]any{
			"summary":     "Dentist",
			"starts_at":   "2026-06-10T14:00:00Z",
			"ends_at":     "2026-06-10T15:00:00Z",
			"all_day":     false,
			"color":       "#2178AF",
			"description": "Bring forms",
		},
		"relationships": map[string]any{
			"category": map[string]any{"data": map[string]any{"id": "1"}},
		},
	}
}

func fakeList(id, label, kind string) map[string]any {
	return map[string]any{
		"id":   id,
		"type": "list",
		"attributes": map[string]any{
			"label":           label,
			"color":           "#2178AF",
			"kind":            kind,
			"hide_from_frame": false,
		},
	}
}

func fakeListItem(id, label, status string) map[string]any {
	return map[string]any{
		"id":   id,
		"type": "list_item",
		"attributes": map[string]any{
			"label":    label,
			"status":   status,
			"position": 1,
		},
	}
}

func fakeRecipe() map[string]any {
	return map[string]any{
		"id": "recipe-1",
		"attributes": map[string]any{
			"summary":     "Tacos",
			"description": "Dinner",
			"ingredients": []string{"beans", "rice"},
			"url":         "https://example.com/tacos",
		},
		"relationships": map[string]any{
			"meal_category": map[string]any{"data": map[string]any{"id": "meal-cat-1"}},
		},
	}
}

func fakeSitting() map[string]any {
	return map[string]any{
		"id": "sitting-1",
		"attributes": map[string]any{
			"summary": "Dinner",
			"date":    "2026-06-10",
		},
		"relationships": map[string]any{
			"meal_category": map[string]any{"data": map[string]any{"id": "meal-cat-1"}},
			"meal_recipe":   map[string]any{"data": map[string]any{"id": "recipe-1"}},
		},
	}
}

func fakeRoutine() map[string]any {
	return map[string]any{
		"id": "routine-1",
		"attributes": map[string]any{
			"title":       "Morning",
			"assignee_id": "1",
			"steps":       []string{"Brush", "Pack"},
		},
	}
}

func fakePortableExport() string {
	return fmt.Sprintf(`{
  "exported_at": "%s",
  "frame_id": 123,
  "chores": [{"summary":"Import chore","category_id":1,"start":"2026-06-10","recurrence_set":["RRULE:FREQ=DAILY;INTERVAL=1"]}],
  "rewards": [{"name":"Import reward","point_value":5,"category_ids":[1]}],
  "lists": [{"label":"Import list","kind":"to_do","color":"#2178AF","list_items":[{"label":"Item","status":"pending"}]}],
  "recipes": [{"summary":"Import recipe","ingredients":["beans"],"meal_category_id":"meal-cat-1"}],
  "meal_sittings": [{"summary":"Import dinner","meal_recipe_id":"recipe-1","meal_category_id":"meal-cat-1","date":"2026-06-10"}],
  "calendar_events": [{"summary":"Import event","starts_at":"2026-06-10T14:00:00Z","ends_at":"2026-06-10T15:00:00Z","category_id":"1","description":"desc"}]
}`, time.Now().UTC().Format(time.RFC3339))
}
