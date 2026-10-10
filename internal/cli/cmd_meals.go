package cli

import (
	"flag"
	"strings"
	"time"

	"github.com/jwmoss/skycli/internal/skylight"
)

func runMeals(rc *runCtx, args []string) int {
	if len(args) == 0 {
		return mealRecipes(rc, nil)
	}
	switch args[0] {
	case "categories":
		return mealCategories(rc, args[1:])
	case "update-category":
		return mealUpdateCategory(rc, args[1:])
	case "recipes":
		return mealRecipes(rc, args[1:])
	case "recipe-info":
		return mealRecipeInfo(rc, args[1:])
	case "create-recipe":
		return mealCreateRecipe(rc, args[1:])
	case "update-recipe":
		return mealUpdateRecipe(rc, args[1:])
	case "delete-recipe":
		return mealDeleteRecipe(rc, args[1:])
	case "sittings":
		return mealSittings(rc, args[1:])
	case "create-sitting":
		return mealCreateSitting(rc, args[1:])
	case "update-sitting":
		return mealUpdateSitting(rc, args[1:])
	case "delete-sitting":
		return mealDeleteSitting(rc, args[1:])
	case "add-to-grocery":
		return mealAddToGrocery(rc, args[1:])
	default:
		return usage(rc, "unknown meals subcommand: "+args[0])
	}
}

func mealCategories(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("meals categories", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.ListMealCategories(rc.ctx, frameID)
	})
}

func mealUpdateCategory(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("meals update-category", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frame := fs.String("frame", "", "frame ID")
	category := fs.String("category-id", "", "meal category ID (required)")
	body, bodyFile := bodyFlags(fs, rc)
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	id, err := positiveMediaID(*category, "category-id")
	if err != nil {
		return usage(rc, err.Error())
	}
	payload, err := readPayload(rc, *body, *bodyFile)
	if err != nil {
		return usage(rc, err.Error())
	}
	if len(payload) == 0 {
		return usage(rc, "provide a non-empty JSON object with --body or --body-file")
	}
	return runFrameResourceJSON(rc, *frame, func(c *skylight.Client, frameID int64) (any, error) {
		return c.UpdateMealCategory(rc.ctx, frameID, id, payload)
	})
}

func mealRecipes(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("meals recipes", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.ListRecipes(rc.ctx, frameID)
	})
}

func mealRecipeInfo(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("meals recipe-info", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	recipeID := fs.String("recipe-id", "", "recipe ID")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if err := requireFlagValue(*recipeID, "recipe-id"); err != nil {
		return usage(rc, err.Error())
	}
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.GetRecipe(rc.ctx, frameID, *recipeID)
	})
}

func mealRecipePayload(rc *runCtx, fs *flag.FlagSet, body, bodyFile, title, description, ingredients, recipeURL, categoryID string) (map[string]any, error) {
	payload, err := readPayload(rc, body, bodyFile)
	if err != nil {
		return nil, err
	}
	addStringIfSet(fs, payload, "title", "summary", title)
	addStringIfSet(fs, payload, "description", "description", description)
	if flagChanged(fs, "ingredients") {
		payload["ingredients"] = parseCSVStrings(ingredients)
	}
	addStringIfSet(fs, payload, "url", "url", recipeURL)
	addStringIfSet(fs, payload, "meal-category-id", "meal_category_id", categoryID)
	return payload, nil
}

func mealCreateRecipe(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("meals create-recipe", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	title := fs.String("title", "", "recipe title")
	description := fs.String("description", "", "recipe description")
	ingredients := fs.String("ingredients", "", "comma-separated ingredients")
	recipeURL := fs.String("url", "", "recipe URL")
	categoryID := fs.String("meal-category-id", "", "meal category ID")
	body, bodyFile := bodyFlags(fs, rc)
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	payload, err := mealRecipePayload(rc, fs, *body, *bodyFile, *title, *description, *ingredients, *recipeURL, *categoryID)
	if err != nil {
		return fail(rc, err)
	}
	if *title != "" {
		payload["summary"] = *title
	}
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.CreateRecipe(rc.ctx, frameID, payload)
	})
}

func mealUpdateRecipe(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("meals update-recipe", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	recipeID := fs.String("recipe-id", "", "recipe ID")
	title := fs.String("title", "", "recipe title")
	description := fs.String("description", "", "recipe description")
	ingredients := fs.String("ingredients", "", "comma-separated ingredients")
	recipeURL := fs.String("url", "", "recipe URL")
	categoryID := fs.String("meal-category-id", "", "meal category ID")
	body, bodyFile := bodyFlags(fs, rc)
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if err := requireFlagValue(*recipeID, "recipe-id"); err != nil {
		return usage(rc, err.Error())
	}
	payload, err := mealRecipePayload(rc, fs, *body, *bodyFile, *title, *description, *ingredients, *recipeURL, *categoryID)
	if err != nil {
		return fail(rc, err)
	}
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.UpdateRecipe(rc.ctx, frameID, *recipeID, payload)
	})
}

func mealDeleteRecipe(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("meals delete-recipe", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	recipeID := fs.String("recipe-id", "", "recipe ID")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if err := requireFlagValue(*recipeID, "recipe-id"); err != nil {
		return usage(rc, err.Error())
	}
	return runFrameResourceOK(rc, *frameStr, map[string]any{"deleted": *recipeID}, func(c *skylight.Client, frameID int64) error {
		return c.DeleteRecipe(rc.ctx, frameID, *recipeID)
	})
}

func mealSittings(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("meals sittings", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	dateMin := fs.String("date-min", "", "minimum date YYYY-MM-DD")
	dateMax := fs.String("date-max", "", "maximum date YYYY-MM-DD")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.ListMealSittings(rc.ctx, frameID, skylight.MealSittingFilter{StartDate: *dateMin, EndDate: *dateMax})
	})
}

func mealCreateSitting(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("meals create-sitting", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	recipeID := fs.String("recipe-id", "", "recipe ID")
	summary := fs.String("summary", "", "sitting summary")
	date := fs.String("date", "", "sitting date YYYY-MM-DD")
	categoryID := fs.String("meal-category-id", "", "meal category ID")
	body, bodyFile := bodyFlags(fs, rc)
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	payload, err := readPayload(rc, *body, *bodyFile)
	if err != nil {
		return fail(rc, err)
	}
	addStringIfSet(fs, payload, "recipe-id", "meal_recipe_id", *recipeID)
	addStringIfSet(fs, payload, "summary", "summary", *summary)
	addStringIfSet(fs, payload, "date", "date", *date)
	addStringIfSet(fs, payload, "meal-category-id", "meal_category_id", *categoryID)
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.CreateMealSitting(rc.ctx, frameID, payload)
	})
}

func mealDeleteSitting(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("meals delete-sitting", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	sittingID := fs.String("sitting-id", "", "sitting ID")
	date := fs.String("date", "", "instance date YYYY-MM-DD")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if err := requireFlagValue(*sittingID, "sitting-id"); err != nil {
		return usage(rc, err.Error())
	}
	if err := requireFlagValue(*date, "date"); err != nil {
		return usage(rc, err.Error())
	}
	return runFrameResourceOK(rc, *frameStr, map[string]any{"deleted": *sittingID, "date": *date}, func(c *skylight.Client, frameID int64) error {
		return c.DeleteMealSitting(rc.ctx, frameID, *sittingID, *date)
	})
}

func mealUpdateSitting(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("meals update-sitting", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frame := fs.String("frame", "", "frame ID")
	sittingID := fs.String("sitting-id", "", "meal sitting ID (required)")
	instanceDate := fs.String("instance-date", "", "original instance date YYYY-MM-DD (required)")
	applyTo := fs.String("apply-to", "", "one | future | all (required)")
	date := fs.String("date", "", "new meal date YYYY-MM-DD")
	rrule := fs.String("rrule", "", "recurrence rule string")
	recipeID := fs.String("recipe-id", "", "recipe ID")
	categoryID := fs.String("meal-category-id", "", "meal category ID")
	summary := fs.String("summary", "", "meal summary")
	body, bodyFile := bodyFlags(fs, rc)
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	id, err := positiveMediaID(*sittingID, "sitting-id")
	if err != nil {
		return usage(rc, err.Error())
	}
	if _, err := time.Parse(dateLayout, *instanceDate); err != nil {
		return usage(rc, "--instance-date requires a valid date in YYYY-MM-DD format")
	}
	switch *applyTo {
	case "one", "future", "all":
	default:
		return usage(rc, "--apply-to requires one, future, or all")
	}
	for _, field := range []struct{ name, value string }{{"recipe-id", *recipeID}, {"meal-category-id", *categoryID}} {
		if flagChanged(fs, field.name) {
			if _, err := positiveMediaID(field.value, field.name); err != nil {
				return usage(rc, err.Error())
			}
		}
	}
	if flagChanged(fs, "rrule") && strings.TrimSpace(*rrule) == "" {
		return usage(rc, "--rrule requires a non-empty recurrence rule")
	}
	payload, err := readPayload(rc, *body, *bodyFile)
	if err != nil {
		return fail(rc, err)
	}
	addStringIfSet(fs, payload, "recipe-id", "meal_recipe_id", *recipeID)
	addStringIfSet(fs, payload, "meal-category-id", "meal_category_id", *categoryID)
	addStringIfSet(fs, payload, "summary", "summary", *summary)
	addStringIfSet(fs, payload, "date", "date", *date)
	addStringIfSet(fs, payload, "rrule", "rrule", *rrule)
	if len(payload) == 0 {
		return usage(rc, "provide at least one update field")
	}
	if value, ok := payload["summary"]; ok && value != nil {
		text, ok := value.(string)
		if !ok || strings.TrimSpace(text) == "" {
			return usage(rc, "summary requires non-empty text")
		}
	}
	if value, ok := payload["date"]; ok {
		text, ok := value.(string)
		if _, err := time.Parse(dateLayout, text); !ok || err != nil {
			return usage(rc, "--date or body date requires a valid date in YYYY-MM-DD format")
		}
	}
	return runFrameResourceJSON(rc, *frame, func(c *skylight.Client, frameID int64) (any, error) {
		return c.UpdateMealSitting(rc.ctx, frameID, id, *instanceDate, *applyTo, payload)
	})
}

func mealAddToGrocery(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("meals add-to-grocery", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	recipeID := fs.String("recipe-id", "", "recipe ID")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if err := requireFlagValue(*recipeID, "recipe-id"); err != nil {
		return usage(rc, err.Error())
	}
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.AddRecipeToGroceryList(rc.ctx, frameID, *recipeID)
	})
}
