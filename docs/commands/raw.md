# raw

Sends a raw HTTP request to the Skylight API. Use this for private endpoint discovery before a typed command exists.

## Examples

```bash
skycli --readonly raw /api/frames/5312425 --json
skycli raw --method POST /api/frames/5312425/lists --body '{"label":"Errands"}' --json
echo '{"summary":"x"}' | skycli raw --method POST --body-file - /api/frames/5312425/task_box/items --json
```

## Safety

`--readonly raw ...` allows GET requests and blocks non-GET requests. Off-origin raw URLs do not receive Skylight auth headers.
