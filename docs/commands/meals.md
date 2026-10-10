# meals

Reads and manages meal categories, recipes, sittings, and grocery sync. `meal` is an alias.

## Subcommands

| Subcommand | Mutates | Purpose |
|------------|---------|---------|
| `categories` | no | List meal categories. |
| `update-category` | yes | Update a meal category. |
| `recipes` | no | List recipes. |
| `recipe-info` | no | Show one recipe. |
| `sittings` | no | List meal sittings. |
| `create-recipe` | yes | Create a recipe. |
| `update-recipe` | yes | Update a recipe. |
| `delete-recipe` | yes | Delete a recipe. |
| `create-sitting` | yes | Create a meal sitting. |
| `update-sitting` | yes | Edit a dated meal sitting. |
| `delete-sitting` | yes | Delete a meal sitting. |
| `add-to-grocery` | yes | Add recipe ingredients to grocery. |

## Examples

```bash
skycli meals categories --json
skycli meals update-category --category-id 71 --body-file ./category.json --json
skycli meals recipes --json
skycli meals recipe-info --recipe-id 789 --json
skycli meals sittings --date-min 2026-06-10 --date-max 2026-06-17 --json
skycli meals create-recipe --title "Tacos" --ingredients "tortillas,beans,cheese" --json
skycli meals add-to-grocery --recipe-id 789 --json
skycli meals update-sitting --sitting-id 456 --instance-date 2026-10-10 \
  --apply-to one --summary "Tacos" --recipe-id 789 --json
```

## Update a meal sitting

`update-sitting` requires `--sitting-id`, `--instance-date`, `--apply-to`, and at least one update field.
The instance date identifies the original occurrence. Use `--date YYYY-MM-DD` to move the meal to another date.
Set `--apply-to one`, `future`, or `all` to select the edit scope.
Use `--summary`, `--recipe-id`, and `--meal-category-id` for individual fields.
Use `--rrule 'FREQ=WEEKLY;INTERVAL=1;BYDAY=SA'` to set a recurrence rule.
The CLI passes the supplied recurrence rule unchanged.
Use `--body` or `--body-file` for additional verified API fields. Flags override fields from the JSON body.
To link a recipe, the body can set `meal_recipe_id` and clear `summary` and `description` with JSON `null`.

Both `--readonly` and global `--dry-run` block the update before an HTTP write.
The request contract comes from the shipped Skylight app. Live meal changes require separate account verification.

## Update a meal category

`update-category` requires `--category-id` and a non-empty JSON object from `--body` or `--body-file`.
The CLI sends that object as the category update. Use verified category fields for your account.
Both `--readonly` and global `--dry-run` block the update before an HTTP write.
