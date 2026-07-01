"""Unit tests for opencode-native provider-config synthesis."""

from __future__ import annotations

import json
import stat
from pathlib import Path

from goalrail.opencode_native_provider import (
    build_opencode_goalrail_mcp_server,
    build_opencode_model_default_config,
    write_opencode_provider_config,
)


def test_build_goalrail_mcp_server_points_serve_mcp_at_bridge_dir() -> None:
    block = build_opencode_goalrail_mcp_server(Path("/tmp/bridge-xyz"))
    assert set(block) == {"goalrail"}
    entry = block["goalrail"]
    assert entry["type"] == "local"
    assert entry["enabled"] is True
    cmd = entry["command"]
    # Launches the SHARED serve-mcp relay, pointed at THIS bridge dir.
    assert cmd[-3:] == ["serve-mcp", "--bridge-dir", "/tmp/bridge-xyz"]
    assert "goalrail.claude_native_bridge" in cmd
    assert entry.get("environment", {}).get("PYTHONUNBUFFERED") == "1"


def test_build_goalrail_mcp_server_honors_python_executable() -> None:
    block = build_opencode_goalrail_mcp_server(Path("/tmp/b"), python_executable="/custom/python")
    assert block["goalrail"]["command"][0] == "/custom/python"


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


def test_build_mcp_block_stdio_and_http() -> None:
    from types import SimpleNamespace as N

    from goalrail.opencode_native_provider import build_opencode_mcp_block

    servers = [
        N(
            name="gh",
            transport="stdio",
            command="npx",
            args=["-y", "server-github"],
            env={"GITHUB_TOKEN": "x"},
            url=None,
            headers={},
        ),
        N(
            name="remote",
            transport="http",
            url="https://mcp.example/sse",
            headers={"X-Key": "k"},
            command=None,
            args=[],
            env={},
        ),
        # Unrepresentable (stdio without a command) → skipped.
        N(name="bad", transport="stdio", command=None, args=[], env={}, url=None, headers={}),
    ]
    block = build_opencode_mcp_block(servers)
    assert set(block) == {"gh", "remote"}
    assert block["gh"] == {
        "type": "local",
        "command": ["npx", "-y", "server-github"],
        "enabled": True,
        "environment": {"GITHUB_TOKEN": "x"},
    }
    assert block["remote"] == {
        "type": "remote",
        "url": "https://mcp.example/sse",
        "enabled": True,
        "headers": {"X-Key": "k"},
    }
