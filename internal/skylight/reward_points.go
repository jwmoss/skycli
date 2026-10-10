package skylight

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func (c *Client) AdjustRewardPoints(ctx context.Context, frameID int64, categoryIDs []int, points int) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/reward_points", frameID), nil, map[string]any{"category_ids": categoryIDs, "points": points})
}
