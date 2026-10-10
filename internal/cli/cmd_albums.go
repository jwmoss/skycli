package cli

import (
	"flag"
	"fmt"
	"strings"

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
	case "create", "update":
		return albumSave(rc, args[1:], args[0] == "update")
	case "delete":
		return albumDelete(rc, args[1:])
	case "add", "remove":
		return albumMembership(rc, args[1:], args[0] == "remove")
	default:
		return usage(rc, "unknown albums subcommand: "+args[0])
	}
}

func albumSave(rc *runCtx, args []string, update bool) int {
	name := "albums create"
	if update {
		name = "albums update"
	}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frame := fs.String("frame", "", "frame ID")
	title := fs.String("title", "", "album title (required)")
	var albumID string
	if update {
		fs.StringVar(&albumID, "album-id", "", "album ID (required)")
	}
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if err := requireFlagValue(*title, "title"); err != nil {
		return usage(rc, err.Error())
	}
	var id int64
	if update {
		var err error
		id, err = positiveMediaID(albumID, "album-id")
		if err != nil {
			return usage(rc, err.Error())
		}
	}
	return runFrameResourceJSON(rc, *frame, func(c *skylight.Client, frameID int64) (any, error) {
		if update {
			return c.UpdateAlbum(rc.ctx, frameID, id, *title)
		}
		return c.CreateAlbum(rc.ctx, frameID, *title)
	})
}

func albumDelete(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("albums delete", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frame := fs.String("frame", "", "frame ID")
	albumID := fs.String("album-id", "", "album ID (required)")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	id, err := positiveMediaID(*albumID, "album-id")
	if err != nil {
		return usage(rc, err.Error())
	}
	return runFrameResourceOK(rc, *frame, map[string]any{"deleted": id}, func(c *skylight.Client, frameID int64) error {
		return c.DeleteAlbum(rc.ctx, frameID, id)
	})
}

func albumMembership(rc *runCtx, args []string, remove bool) int {
	name, albumFlag := "albums add", "album-ids"
	if remove {
		name, albumFlag = "albums remove", "album-id"
	}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frame := fs.String("frame", "", "frame ID")
	albums := fs.String(albumFlag, "", "album IDs (required)")
	messages := fs.String("message-ids", "", "comma-separated message IDs (required)")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	albumIDs, err := positiveMediaIDs(*albums, albumFlag)
	if err != nil {
		return usage(rc, err.Error())
	}
	if remove && len(albumIDs) != 1 {
		return usage(rc, "--album-id requires one album ID")
	}
	messageIDs, err := positiveMediaIDs(*messages, "message-ids")
	if err != nil {
		return usage(rc, err.Error())
	}
	return runFrameResourceJSON(rc, *frame, func(c *skylight.Client, frameID int64) (any, error) {
		if remove {
			return c.RemoveMessagesFromAlbum(rc.ctx, frameID, albumIDs[0], messageIDs)
		}
		return c.AddMessagesToAlbums(rc.ctx, frameID, albumIDs, messageIDs)
	})
}

func positiveMediaID(value, name string) (int64, error) {
	id, err := parseInt64Flag(strings.TrimSpace(value), name)
	if err != nil || id < 1 {
		return 0, fmt.Errorf("--%s requires a positive integer ID", name)
	}
	return id, nil
}

func positiveMediaIDs(value, name string) ([]int64, error) {
	parts := strings.Split(value, ",")
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		id, err := positiveMediaID(part, name)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func albumsList(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("albums list", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
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
		return flagError(rc, err)
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
		return flagError(rc, err)
	}
	if err := requireFlagValue(*albumID, "album-id"); err != nil {
		return usage(rc, err.Error())
	}
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.ListAlbumMessageIDs(rc.ctx, frameID, *albumID)
	})
}
