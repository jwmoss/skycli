# sidekick

Inspect sanitized Plus access state and Sidekick auto-creation history.

These commands are read-only. Sidekick creation workflows are not exposed because they upload source material or create account resources.

## Status

```bash
skycli sidekick
skycli sidekick status
skycli sidekick status --json
```

The status output reports effective Calendar Plus access, active subscription count, assistant trial eligibility, and bundle entitlement availability. It does not print subscription identifiers or billing details.

## History

```bash
skycli sidekick history
skycli sidekick history --json
skycli --frame <frame-id> sidekick history --json
```

Human output reports the number of Auto-creation Intents. JSON output preserves Skylight's response document so agents can inspect intent state and results.

## Safety

Both subcommands work with `--readonly`:

```bash
skycli --readonly sidekick status --json
skycli --readonly sidekick history --json
```
