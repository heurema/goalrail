#!/usr/bin/env python3
"""Deterministic tests for the local Codex correlation probe."""

from __future__ import annotations

from concurrent.futures import ThreadPoolExecutor
import copy
import json
import os
from pathlib import Path
import subprocess
import sys
import unittest
from typing import Any


PROBE_DIR = Path(__file__).resolve().parent
TESTDATA_DIR = PROBE_DIR / "testdata"
MISSING_CONTEXT = object()


def load_fixture(name: str) -> dict[str, Any]:
    with (TESTDATA_DIR / name).open(encoding="utf-8") as fixture_file:
        value = json.load(fixture_file)
    if not isinstance(value, dict):
        raise AssertionError(f"{name} must contain a JSON object")
    return value


def run_probe(
    hook_input: dict[str, Any],
    run_context: dict[str, Any] | object = MISSING_CONTEXT,
) -> tuple[int, dict[str, Any]]:
    environment = os.environ.copy()
    if run_context is MISSING_CONTEXT:
        environment.pop("GOALRAIL_RUN_CONTEXT", None)
    else:
        environment["GOALRAIL_RUN_CONTEXT"] = json.dumps(
            run_context, sort_keys=True, separators=(",", ":")
        )

    completed = subprocess.run(
        [sys.executable, str(PROBE_DIR / "probe.py")],
        input=json.dumps(hook_input),
        text=True,
        capture_output=True,
        env=environment,
        check=False,
    )
    if completed.stderr:
        raise AssertionError(f"probe wrote to stderr: {completed.stderr}")
    result = json.loads(completed.stdout)
    if not isinstance(result, dict):
        raise AssertionError(f"probe output must be an object: {result!r}")
    return completed.returncode, result


class CorrelationProbeTests(unittest.TestCase):
    def setUp(self) -> None:
        self.hook_input = load_fixture("session-start.json")
        self.run_context = load_fixture("run-context.json")

    def assert_verified(
        self,
        hook_input: dict[str, Any],
        run_context: dict[str, Any],
        *,
        expected_matcher: str,
    ) -> dict[str, Any]:
        return_code, result = run_probe(hook_input, run_context)
        self.assertEqual(return_code, 0)
        self.assertEqual(result["status"], "VERIFIED")
        self.assertEqual(
            result["lineage"],
            {
                "change_id": run_context["change_id"],
                "run_id": run_context["run_id"],
                "root_session_id": hook_input["session_id"],
            },
        )
        self.assertEqual(result["evidence"]["hook_matcher"], expected_matcher)
        self.assertEqual(len(result["evidence"]["context_sha256"]), 64)
        return result

    def assert_unlinked(
        self,
        hook_input: dict[str, Any],
        run_context: dict[str, Any] | object,
        *,
        reason_code: str,
    ) -> None:
        return_code, result = run_probe(hook_input, run_context)
        self.assertEqual(return_code, 2)
        self.assertEqual(result["status"], "UNLINKED")
        self.assertEqual(result["reason_code"], reason_code)
        self.assertNotIn("lineage", result)

    def test_session_startup_links_to_root_session(self) -> None:
        self.assert_verified(
            self.hook_input,
            self.run_context,
            expected_matcher="startup",
        )

    def test_session_resume_preserves_root_session(self) -> None:
        hook_input = copy.deepcopy(self.hook_input)
        hook_input["source"] = "resume"
        self.assert_verified(
            hook_input,
            self.run_context,
            expected_matcher="resume",
        )

    def test_session_compaction_preserves_root_session(self) -> None:
        hook_input = copy.deepcopy(self.hook_input)
        hook_input["source"] = "compact"
        self.assert_verified(
            hook_input,
            self.run_context,
            expected_matcher="compact",
        )

    def test_subagent_inherits_parent_session(self) -> None:
        hook_input = copy.deepcopy(self.hook_input)
        hook_input.pop("source")
        hook_input.update(
            {
                "hook_event_name": "SubagentStart",
                "turn_id": "turn-probe-001",
                "agent_id": "agent-probe-001",
                "agent_type": "worker",
            }
        )
        self.assert_verified(
            hook_input,
            self.run_context,
            expected_matcher="worker",
        )

    def test_absent_context_is_unlinked(self) -> None:
        self.assert_unlinked(
            self.hook_input,
            MISSING_CONTEXT,
            reason_code="MISSING_RUN_CONTEXT",
        )

    def test_conflicting_repo_context_is_unlinked(self) -> None:
        run_context = copy.deepcopy(self.run_context)
        run_context["repo_root"] = "/workspace/another-project"
        self.assert_unlinked(
            self.hook_input,
            run_context,
            reason_code="CONTEXT_CONFLICT",
        )

    def test_concurrent_changes_do_not_cross_link(self) -> None:
        second_hook = copy.deepcopy(self.hook_input)
        second_hook["session_id"] = "01900000-0000-7000-8000-000000000002"
        second_context = copy.deepcopy(self.run_context)
        second_context.update(
            {
                "change_id": "second-change",
                "run_id": "run-probe-002",
            }
        )

        cases = (
            (self.hook_input, self.run_context),
            (second_hook, second_context),
        )
        with ThreadPoolExecutor(max_workers=2) as executor:
            results = list(executor.map(lambda case: run_probe(*case), cases))

        self.assertEqual([return_code for return_code, _ in results], [0, 0])
        observed = {
            (
                result["lineage"]["change_id"],
                result["lineage"]["run_id"],
                result["lineage"]["root_session_id"],
            )
            for _, result in results
        }
        self.assertEqual(
            observed,
            {
                (
                    "intent-canary-v0",
                    "run-probe-001",
                    "01900000-0000-7000-8000-000000000001",
                ),
                (
                    "second-change",
                    "run-probe-002",
                    "01900000-0000-7000-8000-000000000002",
                ),
            },
        )
        self.assertNotEqual(
            results[0][1]["evidence"]["context_sha256"],
            results[1][1]["evidence"]["context_sha256"],
        )


if __name__ == "__main__":
    unittest.main(verbosity=2)
