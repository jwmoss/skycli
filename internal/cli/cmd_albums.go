package cli

import (
	"flag"

	"github.com/jwmoss/skycli/internal/skylight"
)

func runAlbums(rc *runCtx, args []string) int {
	if len(args) == 0 {
		return albumsList(rc, nil)
	}
	switch args[0] {
	case "list":
		return albumsList(rc, args[1:])
	case "messages":
		return albumMessages(rc, args[1:])
	case "message-ids":
		return albumMessageIDs(rc, args[1:])
	default:
		return usage(rc, "unknown albums subcommand: "+args[0])
	}
}

func albumsList(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("albums list", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.ListAlbums(rc.ctx, frameID)
	})
}

func albumMessages(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("albums messages", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	albumID := fs.String("album-id", "", "album ID")
	page := fs.Int("page", 1, "messages page")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*albumID, "album-id"); err != nil {
		return usage(rc, err.Error())
	}
	if *page < 1 {
		return usage(rc, "--page must be at least 1")
	}
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.ListAlbumMessages(rc.ctx, frameID, *albumID, *page)
	})
}

func albumMessageIDs(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("albums message-ids", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	albumID := fs.String("album-id", "", "album ID")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*albumID, "album-id"); err != nil {
		return usage(rc, err.Error())
	}
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.ListAlbumMessageIDs(rc.ctx, frameID, *albumID)
	})
}
