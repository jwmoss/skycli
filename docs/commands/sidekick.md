# sidekick

Inspect Plus access and use Skylight's JSON Sidekick workflows.

| Subcommand | Mutates | Purpose |
|---|---|---|
| `status` | no | Report sanitized Plus access; this is the default. |
| `history` | no | List auto-creation intents. |
| `show` | no | Inspect an intent with `--id`. |
| `items` | no | Read an intent's generated items. |
| `drafts` | no | Read generated drafts by kind. |
| `create` | yes | Submit a JSON import or generation request. |
| `approve` | yes | Approve selected draft IDs. |
| `undo` | yes | Undo an intent's created items. |

```bash
skycli --readonly sidekick status --json
skycli --readonly sidekick history --json
skycli sidekick create --engine event_importer --text "Soccer on October 12 at 5pm" --json
skycli sidekick show --id 123 --json
skycli sidekick drafts --id 123 --kind events --timezone America/New_York --json
skycli sidekick approve --id 123 --kind events --ids 456,457 --json
skycli sidekick undo --id 123 --json
```

Create defaults to `draft_first=true` and `created_via=app_form`.
An explicit body value or `--draft-first=false` can change draft behavior.
An intent may complete asynchronously. Inspect `show` before you read or approve its results.
The API controls whether each engine supports drafts.

Known JSON engines include `event_importer`, `list_importer`, `recipe_importer`,
`create_a_recipe`, `meal_sittings_generator`, `activity_ideas_generator`,
`grocery_list_organizer`, and `instacart_shopping_list_generator`.
Use `--text`, `--content-url`, or `--body-file` for the engine's inputs.
Generation engines can need an `engine_inputs` object; see the [contract audit](../api-contracts-2026-10.md).
The CLI requires an engine; Skylight validates engine-specific fields and account access.

Draft kinds are `events`, `recipes`, `meals`, `lists`, and `list-items`.
Approval requires `--ids`. Approve `list-items`, not list containers.
File/photo/PDF uploads and edits within unapproved drafts remain unsupported.

`--readonly` permits status/history/show/items/drafts.
Create, approve, and undo change remote data and require appropriate Plus access.
Write contracts come from the app; live tests do not submit Sidekick jobs.
