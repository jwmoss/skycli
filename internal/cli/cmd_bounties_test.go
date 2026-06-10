package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jwmoss/skycli/internal/config"
)

// A bad --reward-id must be rejected before any chore mutation is sent, so a
// validation error can never leave a half-applied bounty.
func TestBountiesUpdateValidatesRewardIDBeforeMutating(t *testing.T) {
	var sawMutation bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			sawMutation = true
		}
		w.WriteHeader(http.StatusOK)
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
		[]string{"--config", cfgPath, "bounties", "update", "--frame", "123", "--chore-id", "5", "--reward-id", "not-a-number"},
		strings.NewReader(""), &stdout, &stderr)
	if code == exitOK {
		t.Fatalf("expected failure for bad reward-id, got exitOK\nstderr=%s", stderr.String())
	}
	if sawMutation {
		t.Fatal("chore was mutated before reward-id validation failed")
	}
}

func TestBountiesDeleteValidatesRewardIDBeforeMutating(t *testing.T) {
	var sawMutation bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			sawMutation = true
		}
		w.WriteHeader(http.StatusOK)
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
		[]string{"--config", cfgPath, "bounties", "delete", "--frame", "123", "--chore-id", "5", "--reward-id", "not-a-number"},
		strings.NewReader(""), &stdout, &stderr)
	if code == exitOK {
		t.Fatalf("expected failure for bad reward-id, got exitOK\nstderr=%s", stderr.String())
	}
	if sawMutation {
		t.Fatal("chore was deleted before reward-id validation failed")
	}
}

func TestBountiesUpdateReportsPartialWhenRewardUpdateFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/api/frames/123/chores/5":
			fmt.Fprint(w, `{"data":{"id":"5","attributes":{"summary":"Garage","reward_points":10}}}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/frames/123/rewards/9":
			http.Error(w, `{"message":"reward failed"}`, http.StatusInternalServerError)
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
	code := Run(context.Background(),
		[]string{"--config", cfgPath, "--json", "bounties", "update", "--frame", "123", "--chore-id", "5", "--reward-id", "9", "--title", "Garage"},
		strings.NewReader(""), &stdout, &stderr)
	if code != exitErr {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("parse stdout: %v\n%s", err, stdout.String())
	}
	if got["partial"] != true || got["operation"] != "update" {
		t.Fatalf("partial output = %#v", got)
	}
	applied, ok := got["applied"].(map[string]any)
	if !ok || applied["chore"] == nil {
		t.Fatalf("missing applied chore: %#v", got)
	}
}

func TestBountiesDeleteReportsPartialWhenRewardDeleteFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == "/api/frames/123/chores/5":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/frames/123/rewards/9":
			http.Error(w, `{"message":"reward failed"}`, http.StatusInternalServerError)
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
	code := Run(context.Background(),
		[]string{"--config", cfgPath, "--json", "bounties", "delete", "--frame", "123", "--chore-id", "5", "--reward-id", "9"},
		strings.NewReader(""), &stdout, &stderr)
	if code != exitErr {
		t.Fatalf("exit code: got %d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
	}
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("parse stdout: %v\n%s", err, stdout.String())
	}
	if got["partial"] != true || got["operation"] != "delete" {
		t.Fatalf("partial output = %#v", got)
	}
	applied, ok := got["applied"].(map[string]any)
	if !ok || applied["deleted_chore"] != "5" {
		t.Fatalf("missing applied chore delete: %#v", got)
	}
}
