# chores

Lists and manages chores. `chore` is an alias.

## Subcommands

| Subcommand | Mutates | Purpose |
|------------|---------|---------|
| `list` | no | List chores for a date or date range. |
| `week` | no | Show a weekly chore/task view. |
| `streak` | no | Compute completion streaks. |
| `create` | yes | Create an assigned chore. |
| `create-up-for-grabs` | yes | Create a claimable chore. |
| `update` | yes | Update a chore. |
| `claim` | yes | Claim an up-for-grabs chore. |
| `complete` | yes | Complete a generated chore instance. |
| `skip` | yes | Skip a generated chore instance. |
| `delete` | yes | Delete a chore. |
| `bulk` | yes | Bulk-create chores from JSON. |

## Examples

```bash
skycli chores list --date 2026-06-10 --json
skycli chores list --start-date 2026-06-10 --end-date 2026-06-17 --json
skycli chores week --date 2026-06-10 --json
skycli chores streak --days 30 --json
skycli chores create --category 20431525 --summary "Vitamins" --recurrence daily --json
skycli chores create-up-for-grabs --summary "Wash windows" --points 10 --json
skycli chores complete --id 81739438-2026-06-10 --json
skycli chores bulk --file chores.json --sleep 5s --json
```

## Safety

Use `--readonly` for list/week/streak automation. Mutating subcommands call the live private API.
