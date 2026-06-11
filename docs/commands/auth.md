# auth

Manages credentials and reports auth state.

## Subcommands

| Subcommand | Mutates | Purpose |
|------------|---------|---------|
| `login` | yes | Log in with email/password and save returned tokens. |
| `import-mac` | yes | Import tokens from the macOS Skylight app container. |
| `refresh` | yes | Refresh and save a configured token. |
| `set-token` | yes | Store a pasted access token. |
| `status` | no | Show auth/config status without printing secrets. |

## Examples

```bash
skycli auth status --json
skycli auth login --email you@example.com
printf '%s\n' "$SKYLIGHT_PASSWORD" | skycli auth login --email you@example.com --password-stdin --json
skycli auth import-mac --json
skycli auth refresh --json
skycli auth set-token --json
```

## Notes

Token resolution order is `--token`, `SKYLIGHT_ACCESS_TOKEN`, then configured secret storage. Config-backed tokens can auto-refresh when a refresh token and device fingerprint are available.
