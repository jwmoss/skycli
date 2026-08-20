# frames

Lists and inspects Skylight frames. `frame` is an alias.

## Subcommands

| Subcommand | Mutates | Purpose |
|------------|---------|---------|
| `list` | no | List frames available to the account. |
| `show` | no | Show one frame. |
| `devices` | no | Return raw device data for a frame. |
| `device` | no | Show one device by ID. |
| `household-config` | no | Return household display configuration. |
| `alarms` | no | List alarms for one device. |
| `notifications` | no | Return event or task notification settings. |
| `month-reviews` | no | List available month reviews. |
| `reminder-profile` | no | Return the reminder profile. |
| `nudges` | no | List recorded nudges in a time range. |
| `avatars` | no | Return avatar metadata. |
| `colors` | no | Return color metadata. |
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
