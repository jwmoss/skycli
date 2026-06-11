# frames

Lists and inspects Skylight frames. `frame` is an alias.

## Subcommands

| Subcommand | Mutates | Purpose |
|------------|---------|---------|
| `list` | no | List frames available to the account. |
| `show` | no | Show one frame. |
| `devices` | no | Return raw device data for a frame. |
| `avatars` | no | Return avatar metadata. |
| `colors` | no | Return color metadata. |
| `set-default` | yes | Save a default frame ID in config. |

## Examples

```bash
skycli frames --json
skycli frames list --json
skycli frames show --id 5312425 --json
skycli --frame 5312425 frames devices --json
skycli frames set-default 5312425 --json
```

## Output

`frames list` and `frames show` return typed frame data. `devices`, `avatars`, and `colors` return the API response shape.
