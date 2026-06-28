"""
``harness: openai-agents`` wrap.

Thin module exposing :func:`create_app` — the entrypoint the
shared :mod:`goalrail.runtime.harnesses._runner` invokes after
the parent process resolves ``"openai-agents"`` to this module
via :data:`goalrail.runtime.harnesses._HARNESS_MODULES`.

The registry key is ``"openai-agents"`` (matching the Goalrail
YAML ``executor.harness`` spelling and ``GoalrailExecutor``'s
existing harness allowlist); the Python module retains the ``_sdk``
suffix because the underlying SDK package is ``openai-agents`` and
the executor class is :class:`OpenAIAgentsSDKExecutor`.

Internally, instantiates :class:`goalrail.runtime.harnesses._executor_adapter.ExecutorAdapter`
around a :class:`goalrail.inner.openai_agents_sdk_executor.OpenAIAgentsSDKExecutor`
configured from env vars the parent process sets before spawning.
Mirrors the claude-sdk wrap (``claude_sdk_harness.py``), codex
wrap (``codex_harness.py``), and pi wrap (``pi_harness.py``); see
the claude-sdk module's docstring for the v1 config-flow rationale
(env vars vs per-request).

OpenAI Agents SDK is the **simplest** of the four wrapped
harnesses because:

- No CLI binary — pure-Python ``openai-agents`` package, so no
  PATH check / no ``cli_binary`` field on the harness probe.
- No sandbox — the Python SDK runs in-process; there's no
  CLI subprocess to wrap with bwrap.
- No ``os_env`` field — the SDK doesn't host file/shell tools
  the way claude-sdk / codex / pi do; ``sys_os_*`` builtins
  travel through AP's tool surface as usual.
- Model is still a simple constructor override from the spawn env,
  not a CLI/runtime concern. The wrap also uses this model name to
  select the correct OpenAI Agents SDK endpoint default for models
  that need it.

Env vars read at startup:

- ``HARNESS_OPENAI_AGENTS_MODEL``: model identifier the
  inner executor pins for every turn, e.g.
  ``"gpt-5.4-mini"``. Constructor-level override —
  wins over the per-turn ``request.model`` (which under the
  harness contract carries the agent NAME, not an LLM
  identifier). ``None`` falls back to ``cfg.model`` then the
  executor's built-in default.
- ``HARNESS_OPENAI_AGENTS_GATEWAY_BASE_URL``: OpenAI-compatible gateway
  base URL.
- ``HARNESS_OPENAI_AGENTS_API_KEY``: direct OpenAI-compatible API
  key, written when the agent spec declares
  ``executor.auth: {type: api_key, api_key: …}``. Takes precedence
  over the ambient ``OPENAI_API_KEY`` env var so the spec is
  self-contained.
- ``HARNESS_OPENAI_AGENTS_USE_RESPONSES``: ``"1"`` / ``"true"``
  to use the OpenAI ``/responses`` endpoint (default); any other
  truthy-parsing-rejected value falls back to
  ``/chat/completions``. An explicit env-var value wins.
"""

from __future__ import annotations

import logging
import os

from fastapi import FastAPI

from goalrail.inner.executor import Executor
from goalrail.inner.openai_agents_sdk_executor import OpenAIAgentsSDKExecutor
from goalrail.runtime.harnesses._executor_adapter import ExecutorAdapter

_logger = logging.getLogger(__name__)

# Env-var keys the wrap reads at executor construction time. See
# the module docstring for semantics. Centralizing as constants
# so misconfigurations surface as a single grep target.
_ENV_MODEL = "HARNESS_OPENAI_AGENTS_MODEL"
_ENV_USE_RESPONSES = "HARNESS_OPENAI_AGENTS_USE_RESPONSES"
_ENV_GATEWAY_BASE_URL = "HARNESS_OPENAI_AGENTS_GATEWAY_BASE_URL"
_ENV_GATEWAY_AUTH_COMMAND = "HARNESS_OPENAI_AGENTS_GATEWAY_AUTH_COMMAND"
# Direct OpenAI-compatible API key set when the agent spec declares
# executor.auth: {type: api_key, api_key: …}. Takes precedence over
# ambient OPENAI_API_KEY in the caller's environment.
_ENV_API_KEY = "HARNESS_OPENAI_AGENTS_API_KEY"

# Truthy strings the wrap accepts for boolean env vars. Must
# match the claude-sdk / codex / pi wraps' parsers for
# consistency — operators learn one set of conventions, not five.
_TRUTHY_STRINGS = ("1", "true", "yes")


def _parse_truthy(env_var: str, default: bool) -> bool:
    """
    Parse a boolean-style env var the same way the claude-sdk /
    codex / pi wraps do.

    :param env_var: The env-var name (e.g.
        ``HARNESS_OPENAI_AGENTS_USE_RESPONSES``).
    :param default: The fallback when the env var is unset or
        empty.
    :returns: ``True`` if the value is in :data:`_TRUTHY_STRINGS`
        (case-insensitive); ``False`` for any other non-empty
        value; *default* when unset or empty.
    """
    raw = os.environ.get(env_var, "").strip().lower()
    if not raw:
        return default
    return raw in _TRUTHY_STRINGS


def _build_openai_agents_sdk_executor() -> Executor:
    """
    Construct an :class:`OpenAIAgentsSDKExecutor` from env-var
    config.

    Called lazily by the :class:`ExecutorAdapter` on the first
    turn. Client initialization happens at this point — operators
    see the failure surface as a startup error on the first
    request, not at FastAPI app boot.

    :returns: A configured :class:`OpenAIAgentsSDKExecutor`
        instance.
    :raises ImportError: If the ``openai-agents`` package isn't
        installed — the inner executor's ``_ensure_agents_sdk``
        surfaces this as a clear ImportError on first
        :meth:`run_turn` call.
    :raises OSError: If gateway auth is configured without a gateway base URL.
    """
    api_key = os.environ.get(_ENV_API_KEY) or None
    model = os.environ.get(_ENV_MODEL) or None
    use_responses = _parse_truthy(
        _ENV_USE_RESPONSES,
        default=True,
    )
    return OpenAIAgentsSDKExecutor(
        api_key=api_key,
        use_responses=use_responses,
        model=model,
        base_url_override=os.environ.get(_ENV_GATEWAY_BASE_URL) or None,
        gateway_auth_command=os.environ.get(_ENV_GATEWAY_AUTH_COMMAND) or None,
    )


def create_app() -> FastAPI:
    """
    Build the openai-agents-sdk harness's FastAPI app.

    Required entry point per the harness contract — the runner
    imports this module (resolved from
    :data:`goalrail.runtime.harnesses._HARNESS_MODULES`) and
    invokes ``create_app()`` to get the app it serves.

    :returns: The FastAPI app from :class:`ExecutorAdapter`'s
        :meth:`build` method, with all routes from the harness
        API subset wired up. The wrapped
        :class:`OpenAIAgentsSDKExecutor` is constructed lazily
        on the first turn (so an absent ``openai-agents``
        package surfaces as a request-time error, not a FastAPI
        app-boot crash).
    """
    adapter = ExecutorAdapter(executor_factory=_build_openai_agents_sdk_executor)
    return adapter.build()
