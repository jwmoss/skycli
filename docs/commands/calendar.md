# calendar

Lists and manages calendar events and source calendars.

## Subcommands

| Subcommand | Mutates | Purpose |
|------------|---------|---------|
| `list` | no | List events in a date range. |
| `week` | no | Show events for one week. |
| `sources` | no | List connected calendar sources. |
| `search` | no | Search events by text. |
| `countdowns` | no | List countdown events in a date range. |
| `recent-invites` | no | List recently invited event email addresses. |
| `create` | yes | Create an event. |
| `create-countdown` | yes | Create a countdown event. |
| `update` | yes | Update an event. |
| `delete` | yes | Delete an event. |
| `create-source` | yes | Create a source from JSON attributes. |
| `update-source` | yes | Update source attributes. |
| `delete-source` | yes | Remove a source. |
| `default-source` | yes | Set the default source for new events. |
| `map-source` | yes | Update source-to-profile mappings. |
| `update-account` | yes | Select active calendars in a connected account. |

## Examples

```bash
skycli calendar list --start-date 2026-06-10 --end-date 2026-06-17 --json
skycli calendar week --date 2026-06-10 --json
skycli calendar sources --json
skycli calendar search --query "Dentist" --timezone America/New_York --json
skycli calendar countdowns --start-date 2026-06-10 --end-date 2026-07-10 --timezone America/New_York --json
skycli calendar recent-invites --json
skycli calendar create --title "Dentist" --start-at 2026-06-10T14:00:00-04:00 --end-at 2026-06-10T15:00:00-04:00 --json
skycli calendar create-countdown --title "Beach trip" --date 2026-07-01 --json
```

## Notes

Date filters use the live API `date_min` and `date_max` query keys. Countdown
reads require both bounds. Search and countdown commands default to `UTC` and
include related categories; pass `--timezone` to match the frame's timezone.

Weekly views group timed events in the selected frame timezone.
All-day dates stay unchanged. Missing frame timezone data falls back to the host timezone.

## Event fields

Create/update accept `--categories 7,8`, `--recurrence`, `--invite-emails`,
`--description`, `--location`, `--emoji`, `--timezone`, and `--countdown=true|false`.
`--category` selects one profile. Do not combine it with `--categories`.
Use `--categories ''` or `--invite-emails ''` to clear those lists.

Recurrence accepts `daily`, `weekly:MO,FR`, a raw RRULE, or `none`.
The flag replaces the recurrence set. Use a complete `rrule` array through `--body` to retain EXDATE/RDATE exceptions.
Update/delete accept `--apply-to one|future|all` for recurring events.
Create accepts `--calendar-id` and `--calendar-account-id` for a sync destination.
`--event-type` maps to the app's `kind` field on create only.
The legacy `--color` field remains a passthrough; the audited app does not send it.
Use `--body` or `--body-file` for other verified fields.

Countdown creation sets `countdown_enabled=true` and `all_day=true`.
Weekly JSON retains recurrence, profile relationships, and other upstream metadata while it converts timestamps.

## Source calendars

Use `sources --json` to inspect current IDs and attributes.
`create-source` accepts flat attributes through `--body` or `--body-file`; the client adds the required `attributes` wrapper.
`update-source` sends flat updates and requires `--source-id`.
`delete-source` and `default-source` also require `--source-id`.

`map-source --source-id ID --body-file mapping.json` requires a `categorizations` field.
`update-account --account-id ID --body-file calendars.json` requires an `active_calendars` field.
Use structures from the current account and app contract; the CLI does not invent provider-specific fields.
OAuth connection/reconnection and provider consent remain in the Skylight app.
