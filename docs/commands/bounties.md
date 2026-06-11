# bounties

Pairs a chore and reward into a bounty workflow. `bounty` is an alias.

## Subcommands

| Subcommand | Mutates | Purpose |
|------------|---------|---------|
| `list` | no | List inferred bounty pairs. |
| `create` | yes | Create a chore and paired reward. |
| `update` | yes | Update the chore and reward pair. |
| `delete` | yes | Delete the chore and reward pair. |

## Examples

```bash
skycli bounties list --json
skycli bounties create --title "Clean garage shelf" --points 5 --assignee-id 20431525 --reward-title "Garage bounty" --json
skycli bounties update --chore-id 81779567 --reward-id 9957645 --title "Garage updated" --json
```

## Notes

Update/delete can involve two API mutations. JSON output reports partial failures if one side succeeds and the other fails.
