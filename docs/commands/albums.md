# albums

Reads and manages photo albums and their messages. `album` is an alias.

## Subcommands

| Subcommand | Mutates | Purpose |
|------------|---------|---------|
| `list` | no | List albums. |
| `messages` | no | List one page of messages in an album. |
| `message-ids` | no | List every message ID in an album. |
| `create` | yes | Create an album. |
| `update` | yes | Rename an album. |
| `delete` | yes | Delete an album. |
| `add` | yes | Add messages to one or more albums. |
| `remove` | yes | Remove messages from one album. |

## Examples

```bash
skycli albums list --json
skycli albums messages --album-id 123 --page 1 --json
skycli albums message-ids --album-id 123 --json
skycli albums create --title "Vacation" --json
skycli albums update --album-id 123 --title "Summer vacation" --json
skycli albums add --album-ids 123,124 --message-ids 901,902 --json
skycli albums remove --album-id 123 --message-ids 901,902 --json
skycli albums delete --album-id 123 --json
```

These private endpoints are available for Skylight Frames with photo-album
support. An empty album list is valid for accounts that have not created one.

Album and message IDs must be positive integers. Create and update require a non-empty title.
`remove` accepts one album ID. It changes album membership; use `photos delete` to delete photo assets.

Use `--readonly` for reads. Both `--readonly` and global `--dry-run` block album changes before an HTTP write.
Request contracts come from the shipped Skylight app. Live album changes require separate account verification.
