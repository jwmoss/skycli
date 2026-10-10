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

func (c *Client) CreateAlbum(ctx context.Context, frameID int64, title string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/albums", frameID), nil, map[string]any{"title": title})
}

func (c *Client) UpdateAlbum(ctx context.Context, frameID, albumID int64, title string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPatch, fmt.Sprintf("/api/frames/%d/albums/%d", frameID, albumID), nil, map[string]any{"title": title})
}

func (c *Client) DeleteAlbum(ctx context.Context, frameID, albumID int64) error {
	_, err := c.Do(ctx, http.MethodDelete, fmt.Sprintf("/api/frames/%d/albums/%d", frameID, albumID), nil, nil)
	return err
}

func (c *Client) AddMessagesToAlbums(ctx context.Context, frameID int64, albumIDs, messageIDs []int64) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/albums/add_to", frameID), nil, map[string]any{"album_ids": albumIDs, "message_ids": messageIDs})
}

func (c *Client) RemoveMessagesFromAlbum(ctx context.Context, frameID, albumID int64, messageIDs []int64) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/albums/remove_from", frameID), nil, map[string]any{"album_ids": []int64{albumID}, "message_ids": messageIDs})
}
