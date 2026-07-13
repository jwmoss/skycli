package skylight

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type List struct {
	ID         string `json:"id"`
	Type       string `json:"type,omitempty"`
	Attributes struct {
		Label         string `json:"label"`
		Color         string `json:"color"`
		Kind          string `json:"kind"`
		HideFromFrame bool   `json:"hide_from_frame"`
	} `json:"attributes"`
}

type ListItem struct {
	ID         string `json:"id"`
	Type       string `json:"type,omitempty"`
	Attributes struct {
		Label    string `json:"label"`
		Status   string `json:"status"`
		Position int    `json:"position"`
	} `json:"attributes"`
}

type ListDocument struct {
	Data     List       `json:"data"`
	Included []ListItem `json:"included,omitempty"`
	raw      json.RawMessage
}

func (d ListDocument) MarshalJSON() ([]byte, error) {
	if len(d.raw) != 0 {
		return d.raw, nil
	}
	type plain ListDocument
	return json.Marshal(plain(d))
}

func (d ListDocument) Items() []ListItem {
	items := make([]ListItem, 0, len(d.Included))
	for _, item := range d.Included {
		if item.Type == "" || item.Type == "list_item" {
			items = append(items, item)
		}
	}
	return items
}

func (c *Client) ListLists(ctx context.Context, frameID int64) (*Collection[List], error) {
	raw, err := c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/lists", frameID), nil, nil)
	if err != nil {
		return nil, err
	}
	return decodeCollection[List](raw)
}

func (c *Client) GetList(ctx context.Context, frameID int64, listID string) (*ListDocument, error) {
	raw, err := c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/lists/%s", frameID, listID), nil, nil)
	if err != nil {
		return nil, err
	}
	var out ListDocument
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	out.raw = append(out.raw[:0], raw...)
	return &out, nil
}

func (c *Client) CreateList(ctx context.Context, frameID int64, payload map[string]any) (*Document[List], error) {
	raw, err := c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/lists", frameID), nil, payload)
	if err != nil {
		return nil, err
	}
	return decodeDocument[List](raw)
}

func (c *Client) UpdateList(ctx context.Context, frameID int64, listID string, payload map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, fmt.Sprintf("/api/frames/%d/lists/%s", frameID, listID), nil, payload)
}

func (c *Client) DeleteList(ctx context.Context, frameID int64, listID string) error {
	_, err := c.Do(ctx, http.MethodDelete, fmt.Sprintf("/api/frames/%d/lists/%s", frameID, listID), nil, nil)
	return err
}

func (c *Client) CreateListItem(ctx context.Context, frameID int64, listID string, payload map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/lists/%s/list_items", frameID, listID), nil, payload)
}

func (c *Client) UpdateListItem(ctx context.Context, frameID int64, listID, itemID string, payload map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, fmt.Sprintf("/api/frames/%d/lists/%s/list_items/%s", frameID, listID, itemID), nil, payload)
}

func (c *Client) DeleteListItem(ctx context.Context, frameID int64, listID, itemID string) error {
	_, err := c.Do(ctx, http.MethodDelete, fmt.Sprintf("/api/frames/%d/lists/%s/list_items/%s", frameID, listID, itemID), nil, nil)
	return err
}

func (c *Client) OrganizeList(ctx context.Context, frameID int64, listID string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/lists/%s/organize", frameID, listID), nil, nil)
}

func (c *Client) OrderList(ctx context.Context, frameID int64, listID string, payload map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/lists/%s/order", frameID, listID), nil, payload)
}

func (c *Client) ListTaskBoxItems(ctx context.Context, frameID int64) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/task_box/items", frameID), nil, nil)
}

func (c *Client) CreateTaskBoxItem(ctx context.Context, frameID int64, payload map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/task_box/items", frameID), nil, payload)
}
