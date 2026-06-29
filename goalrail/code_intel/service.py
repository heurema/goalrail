"""Shared code-intel invocation + response shaping.

One place that turns a resolved repository root into the JSON envelopes
the API returns. Both the local server route and the future host-side
handler call these, so the wire shape can't drift between "local" and
"remote" code intelligence.

Repository *resolution* stays caller-side on purpose: the local route
resolves against the server filesystem, while the host handler resolves
against the host filesystem (it is the source of truth for ``~`` /
symlinks, like ``host.stat``). Everything downstream of a resolved
``repo_root`` — running the engine, mapping errors to states, building
the dict — lives here.

These functions are synchronous (they shell out to the engine via
:class:`CodeIntelClient`); async callers wrap them in
``asyncio.to_thread``.
"""

from __future__ import annotations

from pathlib import Path
from typing import Any, Protocol, TypeAlias

from goalrail.code_intel.client import (
    CodeIntelNotIndexedError,
    CodeIntelNotInstalledError,
    CodeIntelTimeoutError,
    IndexStatus,
    SearchResults,
)

# Hard cap on hits returned to a UI client, mirroring the code_search
# builtin. Kept here so the route and the host handler clamp identically.
SEARCH_LIMIT_DEFAULT = 20
SEARCH_LIMIT_MAX = 50

# JSON envelope shape returned to API/host callers. Free-form per the
# engine payload, so this is the one place we accept ``Any`` (documented
# escape hatch, matching ``goalrail.code_intel.client``).
_Json: TypeAlias = dict[str, Any]  # type: ignore[explicit-any]  # JSON envelope

# Status reported for host-bound sessions while server-side execution is
# the only path. The server must never read a host workspace path as one
# of its own; this honest state lets the UI keep the Code tab visible.
HOST_UNSUPPORTED_STATUS = "host_unsupported"
HOST_UNSUPPORTED_MESSAGE = "Code intelligence is not available for host workspaces yet."


class _Engine(Protocol):
    """The subset of :class:`CodeIntelClient` the service needs."""

    def index_status(self, repo_root: Path) -> IndexStatus: ...

    def search(
        self, repo_root: Path, query: str, *, limit: int = ..., label: str | None = ...
    ) -> SearchResults: ...


def clamp_search_limit(limit: int) -> int:
    """Clamp a requested hit limit into the allowed range."""
    return min(limit, SEARCH_LIMIT_MAX) if limit > 0 else SEARCH_LIMIT_DEFAULT


def head_info(raw: _Json) -> _Json | None:
    """Extract a compact head/freshness block from an engine git envelope.

    :param raw: The full ``index_status`` engine response.
    :returns: ``{branch, head_sha, base_sha}`` when git info is present,
        else ``None``.
    """
    git = raw.get("git")
    if not isinstance(git, dict):
        return None
    return {
        "branch": git.get("branch"),
        "head_sha": git.get("head_sha"),
        "base_sha": git.get("base_sha"),
    }


def status_envelope(engine: _Engine, repo_root: Path) -> _Json:
    """Run ``index_status`` and shape it into the status envelope.

    Maps a missing/timed-out engine to the ``engine_unavailable`` state
    rather than raising, so the caller can return HTTP 200 and the UI can
    render it.
    """
    try:
        status = engine.index_status(repo_root)
    except (CodeIntelNotInstalledError, CodeIntelTimeoutError) as exc:
        return {
            "repo_root": str(repo_root),
            "indexed": False,
            "status": "engine_unavailable",
            "nodes": None,
            "edges": None,
            "head": None,
            "project": None,
            "message": str(exc),
        }
    return {
        "repo_root": status.repo_root,
        "indexed": status.status == "ready",
        "status": status.status,
        "nodes": status.nodes,
        "edges": status.edges,
        "head": head_info(status.raw),
        "project": status.project,
        "message": None,
    }


def search_envelope(engine: _Engine, repo_root: Path, query: str, limit: int) -> _Json:
    """Run ``search`` and shape it into the search envelope.

    ``not_indexed`` and ``engine_unavailable`` come back as states, not
    exceptions. ``query`` is assumed non-empty (callers validate).
    """
    try:
        results = engine.search(repo_root, query, limit=limit)
    except CodeIntelNotIndexedError as exc:
        return {
            "repo_root": str(repo_root),
            "query": query,
            "status": "not_indexed",
            "total": 0,
            "results": [],
            "message": str(exc),
        }
    except (CodeIntelNotInstalledError, CodeIntelTimeoutError) as exc:
        return {
            "repo_root": str(repo_root),
            "query": query,
            "status": "engine_unavailable",
            "total": 0,
            "results": [],
            "message": str(exc),
        }
    return {
        "repo_root": results.repo_root,
        "query": results.query,
        "status": "ok",
        "total": results.total,
        "results": [
            {
                "name": hit.name,
                "qualified_name": hit.qualified_name,
                "label": hit.label,
                "file": hit.file,
                "signature": hit.signature,
                "return_type": hit.return_type,
            }
            for hit in results.hits
        ],
        "message": None,
    }


def host_unsupported_status_envelope() -> _Json:
    """Status envelope for a host-bound session (no FS read)."""
    return {
        "repo_root": "",
        "indexed": False,
        "status": HOST_UNSUPPORTED_STATUS,
        "nodes": None,
        "edges": None,
        "head": None,
        "project": None,
        "message": HOST_UNSUPPORTED_MESSAGE,
    }


def host_unsupported_search_envelope(query: str) -> _Json:
    """Search envelope for a host-bound session (no FS read)."""
    return {
        "repo_root": "",
        "query": query,
        "status": HOST_UNSUPPORTED_STATUS,
        "total": 0,
        "results": [],
        "message": HOST_UNSUPPORTED_MESSAGE,
    }
