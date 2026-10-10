package cli

import (
	"flag"
	"strings"
	"time"

	"github.com/jwmoss/skycli/internal/skylight"
)

func runSidekick(rc *runCtx, args []string) int {
	if len(args) == 0 {
		return sidekickStatus(rc, nil)
	}
	switch args[0] {
	case "status":
		return sidekickStatus(rc, args[1:])
	case "history":
		return sidekickHistory(rc, args[1:])
	case "show", "items", "undo":
		return sidekickIntent(rc, args[0], args[1:])
	case "create":
		return sidekickCreate(rc, args[1:])
	case "drafts", "approve":
		return sidekickDrafts(rc, args[0], args[1:])
	default:
		return usage(rc, "unknown sidekick subcommand: "+args[0])
	}
}

func sidekickDrafts(rc *runCtx, operation string, args []string) int {
	fs := flag.NewFlagSet("sidekick "+operation, flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frame := fs.String("frame", "", "frame ID")
	id := fs.String("id", "", "auto-creation intent ID")
	kind := fs.String("kind", "", "events | recipes | meals | lists | list-items")
	timezone := fs.String("timezone", "UTC", "IANA timezone for event drafts")
	ids := fs.String("ids", "", "comma-separated draft IDs to approve")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if err := requireFlagValue(*id, "id"); err != nil {
		return usage(rc, err.Error())
	}
	if fs.NArg() != 0 {
		return usage(rc, "unexpected positional arguments")
	}
	if *kind != "events" && *kind != "recipes" && *kind != "meals" && *kind != "lists" && *kind != "list-items" {
		return usage(rc, "--kind must be events, recipes, meals, lists, or list-items")
	}
	if _, err := time.LoadLocation(*timezone); err != nil {
		return usage(rc, "invalid --timezone")
	}
	selected := parseCSVStrings(*ids)
	if operation == "approve" && (len(selected) == 0 || *kind == "lists") {
		return usage(rc, "provide --ids; approve list-items instead of lists")
	}
	return runFrameResourceJSON(rc, *frame, func(c *skylight.Client, f int64) (any, error) {
		if operation == "approve" {
			return c.ApproveAutoCreationDrafts(rc.ctx, f, *id, *kind, selected)
		}
		return c.ListAutoCreationDrafts(rc.ctx, f, *id, *kind, *timezone)
	})
}

func sidekickIntent(rc *runCtx, operation string, args []string) int {
	fs := flag.NewFlagSet("sidekick "+operation, flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frame := fs.String("frame", "", "frame ID")
	id := fs.String("id", "", "auto-creation intent ID")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if err := requireFlagValue(*id, "id"); err != nil {
		return usage(rc, err.Error())
	}
	if fs.NArg() != 0 {
		return usage(rc, "unexpected positional arguments")
	}
	return runFrameResourceJSON(rc, *frame, func(c *skylight.Client, f int64) (any, error) {
		if operation == "undo" {
			return c.UndoAutoCreationIntent(rc.ctx, f, *id)
		}
		if operation == "items" {
			return c.GetAutoCreationItems(rc.ctx, f, *id)
		}
		return c.GetAutoCreationIntent(rc.ctx, f, *id)
	})
}

func sidekickCreate(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("sidekick create", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frame := fs.String("frame", "", "frame ID")
	engine := fs.String("engine", "", "Skylight Sidekick engine")
	text := fs.String("text", "", "source text or planning request")
	contentURL := fs.String("content-url", "", "recipe source URL")
	draft := fs.Bool("draft-first", true, "create a draft for review")
	body, bodyFile := bodyFlags(fs, rc)
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if fs.NArg() != 0 {
		return usage(rc, "unexpected positional arguments")
	}
	payload, err := readPayload(rc, *body, *bodyFile)
	if err != nil {
		return usage(rc, err.Error())
	}
	addStringIfSet(fs, payload, "engine", "engine", *engine)
	addStringIfSet(fs, payload, "text", "text", *text)
	addStringIfSet(fs, payload, "content-url", "content_url", *contentURL)
	value, _ := payload["engine"].(string)
	if strings.TrimSpace(value) == "" {
		return usage(rc, "--engine or the body engine field is required")
	}
	if _, ok := payload["created_via"]; !ok {
		payload["created_via"] = "app_form"
	}
	if _, ok := payload["draft_first"]; !ok || flagChanged(fs, "draft-first") {
		payload["draft_first"] = *draft
	}
	return runFrameResourceJSON(rc, *frame, func(c *skylight.Client, f int64) (any, error) {
		return c.CreateAutoCreationIntent(rc.ctx, f, payload)
	})
}

func sidekickStatus(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("sidekick status", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	access, err := c.GetPlusAccess(rc.ctx)
	if err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(access)
		return exitOK
	}
	rc.out.Line("Calendar Plus:            %s", boolYN(access.ActiveCalendarPlus))
	rc.out.Line("Active subscriptions:     %d", access.ActiveSubscriptionCount)
	rc.out.Line("Assistant trial eligible: %s", boolYN(access.AssistantTrialEligible))
	rc.out.Line("Bundle entitlement:       %s", boolYN(access.BundleEntitlementAvailable))
	return exitOK
}

func sidekickHistory(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("sidekick history", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
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
	intents, err := c.ListAutoCreationIntents(rc.ctx, frameID)
	if err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(intents)
		return exitOK
	}
	rc.out.Line("Sidekick history: %d intent(s)", len(intents.Data))
	return exitOK
}
