package skylight

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Recipe struct {
	ID         string `json:"id"`
	Attributes struct {
		Summary     string   `json:"summary"`
		Description string   `json:"description"`
		Ingredients []string `json:"ingredients"`
		URL         string   `json:"url"`
	} `json:"attributes"`
	Relationships struct {
		MealCategory struct {
			Data *struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"meal_category"`
	} `json:"relationships"`
}

type MealSitting struct {
	ID         string `json:"id"`
	Attributes struct {
		Summary string `json:"summary"`
		Date    string `json:"date"`
	} `json:"attributes"`
	Relationships struct {
		MealCategory struct {
			Data *struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"meal_category"`
		MealRecipe struct {
			Data *struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"meal_recipe"`
	} `json:"relationships"`
}

type MealSittingFilter struct {
	StartDate string
	EndDate   string
}

func (c *Client) ListMealCategories(ctx context.Context, frameID int64) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/meals/categories", frameID), nil, nil)
}

func (c *Client) ListRecipes(ctx context.Context, frameID int64) (*Collection[Recipe], error) {
	raw, err := c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/meals/recipes", frameID), nil, nil)
	if err != nil {
		return nil, err
	}
	return decodeCollection[Recipe](raw)
}

func (c *Client) GetRecipe(ctx context.Context, frameID int64, recipeID string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/meals/recipes/%s", frameID, recipeID), nil, nil)
}

func (c *Client) CreateRecipe(ctx context.Context, frameID int64, payload map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/meals/recipes", frameID), nil, payload)
}

func (c *Client) UpdateRecipe(ctx context.Context, frameID int64, recipeID string, payload map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPatch, fmt.Sprintf("/api/frames/%d/meals/recipes/%s", frameID, recipeID), nil, payload)
}

func (c *Client) DeleteRecipe(ctx context.Context, frameID int64, recipeID string) error {
	_, err := c.Do(ctx, http.MethodDelete, fmt.Sprintf("/api/frames/%d/meals/recipes/%s", frameID, recipeID), nil, nil)
	return err
}

func (c *Client) ListMealSittings(ctx context.Context, frameID int64, filter MealSittingFilter) (*Collection[MealSitting], error) {
	q := url.Values{}
	if filter.StartDate != "" {
		q.Set("date_min", filter.StartDate)
	}
	if filter.EndDate != "" {
		q.Set("date_max", filter.EndDate)
	}
	raw, err := c.Do(ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/meals/sittings", frameID), q, nil)
	if err != nil {
		return nil, err
	}
	return decodeCollection[MealSitting](raw)
}

func (c *Client) CreateMealSitting(ctx context.Context, frameID int64, payload map[string]any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/meals/sittings", frameID), nil, payload)
}

func (c *Client) DeleteMealSitting(ctx context.Context, frameID int64, sittingID, date string) error {
	_, err := c.Do(ctx, http.MethodDelete, fmt.Sprintf("/api/frames/%d/meals/sittings/%s/instances/%s", frameID, sittingID, date), nil, nil)
	return err
}

func (c *Client) AddRecipeToGroceryList(ctx context.Context, frameID int64, recipeID string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/meals/recipes/%s/add_to_grocery_list", frameID, recipeID), nil, nil)
}
