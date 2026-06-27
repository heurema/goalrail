"""Canonical Goalrail environment and state paths."""

from __future__ import annotations

import os
from pathlib import Path

CONFIG_HOME_ENV = "GOALRAIL_CONFIG_HOME"
DATA_HOME_ENV = "GOALRAIL_DATA_DIR"
STATE_DIR_NAME = ".goalrail"


def mirror_legacy_env() -> None:
    """Kept as a startup hook; Goalrail no longer mirrors legacy prefixes."""


def config_home_path() -> Path:
    """Return the Goalrail config home path."""
    if config_home := os.environ.get(CONFIG_HOME_ENV):
        return Path(config_home).expanduser()
    return Path.home() / STATE_DIR_NAME


def data_home_path() -> Path:
    """Return the Goalrail runtime data path."""
    if data_home := os.environ.get(DATA_HOME_ENV):
        return Path(data_home).expanduser()
    return Path.home() / STATE_DIR_NAME
