# Changelog

All notable changes to `skycli` will be documented in this file.

## Unreleased

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
