import os
from pathlib import Path

import pytest

from omnigent._env_compat import config_home_path, data_home_path, mirror_legacy_env

_PREFIXES = ("GOALRAIL_", "OMNIGENT_", "OMNIGENTS_", "OMNIAGENTS_")


@pytest.fixture(autouse=True)
def _restore_prefixed_env():
    original = {name: value for name, value in os.environ.items() if name.startswith(_PREFIXES)}
    yield
    for name in list(os.environ):
        if name.startswith(_PREFIXES):
            del os.environ[name]
    os.environ.update(original)


def _clear_env(monkeypatch, suffix: str) -> None:
    for prefix in _PREFIXES:
        monkeypatch.delenv(prefix + suffix, raising=False)


def test_goalrail_prefix_wins_over_omnigent(monkeypatch):
    _clear_env(monkeypatch, "MODEL")
    monkeypatch.setenv("GOALRAIL_MODEL", "new-model")
    monkeypatch.setenv("OMNIGENT_MODEL", "old-model")

    mirror_legacy_env()

    assert os.environ["GOALRAIL_MODEL"] == "new-model"
    assert os.environ["OMNIGENT_MODEL"] == "new-model"


def test_omnigent_prefix_populates_goalrail_when_absent(monkeypatch):
    _clear_env(monkeypatch, "SKIP_WEB_UI")
    monkeypatch.setenv("OMNIGENT_SKIP_WEB_UI", "1")

    mirror_legacy_env()

    assert os.environ["GOALRAIL_SKIP_WEB_UI"] == "1"
    assert os.environ["OMNIGENT_SKIP_WEB_UI"] == "1"


def test_legacy_prefixes_populate_goalrail_and_omnigent(monkeypatch):
    _clear_env(monkeypatch, "CONFIG_HOME")
    monkeypatch.setenv("OMNIAGENTS_CONFIG_HOME", "/oldest")
    monkeypatch.setenv("OMNIGENTS_CONFIG_HOME", "/newer")

    mirror_legacy_env()

    assert os.environ["GOALRAIL_CONFIG_HOME"] == "/newer"
    assert os.environ["OMNIGENT_CONFIG_HOME"] == "/newer"


def test_goalrail_prefix_wins_over_legacy_prefixes(monkeypatch):
    _clear_env(monkeypatch, "CONFIG_HOME")
    monkeypatch.setenv("GOALRAIL_CONFIG_HOME", "/goalrail")
    monkeypatch.setenv("OMNIGENTS_CONFIG_HOME", "/legacy")

    mirror_legacy_env()

    assert os.environ["GOALRAIL_CONFIG_HOME"] == "/goalrail"
    assert os.environ["OMNIGENT_CONFIG_HOME"] == "/goalrail"


def test_omnigent_prefix_wins_over_older_legacy_prefixes(monkeypatch):
    _clear_env(monkeypatch, "CONFIG_HOME")
    monkeypatch.setenv("OMNIGENT_CONFIG_HOME", "/omnigent")
    monkeypatch.setenv("OMNIGENTS_CONFIG_HOME", "/legacy")

    mirror_legacy_env()

    assert os.environ["GOALRAIL_CONFIG_HOME"] == "/omnigent"
    assert os.environ["OMNIGENT_CONFIG_HOME"] == "/omnigent"


def test_config_home_path_prefers_goalrail_config_home_env(monkeypatch, tmp_path):
    _clear_env(monkeypatch, "CONFIG_HOME")
    goalrail_home = tmp_path / "goalrail-home"
    omnigent_home = tmp_path / "omnigent-home"
    monkeypatch.setenv("GOALRAIL_CONFIG_HOME", str(goalrail_home))
    monkeypatch.setenv("OMNIGENT_CONFIG_HOME", str(omnigent_home))

    assert config_home_path() == goalrail_home


def test_config_home_path_keeps_omnigent_config_home_env(monkeypatch, tmp_path):
    _clear_env(monkeypatch, "CONFIG_HOME")
    config_home = tmp_path / "omnigent-home"
    monkeypatch.setenv("OMNIGENT_CONFIG_HOME", str(config_home))

    assert config_home_path() == config_home


def test_config_home_path_uses_existing_goalrail_home(monkeypatch, tmp_path):
    _clear_env(monkeypatch, "CONFIG_HOME")
    monkeypatch.setenv("HOME", str(tmp_path))
    goalrail_home = tmp_path / ".goalrail"
    omnigent_home = tmp_path / ".omnigent"
    goalrail_home.mkdir()
    omnigent_home.mkdir()

    assert config_home_path() == goalrail_home


def test_config_home_path_reads_existing_omnigent_home(monkeypatch, tmp_path):
    _clear_env(monkeypatch, "CONFIG_HOME")
    monkeypatch.setenv("HOME", str(tmp_path))
    omnigent_home = tmp_path / ".omnigent"
    omnigent_home.mkdir()

    assert config_home_path() == omnigent_home


def test_config_home_path_keeps_fresh_default_at_omnigent(monkeypatch, tmp_path):
    _clear_env(monkeypatch, "CONFIG_HOME")
    monkeypatch.setenv("HOME", str(tmp_path))

    assert config_home_path() == Path(tmp_path) / ".omnigent"


def test_data_home_path_prefers_goalrail_data_dir_env(monkeypatch, tmp_path):
    _clear_env(monkeypatch, "DATA_DIR")
    goalrail_home = tmp_path / "goalrail-data"
    omnigent_home = tmp_path / "omnigent-data"
    monkeypatch.setenv("GOALRAIL_DATA_DIR", str(goalrail_home))
    monkeypatch.setenv("OMNIGENT_DATA_DIR", str(omnigent_home))

    assert data_home_path() == goalrail_home


def test_data_home_path_keeps_omnigent_data_dir_env(monkeypatch, tmp_path):
    _clear_env(monkeypatch, "DATA_DIR")
    data_home = tmp_path / "omnigent-data"
    monkeypatch.setenv("OMNIGENT_DATA_DIR", str(data_home))

    assert data_home_path() == data_home


def test_data_home_path_uses_existing_goalrail_home(monkeypatch, tmp_path):
    _clear_env(monkeypatch, "DATA_DIR")
    monkeypatch.setenv("HOME", str(tmp_path))
    goalrail_home = tmp_path / ".goalrail"
    omnigent_home = tmp_path / ".omnigent"
    goalrail_home.mkdir()
    omnigent_home.mkdir()

    assert data_home_path() == goalrail_home


def test_data_home_path_keeps_fresh_default_at_omnigent(monkeypatch, tmp_path):
    _clear_env(monkeypatch, "DATA_DIR")
    monkeypatch.setenv("HOME", str(tmp_path))

    assert data_home_path() == Path(tmp_path) / ".omnigent"
