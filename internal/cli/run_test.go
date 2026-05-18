package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jwmoss/skycli/internal/config"
)

func TestDoctorUsesEnvAccessToken(t *testing.T) {
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		authHeader = r.Header.Get("Authorization")
		fmt.Fprint(w, `{"data":{"id":"user-1","attributes":{}}}`)
	}))
	defer srv.Close()

	cfgPath := writeTestConfig(t, config.Config{BaseURL: srv.URL})
	t.Setenv("SKYLIGHT_ACCESS_TOKEN", "env-token")

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "doctor"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	if authHeader != "Bearer env-token" {
		t.Fatalf("Authorization: got %q", authHeader)
	}
}

func TestExpiredConfigTokenAutoRefreshes(t *testing.T) {
	var sawRefresh, sawUser bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			sawRefresh = true
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			if got := r.FormValue("refresh_token"); got != "old-refresh" {
				t.Fatalf("refresh_token: got %q", got)
			}
			if got := r.FormValue("skylight_api_client_device_fingerprint"); got != "fp-1" {
				t.Fatalf("fingerprint: got %q", got)
			}
			fmt.Fprint(w, `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`)
		case "/api/user":
			sawUser = true
			if got := r.Header.Get("Authorization"); got != "Bearer new-access" {
				t.Fatalf("Authorization: got %q", got)
			}
			fmt.Fprint(w, `{"data":{"id":"user-1","attributes":{}}}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	cfgPath := writeTestConfig(t, config.Config{
		BaseURL:           srv.URL,
		AuthScheme:        config.DefaultAuthScheme,
		AccessToken:       "old-access",
		RefreshToken:      "old-refresh",
		AccessTokenExpAt:  time.Now().Add(-time.Hour),
		DeviceFingerprint: "fp-1",
	})

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "doctor"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	if !sawRefresh || !sawUser {
		t.Fatalf("sawRefresh=%v sawUser=%v", sawRefresh, sawUser)
	}

	cfg := readTestConfig(t, cfgPath)
	if cfg.AccessToken != "new-access" || cfg.RefreshToken != "new-refresh" {
		t.Fatalf("updated config = %+v", cfg)
	}
	if cfg.AccessTokenExpAt.IsZero() || time.Until(cfg.AccessTokenExpAt) < 55*time.Minute {
		t.Fatalf("access token expiry not updated: %s", cfg.AccessTokenExpAt)
	}
}

func TestAuthRefreshCommandUpdatesConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.FormValue("refresh_token"); got != "cfg-refresh" {
			t.Fatalf("refresh_token: got %q", got)
		}
		if got := r.FormValue("skylight_api_client_device_fingerprint"); got != "cfg-fp" {
			t.Fatalf("fingerprint: got %q", got)
		}
		fmt.Fprint(w, `{"access_token":"cmd-access","refresh_token":"cmd-refresh","expires_in":7200}`)
	}))
	defer srv.Close()

	cfgPath := writeTestConfig(t, config.Config{
		BaseURL:           srv.URL,
		RefreshToken:      "cfg-refresh",
		DeviceFingerprint: "cfg-fp",
	})

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "--json", "auth", "refresh"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	cfg := readTestConfig(t, cfgPath)
	if cfg.AccessToken != "cmd-access" || cfg.RefreshToken != "cmd-refresh" {
		t.Fatalf("updated config = %+v", cfg)
	}
}

func TestAuthLoginCommandSavesOAuthCredentials(t *testing.T) {
	var sawSession, sawAuthorize, sawToken bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/session/new":
			fmt.Fprint(w, `<input name="authenticity_token" value="csrf-login" />`)
		case "/auth/session":
			sawSession = true
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			if got := r.FormValue("email"); got != "user@example.com" {
				t.Fatalf("email: got %q", got)
			}
			if got := r.FormValue("password"); got != "secret-pw" {
				t.Fatalf("password: got %q", got)
			}
			if got := r.FormValue("authenticity_token"); got != "csrf-login" {
				t.Fatalf("csrf: got %q", got)
			}
			http.Redirect(w, r, "/signed-in", http.StatusFound)
		case "/oauth/authorize":
			sawAuthorize = true
			if got := r.URL.Query().Get("skylight_api_client_device_fingerprint"); got != "fp-login" {
				t.Fatalf("fingerprint: got %q", got)
			}
			http.Redirect(w, r, "https://ourskylight.com/welcome?code=code-login", http.StatusFound)
		case "/oauth/token":
			sawToken = true
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			if got := r.FormValue("grant_type"); got != "authorization_code" {
				t.Fatalf("grant_type: got %q", got)
			}
			if got := r.FormValue("code"); got != "code-login" {
				t.Fatalf("code: got %q", got)
			}
			fmt.Fprint(w, `{"access_token":"login-access","refresh_token":"login-refresh","expires_in":3600}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	cfgPath := writeTestConfig(t, config.Config{BaseURL: srv.URL})

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "--json", "auth", "login", "--email", "user@example.com", "--fingerprint", "fp-login", "--password-stdin"}, strings.NewReader("secret-pw\n"), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	if !sawSession || !sawAuthorize || !sawToken {
		t.Fatalf("sawSession=%v sawAuthorize=%v sawToken=%v", sawSession, sawAuthorize, sawToken)
	}
	cfg := readTestConfig(t, cfgPath)
	if cfg.AccessToken != "login-access" || cfg.RefreshToken != "login-refresh" || cfg.DeviceFingerprint != "fp-login" {
		t.Fatalf("updated config = %+v", cfg)
	}
	if cfg.AccessTokenExpAt.IsZero() || time.Until(cfg.AccessTokenExpAt) < 55*time.Minute {
		t.Fatalf("access token expiry not updated: %s", cfg.AccessTokenExpAt)
	}
}

func TestChoresListAfterDoesNotInjectDefaultBefore(t *testing.T) {
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/frames/123/chores" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		query = r.URL.RawQuery
		fmt.Fprint(w, `{"data":[]}`)
	}))
	defer srv.Close()

	cfgPath := writeTestConfig(t, config.Config{
		BaseURL:        srv.URL,
		AuthScheme:     config.DefaultAuthScheme,
		AccessToken:    "tok",
		DefaultFrameID: 123,
	})

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "chores", "list", "--after", "2026-05-15"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	values, err := url.ParseQuery(query)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	if got := values.Get("after"); got != "2026-05-15" {
		t.Fatalf("after: got %q", got)
	}
	if got := values.Get("before"); got != "" {
		t.Fatalf("before should be empty, got %q", got)
	}
}

func TestChoresListStartEndDateAliases(t *testing.T) {
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/frames/123/chores" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		query = r.URL.RawQuery
		fmt.Fprint(w, `{"data":[]}`)
	}))
	defer srv.Close()

	cfgPath := writeTestConfig(t, config.Config{
		BaseURL:        srv.URL,
		AuthScheme:     config.DefaultAuthScheme,
		AccessToken:    "tok",
		DefaultFrameID: 123,
	})

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "chores", "list", "--start-date", "2026-05-15", "--end-date", "2026-05-20"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	values, err := url.ParseQuery(query)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	if got := values.Get("after"); got != "2026-05-15" {
		t.Fatalf("after: got %q", got)
	}
	if got := values.Get("before"); got != "2026-05-20" {
		t.Fatalf("before: got %q", got)
	}
}

func TestFramesDefaultListsFrameIDs(t *testing.T) {
	var requestPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		fmt.Fprint(w, `{"data":[{"id":"123","attributes":{"name":"Kitchen","household_name":"Moss","timezone":"America/New_York","mine":true,"plus":true,"activated":true}}]}`)
	}))
	defer srv.Close()

	cfgPath := writeTestConfig(t, config.Config{
		BaseURL:     srv.URL,
		AuthScheme:  config.DefaultAuthScheme,
		AccessToken: "tok",
	})

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "frames"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	if requestPath != "/api/frames" {
		t.Fatalf("path = %q", requestPath)
	}
	if !strings.Contains(stdout.String(), "123") || !strings.Contains(stdout.String(), "Kitchen") {
		t.Fatalf("stdout missing frame summary: %s", stdout.String())
	}
}

func TestMissingFrameErrorSuggestsDiscovery(t *testing.T) {
	cfgPath := writeTestConfig(t, config.Config{
		BaseURL:     "https://example.invalid",
		AuthScheme:  config.DefaultAuthScheme,
		AccessToken: "tok",
	})

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "categories"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitErr {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "skycli frames") || !strings.Contains(stderr.String(), "set-default") {
		t.Fatalf("stderr missing frame discovery hint: %s", stderr.String())
	}
}

func TestRawUsesAbsoluteURL(t *testing.T) {
	var sawRaw bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/raw" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("x"); got != "1" {
			t.Fatalf("query x: got %q", got)
		}
		sawRaw = true
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	cfgPath := writeTestConfig(t, config.Config{
		BaseURL:     "https://example.invalid",
		AuthScheme:  config.DefaultAuthScheme,
		AccessToken: "tok",
	})

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "raw", srv.URL + "/raw?x=1"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	if !sawRaw {
		t.Fatal("raw endpoint was not called")
	}
}

func TestChoresBulkStopOnErrorReportsAttemptedSuccesses(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "chores.json")
	if err := os.WriteFile(inputPath, []byte(`[{"summary":"","category_id":0},{"summary":"later","category_id":1}]`), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}
	cfgPath := writeTestConfig(t, config.Config{
		BaseURL:        "https://example.invalid",
		AuthScheme:     config.DefaultAuthScheme,
		AccessToken:    "tok",
		DefaultFrameID: 123,
	})

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "chores", "bulk", "--file", inputPath, "--stop-on-error"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitErr {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "done: 0 ok, 1 failed") {
		t.Fatalf("stderr missing attempted count: %s", stderr.String())
	}
}

func TestChoresCreateUpForGrabsCommand(t *testing.T) {
	var requestPath string
	var payload map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		fmt.Fprint(w, `{"data":[{"id":"99","attributes":{"summary":"Bonus","status":"pending","start":"2026-05-18","reward_points":10,"recurring":true,"up_for_grabs":true},"relationships":{"category":{"data":null}}}]}`)
	}))
	defer srv.Close()

	cfgPath := writeTestConfig(t, config.Config{
		BaseURL:        srv.URL,
		AuthScheme:     config.DefaultAuthScheme,
		AccessToken:    "tok",
		DefaultFrameID: 123,
	})

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "chores", "create-up-for-grabs", "--summary", "Bonus", "--points", "10"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	if requestPath != "/api/frames/123/chores/create_multiple" {
		t.Fatalf("path = %q", requestPath)
	}
	if payload["up_for_grabs"] != true {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestChoresUpdateCommand(t *testing.T) {
	var requestPath string
	var payload map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		fmt.Fprint(w, `{"data":{"id":"99","attributes":{"summary":"Mary TV Ticket","status":"pending","start":"2026-05-18","reward_points":1,"recurring":true,"up_for_grabs":false},"relationships":{"category":{"data":{"id":"20431525"}}}}}`)
	}))
	defer srv.Close()

	cfgPath := writeTestConfig(t, config.Config{
		BaseURL:        srv.URL,
		AuthScheme:     config.DefaultAuthScheme,
		AccessToken:    "tok",
		DefaultFrameID: 123,
	})

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "chores", "update", "--id", "99-2026-05-18", "--summary", "Mary TV Ticket", "--points", "1"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	if requestPath != "/api/frames/123/chores/99" {
		t.Fatalf("path = %q", requestPath)
	}
	if payload["summary"] != "Mary TV Ticket" || payload["reward_points"] != float64(1) {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestRewardsUpdateAndRedeemCommands(t *testing.T) {
	var sawUpdate, sawRedeem bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPatch && r.URL.Path == "/api/frames/123/rewards/55":
			sawUpdate = true
			fmt.Fprint(w, `{"data":{"id":"55","attributes":{"name":"Levi TV Ticket","point_value":10,"respawn_on_redemption":true,"redeemed_at":null},"relationships":{"category":{"data":{"id":"20435739"}}}}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/frames/123/rewards/55/redeem":
			sawRedeem = true
			fmt.Fprint(w, `{}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	cfgPath := writeTestConfig(t, config.Config{
		BaseURL:        srv.URL,
		AuthScheme:     config.DefaultAuthScheme,
		AccessToken:    "tok",
		DefaultFrameID: 123,
	})

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "rewards", "update", "--id", "55", "--name", "Levi TV Ticket", "--points", "10", "--respawn"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("update exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"--config", cfgPath, "rewards", "redeem", "--id", "55"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("redeem exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	if !sawUpdate || !sawRedeem {
		t.Fatalf("sawUpdate=%v sawRedeem=%v", sawUpdate, sawRedeem)
	}
}

func TestCalendarCountdownDateAlias(t *testing.T) {
	var payload map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/frames/123/calendar_events" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		fmt.Fprint(w, `{"data":{"id":"cal-1","attributes":{"summary":"Beach trip","starts_at":"2026-07-01","all_day":true}}}`)
	}))
	defer srv.Close()

	cfgPath := writeTestConfig(t, config.Config{
		BaseURL:        srv.URL,
		AuthScheme:     config.DefaultAuthScheme,
		AccessToken:    "tok",
		DefaultFrameID: 123,
	})

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "calendar", "create-countdown", "--title", "Beach trip", "--date", "2026-07-01"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	if payload["starts_at"] != "2026-07-01" || payload["all_day"] != true || payload["event_type"] != "countdown" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestGroceryAddMultipleItems(t *testing.T) {
	var labels []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/frames/123/lists/456/list_items" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		labels = append(labels, fmt.Sprint(payload["label"]))
		fmt.Fprint(w, `{"data":{"id":"item","attributes":{"label":"ok","status":"pending"}}}`)
	}))
	defer srv.Close()

	cfgPath := writeTestConfig(t, config.Config{
		BaseURL:        srv.URL,
		AuthScheme:     config.DefaultAuthScheme,
		AccessToken:    "tok",
		DefaultFrameID: 123,
	})

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "grocery", "add", "--list-id", "456", "--items", "Milk,Eggs"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	if strings.Join(labels, ",") != "Milk,Eggs" {
		t.Fatalf("labels = %#v", labels)
	}
}

func TestReadonlyBlocksMutatingCommands(t *testing.T) {
	cfgPath := writeTestConfig(t, config.Config{})

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "--readonly", "chores", "create", "--summary", "Nope"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitErr {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "readonly mode blocks mutating command") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestAllowCommandsBlocksUnexpectedCommand(t *testing.T) {
	cfgPath := writeTestConfig(t, config.Config{})

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "--allow-commands", "auth status", "version"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitErr {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "not allowed") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestFileSecretsBackendKeepsTokenOutOfConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv(secretsEnvFileKey, "test-secret-key")

	cfgPath := filepath.Join(home, "config.json")
	if err := config.Save(cfgPath, &config.Config{SecretsBackend: secretsBackendFile}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--config", cfgPath, "auth", "set-token"}, strings.NewReader("Bearer secret-token\n"), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	cfg := readTestConfig(t, cfgPath)
	if cfg.AccessToken != "" || cfg.RefreshToken != "" {
		t.Fatalf("tokens leaked into config: %+v", cfg)
	}
	path, err := fileSecretsPath()
	if err != nil {
		t.Fatalf("fileSecretsPath: %v", err)
	}
	encrypted, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read secrets file: %v", err)
	}
	if strings.Contains(string(encrypted), "secret-token") {
		t.Fatalf("secret file contains plaintext token: %s", string(encrypted))
	}
	secrets, err := readFileSecrets()
	if err != nil {
		t.Fatalf("readFileSecrets: %v", err)
	}
	if secrets.AccessToken != "secret-token" {
		t.Fatalf("secret token = %q", secrets.AccessToken)
	}
}

func writeTestConfig(t *testing.T, cfg config.Config) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if cfg.APIVersion == "" {
		cfg.APIVersion = config.DefaultAPIVersion
	}
	if cfg.AuthScheme == "" {
		cfg.AuthScheme = config.DefaultAuthScheme
	}
	if cfg.SecretsBackend == "" {
		cfg.SecretsBackend = secretsBackendConfig
	}
	if err := config.Save(path, &cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	return path
}

func readTestConfig(t *testing.T, path string) config.Config {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse config: %v", err)
	}
	return cfg
}
