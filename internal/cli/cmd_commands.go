package cli

type flagDoc struct {
	Name    string `json:"name"`
	Value   string `json:"value,omitempty"`
	Summary string `json:"summary"`
}

type subcommandDoc struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Mutates bool   `json:"mutates,omitempty"`
}

type catalogCommandDoc struct {
	Name        string          `json:"name"`
	Aliases     []string        `json:"aliases,omitempty"`
	Summary     string          `json:"summary"`
	Docs        string          `json:"docs"`
	Subcommands []subcommandDoc `json:"subcommands,omitempty"`
	ReadOnly    bool            `json:"read_only"`
	Mutates     bool            `json:"mutates"`
	Examples    []string        `json:"examples,omitempty"`
}

type commandCatalogDoc struct {
	Name           string              `json:"name"`
	Summary        string              `json:"summary"`
	Usage          string              `json:"usage"`
	Docs           []string            `json:"docs"`
	OutputContract map[string]string   `json:"output_contract"`
	GlobalFlags    []flagDoc           `json:"global_flags"`
	Environment    []string            `json:"environment"`
	Commands       []catalogCommandDoc `json:"commands"`
}

func runCommands(rc *runCtx, args []string) int {
	if len(args) > 0 {
		return usage(rc, "skycli commands takes no arguments")
	}
	catalog := buildCommandCatalog()
	if rc.g.asJSON {
		_ = rc.out.JSON(catalog)
		return exitOK
	}
	rows := make([][]string, 0, len(catalog.Commands))
	for _, cmd := range catalog.Commands {
		rows = append(rows, []string{cmd.Name, boolYN(cmd.ReadOnly), boolYN(cmd.Mutates), cmd.Summary})
	}
	rc.out.Table([]string{"COMMAND", "READ-ONLY", "MUTATES", "SUMMARY"}, rows)
	return exitOK
}

func buildCommandCatalog() commandCatalogDoc {
	return commandCatalogDoc{
		Name:    "skycli",
		Summary: "Unofficial CLI for the Skylight Calendar private API.",
		Usage:   "skycli [global flags] <command> [args]",
		Docs: []string{
			"README.md",
			"AGENTS.md",
			"docs/README.md",
			"docs/commands/README.md",
		},
		OutputContract: map[string]string{
			"stdout":       "Primary command data only. In --json mode stdout is one JSON document unless a command intentionally streams events.",
			"stderr":       "Diagnostics, prompts, warnings, trace logs, and human guidance.",
			"doctor_flag":  "Use skycli --doctor for readonly token/API connectivity checks.",
			"json_flag":    "Global flag. It may appear before or after commands, before a literal --.",
			"plain_flag":   "Global flag for stable TSV/plain output where available. Mutually exclusive with --json.",
			"exit_codes":   "0 success, 1 runtime/API failure, 2 invalid usage.",
			"secrets":      "Access tokens, refresh tokens, passwords, Keychain values, and 1Password values are never printed by default.",
			"mutations":    "Use --readonly to block mutating commands and --dry-run to refuse non-GET HTTP calls.",
			"raw_endpoint": "raw and raw API-backed commands may return Skylight's full JSON:API envelope by design.",
		},
		GlobalFlags: []flagDoc{
			{Name: "--doctor", Summary: "Run readonly token/API connectivity checks and exit."},
			{Name: "--json", Summary: "Emit JSON to stdout."},
			{Name: "--plain", Summary: "Emit stable TSV/plain output where available."},
			{Name: "--readonly", Summary: "Block mutating commands before they run."},
			{Name: "--dry-run", Summary: "Refuse all non-GET HTTP calls."},
			{Name: "--frame", Value: "id", Summary: "Override the default frame ID."},
			{Name: "--config", Value: "path", Summary: "Use a specific config file."},
			{Name: "--timeout", Value: "duration", Summary: "Set HTTP timeout."},
			{Name: "--token", Value: "token", Summary: "Override the access token for this run."},
			{Name: "--trace-http", Summary: "Log token-safe HTTP request traces to stderr."},
			{Name: "--allow-commands", Value: "prefixes", Summary: "Allow only comma-separated command prefixes."},
			{Name: "--deny-commands", Value: "prefixes", Summary: "Deny comma-separated command prefixes."},
		},
		Environment: []string{
			"SKYLIGHT_ACCESS_TOKEN",
			"SKYLIGHT_AUTH_SCHEME",
			"SKYLIGHT_FRAME_ID",
			"SKYCLI_READONLY",
			"SKYCLI_ALLOW_COMMANDS",
			"SKYCLI_DENY_COMMANDS",
			"SKYCLI_SECRET_BACKEND",
			"SKYCLI_FILE_SECRET_KEY",
		},
		Commands: []catalogCommandDoc{
			{
				Name:     "commands",
				Summary:  "Print the command catalog for agents.",
				Docs:     "docs/commands/commands.md",
				ReadOnly: true,
				Examples: []string{
					"skycli commands --json",
					"skycli --json",
				},
			},
			{
				Name:    "auth",
				Summary: "Manage credentials and inspect auth state.",
				Docs:    "docs/commands/auth.md",
				Subcommands: []subcommandDoc{
					{Name: "login", Summary: "OAuth login with email/password.", Mutates: true},
					{Name: "import-mac", Summary: "Import tokens from the macOS Skylight app.", Mutates: true},
					{Name: "refresh", Summary: "Refresh a stored access token.", Mutates: true},
					{Name: "set-token", Summary: "Store a pasted access token.", Mutates: true},
					{Name: "status", Summary: "Inspect config/auth state without printing secrets."},
				},
				Mutates: true,
				Examples: []string{
					"skycli auth status --json",
					"printf '%s\\n' \"$SKYLIGHT_PASSWORD\" | skycli auth login --email you@example.com --password-stdin --json",
				},
			},
			{
				Name: "frames",
				Aliases: []string{
					"frame",
				},
				Summary:  "List, inspect, and set the default frame.",
				Docs:     "docs/commands/frames.md",
				ReadOnly: false,
				Mutates:  true,
				Subcommands: []subcommandDoc{
					{Name: "list", Summary: "List available frames."},
					{Name: "show", Summary: "Show one frame."},
					{Name: "devices", Summary: "Return raw frame device data."},
					{Name: "device", Summary: "Show one frame device."},
					{Name: "household-config", Summary: "Show household display configuration."},
					{Name: "alarms", Summary: "List alarms for a frame device."},
					{Name: "avatars", Summary: "Return raw frame avatar data."},
					{Name: "colors", Summary: "Return raw frame color data."},
					{Name: "set-default", Summary: "Save the default frame ID.", Mutates: true},
				},
				Examples: []string{
					"skycli frames --json",
					"skycli frames set-default 5312425 --json",
				},
			},
			{
				Name:     "categories",
				Aliases:  []string{"category"},
				Summary:  "List household categories for chores, rewards, and people.",
				Docs:     "docs/commands/categories.md",
				ReadOnly: true,
				Examples: []string{"skycli categories --json"},
			},
			{
				Name:    "chores",
				Aliases: []string{"chore"},
				Summary: "List and manage chores.",
				Docs:    "docs/commands/chores.md",
				Mutates: true,
				Subcommands: []subcommandDoc{
					{Name: "list", Summary: "List chores."},
					{Name: "week", Summary: "Show a weekly chore/task view."},
					{Name: "streak", Summary: "Compute completion streaks."},
					{Name: "create", Summary: "Create an assigned chore.", Mutates: true},
					{Name: "create-up-for-grabs", Summary: "Create a claimable chore.", Mutates: true},
					{Name: "update", Summary: "Update a chore.", Mutates: true},
					{Name: "claim", Summary: "Claim a chore.", Mutates: true},
					{Name: "complete", Summary: "Complete a chore.", Mutates: true},
					{Name: "skip", Summary: "Skip a chore.", Mutates: true},
					{Name: "delete", Summary: "Delete a chore.", Mutates: true},
					{Name: "bulk", Summary: "Bulk-create chores from JSON.", Mutates: true},
				},
				Examples: []string{
					"skycli chores list --json --start-date 2026-06-10 --end-date 2026-06-17",
					"skycli --readonly chores list --json",
				},
			},
			{
				Name:    "rewards",
				Aliases: []string{"reward"},
				Summary: "List and manage rewards and point balances.",
				Docs:    "docs/commands/rewards.md",
				Mutates: true,
				Subcommands: []subcommandDoc{
					{Name: "list", Summary: "List rewards."},
					{Name: "create", Summary: "Create rewards.", Mutates: true},
					{Name: "update", Summary: "Update a reward.", Mutates: true},
					{Name: "delete", Summary: "Delete a reward.", Mutates: true},
					{Name: "redeem", Summary: "Redeem a reward.", Mutates: true},
					{Name: "unredeem", Summary: "Unredeem a reward.", Mutates: true},
					{Name: "bulk", Summary: "Bulk-create rewards.", Mutates: true},
					{Name: "points", Summary: "List point balances."},
				},
				Examples: []string{
					"skycli rewards list --json",
					"skycli rewards points --json",
				},
			},
			{
				Name:    "calendar",
				Summary: "List and manage calendar events and sources.",
				Docs:    "docs/commands/calendar.md",
				Mutates: true,
				Subcommands: []subcommandDoc{
					{Name: "list", Summary: "List events in a date range."},
					{Name: "week", Summary: "Show events for a week."},
					{Name: "create", Summary: "Create an event.", Mutates: true},
					{Name: "create-countdown", Summary: "Create a countdown event.", Mutates: true},
					{Name: "update", Summary: "Update an event.", Mutates: true},
					{Name: "delete", Summary: "Delete an event.", Mutates: true},
					{Name: "sources", Summary: "List connected calendar sources."},
					{Name: "search", Summary: "Search calendar events."},
					{Name: "countdowns", Summary: "List countdown events in a date range."},
					{Name: "recent-invites", Summary: "List recently invited event email addresses."},
				},
				Examples: []string{
					"skycli calendar list --json --start-date 2026-06-10 --end-date 2026-06-17",
					"skycli calendar sources --json",
				},
			},
			{
				Name:    "lists",
				Aliases: []string{"list"},
				Summary: "List and manage Skylight lists and task-box items.",
				Docs:    "docs/commands/lists.md",
				Mutates: true,
				Subcommands: []subcommandDoc{
					{Name: "list", Summary: "List lists."},
					{Name: "show", Summary: "Show a list with items."},
					{Name: "create", Summary: "Create a list.", Mutates: true},
					{Name: "update", Summary: "Update a list.", Mutates: true},
					{Name: "delete", Summary: "Delete a list.", Mutates: true},
					{Name: "add-item", Summary: "Add an item.", Mutates: true},
					{Name: "update-item", Summary: "Update an item.", Mutates: true},
					{Name: "delete-item", Summary: "Delete an item.", Mutates: true},
					{Name: "clear-completed", Summary: "Delete completed items.", Mutates: true},
					{Name: "organize", Summary: "Ask Skylight to organize a list.", Mutates: true},
					{Name: "order", Summary: "Start a grocery order.", Mutates: true},
					{Name: "task-box-items", Summary: "List task-box items."},
					{Name: "task-box-item", Summary: "Create a task-box item.", Mutates: true},
				},
				Examples: []string{
					"skycli lists list --json",
					"skycli lists task-box-items --json",
				},
			},
			{
				Name:    "grocery",
				Summary: "Convenience commands for grocery lists.",
				Docs:    "docs/commands/grocery.md",
				Mutates: true,
				Subcommands: []subcommandDoc{
					{Name: "list", Summary: "List grocery lists."},
					{Name: "show", Summary: "Show a grocery list."},
					{Name: "create", Summary: "Create a grocery list.", Mutates: true},
					{Name: "add", Summary: "Add grocery items.", Mutates: true},
					{Name: "clear", Summary: "Clear completed grocery items.", Mutates: true},
					{Name: "organize", Summary: "Organize grocery items.", Mutates: true},
					{Name: "order", Summary: "Start a grocery order.", Mutates: true},
					{Name: "add-recipe", Summary: "Add recipe ingredients to grocery.", Mutates: true},
				},
				Examples: []string{"skycli grocery list --json"},
			},
			{
				Name:    "meals",
				Aliases: []string{"meal"},
				Summary: "Read and manage meal categories, recipes, sittings, and grocery sync.",
				Docs:    "docs/commands/meals.md",
				Mutates: true,
				Subcommands: []subcommandDoc{
					{Name: "categories", Summary: "List meal categories."},
					{Name: "recipes", Summary: "List recipes."},
					{Name: "recipe-info", Summary: "Show a recipe."},
					{Name: "create-recipe", Summary: "Create a recipe.", Mutates: true},
					{Name: "update-recipe", Summary: "Update a recipe.", Mutates: true},
					{Name: "delete-recipe", Summary: "Delete a recipe.", Mutates: true},
					{Name: "sittings", Summary: "List meal sittings."},
					{Name: "create-sitting", Summary: "Create a meal sitting.", Mutates: true},
					{Name: "delete-sitting", Summary: "Delete a meal sitting.", Mutates: true},
					{Name: "add-to-grocery", Summary: "Add recipe ingredients to grocery.", Mutates: true},
				},
				Examples: []string{"skycli meals recipes --json"},
			},
			{
				Name:    "photos",
				Aliases: []string{"photo"},
				Summary: "List, upload, download, and delete photos.",
				Docs:    "docs/commands/photos.md",
				Mutates: true,
				Subcommands: []subcommandDoc{
					{Name: "list", Summary: "List photos."},
					{Name: "show", Summary: "Show photo/message details."},
					{Name: "likes", Summary: "List likes for a photo/message."},
					{Name: "comments", Summary: "List comments for a photo/message."},
					{Name: "upload", Summary: "Upload a photo.", Mutates: true},
					{Name: "download", Summary: "Download a photo file."},
					{Name: "delete", Summary: "Delete a photo.", Mutates: true},
				},
				Examples: []string{"skycli photos list --json"},
			},
			{
				Name:     "albums",
				Aliases:  []string{"album"},
				Summary:  "Read photo albums and their messages.",
				Docs:     "docs/commands/albums.md",
				ReadOnly: true,
				Subcommands: []subcommandDoc{
					{Name: "list", Summary: "List photo albums."},
					{Name: "messages", Summary: "List messages in an album."},
					{Name: "message-ids", Summary: "List every message ID in an album."},
				},
				Examples: []string{
					"skycli albums list --json",
					"skycli albums messages --album-id 123 --json",
				},
			},
			{
				Name:    "routines",
				Aliases: []string{"routine"},
				Summary: "List and manage routines.",
				Docs:    "docs/commands/routines.md",
				Mutates: true,
				Subcommands: []subcommandDoc{
					{Name: "list", Summary: "List routines."},
					{Name: "create", Summary: "Create a routine.", Mutates: true},
					{Name: "update", Summary: "Update a routine.", Mutates: true},
					{Name: "delete", Summary: "Delete a routine.", Mutates: true},
					{Name: "reorder", Summary: "Reorder routines.", Mutates: true},
				},
				Examples: []string{"skycli routines list --json"},
			},
			{
				Name:     "sidekick",
				Summary:  "Inspect Plus access and Sidekick auto-creation history.",
				Docs:     "docs/commands/sidekick.md",
				ReadOnly: true,
				Subcommands: []subcommandDoc{
					{Name: "status", Summary: "Show sanitized Plus access state."},
					{Name: "history", Summary: "List Sidekick auto-creation intents."},
				},
				Examples: []string{
					"skycli sidekick status --json",
					"skycli sidekick history --json",
				},
			},
			{
				Name:    "bounties",
				Aliases: []string{"bounty"},
				Summary: "Pair chores and rewards into bounty workflows.",
				Docs:    "docs/commands/bounties.md",
				Mutates: true,
				Subcommands: []subcommandDoc{
					{Name: "list", Summary: "List inferred bounty pairs."},
					{Name: "create", Summary: "Create a chore and paired reward.", Mutates: true},
					{Name: "update", Summary: "Update the chore and reward pair.", Mutates: true},
					{Name: "delete", Summary: "Delete the chore and reward pair.", Mutates: true},
				},
				Examples: []string{"skycli bounties list --json"},
			},
			{
				Name:    "rotations",
				Aliases: []string{"rotation"},
				Summary: "Create rotating chore schedules.",
				Docs:    "docs/commands/rotations.md",
				Mutates: true,
				Subcommands: []subcommandDoc{
					{Name: "create", Summary: "Create rotation chores.", Mutates: true},
				},
				Examples: []string{"skycli rotations create --chores Trash,Dishes --assignee-ids 1,2 --json"},
			},
			{
				Name:     "status",
				Summary:  "Show a quick connected-frame overview.",
				Docs:     "docs/commands/status.md",
				ReadOnly: true,
				Examples: []string{"skycli status --json"},
			},
			{
				Name:     "analytics",
				Summary:  "Compute family activity statistics over a date window.",
				Docs:     "docs/commands/analytics.md",
				ReadOnly: true,
				Examples: []string{"skycli analytics --days 30 --json"},
			},
			{
				Name:     "home",
				Summary:  "Show a weekly combined events/tasks/lists view.",
				Docs:     "docs/commands/home.md",
				ReadOnly: true,
				Examples: []string{"skycli home --json --date 2026-06-10"},
			},
			{
				Name:     "watch",
				Summary:  "Poll for resource changes and stream events.",
				Docs:     "docs/commands/watch.md",
				ReadOnly: true,
				Examples: []string{"skycli watch --resources chores,rewards --json"},
			},
			{
				Name:     "export",
				Summary:  "Export frame data to portable JSON.",
				Docs:     "docs/commands/export.md",
				ReadOnly: true,
				Examples: []string{"skycli export --resources all --days 90 --output-file skylight-export.json --json"},
			},
			{
				Name:    "import",
				Summary: "Import a skycli export.",
				Docs:    "docs/commands/import.md",
				Mutates: true,
				Examples: []string{
					"skycli import --file skylight-export.json --dry-run --json",
					"skycli import --file skylight-export.json --resources lists,calendar --json",
				},
			},
			{
				Name:    "config",
				Summary: "Show or update skycli configuration.",
				Docs:    "docs/commands/config.md",
				Mutates: true,
				Subcommands: []subcommandDoc{
					{Name: "show", Summary: "Show non-secret config."},
					{Name: "get", Summary: "Get one config value."},
					{Name: "set", Summary: "Set one config value.", Mutates: true},
					{Name: "unset", Summary: "Unset one config value.", Mutates: true},
					{Name: "edit", Summary: "Edit config in $EDITOR.", Mutates: true},
				},
				Examples: []string{"skycli config show --json"},
			},
			{
				Name:    "raw",
				Summary: "Send a raw HTTP request to the Skylight API.",
				Docs:    "docs/commands/raw.md",
				Mutates: true,
				Examples: []string{
					"skycli raw /api/frames/5312425 --json --readonly",
					"skycli raw --method POST /api/frames/5312425/lists --body '{\"label\":\"Errands\"}' --json",
				},
			},
			{
				Name:     "version",
				Summary:  "Print build version metadata.",
				Docs:     "docs/commands/version.md",
				ReadOnly: true,
				Examples: []string{"skycli version --json"},
			},
		},
	}
}
