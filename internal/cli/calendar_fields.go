package cli

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

type calendarFields struct {
	categories, recurrence, invites, description, location, emoji, timezone *string
	countdown                                                               *bool
}

func calendarExtraFlags(fs *flag.FlagSet) calendarFields {
	return calendarFields{
		categories:  fs.String("categories", "", "comma-separated profile or label IDs"),
		recurrence:  fs.String("recurrence", "", "daily, weekly:MO,FR, raw RRULE, or none"),
		invites:     fs.String("invite-emails", "", "comma-separated invitation emails"),
		description: fs.String("description", "", "event description"),
		location:    fs.String("location", "", "event location"),
		emoji:       fs.String("emoji", "", "event emoji"),
		timezone:    fs.String("timezone", "", "IANA timezone"),
		countdown:   fs.Bool("countdown", false, "show a countdown for this event"),
	}
}

func (f calendarFields) apply(fs *flag.FlagSet, payload map[string]any, category string) error {
	if category != "" && flagChanged(fs, "categories") {
		return fmt.Errorf("choose only one of --category or --categories")
	}
	if category != "" || flagChanged(fs, "categories") {
		selected := *f.categories
		if category != "" {
			selected = category
		}
		ids := parseCSVStrings(selected)
		if len(ids) == 0 {
			ids = []string{}
		}
		for _, id := range ids {
			if value, err := parseInt64Flag(id, "categories"); err != nil || value <= 0 {
				return fmt.Errorf("category IDs must be positive integers")
			}
		}
		payload["category_ids"] = ids
	}
	if flagChanged(fs, "recurrence") {
		if *f.recurrence == "none" {
			payload["rrule"] = nil
		} else {
			rule, err := normalizeRRULE(*f.recurrence)
			if err != nil {
				return err
			}
			payload["rrule"] = []string{rule}
		}
	}
	if flagChanged(fs, "invite-emails") {
		emails := parseCSVStrings(*f.invites)
		if len(emails) == 0 {
			emails = []string{}
		}
		for _, email := range emails {
			if !strings.Contains(email, "@") {
				return fmt.Errorf("invalid invitation email %q", email)
			}
		}
		payload["invited_emails"] = emails
	}
	if flagChanged(fs, "timezone") {
		if _, err := time.LoadLocation(*f.timezone); err != nil {
			return fmt.Errorf("invalid --timezone: %w", err)
		}
	}
	addStringIfSet(fs, payload, "description", "description", *f.description)
	addStringIfSet(fs, payload, "location", "location", *f.location)
	addStringIfSet(fs, payload, "emoji", "emoji_icon", *f.emoji)
	addStringIfSet(fs, payload, "timezone", "timezone", *f.timezone)
	addBoolIfSet(fs, payload, "countdown", "countdown_enabled", *f.countdown)
	return nil
}
