package skylight

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	skylightOAuthClientID    = "skylight-mobile"
	skylightOAuthScope       = "everything"
	skylightOAuthRedirectURI = "https://ourskylight.com/welcome"
	browserUserAgent         = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
	browserAccept            = "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
)

type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func (c *Client) RefreshOAuthToken(ctx context.Context, refreshToken, fingerprint string) (*OAuthTokenResponse, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, fmt.Errorf("refresh token is required")
	}
	if strings.TrimSpace(fingerprint) == "" {
		return nil, fmt.Errorf("device fingerprint is required")
	}

	form := url.Values{
		"grant_type":                             {"refresh_token"},
		"refresh_token":                          {refreshToken},
		"client_id":                              {skylightOAuthClientID},
		"skylight_api_client_device_fingerprint": {fingerprint},
	}
	endpoint := c.baseURL + "/oauth/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", c.userAgent)

	start := time.Now()
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if c.trace != nil {
		c.trace(req.Method, endpoint, resp.StatusCode, time.Since(start))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{
			Status:  resp.StatusCode,
			Method:  req.Method,
			Path:    "/oauth/token",
			Body:    data,
			Message: extractErrorMessage(data),
		}
	}

	var token OAuthTokenResponse
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("oauth response missing access_token")
	}
	return &token, nil
}

func (c *Client) LoginOAuth(ctx context.Context, email, password, fingerprint string) (*OAuthTokenResponse, error) {
	if strings.TrimSpace(email) == "" {
		return nil, fmt.Errorf("email is required")
	}
	if strings.TrimSpace(password) == "" {
		return nil, fmt.Errorf("password is required")
	}
	if strings.TrimSpace(fingerprint) == "" {
		return nil, fmt.Errorf("device fingerprint is required")
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	hc := &http.Client{
		Jar:     jar,
		Timeout: c.http.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	csrf, err := c.fetchLoginCSRF(ctx, hc)
	if err != nil {
		return nil, fmt.Errorf("fetch login csrf token: %w", err)
	}
	if err := c.postLoginSession(ctx, hc, email, password, csrf); err != nil {
		return nil, fmt.Errorf("post login session: %w", err)
	}
	code, err := c.fetchOAuthCode(ctx, hc, fingerprint)
	if err != nil {
		return nil, fmt.Errorf("fetch oauth code: %w", err)
	}
	return c.exchangeOAuthCode(ctx, code, fingerprint)
}

func (c *Client) browserRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", browserAccept)
	req.Header.Set("User-Agent", browserUserAgent)
	return req, nil
}

func (c *Client) fetchLoginCSRF(ctx context.Context, hc *http.Client) (string, error) {
	endpoint := c.baseURL + "/auth/session/new"
	req, err := c.browserRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	start := time.Now()
	resp, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if c.trace != nil {
		c.trace(req.Method, endpoint, resp.StatusCode, time.Since(start))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &APIError{Status: resp.StatusCode, Method: req.Method, Path: "/auth/session/new", Body: data, Message: extractErrorMessage(data)}
	}
	for _, re := range []*regexp.Regexp{
		regexp.MustCompile(`name="authenticity_token"[^>]*value="([^"]+)"`),
		regexp.MustCompile(`value="([^"]+)"[^>]*name="authenticity_token"`),
	} {
		if match := re.FindSubmatch(data); len(match) > 1 {
			return string(match[1]), nil
		}
	}
	return "", fmt.Errorf("authenticity_token not found")
}

func (c *Client) postLoginSession(ctx context.Context, hc *http.Client, email, password, csrf string) error {
	endpoint := c.baseURL + "/auth/session"
	form := url.Values{
		"authenticity_token": {csrf},
		"email":              {email},
		"password":           {password},
	}
	req, err := c.browserRequest(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", c.baseURL+"/auth/session/new")
	req.Header.Set("Origin", c.baseURL)
	start := time.Now()
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if c.trace != nil {
		c.trace(req.Method, endpoint, resp.StatusCode, time.Since(start))
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther {
		return &APIError{Status: resp.StatusCode, Method: req.Method, Path: "/auth/session", Body: data, Message: extractErrorMessage(data)}
	}
	return nil
}

func (c *Client) fetchOAuthCode(ctx context.Context, hc *http.Client, fingerprint string) (string, error) {
	q := url.Values{
		"client_id":                              {skylightOAuthClientID},
		"response_type":                          {"code"},
		"redirect_uri":                           {skylightOAuthRedirectURI},
		"scope":                                  {skylightOAuthScope},
		"skylight_api_client_device_fingerprint": {fingerprint},
	}
	endpoint := c.baseURL + "/oauth/authorize?" + q.Encode()
	req, err := c.browserRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	start := time.Now()
	resp, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if c.trace != nil {
		c.trace(req.Method, endpoint, resp.StatusCode, time.Since(start))
	}
	location := resp.Header.Get("Location")
	if location == "" {
		return "", &APIError{Status: resp.StatusCode, Method: req.Method, Path: "/oauth/authorize", Body: data, Message: "missing Location header"}
	}
	u, err := url.Parse(location)
	if err != nil {
		return "", fmt.Errorf("parse oauth redirect: %w", err)
	}
	code := u.Query().Get("code")
	if code == "" {
		return "", fmt.Errorf("oauth redirect missing code: %s", location)
	}
	return code, nil
}

func (c *Client) exchangeOAuthCode(ctx context.Context, code, fingerprint string) (*OAuthTokenResponse, error) {
	form := url.Values{
		"grant_type":                             {"authorization_code"},
		"code":                                   {code},
		"client_id":                              {skylightOAuthClientID},
		"redirect_uri":                           {skylightOAuthRedirectURI},
		"scope":                                  {skylightOAuthScope},
		"skylight_api_client_device_fingerprint": {fingerprint},
	}
	endpoint := c.baseURL + "/oauth/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", c.userAgent)
	start := time.Now()
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if c.trace != nil {
		c.trace(req.Method, endpoint, resp.StatusCode, time.Since(start))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{Status: resp.StatusCode, Method: req.Method, Path: "/oauth/token", Body: data, Message: extractErrorMessage(data)}
	}
	var token OAuthTokenResponse
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("oauth response missing access_token")
	}
	return &token, nil
}
