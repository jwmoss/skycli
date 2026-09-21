# import

Create resources from a `skycli export` file on the same frame.
This command appends resources. A second run can create duplicates.

## Examples

```bash
skycli import --file skylight-export.json --dry-run --json
skycli import --file skylight-export.json --resources lists,calendar --json
```

## Validation

Dry-run checks the target frame and references before it reports resource counts.
It may make GET requests and requires account access.
Both `skycli --dry-run import ...` and `skycli import --dry-run ...` perform this check.

Import rejects a different source frame because category mapping is not available.
Import verifies category and meal-category IDs before the first write.
It maps imported recipe IDs to their new IDs before it creates meal sittings.
If recipe creation fails, import skips dependent sittings and reports the failure.
Existing recipe references must resolve on the target frame.

This is a portable template import, not a complete account restore.
See [export limits](export.md). Inspect the dry-run result before a real import.
