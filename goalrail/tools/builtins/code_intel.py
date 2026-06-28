"""Built-in tools backed by the code-intel-memory engine.

These tools give the agent native code intelligence without exposing
the engine's CLI, MCP protocol, or the on-disk index store. The target
repository is resolved **server-side** from the session context — the
agent never passes a filesystem path.

First slice: ``code_index_status`` (read-only). ``code_search`` and the
snippet/trace family follow on the same client.
"""

from __future__ import annotations

import json
from dataclasses import asdict
from typing import Any

from goalrail.code_intel import (
    CodeIntelClient,
    CodeIntelNotInstalledError,
    CodeIntelProtocolError,
    CodeIntelTimeoutError,
    CodeIntelToolError,
    RepoBoundaryError,
    resolve_repo_root,
)
from goalrail.tools.base import Tool, ToolContext

_DESCRIPTION = (
    "Report the code-intelligence index status for the repository the "
    "current session is working in. Tells you whether the codebase is "
    "indexed and queryable (status 'ready'), not yet indexed "
    "('not_indexed'), and the knowledge-graph node/edge counts. "
    "Read-only. You do not pass a path — the repository is resolved "
    "automatically from the session."
)


class CodeIndexStatusTool(Tool):
    """Report code-intel index status for the session's repository."""

    @classmethod
    def name(cls) -> str:
        """:returns: ``"code_index_status"``."""
        return "code_index_status"

    @classmethod
    def description(cls) -> str:
        """:returns: Human-readable description of the tool."""
        return _DESCRIPTION

    def get_schema(self) -> dict[str, Any]:
        """:returns: The OpenAI-format tool schema (no parameters)."""
        return {
            "type": "function",
            "function": {
                "name": self.name(),
                "description": _DESCRIPTION,
                "parameters": {
                    "type": "object",
                    "properties": {},
                    "required": [],
                },
            },
        }

    def invoke(self, arguments: str, ctx: ToolContext) -> str:
        """Resolve the repo server-side and return its index status.

        Always returns a JSON string. Operational failures (binary not
        installed, timeout, protocol/engine errors) are returned as a
        structured ``{"error": ...}`` payload rather than raised, so a
        misconfigured engine surfaces an actionable message instead of
        crashing the turn.

        :param arguments: JSON args from the LLM (none required).
        :param ctx: Server-side execution context (provides workspace).
        :returns: JSON string with the index status or an error payload.
        """
        del arguments  # no parameters

        client = CodeIntelClient()
        try:
            repo_root = resolve_repo_root(ctx)
            status = client.index_status(repo_root)
        except CodeIntelNotInstalledError as exc:
            return json.dumps(
                {
                    "error": "code_intel_not_installed",
                    "message": str(exc),
                }
            )
        except CodeIntelTimeoutError as exc:
            return json.dumps({"error": "timeout", "message": str(exc)})
        except CodeIntelToolError as exc:
            return json.dumps(
                {
                    "error": "engine_error",
                    "message": exc.message,
                    "code": exc.code,
                }
            )
        except (CodeIntelProtocolError, RepoBoundaryError) as exc:
            return json.dumps(
                {
                    "error": type(exc).__name__,
                    "message": str(exc),
                }
            )

        payload = asdict(status)
        # ``raw`` is the full engine envelope; drop it from the
        # agent-facing result to keep the response compact.
        payload.pop("raw", None)
        return json.dumps(payload)
