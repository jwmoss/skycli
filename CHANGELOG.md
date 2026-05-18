# Changelog

All notable changes to `skycli` will be documented in this file.

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
