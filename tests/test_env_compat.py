import os

from omnigent._env_compat import mirror_legacy_env

_PREFIXES = ("GOALRAIL_", "OMNIGENT_", "OMNIGENTS_", "OMNIAGENTS_")


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
