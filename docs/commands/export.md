# export

Exports frame data to portable JSON.

## Examples

```bash
skycli export --resources all --days 90 --output-file skylight-export.json --json
skycli --readonly export --resources lists,calendar --days 30 --json
```

## Output

When `--output-file` is set, the portable export is written to that file and JSON stdout reports the export result. Without `--output-file`, stdout contains the export document.

## Safety

Read-only against the Skylight account. It may write a local output file.
