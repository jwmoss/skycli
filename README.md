# skycli

CLI for the [Skylight Calendar](https://www.myskylight.com/) private API.

Unofficial. Not affiliated with Skylight. Use with accounts you own.

`skycli` wraps a broad private-API surface: frames, categories, chores,
rewards, calendar events, lists/grocery, meals, photos, routines, bounties,
rotations, export/import, status, analytics, and watch.

## Docs

- [Agent instructions](AGENTS.md)
- [Command index](docs/commands/README.md)
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

## Authentication

Three ways to give `skycli` a token:

### 1. Login with email/password

```bash
skycli auth login --email you@example.com
```

This performs a headless OAuth login, prompts for your password with terminal
echo disabled, and stores the returned refresh token plus device fingerprint.
The password is not stored.

For non-interactive use:

```bash
printf '%s\n' "$SKYLIGHT_PASSWORD" | skycli auth login --email you@example.com --password-stdin
```

### 2. Import from the macOS app

If the **Skylight** iOS app is installed and signed in on your Mac (it ships
in the iOS-on-Mac App Store as `Skylight.app`):

```bash
skycli auth import-mac
```

Reads the access token, refresh token, and expiry from the app's MMKV store at
`~/Library/Containers/com.skylightframe.mobile/Data/Documents/mmkv/mmkv.default`.
On macOS, tokens are stored in Keychain by default and non-secret settings are
written to `~/Library/Application Support/skycli/config.json` (mode 0600).

The access token expires every ~2 hours. `auth import-mac` also saves the app's
refresh token and device fingerprint, so future config-backed commands refresh
automatically when the access token is expired or close to expiry.

You can also refresh explicitly:

```bash
skycli auth refresh
```

### 3. Paste a token captured from a proxy

If you can't use the login flow or Mac app, capture a token from your own
account with **Proxyman**, **Charles**, or **mitmproxy**. Then:

```bash
skycli auth set-token                  # paste "Bearer <token>" or "Basic <token>"
skycli auth set-token --scheme basic   # explicit scheme if pasting a bare token
```

Interactive token entry also disables terminal echo.

### Secret storage

Secret backends:

- `keychain`: default on macOS; stores tokens in macOS Keychain.
- `file`: encrypted file under the skycli config directory; requires
  `SKYCLI_FILE_SECRET_KEY`.
- `config`: stores tokens directly in the JSON config file, mostly for tests or
  throwaway local use.

```bash
skycli config set secrets_backend file
export SKYCLI_FILE_SECRET_KEY="$(openssl rand -base64 32)"
skycli auth set-token
```

### Token resolution order

Every call resolves the token: `--token` flag → `SKYLIGHT_ACCESS_TOKEN` env var
→ secret store/config file. Only configured tokens are auto-refreshed; explicit
flag/env tokens are used as-is.

`skycli auth status` shows the current config, scheme, and expiry.

## Quick start

```bash
skycli auth login --email you@example.com
skycli frames                        # find your frame ID
skycli frames set-default 5312425
skycli --doctor                      # verify token + connectivity
skycli categories                    # show category IDs (one per kid + "Family")
skycli chores list
```

## Commands

### Chores

```bash
# List chores for today (default)
skycli chores list

# List for a date range, JSON output
skycli --json chores list --after 2026-05-15 --before 2026-05-20
skycli --json chores list --start-date 2026-05-15 --end-date 2026-05-20

# Create one assigned chore
skycli chores create \
  --category 20431525 \
  --summary "Vitamins" \
  --recurrence daily

# Create a Mon/Fri recurring chore
skycli chores create \
  --category 20431525 \
  --summary "Water plants" \
  --recurrence "weekly:MO,FR"

# Create an Up-for-Grabs (claimable) chore with points
skycli chores create-up-for-grabs \
  --summary "Wash all windows" \
  --points 10

# Rename or adjust a recurring chore series
skycli chores update --id 81779567 --summary "Mary TV Ticket" --points 1

# Claim/complete one generated instance
skycli chores claim --id 81739438-2026-05-18 --category 20435739
skycli chores complete --id 81739438-2026-05-18
skycli chores streak --days 30
skycli chores week --date 2026-05-18

# Delete a chore (recurring → all instances)
skycli chores delete --id 81832922 --apply-to all

# Bulk create from a JSON file, 5s between posts
skycli chores bulk --file chores.json --sleep 5s
```

`chores.json`:

```json
[
  {"summary": "Vitamins", "category_id": 20431525},
  {"summary": "Water plants", "category_id": 20431525, "recurrence": "weekly:MO,FR"},
  {"summary": "Wash all windows", "category_id": 20450972, "up_for_grabs": true, "reward_points": 10}
]
```

Each item needs `summary` + `category_id`. Defaults: `recurrence: "daily"`,
`start: today`, `up_for_grabs: false`, `reward_points: null`.

True Up-for-Grabs chores use Skylight's `create_multiple` endpoint and have no
category until someone claims them. Use `chores create-up-for-grabs` for those;
the bulk format is for assigned chores.

### Rewards

```bash
# List rewards and point balances
skycli rewards list
skycli rewards points

# Create one reward per child category
skycli rewards create --name "TV Ticket" --points 10 --categories 20431525,20435739 --respawn

# Rename/change a reward and redeem it
skycli rewards update --id 9957645 --name "Levi TV Ticket" --points 10 --respawn
skycli rewards redeem --id 9957645
```

### Calendar

```bash
skycli calendar list --start-date 2026-05-18 --end-date 2026-05-24
skycli calendar week --date 2026-05-18
skycli calendar create --title "Dentist" --start-at 2026-05-19T14:00:00-04:00 --end-at 2026-05-19T15:00:00-04:00
skycli calendar create-countdown --title "Beach trip" --date 2026-07-01
skycli calendar sources
```

### Lists And Grocery

```bash
skycli lists list
skycli lists task-box-items
skycli lists create --title "Errands" --kind to_do
skycli lists add-item --list-id 123 --title "Return library books"
skycli lists clear-completed --list-id 123

skycli grocery create --title "Groceries"
skycli grocery add --list-id 456 --title "Milk"
skycli grocery organize --list-id 456
skycli grocery order --list-id 456 --retailer costco
```

### Meals, Photos, Routines

```bash
skycli meals categories
skycli meals create-recipe --title "Tacos" --ingredients "tortillas,beans,cheese"
skycli meals create-sitting --summary "Taco night" --date 2026-05-20
skycli meals add-to-grocery --recipe-id 789

skycli photos list
skycli photos upload --file ./photo.jpg --caption "May"
skycli routines create --title "Bedtime" --assignee-id 20431525 --steps "Brush teeth,Read"
```

### Bounties, Rotations, Reports

```bash
skycli bounties create --title "Clean garage shelf" --points 5 --assignee-id 20431525 --reward-title "Garage bounty"
skycli rotations create --chores "Trash,Dishes" --assignee-ids 20431525,20435739 --weeks 4 --points 1

skycli status
skycli analytics --days 30
skycli home
skycli watch --resources rewards,chores --interval 30s
```

### Export And Import

```bash
skycli export --output-file skylight-export.json --resources all --days 90
skycli import --file skylight-export.json --dry-run
skycli import --file skylight-export.json --resources rewards,chores
```

### Raw HTTP

For newly discovered endpoints or fields before a typed flag exists:

```bash
skycli raw /api/frames/5312425/lists
skycli raw --method POST --body '{"summary":"x"}' /api/frames/5312425/task_box/items
echo '{"summary":"x"}' | skycli raw --method POST --body-file - /api/frames/5312425/task_box/items
```

### Global flags

| Flag             | Default | Notes |
|------------------|---------|-------|
| `--config PATH`  | `$XDG_CONFIG_HOME/skycli/config.json` | Override config path |
| `--doctor`       | off     | Run readonly token/API connectivity checks and exit |
| `--json`         | off     | Emit JSON to stdout |
| `--plain`        | off     | Emit stable TSV/plain output where available |
| `--timeout DUR`  | 30s     | HTTP timeout |
| `--trace-http`   | off     | Log every request to stderr |
| `--dry-run`      | off     | Refuse non-GET HTTP calls |
| `--readonly`     | off     | Block mutating commands |
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
tagged builds still validate the GoReleaser configuration and Homebrew formula
generation path.

## Release

The GoReleaser version is pinned by `GORELEASER_VERSION` in `Makefile` and must
match `.github/workflows/release.yml`. Before tagging, run:

```bash
make ci
make release-check
make release-snapshot
```

Tagging `vX.Y.Z` triggers GoReleaser to build release archives, publish GitHub
release assets, and update the `jwmoss/homebrew-tap` formula. The release
workflow needs a `HOMEBREW_TAP_TOKEN` secret with write access to that tap
repository.

## Status

Early, unofficial, and private-API backed, with `--body` / `--body-file`
available on several typed commands for fields that are discovered before
first-class flags are added.

## License

MIT
