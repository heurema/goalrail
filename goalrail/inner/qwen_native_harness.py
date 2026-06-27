"""``harness: qwen-native`` wrap (the native qwen TUI).

Thin module exposing :func:`create_app` — the entry point the shared
:mod:`goalrail.runtime.harnesses._runner` invokes after the parent process
resolves ``"qwen-native"`` to this module via
:data:`goalrail.runtime.harnesses._HARNESS_MODULES`.

Wraps a :class:`goalrail.inner.qwen_native_executor.QwenNativeExecutor`, which
appends web-UI messages to the running ``qwen`` TUI's ``--input-file`` (launched
by ``goalrail qwen`` in the session terminal). The bridge dir is read from
:data:`~goalrail.qwen_native_bridge.BRIDGE_DIR_ENV_VAR` in the spawn env.

Tool policies: in this first cut, qwen runs its tools inside its own TUI and
gates them with its own in-terminal approval (like goose-/cursor-native), which
Goalrail does not intercept. qwen *can* delegate approval externally
(``can_use_tool`` control requests on ``--json-file``, answered via
``confirmation_response`` on ``--input-file``) — wiring that through Goalrail's
TOOL_CALL policy is the documented follow-up; see ``docs/QWEN_NATIVE_DESIGN.md``.
"""

from __future__ import annotations

from fastapi import FastAPI

from goalrail.inner.executor import Executor
from goalrail.inner.qwen_native_executor import QwenNativeExecutor
from goalrail.runtime.harnesses._executor_adapter import ExecutorAdapter


def _build_qwen_native_executor() -> Executor:
    """Construct a :class:`QwenNativeExecutor` (reads the bridge dir from env)."""
    return QwenNativeExecutor()


def create_app() -> FastAPI:
    """Build the qwen-native harness's FastAPI app (required entry point)."""
    adapter = ExecutorAdapter(executor_factory=_build_qwen_native_executor)
    return adapter.build()
