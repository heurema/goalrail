"""``harness: hermes-native`` wrap (the native Hermes TUI).

Thin module exposing :func:`create_app` — the entry point the shared
:mod:`goalrail.runtime.harnesses._runner` invokes after the parent process
resolves ``"hermes-native"`` to this module via
:data:`goalrail.runtime.harnesses._HARNESS_MODULES`.

Wraps a :class:`goalrail.inner.hermes_native_executor.HermesNativeExecutor`, which
injects web-UI messages into the running ``hermes`` TUI (launched by
``goalrail hermes`` in the session terminal) via tmux. The bridge dir is read from
:data:`~goalrail.hermes_native_bridge.BRIDGE_DIR_ENV_VAR` in the spawn env.

Tool policies: Goalrail policies are enforced via a per-session ``HERMES_HOME``
that registers a ``pre_tool_call`` shell hook (the same hook the headless
``hermes`` harness uses). The runner writes this before launching the TUI (see
:func:`goalrail.hermes_native_bridge.write_policy_hook_config`). Hermes' own
in-terminal approval prompt still fires for dangerous commands and is mirrored
to the web UI by :mod:`goalrail.hermes_native_permissions`.
"""

from __future__ import annotations

from fastapi import FastAPI

from goalrail.inner.executor import Executor
from goalrail.inner.hermes_native_executor import HermesNativeExecutor
from goalrail.runtime.harnesses._executor_adapter import ExecutorAdapter


def _build_hermes_native_executor() -> Executor:
    """Construct a :class:`HermesNativeExecutor` (reads the bridge dir from env)."""
    return HermesNativeExecutor()


def create_app() -> FastAPI:
    """Build the hermes-native harness's FastAPI app (required entry point)."""
    adapter = ExecutorAdapter(executor_factory=_build_hermes_native_executor)
    return adapter.build()
