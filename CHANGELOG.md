# Changelog

All notable changes to `skycli` will be documented in this file.

## v0.1.0 - 2026-05-18

Initial release of `skycli`, an unofficial CLI for the Skylight Calendar
private API.

### Added

- Headless email/password OAuth login, token refresh, macOS app token import,
  token paste support, and configurable secret storage.
- Frame discovery and default-frame configuration.
- Typed command groups for chores, rewards, calendar events, lists, grocery,
  meals, photos, routines, bounties, rotations, reports, status, analytics,
  export/import, watch, and raw HTTP calls.
- Agent-friendly output modes with JSON, plain TSV-style output, trace logging,
  dry-run/read-only safety controls, and command allow/deny lists.
- GoReleaser release automation for GitHub release assets across macOS, Linux,
  and Windows on amd64/arm64.
- Homebrew formula publishing through `jwmoss/homebrew-tap`.
