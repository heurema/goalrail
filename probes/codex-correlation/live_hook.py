#!/usr/bin/env python3
"""Capture one sanitized receipt from a live Codex SessionStart hook."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
from typing import Any


PROBE_PATH = Path(__file__).with_name("probe.py")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--receipt-dir", type=Path, required=True)
    return parser.parse_args()


def load_object(raw: str) -> dict[str, Any]:
    value = json.loads(raw)
    if not isinstance(value, dict):
        raise ValueError("hook input must be a JSON object")
    return value


def required_string(value: dict[str, Any], key: str) -> str:
    candidate = value.get(key)
    if not isinstance(candidate, str) or not candidate.strip():
        raise ValueError(f"hook input {key} must be a non-empty string")
    return candidate


def write_once(path: Path, payload: dict[str, Any]) -> None:
    descriptor = os.open(path, os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o600)
    with os.fdopen(descriptor, "w", encoding="utf-8") as receipt_file:
        json.dump(payload, receipt_file, sort_keys=True, separators=(",", ":"))
        receipt_file.write("\n")


def main() -> None:
    args = parse_args()
    raw_hook = sys.stdin.read()
    hook_input = load_object(raw_hook)
    session_id = required_string(hook_input, "session_id")
    hook_event_name = required_string(hook_input, "hook_event_name")
    hook_matcher = hook_input.get("source") or hook_input.get("agent_type")

    completed = subprocess.run(
        [sys.executable, str(PROBE_PATH)],
        input=raw_hook,
        text=True,
        capture_output=True,
        env=os.environ.copy(),
        check=False,
    )
    result = load_object(completed.stdout)
    if completed.returncode not in (0, 2):
        raise RuntimeError(f"probe exited with {completed.returncode}")

    surface_hint = os.environ.get("GOALRAIL_PROBE_SURFACE", "unset")
    receipt = {
        "schema_version": 1,
        "surface_hint": surface_hint,
        "observed_session_id": session_id,
        "hook_event_name": hook_event_name,
        "hook_matcher": hook_matcher,
        "probe_exit_code": completed.returncode,
        "result": result,
    }
    receipt_key = "\0".join(
        (session_id, hook_event_name, str(hook_matcher), surface_hint)
    ).encode("utf-8")
    args.receipt_dir.mkdir(parents=True, exist_ok=True)
    write_once(
        args.receipt_dir / f"{hashlib.sha256(receipt_key).hexdigest()}.json",
        receipt,
    )


if __name__ == "__main__":
    main()
