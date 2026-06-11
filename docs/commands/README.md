# Command index

Public command pages are listed in CLI-help order.

| Command | Purpose |
|---------|---------|
| [`commands`](commands.md) | Print the command catalog for agents. |
| [`auth`](auth.md) | Manage credentials and inspect auth state. |
| [`frames`](frames.md) | List, inspect, and set the default frame. |
| [`categories`](categories.md) | List household categories. |
| [`chores`](chores.md) | List and manage chores. |
| [`rewards`](rewards.md) | List and manage rewards and point balances. |
| [`calendar`](calendar.md) | List and manage events and calendar sources. |
| [`lists`](lists.md) | List and manage Skylight lists and task-box items. |
| [`grocery`](grocery.md) | Convenience commands for grocery lists. |
| [`meals`](meals.md) | Read and manage meal categories, recipes, and sittings. |
| [`photos`](photos.md) | List, upload, download, and delete photos. |
| [`routines`](routines.md) | List and manage routines when the private endpoint is available. |
| [`bounties`](bounties.md) | Pair chores and rewards into bounty workflows. |
| [`rotations`](rotations.md) | Create rotating chore schedules. |
| [`status`](status.md) | Show a quick connected-frame overview. |
| [`analytics`](analytics.md) | Compute family activity statistics. |
| [`home`](home.md) | Show a weekly combined home view. |
| [`watch`](watch.md) | Poll for resource changes. |
| [`export`](export.md) | Export frame data to portable JSON. |
| [`import`](import.md) | Import a skycli export. |
| [`config`](config.md) | Show or update skycli configuration. |
| [`raw`](raw.md) | Send raw HTTP requests to the Skylight API. |
| [`version`](version.md) | Print build version metadata. |

Health checks are a root flag, not a command:

```bash
skycli --doctor --json
```

See [doctor](doctor.md).
