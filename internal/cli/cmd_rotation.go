package cli

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/jwmoss/skycli/internal/skylight"
)

type rotationResult struct {
	Chores []*skylight.Chore `json:"chores"`
}

func runRotations(rc *runCtx, args []string) int {
	if len(args) == 0 {
		return usage(rc, "skycli rotations create --chores A,B --assignee-ids 1,2")
	}
	switch args[0] {
	case "create":
		return rotationCreate(rc, args[1:])
	default:
		return usage(rc, "unknown rotations subcommand: "+args[0])
	}
}

func runRotation(rc *runCtx, args []string) int {
	return runRotations(rc, args)
}

func rotationCreate(rc *runCtx, args []string) int {
	fs := flag.NewFlagSet("rotations create", flag.ContinueOnError)
	fs.SetOutput(rc.stderr)
	frameStr := fs.String("frame", "", "frame ID")
	choreList := fs.String("chores", "", "comma-separated chore titles")
	assigneeList := fs.String("assignee-ids", "", "comma-separated assignee/category IDs")
	startDate := fs.String("start-date", today(), "first week start date YYYY-MM-DD")
	weeks := fs.Int("weeks", 4, "number of weekly rotations")
	points := fs.Int("points", 0, "reward points per chore")
	recurrence := fs.String("recurrence", "", "optional recurrence shorthand/raw RRULE for each created chore")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	chores := parseCSVStrings(*choreList)
	assignees, err := parseCategoryList(*assigneeList)
	if err != nil {
		return usage(rc, "--assignee-ids is required")
	}
	if len(chores) == 0 {
		return usage(rc, "--chores is required")
	}
	if *weeks <= 0 {
		return usage(rc, "--weeks must be greater than 0")
	}
	start, err := time.Parse(dateLayout, *startDate)
	if err != nil {
		return fail(rc, fmt.Errorf("invalid --start-date: %w", err))
	}
	var recurrenceSet []string
	if strings.TrimSpace(*recurrence) != "" {
		rule, err := normalizeRRULE(*recurrence)
		if err != nil {
			return fail(rc, err)
		}
		recurrenceSet = []string{rule}
	}
	frameID, err := resolveFrame(rc, *frameStr)
	if err != nil {
		return fail(rc, err)
	}
	c, err := rc.client()
	if err != nil {
		return fail(rc, err)
	}
	created := make([]*skylight.Chore, 0, len(chores)*(*weeks))
	for week := 0; week < *weeks; week++ {
		due := start.AddDate(0, 0, week*7).Format(dateLayout)
		for choreIdx, title := range chores {
			assigneeID := assignees[(choreIdx+week)%len(assignees)]
			in := skylight.ChoreCreate{
				Summary:       title,
				CategoryID:    assigneeID,
				Start:         due,
				RecurrenceSet: recurrenceSet,
			}
			if *points > 0 {
				p := *points
				in.RewardPoints = &p
			}
			ch, err := c.CreateChore(rc.ctx, frameID, in)
			if err != nil {
				if rc.g.asJSON {
					_ = rc.out.JSON(map[string]any{
						"error":   err.Error(),
						"created": created,
					})
				} else {
					fmt.Fprintf(rc.stderr, "error creating %q for week %d: %v\n", title, week+1, err)
				}
				return exitErr
			}
			created = append(created, ch)
		}
	}
	_ = rc.out.JSON(rotationResult{Chores: created})
	return exitOK
}
