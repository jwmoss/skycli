# categories

Read and manage household profiles and labels. `category` is an alias.

| Subcommand | Mutates | Purpose |
|---|---|---|
| `list` | no | List profiles and labels; this is the default. |
| `show` | no | Read one category with `--id`. |
| `create` | yes | Create a profile or label. |
| `update` | yes | Edit a category or convert its type. |
| `delete` | yes | Delete a category with optional reassignment. |

```bash
skycli --readonly categories --json
skycli categories show --id 7 --json
skycli categories create --label "Example" --color '#123456' --profile --json
skycli categories update --id 7 --profile=false --json
skycli categories update --id 7 --body '{"hat_id":"8"}' --json
skycli categories delete --id 7 --reassign-to 8 --json
```

Create requires `--label` and `--color`, or matching JSON fields.
Use `--profile=true` for a profile and `--profile=false` for a label.
The CLI also sets `selected_for_chore_chart` to match that choice.
Create and update accept `--body` or `--body-file` for app fields such as `avatar_id` and `hat_id`.
Use `frames hats` to inspect available hat packs. Custom profile-image uploads are not supported.

`--reassign-to` supplies the replacement category for deletion.
Omit it only when you intend deletion without reassignment.
JSON list output preserves all upstream attributes and relationships.
`--readonly` permits list/show and blocks all writes.
