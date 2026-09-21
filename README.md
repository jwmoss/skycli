# skycli

CLI for the [Skylight Calendar](https://www.myskylight.com/) private API.

Unofficial. Not affiliated with Skylight. Use with accounts you own.

`skycli` wraps a broad private-API surface: frames, categories, chores,
rewards, calendar events, lists/grocery, meals, photos/albums, routines, bounties,
rotations, notifications, nudges, month reviews, Sidekick access/history,
export/import, status, analytics, and watch.

## Docs

- [Agent instructions](AGENTS.md)
- [Command index](docs/commands/README.md)
- [API capabilities and gaps](docs/api-capabilities.md)
- Machine-readable command catalog: `skycli commands --json`

## Install

### Homebrew

```bash
brew tap jwmoss/tap
brew install skycli
```

Or:

```bash
brew install jwmoss/tap/skycli
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

The current published release uses a Homebrew formula. The next release uses a cask.
After that release is published, remove the formula and install the cask:

```bash
brew uninstall --formula skycli
brew install --cask jwmoss/tap/skycli
```

The cask is not published by this local change.

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
- [Chores](docs/commands/chores.md)
- [Rewards](docs/commands/rewards.md)
- [Calendar](docs/commands/calendar.md)
- [Lists](docs/commands/lists.md) and [grocery](docs/commands/grocery.md)
- [Meals](docs/commands/meals.md), [photos](docs/commands/photos.md),
  [albums](docs/commands/albums.md), and [routines](docs/commands/routines.md)
- [Sidekick](docs/commands/sidekick.md) access and auto-creation history
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
| `--dry-run`      | off     | Refuse non-GET HTTP calls |
| `--readonly`     | off     | Block mutating commands and refuse non-GET HTTP calls |
| `--allow-commands LIST` | — | Comma-separated command allowlist |
| `--deny-commands LIST`  | — | Comma-separated command denylist |
| `--token TOK`    | —       | Token override (also `SKYLIGHT_ACCESS_TOKEN`) |
| `--frame ID`     | —       | Frame override (also `SKYLIGHT_FRAME_ID`) |

## Output for agents

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

Run real-account read-only integration checks with:

```bash
make live-readonly-smoke
```

## Development

```bash
make fmt
make test
make vet
make ci
make build
make live-readonly-smoke
make release-check
make release-snapshot
```

CI intentionally stays small: cross-platform `go test`, `go vet`, and
`go build`. Release checks are separate so local development stays fast while
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
Optional live smoke checks skip only explicit HTTP 403/404 responses.
Authentication, rate-limit, server, transport, and JSON failures fail the smoke check.
