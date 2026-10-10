# photos

Lists, uploads, downloads, and manages photos. `photo` is an alias.

## Subcommands

| Subcommand | Mutates | Purpose |
|------------|---------|---------|
| `list` | no | List photos/messages. |
| `show` | no | Show photo/message details. |
| `likes` | no | List likes on a photo/message. |
| `comments` | no | List comments on a photo/message. |
| `download` | no | Download a photo asset URL. |
| `upload` | yes | Upload a photo. |
| `delete` | yes | Delete photos/messages. |
| `caption` | yes | Set or clear a photo caption. |
| `like` | yes | Like a photo. |
| `unlike` | yes | Remove your like. |
| `comment` | yes | Add a photo comment. |
| `delete-comment` | yes | Delete a photo comment. |
| `copy` | yes | Copy photos to other frames. |

## Examples

```bash
skycli photos list --json
skycli photos show --message-id 1816641037 --json
skycli photos likes --message-id 1816641037 --json
skycli photos comments --message-id 1816641037 --page 1 --json
skycli photos upload --file ./photo.jpg --caption "May" --json
skycli photos download --asset-url "$URL" --out ./photo.jpg --json
skycli photos delete --message-ids 10,11 --json
skycli photos caption --message-id 1816641037 --caption "Fall" --json
skycli photos caption --message-id 1816641037 --caption "" --json
skycli photos like --message-id 1816641037 --json
skycli photos unlike --message-id 1816641037 --json
skycli photos comment --message-id 1816641037 --text "Great photo!" --json
skycli photos delete-comment --message-id 1816641037 --comment-id 72 --json
skycli photos copy --frame 123 --message-ids 10,11 --frame-ids 124,125 --json
```

## Safety

`download` writes a local file. The commands marked `yes` mutate the Skylight account.
Both `--readonly` and global `--dry-run` block these account changes before an HTTP write.

`copy` uses `--frame` as the source and `--frame-ids` as the destinations.
Photo, comment, and frame identifiers must be positive integers for these write commands.
`delete` sends photo IDs as repeated `message_ids[]` query parameters, as the shipped app does.

Upload and download use the global `--timeout` value.
Uploads stream the local file. Downloads replace the destination only after a complete transfer.
Asset requests do not receive Skylight credentials.
