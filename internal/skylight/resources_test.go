package skylight

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResourceCollectionsOwnPathsQueriesAndDecoding(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/frames/123/calendar_events":
			if got := r.URL.Query().Get("date_min"); got != "2026-07-01" {
				t.Fatalf("calendar date_min = %q", got)
			}
			if got := r.URL.Query().Get("date_max"); got != "2026-07-07" {
				t.Fatalf("calendar date_max = %q", got)
			}
			fmt.Fprint(w, `{"data":[{"id":"event-1","attributes":{"summary":"Camp"}}],"meta":{"page":1}}`)
		case "/api/frames/123/lists":
			fmt.Fprint(w, `{"data":[{"id":"list-1","type":"list","attributes":{"label":"Groceries","kind":"grocery"}}]}`)
		case "/api/frames/123/lists/list-1":
			fmt.Fprint(w, `{"data":{"id":"list-1","type":"list","attributes":{"label":"Groceries"}},"included":[{"id":"item-1","type":"list_item","attributes":{"label":"Milk","status":"pending"}},{"id":"other-1","type":"other","attributes":{"label":"ignore"}}]}`)
		case "/api/frames/123/meals/recipes":
			fmt.Fprint(w, `{"data":[{"id":"recipe-1","attributes":{"summary":"Tacos"}}]}`)
		case "/api/frames/123/meals/sittings":
			if got := r.URL.Query().Get("date_min"); got != "2026-07-01" {
				t.Fatalf("sittings date_min = %q", got)
			}
			fmt.Fprint(w, `{"data":[{"id":"sitting-1","attributes":{"summary":"Dinner","date":"2026-07-01"}}]}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.RequestURI())
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	ctx := context.Background()
	events, err := c.ListCalendarEvents(ctx, 123, CalendarEventFilter{StartDate: "2026-07-01", EndDate: "2026-07-07"})
	if err != nil || len(events.Data) != 1 || events.Data[0].ID != "event-1" {
		t.Fatalf("ListCalendarEvents: events=%+v err=%v", events, err)
	}
	raw, err := json.Marshal(events)
	if err != nil || !strings.Contains(string(raw), `"meta":{"page":1}`) {
		t.Fatalf("calendar document did not preserve raw metadata: %s err=%v", raw, err)
	}
	lists, err := c.ListLists(ctx, 123)
	if err != nil || len(lists.Data) != 1 || lists.Data[0].Attributes.Kind != "grocery" {
		t.Fatalf("ListLists: lists=%+v err=%v", lists, err)
	}
	list, err := c.GetList(ctx, 123, "list-1")
	if err != nil || len(list.Items()) != 1 || list.Items()[0].ID != "item-1" {
		t.Fatalf("GetList: list=%+v err=%v", list, err)
	}
	recipes, err := c.ListRecipes(ctx, 123)
	if err != nil || len(recipes.Data) != 1 || recipes.Data[0].Attributes.Summary != "Tacos" {
		t.Fatalf("ListRecipes: recipes=%+v err=%v", recipes, err)
	}
	sittings, err := c.ListMealSittings(ctx, 123, MealSittingFilter{StartDate: "2026-07-01", EndDate: "2026-07-07"})
	if err != nil || len(sittings.Data) != 1 || sittings.Data[0].ID != "sitting-1" {
		t.Fatalf("ListMealSittings: sittings=%+v err=%v", sittings, err)
	}
}

func TestResourceMutationsOwnExactPrivateEndpoints(t *testing.T) {
	var requests []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/frames/123/lists/list-1/list_items":
			fmt.Fprint(w, `{"data":{"id":"item-1"}}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/frames/123/meals/sittings/sitting-1/instances/2026-07-01":
			fmt.Fprint(w, `{}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/frames/123/routines/reorder":
			fmt.Fprint(w, `{}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	ctx := context.Background()
	if _, err := c.CreateListItem(ctx, 123, "list-1", map[string]any{"label": "Milk"}); err != nil {
		t.Fatalf("CreateListItem: %v", err)
	}
	if err := c.DeleteMealSitting(ctx, 123, "sitting-1", "2026-07-01"); err != nil {
		t.Fatalf("DeleteMealSitting: %v", err)
	}
	if _, err := c.ReorderRoutines(ctx, 123, []string{"r2", "r1"}); err != nil {
		t.Fatalf("ReorderRoutines: %v", err)
	}
	if len(requests) != 3 {
		t.Fatalf("requests = %#v", requests)
	}
}

func TestSidekickReadsUseSanitizedAccessAndIntentResources(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/plus_access":
			fmt.Fprint(w, `{"data":{"bundle_entitlement":{"available":false},"self_serve_trial_eligibility":{"assistant":true},"subscriptions":[{"id":"private-subscription-id","attributes":{"plus_type":"cal_plus","status":"active","billing_provider":"stripe"}},{"id":"old","attributes":{"plus_type":"cal_plus","status":"expired"}}]}}`)
		case "/api/frames/123/auto_creation_intents":
			fmt.Fprint(w, `{"data":[{"id":"intent-1","type":"auto_creation_intent"}]}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "tok")
	access, err := c.GetPlusAccess(context.Background())
	if err != nil {
		t.Fatalf("GetPlusAccess: %v", err)
	}
	if !access.ActiveCalendarPlus || !access.AssistantTrialEligible || access.ActiveSubscriptionCount != 1 {
		t.Fatalf("access = %+v", access)
	}
	encoded, err := json.Marshal(access)
	if err != nil {
		t.Fatalf("marshal access: %v", err)
	}
	if strings.Contains(string(encoded), "private-subscription-id") || strings.Contains(string(encoded), "stripe") {
		t.Fatalf("sanitized access leaked subscription details: %s", encoded)
	}
	intents, err := c.ListAutoCreationIntents(context.Background(), 123)
	if err != nil || len(intents.Data) != 1 {
		t.Fatalf("ListAutoCreationIntents: intents=%+v err=%v", intents, err)
	}
}

func TestResourceMethodsKeepPrivateEndpointDetailsBehindClient(t *testing.T) {
	type callFunc func(context.Context, *Client) error
	tests := []struct {
		name     string
		method   string
		path     string
		query    string
		response string
		call     callFunc
	}{
		{"frame devices", http.MethodGet, "/api/frames/123/devices", "", `{}`, func(ctx context.Context, c *Client) error { _, err := c.ListFrameDevices(ctx, 123); return err }},
		{"frame device", http.MethodGet, "/api/frames/123/devices/device-1", "", `{}`, func(ctx context.Context, c *Client) error {
			_, err := c.GetFrameDevice(ctx, 123, "device-1")
			return err
		}},
		{"household config", http.MethodGet, "/api/frames/123/household_config", "", `{}`, func(ctx context.Context, c *Client) error { _, err := c.GetHouseholdConfig(ctx, 123); return err }},
		{"device alarms", http.MethodGet, "/api/frames/123/devices/device-1/alarms", "", `{}`, func(ctx context.Context, c *Client) error {
			_, err := c.ListDeviceAlarms(ctx, 123, "device-1")
			return err
		}},
		{"avatars", http.MethodGet, "/api/avatars", "", `{}`, func(ctx context.Context, c *Client) error { _, err := c.ListAvatars(ctx); return err }},
		{"colors", http.MethodGet, "/api/colors", "", `{}`, func(ctx context.Context, c *Client) error { _, err := c.ListColors(ctx); return err }},
		{"source calendars", http.MethodGet, "/api/frames/123/source_calendars", "", `{}`, func(ctx context.Context, c *Client) error { _, err := c.ListSourceCalendars(ctx, 123); return err }},
		{"search calendar events", http.MethodGet, "/api/frames/123/calendar_events/search", "include=categories&search_query=Camp&timezone=UTC", `{}`, func(ctx context.Context, c *Client) error {
			_, err := c.SearchCalendarEvents(ctx, 123, CalendarSearchFilter{Query: "Camp", Timezone: "UTC", Include: "categories"})
			return err
		}},
		{"countdown events", http.MethodGet, "/api/frames/123/calendar_events/countdowns", "date_max=2026-07-31&date_min=2026-07-01&include=categories&timezone=UTC", `{}`, func(ctx context.Context, c *Client) error {
			_, err := c.ListCountdownEvents(ctx, 123, CalendarEventFilter{StartDate: "2026-07-01", EndDate: "2026-07-31"}, "UTC", "categories")
			return err
		}},
		{"recent invited emails", http.MethodGet, "/api/frames/123/calendar_events/recent_invited_emails", "", `{}`, func(ctx context.Context, c *Client) error { _, err := c.ListRecentInvitedEmails(ctx, 123); return err }},
		{"create calendar event", http.MethodPost, "/api/frames/123/calendar_events", "", `{}`, func(ctx context.Context, c *Client) error {
			_, err := c.CreateCalendarEvent(ctx, 123, map[string]any{"summary": "Camp"})
			return err
		}},
		{"update calendar event", http.MethodPut, "/api/frames/123/calendar_events/event-1", "", `{}`, func(ctx context.Context, c *Client) error {
			_, err := c.UpdateCalendarEvent(ctx, 123, "event-1", map[string]any{"summary": "Camp"})
			return err
		}},
		{"delete calendar event", http.MethodDelete, "/api/frames/123/calendar_events/event-1", "", `{}`, func(ctx context.Context, c *Client) error { return c.DeleteCalendarEvent(ctx, 123, "event-1") }},
		{"create list", http.MethodPost, "/api/frames/123/lists", "", `{"data":{"id":"list-1"}}`, func(ctx context.Context, c *Client) error {
			_, err := c.CreateList(ctx, 123, map[string]any{"label": "Groceries"})
			return err
		}},
		{"update list", http.MethodPut, "/api/frames/123/lists/list-1", "", `{}`, func(ctx context.Context, c *Client) error {
			_, err := c.UpdateList(ctx, 123, "list-1", map[string]any{"label": "Groceries"})
			return err
		}},
		{"delete list", http.MethodDelete, "/api/frames/123/lists/list-1", "", `{}`, func(ctx context.Context, c *Client) error { return c.DeleteList(ctx, 123, "list-1") }},
		{"update list item", http.MethodPut, "/api/frames/123/lists/list-1/list_items/item-1", "", `{}`, func(ctx context.Context, c *Client) error {
			_, err := c.UpdateListItem(ctx, 123, "list-1", "item-1", map[string]any{"status": "completed"})
			return err
		}},
		{"delete list item", http.MethodDelete, "/api/frames/123/lists/list-1/list_items/item-1", "", `{}`, func(ctx context.Context, c *Client) error { return c.DeleteListItem(ctx, 123, "list-1", "item-1") }},
		{"organize list", http.MethodPost, "/api/frames/123/lists/list-1/organize", "", `{}`, func(ctx context.Context, c *Client) error { _, err := c.OrganizeList(ctx, 123, "list-1"); return err }},
		{"order list", http.MethodPost, "/api/frames/123/lists/list-1/order", "", `{}`, func(ctx context.Context, c *Client) error {
			_, err := c.OrderList(ctx, 123, "list-1", map[string]any{"retailer": "instacart"})
			return err
		}},
		{"list task box", http.MethodGet, "/api/frames/123/task_box/items", "", `{}`, func(ctx context.Context, c *Client) error { _, err := c.ListTaskBoxItems(ctx, 123); return err }},
		{"create task box item", http.MethodPost, "/api/frames/123/task_box/items", "", `{}`, func(ctx context.Context, c *Client) error {
			_, err := c.CreateTaskBoxItem(ctx, 123, map[string]any{"task_box_item": map[string]any{"title": "Inbox"}})
			return err
		}},
		{"meal categories", http.MethodGet, "/api/frames/123/meals/categories", "", `{}`, func(ctx context.Context, c *Client) error { _, err := c.ListMealCategories(ctx, 123); return err }},
		{"get recipe", http.MethodGet, "/api/frames/123/meals/recipes/recipe-1", "", `{}`, func(ctx context.Context, c *Client) error { _, err := c.GetRecipe(ctx, 123, "recipe-1"); return err }},
		{"create recipe", http.MethodPost, "/api/frames/123/meals/recipes", "", `{}`, func(ctx context.Context, c *Client) error {
			_, err := c.CreateRecipe(ctx, 123, map[string]any{"summary": "Tacos"})
			return err
		}},
		{"update recipe", http.MethodPatch, "/api/frames/123/meals/recipes/recipe-1", "", `{}`, func(ctx context.Context, c *Client) error {
			_, err := c.UpdateRecipe(ctx, 123, "recipe-1", map[string]any{"summary": "Tacos"})
			return err
		}},
		{"delete recipe", http.MethodDelete, "/api/frames/123/meals/recipes/recipe-1", "", `{}`, func(ctx context.Context, c *Client) error { return c.DeleteRecipe(ctx, 123, "recipe-1") }},
		{"create sitting", http.MethodPost, "/api/frames/123/meals/sittings", "", `{}`, func(ctx context.Context, c *Client) error {
			_, err := c.CreateMealSitting(ctx, 123, map[string]any{"date": "2026-07-01"})
			return err
		}},
		{"add recipe to grocery", http.MethodPost, "/api/frames/123/meals/recipes/recipe-1/add_to_grocery_list", "", `{}`, func(ctx context.Context, c *Client) error {
			_, err := c.AddRecipeToGroceryList(ctx, 123, "recipe-1")
			return err
		}},
		{"list photos", http.MethodGet, "/api/frames/123/messages", "page_token=next", `{}`, func(ctx context.Context, c *Client) error {
			_, err := c.ListPhotoMessages(ctx, 123, "next")
			return err
		}},
		{"get photo", http.MethodGet, "/api/frames/123/messages/10", "", `{}`, func(ctx context.Context, c *Client) error { _, err := c.GetPhotoMessage(ctx, 123, "10"); return err }},
		{"photo likes", http.MethodGet, "/api/frames/123/messages/10/all_likes", "", `{}`, func(ctx context.Context, c *Client) error {
			_, err := c.ListPhotoMessageLikes(ctx, 123, "10")
			return err
		}},
		{"photo comments", http.MethodGet, "/api/frames/123/messages/10/comments", "page=2", `{}`, func(ctx context.Context, c *Client) error {
			_, err := c.ListPhotoMessageComments(ctx, 123, "10", 2)
			return err
		}},
		{"list albums", http.MethodGet, "/api/frames/123/albums", "", `{}`, func(ctx context.Context, c *Client) error { _, err := c.ListAlbums(ctx, 123); return err }},
		{"album messages", http.MethodGet, "/api/frames/123/albums/album-1/messages", "page=2", `{}`, func(ctx context.Context, c *Client) error {
			_, err := c.ListAlbumMessages(ctx, 123, "album-1", 2)
			return err
		}},
		{"album message ids", http.MethodGet, "/api/frames/123/albums/album-1/messages/all_ids", "", `{}`, func(ctx context.Context, c *Client) error {
			_, err := c.ListAlbumMessageIDs(ctx, 123, "album-1")
			return err
		}},
		{"delete photos", http.MethodDelete, "/api/frames/123/messages/destroy_multiple", "", `{}`, func(ctx context.Context, c *Client) error { return c.DeletePhotoMessages(ctx, 123, []int{10, 11}) }},
		{"create photo upload", http.MethodPost, "/api/upload_url", "", `{"data":{"url":"https://upload.invalid/photo","key":"photo"}}`, func(ctx context.Context, c *Client) error {
			_, err := c.CreatePhotoUpload(ctx, "jpg", []string{"123"}, "Summer")
			return err
		}},
		{"list routines", http.MethodGet, "/api/frames/123/routines", "", `{}`, func(ctx context.Context, c *Client) error { _, err := c.ListRoutines(ctx, 123); return err }},
		{"create routine", http.MethodPost, "/api/frames/123/routines", "", `{}`, func(ctx context.Context, c *Client) error {
			_, err := c.CreateRoutine(ctx, 123, map[string]any{"title": "Morning"})
			return err
		}},
		{"update routine", http.MethodPut, "/api/frames/123/routines/routine-1", "", `{}`, func(ctx context.Context, c *Client) error {
			_, err := c.UpdateRoutine(ctx, 123, "routine-1", map[string]any{"title": "Evening"})
			return err
		}},
		{"delete routine", http.MethodDelete, "/api/frames/123/routines/routine-1", "", `{}`, func(ctx context.Context, c *Client) error { return c.DeleteRoutine(ctx, 123, "routine-1") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method || r.URL.Path != tt.path || r.URL.RawQuery != tt.query {
					t.Fatalf("request = %s %s?%s, want %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery, tt.method, tt.path, tt.query)
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, tt.response)
			}))
			defer srv.Close()

			if err := tt.call(context.Background(), New(srv.URL, "tok")); err != nil {
				t.Fatalf("call: %v", err)
			}
		})
	}
}
