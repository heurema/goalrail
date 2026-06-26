"""Backward-compatibility shim for env-var prefix renames.

The project's env-var prefix has changed twice as the name evolved:
``OMNIAGENTS_`` (original) -> ``OMNIGENTS_`` -> ``OMNIGENT_`` -> ``GOALRAIL_``.
Most runtime code still reads the compatibility ``OMNIGENT_`` names. To keep
existing deployments, CI configs, and shell profiles working while introducing
the Goalrail prefix, this shim mirrors every known prefix onto both
``GOALRAIL_*`` and ``OMNIGENT_*`` equivalents at process startup.

Precedence is:

1. ``GOALRAIL_*``
2. ``OMNIGENT_*``
3. ``OMNIGENTS_*``
4. ``OMNIAGENTS_*``

The mirror is installed once, as early as possible, from
``omnigent/__init__.py`` so it runs before any submodule reads the
environment. Out-of-package entry points that read env *before* importing the
``omnigent`` package (the Docker / Databricks deploy entrypoints) call
:func:`mirror_legacy_env` directly.
"""

from __future__ import annotations

import os
from pathlib import Path

# The canonical public prefix and the compatibility prefix most existing
# runtime readers still consume.
_CANONICAL_PREFIX = "GOALRAIL_"
_COMPAT_PREFIX = "OMNIGENT_"
_CANONICAL_CONFIG_HOME_ENV = "GOALRAIL_CONFIG_HOME"
_COMPAT_CONFIG_HOME_ENV = "OMNIGENT_CONFIG_HOME"
_CANONICAL_DATA_HOME_ENV = "GOALRAIL_DATA_DIR"
_COMPAT_DATA_HOME_ENV = "OMNIGENT_DATA_DIR"
_CANONICAL_CONFIG_HOME_DIR = ".goalrail"
_COMPAT_CONFIG_HOME_DIR = ".omnigent"

# Legacy prefixes are ordered newest-first so that when more than one legacy
# prefix is set for the same variable, the newer one wins.
_LEGACY_PREFIXES = ("OMNIGENTS_", "OMNIAGENTS_")
_LEGACY_CONFIG_HOME_DIRS = (".omnigents", ".omniagents")


def mirror_legacy_env() -> None:
    """
    Mirror Goalrail and legacy env-var prefixes onto compatibility names.

    For every supported prefix, populate the corresponding ``GOALRAIL_*`` and
    ``OMNIGENT_*`` variables according to the precedence documented above.
    Existing runtime code can keep reading ``OMNIGENT_*`` during the migration,
    while new callers can use ``GOALRAIL_*``.

    Example: with ``GOALRAIL_SKIP_WEB_UI=1`` and
    ``OMNIGENT_SKIP_WEB_UI=0``, this leaves both names set to ``1`` because the
    canonical Goalrail prefix wins. With only ``OMNIAGENTS_SKIP_WEB_UI=1`` set,
    both ``GOALRAIL_SKIP_WEB_UI`` and ``OMNIGENT_SKIP_WEB_UI`` become ``1``.

    :returns: ``None``. Mutates :data:`os.environ` in place.
    """
    for name, value in list(os.environ.items()):
        if name.startswith(_CANONICAL_PREFIX):
            suffix = name[len(_CANONICAL_PREFIX) :]
            os.environ[_COMPAT_PREFIX + suffix] = value

    for name, value in list(os.environ.items()):
        if name.startswith(_COMPAT_PREFIX):
            suffix = name[len(_COMPAT_PREFIX) :]
            os.environ.setdefault(_CANONICAL_PREFIX + suffix, value)

    for prefix in _LEGACY_PREFIXES:
        for name, value in list(os.environ.items()):
            if not name.startswith(prefix):
                continue
            suffix = name[len(prefix) :]
            canonical_name = _CANONICAL_PREFIX + suffix
            compat_name = _COMPAT_PREFIX + suffix
            os.environ.setdefault(canonical_name, value)
            os.environ.setdefault(compat_name, os.environ[canonical_name])


def config_home_path() -> Path:
    """
    Resolve the per-user config home during the Goalrail rename.

    Explicit env overrides win first, with ``GOALRAIL_CONFIG_HOME`` taking
    precedence over ``OMNIGENT_CONFIG_HOME``. Without an override, existing
    directories are honored newest-first so users can opt into
    ``~/.goalrail`` while existing ``~/.omnigent`` installs keep working.
    Fresh installs still default to ``~/.omnigent`` until the broader state
    migration is handled separately.

    :returns: The effective config home path.
    """
    if config_home := os.environ.get(_CANONICAL_CONFIG_HOME_ENV):
        return Path(config_home).expanduser()
    if config_home := os.environ.get(_COMPAT_CONFIG_HOME_ENV):
        return Path(config_home).expanduser()

    home = Path.home()
    candidates = (
        home / _CANONICAL_CONFIG_HOME_DIR,
        home / _COMPAT_CONFIG_HOME_DIR,
        *(home / directory for directory in _LEGACY_CONFIG_HOME_DIRS),
    )
    for candidate in candidates:
        if candidate.exists():
            return candidate
    return home / _COMPAT_CONFIG_HOME_DIR


def data_home_path() -> Path:
    """
    Resolve the per-user runtime data home during the Goalrail rename.

    Explicit data-dir overrides win first, with ``GOALRAIL_DATA_DIR`` taking
    precedence over ``OMNIGENT_DATA_DIR``. Without an override, existing
    directories are honored newest-first so users can opt into
    ``~/.goalrail`` while existing ``~/.omnigent`` installs keep working.
    Fresh installs still default to ``~/.omnigent`` until daemon/data state
    migration is handled separately.

    :returns: The effective runtime data home path.
    """
    if data_home := os.environ.get(_CANONICAL_DATA_HOME_ENV):
        return Path(data_home).expanduser()
    if data_home := os.environ.get(_COMPAT_DATA_HOME_ENV):
        return Path(data_home).expanduser()

    home = Path.home()
    candidates = (
        home / _CANONICAL_CONFIG_HOME_DIR,
        home / _COMPAT_CONFIG_HOME_DIR,
        *(home / directory for directory in _LEGACY_CONFIG_HOME_DIRS),
    )
    for candidate in candidates:
        if candidate.exists():
            return candidate
    return home / _COMPAT_CONFIG_HOME_DIR
