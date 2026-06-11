# photos

Lists, uploads, downloads, and deletes photos. `photo` is an alias.

## Subcommands

| Subcommand | Mutates | Purpose |
|------------|---------|---------|
| `list` | no | List photos/messages. |
| `download` | no | Download a photo asset URL. |
| `upload` | yes | Upload a photo. |
| `delete` | yes | Delete photos/messages. |

## Examples

```bash
skycli photos list --json
skycli photos upload --file ./photo.jpg --caption "May" --json
skycli photos download --asset-url "$URL" --out ./photo.jpg --json
skycli photos delete --message-ids 10,11 --json
```

## Safety

`download` is a file write but does not mutate the Skylight account. `upload` and `delete` mutate the account.
