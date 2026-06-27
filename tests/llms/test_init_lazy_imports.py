"""Regression test for the goalrail.llms <-> goalrail.reasoning_effort cycle.

The eager top-level imports in ``goalrail/llms/__init__.py`` created
a circular load when any caller imported ``goalrail.llms.errors``
during the load of ``goalrail.reasoning_effort`` (which happens on
every server-routes import via ``server/routes/sessions.py``).

The fix in ``goalrail/llms/__init__.py`` switches to a
``__getattr__`` shim so ``Client`` and ``get_model_context_window``
are resolved lazily on first access. This test guards against
re-introducing the cycle by re-importing the affected modules
in a fresh interpreter-style namespace and asserting both the
short-form and long-form import paths work.
"""

from __future__ import annotations

import importlib
import sys
from collections.abc import Iterator
from typing import Any

import pytest

_PURGED_PREFIXES = (
    "goalrail.llms",
    "goalrail.reasoning_effort",
    "goalrail.server.routes.sessions",
)
_PARENT_ATTRS = (
    ("goalrail", "llms"),
    ("goalrail", "reasoning_effort"),
    ("goalrail.server.routes", "sessions"),
)


def _matches_purged_prefix(mod_name: str) -> bool:
    return any(
        mod_name == prefix or mod_name.startswith(prefix + ".") for prefix in _PURGED_PREFIXES
    )


@pytest.fixture(autouse=True)
def _restore_import_state() -> Iterator[None]:
    """Keep destructive sys.modules import-order probes test-local."""
    saved_modules = {
        mod_name: module
        for mod_name, module in sys.modules.items()
        if _matches_purged_prefix(mod_name)
    }
    saved_attrs: dict[tuple[str, str], tuple[bool, Any]] = {}
    for parent_name, attr_name in _PARENT_ATTRS:
        parent = sys.modules.get(parent_name)
        saved_attrs[(parent_name, attr_name)] = (
            parent is not None and hasattr(parent, attr_name),
            getattr(parent, attr_name, None) if parent is not None else None,
        )

    yield

    for mod_name in list(sys.modules):
        if _matches_purged_prefix(mod_name):
            sys.modules.pop(mod_name, None)
    sys.modules.update(saved_modules)
    for parent_name, attr_name in _PARENT_ATTRS:
        parent = sys.modules.get(parent_name)
        if parent is None:
            continue
        existed, value = saved_attrs[(parent_name, attr_name)]
        if existed:
            setattr(parent, attr_name, value)
        elif hasattr(parent, attr_name):
            delattr(parent, attr_name)


def _purge(prefix: str) -> None:
    """Drop any already-loaded modules under ``prefix`` so a fresh
    ``import`` exercises the module-load order again."""
    for mod_name in list(sys.modules):
        if mod_name == prefix or mod_name.startswith(prefix + "."):
            sys.modules.pop(mod_name, None)


def test_sessions_routes_import_does_not_trigger_cycle() -> None:
    """The original failure shape: importing the server routes module
    triggered ``reasoning_effort`` -> ``llms.errors`` -> ``llms.__init__``
    -> ``llms.client`` -> ``reasoning_effort`` re-entry."""
    _purge("goalrail.llms")
    _purge("goalrail.reasoning_effort")
    _purge("goalrail.server.routes.sessions")
    importlib.import_module("goalrail.server.routes.sessions")


def test_short_form_import_still_works() -> None:
    """``from goalrail.llms import Client`` must keep working
    after the lazy-attribute switch."""
    _purge("goalrail.llms")
    from goalrail.llms import Client, get_model_context_window

    assert Client is not None
    assert callable(get_model_context_window)


def test_module_only_import_does_not_load_client() -> None:
    """Importing ``goalrail.llms`` by itself should NOT eagerly pull
    in ``client.py`` -- that's the whole point of the lazy shim."""
    _purge("goalrail.llms")
    importlib.import_module("goalrail.llms")
    assert "goalrail.llms.client" not in sys.modules, (
        "goalrail.llms.client was imported eagerly; lazy shim regressed"
    )


def test_unknown_attribute_raises_attribute_error() -> None:
    """The ``__getattr__`` shim should preserve normal AttributeError
    semantics for unknown names."""
    _purge("goalrail.llms")
    import goalrail.llms as llms_pkg

    try:
        llms_pkg.does_not_exist  # noqa: B018
    except AttributeError as e:
        assert "does_not_exist" in str(e)
    else:
        raise AssertionError("expected AttributeError")
