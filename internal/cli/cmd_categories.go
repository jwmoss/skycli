package cli

import (
	"flag"
	"strings"

	"github.com/jwmoss/skycli/internal/skylight"
)

func runCategories(rc *runCtx, args []string) int {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "list":
			return categoriesList(rc, args[1:])
		case "show", "create", "update", "delete":
			return categoriesChange(rc, args[0], args[1:])
		default:
			return usage(rc, "unknown categories subcommand: "+args[0])
		}
	}
	return categoriesList(rc, args)
}

func categoriesList(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("categories", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID (default: config default)")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	cats, err := c.ListCategories(rc.ctx, frameID)
	if err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(cats)
		return exitOK
	}
	rows := make([][]string, 0, len(cats))
	for _, c := range cats {
		rows = append(rows, []string{c.ID, c.Attributes.Label, c.Attributes.Color, boolYN(c.Attributes.LinkedToProfile)})
	}
	rc.out.Table([]string{"ID", "LABEL", "COLOR", "LINKED-TO-PROFILE"}, rows)
	return exitOK
}

func categoriesChange(rc *runCtx, operation string, args []string) int {
	fs := flag.NewFlagSet("categories "+operation, flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frame := fs.String("frame", "", "frame ID")
	id := fs.String("id", "", "category ID")
	var label, color, body, bodyFile, reassign *string
	var profile *bool
	if operation == "create" || operation == "update" {
		label = fs.String("label", "", "profile or label name")
		color = fs.String("color", "", "category color")
		profile = fs.Bool("profile", false, "link this category to a profile")
		body, bodyFile = bodyFlags(fs, rc)
	}
	if operation == "delete" {
		reassign = fs.String("reassign-to", "", "replacement category ID for linked content")
	}
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if fs.NArg() != 0 {
		return usage(rc, "unexpected positional arguments")
	}
	if operation != "create" {
		if value, err := parseInt64Flag(*id, "id"); err != nil || value <= 0 {
			return usage(rc, "--id must be a positive integer")
		}
	}
	if operation == "show" {
		return runFrameResourceJSON(rc, *frame, func(c *skylight.Client, f int64) (any, error) {
			return c.GetCategory(rc.ctx, f, *id)
		})
	}
	if operation == "delete" {
		if *reassign != "" {
			if value, err := parseInt64Flag(*reassign, "reassign-to"); err != nil || value <= 0 {
				return usage(rc, "--reassign-to must be a positive integer")
			}
			if *reassign == *id {
				return usage(rc, "replacement category must differ from --id")
			}
		}
		return runFrameResourceOK(rc, *frame, map[string]any{"deleted": *id}, func(c *skylight.Client, f int64) error {
			return c.DeleteCategory(rc.ctx, f, *id, *reassign)
		})
	}
	payload, err := readPayload(rc, *body, *bodyFile)
	if err != nil {
		return usage(rc, err.Error())
	}
	addStringIfSet(fs, payload, "label", "label", *label)
	addStringIfSet(fs, payload, "color", "color", *color)
	addBoolIfSet(fs, payload, "profile", "linked_to_profile", *profile)
	if linked, ok := payload["linked_to_profile"]; ok {
		payload["selected_for_chore_chart"] = linked
	}
	if operation == "create" {
		for _, field := range []string{"label", "color"} {
			value, _ := payload[field].(string)
			if strings.TrimSpace(value) == "" {
				return usage(rc, "--"+field+" or its body field is required")
			}
		}
	}
	if len(payload) == 0 {
		return usage(rc, "provide at least one update field")
	}
	return runFrameResourceJSON(rc, *frame, func(c *skylight.Client, f int64) (any, error) {
		if operation == "create" {
			return c.CreateCategory(rc.ctx, f, payload)
		}
		return c.UpdateCategory(rc.ctx, f, *id, payload)
	})
}

func resolveFrame(rc *runCtx, fromFlag string) (int64, error) {
	if fromFlag != "" {
		return parseInt64Flag(fromFlag, "frame")
	}
	return rc.requireFrame()
}
