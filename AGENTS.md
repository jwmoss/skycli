# AGENTS.md

Work style: concise, direct, repo-first. Prefer short bullets over long prose.

## Core

- Repo: `github.com/jwmoss/skycli`.
- Purpose: unofficial CLI for the Skylight Calendar private API.
- Private API: endpoints can change. Verify behavior with `doctor`, `raw`, or typed commands before broad changes.
- Secrets: never print access tokens, refresh tokens, passwords, Keychain values, or 1Password values.
- User-facing behavior changes: update `README.md` and `CHANGELOG.md`.
- Skills are canonical for usage workflow. Read `.agents/skills/skycli/SKILL.md` when using or extending the CLI.

## Build / Test

- Fast gate: `make ci`.
- Individual checks: `make fmt`, `make test`, `make vet`, `make build`.
- Release config check: `go run github.com/goreleaser/goreleaser/v2@latest check`.
- Local release dry run: `go run github.com/goreleaser/goreleaser/v2@latest release --snapshot --clean`.
- Homebrew verify after a release: `brew install jwmoss/tap/skycli && skycli version`.

## Project Defaults

- Go version comes from `go.mod`; do not swap build tools without approval.
- Keep typed command behavior in `internal/cli/`; keep HTTP/API mechanics in `internal/skylight/`.
- Prefer adding focused command helpers over ad hoc JSON string handling.
- Preserve machine-readable output: `--json` writes data to stdout; logs/errors stay on stderr.
- Keep trace logging token-safe.
- Bugs in auth, request construction, output formatting, or safety flags should get tests when practical.

## Live API Safety

- Use `--readonly` or `--dry-run` first when exploring.
- Prefer GET requests and `skycli raw` for endpoint discovery.
- Before mutations, identify frame/category/resource IDs explicitly.
- Do not run destructive commands against a live account unless the user asked for that exact operation.
- When credentials are needed, use existing auth/config/Keychain/1Password paths. Do not paste secrets into code, docs, commits, logs, or release notes.

## Git

- Work from the current checkout unless the user asks for a branch/worktree.
- Safe commands by default: `git status`, `git diff`, `git log`.
- Push only when the user asks.
- Commit directly to `main` when the user explicitly asks.
- Use Conventional Commits: `feat|fix|refactor|build|ci|chore|docs|style|perf|test`.
- Destructive operations require explicit user request: `reset --hard`, `clean`, `restore`, deleting branches/tags, or removing user files.
- If unexpected changes appear, assume another agent/user made them. Work around them or ask if they block the task.

## Release

- Changelog first: user-facing changes belong in `CHANGELOG.md`.
- Tag format: `vX.Y.Z`.
- Release workflow publishes GitHub assets and updates `jwmoss/homebrew-tap`.
- After release, verify GitHub release assets, tap formula, `brew install`, and `skycli version`.
