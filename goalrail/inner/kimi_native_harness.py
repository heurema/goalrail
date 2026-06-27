"""``harness: kimi-native`` wrap (the native Kimi Code TUI).

Thin module exposing :func:`create_app` — the entry point the shared
:mod:`goalrail.runtime.harnesses._runner` invokes after the parent process
resolves ``"kimi-native"`` to this module via
:data:`goalrail.runtime.harnesses._HARNESS_MODULES`.

Wraps a :class:`goalrail.inner.kimi_native_executor.KimiNativeExecutor`,
which injects web-UI messages into the running ``kimi`` TUI (launched by
``goalrail kimi`` in the session terminal) via tmux. The bridge dir is read
from :data:`~goalrail.kimi_native_bridge.BRIDGE_DIR_ENV_VAR` in the spawn env.

Tool policies: kimi-native enforces Goalrail's tool deny-policy via a
``PreToolUse`` hook (registered in the per-session ``config.toml`` built by
:mod:`goalrail.kimi_native_credentials`, dispatched to
:mod:`goalrail.kimi_native_hook`). A ``POLICY_ACTION_DENY`` verdict blocks the
tool with the policy reason; everything else is "no opinion", so ``kimi``'s own
in-TUI approval prompt still runs — the deployment's deny-gate and the user's
own consent are kept as two independent gates. A companion ``PermissionRequest``
hook surfaces the pending approval in the web UI read-only (the yes/no is
answered in the TUI, which Goalrail cannot intercept). Connector/tool ASK
policies are not enforced (kimi owns the ask); treat the kimi TUI as the
approval surface, with Goalrail able to hard-deny.
"""

from __future__ import annotations

from fastapi import FastAPI

from goalrail.inner.executor import Executor
from goalrail.inner.kimi_native_executor import KimiNativeExecutor
from goalrail.runtime.harnesses._executor_adapter import ExecutorAdapter


def _build_kimi_native_executor() -> Executor:
    """Construct a :class:`KimiNativeExecutor` (reads the bridge dir from env)."""
    return KimiNativeExecutor()


def create_app() -> FastAPI:
    """Build the kimi-native harness's FastAPI app (required entry point)."""
    adapter = ExecutorAdapter(executor_factory=_build_kimi_native_executor)
    return adapter.build()
