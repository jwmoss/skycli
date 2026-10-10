# lists

Lists and manages Skylight lists and task-box items. `list` is an alias.

## Subcommands

| Subcommand | Mutates | Purpose |
|------------|---------|---------|
| `list` | no | List lists. |
| `show` | no | Show a list and included items. |
| `task-box-items` | no | List task-box items. |
| `create` | yes | Create a list. |
| `update` | yes | Update a list. |
| `delete` | yes | Delete a list. |
| `add-item` | yes | Add an item. |
| `update-item` | yes | Update an item. |
| `delete-item` | yes | Delete an item. |
| `clear-completed` | yes | Delete completed items. |
| `organize` | yes | Ask Skylight to organize a list. |
| `order` | yes | Start a grocery order. |
| `task-box-item` | yes | Create a task-box item. |
| `update-task-box-item` | yes | Edit a saved task template. |
| `delete-task-box-item` | yes | Delete a saved task template. |

## Examples

```bash
skycli lists list --json
skycli lists show --list-id 123 --json
skycli lists task-box-items --json
skycli lists create --title "Errands" --kind to_do --json
skycli lists add-item --list-id 123 --title "Return books" --json
```

## Safety

Use `--readonly` for `list`, `show`, and `task-box-items`.

## Task Box

```bash
skycli lists task-box-item --title "Read" --body '{"emoji_icon":"📚","reward_points":2}' --json
skycli lists update-task-box-item --item-id 7 --title "Read a chapter" --json
skycli lists delete-task-box-item --item-id 7 --json
```

Create/update accept `--body` or `--body-file`. `--title` maps to the flat `summary` field.
App fields include `summary`, `emoji_icon`, `routine`, and `reward_points`.
To assign a template, read its fields and use `chores create` or `routines create`.
The app has no separate Task Box apply endpoint in the audited bundle.
