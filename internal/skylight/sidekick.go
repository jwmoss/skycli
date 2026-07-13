package skylight

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type PlusAccess struct {
	BundleEntitlementAvailable bool `json:"bundle_entitlement_available"`
	AssistantTrialEligible     bool `json:"assistant_trial_eligible"`
	ActiveCalendarPlus         bool `json:"active_calendar_plus"`
	ActiveSubscriptionCount    int  `json:"active_subscription_count"`
}

func (c *Client) GetPlusAccess(ctx context.Context) (*PlusAccess, error) {
	raw, err := c.Do(ctx, http.MethodGet, "/api/plus_access", nil, nil)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Data struct {
			BundleEntitlement struct {
				Available bool `json:"available"`
			} `json:"bundle_entitlement"`
			SelfServeTrialEligibility struct {
				Assistant bool `json:"assistant"`
			} `json:"self_serve_trial_eligibility"`
			Subscriptions []struct {
				Attributes struct {
					PlusType string `json:"plus_type"`
					Status   string `json:"status"`
				} `json:"attributes"`
			} `json:"subscriptions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	out := &PlusAccess{
		BundleEntitlementAvailable: doc.Data.BundleEntitlement.Available,
		AssistantTrialEligible:     doc.Data.SelfServeTrialEligibility.Assistant,
	}
	for _, subscription := range doc.Data.Subscriptions {
		if subscription.Attributes.Status != "active" {
			continue
		}
		out.ActiveSubscriptionCount++
		if subscription.Attributes.PlusType == "cal_plus" {
			out.ActiveCalendarPlus = true
		}
	}
	return out, nil
}

func (c *Client) ListAutoCreationIntents(ctx context.Context, frameID int64) (*Collection[json.RawMessage], error) {
	raw, err := c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/auto_creation_intents", frameID), nil, nil)
	if err != nil {
		return nil, err
	}
	return decodeCollection[json.RawMessage](raw)
}
