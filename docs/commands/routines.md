# routines

Manage routine tasks through the chores API. `routine` is an alias.
Routines have `attributes.routine: true`. Each routine task has its own summary, assignee, schedule, and completion state.

## Subcommands

| Subcommand | Mutates | Purpose |
|------------|---------|---------|
| `list` | no | List routine tasks for a date or range. |
| `create` | yes | Create routine tasks for one or more profiles. |
| `update` | yes | Change a routine task. |
| `complete` | yes | Complete one occurrence. |
| `skip` | yes | Skip one occurrence. |
| `undo`, `unskip` | yes | Return an occurrence to pending. |
| `delete` | yes | Delete an occurrence or series. |
| `move` | yes | Place a task before or after another series. |
| `reorder` | yes | Order several routine series through sequential move requests. |

## Examples

```bash
skycli --readonly routines list --date 2026-10-10 --json
skycli routines create --summary "Brush teeth" --category 20431525 --time-of-day morning --track-habit --json
skycli routines create --title "Read" --categories 20431525,20435739 --time-of-day evening --recurrence weekly:MO,FR --json
skycli routines update --routine-id 81739438-2026-10-10 --description "Read for 20 minutes" --apply-to future --json
skycli routines complete --id 81739438-2026-10-10 --instance-time 20:00 --category 20431525 --json
skycli routines undo --id 81739438-2026-10-10 --json
skycli routines move --id 81739438 --before 81739439 --json
skycli routines reorder --routine-ids 81739438,81739439,81739440 --json
```

`list` accepts `--date`, `--after`, `--before`, `--assignee-id`, and `--status`.
It defaults to today. JSON preserves returned task attributes, including available habit data.

`create` and `update` accept `--summary` (`--title`), `--category` (`--assignee-id`), `--categories`, `--description`, and `--emoji`.
They also accept `--start`, `--recurrence`, `--time-of-day`, `--points`, and `--track-habit`.
Use `--track-habit=false` to disable habit tracking.
Use `--body` or `--body-file` for additional verified task fields. Explicit flags override matching body fields.

New routines default to daily recurrence and the morning segment.
The app represents morning, afternoon, and evening with RRULE `BYHOUR` values 6, 14, and 20.
For a schedule update, supply both `--recurrence` and `--time-of-day`, or a raw RRULE with `BYHOUR`.
A summary-only update preserves the existing schedule.

The old `--steps` option is unsupported. Create one routine task for each step.
The separate `/routines` endpoint does not represent the current app's routine model.

Use an occurrence ID for occurrence changes. Update and delete preserve composite IDs.
Use series IDs for `move` neighbors and `reorder`.
`--apply-to` accepts `one`, `future`, or `all`. `--apply-to-profiles` accepts `one` or `all`.
Delete defaults to `all`.
Reorder stops at the first API error; earlier moves remain applied.

Use `--dry-run` to inspect writes before execution. Live task writes require an explicit user request.
