package skylight

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func (c *Client) ListFrameDevices(ctx context.Context, frameID int64) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/devices", frameID), nil, nil)
}

func (c *Client) GetFrameDevice(ctx context.Context, frameID int64, deviceID string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/devices/%s", frameID, deviceID), nil, nil)
}

func (c *Client) UpdateFrameDevice(ctx context.Context, frameID, deviceID int64, payload map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, fmt.Sprintf("/api/frames/%d/devices/%d", frameID, deviceID), nil, payload)
}

func (c *Client) SetDeviceSleep(ctx context.Context, frameID, deviceID int64, asleep bool) (json.RawMessage, error) {
	action := "end_sleep"
	if asleep {
		action = "start_sleep"
	}
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/devices/%d/%s", frameID, deviceID, action), nil, nil)
}

func (c *Client) GetHouseholdConfig(ctx context.Context, frameID int64) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/household_config", frameID), nil, nil)
}

func (c *Client) UpdateHouseholdConfig(ctx context.Context, frameID int64, payload map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPatch, fmt.Sprintf("/api/frames/%d/household_config", frameID), nil, payload)
}

func (c *Client) ListDeviceAlarms(ctx context.Context, frameID int64, deviceID string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/devices/%s/alarms", frameID, deviceID), nil, nil)
}

func (c *Client) ListAvatars(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/avatars", nil, nil)
}

func (c *Client) ListColors(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/colors", nil, nil)
}

func (c *Client) ListHatPacks(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/hat_packs", nil, nil)
}
