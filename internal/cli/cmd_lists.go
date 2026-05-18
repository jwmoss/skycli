package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
)

func runLists(rc *runCtx, args []string) int {
	if len(args) == 0 {
		return listsAll(rc, nil)
	}
	switch args[0] {
	case "all", "list":
		return listsAll(rc, args[1:])
	case "info", "show":
		return listsInfo(rc, args[1:])
	case "create":
		return listsCreate(rc, args[1:])
	case "update":
		return listsUpdate(rc, args[1:])
	case "delete":
		return listsDelete(rc, args[1:])
	case "add-item":
		return listsAddItem(rc, args[1:])
	case "update-item":
		return listsUpdateItem(rc, args[1:])
	case "delete-item":
		return listsDeleteItem(rc, args[1:])
	case "clear-completed":
		return listsClearCompleted(rc, args[1:])
	case "organize":
		return listsAction(rc, args[1:], "organize")
	case "order":
		return listsOrder(rc, args[1:])
	case "task-box-item":
		return taskBoxItemCreate(rc, args[1:])
	default:
		return usage(rc, "unknown lists subcommand: "+args[0])
	}
}

func runGrocery(rc *runCtx, args []string) int {
	if len(args) == 0 {
		return listsAllKind(rc, nil, "grocery")
	}
	switch args[0] {
	case "list":
		return listsAllKind(rc, args[1:], "grocery")
	case "show":
		return listsInfo(rc, args[1:])
	case "create":
		return groceryCreate(rc, args[1:])
	case "add":
		return groceryAdd(rc, args[1:])
	case "clear":
		return listsClearCompleted(rc, args[1:])
	case "organize":
		return listsAction(rc, args[1:], "organize")
	case "order":
		return listsOrder(rc, args[1:])
	case "add-recipe":
		return mealAddToGrocery(rc, args[1:])
	default:
		return usage(rc, "unknown grocery subcommand: "+args[0])
	}
}

func listsAll(rc *runCtx, args []string) int {
	return listsAllKind(rc, args, "")
}

func listsAllKind(rc *runCtx, args []string, kind string) int {
	fs := flag.NewFlagSet("lists all", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if kind == "" {
		return doFrameJSON(rc, *frameStr, http.MethodGet, "/api/frames/%d/lists", nil, nil)
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	lists, err := fetchLists(rc, frameID)
	if err != nil {
		return fail(rc, err)
	}
	filtered := make([]listEntry, 0, len(lists))
	for _, l := range lists {
		if l.Attributes.Kind == kind {
			filtered = append(filtered, l)
		}
	}
	_ = rc.out.JSON(map[string]any{"data": filtered})
	return exitOK
}

func listsInfo(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("lists info", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	listID := fs.String("list-id", "", "list ID")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*listID, "list-id"); err != nil {
		return usage(rc, err.Error())
	}
	return doFrameJSON(rc, *frameStr, http.MethodGet, "/api/frames/%d/lists/%s", nil, nil, *listID)
}

func listsCreate(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("lists create", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	title := fs.String("title", "", "list title")
	color := fs.String("color", "#2178AF", "list color")
	kind := fs.String("kind", "to_do", "list kind")
	hide := fs.Bool("hide-from-frame", false, "hide list from frame")
	body, bodyFile := bodyFlags(fs, rc)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	payload, err := readPayload(rc, *body, *bodyFile)
	if err != nil {
		return fail(rc, err)
	}
	if *title != "" {
		payload["label"] = *title
	}
	if *color != "" {
		payload["color"] = *color
	}
	if *kind != "" {
		payload["kind"] = *kind
	}
	addBoolIfSet(fs, payload, "hide-from-frame", "hide_from_frame", *hide)
	return doFrameJSON(rc, *frameStr, http.MethodPost, "/api/frames/%d/lists", nil, payload)
}

func groceryCreate(rc *runCtx, args []string) int {
	args = append([]string{"--kind", "grocery"}, args...)
	return listsCreate(rc, args)
}

func groceryAdd(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("grocery add", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	listID := fs.String("list-id", "", "list ID")
	title := fs.String("title", "", "single item title")
	itemsRaw := fs.String("items", "", "comma-separated item titles")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*listID, "list-id"); err != nil {
		return usage(rc, err.Error())
	}
	items := parseCSVStrings(*itemsRaw)
	if *title != "" {
		items = append([]string{*title}, items...)
	}
	if len(items) == 0 {
		return usage(rc, "--title or --items is required")
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	created := []any{}
	for _, item := range items {
		raw, err := c.Do(rc.ctx, http.MethodPost, fmt.Sprintf("/api/frames/%d/lists/%s/list_items", frameID, *listID), nil, map[string]any{"label": item})
		if err != nil {
			return fail(rc, err)
		}
		var decoded any
		if json.Unmarshal(raw, &decoded) == nil {
			created = append(created, decoded)
		}
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(map[string]any{"count": len(items), "items": created})
	} else {
		rc.out.Line("added %d item(s)", len(items))
	}
	return exitOK
}

func listsUpdate(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("lists update", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	listID := fs.String("list-id", "", "list ID")
	title := fs.String("title", "", "list title")
	color := fs.String("color", "", "list color")
	kind := fs.String("kind", "", "list kind")
	hide := fs.Bool("hide-from-frame", false, "hide list from frame")
	body, bodyFile := bodyFlags(fs, rc)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*listID, "list-id"); err != nil {
		return usage(rc, err.Error())
	}
	payload, err := readPayload(rc, *body, *bodyFile)
	if err != nil {
		return fail(rc, err)
	}
	addStringIfSet(fs, payload, "title", "label", *title)
	addStringIfSet(fs, payload, "color", "color", *color)
	addStringIfSet(fs, payload, "kind", "kind", *kind)
	addBoolIfSet(fs, payload, "hide-from-frame", "hide_from_frame", *hide)
	return doFrameJSON(rc, *frameStr, http.MethodPut, "/api/frames/%d/lists/%s", nil, payload, *listID)
}

func listsDelete(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("lists delete", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	listID := fs.String("list-id", "", "list ID")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*listID, "list-id"); err != nil {
		return usage(rc, err.Error())
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	return doNoContent(rc, http.MethodDelete, "/api/frames/"+formatID(frameID)+"/lists/"+*listID, nil, nil, map[string]any{"deleted": *listID})
}

func listsAddItem(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("lists add-item", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	listID := fs.String("list-id", "", "list ID")
	title := fs.String("title", "", "item title")
	position := fs.Int("position", 0, "item position")
	completed := fs.Bool("completed", false, "mark item completed")
	body, bodyFile := bodyFlags(fs, rc)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*listID, "list-id"); err != nil {
		return usage(rc, err.Error())
	}
	payload, err := readPayload(rc, *body, *bodyFile)
	if err != nil {
		return fail(rc, err)
	}
	if *title != "" {
		payload["label"] = *title
	}
	addIntIfSet(fs, payload, "position", "position", *position)
	if flagChanged(fs, "completed") && *completed {
		payload["status"] = "completed"
	}
	return doFrameJSON(rc, *frameStr, http.MethodPost, "/api/frames/%d/lists/%s/list_items", nil, payload, *listID)
}

func listsUpdateItem(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("lists update-item", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	listID := fs.String("list-id", "", "list ID")
	itemID := fs.String("item-id", "", "item ID")
	title := fs.String("title", "", "item title")
	position := fs.Int("position", 0, "item position")
	completed := fs.Bool("completed", false, "mark item completed")
	pending := fs.Bool("pending", false, "mark item pending")
	body, bodyFile := bodyFlags(fs, rc)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*listID, "list-id"); err != nil {
		return usage(rc, err.Error())
	}
	if err := requireFlagValue(*itemID, "item-id"); err != nil {
		return usage(rc, err.Error())
	}
	payload, err := readPayload(rc, *body, *bodyFile)
	if err != nil {
		return fail(rc, err)
	}
	addStringIfSet(fs, payload, "title", "label", *title)
	addIntIfSet(fs, payload, "position", "position", *position)
	if *completed && *pending {
		return usage(rc, "choose only one of --completed or --pending")
	}
	if flagChanged(fs, "completed") && *completed {
		payload["status"] = "completed"
	}
	if flagChanged(fs, "pending") && *pending {
		payload["status"] = "pending"
	}
	return doFrameJSON(rc, *frameStr, http.MethodPut, "/api/frames/%d/lists/%s/list_items/%s", nil, payload, *listID, *itemID)
}

func listsDeleteItem(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("lists delete-item", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	listID := fs.String("list-id", "", "list ID")
	itemID := fs.String("item-id", "", "item ID")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*listID, "list-id"); err != nil {
		return usage(rc, err.Error())
	}
	if err := requireFlagValue(*itemID, "item-id"); err != nil {
		return usage(rc, err.Error())
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	path := "/api/frames/" + formatID(frameID) + "/lists/" + *listID + "/list_items/" + *itemID
	return doNoContent(rc, http.MethodDelete, path, nil, nil, map[string]any{"deleted": *itemID})
}

func listsAction(rc *runCtx, args []string, action string) int {
	fs := flag.NewFlagSet("lists "+action, flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	listID := fs.String("list-id", "", "list ID")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*listID, "list-id"); err != nil {
		return usage(rc, err.Error())
	}
	return doFrameJSON(rc, *frameStr, http.MethodPost, "/api/frames/%d/lists/%s/"+action, nil, nil, *listID)
}

func listsOrder(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("lists order", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	listID := fs.String("list-id", "", "list ID")
	retailer := fs.String("retailer", "", "retailer slug")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*listID, "list-id"); err != nil {
		return usage(rc, err.Error())
	}
	body := map[string]any{}
	if *retailer != "" {
		body["retailer"] = *retailer
	}
	return doFrameJSON(rc, *frameStr, http.MethodPost, "/api/frames/%d/lists/%s/order", nil, body, *listID)
}

func taskBoxItemCreate(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("lists task-box-item", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	title := fs.String("title", "", "task box item title")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*title, "title"); err != nil {
		return usage(rc, err.Error())
	}
	body := map[string]any{"task_box_item": map[string]any{"title": *title}}
	return doFrameJSON(rc, *frameStr, http.MethodPost, "/api/frames/%d/task_box_items", nil, body)
}

func listsClearCompleted(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("lists clear-completed", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	listID := fs.String("list-id", "", "list ID")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*listID, "list-id"); err != nil {
		return usage(rc, err.Error())
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	items, err := fetchListItems(rc, frameID, *listID)
	if err != nil {
		return fail(rc, err)
	}
	deleted := []string{}
	for _, item := range items {
		if item.Attributes.Status != "completed" {
			continue
		}
		path := "/api/frames/" + formatID(frameID) + "/lists/" + *listID + "/list_items/" + item.ID
		c, err := rc.client()
		if err != nil {
			return fail(rc, err)
		}
		if _, err := c.Do(rc.ctx, http.MethodDelete, path, nil, nil); err != nil {
			return fail(rc, fmt.Errorf("delete item %s: %w", item.ID, err))
		}
		deleted = append(deleted, item.ID)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(map[string]any{"deleted": deleted, "count": len(deleted)})
	} else {
		rc.out.Line("cleared %d completed item(s)", len(deleted))
	}
	return exitOK
}

type listEntry struct {
	ID         string `json:"id"`
	Type       string `json:"type,omitempty"`
	Attributes struct {
		Label         string `json:"label"`
		Color         string `json:"color"`
		Kind          string `json:"kind"`
		HideFromFrame bool   `json:"hide_from_frame"`
	} `json:"attributes"`
}

type listItemEntry struct {
	ID         string `json:"id"`
	Type       string `json:"type,omitempty"`
	Attributes struct {
		Label    string `json:"label"`
		Status   string `json:"status"`
		Position int    `json:"position"`
	} `json:"attributes"`
}

func fetchLists(rc *runCtx, frameID int64) ([]listEntry, error) {
	c, err := rc.client()
	if err != nil {
		return nil, err
	}
	raw, err := c.Do(rc.ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/lists", frameID), nil, nil)
	if err != nil {
		return nil, err
	}
	var env struct {
		Data []listEntry `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

func fetchListItems(rc *runCtx, frameID int64, listID string) ([]listItemEntry, error) {
	c, err := rc.client()
	if err != nil {
		return nil, err
	}
	raw, err := c.Do(rc.ctx, http.MethodGet, fmt.Sprintf("/api/frames/%d/lists/%s", frameID, listID), nil, nil)
	if err != nil {
		return nil, err
	}
	var env struct {
		Included []listItemEntry `json:"included"`
		Data     struct {
			Relationships map[string]any `json:"relationships"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	items := make([]listItemEntry, 0, len(env.Included))
	for _, item := range env.Included {
		if item.Type == "" || item.Type == "list_item" {
			items = append(items, item)
		}
	}
	return items, nil
}
