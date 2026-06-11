# meals

Reads and manages meal categories, recipes, sittings, and grocery sync. `meal` is an alias.

## Subcommands

| Subcommand | Mutates | Purpose |
|------------|---------|---------|
| `categories` | no | List meal categories. |
| `recipes` | no | List recipes. |
| `recipe-info` | no | Show one recipe. |
| `sittings` | no | List meal sittings. |
| `create-recipe` | yes | Create a recipe. |
| `update-recipe` | yes | Update a recipe. |
| `delete-recipe` | yes | Delete a recipe. |
| `create-sitting` | yes | Create a meal sitting. |
| `delete-sitting` | yes | Delete a meal sitting. |
| `add-to-grocery` | yes | Add recipe ingredients to grocery. |

## Examples

```bash
skycli meals categories --json
skycli meals recipes --json
skycli meals recipe-info --recipe-id 789 --json
skycli meals sittings --date-min 2026-06-10 --date-max 2026-06-17 --json
skycli meals create-recipe --title "Tacos" --ingredients "tortillas,beans,cheese" --json
skycli meals add-to-grocery --recipe-id 789 --json
```
