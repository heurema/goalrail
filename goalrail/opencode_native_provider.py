"""Synthesize OpenCode config for the native-server harness."""

from __future__ import annotations

import json
import os
import tempfile
from collections.abc import Mapping
from pathlib import Path


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
