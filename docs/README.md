# skycli docs

`skycli` is an unofficial CLI for the Skylight Calendar private API. The API is private and may change, so prefer JSON output, readonly checks, and small probes before broad automation.

## Start here

- [Agent instructions](../AGENTS.md) - safe machine-readable usage patterns.
- [Command index](commands/README.md) - one page for each public command.
- [Health check flag](commands/doctor.md) - `skycli --doctor`.

## Machine-readable discovery

```bash
skycli commands --json
skycli --json
skycli --doctor --json
```

In `--json` mode, stdout is command data only and should be a single JSON document unless a command explicitly streams events. Diagnostics and trace logs go to stderr.
