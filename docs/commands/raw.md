# raw

Sends a raw HTTP request to the Skylight API. Use this for private endpoint discovery before a typed command exists.

## Examples

```bash
skycli --readonly raw /api/frames/5312425 --json
skycli raw --method POST --body '{"label":"Errands"}' /api/frames/5312425/lists --json
echo '{"summary":"x"}' | skycli raw --method POST --body-file - /api/frames/5312425/task_box/items --json
```

## Safety

`--readonly raw ...` allows GET requests and blocks non-GET requests. Off-origin raw URLs do not receive Skylight auth headers.

Place raw command flags before the path. Trailing arguments cause a usage error.
JSON bodies and responses preserve integer precision.
Cross-origin redirects fail before the next request, including OAuth token exchanges.
