"""Tests for shared Goalrail runtime data path compatibility."""

from __future__ import annotations

from pathlib import Path

import pytest

from goalrail.chat import _goalrail_log_dir, _goalrail_persistent_dir
from goalrail.host.connect import _runner_log_dir


def test_chat_persistent_dir_uses_goalrail_data_dir(
    monkeypatch: pytest.MonkeyPatch,
    tmp_path: Path,
) -> None:
    """``GOALRAIL_DATA_DIR`` redirects chat runtime state."""
    data_home = tmp_path / "goalrail-data"
    monkeypatch.setenv("GOALRAIL_DATA_DIR", str(data_home))
    monkeypatch.setenv("GOALRAIL_DATA_DIR", str(tmp_path / "goalrail-data"))

    assert _goalrail_persistent_dir() == data_home
    assert (data_home / "artifacts").is_dir()


def test_chat_log_dir_uses_goalrail_data_dir(
    monkeypatch: pytest.MonkeyPatch,
    tmp_path: Path,
) -> None:
    """Process logs live under the effective runtime data home."""
    data_home = tmp_path / "goalrail-data"
    monkeypatch.setenv("GOALRAIL_DATA_DIR", str(data_home))

    assert _goalrail_log_dir() == data_home / "logs"
    assert (data_home / "logs").is_dir()


def test_host_runner_log_dir_uses_goalrail_data_dir(
    monkeypatch: pytest.MonkeyPatch,
    tmp_path: Path,
) -> None:
    """Host runner logs live under the effective runtime data home."""
    data_home = tmp_path / "goalrail-data"
    monkeypatch.setenv("GOALRAIL_DATA_DIR", str(data_home))

    assert _runner_log_dir() == data_home / "logs" / "host-runner"
