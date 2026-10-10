package skylight

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (c *Client) GetCategory(ctx context.Context, frameID int64, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/categories/%s", frameID, id), nil, nil)
}

func (c *Client) CreateCategory(ctx context.Context, frameID int64, body map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/categories", frameID), nil, body)
}

func (c *Client) UpdateCategory(ctx context.Context, frameID int64, id string, body map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, fmt.Sprintf("/api/frames/%d/categories/%s", frameID, id), nil, body)
}

func (c *Client) DeleteCategory(ctx context.Context, frameID int64, id, reassignTo string) error {
	q := url.Values{}
	if reassignTo != "" {
		q.Set("reassign_to_category_id", reassignTo)
	}
	_, err := c.Do(ctx, http.MethodDelete, fmt.Sprintf("/api/frames/%d/categories/%s", frameID, id), q, nil)
	return err
}
