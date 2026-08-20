# Changelog

All notable changes to `skycli` will be documented in this file.

## Unreleased

## v0.1.9 - 2026-08-19

### Added

- Add read-only chore search with ended-chore lookback and result-limit flags.
- Add event and task notification settings, reminder profile, and month review
  reads.
- Add recorded nudge listing for a required RFC3339 time range.

## v0.1.8 - 2026-08-05

### Added

- Add read-only frame device detail, household configuration, and device alarm
  commands.
- Add calendar event search, countdown listing, and recent invite history.
- Add photo detail, likes, and comments reads.
- Add read-only photo album listing, paged messages, and complete message ID
  lookup.

## v0.1.7 - 2026-07-13

### Added

- Add read-only `sidekick status` and `sidekick history` commands for sanitized
  Plus access state and Sidekick Auto-creation Intents discovered in Skylight
  app 2.9.0.
- Add `CONTEXT.md` with the project vocabulary for Frames, Profiles, Labels,
  Categories, Sidekick, Auto-creation Intents, and Plus Access.

### Changed

- Deepen `internal/skylight` resource access so private endpoint paths, query
  rules, response decoding, and resource representations no longer leak into
  command modules. The generic transport remains public only for `raw`.
- Update pinned GitHub Actions and Go module dependencies.

## v0.1.6 - 2026-06-11

### Fixed

- Block `--readonly raw -method=POST` (single-dash `=` form), which previously
  bypassed the readonly command check and sent the mutation.
- Enforce `--readonly` inside the HTTP client as well: non-GET API calls are
  refused even if a command-level check is missed.
- Serialize concurrent token refreshes with a file lock and re-read stored
  credentials before refreshing, so parallel invocations no longer clobber
  each other's rotated refresh token.
- Write the encrypted secrets file atomically (write-then-rename) so an
  interrupted write cannot corrupt stored tokens.

## v0.1.5 - 2026-06-11

### Added

- Add `skycli commands` and `skycli --json` command-catalog output for agents,
  including docs paths, mutation markers, examples, global flags, environment
  variables, and output contracts.
- Add a Crabbox-style docs tree under `docs/commands/` with one page for each
  public command surface, and document agent usage in `AGENTS.md`.
- Add `make live-readonly-smoke`, a real-account GET/read-only integration
  check that validates JSON stdout for the command surface against the live API.

### Changed

- Make `skycli --doctor` the only health-check interface.
- Allow global flags such as `--json`, `--readonly`, and `--frame` after the
  command so agent-generated invocations like `skycli chores list --json`
  behave as expected.
- Make `watch --once --json` emit a bounded JSON status document instead of
  streaming human output.

### Fixed

- Fix commands and flags that accepted `--json` after the command but still
  wrote human text, including `--doctor`, `config get`, `config set`, and
  `config unset`.
- Return structured JSON for JSON-mode usage errors and unknown commands.

## v0.1.4 - 2026-06-10

### Fixed

- Mask `config get access_token` and `config get refresh_token` output by
  default; use `--show-secrets` only when raw token output is explicitly needed.
- Make `doctor` return non-zero and `ok: false` whenever any reported check
  fails, including a missing default frame.
- Send calendar list/week/export date ranges with Skylight's current
  `date_min`/`date_max` query keys so typed calendar reads work against the
  live API.
- Preserve calendar event descriptions and categories in export/import.
- Return structured partial-failure JSON when a bounty update/delete applies
  the chore change but the paired reward mutation fails.
- Add `lists task-box-items` for reading task-box items and fix
  `lists task-box-item` to use Skylight's live `/task_box/items` endpoint
  instead of the obsolete `/task_box_items` path.

### Changed

- Pin GitHub Actions to commit SHAs and bump to Node 24 runtimes
  (`checkout` v6, `setup-go` v6, `goreleaser-action` v7), set
  `persist-credentials: false`, and disable the Go cache in the release job to
  avoid cache poisoning. Add a Dependabot config to track Actions and Go
  modules with grouped, 7-day-cooldown updates.

## v0.1.3 - 2026-06-07

### Fixed

- Make `skycli version` report the Go module version for `go install
  github.com/jwmoss/skycli@vX.Y.Z` builds instead of falling back to an old
  source default.
- Stop sending the Skylight access token to off-origin URLs passed to `skycli
  raw`; the `Authorization` and `skylight-api-version` headers are now only
  attached when the request host matches the configured base URL.
- Validate bounty `--reward-id` and `--category-ids` before any chore is
  created, updated, or deleted so a bad ID can no longer leave a half-applied
  bounty.
- Fail `skycli export` when a list's items cannot be fetched instead of writing
  a backup with lists missing their contents.
- Reject unknown `--resources` tokens in `export`/`import` with a usage error
  instead of silently producing a partial or empty result.
- Parse `EDITOR`/`VISUAL` values with arguments (e.g. `code -w`) in `config
  edit` instead of treating the whole value as a single executable name.

### Changed

- Pin GoReleaser for release builds, add Make targets for release validation,
  and document the expected release check flow.

## v0.1.2 - 2026-05-18

### Fixed

- Republish the v0.1.1 auth prompt and frame discovery fixes through the
  tag-triggered release workflow so the published release state is clean.

## v0.1.1 - 2026-05-18

### Fixed

- Hide interactive password and token entry in the terminal so pasted secrets do
  not echo back in plaintext.
- Make `skycli frames` list available frames by default, so users can discover
  frame IDs before setting a default.
- Improve missing-frame errors with the exact discovery and default-frame
  commands to run next.
- Add `chores list --start-date` and `--end-date` aliases for the existing
  `--after` and `--before` filters.

## v0.1.0 - 2026-05-18

Initial release of `skycli`, an unofficial CLI for the Skylight Calendar
private API. This release establishes the core command surface for managing a
Skylight account from a terminal.

### Added

- Authentication commands for signing in with a Skylight email/password,
  importing tokens from the macOS Skylight app, pasting a captured token,
  checking auth status, and refreshing stored tokens.
- Configurable token storage with macOS Keychain support, encrypted local-file
  support, and a config-file fallback for tests or throwaway environments.
- Frame discovery and default-frame configuration so commands can target a
  specific Skylight frame without repeating the frame ID.
- Category lookup for finding the people, household, and shared categories used
  by chores, rewards, and other account resources.
- Chore commands for listing scheduled chores, creating assigned chores,
  creating true Up-for-Grabs chores, bulk-loading chores from JSON, updating or
  deleting chore series, claiming chores, completing generated chore instances,
  checking streaks, and viewing a week of task-box activity.
- Reward commands for listing rewards, checking point balances, creating
  rewards across one or more child categories, updating reward names or point
  values, and redeeming rewards.
- Calendar commands for listing events by date range, showing a weekly view,
  creating events, creating countdowns, and inspecting connected calendar
  sources.
- List and grocery commands for creating lists, adding items, clearing
  completed items, organizing grocery lists, and starting grocery orders.
- Meal commands for reading meal categories, creating recipes, creating meal
  sittings, and adding recipe ingredients to grocery lists.
- Photo commands for listing photos and uploading new photos with captions.
- Routine commands for creating multi-step routines assigned to a household
  member.
- Bounty and rotation commands for one-off point opportunities and recurring
  chore rotation schedules.
- Status, analytics, home, reports, and watch commands for inspecting account
  state and monitoring selected resources over time.
- Export and import commands for backing up account data to JSON, reviewing an
  import with `--dry-run`, and restoring selected resources.
- Raw HTTP commands for calling newly discovered Skylight private API endpoints
  before they have first-class command wrappers.
- Machine-friendly output modes with JSON and plain output, plus trace logging
  that records request flow without printing bearer tokens.
- Safety controls including global `--dry-run`, `--readonly`, command
  allowlists, and command denylists for scripting and agent use.
