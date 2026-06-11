# Agent guide

Use this guide when another tool, script, or coding agent drives `skycli`.

## Discovery

```bash
skycli commands --json
skycli --doctor --json
skycli --readonly frames list --json
```

`skycli commands --json` returns the supported command catalog, docs paths, output contract, global flags, environment variables, examples, and mutation markers.

## Output contract

- `--json` writes one JSON document to stdout for bounded commands.
- Human diagnostics, prompts, warnings, and `--trace-http` lines go to stderr.
- Exit code `0` means success, `1` means runtime or API failure, and `2` means invalid usage.
- Usage errors in JSON mode return a JSON object with `kind: "usage"`.
- Secrets are masked by default. Do not pass `--show-secrets` unless the user explicitly needs raw token values.

## Safety

Start with read-only mode:

```bash
skycli --readonly --doctor --json
skycli --readonly chores list --json
skycli --readonly raw /api/frames/$SKYLIGHT_FRAME_ID --json
```

Use `--readonly` to block mutating commands before they run. Use `--dry-run` when testing a workflow that includes command parsing for non-GET calls. Use command allowlists for automation:

```bash
SKYCLI_ALLOW_COMMANDS="frames,calendar list,chores list,rewards list" skycli --readonly calendar list --json
```

## Live readonly integration

Run the real-account smoke check with:

```bash
make live-readonly-smoke
```

The smoke test builds `skycli`, discovers a frame, and runs GET/read-only commands only. It validates that each command writes exactly one JSON document to stdout. Feature-specific private endpoints, such as routines, may be unavailable for a given account; the smoke test skips those endpoints when the live API returns an unavailable response.

## Global flag placement

Global flags may appear before or after commands, before a literal `--`:

```bash
skycli chores list --json
skycli --json chores list
skycli chores list --readonly --json
```

Prefer placing `--dry-run` before the command when a subcommand also has its own `--dry-run` flag.
