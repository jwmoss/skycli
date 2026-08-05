package skylight

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type PhotoUploadTarget struct {
	UploadURL  string   `json:"url"`
	Key        string   `json:"key"`
	GetURL     string   `json:"get_url"`
	MessageIDs []int    `json:"message_ids"`
	FrameNames []string `json:"frame_names"`
}

func (c *Client) ListPhotoMessages(ctx context.Context, frameID int64, pageToken string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("page_token", pageToken)
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/messages", frameID), q, nil)
}

func (c *Client) GetPhotoMessage(ctx context.Context, frameID int64, messageID string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/messages/%s", frameID, messageID), nil, nil)
}

func (c *Client) ListPhotoMessageLikes(ctx context.Context, frameID int64, messageID string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/messages/%s/all_likes", frameID, messageID), nil, nil)
}

func (c *Client) ListPhotoMessageComments(ctx context.Context, frameID int64, messageID string, page int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("page", fmt.Sprintf("%d", page))
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/messages/%s/comments", frameID, messageID), q, nil)
}

func (c *Client) DeletePhotoMessages(ctx context.Context, frameID int64, messageIDs []int) error {
	body := map[string]any{"message_ids": messageIDs}
	_, err := c.Do(ctx, http.MethodDelete, fmt.Sprintf("/api/frames/%d/messages/destroy_multiple", frameID), nil, body)
	return err
}

func (c *Client) CreatePhotoUpload(ctx context.Context, extension string, frameIDs []string, caption string) (*PhotoUploadTarget, error) {
	body := map[string]any{"ext": extension, "frame_ids": frameIDs}
	if caption != "" {
		body["caption"] = caption
	}
	raw, err := c.Do(ctx, http.MethodPost, "/api/upload_url", nil, body)
	if err != nil {
		return nil, err
	}
	var doc Document[PhotoUploadTarget]
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	if doc.Data.UploadURL == "" {
		return nil, fmt.Errorf("empty upload URL in response")
	}
	return &doc.Data, nil
}
