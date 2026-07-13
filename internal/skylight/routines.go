package skylight

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func (c *Client) ListRoutines(ctx context.Context, frameID int64) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/routines", frameID), nil, nil)
}

func (c *Client) CreateRoutine(ctx context.Context, frameID int64, payload map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/routines", frameID), nil, payload)
}

func (c *Client) UpdateRoutine(ctx context.Context, frameID int64, routineID string, payload map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, fmt.Sprintf("/api/frames/%d/routines/%s", frameID, routineID), nil, payload)
}

func (c *Client) DeleteRoutine(ctx context.Context, frameID int64, routineID string) error {
	_, err := c.Do(ctx, http.MethodDelete, fmt.Sprintf("/api/frames/%d/routines/%s", frameID, routineID), nil, nil)
	return err
}

func (c *Client) ReorderRoutines(ctx context.Context, frameID int64, routineIDs []string) (json.RawMessage, error) {
	body := map[string]any{"ids": routineIDs}
	return c.Do(ctx, http.MethodPatch, fmt.Sprintf("/api/frames/%d/routines/reorder", frameID), nil, body)
}
