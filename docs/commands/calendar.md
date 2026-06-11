# calendar

Lists and manages calendar events and source calendars.

## Subcommands

| Subcommand | Mutates | Purpose |
|------------|---------|---------|
| `list` | no | List events in a date range. |
| `week` | no | Show events for one week. |
| `sources` | no | List connected calendar sources. |
| `create` | yes | Create an event. |
| `create-countdown` | yes | Create a countdown event. |
| `update` | yes | Update an event. |
| `delete` | yes | Delete an event. |

## Examples

```bash
skycli calendar list --start-date 2026-06-10 --end-date 2026-06-17 --json
skycli calendar week --date 2026-06-10 --json
skycli calendar sources --json
skycli calendar create --title "Dentist" --start-at 2026-06-10T14:00:00-04:00 --end-at 2026-06-10T15:00:00-04:00 --json
skycli calendar create-countdown --title "Beach trip" --date 2026-07-01 --json
```

## Notes

Date filters use the live API `date_min` and `date_max` query keys.
