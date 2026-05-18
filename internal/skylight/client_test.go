package skylight

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestDoUsesAbsoluteURLWithoutPrependingBase(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/raw" {
			t.Fatalf("path: got %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("x"); got != "1" {
			t.Fatalf("query x: got %q", got)
		}
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	c := New("https://example.invalid", "tok")
	data, err := c.Do(context.Background(), http.MethodGet, srv.URL+"/raw", url.Values{"x": {"1"}}, nil)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if string(data) != `{"ok":true}` {
		t.Fatalf("body: got %s", data)
	}
}

func TestCreateUpForGrabsChoreUsesCreateMultiple(t *testing.T) {
	var payload map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/frames/123/chores/create_multiple" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		fmt.Fprint(w, `{"data":[{"id":"99","attributes":{"summary":"Bonus","status":"pending","start":"2026-05-18","reward_points":10,"recurring":true,"up_for_grabs":true},"relationships":{"category":{"data":null}}}]}`)
	}))
	defer srv.Close()

	points := 10
	c := New(srv.URL, "tok")
	chore, err := c.CreateUpForGrabsChore(context.Background(), 123, ChoreCreate{
		Summary:       "Bonus",
		CategoryID:    456,
		Start:         "2026-05-18",
		RecurrenceSet: []string{"RRULE:FREQ=DAILY;INTERVAL=1"},
		RewardPoints:  &points,
	})
	if err != nil {
		t.Fatalf("CreateUpForGrabsChore: %v", err)
	}
	if chore.ID != "99" || !chore.Attributes.UpForGrabs {
		t.Fatalf("chore = %+v", chore)
	}
	if payload["up_for_grabs"] != true {
		t.Fatalf("up_for_grabs payload = %v", payload["up_for_grabs"])
	}
	if _, ok := payload["category_id"]; ok {
		t.Fatalf("category_id should be omitted for create_multiple: %#v", payload)
	}
}

func TestUpdateChoreUsesBaseInstanceID(t *testing.T) {
	var payload map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/frames/123/chores/99" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		fmt.Fprint(w, `{"data":{"id":"99","attributes":{"summary":"TV Ticket","status":"pending","start":"2026-05-18","reward_points":1,"recurring":true,"up_for_grabs":false},"relationships":{"category":{"data":{"id":"20431525"}}}}}`)
	}))
	defer srv.Close()

	summary := "TV Ticket"
	points := 1
	categoryID := int64(20431525)
	c := New(srv.URL, "tok")
	chore, err := c.UpdateChore(context.Background(), 123, "99-2026-05-18", ChoreUpdate{
		Summary:      &summary,
		CategoryID:   &categoryID,
		RewardPoints: &points,
	})
	if err != nil {
		t.Fatalf("UpdateChore: %v", err)
	}
	if chore.Attributes.Summary != "TV Ticket" {
		t.Fatalf("summary = %q", chore.Attributes.Summary)
	}
	if payload["summary"] != "TV Ticket" || payload["reward_points"] != float64(1) || payload["category_id"] != float64(20431525) {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestSetChoreCompletionSendsInstanceDate(t *testing.T) {
	var payload map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/frames/123/chores/99/completions" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		fmt.Fprint(w, `{"data":{"id":"99-2026-05-18","attributes":{"summary":"Bonus","status":"complete","start":"2026-05-18","reward_points":10,"recurring":true,"up_for_grabs":false},"relationships":{"category":{"data":{"id":"20435739"}}}}}`)
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	chore, err := c.SetChoreCompletion(context.Background(), 123, "99-2026-05-18", "complete")
	if err != nil {
		t.Fatalf("SetChoreCompletion: %v", err)
	}
	if chore.Attributes.Status != "complete" {
		t.Fatalf("status = %q", chore.Attributes.Status)
	}
	if payload["status"] != "complete" || payload["instance_date"] != "2026-05-18" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestRewardUpdateAndRedeemEndpoints(t *testing.T) {
	var sawUpdate, sawRedeem bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPatch && r.URL.Path == "/api/frames/123/rewards/55":
			sawUpdate = true
			fmt.Fprint(w, `{"data":{"id":"55","attributes":{"name":"TV Ticket","point_value":10,"respawn_on_redemption":true,"redeemed_at":null},"relationships":{"category":{"data":{"id":"20431525"}}}}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/frames/123/rewards/55/redeem":
			sawRedeem = true
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	name := "TV Ticket"
	c := New(srv.URL, "tok")
	reward, err := c.UpdateReward(context.Background(), 123, 55, RewardUpdate{Name: &name})
	if err != nil {
		t.Fatalf("UpdateReward: %v", err)
	}
	if reward.Attributes.Name != "TV Ticket" {
		t.Fatalf("reward name = %q", reward.Attributes.Name)
	}
	if err := c.RedeemReward(context.Background(), 123, 55); err != nil {
		t.Fatalf("RedeemReward: %v", err)
	}
	if !sawUpdate || !sawRedeem {
		t.Fatalf("sawUpdate=%v sawRedeem=%v", sawUpdate, sawRedeem)
	}
}
