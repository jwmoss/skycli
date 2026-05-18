package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/jwmoss/skycli/internal/skylight"
)

func runRewards(rc *runCtx, args []string) int {
	if len(args) == 0 {
		return rewardsList(rc, nil)
	}
	switch args[0] {
	case "list":
		return rewardsList(rc, args[1:])
	case "create":
		return rewardsCreate(rc, args[1:])
	case "update":
		return rewardsUpdate(rc, args[1:])
	case "delete":
		return rewardsDelete(rc, args[1:])
	case "redeem":
		return rewardsRedeem(rc, args[1:], true)
	case "unredeem":
		return rewardsRedeem(rc, args[1:], false)
	case "bulk":
		return rewardsBulk(rc, args[1:])
	case "points":
		return rewardPointsCmd(rc, args[1:])
	default:
		return usage(rc, fmt.Sprintf("unknown rewards subcommand: %s", args[0]))
	}
}

func rewardsList(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("rewards list", flag.ContinueOnError)
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
	rewards, err := c.ListRewards(rc.ctx, frameID)
	if err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(rewards)
		return exitOK
	}
	rows := make([][]string, 0, len(rewards))
	for _, r := range rewards {
		catID := "-"
		if r.Relationships.Category.Data != nil {
			catID = r.Relationships.Category.Data.ID
		}
		redeemed := "-"
		if r.Attributes.RedeemedAt != nil {
			redeemed = *r.Attributes.RedeemedAt
		}
		rows = append(rows, []string{
			r.ID, catID, truncate(r.Attributes.Name, 36),
			strconv.Itoa(r.Attributes.PointValue),
			boolYN(r.Attributes.RespawnOnRedemption),
			redeemed,
		})
	}
	rc.out.Table([]string{"ID", "CAT", "NAME", "COST", "RESPAWN", "REDEEMED"}, rows)
	return exitOK
}

func rewardsCreate(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("rewards create", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	name := fs.String("name", "", "reward name (required)")
	points := fs.Int("points", 0, "point cost (required, >0)")
	categories := fs.String("categories", "", "comma-separated category IDs (required) — multi creates one per kid")
	desc := fs.String("description", "", "optional description")
	emoji := fs.String("emoji", "", "optional emoji_icon")
	respawn := fs.Bool("respawn", false, "auto-recreate after redemption")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if strings.TrimSpace(*name) == "" || *points <= 0 || strings.TrimSpace(*categories) == "" {
		return usage(rc, "--name, --points (>0), --categories are required")
	}
	cats, err := parseCategoryList(*categories)
	if err != nil {
		return fail(rc, err)
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	out, err := c.CreateRewards(rc.ctx, frameID, skylight.RewardCreate{
		Name:                *name,
		PointValue:          *points,
		CategoryIDs:         cats,
		Description:         *desc,
		EmojiIcon:           *emoji,
		RespawnOnRedemption: *respawn,
	})
	if err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(out)
	} else {
		for _, r := range out {
			catID := "-"
			if r.Relationships.Category.Data != nil {
				catID = r.Relationships.Category.Data.ID
			}
			rc.out.Line("created id=%s cat=%s name=%q cost=%d", r.ID, catID, r.Attributes.Name, r.Attributes.PointValue)
		}
	}
	return exitOK
}

func rewardsUpdate(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("rewards update", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	idStr := fs.String("id", "", "reward ID (required)")
	name := fs.String("name", "", "new reward name")
	points := fs.Int("points", -1, "new point cost (0+)")
	desc := fs.String("description", "", "new description")
	emoji := fs.String("emoji", "", "new emoji_icon")
	respawn := fs.Bool("respawn", false, "set respawn_on_redemption=true")
	noRespawn := fs.Bool("no-respawn", false, "set respawn_on_redemption=false")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if strings.TrimSpace(*idStr) == "" {
		return usage(rc, "--id is required")
	}
	id, err := parseInt64Flag(*idStr, "id")
	if err != nil {
		return fail(rc, err)
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	in := skylight.RewardUpdate{}
	var changed bool
	if strings.TrimSpace(*name) != "" {
		v := *name
		in.Name = &v
		changed = true
	}
	if *points >= 0 {
		v := *points
		in.PointValue = &v
		changed = true
	}
	if strings.TrimSpace(*desc) != "" {
		v := *desc
		in.Description = &v
		changed = true
	}
	if strings.TrimSpace(*emoji) != "" {
		v := *emoji
		in.EmojiIcon = &v
		changed = true
	}
	if *respawn && *noRespawn {
		return usage(rc, "choose only one of --respawn or --no-respawn")
	}
	if *respawn || *noRespawn {
		v := *respawn
		in.RespawnOnRedemption = &v
		changed = true
	}
	if !changed {
		return usage(rc, "provide at least one update field")
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	reward, err := c.UpdateReward(rc.ctx, frameID, id, in)
	if err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(reward)
	} else {
		rc.out.Line("updated reward id=%s name=%q cost=%d", reward.ID, reward.Attributes.Name, reward.Attributes.PointValue)
	}
	return exitOK
}

func rewardsDelete(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("rewards delete", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	idStr := fs.String("id", "", "reward ID (required)")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	id, err := parseInt64Flag(*idStr, "id")
	if err != nil {
		return fail(rc, err)
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	if err := c.DeleteReward(rc.ctx, frameID, id); err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(map[string]any{"deleted": id})
	} else {
		rc.out.Line("deleted reward %d", id)
	}
	return exitOK
}

func rewardsRedeem(rc *runCtx, args []string, redeem bool) int {
	action := "redeem"
	if !redeem {
		action = "unredeem"
	}
	fs := flag.NewFlagSet("rewards "+action, flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	idStr := fs.String("id", "", "reward ID (required)")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if strings.TrimSpace(*idStr) == "" {
		return usage(rc, "--id is required")
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	id, err := parseInt64Flag(*idStr, "id")
	if err != nil {
		return fail(rc, err)
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	if redeem {
		err = c.RedeemReward(rc.ctx, frameID, id)
	} else {
		err = c.UnredeemReward(rc.ctx, frameID, id)
	}
	if err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(map[string]any{action: id})
	} else {
		rc.out.Line("%sed reward %d", action, id)
	}
	return exitOK
}

type bulkReward struct {
	Name                string  `json:"name"`
	PointValue          int     `json:"point_value"`
	CategoryIDs         []int64 `json:"category_ids"`
	Description         string  `json:"description,omitempty"`
	EmojiIcon           string  `json:"emoji_icon,omitempty"`
	RespawnOnRedemption bool    `json:"respawn_on_redemption,omitempty"`
}

func rewardsBulk(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("rewards bulk", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	file := fs.String("file", "-", "JSON array of reward specs; - for stdin")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	var rdr io.Reader
	if *file == "-" {
		rdr = rc.stdin
	} else {
		f, err := os.Open(*file)
		if err != nil {
			return fail(rc, err)
		}
		defer f.Close()
		rdr = f
	}
	var items []bulkReward
	if err := json.NewDecoder(rdr).Decode(&items); err != nil {
		return fail(rc, fmt.Errorf("parse bulk file: %w", err))
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	results := make([]map[string]any, 0, len(items))
	var failures int
	for i, it := range items {
		if it.Name == "" || it.PointValue <= 0 || len(it.CategoryIDs) == 0 {
			results = append(results, map[string]any{"index": i, "ok": false, "error": "name, point_value (>0), category_ids required"})
			failures++
			continue
		}
		out, err := c.CreateRewards(rc.ctx, frameID, skylight.RewardCreate{
			Name:                it.Name,
			PointValue:          it.PointValue,
			CategoryIDs:         it.CategoryIDs,
			Description:         it.Description,
			EmojiIcon:           it.EmojiIcon,
			RespawnOnRedemption: it.RespawnOnRedemption,
		})
		if err != nil {
			results = append(results, map[string]any{"index": i, "ok": false, "name": it.Name, "error": err.Error()})
			failures++
			if !rc.g.asJSON {
				fmt.Fprintf(rc.stderr, "[%d/%d] FAIL %s: %v\n", i+1, len(items), it.Name, err)
			}
			continue
		}
		ids := make([]string, len(out))
		for j, r := range out {
			ids[j] = r.ID
		}
		results = append(results, map[string]any{"index": i, "ok": true, "name": it.Name, "ids": ids})
		if !rc.g.asJSON {
			fmt.Fprintf(rc.stderr, "[%d/%d] OK   %s -> ids=%s\n", i+1, len(items), it.Name, strings.Join(ids, ","))
		}
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(map[string]any{"total": len(items), "failures": failures, "results": results})
	} else {
		fmt.Fprintf(rc.stderr, "done: %d ok, %d failed\n", len(items)-failures, failures)
	}
	if failures > 0 {
		return exitErr
	}
	return exitOK
}

func rewardPointsCmd(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("rewards points", flag.ContinueOnError)
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
	pts, err := c.ListRewardPoints(rc.ctx, frameID)
	if err != nil {
		return fail(rc, err)
	}
	if rc.g.asJSON {
		_ = rc.out.JSON(pts)
		return exitOK
	}
	rows := make([][]string, 0, len(pts))
	for _, p := range pts {
		rows = append(rows, []string{
			strconv.FormatInt(p.CategoryID, 10),
			strconv.Itoa(p.CurrentPointBalance),
			strconv.Itoa(p.LifetimePointsEarned),
		})
	}
	rc.out.Table([]string{"CATEGORY", "CURRENT", "LIFETIME"}, rows)
	return exitOK
}

func parseCategoryList(s string) ([]int64, error) {
	parts := strings.Split(s, ",")
	out := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := skylight.ParseID(p)
		if err != nil {
			return nil, fmt.Errorf("bad category id %q: %w", p, err)
		}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no category ids")
	}
	return out, nil
}
