#!/usr/bin/env python3
"""Evaluate a synthetic Codex hook input against process-scoped run context."""

from __future__ import annotations

import hashlib
import json
import os
import re
import sys
from pathlib import PurePath
from typing import Any, NoReturn


CONTEXT_ENV = "GOALRAIL_RUN_CONTEXT"
ID_PATTERN = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$")
REQUIRED_CONTEXT_KEYS = {
    "schema_version",
    "change_id",
    "run_id",
    "repo_root",
}
SESSION_START_SOURCES = {"startup", "resume", "clear", "compact"}


def emit(payload: dict[str, Any]) -> None:
    json.dump(payload, sys.stdout, sort_keys=True, separators=(",", ":"))
    sys.stdout.write("\n")


def unlink(reason_code: str, message: str) -> NoReturn:
    emit(
        {
            "schema_version": 1,
            "status": "UNLINKED",
            "reason_code": reason_code,
            "message": message,
        }
    )
    raise SystemExit(2)


def load_json_object(raw: str, *, reason_code: str, label: str) -> dict[str, Any]:
    try:
        value = json.loads(raw)
    except json.JSONDecodeError:
        unlink(reason_code, f"{label} is not valid JSON")
    if not isinstance(value, dict):
        unlink(reason_code, f"{label} must be a JSON object")
    return value


def require_nonempty_string(
    value: dict[str, Any], key: str, *, reason_code: str, label: str
) -> str:
    candidate = value.get(key)
    if not isinstance(candidate, str) or not candidate.strip():
        unlink(reason_code, f"{label}.{key} must be a non-empty string")
    return candidate


def require_id(value: dict[str, Any], key: str) -> str:
    candidate = require_nonempty_string(
        value,
        key,
        reason_code="INVALID_RUN_CONTEXT",
        label="run_context",
    )
    if not ID_PATTERN.fullmatch(candidate):
        unlink(
            "INVALID_RUN_CONTEXT",
            f"run_context.{key} contains unsupported characters or is too long",
        )
    return candidate


def normalized_path(raw: str, *, label: str) -> str:
    path = PurePath(raw)
    if not path.is_absolute():
        unlink("INVALID_PATH_CONTEXT", f"{label} must be an absolute path")
    return os.path.normpath(raw)


def is_within_repo(cwd: str, repo_root: str) -> bool:
    try:
        return os.path.commonpath((cwd, repo_root)) == repo_root
    except ValueError:
        return False


def validate_hook_event(hook_input: dict[str, Any], hook_event_name: str) -> str:
    if hook_event_name == "SessionStart":
        source = require_nonempty_string(
            hook_input,
            "source",
            reason_code="INVALID_HOOK_INPUT",
            label="hook_input",
        )
        if source not in SESSION_START_SOURCES:
            unlink(
                "INVALID_HOOK_INPUT",
                "hook_input.source is not a supported SessionStart source",
            )
        return source

    if hook_event_name == "SubagentStart":
        require_nonempty_string(
            hook_input,
            "turn_id",
            reason_code="INVALID_HOOK_INPUT",
            label="hook_input",
        )
        require_nonempty_string(
            hook_input,
            "agent_id",
            reason_code="INVALID_HOOK_INPUT",
            label="hook_input",
        )
        return require_nonempty_string(
            hook_input,
            "agent_type",
            reason_code="INVALID_HOOK_INPUT",
            label="hook_input",
        )

    unlink(
        "UNSUPPORTED_HOOK_EVENT",
        "hook_input.hook_event_name must be SessionStart or SubagentStart",
    )


def main() -> None:
    raw_context = os.environ.get(CONTEXT_ENV)
    if raw_context is None:
        unlink("MISSING_RUN_CONTEXT", f"{CONTEXT_ENV} is not set")

    run_context = load_json_object(
        raw_context,
        reason_code="INVALID_RUN_CONTEXT",
        label="run_context",
    )
    if set(run_context) != REQUIRED_CONTEXT_KEYS:
        unlink(
            "INVALID_RUN_CONTEXT",
            "run_context must contain exactly schema_version, change_id, run_id, and repo_root",
        )
    if run_context["schema_version"] != 1:
        unlink("UNSUPPORTED_CONTEXT_VERSION", "run_context.schema_version must equal 1")

    try:
        raw_hook = sys.stdin.read()
    except OSError:
        unlink("HOOK_INPUT_READ_FAILED", "Codex hook input could not be read")
    hook_input = load_json_object(
        raw_hook,
        reason_code="INVALID_HOOK_INPUT",
        label="hook_input",
    )

    change_id = require_id(run_context, "change_id")
    run_id = require_id(run_context, "run_id")
    repo_root = normalized_path(
        require_nonempty_string(
            run_context,
            "repo_root",
            reason_code="INVALID_RUN_CONTEXT",
            label="run_context",
        ),
        label="run_context.repo_root",
    )
    session_id = require_nonempty_string(
        hook_input,
        "session_id",
        reason_code="INVALID_HOOK_INPUT",
        label="hook_input",
    )
    cwd = normalized_path(
        require_nonempty_string(
            hook_input,
            "cwd",
            reason_code="INVALID_HOOK_INPUT",
            label="hook_input",
        ),
        label="hook_input.cwd",
    )
    hook_event_name = require_nonempty_string(
        hook_input,
        "hook_event_name",
        reason_code="INVALID_HOOK_INPUT",
        label="hook_input",
    )
    hook_matcher = validate_hook_event(hook_input, hook_event_name)

    if not is_within_repo(cwd, repo_root):
        unlink(
            "CONTEXT_CONFLICT",
            "hook_input.cwd is outside run_context.repo_root",
        )

    canonical_context = json.dumps(
        run_context, sort_keys=True, separators=(",", ":")
    ).encode("utf-8")
    emit(
        {
            "schema_version": 1,
            "status": "VERIFIED",
            "lineage": {
                "change_id": change_id,
                "run_id": run_id,
                "root_session_id": session_id,
            },
            "evidence": {
                "context_sha256": hashlib.sha256(canonical_context).hexdigest(),
                "cwd": cwd,
                "hook_event_name": hook_event_name,
                "hook_matcher": hook_matcher,
            },
        }
    )


if __name__ == "__main__":
    main()
