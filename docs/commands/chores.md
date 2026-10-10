# chores

List and manage tasks. `chore` is an alias. Use `routines` for routine-specific defaults and filters.

## Subcommands

| Subcommand | Mutates | Purpose |
|------------|---------|---------|
| `list` | no | List tasks for a date or date range. |
| `search` | no | Search current and ended tasks by text. |
| `week` | no | Show a weekly task view. |
| `streak` | no | Compute local completion streaks. |
| `create` | yes | Create a task for one or more profiles. |
| `create-up-for-grabs` | yes | Create a claimable task. |
| `update` | yes | Change a task. |
| `claim` | yes | Assign an up-for-grabs task. |
| `complete` | yes | Complete an occurrence. |
| `skip` | yes | Skip an occurrence. |
| `undo`, `unskip` | yes | Return an occurrence to pending. |
| `move` | yes | Place a task before or after another series. |
| `delete` | yes | Delete an occurrence or series. |
| `bulk` | yes | Create tasks from JSON. |

## Examples

```bash
skycli --readonly chores list --date 2026-10-10 --json
skycli chores list --start-date 2026-10-10 --end-date 2026-10-17 --json
skycli chores search --query "Laundry" --ended-lookback-days 30 --json
skycli chores create --category 20431525 --summary "Vitamins" --recurrence daily --start-time 08:00 --json
skycli chores create --categories 20431525,20435739 --summary "Tidy room" --recurrence none --json
skycli chores create --category 20431525 --summary "Change sheets" --renewal-interval 2 --renewal-unit week --json
skycli chores update --id 81739438-2026-10-10 --description "Use clean towels" --apply-to future --json
skycli chores create-up-for-grabs --summary "Wash windows" --points 10 --json
skycli chores complete --id 81739438-2026-10-10 --instance-time 08:00 --category 20431525 --json
skycli chores unskip --id 81739438-2026-10-10 --json
skycli chores move --id 81739438 --after 81739439 --json
skycli chores bulk --file chores.json --sleep 5s --json
```

## Schedules and assignments

`create` and `update` accept `--description`, `--emoji`, `--recurrence`, `--start-time`, and `--recurring-until`.
Use `--recurrence none` for a task without recurrence.
Use `--categories` with comma-separated profile IDs for multiple assignments.
Do not combine `--category` and `--categories`.

Use `--renewal-interval` and `--renewal-unit` together for recurrence after completion.
Units are `day`, `week`, `month`, and `year`. Do not combine renewal flags with `--recurrence`.
An update that selects calendar recurrence clears completion renewal. An update that selects renewal clears calendar recurrence.
Empty `--description`, `--emoji`, `--start-time`, and `--recurring-until` values clear those fields on update.

Update and delete preserve composite occurrence IDs.
`--apply-to` accepts `one`, `future`, or `all`.
The legacy values `this_only` and `this_and_following` map to `one` and `future`.
`--apply-to-profiles` accepts `one` or `all`. Delete defaults to `all`.

## Completion and order

`complete`, `skip`, `undo`, and `unskip` accept a series or occurrence ID.
Use `--date` to supply or override the occurrence date.
Use `--instance-time` and `--category` when an occurrence needs those fields.
`--completed-on` accepts an RFC3339 timestamp for completion or skip.
Undo and unskip send `status: pending` without a completion timestamp.

`move` requires exactly one of `--before` or `--after`.
Use a series ID for the neighboring task. A composite moved-task ID resolves to its series.

## Safety and streaks

Use `--readonly` for reads and `--dry-run` to inspect writes.
Live task writes require an explicit user request.

`streak` measures days when an assignee completes all recorded chores.
It is separate from Skylight's Habit Tracker score.
Skipped chores do not count toward totals and do not break this local streak.
