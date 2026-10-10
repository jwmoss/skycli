package skylight

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Routines share the task endpoints and carry routine=true.
func (c *Client) ListRoutines(ctx context.Context, frameID int64, filters ...ChoreFilter) (json.RawMessage, error) {
	filter := ChoreFilter{}
	if len(filters) > 0 {
		filter = filters[0]
	}
	filter.OnlyRoutines = true
	chores, err := c.ListChores(ctx, frameID, filter)
	if err != nil {
		return nil, err
	}
	return json.Marshal(choresListEnvelope{Data: chores})
}

func (c *Client) CreateMultipleChores(ctx context.Context, frameID int64, payload map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/chores/create_multiple", frameID), nil, payload)
}

func (c *Client) CreateRoutine(ctx context.Context, frameID int64, payload map[string]any) (json.RawMessage, error) {
	payload["routine"] = true
	return c.CreateMultipleChores(ctx, frameID, payload)
}

func (c *Client) UpdateRoutine(ctx context.Context, frameID int64, routineID string, payload map[string]any) (json.RawMessage, error) {
	payload["routine"] = true
	return c.Do(ctx, http.MethodPut, fmt.Sprintf("/api/frames/%d/chores/%s", frameID, url.PathEscape(routineID)), nil, payload)
}

func (c *Client) DeleteRoutine(ctx context.Context, frameID int64, routineID string) error {
	baseID, _ := SplitChoreInstanceID(routineID)
	_, err := ParseID(baseID)
	if err != nil {
		return fmt.Errorf("invalid routine ID: %w", err)
	}
	return c.DeleteChore(ctx, frameID, routineID, "all")
}

func (c *Client) MoveChore(ctx context.Context, frameID int64, choreID string, before, after string) (json.RawMessage, error) {
	position := map[string]string{}
	if before != "" {
		position["before"] = before
	}
	if after != "" {
		position["after"] = after
	}
	baseID, _ := SplitChoreInstanceID(choreID)
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/chores/%s/move", frameID, baseID), nil, map[string]any{"position": position})
}

func (c *Client) ReorderRoutines(ctx context.Context, frameID int64, routineIDs []string) (json.RawMessage, error) {
	results := make([]json.RawMessage, 0, len(routineIDs))
	for i := len(routineIDs) - 2; i >= 0; i-- {
		result, err := c.MoveChore(ctx, frameID, routineIDs[i], routineIDs[i+1], "")
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return json.Marshal(results)
}
