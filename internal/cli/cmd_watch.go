package cli

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jwmoss/skycli/internal/config"
	"github.com/jwmoss/skycli/internal/skylight"
)

type watchState struct {
	SeenRewardIDs map[string]bool `json:"seen_reward_ids"`
	SeenChoreIDs  map[string]bool `json:"seen_chore_ids"`
	SeenEventIDs  map[string]bool `json:"seen_event_ids"`
	seeding       bool
}

func runWatch(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	interval := fs.Duration("interval", 60*time.Second, "poll interval")
	resourcesRaw := fs.String("resources", "all", "comma-separated resources: rewards,chores,calendar")
	persist := fs.Bool("persist", false, "persist seen reward IDs across restarts")
	once := fs.Bool("once", false, "poll once and exit after seeding")
	if err := fs.Parse(args); err != nil {
		return flagError(rc, err)
	}
	if *interval < time.Second {
		return usage(rc, "--interval must be at least 1s")
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	resources, err := parseWatchResourceList(*resourcesRaw)
	if err != nil {
		return usage(rc, err.Error())
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	state := newWatchState()
	statePath := ""
	if *persist {
		user, err := c.GetUser(rc.ctx)
		if err != nil {
			return fail(rc, err)
		}
		if user.ID == "" {
			return fail(rc, fmt.Errorf("user response has no ID for watch persistence"))
		}
		statePath, err = rc.watchStatePath(frameID, user.ID)
		if err != nil {
			return fail(rc, err)
		}
		if err := loadWatchState(statePath, state); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fail(rc, err)
		}
	}
	loc, err := rc.frameLocation(frameID)
	if err != nil {
		return fail(rc, err)
	}
	ctx := rc.ctx
	state.seeding = true
	if err := pollWatch(ctx, rc, frameID, state, resources, loc); err != nil {
		return fail(rc, err)
	}
	state.seeding = false
	if *persist && statePath != "" {
		if err := saveWatchState(statePath, state); err != nil {
			return fail(rc, err)
		}
	}
	if *once {
		if rc.g.asJSON {
			_ = rc.out.JSON(map[string]any{
				"seeded":    true,
				"resources": resources,
				"seen": map[string]int{
					"rewards":  len(state.SeenRewardIDs),
					"chores":   len(state.SeenChoreIDs),
					"calendar": len(state.SeenEventIDs),
				},
			})
		}
		return exitOK
	}
	if !rc.g.asJSON {
		rc.out.Line("watching %s every %s; press Ctrl+C to stop", strings.Join(resources, ","), interval.String())
	}
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			if *persist && statePath != "" {
				if err := saveWatchState(statePath, state); err != nil {
					return fail(rc, err)
				}
			}
			if !rc.g.asJSON {
				rc.out.Line("stopped")
			}
			return exitOK
		case <-ticker.C:
			if err := pollWatch(ctx, rc, frameID, state, resources, loc); err != nil {
				return fail(rc, err)
			}
			if *persist && statePath != "" {
				if err := saveWatchState(statePath, state); err != nil {
					return fail(rc, err)
				}
			}
		}
	}
}

func newWatchState() *watchState {
	return &watchState{
		SeenRewardIDs: map[string]bool{},
		SeenChoreIDs:  map[string]bool{},
		SeenEventIDs:  map[string]bool{},
	}
}

func parseWatchResourceList(raw string) ([]string, error) {
	all := []string{"rewards", "chores", "calendar"}
	selected, err := parseResourceSelection(raw, all)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, r := range all {
		if selected[r] {
			out = append(out, r)
		}
	}
	return out, nil
}

func pollWatch(ctx context.Context, rc *runCtx, frameID int64, state *watchState, resources []string, loc *time.Location) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c, err := rc.client()
	if err != nil {
		return err
	}
	todayDate := time.Now().In(loc).Format(dateLayout)
	for _, resource := range resources {
		switch resource {
		case "rewards":
			err = pollWatchRewards(rc, c, frameID, state)
		case "chores":
			err = pollWatchChores(rc, c, frameID, state, todayDate)
		case "calendar":
			err = pollWatchCalendar(rc, c, frameID, state, todayDate)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func pollWatchRewards(rc *runCtx, c *skylight.Client, frameID int64, state *watchState) error {
	rewards, err := c.ListRewards(rc.ctx, frameID)
	if err != nil {
		return fmt.Errorf("watch rewards: %w", err)
	}
	for _, r := range rewards {
		if r.Attributes.RedeemedAt == nil || state.SeenRewardIDs[r.ID] {
			continue
		}
		state.SeenRewardIDs[r.ID] = true
		if state.seeding {
			continue
		}
		event := map[string]any{"type": "reward_redeemed", "id": r.ID, "title": r.Attributes.Name, "points": r.Attributes.PointValue, "ts": time.Now().Format(time.RFC3339)}
		printWatchEvent(rc, event, "REWARD REDEEMED %s (%d pts)", r.Attributes.Name, r.Attributes.PointValue)
	}
	return nil
}

func pollWatchChores(rc *runCtx, c *skylight.Client, frameID int64, state *watchState, todayDate string) error {
	chores, err := c.ListChores(rc.ctx, frameID, skylight.ChoreFilter{
		Date:        todayDate,
		After:       todayDate,
		Before:      todayDate,
		Status:      "complete",
		IncludeLate: true,
	})
	if err != nil {
		return fmt.Errorf("watch chores: %w", err)
	}
	for _, ch := range chores {
		if state.SeenChoreIDs[ch.ID] {
			continue
		}
		state.SeenChoreIDs[ch.ID] = true
		if state.seeding {
			continue
		}
		event := map[string]any{"type": "chore_completed", "id": ch.ID, "title": ch.Attributes.Summary, "ts": time.Now().Format(time.RFC3339)}
		printWatchEvent(rc, event, "CHORE COMPLETED %s", ch.Attributes.Summary)
	}
	return nil
}

func pollWatchCalendar(rc *runCtx, c *skylight.Client, frameID int64, state *watchState, todayDate string) error {
	events, err := c.ListCalendarEvents(rc.ctx, frameID, skylight.CalendarEventFilter{StartDate: todayDate, EndDate: todayDate})
	if err != nil {
		return fmt.Errorf("watch calendar: %w", err)
	}
	now := time.Now()
	for _, ev := range events.Data {
		if state.SeenEventIDs[ev.ID] || ev.Attributes.AllDay || ev.Attributes.StartsAt == "" {
			continue
		}
		start, err := time.Parse(time.RFC3339, ev.Attributes.StartsAt)
		if err != nil {
			continue
		}
		until := time.Until(start)
		if until <= 0 || until > time.Hour {
			continue
		}
		state.SeenEventIDs[ev.ID] = true
		if state.seeding {
			continue
		}
		event := map[string]any{"type": "event_soon", "id": ev.ID, "title": ev.Attributes.Summary, "start_at": ev.Attributes.StartsAt, "minutes_until": int(until.Minutes()), "ts": now.Format(time.RFC3339)}
		printWatchEvent(rc, event, "EVENT SOON %s starts in %d min", ev.Attributes.Summary, int(until.Minutes()))
	}
	return nil
}

func printWatchEvent(rc *runCtx, event map[string]any, format string, args ...any) {
	if rc.g.asJSON {
		_ = rc.out.JSON(event)
		return
	}
	rc.out.Line("[%s] "+format, append([]any{time.Now().Format("15:04:05")}, args...)...)
}

func (rc *runCtx) watchStatePath(frameID int64, userID string) (string, error) {
	cfgPath := rc.g.configPath
	if cfgPath == "" {
		var err error
		cfgPath, err = config.DefaultPath()
		if err != nil {
			return "", err
		}
	}
	key := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d", rc.cfg.BaseURL, userID, frameID)))
	return fmt.Sprintf("%s.watch-%x.json", cfgPath, key[:12]), nil
}

func loadWatchState(path string, state *watchState) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, state); err != nil {
		return err
	}
	if state.SeenRewardIDs == nil {
		state.SeenRewardIDs = map[string]bool{}
	}
	if state.SeenChoreIDs == nil {
		state.SeenChoreIDs = map[string]bool{}
	}
	if state.SeenEventIDs == nil {
		state.SeenEventIDs = map[string]bool{}
	}
	return nil
}

func saveWatchState(path string, state *watchState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	f, err := os.CreateTemp(filepath.Dir(path), ".skycli-watch-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	defer func() { _ = f.Close() }()
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
