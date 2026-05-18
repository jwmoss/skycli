package cli

import (
	"flag"
)

func runCategories(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("categories", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID (default: config default)")
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

func resolveFrame(rc *runCtx, fromFlag string) (int64, error) {
	if fromFlag != "" {
		return parseInt64Flag(fromFlag, "frame")
	}
	return rc.requireFrame()
}
