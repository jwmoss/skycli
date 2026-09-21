package cli

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/jwmoss/skycli/internal/skylight"
)

type bountyResult struct {
	Chore  *skylight.Chore  `json:"chore,omitempty"`
	Reward *skylight.Reward `json:"reward,omitempty"`
}

func runBounties(rc *runCtx, args []string) int {
	if len(args) == 0 {
		return bountiesList(rc, nil)
	}
	switch args[0] {
	case "list":
		return bountiesList(rc, args[1:])
	case "create":
		return bountiesCreate(rc, args[1:])
	case "update":
		return bountiesUpdate(rc, args[1:])
	case "delete":
		return bountiesDelete(rc, args[1:])
	default:
		return usage(rc, "unknown bounties subcommand: "+args[0])
	}
}

func runBounty(rc *runCtx, args []string) int {
	return runBounties(rc, args)
}

func bountiesCreate(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("bounties create", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	title := fs.String("title", "", "chore title")
	points := fs.Int("points", 0, "point value")
	assigneeID := fs.String("assignee-id", "", "assignee/category ID")
	dueDate := fs.String("due-date", today(), "due date YYYY-MM-DD")
	rewardTitle := fs.String("reward-title", "", "reward title")
	emoji := fs.String("emoji-icon", "", "reward emoji icon")
	recurring := fs.Bool("recurring", false, "make chore recurring")
	categoryIDs := fs.String("category-ids", "", "comma-separated reward category IDs")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if strings.TrimSpace(*title) == "" || *points <= 0 || strings.TrimSpace(*rewardTitle) == "" {
		return usage(rc, "--title, --points (>0), and --reward-title are required")
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	catID, err := parseInt64Flag(*assigneeID, "assignee-id")
	if err != nil {
		return fail(rc, err)
	}
	rewardCats := []int64{catID}
	if strings.TrimSpace(*categoryIDs) != "" {
		rewardCats, err = parseCategoryList(*categoryIDs)
		if err != nil {
			return fail(rc, err)
		}
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	chore := skylight.ChoreCreate{
		Summary:       *title,
		CategoryID:    catID,
		Start:         *dueDate,
		RecurrenceSet: []string{},
		RewardPoints:  points,
	}
	if *recurring {
		chore.RecurrenceSet = []string{"RRULE:FREQ=DAILY;INTERVAL=1"}
	}
	createdChore, err := c.CreateChore(rc.ctx, frameID, chore)
	if err != nil {
		return fail(rc, fmt.Errorf("create bounty chore: %w", err))
	}
	rewards, err := c.CreateRewards(rc.ctx, frameID, skylight.RewardCreate{
		Name:        *rewardTitle,
		PointValue:  *points,
		CategoryIDs: rewardCats,
		EmojiIcon:   *emoji,
	})
	if err != nil {
		_ = c.DeleteChore(rc.ctx, frameID, mustParseID(createdChore.ID), "all")
		return fail(rc, fmt.Errorf("create bounty reward: %w", err))
	}
	var reward *skylight.Reward
	if len(rewards) > 0 {
		reward = &rewards[0]
	}
	_ = rc.out.JSON(bountyResult{Chore: createdChore, Reward: reward})
	return exitOK
}

func bountiesList(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("bounties list", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	fs.String("frame", "", "frame ID")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	_, _ = fmt.Fprintln(rc.stderr, "skycli has no verified bounty links. Use chores list and rewards list; keep explicit IDs from bounties create.")
	if err := rc.out.JSON([]bountyResult{}); err != nil {
		return fail(rc, err)
	}
	return exitOK
}

func bountiesUpdate(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("bounties update", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	choreID := fs.String("chore-id", "", "chore ID")
	rewardID := fs.String("reward-id", "", "reward ID")
	title := fs.String("title", "", "new chore title")
	rewardTitle := fs.String("reward-title", "", "new reward title")
	points := fs.Int("points", -1, "new point value")
	emoji := fs.String("emoji-icon", "", "new reward emoji icon")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if err := requireFlagValue(*choreID, "chore-id"); err != nil {
		return usage(rc, err.Error())
	}
	if err := requireFlagValue(*rewardID, "reward-id"); err != nil {
		return usage(rc, err.Error())
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	rewardNum, err := parseInt64Flag(*rewardID, "reward-id")
	if err != nil {
		return fail(rc, err)
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	choreUpdate := skylight.ChoreUpdate{}
	if *title != "" {
		choreUpdate.Summary = title
	}
	if *points >= 0 {
		choreUpdate.RewardPoints = points
	}
	chore, err := c.UpdateChore(rc.ctx, frameID, *choreID, choreUpdate)
	if err != nil {
		return fail(rc, fmt.Errorf("update bounty chore: %w", err))
	}
	rewardUpdate := skylight.RewardUpdate{}
	if *rewardTitle != "" {
		rewardUpdate.Name = rewardTitle
	}
	if *points >= 0 {
		rewardUpdate.PointValue = points
	}
	if *emoji != "" {
		rewardUpdate.EmojiIcon = emoji
	}
	reward, err := c.UpdateReward(rc.ctx, frameID, rewardNum, rewardUpdate)
	if err != nil {
		return failBountyPartial(rc, "update", map[string]any{"chore": chore}, fmt.Errorf("update bounty reward: %w", err))
	}
	_ = rc.out.JSON(bountyResult{Chore: chore, Reward: reward})
	return exitOK
}

func bountiesDelete(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("bounties delete", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	choreID := fs.String("chore-id", "", "chore ID")
	rewardID := fs.String("reward-id", "", "reward ID")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if err := requireFlagValue(*choreID, "chore-id"); err != nil {
		return usage(rc, err.Error())
	}
	if err := requireFlagValue(*rewardID, "reward-id"); err != nil {
		return usage(rc, err.Error())
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	choreNum, err := parseInt64Flag(*choreID, "chore-id")
	if err != nil {
		return fail(rc, err)
	}
	rewardNum, err := parseInt64Flag(*rewardID, "reward-id")
	if err != nil {
		return fail(rc, err)
	}
	if err := c.DeleteChore(rc.ctx, frameID, choreNum, "all"); err != nil {
		return fail(rc, fmt.Errorf("delete bounty chore: %w", err))
	}
	if err := c.DeleteReward(rc.ctx, frameID, rewardNum); err != nil {
		return failBountyPartial(rc, "delete", map[string]any{"deleted_chore": *choreID}, fmt.Errorf("delete bounty reward: %w", err))
	}
	_ = rc.out.JSON(map[string]any{"deleted_chore": *choreID, "deleted_reward": *rewardID})
	return exitOK
}

func failBountyPartial(rc *runCtx, operation string, applied map[string]any, err error) int {
	_ = rc.out.JSON(map[string]any{
		"error":     err.Error(),
		"operation": operation,
		"partial":   true,
		"applied":   applied,
	})
	if !rc.g.asJSON {
		_, _ = fmt.Fprintln(rc.stderr, "error:", err.Error())
	}
	return exitErr
}

func mustParseID(s string) int64 {
	id, _ := strconv.ParseInt(strings.Split(s, "-")[0], 10, 64)
	return id
}
