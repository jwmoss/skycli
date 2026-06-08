package cli

import (
	"bytes"
	"context"
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
