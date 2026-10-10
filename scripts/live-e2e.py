#!/usr/bin/env python3
"""Test the built CLI against a real Skylight account."""

import argparse
import json
import os
import shlex
import subprocess
import sys
import uuid
from datetime import date, timedelta


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--readonly", action="store_true", help="run GET checks only")
    parser.add_argument("--frame", default=os.environ.get("SKYLIGHT_FRAME_ID"))
    args = parser.parse_args()
    cli = shlex.split(os.environ.get("SKYCLI_BIN", "./skycli"))

    def call(*command):
        result = subprocess.run(
            [*cli, "--json", "--timeout", "30s", *command],
            capture_output=True, text=True, timeout=40,
        )
        try:
            response = json.loads(result.stdout)
        except ValueError as error:
            raise RuntimeError("CLI returned invalid JSON") from error
        if result.returncode:
            status = response.get("http_status", "unavailable")
            action = " ".join(command[:2])
            raise RuntimeError(f"{action} failed: exit {result.returncode}, HTTP {status}")
        return response

    frame = str(args.frame or call("--readonly", "config", "get", "frame")["value"])
    if not frame.isdecimal() or int(frame) < 1:
        raise RuntimeError("Set a default frame or pass --frame ID")
    cli += ["--frame", frame]
    print(f"Live Skylight tests: frame {frame}", flush=True)

    today = date.today()
    for command in [("categories",), ("chores", "list"), ("routines", "list"),
                    ("calendar", "list", "--start-date", today.isoformat(),
                     "--end-date", (today + timedelta(days=7)).isoformat()),
                    ("lists", "list")]:
        response = call("--readonly", *command)
        rows = response if isinstance(response, list) else response.get("data")
        if not isinstance(rows, list):
            raise RuntimeError(f"{' '.join(command)} did not return a collection")
        print(f"PASS GET {' '.join(command)}", flush=True)

    if args.readonly:
        return

    title = f"skycli smoke test {uuid.uuid4().hex}"
    print(f"Temporary hidden list: {title}", flush=True)
    created = call("lists", "create", "--title", title, "--hide-from-frame")
    list_id = str(created["data"]["id"])
    if not list_id.isdecimal() or int(list_id) < 1:
        raise RuntimeError(f"Create returned an invalid ID; inspect the list named {title}")
    try:
        saved = call("--readonly", "lists", "show", "--list-id", list_id)["data"]
        if saved["id"] != list_id or saved["attributes"]["label"] != title:
            raise RuntimeError("Created list did not match the saved list")
        print("PASS POST list and GET saved list", flush=True)
    finally:
        print(f"Remove temporary list {list_id}", flush=True)
        call("lists", "delete", "--list-id", list_id)
        remaining = call("--readonly", "lists", "list")["data"]
        if any(str(item["id"]) == list_id for item in remaining):
            raise RuntimeError(f"Temporary list {list_id} remains after deletion")
        print("PASS temporary list cleanup", flush=True)


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, KeyError, TypeError, OSError, subprocess.TimeoutExpired) as error:
        print(f"FAIL {error}", file=sys.stderr)
        sys.exit(1)
