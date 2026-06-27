import os
from pathlib import Path

import pytest

from goalrail._env_compat import config_home_path, data_home_path, mirror_legacy_env


@pytest.fixture(autouse=True)
def _restore_goalrail_env():
    original = {name: value for name, value in os.environ.items() if name.startswith("GOALRAIL_")}
    yield
    for name in list(os.environ):
        if name.startswith("GOALRAIL_"):
            del os.environ[name]
    os.environ.update(original)


def test_mirror_legacy_env_is_noop(monkeypatch):
    monkeypatch.setenv("GOALRAIL_MODEL", "current")

    mirror_legacy_env()

    assert os.environ["GOALRAIL_MODEL"] == "current"


def test_config_home_path_prefers_goalrail_config_home_env(monkeypatch, tmp_path):
    config_home = tmp_path / "config-home"
    monkeypatch.setenv("GOALRAIL_CONFIG_HOME", str(config_home))

    assert config_home_path() == config_home


def test_config_home_path_defaults_to_goalrail_home(monkeypatch, tmp_path):
    monkeypatch.setenv("HOME", str(tmp_path))
    monkeypatch.delenv("GOALRAIL_CONFIG_HOME", raising=False)

    assert config_home_path() == Path(tmp_path) / ".goalrail"


def test_data_home_path_prefers_goalrail_data_dir_env(monkeypatch, tmp_path):
    data_home = tmp_path / "data-home"
    monkeypatch.setenv("GOALRAIL_DATA_DIR", str(data_home))

    assert data_home_path() == data_home


def test_data_home_path_defaults_to_goalrail_home(monkeypatch, tmp_path):
    monkeypatch.setenv("HOME", str(tmp_path))
    monkeypatch.delenv("GOALRAIL_DATA_DIR", raising=False)

    assert data_home_path() == Path(tmp_path) / ".goalrail"
