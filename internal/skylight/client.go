package skylight

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	apiVersion string
	authScheme string
	token      string
	http       *http.Client
	userAgent  string
	trace      func(method, url string, status int, dur time.Duration)
	dryRun     bool
	readOnly   bool
}

type Option func(*Client)

func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.http.Timeout = d }
}

func WithTrace(fn func(method, url string, status int, dur time.Duration)) Option {
	return func(c *Client) { c.trace = fn }
}

func WithDryRun(on bool) Option {
	return func(c *Client) { c.dryRun = on }
}

// WithReadOnly makes Do refuse non-GET requests. This backstops the CLI-level
// command allowlist so --readonly cannot be bypassed by argv parsing drift.
func WithReadOnly(on bool) Option {
	return func(c *Client) { c.readOnly = on }
}

func WithUserAgent(ua string) Option {
	return func(c *Client) { c.userAgent = ua }
}

func WithAPIVersion(v string) Option {
	return func(c *Client) { c.apiVersion = v }
}

func WithAuthScheme(s string) Option {
	return func(c *Client) {
		if s != "" {
			c.authScheme = s
		}
	}
}

func New(baseURL, token string, opts ...Option) *Client {
	c := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiVersion: "2026-04-15",
		authScheme: "Bearer",
		token:      token,
		http:       &http.Client{Timeout: 30 * time.Second},
		userAgent:  "skycli/0.1",
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

type APIError struct {
	Status  int
	Method  string
	Path    string
	Body    []byte
	Message string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("skylight %s %s: HTTP %d: %s", e.Method, e.Path, e.Status, e.Message)
	}
	return fmt.Sprintf("skylight %s %s: HTTP %d", e.Method, e.Path, e.Status)
}

func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body any) ([]byte, error) {
	if method != http.MethodGet {
		if c.dryRun {
			return nil, fmt.Errorf("dry-run: refusing %s %s", method, path)
		}
		if c.readOnly {
			return nil, fmt.Errorf("readonly: refusing %s %s", method, path)
		}
	}
	u := path
	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		u = c.baseURL + path
	}
	if len(query) > 0 {
		sep := "?"
		if strings.Contains(u, "?") {
			sep = "&"
		}
		u += sep + query.Encode()
	}
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return nil, err
	}
	if sameOrigin(c.baseURL, u) {
		req.Header.Set("Authorization", c.authScheme+" "+c.token)
		req.Header.Set("skylight-api-version", c.apiVersion)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
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
		c.trace(method, u, resp.StatusCode, time.Since(start))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return data, &APIError{
			Status:  resp.StatusCode,
			Method:  method,
			Path:    path,
			Body:    data,
			Message: extractErrorMessage(data),
		}
	}
	return data, nil
}

// sameOrigin reports whether reqURL targets the same scheme+host as the
// configured Skylight base URL. Off-origin absolute URLs (e.g. an arbitrary
// target passed to `skycli raw`) must not receive the user's bearer token.
func sameOrigin(baseURL, reqURL string) bool {
	b, err := url.Parse(baseURL)
	if err != nil {
		return false
	}
	r, err := url.Parse(reqURL)
	if err != nil {
		return false
	}
	return strings.EqualFold(b.Scheme, r.Scheme) && strings.EqualFold(b.Host, r.Host)
}

func extractErrorMessage(data []byte) string {
	var generic map[string]any
	if err := json.Unmarshal(data, &generic); err != nil {
		return strings.TrimSpace(string(data))
	}
	if e, ok := generic["errors"]; ok {
		switch v := e.(type) {
		case []any:
			parts := make([]string, 0, len(v))
			for _, x := range v {
				parts = append(parts, fmt.Sprint(x))
			}
			return strings.Join(parts, "; ")
		case map[string]any:
			parts := make([]string, 0, len(v))
			for k, x := range v {
				parts = append(parts, fmt.Sprintf("%s: %v", k, x))
			}
			return strings.Join(parts, "; ")
		}
	}
	if m, ok := generic["message"].(string); ok {
		return m
	}
	if m, ok := generic["error"].(string); ok {
		return m
	}
	return strings.TrimSpace(string(data))
}

// ---- Typed responses ----

type User struct {
	ID         string `json:"id"`
	Attributes struct {
		Email              string `json:"email"`
		SubscriptionStatus string `json:"subscription_status"`
		Profile            struct {
			ID int64 `json:"id"`
		} `json:"profile"`
	} `json:"attributes"`
}

type userEnvelope struct {
	Data User `json:"data"`
}

func (c *Client) GetUser(ctx context.Context) (*User, error) {
	data, err := c.Do(ctx, http.MethodGet, "/api/user", nil, nil)
	if err != nil {
		return nil, err
	}
	var env userEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

type Frame struct {
	ID         string `json:"id"`
	Attributes struct {
		Name          string `json:"name"`
		HouseholdName string `json:"household_name"`
		Timezone      string `json:"timezone"`
		HardwareModel string `json:"hardware_model"`
		Mine          bool   `json:"mine"`
		Plus          bool   `json:"plus"`
		Activated     bool   `json:"activated"`
	} `json:"attributes"`
}

type frameEnvelope struct {
	Data Frame `json:"data"`
}

type framesEnvelope struct {
	Data []Frame `json:"data"`
}

func (c *Client) ListFrames(ctx context.Context) ([]Frame, error) {
	data, err := c.Do(ctx, http.MethodGet, "/api/frames", nil, nil)
	if err != nil {
		return nil, err
	}
	var env framesEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

func (c *Client) GetFrame(ctx context.Context, frameID int64) (*Frame, error) {
	data, err := c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d", frameID), nil, nil)
	if err != nil {
		return nil, err
	}
	var env frameEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

type Category struct {
	ID         string `json:"id"`
	Attributes struct {
		Color                 string `json:"color"`
		Label                 string `json:"label"`
		SelectedForChoreChart bool   `json:"selected_for_chore_chart"`
		LinkedToProfile       bool   `json:"linked_to_profile"`
	} `json:"attributes"`
}

type categoriesEnvelope struct {
	Data []Category `json:"data"`
}

func (c *Client) ListCategories(ctx context.Context, frameID int64) ([]Category, error) {
	data, err := c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/categories", frameID), nil, nil)
	if err != nil {
		return nil, err
	}
	var env categoriesEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

type Chore struct {
	ID         string `json:"id"`
	Attributes struct {
		Summary        string   `json:"summary"`
		Description    *string  `json:"description"`
		Status         string   `json:"status"`
		Start          string   `json:"start"`
		RecurrenceSet  []string `json:"recurrence_set"`
		RecurringUntil *string  `json:"recurring_until"`
		CompletedOn    *string  `json:"completed_on"`
		RewardPoints   *int     `json:"reward_points"`
		EmojiIcon      *string  `json:"emoji_icon"`
		UpForGrabs     bool     `json:"up_for_grabs"`
		Recurring      bool     `json:"recurring"`
		Position       int      `json:"position"`
	} `json:"attributes"`
	Relationships struct {
		Category struct {
			Data *struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"category"`
	} `json:"relationships"`
}

type choresListEnvelope struct {
	Data []Chore `json:"data"`
}

type choreEnvelope struct {
	Data Chore `json:"data"`
}

type ChoreFilter struct {
	Date              string
	Status            string
	AssigneeID        string
	After             string
	Before            string
	IncludeLate       bool
	IncludeUpForGrabs bool
	OnlyUpForGrabs    bool
	LinkedToProfile   bool
}

func (c *Client) ListChores(ctx context.Context, frameID int64, f ChoreFilter) ([]Chore, error) {
	q := url.Values{}
	if f.Date != "" {
		q.Set("date", f.Date)
	}
	if f.Status != "" {
		q.Set("status", f.Status)
	}
	if f.AssigneeID != "" {
		q.Set("assignee_id", f.AssigneeID)
	}
	if f.After != "" {
		q.Set("after", f.After)
	}
	if f.Before != "" {
		q.Set("before", f.Before)
	}
	if f.IncludeLate {
		q.Set("include_late", "true")
	}
	if f.IncludeUpForGrabs {
		q.Set("include_up_for_grabs", "true")
	}
	if f.LinkedToProfile {
		q.Set("filter", "linked_to_profile")
	}
	data, err := c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/chores", frameID), q, nil)
	if err != nil {
		return nil, err
	}
	var env choresListEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	if f.OnlyUpForGrabs {
		filtered := make([]Chore, 0, len(env.Data))
		for _, ch := range env.Data {
			if ch.Attributes.UpForGrabs {
				filtered = append(filtered, ch)
			}
		}
		return filtered, nil
	}
	return env.Data, nil
}

type ChoreCreate struct {
	Summary       string   `json:"summary"`
	CategoryID    int64    `json:"category_id,omitempty"`
	Start         string   `json:"start"`
	RecurrenceSet []string `json:"recurrence_set,omitempty"`
	UpForGrabs    bool     `json:"up_for_grabs"`
	RewardPoints  *int     `json:"reward_points,omitempty"`
	Description   string   `json:"description,omitempty"`
	EmojiIcon     string   `json:"emoji_icon,omitempty"`
}

func (c *Client) CreateChore(ctx context.Context, frameID int64, in ChoreCreate) (*Chore, error) {
	data, err := c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/chores", frameID), nil, in)
	if err != nil {
		return nil, err
	}
	var env choreEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) CreateUpForGrabsChore(ctx context.Context, frameID int64, in ChoreCreate) (*Chore, error) {
	in.CategoryID = 0
	in.UpForGrabs = true
	data, err := c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/chores/create_multiple", frameID), nil, in)
	if err != nil {
		return nil, err
	}
	var env choresListEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	if len(env.Data) == 0 {
		return nil, fmt.Errorf("create up-for-grabs chore: empty response")
	}
	return &env.Data[0], nil
}

type ChoreUpdate struct {
	Summary      *string `json:"summary,omitempty"`
	CategoryID   *int64  `json:"category_id,omitempty"`
	Start        *string `json:"start,omitempty"`
	Status       *string `json:"status,omitempty"`
	RewardPoints *int    `json:"reward_points,omitempty"`
	UpForGrabs   *bool   `json:"up_for_grabs,omitempty"`
}

func (c *Client) UpdateChore(ctx context.Context, frameID int64, choreID string, in ChoreUpdate) (*Chore, error) {
	baseID, _ := SplitChoreInstanceID(choreID)
	data, err := c.Do(ctx, http.MethodPut, fmt.Sprintf("/api/frames/%d/chores/%s", frameID, baseID), nil, in)
	if err != nil {
		return nil, err
	}
	var env choreEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) DeleteChore(ctx context.Context, frameID, choreID int64, applyTo string) error {
	q := url.Values{}
	q.Set("apply_to", applyTo)
	_, err := c.Do(ctx, http.MethodDelete, fmt.Sprintf("/api/frames/%d/chores/%d", frameID, choreID), q, nil)
	return err
}

type ChoreCompletion struct {
	Status       string `json:"status"`
	InstanceDate string `json:"instance_date,omitempty"`
}

func (c *Client) SetChoreCompletion(ctx context.Context, frameID int64, choreID string, status string) (*Chore, error) {
	baseID, instanceDate := SplitChoreInstanceID(choreID)
	data, err := c.Do(ctx, http.MethodPut, fmt.Sprintf("/api/frames/%d/chores/%s/completions", frameID, baseID), nil, ChoreCompletion{
		Status:       status,
		InstanceDate: instanceDate,
	})
	if err != nil {
		return nil, err
	}
	var env choreEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) ClaimChore(ctx context.Context, frameID int64, choreID string, categoryID int64) (*Chore, error) {
	return c.UpdateChore(ctx, frameID, choreID, ChoreUpdate{CategoryID: &categoryID})
}

// ---- Rewards ----

type Reward struct {
	ID         string `json:"id"`
	Attributes struct {
		Name                string  `json:"name"`
		EmojiIcon           *string `json:"emoji_icon"`
		Description         *string `json:"description"`
		PointValue          int     `json:"point_value"`
		RespawnOnRedemption bool    `json:"respawn_on_redemption"`
		RedeemedAt          *string `json:"redeemed_at"`
	} `json:"attributes"`
	Relationships struct {
		Category struct {
			Data *struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"category"`
	} `json:"relationships"`
}

type rewardsListEnvelope struct {
	Data []Reward `json:"data"`
}

func (c *Client) ListRewards(ctx context.Context, frameID int64) ([]Reward, error) {
	data, err := c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/rewards", frameID), nil, nil)
	if err != nil {
		return nil, err
	}
	var env rewardsListEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

type RewardCreate struct {
	Name                string  `json:"name"`
	PointValue          int     `json:"point_value"`
	CategoryIDs         []int64 `json:"category_ids"`
	EmojiIcon           string  `json:"emoji_icon,omitempty"`
	Description         string  `json:"description,omitempty"`
	RespawnOnRedemption bool    `json:"respawn_on_redemption,omitempty"`
}

// CreateRewards creates one reward per category_id; Skylight returns the array.
func (c *Client) CreateRewards(ctx context.Context, frameID int64, in RewardCreate) ([]Reward, error) {
	data, err := c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/rewards", frameID), nil, in)
	if err != nil {
		return nil, err
	}
	var env rewardsListEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

type RewardUpdate struct {
	Name                *string `json:"name,omitempty"`
	PointValue          *int    `json:"point_value,omitempty"`
	EmojiIcon           *string `json:"emoji_icon,omitempty"`
	Description         *string `json:"description,omitempty"`
	RespawnOnRedemption *bool   `json:"respawn_on_redemption,omitempty"`
}

func (c *Client) UpdateReward(ctx context.Context, frameID, rewardID int64, in RewardUpdate) (*Reward, error) {
	data, err := c.Do(ctx, http.MethodPatch, fmt.Sprintf("/api/frames/%d/rewards/%d", frameID, rewardID), nil, in)
	if err != nil {
		return nil, err
	}
	var env struct {
		Data Reward `json:"data"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) DeleteReward(ctx context.Context, frameID, rewardID int64) error {
	_, err := c.Do(ctx, http.MethodDelete, fmt.Sprintf("/api/frames/%d/rewards/%d", frameID, rewardID), nil, nil)
	return err
}

func (c *Client) RedeemReward(ctx context.Context, frameID, rewardID int64) error {
	_, err := c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/rewards/%d/redeem", frameID, rewardID), nil, nil)
	return err
}

func (c *Client) UnredeemReward(ctx context.Context, frameID, rewardID int64) error {
	_, err := c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/rewards/%d/unredeem", frameID, rewardID), nil, nil)
	return err
}

type RewardPoint struct {
	CategoryID           int64 `json:"category_id"`
	CurrentPointBalance  int   `json:"current_point_balance"`
	LifetimePointsEarned int   `json:"lifetime_points_earned"`
}

func (c *Client) ListRewardPoints(ctx context.Context, frameID int64) ([]RewardPoint, error) {
	data, err := c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/reward_points", frameID), nil, nil)
	if err != nil {
		return nil, err
	}
	var arr []RewardPoint
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, err
	}
	return arr, nil
}

// ---- helpers ----

func ParseID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

var choreInstanceIDRe = regexp.MustCompile(`^(\d+)-(\d{4}-\d{2}-\d{2})`)

func SplitChoreInstanceID(choreID string) (baseID, instanceDate string) {
	if m := choreInstanceIDRe.FindStringSubmatch(choreID); m != nil {
		return m[1], m[2]
	}
	return choreID, ""
}
