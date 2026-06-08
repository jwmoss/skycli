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
