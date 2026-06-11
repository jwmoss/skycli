# --doctor

Runs readonly token and API connectivity checks.

## Usage

```bash
skycli --doctor
skycli --doctor --json
skycli --readonly --doctor --json
```

## Output

Human mode prints the health-check summary. JSON mode emits one document with the checks and final status. A failing check returns a non-zero exit code.

## Notes

`--doctor` is the public interface. `skycli doctor` remains available as a compatibility alias for older scripts, but docs and automation should prefer the root flag.
