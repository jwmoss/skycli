# rewards

Lists and manages rewards and point balances. `reward` is an alias.

## Subcommands

| Subcommand | Mutates | Purpose |
|------------|---------|---------|
| `list` | no | List rewards. |
| `points` | no | List point balances. |
| `create` | yes | Create rewards. |
| `update` | yes | Update a reward. |
| `redeem` | yes | Redeem a reward. |
| `unredeem` | yes | Undo a redemption. |
| `delete` | yes | Delete a reward. |
| `bulk` | yes | Bulk-create rewards. |

## Examples

```bash
skycli rewards list --json
skycli rewards points --json
skycli rewards create --name "TV Ticket" --points 10 --categories 20431525,20435739 --respawn --json
skycli rewards redeem --id 9957645 --json
```

## Safety

Use `--readonly` for `list` and `points`. Other subcommands mutate rewards.
