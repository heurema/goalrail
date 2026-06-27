"""``harness: cursor-native`` wrap (the native Cursor TUI).

Thin module exposing :func:`create_app` — the entry point the shared
:mod:`goalrail.runtime.harnesses._runner` invokes after the parent process
resolves ``"cursor-native"`` to this module via
:data:`goalrail.runtime.harnesses._HARNESS_MODULES`.

Wraps a :class:`goalrail.inner.cursor_native_executor.CursorNativeExecutor`,
which injects web-UI messages into the running ``cursor-agent`` TUI (launched by
``goalrail cursor`` in the session terminal) via tmux. The bridge dir is read
from :data:`~goalrail.cursor_native_bridge.BRIDGE_DIR_ENV_VAR` in the spawn env.

Tool policies: Goalrail's PreToolUse/PostToolUse policy gates (which claude- and
codex-native enforce via hooks) do NOT apply to cursor-native — ``cursor-agent``
runs its tools inside its own TUI and gates them with its own in-terminal
approval prompts (and ``--force``/``--yolo``/sandbox config), which goalrail does
not intercept. Treat the cursor TUI's own approval as the sole tool gate; do not
assume Goalrail connector/tool deny-policies constrain a cursor-native session.
"""

from __future__ import annotations

from fastapi import FastAPI

from goalrail.inner.cursor_native_executor import CursorNativeExecutor
from goalrail.inner.executor import Executor
from goalrail.runtime.harnesses._executor_adapter import ExecutorAdapter


def _build_cursor_native_executor() -> Executor:
    """Construct a :class:`CursorNativeExecutor` (reads the bridge dir from env)."""
    return CursorNativeExecutor()


def create_app() -> FastAPI:
    """Build the cursor-native harness's FastAPI app (required entry point)."""
    adapter = ExecutorAdapter(executor_factory=_build_cursor_native_executor)
    return adapter.build()
