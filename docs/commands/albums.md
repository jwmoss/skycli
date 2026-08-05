# albums

Reads photo albums and the messages assigned to them. `album` is an alias.

## Subcommands

| Subcommand | Mutates | Purpose |
|------------|---------|---------|
| `list` | no | List albums. |
| `messages` | no | List one page of messages in an album. |
| `message-ids` | no | List every message ID in an album. |

## Examples

```bash
skycli albums list --json
skycli albums messages --album-id 123 --page 1 --json
skycli albums message-ids --album-id 123 --json
```

These private endpoints are available for Skylight Frames with photo-album
support. An empty album list is valid for accounts that have not created one.
