# routines

Lists and manages routines when the private routines endpoint is available. `routine` is an alias.

## Subcommands

| Subcommand | Mutates | Purpose |
|------------|---------|---------|
| `list` | no | List routines. |
| `create` | yes | Create a routine. |
| `update` | yes | Update a routine. |
| `delete` | yes | Delete a routine. |
| `reorder` | yes | Reorder routines. |

## Examples

```bash
skycli routines list --json
skycli routines create --title "Bedtime" --assignee-id 20431525 --steps "Brush teeth,Read" --json
skycli routines reorder --routine-ids routine-1,routine-2 --json
```

## Private API note

The routines endpoint is account and feature dependent. The live read-only smoke test skips `routines list` when the private API returns unavailable for the current account.
