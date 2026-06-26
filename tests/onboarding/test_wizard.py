"""Tests for :mod:`omnigent.onboarding.wizard`."""

from __future__ import annotations

from unittest.mock import Mock

import pytest

from omnigent.onboarding import wizard as wizard_mod


def test_agents_dir_uses_goalrail_config_home(
    monkeypatch: pytest.MonkeyPatch,
    tmp_path,
) -> None:
    """Wizard-generated agents live beside the effective user config."""
    config_home = tmp_path / "goalrail-config"
    monkeypatch.setenv("GOALRAIL_CONFIG_HOME", str(config_home))
    monkeypatch.delenv("OMNIGENT_CONFIG_HOME", raising=False)

    assert wizard_mod._agents_dir() == config_home / "agents"


def test_save_yaml_uses_goalrail_config_home(
    monkeypatch: pytest.MonkeyPatch,
    tmp_path,
) -> None:
    """Wizard YAML generation writes under the effective config home."""
    config_home = tmp_path / "goalrail-config"
    legacy_home = tmp_path / "home"
    monkeypatch.setenv("GOALRAIL_CONFIG_HOME", str(config_home))
    monkeypatch.delenv("OMNIGENT_CONFIG_HOME", raising=False)
    monkeypatch.setenv("HOME", str(legacy_home))

    path = wizard_mod._save_yaml("name: test\n", "agent.yaml")

    assert path == config_home / "agents" / "agent.yaml"
    assert path.read_text(encoding="utf-8") == "name: test\n"
    assert not (legacy_home / ".omnigent" / "agents").exists()


def test_global_auth_prompt_uses_goalrail_product_name(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """The interactive setup prompt should use the public Goalrail name."""
    console = Mock()
    prompt_values = iter(["sk-test", ""])

    monkeypatch.setattr(wizard_mod, "console", console)
    monkeypatch.setattr(wizard_mod, "_list_databricks_profiles", list)
    monkeypatch.setattr(wizard_mod, "_arrow_menu", lambda options: 0)
    monkeypatch.setattr(
        wizard_mod,
        "_text_prompt",
        lambda *args, **kwargs: next(prompt_values),
    )

    auth, _ = wizard_mod._prompt_global_auth()

    printed = " ".join(str(call.args[0]) for call in console.print.call_args_list if call.args)
    assert auth == {"type": "api_key", "api_key": "sk-test"}
    assert "How will Goalrail authenticate with the LLM?" in printed
    assert "omnigent authenticate" not in printed
