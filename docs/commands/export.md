# export

Export selected resource definitions to portable JSON. This is not a complete account backup.

## Examples

```bash
skycli export --resources all --days 90 --output-file skylight-export.json --json
skycli --readonly export --resources lists,calendar --days 30 --json
```

## Output

When `--output-file` is set, the portable export is written to that file and JSON stdout reports the export result. Without `--output-file`, stdout contains the export document.

## Safety

Read-only against the Skylight account. It may write a local output file.

## Limits

`all` includes chores, rewards, lists and their items, recipes, meal sittings, and calendar events.
The date window limits chores, sittings, and events.
Export omits profiles, routines, media, device settings, subscriptions, and Sidekick history.
It also omits chore completion state, reward redemption state, and calendar recurrence/source fields.
Use the file as a template within the same frame. Import creates new resources.
