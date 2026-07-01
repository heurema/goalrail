"""Synthesize OpenCode config for the native-server harness."""

from __future__ import annotations

import json
import os
import tempfile
from collections.abc import Mapping, Sequence
from pathlib import Path
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from goalrail.spec.types import MCPServerConfig


def build_opencode_model_default_config(model: str) -> dict[str, object]:
    """
    Build a minimal ``opencode.json`` that only pins the default model.

    Used when the user's own provider auth (``opencode auth login`` /
    provider env keys) already supplies credentials, but a default model has
    been chosen — via ``goalrail opencode --model`` or the ``goalrail setup`` OpenCode
    default — so the per-session TUI (and the first turn) launch on that model
    instead of OpenCode's built-in default (``opencode/big-pickle``). No
    provider block: OpenCode resolves the provider from the model id's prefix
    against its own ``auth.json``.

    :param model: A ``provider/model`` id, e.g. ``"anthropic/claude-sonnet-4-5"``.
    :returns: A config dict ready to serialize to ``opencode.json``.
    """
    return {"$schema": "https://opencode.ai/config.json", "model": model}


def build_opencode_mcp_block(
    servers: Sequence[MCPServerConfig],
) -> dict[str, dict[str, object]]:
    """
    Translate Goalrail MCP server declarations into opencode.json's ``mcp`` block.

    Mirrors how codex/claude expose the agent's MCP servers, but via opencode's
    own config (no relay): ``stdio`` → ``{type:"local", command:[cmd, *args],
    environment, enabled}``; ``http`` → ``{type:"remote", url, headers,
    enabled}``. Entries opencode can't represent (missing command / url) are
    skipped.

    :param servers: The agent spec's ``mcp_servers``.
    :returns: An opencode ``mcp`` block keyed by server name (empty when none
        are representable).
    """
    block: dict[str, dict[str, object]] = {}
    for server in servers:
        name = getattr(server, "name", None)
        if not name:
            continue
        if getattr(server, "transport", "http") == "stdio":
            command = getattr(server, "command", None)
            if not command:
                continue
            entry: dict[str, object] = {
                "type": "local",
                "command": [command, *getattr(server, "args", [])],
                "enabled": True,
            }
            env = dict(getattr(server, "env", {}) or {})
            if env:
                entry["environment"] = env
        else:
            url = getattr(server, "url", None)
            if not url:
                continue
            headers = dict(getattr(server, "headers", {}) or {})
            entry = {"type": "remote", "url": url, "enabled": True}
            if headers:
                entry["headers"] = headers
        block[str(name)] = entry
    return block


def build_opencode_goalrail_mcp_server(
    bridge_dir: Path, *, python_executable: str | None = None
) -> dict[str, dict[str, object]]:
    """
    Build the opencode ``mcp`` entry that connects opencode to Goalrail's MCP.

    This is what makes opencode's model call the Goalrail builtin tools
    (``sys_session_*``, ``sys_agent_*``, ``load_skill``, ``web_fetch``,
    ``list_comments``/``update_comment``, policy tools, …). opencode launches the
    SHARED ``goalrail.claude_native_bridge serve-mcp`` as a ``{type:"local"}``
    stdio MCP server (the same relay codex/cursor/qwen use); ``serve-mcp`` reads
    the relay URL+token from ``tool_relay.json`` in *bridge_dir* (written by the
    runner's comment relay) and proxies each tool call back through the Goalrail
    server, where policy is enforced. The command is sourced from
    :func:`claude_native_bridge.build_mcp_config` so the invocation stays in one
    place.

    :param bridge_dir: OpenCode-native bridge directory (must hold ``bridge.json``
        + ``tool_relay.json``).
    :param python_executable: Python to run ``serve-mcp`` with; ``None`` uses the
        runner interpreter (has ``goalrail`` importable).
    :returns: A one-entry ``mcp`` block ``{"goalrail": {type:"local", …}}``.
    """
    from goalrail.claude_native_bridge import build_mcp_config

    claude_cfg = build_mcp_config(bridge_dir, python_executable=python_executable)
    # build_mcp_config returns {"mcpServers": {"<name>": {command, args, env}}};
    # opencode wants a flat command list + ``environment``.
    name, server = next(iter(claude_cfg["mcpServers"].items()))
    entry: dict[str, object] = {
        "type": "local",
        "command": [server["command"], *server.get("args", [])],
        "enabled": True,
    }
    env = dict(server.get("env", {}) or {})
    if env:
        entry["environment"] = env
    return {str(name): entry}


def write_opencode_provider_config(xdg_config_home: Path, config: Mapping[str, object]) -> Path:
    """
    Atomically write ``<xdg_config_home>/opencode/opencode.json`` (``0600``).

    :param xdg_config_home: The per-session ``XDG_CONFIG_HOME`` the server uses.
    :param config: The provider config dict (see
        :func:`build_opencode_provider_config`).
    :returns: The path written.
    """
    cfg_dir = xdg_config_home / "opencode"
    cfg_dir.mkdir(mode=0o700, parents=True, exist_ok=True)
    path = cfg_dir / "opencode.json"
    payload = json.dumps(config, indent=2, sort_keys=True) + "\n"
    fd, tmp_name = tempfile.mkstemp(prefix="opencode.json.", dir=str(cfg_dir))
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as handle:
            handle.write(payload)
        os.chmod(tmp_name, 0o600)
        os.replace(tmp_name, path)
    finally:
        if os.path.exists(tmp_name):
            os.unlink(tmp_name)
    return path
