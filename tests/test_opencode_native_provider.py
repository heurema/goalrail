"""Unit tests for opencode-native provider-config synthesis."""

from __future__ import annotations

import json
import stat
from pathlib import Path

from goalrail.opencode_native_provider import (
    build_opencode_model_default_config,
    write_opencode_provider_config,
)


def test_build_model_default_config_pins_model_without_provider_block() -> None:
    cfg = build_opencode_model_default_config("anthropic/claude-sonnet-4-5")
    assert cfg == {
        "$schema": "https://opencode.ai/config.json",
        "model": "anthropic/claude-sonnet-4-5",
    }
    assert "provider" not in cfg


def test_model_default_config_round_trips_through_writer(tmp_path: Path) -> None:
    path = write_opencode_provider_config(
        tmp_path, build_opencode_model_default_config("openai/gpt-5.5")
    )
    written = json.loads(path.read_text(encoding="utf-8"))
    assert written["model"] == "openai/gpt-5.5"


def test_write_provider_config_is_0600_and_valid_json(tmp_path: Path) -> None:
    config = build_opencode_model_default_config("openai/gpt-5.5")
    path = write_opencode_provider_config(tmp_path, config)
    assert path == tmp_path / "opencode" / "opencode.json"
    assert stat.S_IMODE(path.stat().st_mode) == 0o600
    assert json.loads(path.read_text(encoding="utf-8")) == config
