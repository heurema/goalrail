"""
Harness package — per-conversation subprocesses that implement a
subset of the Goalrail REST API.

See ``designs/SERVER_HARNESS_CONTRACT.md`` for the full contract.
The harness IS an HTTP service speaking the same Pydantic models AP
serves to external clients (re-use ``goalrail.server.schemas`` —
there is no separate protocol module).

This package contains:

- ``_HARNESS_MODULES``: registry mapping harness name (the value of
  ``spec.executor.harness`` in an agent spec) to the fully-qualified
  Python module path that exports a zero-argument ``create_app() ->
  FastAPI``. Populated as per-harness wraps land (Phase 1 step 4).
- ``process_manager``: ``HarnessProcessManager`` — owns
  per-conversation subprocess lifecycle.
- ``_runner``: shared ``python -m`` entrypoint that any registered
  harness's ``create_app()`` is served through.

The package directory is intentionally small. Behavior lives in the
sibling modules; this ``__init__.py`` is just the registry.
"""

from __future__ import annotations

# Harness-name → fully-qualified module path. Each module must
# export ``create_app() -> FastAPI``; the runner imports the module,
# calls the factory, and serves the result over a Unix socket.
#
# Populated as per-harness wraps land in Phase 1 step 4. The test
# suite injects fixture entries at test time (via direct dict
# mutation in conftest fixtures).
_HARNESS_MODULES: dict[str, str] = {
    # Step 4b: claude-sdk harness wrap. See
    # goalrail/inner/claude_sdk_harness.py.
    "claude-sdk": "goalrail.inner.claude_sdk_harness",
    # User-facing alias accepted in specs / Goalrail harness dispatch.
    "claude": "goalrail.inner.claude_sdk_harness",
    # Native Claude Code terminal bridge used by ``goalrail claude``.
    "claude-native": "goalrail.inner.claude_native_harness",
    # Native Codex TUI terminal bridge used by ``goalrail codex``.
    "codex-native": "goalrail.inner.codex_native_harness",
    # Step 4c: codex harness wrap. See
    # goalrail/inner/codex_harness.py.
    "codex": "goalrail.inner.codex_harness",
    # Step 4d: pi harness wrap. See
    # goalrail/inner/pi_harness.py.
    "pi": "goalrail.inner.pi_harness",
    # Native Pi TUI bridge used by ``goalrail pi``.
    "pi-native": "goalrail.inner.pi_native_harness",
    # Native Antigravity (agy) TUI terminal bridge used by
    # ``goalrail antigravity``. The in-process SDK counterpart is the
    # canonical ``antigravity`` harness registered below.
    "antigravity-native": "goalrail.inner.antigravity_native_harness",
    # Step 4e: openai-agents harness wrap. See
    # goalrail/inner/openai_agents_sdk_harness.py. Registry
    # key is the Goalrail-side spelling (``openai-agents``,
    # no ``-sdk`` suffix) to match
    # ``GoalrailExecutor``'s harness allowlist and the
    # ``executor.harness`` field used in Goalrail YAML; the
    # backing Python module retains the ``_sdk`` suffix because
    # the underlying SDK package is ``openai-agents`` and the
    # executor class is :class:`OpenAIAgentsSDKExecutor`.
    "openai-agents": "goalrail.inner.openai_agents_sdk_harness",
    # cursor harness wrap (Cursor's ``cursor-agent`` CLI, headless). See
    # goalrail/inner/cursor_harness.py.
    "cursor": "goalrail.inner.cursor_harness",
    # Kimi Code CLI harness wrap (Moonshot AI's ``kimi`` CLI, headless). See
    # goalrail/inner/kimi_harness.py. Drives ``kimi --print --output-format
    # stream-json`` per turn; resumes via ``--session <uuid>`` captured from
    # the prior turn's stderr.
    "kimi": "goalrail.inner.kimi_harness",
    # User-facing alias matching the upstream product name ("Kimi Code").
    "kimi-code": "goalrail.inner.kimi_harness",
    # cursor-native harness wrap. Drives the resident ``cursor-agent`` TUI by
    # injecting each web-UI turn into its tmux pane and mirroring the transcript
    # back — a native-CLI harness like claude/codex/pi-native, so it IS in
    # ``NATIVE_HARNESSES``. See goalrail/inner/cursor_native_harness.py.
    "cursor-native": "goalrail.inner.cursor_native_harness",
    # Native Kiro TUI bridge used by ``goalrail kiro``.
    "kiro-native": "goalrail.inner.kiro_native_harness",
    # goose-native harness wrap. Drives the resident ``goose session`` TUI by
    # injecting each web-UI turn into its tmux pane and mirroring the transcript
    # back from Goose's SQLite session store — a native-CLI harness like
    # cursor-native, so it IS in ``NATIVE_HARNESSES``. See
    # goalrail/inner/goose_native_harness.py.
    "goose-native": "goalrail.inner.goose_native_harness",
    # qwen-native harness wrap. Drives the resident ``qwen`` TUI by appending
    # JSONL ``submit`` commands to its ``--input-file`` and mirroring the
    # transcript back from its ``--json-file`` event stream — a native-CLI
    # harness like goose-native, so it IS in ``NATIVE_HARNESSES``. The bare
    # ``qwen`` name stays the ACP-piped harness. See
    # goalrail/inner/qwen_native_harness.py.
    "qwen-native": "goalrail.inner.qwen_native_harness",
    # Native Kimi Code TUI bridge used by ``goalrail kimi``. Drives the resident
    # ``kimi`` TUI by injecting each web-UI turn into its tmux pane (tmux paste)
    # — a native-CLI harness like claude/codex/cursor-native, so it IS in
    # ``NATIVE_HARNESSES``. Distinct from the headless ``kimi`` SDK harness
    # above. See goalrail/inner/kimi_native_harness.py.
    "kimi-native": "goalrail.inner.kimi_native_harness",
    # Google Antigravity SDK harness wrap. See
    # goalrail/inner/antigravity_harness.py. In-process SDK harness
    # (``google-antigravity``), like openai-agents — Goalrail spawns no CLI
    # binary or sandbox subprocess (the SDK itself launches a native
    # localharness binary; needs glibc >=~2.36). Drives Gemini 3.5 Flash by
    # default (also Claude / GPT-OSS), with Gemini API-key or Vertex AI auth.
    "antigravity": "goalrail.inner.antigravity_harness",
    # Qwen Code harness wrap. See goalrail/inner/qwen_harness.py.
    # Drives the ``qwen`` CLI in ACP mode (``qwen --acp``) for agent execution.
    "qwen": "goalrail.inner.qwen_harness",
    # Headless Goose harness wrap. See goalrail/inner/goose_harness.py.
    # Drives Block's ``goose`` CLI in ACP mode (``goose acp``) — the chat-first
    # counterpart to the terminal-first ``goose-native`` TUI harness. Tool
    # approvals surface as web elicitation cards via session/request_permission.
    "goose": "goalrail.inner.goose_harness",
    # Native OpenCode server bridge used by ``goalrail opencode``. The runner
    # owns ``opencode serve`` + an SSE forwarder and this harness injects each
    # web-UI turn over loopback HTTP — a native-server harness like
    # codex-native, so both ``opencode-native`` and its ``native-opencode``
    # alias are in ``NATIVE_HARNESSES``. See
    # goalrail/inner/opencode_native_harness.py.
    "opencode-native": "goalrail.inner.opencode_native_harness",
    # ``opencode`` is accepted as a friendly alias for the canonical
    # ``opencode-native`` (there is no separate SDK ``opencode`` harness).
    "opencode": "goalrail.inner.opencode_native_harness",
    # GitHub Copilot SDK harness wrap. See goalrail/inner/copilot_harness.py.
    # In-process SDK harness (``github-copilot-sdk``), like cursor / antigravity:
    # the SDK bundles the Copilot CLI binary it drives as a backing server, so
    # Goalrail spawns no separately-installed CLI. Authenticates against GitHub's
    # Copilot backend with a GitHub token.
    "copilot": "goalrail.inner.copilot_harness",
    # Hermes Agent harness wrap. Runs the ``hermes`` CLI as a subprocess
    # for each turn, managing its own session state via Hermes' SQLite
    # session store. See goalrail/inner/hermes_harness.py and
    # goalrail/inner/hermes_executor.py. The ``hermes`` binary must be
    # on PATH (or set by HARNESS_HERMES_PATH).
    "hermes": "goalrail.inner.hermes_harness",
    # hermes-native harness wrap. Drives the resident ``hermes`` TUI by
    # injecting each web-UI turn into its tmux pane and mirroring the transcript
    # back from Hermes' SQLite ``state.db`` session store — a native-CLI harness
    # like goose-native, so it IS in ``NATIVE_HARNESSES``. The bare ``hermes``
    # name stays the headless subprocess harness. See
    # goalrail/inner/hermes_native_harness.py.
    "hermes-native": "goalrail.inner.hermes_native_harness",
}

__all__ = ["_HARNESS_MODULES"]
