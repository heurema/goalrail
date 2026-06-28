"""Unit tests for ``goalrail/model_catalog.py`` provider enumeration."""

from __future__ import annotations

from collections.abc import Iterator
from pathlib import Path

import httpx
import pytest
import yaml

import goalrail.model_catalog as model_catalog
from goalrail.model_catalog import catalog_for_spec, list_models_for_worker, resolve_model_provider
from goalrail.spec.types import AgentSpec, ApiKeyAuth, ExecutorSpec


@pytest.fixture(autouse=True)
def _fresh_catalog_cache() -> Iterator[None]:
    """Reset provider-listing cache between tests."""
    model_catalog.clear_model_catalog_cache()
    yield
    model_catalog.clear_model_catalog_cache()


@pytest.fixture(autouse=True)
def _isolated_config(monkeypatch: pytest.MonkeyPatch, tmp_path: Path) -> Path:
    """Point provider config reads at an isolated temp directory."""
    monkeypatch.setenv("GOALRAIL_CONFIG_HOME", str(tmp_path))
    monkeypatch.setenv("GOALRAIL_DISABLE_KEYRING", "1")
    monkeypatch.setattr("goalrail.onboarding.detected.detect_providers", list)
    return tmp_path


def _write_config(config_home: Path, config: dict[str, object]) -> None:
    (config_home / "config.yaml").write_text(yaml.safe_dump(config))


def _worker_spec(
    harness: str,
    *,
    model: str | None = None,
    auth: ApiKeyAuth | None = None,
    sub_agents: list[AgentSpec] | None = None,
) -> AgentSpec:
    config: dict[str, object] = {"harness": harness}
    if model is not None:
        config["model"] = model
    return AgentSpec(
        spec_version=1,
        name="worker",
        instructions="test",
        executor=ExecutorSpec(type="goalrail", config=config, model=model, auth=auth),
        sub_agents=sub_agents or [],
    )


def _models_transport(
    *,
    expected_url: str,
    expected_header: tuple[str, str],
    data: list[dict[str, object]],
    seen: list[httpx.Request] | None = None,
) -> httpx.MockTransport:
    def handler(request: httpx.Request) -> httpx.Response:
        if seen is not None:
            seen.append(request)
        assert str(request.url) == expected_url
        key, value = expected_header
        assert request.headers[key] == value
        return httpx.Response(200, json={"data": data})

    return httpx.MockTransport(handler)


def test_resolve_gateway_default_for_codex(
    _isolated_config: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    """A default OpenAI-family gateway resolves for Codex workers."""
    monkeypatch.setenv("OPENROUTER_API_KEY", "sk-or-test")
    _write_config(
        _isolated_config,
        {
            "providers": {
                "openrouter": {
                    "kind": "gateway",
                    "default": True,
                    "openai": {
                        "base_url": "https://openrouter.example/api/v1",
                        "api_key": "$OPENROUTER_API_KEY",
                        "models": {"default": "gpt-5.4"},
                    },
                }
            }
        },
    )

    provider = resolve_model_provider(_worker_spec("codex-native"), "codex-native")

    assert provider.kind == "gateway"
    assert provider.family == "openai"
    assert provider.base_url == "https://openrouter.example/api/v1"
    assert provider.api_key == "sk-or-test"
    assert provider.detail == "provider 'openrouter'"


def test_legacy_openai_agents_api_key_auth_resolves_provider() -> None:
    """OpenAI-Agents still supports direct spec api_key auth fallback."""
    spec = _worker_spec(
        "openai-agents",
        auth=ApiKeyAuth(api_key="sk-test", base_url="https://llm.example/v1"),
    )

    provider = resolve_model_provider(spec, "openai-agents")

    assert provider.kind == "key"
    assert provider.family == "openai"
    assert provider.base_url == "https://llm.example/v1"
    assert provider.api_key == "sk-test"


def test_list_models_fetches_openai_compatible_models(
    _isolated_config: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    """OpenAI-compatible providers are fetched through bearer auth."""
    monkeypatch.setenv("OPENROUTER_API_KEY", "sk-or-test")
    _write_config(
        _isolated_config,
        {
            "providers": {
                "openrouter": {
                    "kind": "gateway",
                    "default": True,
                    "openai": {
                        "base_url": "https://openrouter.example/api/v1",
                        "api_key": "$OPENROUTER_API_KEY",
                    },
                }
            }
        },
    )
    seen: list[httpx.Request] = []
    transport = _models_transport(
        expected_url="https://openrouter.example/api/v1/models",
        expected_header=("authorization", "Bearer sk-or-test"),
        data=[
            {"id": "gpt-5.4", "context_length": 200000},
            {"id": "claude-opus-4-8"},
        ],
        seen=seen,
    )

    listing = list_models_for_worker(
        _worker_spec("codex-native"), "codex-native", transport=transport
    )

    assert len(seen) == 1
    assert listing.source == "openai-compatible"
    assert listing.verified is True
    assert [m.id for m in listing.models] == ["gpt-5.4"]
    assert listing.models[0].context_window == 200000


def test_list_models_fetches_anthropic_models(
    _isolated_config: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    """Anthropic key providers use Anthropic headers and Claude filtering."""
    monkeypatch.setenv("ANTHROPIC_API_KEY", "sk-ant-test")
    _write_config(
        _isolated_config,
        {
            "providers": {
                "anthropic": {
                    "kind": "key",
                    "default": True,
                    "anthropic": {
                        "base_url": "https://api.anthropic.com",
                        "api_key": "$ANTHROPIC_API_KEY",
                    },
                }
            }
        },
    )
    transport = _models_transport(
        expected_url="https://api.anthropic.com/v1/models",
        expected_header=("x-api-key", "sk-ant-test"),
        data=[{"id": "claude-opus-4-8"}, {"id": "gpt-5.4"}],
    )

    listing = list_models_for_worker(
        _worker_spec("claude-native"), "claude-native", transport=transport
    )

    assert listing.source == "anthropic-api"
    assert listing.verified is True
    assert [m.id for m in listing.models] == ["claude-opus-4-8"]


def test_subscription_provider_uses_static_catalog(_isolated_config: Path) -> None:
    """Subscription CLI providers expose a curated static model list."""
    _write_config(
        _isolated_config,
        {
            "providers": {
                "claude-login": {
                    "kind": "subscription",
                    "cli": "claude",
                    "default": True,
                }
            }
        },
    )

    listing = list_models_for_worker(_worker_spec("claude-native"), "claude-native")

    assert listing.source == "static"
    assert listing.verified is False
    assert "claude-opus-4-8" in {m.id for m in listing.models}


def test_no_provider_reports_none() -> None:
    """Workers with no resolved provider return an explanatory empty listing."""
    listing = list_models_for_worker(_worker_spec("codex-native"), "codex-native")

    assert listing.source == "none"
    assert listing.verified is False
    assert listing.models == ()
    assert "no usable model provider" in listing.note


def test_catalog_for_spec_includes_subagents_and_self(
    _isolated_config: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    """Catalog rows are emitted for each sub-agent plus the caller itself."""
    monkeypatch.setenv("OPENROUTER_API_KEY", "sk-or-test")
    _write_config(
        _isolated_config,
        {
            "providers": {
                "openrouter": {
                    "kind": "gateway",
                    "default": True,
                    "openai": {
                        "base_url": "https://openrouter.example/api/v1",
                        "api_key": "$OPENROUTER_API_KEY",
                    },
                }
            }
        },
    )
    child = _worker_spec("codex-native", model="gpt-5.4")
    child.name = "coder"
    parent = _worker_spec("codex-native", sub_agents=[child])
    transport = _models_transport(
        expected_url="https://openrouter.example/api/v1/models",
        expected_header=("authorization", "Bearer sk-or-test"),
        data=[{"id": "gpt-5.4"}],
    )

    catalog = catalog_for_spec(parent, transport=transport)

    assert set(catalog) == {"coder", "self"}
    assert catalog["coder"]["source"] == "openai-compatible"
    assert catalog["self"]["models"] == [{"id": "gpt-5.4", "family": "openai"}]
