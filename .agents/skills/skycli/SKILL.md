---
name: skycli
description: "skycli CLI: safe Skylight Calendar private API automation, JSON output, auth, scoped reads/writes."
---

# skycli

Use `skycli` when built-in tools cannot inspect or automate a Skylight Calendar
account, when shell automation needs stable JSON, or when you need to discover
private API behavior before adding typed command support.

## Fast Path

```bash
skycli version
skycli auth status
skycli --doctor --json
skycli --json frames list
skycli --json categories
```

Pick the frame explicitly for API work when possible:

```bash
skycli --frame <frame-id> --json chores list
skycli --frame <frame-id> --json rewards list
skycli --frame <frame-id> --readonly --json raw /api/frames/<frame-id>
```

Prefer `--json` for agent parsing. Human hints, traces, and errors should stay
on stderr; stdout is for data.

## Safety Rules

- Do not print access tokens, refresh tokens, passwords, Keychain values, or
  1Password secret values.
- Use `--readonly` or `--dry-run` first when exploring.
- Destructive or live-account mutations require the user to ask for that exact
  operation.
- Before writes, identify the frame ID, category/resource ID, endpoint, method,
  and payload.
- Use `--trace-http` only when needed; it must not reveal bearer tokens.
- Keep temporary payloads and exports out of commits unless the user explicitly
  wants fixtures.

Command guards:

```bash
skycli --readonly --json raw /api/frames/<frame-id>/task_box/items
skycli --dry-run chores create --category <category-id> --summary "Example"
skycli --allow-commands "chores list,rewards list" --json chores list
skycli --deny-commands "chores delete,rewards redeem" --json rewards list
```

## Auth

Human login:

```bash
skycli auth login --email user@example.com
skycli auth refresh
skycli auth status
```

Automation login:

```bash
printf '%s\n' "$SKYLIGHT_PASSWORD" | skycli auth login --email user@example.com --password-stdin
```

Mac app import:

```bash
skycli auth import-mac
```

Token resolution order: `--token`, `SKYLIGHT_ACCESS_TOKEN`, then stored
config/secret backend. Explicit tokens are not auto-refreshed.

## Common Reads

```bash
skycli --json frames list
skycli --json categories
skycli --json chores list
skycli --json chores week --date 2026-05-18
skycli --json rewards list
skycli --json rewards points
skycli --json calendar list --start-date 2026-05-18 --end-date 2026-05-24
skycli --json lists list
skycli --json photos list
skycli --json status
skycli --json analytics --days 30
```

For endpoint discovery, prefer GETs through `raw` before writing code:

```bash
skycli --readonly --json raw /api/frames/<frame-id>/lists
skycli --readonly --json raw /api/frames/<frame-id>/rewards
```

## Writes

Before writes, confirm the account, frame, category/resource ID, and exact
mutation. Prefer `--dry-run` when the command supports it.

```bash
skycli chores create --category <category-id> --summary "Vitamins" --recurrence daily
skycli chores create-up-for-grabs --summary "Wash windows" --points 10
skycli chores claim --id <instance-id> --category <category-id>
skycli chores complete --id <instance-id>
skycli rewards create --name "TV Ticket" --points 10 --categories <id1>,<id2> --respawn
skycli rewards redeem --id <reward-id>
skycli calendar create --title "Dentist" --start-at 2026-05-19T14:00:00-04:00 --end-at 2026-05-19T15:00:00-04:00
skycli grocery add --list-id <list-id> --title "Milk"
```

After mutation, verify with a typed list command or a GET through `raw`.

## Backup / Restore

```bash
skycli export --output-file skylight-export.json --resources all --days 90
skycli import --file skylight-export.json --dry-run
```

Do not commit real exports unless the user explicitly says they are sanitized
fixtures.

## Discovery

Use help and raw responses instead of guessing flags:

```bash
skycli --help
skycli chores --help
skycli raw --help
```

Docs:

- `README.md`
- `CHANGELOG.md`
- `AGENTS.md`

Repo paths:

- CLI entrypoint: `main.go`
- Command implementations: `internal/cli/`
- HTTP client and OAuth: `internal/skylight/`
- Config defaults: `internal/config/`
