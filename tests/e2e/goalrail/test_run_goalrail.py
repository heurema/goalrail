"""Phase 5 integration-code tests -- ``goalrail run`` shim (mock LLM).

Migrated to mock LLM: uses canned responses so the tests are
deterministic and need no real credentials.

**What breaks if this fails:**
- The ``run`` dispatch site regresses.
- The shim's YAML preparation pipeline breaks silently.
- The in-process goalrail app fails to answer.
- The output extraction regresses.
- ``GOALRAIL_RUNTIME=1`` stops being honored.
- ``goalrail version`` diverges.
"""

from __future__ import annotations

import os
import subprocess
from pathlib import Path
from typing import Any

from tests.e2e.conftest import configure_mock_llm, reset_mock_llm

_MODEL = "mock-run-goalrail-model"

_HARNESS = "openai-agents"

_PROMPT = "say hi in 5 words"

_MIN_ASSISTANT_CHARS = 4

_RUN_TIMEOUT_SEC = 60


def _run_goalrail_run_goalrail(
    *,
    goalrail_python: Path,
    goalrail_repo_root: Path,
    mock_credentials_env: dict[str, str],
    extra_env: dict[str, str] | None = None,
) -> subprocess.CompletedProcess[str]:
    """Execute ``goalrail run <hello_world.yaml> ... -p <prompt>``."""
    yaml_path = goalrail_repo_root / "tests" / "resources" / "examples" / "hello_world.yaml"
    argv: list[str] = [
        str(goalrail_python),
        "-m",
        "goalrail",
        "run",
        str(yaml_path),
        "--model",
        _MODEL,
        "--harness",
        _HARNESS,
        "-p",
        _PROMPT,
        "--no-log",
        "--no-session",
    ]
    env = dict(mock_credentials_env)
    if extra_env is not None:
        env.update(extra_env)
    return subprocess.run(
        argv,
        env=env,
        cwd=str(goalrail_repo_root),
        capture_output=True,
        text=True,
        timeout=_RUN_TIMEOUT_SEC,
    )


def _structural_observations(
    result: subprocess.CompletedProcess[str],
) -> dict[str, Any]:
    """Distill structural properties of an ``goalrail run`` result."""
    text = result.stdout.strip()
    return {
        "exit_code": result.returncode,
        "assistant_text_nonempty": bool(text),
        "assistant_text_meets_min_length": len(text) >= _MIN_ASSISTANT_CHARS,
    }


def test_run_goalrail_smoke(
    goalrail_python: Path,
    goalrail_repo_root: Path,
    mock_credentials_env: dict[str, str],
    mock_llm_server_url: str,
) -> None:
    """
    ``goalrail run hello_world.yaml -p <prompt>`` exits 0,
    prints non-trivial assistant text, and does not re-emit the
    pre-phase-5 hard-error on stderr.
    """
    reset_mock_llm(mock_llm_server_url)
    configure_mock_llm(
        mock_llm_server_url,
        [{"text": "Hello there nice to meet!"}],
        key=_MODEL,
    )

    result = _run_goalrail_run_goalrail(
        goalrail_python=goalrail_python,
        goalrail_repo_root=goalrail_repo_root,
        mock_credentials_env=mock_credentials_env,
    )
    assert result.returncode == 0, (
        f"expected exit 0, got {result.returncode}.\n"
        f"stdout:\n{result.stdout!r}\n\nstderr:\n{result.stderr!r}"
    )
    assistant_text = result.stdout.strip()
    assert len(assistant_text) >= _MIN_ASSISTANT_CHARS, (
        f"assistant text shorter than {_MIN_ASSISTANT_CHARS} chars; got {assistant_text!r}"
    )
    assert "phase 5" not in result.stderr, (
        f"Regression: stderr contains the pre-phase-5 hard-error wording. stderr={result.stderr!r}"
    )


def test_run_goalrail_matches_structural_fields(
    goalrail_python: Path,
    goalrail_repo_root: Path,
    mock_credentials_env: dict[str, str],
    mock_llm_server_url: str,
) -> None:
    """
    Two successive runs agree on structural fields -- proves
    structural stability.
    """
    reset_mock_llm(mock_llm_server_url)
    configure_mock_llm(
        mock_llm_server_url,
        [
            {"text": "Hello there nice to meet!"},
            {"text": "Hello there nice to meet!"},
        ],
        key=_MODEL,
    )

    first = _run_goalrail_run_goalrail(
        goalrail_python=goalrail_python,
        goalrail_repo_root=goalrail_repo_root,
        mock_credentials_env=mock_credentials_env,
    )
    second = _run_goalrail_run_goalrail(
        goalrail_python=goalrail_python,
        goalrail_repo_root=goalrail_repo_root,
        mock_credentials_env=mock_credentials_env,
    )
    first_obs = _structural_observations(first)
    second_obs = _structural_observations(second)
    assert first_obs == second_obs, (
        "Structural observations diverge between runs:\n"
        f"first={first_obs!r}\n"
        f"second={second_obs!r}\n\n"
        f"first stdout: {first.stdout!r}\n"
        f"second stdout: {second.stdout!r}"
    )


def test_run_goalrail_env_var_enables_integration(
    goalrail_python: Path,
    goalrail_repo_root: Path,
    mock_credentials_env: dict[str, str],
    mock_llm_server_url: str,
) -> None:
    """
    ``GOALRAIL_RUNTIME=1`` (with no flag on argv) must route
    through the goalrail shim.
    """
    reset_mock_llm(mock_llm_server_url)
    configure_mock_llm(
        mock_llm_server_url,
        [{"text": "Hello from GOALRAIL_RUNTIME path!"}],
        key=_MODEL,
    )

    result = _run_goalrail_run_goalrail(
        goalrail_python=goalrail_python,
        goalrail_repo_root=goalrail_repo_root,
        mock_credentials_env=mock_credentials_env,
        extra_env={"GOALRAIL_RUNTIME": "1"},
    )
    assert result.returncode == 0, (
        f"GOALRAIL_RUNTIME=1 did not yield exit 0; "
        f"got {result.returncode}.\n"
        f"stdout:\n{result.stdout!r}\n\nstderr:\n{result.stderr!r}"
    )
    assistant_text = result.stdout.strip()
    assert len(assistant_text) >= _MIN_ASSISTANT_CHARS, (
        f"GOALRAIL_RUNTIME=1 assistant text shorter than "
        f"{_MIN_ASSISTANT_CHARS} chars; got {assistant_text!r}"
    )
    assert "phase 5" not in result.stderr, (
        f"GOALRAIL_RUNTIME=1 fell back to the pre-phase-5 hard error. stderr={result.stderr!r}"
    )


def test_version_goalrail_matches_version(
    goalrail_python: Path,
    goalrail_repo_root: Path,
) -> None:
    """
    ``goalrail version`` must be stable and independent of
    GOALRAIL_RUNTIME. No LLM credentials needed.
    """
    baseline = subprocess.run(
        [
            str(goalrail_python),
            "-m",
            "goalrail",
            "version",
        ],
        env={k: v for k, v in os.environ.items() if k != "GOALRAIL_RUNTIME"},
        cwd=str(goalrail_repo_root),
        capture_output=True,
        text=True,
        timeout=_RUN_TIMEOUT_SEC,
    )
    with_ap = subprocess.run(
        [
            str(goalrail_python),
            "-m",
            "goalrail",
            "version",
        ],
        env={**os.environ, "GOALRAIL_RUNTIME": "1"},
        cwd=str(goalrail_repo_root),
        capture_output=True,
        text=True,
        timeout=_RUN_TIMEOUT_SEC,
    )
    assert baseline.returncode == 0
    assert with_ap.returncode == 0
    assert baseline.stdout == with_ap.stdout, (
        "goalrail version diverged between baseline and GOALRAIL_RUNTIME=1. "
        f"baseline={baseline.stdout!r} ap={with_ap.stdout!r}"
    )
    version_text = baseline.stdout.strip()
    assert version_text, "goalrail version printed no stdout"
    assert version_text.startswith("goalrail "), f"unexpected version output: {baseline.stdout!r}"
    after_prefix = version_text[len("goalrail ") :]
    assert after_prefix and after_prefix[0].isdigit(), (
        f"unexpected version output: {baseline.stdout!r}"
    )
