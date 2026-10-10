# frames

Lists Skylight frames and manages device settings. `frame` is an alias.

## Subcommands

| Subcommand | Mutates | Purpose |
|------------|---------|---------|
| `list` | no | List frames available to the account. |
| `show` | no | Show one frame. |
| `devices` | no | Return raw device data for a frame. |
| `users` | no | List household users. |
| `device-config` | no | Read device feature configuration. |
| `update-device-config` | yes | Update device feature configuration. |
| `device` | no | Show one device by ID. |
| `update-device` | yes | Update device settings from a JSON object. |
| `sleep-device` | yes | Put one device into sleep mode. |
| `wake-device` | yes | Wake one device from sleep mode. |
| `household-config` | no | Return household display configuration. |
| `update-household-config` | yes | Update household configuration from a JSON object. |
| `alarms` | no | List alarms for one device. |
| `notifications` | no | Return event or task notification settings. |
| `update-notifications` | yes | Update event or task notification settings. |
| `month-reviews` | no | List available month reviews. |
| `reminder-profile` | no | Return the reminder profile. |
| `nudges` | no | List recorded nudges in a time range. |
| `create-nudge` | yes | Create a JSON text nudge. |
| `update-nudge` | yes | Update a JSON text nudge. |
| `delete-nudge` | yes | Delete a nudge or one scheduled occurrence. |
| `avatars` | no | Return avatar metadata. |
| `colors` | no | Return color metadata. |
| `hats` | no | Return available profile hat packs. |
| `set-default` | yes | Save a default frame ID in config. |

## Examples

```bash
skycli frames --json
skycli frames list --json
skycli frames show --id 5312425 --json
skycli --frame 5312425 frames devices --json
skycli --frame 5312425 frames device --device-id 5596817 --json
skycli --frame 5312425 frames household-config --json
skycli --frame 5312425 frames alarms --device-id 5596817 --json
skycli --frame 5312425 frames notifications --type event --json
skycli --frame 5312425 frames notifications --type task --json
skycli frames month-reviews --json
skycli frames reminder-profile --json
skycli --frame 5312425 frames nudges \
  --after 2026-08-18T00:00:00Z --before 2026-08-19T00:00:00Z --json
skycli frames set-default 5312425 --json
```

## Output

`frames list` and `frames show` return typed frame data. Other read commands
return Skylight's API response shape.

## Change settings

Device changes require a positive `--device-id`. Use `frames devices` to find device IDs.
Update commands accept `--body` or `--body-file`. The body must contain a non-empty JSON object.
Use the corresponding read command to inspect current settings before a change.

```bash
skycli frames update-device --device-id 5596817 --body '{"brightness":50}' --json
skycli frames sleep-device --device-id 5596817 --json
skycli frames wake-device --device-id 5596817 --json
skycli frames update-household-config --body-file household-updates.json --json
skycli frames update-notifications --type event --on-time=true \
  --early=true --early-minutes-before 30 --json
skycli frames update-notifications --type task \
  --body '{"task_due":{"enabled":true},"task_completed":{"enabled":false}}' --json
```

Event notification flags override fields from the JSON body.
Task notification settings use nested objects. Their `enabled` fields accept booleans.
Both `--readonly` and global `--dry-run` block settings changes before an HTTP write.
These request contracts come from the shipped Skylight app. Live settings changes require separate account verification.

## Device features and nudges

`device-config` and `update-device-config` require `--device-id`.
The update accepts a flat JSON object through `--body` or `--body-file`.
The audit verifies the route, but does not identify the new announcement-frequency field.

Nudge create/update accept `--body` or `--body-file`.
Update/delete require `--id`. Delete accepts optional `--deliver-at` in RFC3339 form to select an occurrence.
JSON nudges support `kirk_voice` or `silent`; parent-voice audio requires an unsupported multipart upload.
Nudge write contracts come from the app; live tests do not send reminders to the household.
