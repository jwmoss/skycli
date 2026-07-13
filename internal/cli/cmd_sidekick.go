package cli

import "flag"

func runSidekick(rc *runCtx, args []string) int {
	if len(args) == 0 {
		return sidekickStatus(rc, nil)
	}
	switch args[0] {
	case "status":
		return sidekickStatus(rc, args[1:])
	case "history":
		return sidekickHistory(rc, args[1:])
	default:
		return usage(rc, "unknown sidekick subcommand: "+args[0])
	}
}

func sidekickStatus(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("sidekick status", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
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
		return exitUsage
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
