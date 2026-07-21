#!/usr/bin/env python3
"""Tests for the sanitized live-hook receipt wrapper."""

from __future__ import annotations

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from typing import Any


PROBE_DIR = Path(__file__).resolve().parent
TESTDATA_DIR = PROBE_DIR / "testdata"


def load_fixture(name: str) -> dict[str, Any]:
    with (TESTDATA_DIR / name).open(encoding="utf-8") as fixture_file:
        value = json.load(fixture_file)
    if not isinstance(value, dict):
        raise AssertionError(f"{name} must contain a JSON object")
    return value


def run_live_hook(*, with_context: bool) -> dict[str, Any]:
    hook_input = load_fixture("session-start.json")
    environment = os.environ.copy()
    environment["GOALRAIL_PROBE_SURFACE"] = "test"
    if with_context:
        environment["GOALRAIL_RUN_CONTEXT"] = json.dumps(
            load_fixture("run-context.json"),
            sort_keys=True,
            separators=(",", ":"),
        )
    else:
        environment.pop("GOALRAIL_RUN_CONTEXT", None)

    with tempfile.TemporaryDirectory() as receipt_dir:
        completed = subprocess.run(
            [
                sys.executable,
                str(PROBE_DIR / "live_hook.py"),
                "--receipt-dir",
                receipt_dir,
            ],
            input=json.dumps(hook_input),
            text=True,
            capture_output=True,
            env=environment,
            check=False,
        )
        if completed.returncode != 0 or completed.stdout or completed.stderr:
            raise AssertionError(
                "live hook must exit silently with zero: "
                f"code={completed.returncode} stdout={completed.stdout!r} "
                f"stderr={completed.stderr!r}"
            )
        receipt_paths = list(Path(receipt_dir).glob("*.json"))
        if len(receipt_paths) != 1:
            raise AssertionError(f"expected one receipt, found {receipt_paths!r}")
        with receipt_paths[0].open(encoding="utf-8") as receipt_file:
            receipt = json.load(receipt_file)
        if not isinstance(receipt, dict):
            raise AssertionError("receipt must contain a JSON object")
        return receipt


class LiveHookTests(unittest.TestCase):
    def test_verified_receipt_contains_only_sanitized_metadata(self) -> None:
        receipt = run_live_hook(with_context=True)
        self.assertEqual(receipt["surface_hint"], "test")
        self.assertEqual(receipt["result"]["status"], "VERIFIED")
        self.assertNotIn("transcript", json.dumps(receipt))

    def test_missing_context_receipt_is_explicitly_unlinked(self) -> None:
        receipt = run_live_hook(with_context=False)
        self.assertEqual(receipt["probe_exit_code"], 2)
        self.assertEqual(receipt["result"]["status"], "UNLINKED")
        self.assertEqual(receipt["result"]["reason_code"], "MISSING_RUN_CONTEXT")


if __name__ == "__main__":
    unittest.main(verbosity=2)
