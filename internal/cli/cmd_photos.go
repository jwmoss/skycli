package cli

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jwmoss/skycli/internal/skylight"
)

func runPhotos(rc *runCtx, args []string) int {
	if len(args) == 0 {
		return photosList(rc, nil)
	}
	switch args[0] {
	case "list":
		return photosList(rc, args[1:])
	case "upload":
		return photosUpload(rc, args[1:])
	case "delete":
		return photosDelete(rc, args[1:])
	case "download":
		return photosDownload(rc, args[1:])
	default:
		return usage(rc, "unknown photos subcommand: "+args[0])
	}
}

func runPhoto(rc *runCtx, args []string) int {
	return runPhotos(rc, args)
}

func photosList(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("photos list", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	pageToken := fs.String("page-token", "__START__", "pagination token")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	return runFrameResourceJSON(rc, *frameStr, func(c *skylight.Client, frameID int64) (any, error) {
		return c.ListPhotoMessages(rc.ctx, frameID, *pageToken)
	})
}

func photosDelete(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("photos delete", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	ids := fs.String("message-ids", "", "comma-separated message IDs")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*ids, "message-ids"); err != nil {
		return usage(rc, err.Error())
	}
	messageIDs, err := parseCSVInts(*ids, "message-ids")
	if err != nil {
		return fail(rc, err)
	}
	return runFrameResourceOK(rc, *frameStr, map[string]any{"deleted": messageIDs}, func(c *skylight.Client, frameID int64) error {
		return c.DeletePhotoMessages(rc.ctx, frameID, messageIDs)
	})
}

func photosUpload(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("photos upload", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	filePath := fs.String("file", "", "image/video file to upload")
	ext := fs.String("ext", "", "file extension override")
	caption := fs.String("caption", "", "caption")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*filePath, "file"); err != nil {
		return usage(rc, err.Error())
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	data, err := os.ReadFile(*filePath)
	if err != nil {
		return fail(rc, err)
	}
	e := strings.TrimPrefix(*ext, ".")
	if e == "" {
		e = strings.TrimPrefix(strings.ToLower(filepath.Ext(*filePath)), ".")
	}
	if e == "" {
		return usage(rc, "--ext is required when --file has no extension")
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	target, err := c.CreatePhotoUpload(rc.ctx, strings.ToLower(e), []string{formatID(frameID)}, *caption)
	if err != nil {
		return fail(rc, err)
	}
	req, err := http.NewRequestWithContext(rc.ctx, http.MethodPut, target.UploadURL, bytes.NewReader(data))
	if err != nil {
		return fail(rc, err)
	}
	req.Header.Set("Content-Type", contentTypeForExt(e))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fail(rc, err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fail(rc, fmt.Errorf("upload failed with status %d", resp.StatusCode))
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(target)
	} else {
		rc.out.Line("uploaded key=%s messages=%v", target.Key, target.MessageIDs)
	}
	return exitOK
}

func photosDownload(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("photos download", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	assetURL := fs.String("asset-url", "", "asset URL from photos list")
	out := fs.String("out", "", "output file path")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if err := requireFlagValue(*assetURL, "asset-url"); err != nil {
		return usage(rc, err.Error())
	}
	if err := requireFlagValue(*out, "out"); err != nil {
		return usage(rc, err.Error())
	}
	req, err := http.NewRequestWithContext(rc.ctx, http.MethodGet, *assetURL, nil)
	if err != nil {
		return fail(rc, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fail(rc, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fail(rc, fmt.Errorf("download failed with status %d", resp.StatusCode))
	}
	f, err := os.Create(*out)
	if err != nil {
		return fail(rc, err)
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(map[string]any{"downloaded": *out})
	} else {
		rc.out.Line("downloaded %s", *out)
	}
	return exitOK
}

func contentTypeForExt(ext string) string {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "mp4":
		return "video/mp4"
	case "mov":
		return "video/quicktime"
	default:
		return "image/jpeg"
	}
}
