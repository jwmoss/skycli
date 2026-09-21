package skylight

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRejectCrossOriginRedirects(t *testing.T) {
	for _, kind := range []string{"api", "refresh", "exchange"} {
		t.Run(kind, func(t *testing.T) {
			called := false
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				_, _ = fmt.Fprint(w, `{"access_token":"fixture"}`)
			}))
			defer target.Close()
			origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
			}))
			defer origin.Close()
			c := New(origin.URL, "fixture")
			var err error
			switch kind {
			case "api":
				_, err = c.Do(context.Background(), http.MethodGet, "/api/user", nil, nil)
			case "refresh":
				_, err = c.RefreshOAuthToken(context.Background(), "fixture", "device")
			case "exchange":
				_, err = c.exchangeOAuthCode(context.Background(), "fixture", "device")
			}
			if err == nil || called {
				t.Fatalf("redirect must fail before contacting another origin: error=%v contacted=%v", err != nil, called)
			}
		})
	}
}
