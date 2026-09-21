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

Read-only against account data. Auth refresh can update stored credentials.

Each poll obtains current credentials. Poll and persistence failures return a nonzero exit code.
`--once` reports `seeded: true` only after every requested resource succeeds.
Unknown resource names cause a usage error.

`--persist` stores seen IDs beside the selected config file.
The filename includes a hash of the API origin, account ID, and frame ID.
Old global `watch-state.json` files are not reused.
An initial successful poll seeds existing events; watch reports later changes.
Use a process supervisor to restart watch after a reported failure.
Dates use the selected frame timezone. Missing frame timezone data falls back to the host timezone.
