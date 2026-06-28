"""Tests for ``_build_claude_sdk_spawn_env`` in ``goalrail/runtime/workflow.py``."""

from __future__ import annotations

from pathlib import Path

import pytest
import yaml as _yaml

from goalrail.runtime.workflow import _build_claude_sdk_spawn_env
from goalrail.spec.types import AgentSpec, ApiKeyAuth, ExecutorSpec, LLMConfig


@pytest.fixture(autouse=True)
def _isolate_global_config(monkeypatch: pytest.MonkeyPatch, tmp_path: Path) -> None:
    """Keep tests independent from the developer's real global config."""
    monkeypatch.setenv("GOALRAIL_CONFIG_HOME", str(tmp_path))


def _make_spec(
    *,
    model: str | None = "anthropic/claude-sonnet-4-6",
    auth: ApiKeyAuth | None = None,
) -> AgentSpec:
    """Build a minimal claude-sdk :class:`AgentSpec` for spawn-env tests."""
    config: dict[str, object] = {"harness": "claude-sdk"}
    if model is not None:
        config["model"] = model
    return AgentSpec(
        spec_version=1,
        name="test-claude-sdk",
        instructions="You are a test agent.",
        executor=ExecutorSpec(type="goalrail", config=config, model=model, auth=auth),
        llm=LLMConfig(model=model) if model is not None else None,
    )


def test_model_threads_into_env_var() -> None:
    env = _build_claude_sdk_spawn_env(_make_spec(model="anthropic/claude-opus-4-8"))
    assert env["HARNESS_CLAUDE_SDK_MODEL"] == "anthropic/claude-opus-4-8"


def test_api_key_auth_sets_helper_env_var() -> None:
    spec = _make_spec(model=None, auth=ApiKeyAuth(api_key="sk-ant-test-123"))
    env = _build_claude_sdk_spawn_env(spec, workdir=None)

    assert "HARNESS_CLAUDE_SDK_API_KEY_HELPER" in env
    assert "sk-ant-test-123" in env["HARNESS_CLAUDE_SDK_API_KEY_HELPER"]
    assert "HARNESS_CLAUDE_SDK_GATEWAY" not in env


def test_api_key_auth_with_special_chars_is_shell_safe() -> None:
    spec = _make_spec(model=None, auth=ApiKeyAuth(api_key="sk-$weird 'key'"))
    env = _build_claude_sdk_spawn_env(spec, workdir=None)

    helper = env["HARNESS_CLAUDE_SDK_API_KEY_HELPER"]
    assert "sk-$weird 'key'" not in helper
    assert "sk-" in helper


def test_api_key_auth_with_base_url_enables_gateway() -> None:
    spec = _make_spec(
        model="anthropic/claude-sonnet-4-6",
        auth=ApiKeyAuth(api_key="sk-ant-test", base_url="https://gw.example.com/v1"),
    )
    env = _build_claude_sdk_spawn_env(spec, workdir=None)

    assert env["HARNESS_CLAUDE_SDK_GATEWAY"] == "true"
    assert env["HARNESS_CLAUDE_SDK_GATEWAY_BASE_URL"] == "https://gw.example.com/v1"
    assert "HARNESS_CLAUDE_SDK_GATEWAY_AUTH_COMMAND" in env


def test_global_api_key_auth_applied_when_spec_has_no_auth(
    monkeypatch: pytest.MonkeyPatch,
    tmp_path: Path,
) -> None:
    cfg_path = tmp_path / "config.yaml"
    cfg_path.write_text(_yaml.dump({"auth": {"type": "api_key", "api_key": "sk-global"}}))
    monkeypatch.setenv("GOALRAIL_CONFIG_HOME", str(tmp_path))

    env = _build_claude_sdk_spawn_env(_make_spec(auth=None), workdir=None)

    assert "sk-global" in env["HARNESS_CLAUDE_SDK_API_KEY_HELPER"]


def test_spec_auth_takes_precedence_over_global_config(
    monkeypatch: pytest.MonkeyPatch,
    tmp_path: Path,
) -> None:
    cfg_path = tmp_path / "config.yaml"
    cfg_path.write_text(_yaml.dump({"auth": {"type": "api_key", "api_key": "sk-global"}}))
    monkeypatch.setenv("GOALRAIL_CONFIG_HOME", str(tmp_path))

    env = _build_claude_sdk_spawn_env(
        _make_spec(auth=ApiKeyAuth(api_key="sk-spec")),
        workdir=None,
    )

    assert "sk-spec" in env["HARNESS_CLAUDE_SDK_API_KEY_HELPER"]
    assert "sk-global" not in env["HARNESS_CLAUDE_SDK_API_KEY_HELPER"]


def test_workdir_threads_bundle_dir(tmp_path: Path) -> None:
    env = _build_claude_sdk_spawn_env(_make_spec(), workdir=tmp_path)
    assert env["HARNESS_CLAUDE_SDK_BUNDLE_DIR"] == str(tmp_path)
