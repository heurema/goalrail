"""Tests for goalrail.onboarding.secrets."""

from __future__ import annotations

from pathlib import Path

import pytest

from goalrail.onboarding.secrets import _config_home


def test_config_home_uses_goalrail_config_home_env(
    monkeypatch: pytest.MonkeyPatch,
    tmp_path: Path,
) -> None:
    """``GOALRAIL_CONFIG_HOME`` redirects the secrets file backend."""
    monkeypatch.setenv("GOALRAIL_CONFIG_HOME", str(tmp_path))

    assert _config_home() == str(tmp_path)
