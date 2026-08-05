package skylight

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (c *Client) ListAlbums(ctx context.Context, frameID int64) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/albums", frameID), nil, nil)
}

func (c *Client) ListAlbumMessages(ctx context.Context, frameID int64, albumID string, page int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("page", fmt.Sprintf("%d", page))
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/albums/%s/messages", frameID, albumID), q, nil)
}

func (c *Client) ListAlbumMessageIDs(ctx context.Context, frameID int64, albumID string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/albums/%s/messages/all_ids", frameID, albumID), nil, nil)
}
