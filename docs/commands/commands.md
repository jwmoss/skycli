# commands

Prints the machine-readable command catalog.

## Examples

```bash
skycli commands
skycli commands --json
skycli --json
```

## Output

Human mode prints a table of commands, read-only status, mutation status, and summaries. JSON mode prints the full catalog, including docs paths, global flags, environment variables, examples, and output contract.

## Safety

Read-only. No Skylight API request is made.
