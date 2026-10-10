# skycli

CLI for the [Skylight Calendar](https://www.myskylight.com/) private API.

Unofficial. Not affiliated with Skylight. Use with accounts you own.

`skycli` wraps a broad private-API surface: frames, categories, chores,
rewards, calendar events, lists/grocery, meals, photos/albums, routines, bounties,
rotations, device settings, notifications, nudges, month reviews, Sidekick imports/drafts,
export/import, status, analytics, and watch.

## Docs

- [Agent instructions](AGENTS.md)
- [Command index](docs/commands/README.md)
- [API capabilities and gaps](docs/api-capabilities.md)
- Machine-readable command catalog: `skycli commands --json`

## Compatibility

The October 2026 audit adds profile management, modern tasks and routines,
calendar sources, Sidekick drafts, album/photo edits, meal edits, and device controls.
The default API version is `2026-08-05`. Explicit config overrides remain unchanged.

Contracts come from Skylight 2.26.0 and live read-only checks.
The App Store lists 2.27.0; its binary is not inspected.
Most writes still need live account verification. The test suite uses real requests only.
See the [coverage ledger](docs/api-capabilities.md) for remaining gaps and
[contract evidence](docs/api-contracts-2026-10.md) for methods and payloads.

## Install

### Homebrew

```bash
brew tap jwmoss/tap
brew install --cask skycli
```

Or:

```bash
brew install --cask jwmoss/tap/skycli
```

### Go

```bash
go install github.com/jwmoss/skycli@latest
```

### Source

```bash
git clone https://github.com/jwmoss/skycli ~/github/skycli
cd ~/github/skycli
make build
./skycli version
```

Version 0.2.0 uses a Homebrew cask. To upgrade from the previous formula,
remove the formula and install the cask:

```bash
brew uninstall --formula skycli
brew install --cask jwmoss/tap/skycli
```

The cask supplies macOS and Linux binaries. Windows users can use the release ZIP.

## Quick start

```bash
skycli auth login --email you@example.com
skycli frames list                   # find your frame ID
skycli frames set-default 5312425
skycli --doctor                      # verify token + connectivity
skycli categories                    # find category/person IDs
skycli chores list --json
```

## Command docs

The full command surface is documented under
[docs/commands](docs/commands/README.md).

- [Auth](docs/commands/auth.md)
- [Frames](docs/commands/frames.md)
- [Profiles and labels](docs/commands/categories.md)
- [Chores](docs/commands/chores.md)
- [Rewards](docs/commands/rewards.md)
- [Calendar](docs/commands/calendar.md)
- [Lists](docs/commands/lists.md) and [grocery](docs/commands/grocery.md)
- [Meals](docs/commands/meals.md), [photos](docs/commands/photos.md),
  [albums](docs/commands/albums.md), and [routines](docs/commands/routines.md)
- [Sidekick](docs/commands/sidekick.md) imports, generated plans, drafts, and history
- [Reports](docs/commands/status.md), [analytics](docs/commands/analytics.md),
  [home](docs/commands/home.md), and [watch](docs/commands/watch.md)
- [Export](docs/commands/export.md), [import](docs/commands/import.md),
  [config](docs/commands/config.md), and [raw HTTP](docs/commands/raw.md)

For scripts and agents:

```bash
skycli commands --json
skycli --json
```

## Authentication

Use one of the auth commands, then verify the account with `skycli --doctor`.

```bash
skycli auth login --email you@example.com
skycli auth import-mac
skycli auth set-token
skycli auth status --json
```

See [auth docs](docs/commands/auth.md) for login, macOS import, token storage,
and non-interactive usage.

## Global flags

| Flag             | Default | Notes |
|------------------|---------|-------|
| `--config PATH`  | `$XDG_CONFIG_HOME/skycli/config.json` | Override config path |
| `--doctor`       | off     | Run readonly token/API connectivity checks and exit |
| `--json`         | off     | Emit JSON to stdout |
| `--plain`        | off     | Emit stable TSV/plain output where available |
| `--timeout DUR`  | 30s     | HTTP timeout |
| `--trace-http`   | off     | Log every request to stderr |
| `--dry-run`      | off     | Refuse non-GET HTTP calls and configuration or credential changes |
| `--readonly`     | off     | Block mutating commands and refuse non-GET HTTP calls |
| `--allow-commands LIST` | — | Comma-separated command allowlist |
| `--deny-commands LIST`  | — | Comma-separated command denylist |
| `--token TOK`    | —       | Token override (also `SKYLIGHT_ACCESS_TOKEN`) |
| `--frame ID`     | —       | Frame override (also `SKYLIGHT_FRAME_ID`) |

## Output for agents

Safety flags refuse OAuth login and token refresh. If a stored token expires,
refresh it without a safety flag or supply a valid explicit token.
Safe reads do not migrate credentials to another secret backend.
Place global `--dry-run` before the command. Import also accepts its own
`--dry-run` after the command to validate references and report counts.
Read commands can still save requested exports, downloads, or watch state.

Use `skycli commands --json` to discover the command surface and docs paths.
Every bounded command supports `--json`; table-style commands also support
`--plain` for stable TSV. Data goes to stdout, logs/errors to stderr. Exit
codes: `0` success, `1` runtime error, `2` usage error. `--trace-http` emits
one line per request to stderr without including the bearer token.

Global flags such as `--json`, `--readonly`, and `--frame` may appear before or
after the command, before a literal `--`:

```bash
skycli chores list --json
skycli --json chores list
skycli chores list --readonly --json
```

Run simple live end-to-end tests with:

```bash
make test                  # GET checks, then create/read/delete one temporary list
make live-readonly-smoke    # GET checks only
```

Tests use your configured account and frame. The temporary list stays hidden from the display.
The script deletes only the list it creates and checks that cleanup succeeds.
Set `SKYLIGHT_FRAME_ID` to select another frame.
Use `SKYCLI_BIN` to test another binary or supply `--config PATH`.


## Development

```bash
make fmt
make vet
make ci
make test
make live-readonly-smoke
make release-check
make release-snapshot
```

CI runs cross-platform builds, formatting, vet, dependency, and security checks.
The [live test script](scripts/live-e2e.py) uses Python's standard library and real Skylight requests.
Live tests stay outside CI because they use an account and create temporary data.
The suite checks basic reads and one write cycle. It does not cover every command.
Release checks are separate so local development stays fast while
tagged builds still validate the GoReleaser configuration and Homebrew cask
generation path.

## Release

The GoReleaser version is pinned in `.goreleaser-version`. Local checks and CI read the same file. Before tagging, run:

```bash
make ci
make release-check
make release-snapshot
```

Tagging `vX.Y.Z` triggers GoReleaser to build release archives, publish GitHub
release assets, and update the `jwmoss/homebrew-tap` cask. The release
workflow needs a `HOMEBREW_TAP_TOKEN` secret with write access to that tap
repository.

## Status

Early, unofficial, and private-API backed, with `--body` / `--body-file`
available on several typed commands for fields that are discovered before
first-class flags are added.

## License

MIT

Export/import handles selected resource templates, not full account backups.
Imports validate references and support the same frame only.
Use `skycli <group> --help` to list commands, then `skycli <group> <command> --help` for flags.
HTTP, authentication, transport, and JSON errors fail the live test.
