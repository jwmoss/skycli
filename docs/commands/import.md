# import

Imports data from a `skycli export` file.

## Examples

```bash
skycli import --file skylight-export.json --dry-run --json
skycli import --file skylight-export.json --resources lists,calendar --json
```

## Safety

This command mutates the Skylight account unless `--dry-run` is set. Always run a dry run first and inspect the JSON plan before importing.
