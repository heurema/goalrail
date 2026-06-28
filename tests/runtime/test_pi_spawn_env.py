"""
Tests for ``_build_pi_spawn_env`` in ``goalrail/runtime/workflow.py``.

The spawn-env builder maps ``spec.executor`` fields to ``HARNESS_PI_*``
env vars that the pi harness wrap reads at executor-construction time.
Mirrors ``test_claude_sdk_spawn_env.py`` — pi must have the same
direct model/gateway env mapping that claude-sdk has.

This is a unit test — no subprocess spawn, no real pi CLI.
"""

from __future__ import annotations

from pathlib import Path

import pytest

from goalrail.runtime.workflow import _build_pi_spawn_env
from goalrail.spec.types import AgentSpec, ExecutorSpec, LLMConfig


@pytest.fixture(autouse=True)
def _isolate_global_config(monkeypatch: pytest.MonkeyPatch, tmp_path: Path) -> None:
    """
    Point GOALRAIL_CONFIG_HOME at an empty temp dir for every test in
    this file so the developer's real ``~/.goalrail/config.yaml`` cannot
    influence provider resolution under test.

    :param monkeypatch: Pytest monkeypatch fixture.
    :param tmp_path: Temporary directory for the isolated config.
    """
    monkeypatch.setenv("GOALRAIL_CONFIG_HOME", str(tmp_path))


def _make_spec(*, model: str | None = None) -> AgentSpec:
    """
    Build a minimal pi :class:`AgentSpec` for spawn-env tests.

    :param model: Model identifier threaded into executor config and
        ``spec.llm``, e.g. ``"anthropic/claude-sonnet-4-6"``. ``None``
        omits it (no model pinned in YAML — the nessie shape).
    :returns: A populated :class:`AgentSpec`.
    """
    config: dict[str, object] = {"harness": "pi"}
    if model is not None:
        config["model"] = model
    return AgentSpec(
        spec_version=1,
        name="test-pi",
        instructions="You are a test agent.",
        executor=ExecutorSpec(type="goalrail", config=config, model=model),
        llm=LLMConfig(model=model) if model is not None else None,
    )


def test_pi_spawn_env_threads_cwd_separately_from_bundle_dir(tmp_path: Path) -> None:
    """
    Pi gets the session workspace as ``HARNESS_PI_CWD``.

    ``workdir`` is the extracted agent bundle, not the user's project
    workspace. If these are conflated, Pi launches in the wrong repository.
    """
    workspace = tmp_path / "repo"
    workspace.mkdir()
    bundle_dir = tmp_path / "runner-specs" / "ag_pi-v1"
    bundle_dir.mkdir(parents=True)

    env = _build_pi_spawn_env(_make_spec(), cwd=workspace, workdir=bundle_dir)

    assert env["HARNESS_PI_CWD"] == str(workspace)
    assert env["HARNESS_PI_BUNDLE_DIR"] == str(bundle_dir)
