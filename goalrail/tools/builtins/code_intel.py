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
    CodeIntelError,
    CodeIntelNotIndexedError,
    CodeIntelNotInstalledError,
    CodeIntelProtocolError,
    CodeIntelTimeoutError,
    CodeIntelToolError,
    RepoBoundaryError,
    resolve_repo_root,
)
from goalrail.tools.base import Tool, ToolContext

# Cap on hits returned to the agent — keeps responses token-light.
_SEARCH_LIMIT_DEFAULT = 20
_SEARCH_LIMIT_MAX = 50


def _error_payload(exc: CodeIntelError) -> dict[str, Any]:
    """Map a code-intel exception to a structured agent-facing payload.

    Shared by all code-intel tools so a misconfigured engine or an
    unindexed repo surfaces an actionable ``{"error": ...}`` object
    instead of crashing the turn.
    """
    if isinstance(exc, CodeIntelNotInstalledError):
        return {"error": "code_intel_not_installed", "message": str(exc)}
    if isinstance(exc, CodeIntelTimeoutError):
        return {"error": "timeout", "message": str(exc)}
    if isinstance(exc, CodeIntelNotIndexedError):
        return {"error": "not_indexed", "message": str(exc)}
    if isinstance(exc, CodeIntelToolError):
        return {"error": "engine_error", "message": exc.message, "code": exc.code}
    if isinstance(exc, RepoBoundaryError):
        return {"error": "boundary_error", "message": str(exc)}
    if isinstance(exc, CodeIntelProtocolError):
        return {"error": "protocol_error", "message": str(exc)}
    return {"error": "code_intel_error", "message": str(exc)}


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
        except CodeIntelError as exc:
            return json.dumps(_error_payload(exc))

        payload = asdict(status)
        # ``raw`` is the full engine envelope; drop it from the
        # agent-facing result to keep the response compact.
        payload.pop("raw", None)
        return json.dumps(payload)


_SEARCH_DESCRIPTION = (
    "Search the code-intelligence knowledge graph for symbols "
    "(functions, classes, methods) whose name matches a pattern, in the "
    "repository the current session is working in. Returns each match's "
    "qualified_name, kind, file, and signature — use the qualified_name "
    "for follow-up lookups. Read-only. You do not pass a path — the "
    "repository is resolved automatically from the session."
)


class CodeSearchTool(Tool):
    """Search the session repository's knowledge graph by symbol name."""

    @classmethod
    def name(cls) -> str:
        """:returns: ``"code_search"``."""
        return "code_search"

    @classmethod
    def description(cls) -> str:
        """:returns: Human-readable description of the tool."""
        return _SEARCH_DESCRIPTION

    def get_schema(self) -> dict[str, Any]:
        """:returns: The OpenAI-format tool schema."""
        return {
            "type": "function",
            "function": {
                "name": self.name(),
                "description": _SEARCH_DESCRIPTION,
                "parameters": {
                    "type": "object",
                    "properties": {
                        "query": {
                            "type": "string",
                            "description": (
                                "Name pattern to match against symbol names, "
                                "e.g. 'resolve_repo_root' or 'Tool'."
                            ),
                        },
                        "limit": {
                            "type": "integer",
                            "description": (
                                f"Max hits to return. Default "
                                f"{_SEARCH_LIMIT_DEFAULT}, max {_SEARCH_LIMIT_MAX}."
                            ),
                        },
                        "label": {
                            "type": "string",
                            "description": (
                                "Optional node-kind filter, e.g. 'Function', 'Class', 'Method'."
                            ),
                        },
                    },
                    "required": ["query"],
                },
            },
        }

    def invoke(self, arguments: str, ctx: ToolContext) -> str:
        """Resolve the repo server-side and search the knowledge graph.

        Always returns a JSON string; operational failures (binary not
        installed, not indexed, timeout, engine/protocol errors) come
        back as a structured ``{"error": ...}`` payload.

        :param arguments: JSON with ``query`` (required), optional
            ``limit`` and ``label``.
        :param ctx: Server-side execution context (provides sandbox root).
        :returns: JSON string with search results or an error payload.
        """
        args: dict[str, Any] = json.loads(arguments) if arguments else {}
        query = args.get("query")
        if not isinstance(query, str) or not query.strip():
            return json.dumps({"error": "invalid_arguments", "message": "query is required"})
        limit = args.get("limit", _SEARCH_LIMIT_DEFAULT)
        if not isinstance(limit, int) or limit <= 0:
            limit = _SEARCH_LIMIT_DEFAULT
        limit = min(limit, _SEARCH_LIMIT_MAX)
        label = args.get("label") if isinstance(args.get("label"), str) else None

        client = CodeIntelClient()
        try:
            repo_root = resolve_repo_root(ctx)
            results = client.search(repo_root, query, limit=limit, label=label)
        except CodeIntelError as exc:
            return json.dumps(_error_payload(exc))

        return json.dumps(
            {
                "repo_root": results.repo_root,
                "query": results.query,
                "total": results.total,
                "results": [asdict(hit) for hit in results.hits],
            }
        )
