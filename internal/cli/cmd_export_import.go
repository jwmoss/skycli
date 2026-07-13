package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jwmoss/skycli/internal/skylight"
)

var allPortableResources = []string{"chores", "rewards", "lists", "recipes", "sittings", "calendar"}

type portableExport struct {
	ExportedAt     string                  `json:"exported_at"`
	FrameID        int64                   `json:"frame_id"`
	Chores         []portableChore         `json:"chores,omitempty"`
	Rewards        []portableReward        `json:"rewards,omitempty"`
	Lists          []portableList          `json:"lists,omitempty"`
	Recipes        []portableRecipe        `json:"recipes,omitempty"`
	MealSittings   []portableMealSitting   `json:"meal_sittings,omitempty"`
	CalendarEvents []portableCalendarEvent `json:"calendar_events,omitempty"`
}

type portableChore struct {
	ID            string   `json:"id,omitempty"`
	Summary       string   `json:"summary"`
	CategoryID    int64    `json:"category_id,omitempty"`
	Start         string   `json:"start"`
	RecurrenceSet []string `json:"recurrence_set,omitempty"`
	UpForGrabs    bool     `json:"up_for_grabs,omitempty"`
	RewardPoints  *int     `json:"reward_points,omitempty"`
	Description   string   `json:"description,omitempty"`
	EmojiIcon     string   `json:"emoji_icon,omitempty"`
}

type portableReward struct {
	ID                  string  `json:"id,omitempty"`
	Name                string  `json:"name"`
	PointValue          int     `json:"point_value"`
	CategoryID          int64   `json:"category_id,omitempty"`
	CategoryIDs         []int64 `json:"category_ids,omitempty"`
	EmojiIcon           string  `json:"emoji_icon,omitempty"`
	Description         string  `json:"description,omitempty"`
	RespawnOnRedemption bool    `json:"respawn_on_redemption,omitempty"`
}

type portableList struct {
	ID            string             `json:"id,omitempty"`
	Label         string             `json:"label"`
	Color         string             `json:"color,omitempty"`
	Kind          string             `json:"kind,omitempty"`
	HideFromFrame bool               `json:"hide_from_frame,omitempty"`
	Items         []portableListItem `json:"list_items,omitempty"`
}

type portableListItem struct {
	ID       string `json:"id,omitempty"`
	Label    string `json:"label"`
	Status   string `json:"status,omitempty"`
	Position int    `json:"position,omitempty"`
}

type portableRecipe struct {
	ID             string   `json:"id,omitempty"`
	Summary        string   `json:"summary"`
	Description    string   `json:"description,omitempty"`
	Ingredients    []string `json:"ingredients,omitempty"`
	URL            string   `json:"url,omitempty"`
	MealCategoryID string   `json:"meal_category_id,omitempty"`
}

type portableMealSitting struct {
	ID             string `json:"id,omitempty"`
	Summary        string `json:"summary"`
	RecipeID       string `json:"meal_recipe_id,omitempty"`
	MealCategoryID string `json:"meal_category_id,omitempty"`
	Date           string `json:"date,omitempty"`
}

type portableCalendarEvent struct {
	ID          string `json:"id,omitempty"`
	Summary     string `json:"summary"`
	StartsAt    string `json:"starts_at"`
	EndsAt      string `json:"ends_at,omitempty"`
	AllDay      bool   `json:"all_day,omitempty"`
	Color       string `json:"color,omitempty"`
	CategoryID  string `json:"category_id,omitempty"`
	Description string `json:"description,omitempty"`
}

type recipeEntry = skylight.Recipe

type sittingEntry = skylight.MealSitting

func runExport(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	output := fs.String("output-file", "", "output file path, or stdout when omitted")
	resourcesRaw := fs.String("resources", "all", "comma-separated resources: chores,rewards,lists,recipes,sittings,calendar")
	days := fs.Int("days", 90, "window in days before/after today for time-bounded resources")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	resources, err := parseResourceSelection(*resourcesRaw, allPortableResources)
	if err != nil {
		return usage(rc, err.Error())
	}
	out, err := collectPortableExport(rc, frameID, resources, *days)
	if err != nil {
		return fail(rc, err)
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fail(rc, err)
	}
	data = append(data, '\n')
	if *output == "" || *output == "-" {
		fmt.Fprint(rc.stdout, string(data))
		return exitOK
	}
	if err := os.WriteFile(*output, data, 0o600); err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(map[string]any{"output_file": *output})
	} else {
		rc.out.Line("exported to %s", *output)
	}
	return exitOK
}

func collectPortableExport(rc *runCtx, frameID int64, resources map[string]bool, days int) (portableExport, error) {
	now := time.Now()
	start := now.AddDate(0, 0, -days).Format(dateLayout)
	end := now.AddDate(0, 0, days).Format(dateLayout)
	c, err := rc.client()
	if err != nil {
		return portableExport{}, err
	}
	out := portableExport{ExportedAt: now.Format(time.RFC3339), FrameID: frameID}
	if resources["chores"] {
		chores, err := c.ListChores(rc.ctx, frameID, skylight.ChoreFilter{After: start, Before: end, IncludeLate: true, IncludeUpForGrabs: true})
		if err != nil {
			return portableExport{}, err
		}
		out.Chores = portableChores(chores)
	}
	if resources["rewards"] {
		rewards, err := c.ListRewards(rc.ctx, frameID)
		if err != nil {
			return portableExport{}, err
		}
		out.Rewards = portableRewards(rewards)
	}
	if resources["lists"] {
		lists, err := fetchLists(rc, frameID)
		if err != nil {
			return portableExport{}, err
		}
		for _, l := range lists {
			pl := portableList{
				ID:            l.ID,
				Label:         l.Attributes.Label,
				Color:         l.Attributes.Color,
				Kind:          l.Attributes.Kind,
				HideFromFrame: l.Attributes.HideFromFrame,
			}
			items, err := fetchListItems(rc, frameID, l.ID)
			if err != nil {
				return portableExport{}, fmt.Errorf("fetch items for list %s: %w", l.ID, err)
			}
			for _, it := range items {
				pl.Items = append(pl.Items, portableListItem{ID: it.ID, Label: it.Attributes.Label, Status: it.Attributes.Status, Position: it.Attributes.Position})
			}
			out.Lists = append(out.Lists, pl)
		}
	}
	if resources["recipes"] {
		recipes, err := fetchRecipes(rc, frameID)
		if err != nil {
			return portableExport{}, err
		}
		out.Recipes = portableRecipes(recipes)
	}
	if resources["sittings"] {
		sittings, err := fetchSittings(rc, frameID, start, end)
		if err != nil {
			return portableExport{}, err
		}
		out.MealSittings = portableSittings(sittings)
	}
	if resources["calendar"] {
		events, err := fetchCalendarEvents(rc, frameID, start, end)
		if err != nil {
			return portableExport{}, err
		}
		out.CalendarEvents = portableCalendarEvents(events)
	}
	return out, nil
}

func runImport(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	file := fs.String("file", "", "export JSON file")
	resourcesRaw := fs.String("resources", "all", "comma-separated resources to import")
	dryRun := fs.Bool("dry-run", false, "show counts without creating resources")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*file, "file"); err != nil {
		return usage(rc, err.Error())
	}
	raw, err := os.ReadFile(*file)
	if err != nil {
		return fail(rc, err)
	}
	var data portableExport
	if err := json.Unmarshal(raw, &data); err != nil {
		return fail(rc, fmt.Errorf("parse export file: %w", err))
	}
	resources, err := parseResourceSelection(*resourcesRaw, allPortableResources)
	if err != nil {
		return usage(rc, err.Error())
	}
	if *dryRun {
		_ = rc.out.JSON(importCounts(data, resources))
		return exitOK
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	result := map[string]any{"created": map[string]int{}, "failed": []map[string]string{}}
	created := result["created"].(map[string]int)
	failures := result["failed"].([]map[string]string)
	recordFailure := func(resource, name string, err error) {
		failures = append(failures, map[string]string{"resource": resource, "name": name, "error": err.Error()})
	}
	if resources["rewards"] {
		for _, r := range data.Rewards {
			cats := r.CategoryIDs
			if len(cats) == 0 && r.CategoryID != 0 {
				cats = []int64{r.CategoryID}
			}
			if len(cats) == 0 {
				recordFailure("rewards", r.Name, fmt.Errorf("missing category_ids"))
				continue
			}
			_, err := c.CreateRewards(rc.ctx, frameID, skylight.RewardCreate{Name: r.Name, PointValue: r.PointValue, CategoryIDs: cats, EmojiIcon: r.EmojiIcon, Description: r.Description, RespawnOnRedemption: r.RespawnOnRedemption})
			if err != nil {
				recordFailure("rewards", r.Name, err)
				continue
			}
			created["rewards"]++
		}
	}
	if resources["chores"] {
		for _, ch := range data.Chores {
			in := skylight.ChoreCreate{Summary: ch.Summary, CategoryID: ch.CategoryID, Start: ch.Start, RecurrenceSet: ch.RecurrenceSet, UpForGrabs: ch.UpForGrabs, RewardPoints: ch.RewardPoints, Description: ch.Description, EmojiIcon: ch.EmojiIcon}
			if in.Start == "" {
				in.Start = today()
			}
			var err error
			if ch.UpForGrabs {
				_, err = c.CreateUpForGrabsChore(rc.ctx, frameID, in)
			} else {
				_, err = c.CreateChore(rc.ctx, frameID, in)
			}
			if err != nil {
				recordFailure("chores", ch.Summary, err)
				continue
			}
			created["chores"]++
		}
	}
	if resources["lists"] {
		importLists(rc, frameID, data.Lists, created, &failures)
	}
	if resources["recipes"] {
		importRecipes(rc, frameID, data.Recipes, created, &failures)
	}
	if resources["sittings"] {
		importSittings(rc, frameID, data.MealSittings, created, &failures)
	}
	if resources["calendar"] {
		importCalendarEvents(rc, frameID, data.CalendarEvents, created, &failures)
	}
	result["failed"] = failures
	_ = rc.out.JSON(result)
	if len(failures) > 0 {
		return exitErr
	}
	return exitOK
}

func parseResourceSelection(raw string, all []string) (map[string]bool, error) {
	selected := map[string]bool{}
	if raw == "" || raw == "all" {
		for _, r := range all {
			selected[r] = true
		}
		return selected, nil
	}
	valid := map[string]bool{}
	for _, r := range all {
		valid[r] = true
	}
	for _, r := range strings.Split(raw, ",") {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		if !valid[r] {
			return nil, fmt.Errorf("unknown resource %q (valid: %s)", r, strings.Join(all, ","))
		}
		selected[r] = true
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("no resources selected (valid: %s)", strings.Join(all, ","))
	}
	return selected, nil
}

func portableChores(chores []skylight.Chore) []portableChore {
	out := make([]portableChore, 0, len(chores))
	for _, ch := range chores {
		var catID int64
		if ch.Relationships.Category.Data != nil {
			catID, _ = strconv.ParseInt(ch.Relationships.Category.Data.ID, 10, 64)
		}
		out = append(out, portableChore{
			ID:            ch.ID,
			Summary:       ch.Attributes.Summary,
			CategoryID:    catID,
			Start:         ch.Attributes.Start,
			RecurrenceSet: ch.Attributes.RecurrenceSet,
			UpForGrabs:    ch.Attributes.UpForGrabs,
			RewardPoints:  ch.Attributes.RewardPoints,
			Description:   ptrStringValue(ch.Attributes.Description),
			EmojiIcon:     ptrStringValue(ch.Attributes.EmojiIcon),
		})
	}
	return out
}

func portableRewards(rewards []skylight.Reward) []portableReward {
	out := make([]portableReward, 0, len(rewards))
	for _, r := range rewards {
		var catID int64
		if r.Relationships.Category.Data != nil {
			catID, _ = strconv.ParseInt(r.Relationships.Category.Data.ID, 10, 64)
		}
		categoryIDs := []int64{}
		if catID != 0 {
			categoryIDs = []int64{catID}
		}
		out = append(out, portableReward{
			ID:                  r.ID,
			Name:                r.Attributes.Name,
			PointValue:          r.Attributes.PointValue,
			CategoryID:          catID,
			CategoryIDs:         categoryIDs,
			EmojiIcon:           ptrStringValue(r.Attributes.EmojiIcon),
			Description:         ptrStringValue(r.Attributes.Description),
			RespawnOnRedemption: r.Attributes.RespawnOnRedemption,
		})
	}
	return out
}

func portableCalendarEvents(events []calendarEventEntry) []portableCalendarEvent {
	out := make([]portableCalendarEvent, 0, len(events))
	for _, ev := range events {
		categoryID := ""
		if ev.Relationships.Category.Data != nil {
			categoryID = ev.Relationships.Category.Data.ID
		}
		out = append(out, portableCalendarEvent{
			ID:          ev.ID,
			Summary:     ev.Attributes.Summary,
			StartsAt:    ev.Attributes.StartsAt,
			EndsAt:      ev.Attributes.EndsAt,
			AllDay:      ev.Attributes.AllDay,
			Color:       ev.Attributes.Color,
			CategoryID:  categoryID,
			Description: ev.Attributes.Description,
		})
	}
	return out
}

func fetchRecipes(rc *runCtx, frameID int64) ([]recipeEntry, error) {
	c, err := rc.client()
	if err != nil {
		return nil, err
	}
	recipes, err := c.ListRecipes(rc.ctx, frameID)
	if err != nil {
		return nil, err
	}
	return recipes.Data, nil
}

func portableRecipes(recipes []recipeEntry) []portableRecipe {
	out := make([]portableRecipe, 0, len(recipes))
	for _, r := range recipes {
		catID := ""
		if r.Relationships.MealCategory.Data != nil {
			catID = r.Relationships.MealCategory.Data.ID
		}
		out = append(out, portableRecipe{ID: r.ID, Summary: r.Attributes.Summary, Description: r.Attributes.Description, Ingredients: r.Attributes.Ingredients, URL: r.Attributes.URL, MealCategoryID: catID})
	}
	return out
}

func fetchSittings(rc *runCtx, frameID int64, start, end string) ([]sittingEntry, error) {
	c, err := rc.client()
	if err != nil {
		return nil, err
	}
	sittings, err := c.ListMealSittings(rc.ctx, frameID, skylight.MealSittingFilter{StartDate: start, EndDate: end})
	if err != nil {
		return nil, err
	}
	return sittings.Data, nil
}

func portableSittings(sittings []sittingEntry) []portableMealSitting {
	out := make([]portableMealSitting, 0, len(sittings))
	for _, s := range sittings {
		recipeID, catID := "", ""
		if s.Relationships.MealRecipe.Data != nil {
			recipeID = s.Relationships.MealRecipe.Data.ID
		}
		if s.Relationships.MealCategory.Data != nil {
			catID = s.Relationships.MealCategory.Data.ID
		}
		out = append(out, portableMealSitting{ID: s.ID, Summary: s.Attributes.Summary, Date: s.Attributes.Date, RecipeID: recipeID, MealCategoryID: catID})
	}
	return out
}

func importCounts(data portableExport, resources map[string]bool) map[string]int {
	counts := map[string]int{}
	if resources["chores"] {
		counts["chores"] = len(data.Chores)
	}
	if resources["rewards"] {
		counts["rewards"] = len(data.Rewards)
	}
	if resources["lists"] {
		counts["lists"] = len(data.Lists)
	}
	if resources["recipes"] {
		counts["recipes"] = len(data.Recipes)
	}
	if resources["sittings"] {
		counts["sittings"] = len(data.MealSittings)
	}
	if resources["calendar"] {
		counts["calendar"] = len(data.CalendarEvents)
	}
	return counts
}

func importLists(rc *runCtx, frameID int64, lists []portableList, created map[string]int, failures *[]map[string]string) {
	for _, l := range lists {
		payload := map[string]any{"label": l.Label, "kind": l.Kind, "color": l.Color, "hide_from_frame": l.HideFromFrame}
		c, err := rc.client()
		if err != nil {
			*failures = append(*failures, map[string]string{"resource": "lists", "name": l.Label, "error": err.Error()})
			continue
		}
		doc, err := c.CreateList(rc.ctx, frameID, payload)
		if err != nil {
			*failures = append(*failures, map[string]string{"resource": "lists", "name": l.Label, "error": err.Error()})
			continue
		}
		created["lists"]++
		for _, item := range l.Items {
			itemPayload := map[string]any{"label": item.Label, "status": item.Status, "position": item.Position}
			if _, err := c.CreateListItem(rc.ctx, frameID, doc.Data.ID, itemPayload); err != nil {
				*failures = append(*failures, map[string]string{"resource": "list_items", "name": item.Label, "error": err.Error()})
			}
		}
	}
}

func importRecipes(rc *runCtx, frameID int64, recipes []portableRecipe, created map[string]int, failures *[]map[string]string) {
	if len(recipes) == 0 {
		return
	}
	c, err := rc.client()
	if err != nil {
		for _, recipe := range recipes {
			*failures = append(*failures, map[string]string{"resource": "recipes", "name": recipe.Summary, "error": err.Error()})
		}
		return
	}
	for _, r := range recipes {
		payload := map[string]any{"summary": r.Summary, "description": r.Description, "ingredients": r.Ingredients, "url": r.URL, "meal_category_id": r.MealCategoryID}
		if _, err := c.CreateRecipe(rc.ctx, frameID, payload); err != nil {
			*failures = append(*failures, map[string]string{"resource": "recipes", "name": r.Summary, "error": err.Error()})
			continue
		}
		created["recipes"]++
	}
}

func importSittings(rc *runCtx, frameID int64, sittings []portableMealSitting, created map[string]int, failures *[]map[string]string) {
	if len(sittings) == 0 {
		return
	}
	c, err := rc.client()
	if err != nil {
		for _, sitting := range sittings {
			*failures = append(*failures, map[string]string{"resource": "sittings", "name": sitting.Summary, "error": err.Error()})
		}
		return
	}
	for _, s := range sittings {
		payload := map[string]any{"summary": s.Summary, "date": s.Date, "meal_recipe_id": s.RecipeID, "meal_category_id": s.MealCategoryID}
		if _, err := c.CreateMealSitting(rc.ctx, frameID, payload); err != nil {
			*failures = append(*failures, map[string]string{"resource": "sittings", "name": s.Summary, "error": err.Error()})
			continue
		}
		created["sittings"]++
	}
}

func importCalendarEvents(rc *runCtx, frameID int64, events []portableCalendarEvent, created map[string]int, failures *[]map[string]string) {
	if len(events) == 0 {
		return
	}
	c, err := rc.client()
	if err != nil {
		for _, event := range events {
			*failures = append(*failures, map[string]string{"resource": "calendar", "name": event.Summary, "error": err.Error()})
		}
		return
	}
	for _, ev := range events {
		payload := map[string]any{"summary": ev.Summary, "starts_at": ev.StartsAt, "ends_at": ev.EndsAt, "all_day": ev.AllDay}
		if ev.Color != "" {
			payload["color"] = ev.Color
		}
		if ev.CategoryID != "" {
			payload["category_id"] = ev.CategoryID
		}
		if ev.Description != "" {
			payload["description"] = ev.Description
		}
		if _, err := c.CreateCalendarEvent(rc.ctx, frameID, payload); err != nil {
			*failures = append(*failures, map[string]string{"resource": "calendar", "name": ev.Summary, "error": err.Error()})
			continue
		}
		created["calendar"]++
	}
}

func ptrStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
