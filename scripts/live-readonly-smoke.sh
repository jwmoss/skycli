#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if [[ -n "${SKYCLI_BIN:-}" ]]; then
  # shellcheck disable=SC2206
  SKYCLI_CMD=(${SKYCLI_BIN})
elif [[ -x "./skycli" ]]; then
  SKYCLI_CMD=(./skycli)
else
  SKYCLI_CMD=(go run .)
fi

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

TODAY="$(python3 - <<'PY'
from datetime import date
print(date.today().isoformat())
PY
)"
NEXT_WEEK="$(python3 - <<'PY'
from datetime import date, timedelta
print((date.today() + timedelta(days=7)).isoformat())
PY
)"

json_value() {
  python3 - "$1" "$2" <<'PY'
import json
import sys

path, expr = sys.argv[1], sys.argv[2]
with open(path, "r", encoding="utf-8") as f:
    data = json.load(f)

cur = data
for part in expr.split("."):
    if not part:
        continue
    if part.endswith("]"):
        name, idx = part[:-1].split("[", 1)
        if name:
            cur = cur.get(name)
        cur = cur[int(idx)]
    else:
        cur = cur.get(part)
    if cur is None:
        sys.exit(0)

print(cur)
PY
}

validate_json() {
  local file="$1"
  python3 - "$file" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as f:
    decoder = json.JSONDecoder()
    data = f.read().strip()
    if not data:
        raise SystemExit("empty stdout")
    _, idx = decoder.raw_decode(data)
    if data[idx:].strip():
        raise SystemExit("stdout contains multiple JSON documents")
PY
}

run_json() {
  local name="$1"
  shift
  local out="$TMPDIR/${name//[^A-Za-z0-9_.-]/_}.json"
  local err="$TMPDIR/${name//[^A-Za-z0-9_.-]/_}.err"
  printf '==> %s\n' "$name" >&2
  if ! "${SKYCLI_CMD[@]}" "$@" >"$out" 2>"$err"; then
    printf 'FAIL %s\n' "$name" >&2
    sed -n '1,120p' "$err" >&2
    return 1
  fi
  if ! validate_json "$out"; then
    printf 'FAIL %s: stdout was not a single JSON document\n' "$name" >&2
    sed -n '1,80p' "$out" >&2
    sed -n '1,80p' "$err" >&2
    return 1
  fi
  printf 'OK   %s\n' "$name" >&2
  printf '%s\n' "$out"
}

run_optional_json() {
  local name="$1"
  shift
  if ! run_json "$name" "$@" >/dev/null; then
    printf 'SKIP %s: endpoint unavailable for this account or feature set\n' "$name" >&2
  fi
  return 0
}

run_json "commands" commands --json >/dev/null
run_json "auth-status" auth status --json >/dev/null

frames_out="$(run_json "frames-list" --readonly frames list --json)"
FRAME_ID="${SKYCLI_FRAME_ID:-${SKYLIGHT_FRAME_ID:-}}"
if [[ -z "$FRAME_ID" ]]; then
  FRAME_ID="$(json_value "$frames_out" "[0].id")"
fi
if [[ -z "$FRAME_ID" ]]; then
  printf 'FAIL no frame id found; set SKYCLI_FRAME_ID or SKYLIGHT_FRAME_ID\n' >&2
  exit 1
fi

FRAME_FLAGS=(--readonly --frame "$FRAME_ID")

run_json "doctor-flag" "${FRAME_FLAGS[@]}" --doctor --json >/dev/null
run_json "frames-show" --readonly frames show --id "$FRAME_ID" --json >/dev/null
run_json "frames-devices" "${FRAME_FLAGS[@]}" frames devices --json >/dev/null
run_json "frames-avatars" --readonly frames avatars --json >/dev/null
run_json "frames-colors" --readonly frames colors --json >/dev/null
run_json "categories" "${FRAME_FLAGS[@]}" categories --json >/dev/null
run_json "chores-list" "${FRAME_FLAGS[@]}" chores list --date "$TODAY" --json >/dev/null
run_json "chores-week" "${FRAME_FLAGS[@]}" chores week --date "$TODAY" --json >/dev/null
run_json "chores-streak" "${FRAME_FLAGS[@]}" chores streak --days 7 --json >/dev/null
run_json "rewards-list" "${FRAME_FLAGS[@]}" rewards list --json >/dev/null
run_json "rewards-points" "${FRAME_FLAGS[@]}" rewards points --json >/dev/null
run_json "calendar-list" "${FRAME_FLAGS[@]}" calendar list --start-date "$TODAY" --end-date "$NEXT_WEEK" --json >/dev/null
run_json "calendar-week" "${FRAME_FLAGS[@]}" calendar week --date "$TODAY" --json >/dev/null
run_json "calendar-sources" "${FRAME_FLAGS[@]}" calendar sources --json >/dev/null

lists_out="$(run_json "lists-list" "${FRAME_FLAGS[@]}" lists list --json)"
LIST_ID="$(json_value "$lists_out" "data[0].id")"
if [[ -n "$LIST_ID" ]]; then
  run_json "lists-show" "${FRAME_FLAGS[@]}" lists show --list-id "$LIST_ID" --json >/dev/null
else
  printf 'SKIP lists-show: no lists returned\n' >&2
fi
run_json "lists-task-box-items" "${FRAME_FLAGS[@]}" lists task-box-items --json >/dev/null
run_json "grocery-list" "${FRAME_FLAGS[@]}" grocery list --json >/dev/null

run_json "meals-categories" "${FRAME_FLAGS[@]}" meals categories --json >/dev/null
recipes_out="$(run_json "meals-recipes" "${FRAME_FLAGS[@]}" meals recipes --json)"
RECIPE_ID="$(json_value "$recipes_out" "data[0].id")"
if [[ -n "$RECIPE_ID" ]]; then
  run_json "meals-recipe-info" "${FRAME_FLAGS[@]}" meals recipe-info --recipe-id "$RECIPE_ID" --json >/dev/null
else
  printf 'SKIP meals-recipe-info: no recipes returned\n' >&2
fi
run_json "meals-sittings" "${FRAME_FLAGS[@]}" meals sittings --date-min "$TODAY" --date-max "$NEXT_WEEK" --json >/dev/null
run_json "photos-list" "${FRAME_FLAGS[@]}" photos list --json >/dev/null
run_optional_json "routines-list" "${FRAME_FLAGS[@]}" routines list --json
run_json "bounties-list" "${FRAME_FLAGS[@]}" bounties list --json >/dev/null
run_json "status" "${FRAME_FLAGS[@]}" status --json >/dev/null
run_json "analytics" "${FRAME_FLAGS[@]}" analytics --days 7 --json >/dev/null
run_json "home" "${FRAME_FLAGS[@]}" home --date "$TODAY" --json >/dev/null
run_json "watch-once" "${FRAME_FLAGS[@]}" watch --once --json >/dev/null
run_json "export-all" "${FRAME_FLAGS[@]}" export --resources all --days 1 --output-file "$TMPDIR/export.json" --json >/dev/null
run_json "raw-frame" --readonly raw "/api/frames/$FRAME_ID" --json >/dev/null

printf 'readonly live smoke passed for frame %s\n' "$FRAME_ID" >&2
