package cli

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"

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
		return exitUsage
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
	rewardCats, err := parseCategoryList(*categoryIDs)
	if err != nil && strings.TrimSpace(*categoryIDs) != "" {
		_ = c.DeleteChore(rc.ctx, frameID, mustParseID(createdChore.ID), "all")
		return fail(rc, err)
	}
	if len(rewardCats) == 0 {
		rewardCats = []int64{catID}
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
	now := time.Now()
	chores, err := c.ListChores(rc.ctx, frameID, skylight.ChoreFilter{
		After:       now.AddDate(0, 0, -1).Format("2006-01-02"),
		Before:      now.AddDate(0, 1, 0).Format("2006-01-02"),
		IncludeLate: true,
	})
	if err != nil {
		return fail(rc, err)
	}
	rewards, err := c.ListRewards(rc.ctx, frameID)
	if err != nil {
		return fail(rc, err)
	}
	rewardsByPoints := map[int][]skylight.Reward{}
	for _, r := range rewards {
		if r.Attributes.RedeemedAt == nil {
			rewardsByPoints[r.Attributes.PointValue] = append(rewardsByPoints[r.Attributes.PointValue], r)
		}
	}
	out := []bountyResult{}
	for _, ch := range chores {
		if ch.Attributes.Status != "pending" || ch.Attributes.RewardPoints == nil || *ch.Attributes.RewardPoints <= 0 {
			continue
		}
		rs := rewardsByPoints[*ch.Attributes.RewardPoints]
		if len(rs) == 0 {
			continue
		}
		reward := rs[0]
		rewardsByPoints[*ch.Attributes.RewardPoints] = rs[1:]
		chCopy := ch
		out = append(out, bountyResult{Chore: &chCopy, Reward: &reward})
	}
	_ = rc.out.JSON(out)
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
		return exitUsage
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
	rewardNum, err := parseInt64Flag(*rewardID, "reward-id")
	if err != nil {
		return fail(rc, err)
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
		return fail(rc, fmt.Errorf("update bounty reward: %w", err))
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
		return exitUsage
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
	if err := c.DeleteChore(rc.ctx, frameID, choreNum, "all"); err != nil {
		return fail(rc, fmt.Errorf("delete bounty chore: %w", err))
	}
	rewardNum, err := parseInt64Flag(*rewardID, "reward-id")
	if err != nil {
		return fail(rc, err)
	}
	if err := c.DeleteReward(rc.ctx, frameID, rewardNum); err != nil {
		return fail(rc, fmt.Errorf("delete bounty reward: %w", err))
	}
	_ = rc.out.JSON(map[string]any{"deleted_chore": *choreID, "deleted_reward": *rewardID})
	return exitOK
}

func mustParseID(s string) int64 {
	id, _ := strconv.ParseInt(strings.Split(s, "-")[0], 10, 64)
	return id
}
