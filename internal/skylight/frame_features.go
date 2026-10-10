package skylight

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (c *Client) ListFrameUsers(ctx context.Context, frameID int64) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/users", frameID), nil, nil)
}

func (c *Client) GetDeviceConfig(ctx context.Context, frameID, deviceID int64) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/devices/%d/device_config", frameID, deviceID), nil, nil)
}

func (c *Client) UpdateDeviceConfig(ctx context.Context, frameID, deviceID int64, payload map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPatch, fmt.Sprintf("/api/frames/%d/devices/%d/device_config", frameID, deviceID), nil, payload)
}

func (c *Client) CreateNudge(ctx context.Context, frameID int64, payload map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/nudges", frameID), nil, payload)
}

func (c *Client) UpdateNudge(ctx context.Context, frameID, nudgeID int64, payload map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPatch, fmt.Sprintf("/api/frames/%d/nudges/%d", frameID, nudgeID), nil, payload)
}

func (c *Client) DeleteNudge(ctx context.Context, frameID, nudgeID int64, deliverAt string) (json.RawMessage, error) {
	query := url.Values{}
	if deliverAt != "" {
		query.Set("deliver_at", deliverAt)
	}
	return c.Do(ctx, http.MethodDelete, fmt.Sprintf("/api/frames/%d/nudges/%d", frameID, nudgeID), query, nil)
}
