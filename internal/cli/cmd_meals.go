package cli

import (
	"flag"
	"net/http"
	"net/url"
)

func runMeals(rc *runCtx, args []string) int {
	if len(args) == 0 {
		return mealRecipes(rc, nil)
	}
	switch args[0] {
	case "categories":
		return mealCategories(rc, args[1:])
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
		return exitUsage
	}
	return doFrameJSON(rc, *frameStr, http.MethodGet, "/api/frames/%d/meals/categories", nil, nil)
}

func mealRecipes(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("meals recipes", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	return doFrameJSON(rc, *frameStr, http.MethodGet, "/api/frames/%d/meals/recipes", nil, nil)
}

func mealRecipeInfo(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("meals recipe-info", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	recipeID := fs.String("recipe-id", "", "recipe ID")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*recipeID, "recipe-id"); err != nil {
		return usage(rc, err.Error())
	}
	return doFrameJSON(rc, *frameStr, http.MethodGet, "/api/frames/%d/meals/recipes/%s", nil, nil, *recipeID)
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
		return exitUsage
	}
	payload, err := mealRecipePayload(rc, fs, *body, *bodyFile, *title, *description, *ingredients, *recipeURL, *categoryID)
	if err != nil {
		return fail(rc, err)
	}
	if *title != "" {
		payload["summary"] = *title
	}
	return doFrameJSON(rc, *frameStr, http.MethodPost, "/api/frames/%d/meals/recipes", nil, payload)
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
		return exitUsage
	}
	if err := requireFlagValue(*recipeID, "recipe-id"); err != nil {
		return usage(rc, err.Error())
	}
	payload, err := mealRecipePayload(rc, fs, *body, *bodyFile, *title, *description, *ingredients, *recipeURL, *categoryID)
	if err != nil {
		return fail(rc, err)
	}
	return doFrameJSON(rc, *frameStr, http.MethodPatch, "/api/frames/%d/meals/recipes/%s", nil, payload, *recipeID)
}

func mealDeleteRecipe(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("meals delete-recipe", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	recipeID := fs.String("recipe-id", "", "recipe ID")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*recipeID, "recipe-id"); err != nil {
		return usage(rc, err.Error())
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	return doNoContent(rc, http.MethodDelete, "/api/frames/"+formatID(frameID)+"/meals/recipes/"+*recipeID, nil, nil, map[string]any{"deleted": *recipeID})
}

func mealSittings(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("meals sittings", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	dateMin := fs.String("date-min", "", "minimum date YYYY-MM-DD")
	dateMax := fs.String("date-max", "", "maximum date YYYY-MM-DD")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	q := url.Values{}
	if *dateMin != "" {
		q.Set("date_min", *dateMin)
	}
	if *dateMax != "" {
		q.Set("date_max", *dateMax)
	}
	return doFrameJSON(rc, *frameStr, http.MethodGet, "/api/frames/%d/meals/sittings", q, nil)
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
		return exitUsage
	}
	payload, err := readPayload(rc, *body, *bodyFile)
	if err != nil {
		return fail(rc, err)
	}
	addStringIfSet(fs, payload, "recipe-id", "meal_recipe_id", *recipeID)
	addStringIfSet(fs, payload, "summary", "summary", *summary)
	addStringIfSet(fs, payload, "date", "date", *date)
	addStringIfSet(fs, payload, "meal-category-id", "meal_category_id", *categoryID)
	return doFrameJSON(rc, *frameStr, http.MethodPost, "/api/frames/%d/meals/sittings", nil, payload)
}

func mealDeleteSitting(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("meals delete-sitting", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	sittingID := fs.String("sitting-id", "", "sitting ID")
	date := fs.String("date", "", "instance date YYYY-MM-DD")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*sittingID, "sitting-id"); err != nil {
		return usage(rc, err.Error())
	}
	if err := requireFlagValue(*date, "date"); err != nil {
		return usage(rc, err.Error())
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	path := "/api/frames/" + formatID(frameID) + "/meals/sittings/" + *sittingID + "/instances/" + *date
	return doNoContent(rc, http.MethodDelete, path, nil, nil, map[string]any{"deleted": *sittingID, "date": *date})
}

func mealAddToGrocery(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("meals add-to-grocery", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	recipeID := fs.String("recipe-id", "", "recipe ID")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*recipeID, "recipe-id"); err != nil {
		return usage(rc, err.Error())
	}
	return doFrameJSON(rc, *frameStr, http.MethodPost, "/api/frames/%d/meals/recipes/%s/add_to_grocery_list", nil, nil, *recipeID)
}
