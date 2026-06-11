# config

Shows or updates local skycli configuration.

## Subcommands

| Subcommand | Mutates | Purpose |
|------------|---------|---------|
| `show` | no | Show non-secret config. |
| `get` | no | Get one config value. |
| `set` | yes | Set one config value. |
| `unset` | yes | Unset one config value. |
| `edit` | yes | Edit config in `$EDITOR`. |

## Examples

```bash
skycli config show --json
skycli config get base_url --json
skycli config set api_version 2026-04-15 --json
skycli config unset default_frame_id --json
```

## Secret handling

Secret values are masked by default. `config get access_token --show-secrets` and `config get refresh_token --show-secrets` print raw values and should only be used deliberately.
