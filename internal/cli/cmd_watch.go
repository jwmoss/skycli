package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

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
		return exitUsage
	}
	if *interval < time.Second {
		return usage(rc, "--interval must be at least 1s")
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	resources := parseWatchResourceList(*resourcesRaw)
	if len(resources) == 0 {
		return usage(rc, "no valid resources selected")
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	state := newWatchState()
	statePath := ""
	if *persist {
		statePath = watchStatePath()
		_ = loadWatchState(statePath, state)
	}
	ctx, stop := signal.NotifyContext(rc.ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	state.seeding = true
	pollWatch(ctx, rc, c, frameID, state, resources)
	state.seeding = false
	if *persist && statePath != "" {
		_ = saveWatchState(statePath, state)
	}
	if *once {
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
				_ = saveWatchState(statePath, state)
			}
			if !rc.g.asJSON {
				rc.out.Line("stopped")
			}
			return exitOK
		case <-ticker.C:
			pollWatch(ctx, rc, c, frameID, state, resources)
			if *persist && statePath != "" {
				_ = saveWatchState(statePath, state)
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

func parseWatchResourceList(raw string) []string {
	all := []string{"rewards", "chores", "calendar"}
	if raw == "" || raw == "all" {
		return all
	}
	valid := map[string]bool{"rewards": true, "chores": true, "calendar": true}
	out := []string{}
	for _, r := range parseCSVStrings(raw) {
		if valid[r] {
			out = append(out, r)
		}
	}
	return out
}

func pollWatch(ctx context.Context, rc *runCtx, c *skylight.Client, frameID int64, state *watchState, resources []string) {
	_ = ctx
	for _, resource := range resources {
		switch resource {
		case "rewards":
			pollWatchRewards(rc, c, frameID, state)
		case "chores":
			pollWatchChores(rc, c, frameID, state)
		case "calendar":
			pollWatchCalendar(rc, frameID, state)
		}
	}
}

func pollWatchRewards(rc *runCtx, c *skylight.Client, frameID int64, state *watchState) {
	rewards, err := c.ListRewards(rc.ctx, frameID)
	if err != nil {
		fmt.Fprintf(rc.stderr, "watch rewards: %v\n", err)
		return
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
}

func pollWatchChores(rc *runCtx, c *skylight.Client, frameID int64, state *watchState) {
	chores, err := c.ListChores(rc.ctx, frameID, skylight.ChoreFilter{Date: today(), Status: "complete", IncludeLate: true})
	if err != nil {
		fmt.Fprintf(rc.stderr, "watch chores: %v\n", err)
		return
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
}

func pollWatchCalendar(rc *runCtx, frameID int64, state *watchState) {
	events, err := fetchCalendarEvents(rc, frameID, today(), today())
	if err != nil {
		fmt.Fprintf(rc.stderr, "watch calendar: %v\n", err)
		return
	}
	now := time.Now()
	for _, ev := range events {
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
}

func printWatchEvent(rc *runCtx, event map[string]any, format string, args ...any) {
	if rc.g.asJSON {
		_ = rc.out.JSON(event)
		return
	}
	rc.out.Line("[%s] "+format, append([]any{time.Now().Format("15:04:05")}, args...)...)
}

func watchStatePath() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		return filepath.Join(".", "skycli-watch-state.json")
	}
	return filepath.Join(dir, "skycli", "watch-state.json")
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
	return os.WriteFile(path, data, 0o600)
}
