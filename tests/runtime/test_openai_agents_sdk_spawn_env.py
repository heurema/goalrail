"""Tests for ``_build_openai_agents_sdk_spawn_env`` in ``goalrail/runtime/workflow.py``."""

from __future__ import annotations

from pathlib import Path

import pytest
import yaml as _yaml

from goalrail.runtime.workflow import _build_openai_agents_sdk_spawn_env, _load_global_auth
from goalrail.spec.types import AgentSpec, ApiKeyAuth, ExecutorSpec, LLMConfig


@pytest.fixture(autouse=True)
def _isolate_global_config(monkeypatch: pytest.MonkeyPatch, tmp_path: Path) -> None:
    """Keep tests independent from the developer's real global config."""
    monkeypatch.setenv("GOALRAIL_CONFIG_HOME", str(tmp_path))


def _make_spec(
    *,
    model: str | None = "openai/gpt-5-4-mini",
    use_responses: bool | None = None,
    auth: ApiKeyAuth | None = None,
) -> AgentSpec:
    """Build a minimal openai-agents :class:`AgentSpec` for spawn-env tests."""
    config: dict[str, object] = {"harness": "openai-agents"}
    if model is not None:
        config["model"] = model
    if use_responses is not None:
        config["use_responses"] = use_responses
    return AgentSpec(
        spec_version=1,
        name="test-openai-agents",
        instructions="You are a test agent.",
        executor=ExecutorSpec(type="goalrail", config=config, model=model, auth=auth),
        llm=LLMConfig(model=model) if model is not None else None,
    )


def test_model_threads_into_env_var() -> None:
    env = _build_openai_agents_sdk_spawn_env(_make_spec(model="openai/gpt-5-4-mini"))
    assert env["HARNESS_OPENAI_AGENTS_MODEL"] == "openai/gpt-5-4-mini"


def test_api_key_auth_sets_api_key_env_var() -> None:
    env = _build_openai_agents_sdk_spawn_env(
        _make_spec(model="gpt-4o", auth=ApiKeyAuth(api_key="sk-test-456"))
    )
    assert env["HARNESS_OPENAI_AGENTS_API_KEY"] == "sk-test-456"


def test_api_key_auth_base_url_sets_gateway_base_url_env_var() -> None:
    env = _build_openai_agents_sdk_spawn_env(
        _make_spec(
            model="gpt-4o",
            auth=ApiKeyAuth(api_key="sk-test-789", base_url="https://my-gw.example.com/v1"),
        )
    )
    assert env["HARNESS_OPENAI_AGENTS_API_KEY"] == "sk-test-789"
    assert env["HARNESS_OPENAI_AGENTS_GATEWAY_BASE_URL"] == "https://my-gw.example.com/v1"


def test_use_responses_threads_into_env_var() -> None:
    env = _build_openai_agents_sdk_spawn_env(_make_spec(use_responses=False))
    assert env["HARNESS_OPENAI_AGENTS_USE_RESPONSES"] == "false"


def test_spec_auth_takes_precedence_over_global_config(
    monkeypatch: pytest.MonkeyPatch,
    tmp_path: Path,
) -> None:
    cfg_path = tmp_path / "config.yaml"
    cfg_path.write_text(_yaml.dump({"auth": {"type": "api_key", "api_key": "sk-global"}}))
    monkeypatch.setenv("GOALRAIL_CONFIG_HOME", str(tmp_path))

    env = _build_openai_agents_sdk_spawn_env(
        _make_spec(model="openai/gpt-5-4-mini", auth=ApiKeyAuth(api_key="sk-spec"))
    )

    assert env["HARNESS_OPENAI_AGENTS_API_KEY"] == "sk-spec"


def test_global_config_auth_used_when_spec_auth_absent(
    monkeypatch: pytest.MonkeyPatch,
    tmp_path: Path,
) -> None:
    cfg_path = tmp_path / "config.yaml"
    cfg_path.write_text(_yaml.dump({"auth": {"type": "api_key", "api_key": "sk-global"}}))
    monkeypatch.setenv("GOALRAIL_CONFIG_HOME", str(tmp_path))

    env = _build_openai_agents_sdk_spawn_env(_make_spec(auth=None))

    assert env["HARNESS_OPENAI_AGENTS_API_KEY"] == "sk-global"


def test_load_global_auth_api_key(monkeypatch: pytest.MonkeyPatch, tmp_path: Path) -> None:
    monkeypatch.setenv("MY_GLOBAL_KEY", "sk-global-999")
    cfg_path = tmp_path / "config.yaml"
    cfg_path.write_text(_yaml.dump({"auth": {"type": "api_key", "api_key": "$MY_GLOBAL_KEY"}}))
    monkeypatch.setenv("GOALRAIL_CONFIG_HOME", str(tmp_path))

    result = _load_global_auth()

    assert isinstance(result, ApiKeyAuth)
    assert result.api_key == "sk-global-999"


def test_load_global_auth_missing_file(monkeypatch: pytest.MonkeyPatch, tmp_path: Path) -> None:
    monkeypatch.setenv("GOALRAIL_CONFIG_HOME", str(tmp_path))
    assert _load_global_auth() is None


def test_load_global_auth_unresolved_env_var_raises(
    monkeypatch: pytest.MonkeyPatch,
    tmp_path: Path,
) -> None:
    from goalrail.errors import GoalrailError

    monkeypatch.delenv("MISSING_KEY", raising=False)
    cfg_path = tmp_path / "config.yaml"
    cfg_path.write_text(_yaml.dump({"auth": {"type": "api_key", "api_key": "$MISSING_KEY"}}))
    monkeypatch.setenv("GOALRAIL_CONFIG_HOME", str(tmp_path))

    with pytest.raises(GoalrailError):
        _load_global_auth()
