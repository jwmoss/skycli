# Status and analytics report reads

Both reports resolve their frame and credentials once. They reuse one authenticated client for five GET requests in batches of up to three. Each request uses the command context. Errors retain their resource labels and follow the original command order. Analytics resolves count ties by title and name ties by category ID.

Five alternating live runs per binary and command produced these median command times. Each run returned five successful GET responses. The status JSON matched. Analytics counts, balances, dates, and reward totals matched. Repeated runs also returned the same ranked output on the new binary.

| Command | Sequential | Batches of three |
| --- | ---: | ---: |
| `status` | 614 ms | 408 ms |
| `analytics --days 7` | 689 ms | 422 ms |
