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
	q := url.Values{}
	for _, id := range messageIDs {
		q.Add("message_ids[]", fmt.Sprintf("%d", id))
	}
	_, err := c.Do(ctx, http.MethodDelete, fmt.Sprintf("/api/frames/%d/messages/destroy_multiple", frameID), q, nil)
	return err
}

func (c *Client) UpdatePhotoCaption(ctx context.Context, frameID, messageID int64, caption string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, fmt.Sprintf("/api/frames/%d/messages/%d/caption", frameID, messageID), nil, map[string]any{"caption": caption})
}

func (c *Client) SetPhotoLike(ctx context.Context, frameID, messageID int64, liked bool) (json.RawMessage, error) {
	method := http.MethodPost
	if !liked {
		method = http.MethodDelete
	}
	return c.Do(ctx, method, fmt.Sprintf("/api/frames/%d/messages/%d/likes", frameID, messageID), nil, nil)
}

func (c *Client) CreatePhotoComment(ctx context.Context, frameID, messageID int64, text string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/messages/%d/comments", frameID, messageID), nil, map[string]any{"body": text})
}

func (c *Client) DeletePhotoComment(ctx context.Context, frameID, messageID, commentID int64) error {
	_, err := c.Do(ctx, http.MethodDelete, fmt.Sprintf("/api/frames/%d/messages/%d/comments/%d", frameID, messageID, commentID), nil, nil)
	return err
}

func (c *Client) CopyPhotoMessages(ctx context.Context, frameID int64, messageIDs, newFrameIDs []int64) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/copy_to_frames", frameID), nil, map[string]any{"message_ids": messageIDs, "new_frame_ids": newFrameIDs})
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
