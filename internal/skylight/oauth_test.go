package skylight

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestRefreshOAuthTokenPostsRefreshGrant(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			t.Fatalf("path: got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method: got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/x-www-form-urlencoded") {
			t.Fatalf("content-type: got %q", ct)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		checkForm(t, r, "grant_type", "refresh_token")
		checkForm(t, r, "refresh_token", "oldrefresh")
		checkForm(t, r, "client_id", "skylight-mobile")
		checkForm(t, r, "skylight_api_client_device_fingerprint", "fp-1")

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"newaccess","refresh_token":"newrefresh","expires_in":7200,"token_type":"Bearer"}`)
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	tok, err := c.RefreshOAuthToken(context.Background(), "oldrefresh", "fp-1")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if tok.AccessToken != "newaccess" {
		t.Fatalf("access token: got %q", tok.AccessToken)
	}
	if tok.RefreshToken != "newrefresh" {
		t.Fatalf("refresh token: got %q", tok.RefreshToken)
	}
	if tok.ExpiresIn != 7200 {
		t.Fatalf("expires_in: got %d", tok.ExpiresIn)
	}
}

func checkForm(t *testing.T, r *http.Request, key, want string) {
	t.Helper()
	if got := r.FormValue(key); got != want {
		t.Fatalf("%s: got %q, want %q", key, got, want)
	}
}

func TestLoginOAuthHeadlessFlow(t *testing.T) {
	var sawLoginPage, sawSession, sawAuthorize, sawToken bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/session/new":
			sawLoginPage = true
			if r.Method != http.MethodGet {
				t.Fatalf("session new method: %s", r.Method)
			}
			fmt.Fprint(w, `<input type="hidden" name="authenticity_token" value="csrf-123" />`)
		case "/auth/session":
			sawSession = true
			if r.Method != http.MethodPost {
				t.Fatalf("session method: %s", r.Method)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			checkForm(t, r, "authenticity_token", "csrf-123")
			checkForm(t, r, "email", "user@example.com")
			checkForm(t, r, "password", "secret")
			http.Redirect(w, r, "/signed-in", http.StatusFound)
		case "/oauth/authorize":
			sawAuthorize = true
			if r.Method != http.MethodGet {
				t.Fatalf("authorize method: %s", r.Method)
			}
			q := r.URL.Query()
			if q.Get("client_id") != "skylight-mobile" ||
				q.Get("response_type") != "code" ||
				q.Get("redirect_uri") != "https://ourskylight.com/welcome" ||
				q.Get("scope") != "everything" ||
				q.Get("skylight_api_client_device_fingerprint") != "fp-login" {
				t.Fatalf("authorize query = %s", r.URL.RawQuery)
			}
			redirect := "https://ourskylight.com/welcome?" + url.Values{"code": {"auth-code-1"}}.Encode()
			http.Redirect(w, r, redirect, http.StatusFound)
		case "/oauth/token":
			sawToken = true
			if r.Method != http.MethodPost {
				t.Fatalf("token method: %s", r.Method)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			checkForm(t, r, "grant_type", "authorization_code")
			checkForm(t, r, "code", "auth-code-1")
			checkForm(t, r, "client_id", "skylight-mobile")
			checkForm(t, r, "redirect_uri", "https://ourskylight.com/welcome")
			checkForm(t, r, "scope", "everything")
			checkForm(t, r, "skylight_api_client_device_fingerprint", "fp-login")
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"access-login","refresh_token":"refresh-login","expires_in":3600,"token_type":"Bearer"}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	tok, err := c.LoginOAuth(context.Background(), "user@example.com", "secret", "fp-login")
	if err != nil {
		t.Fatalf("LoginOAuth: %v", err)
	}
	if tok.AccessToken != "access-login" || tok.RefreshToken != "refresh-login" || tok.ExpiresIn != 3600 {
		t.Fatalf("token = %+v", tok)
	}
	if !sawLoginPage || !sawSession || !sawAuthorize || !sawToken {
		t.Fatalf("sawLoginPage=%v sawSession=%v sawAuthorize=%v sawToken=%v", sawLoginPage, sawSession, sawAuthorize, sawToken)
	}
}
