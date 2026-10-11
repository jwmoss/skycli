# doctor

Runs readonly token and API connectivity checks.

## Usage

```bash
skycli doctor
skycli --doctor
skycli --doctor --json
skycli --readonly --doctor --json
```

## Output

Human mode prints the health-check summary. JSON mode emits one document with the checks and final status. A failing check returns a non-zero exit code.

## Notes

`doctor` and the root `--doctor` flag run the same read-only checks.
Refresh expired credentials explicitly before these checks.
