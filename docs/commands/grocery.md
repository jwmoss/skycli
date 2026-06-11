# grocery

Convenience commands for grocery lists.

## Subcommands

| Subcommand | Mutates | Purpose |
|------------|---------|---------|
| `list` | no | List grocery lists. |
| `show` | no | Show one grocery list. |
| `create` | yes | Create a grocery list. |
| `add` | yes | Add grocery items. |
| `clear` | yes | Clear completed grocery items. |
| `organize` | yes | Organize grocery items. |
| `order` | yes | Start a grocery order. |
| `add-recipe` | yes | Add recipe ingredients to grocery. |

## Examples

```bash
skycli grocery list --json
skycli grocery show --list-id 456 --json
skycli grocery create --title "Groceries" --json
skycli grocery add --list-id 456 --title "Milk" --json
skycli grocery add-recipe --recipe-id 789 --json
```
