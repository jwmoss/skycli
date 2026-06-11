# rotations

Creates rotating chore schedules. `rotation` is an alias.

## Subcommands

| Subcommand | Mutates | Purpose |
|------------|---------|---------|
| `create` | yes | Create chore assignments across assignees and weeks. |

## Examples

```bash
skycli rotations create --chores "Trash,Dishes" --assignee-ids 20431525,20435739 --weeks 4 --points 1 --json
```

## Safety

This command creates chores. Use test input and `--dry-run` when validating command shape.
