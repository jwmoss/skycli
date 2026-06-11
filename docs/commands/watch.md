# watch

Polls for newly completed chores, redeemed rewards, and upcoming events.

## Examples

```bash
skycli watch --resources rewards,chores --interval 30s
skycli watch --once --json
skycli --readonly watch --once --json
```

## Output

Streaming watch mode may emit multiple event records. `watch --once --json` emits one bounded JSON status document and is used by the integration smoke test.

## Safety

Read-only.
