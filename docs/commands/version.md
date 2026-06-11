# version

Prints build version metadata.

## Examples

```bash
skycli version
skycli version --json
```

## Output

JSON mode reports version, commit, build date, and any module-version fallback data available from Go build info.

## Safety

Read-only. No Skylight API request is made.
