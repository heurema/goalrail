"""Session-scoped code-intelligence routes.

Exposes the native :class:`~goalrail.code_intel.CodeIntelClient` to the
web UI without inventing a second logic layer: for local sessions, the
repository is resolved **server-side** from the session's stored workspace
(the canonical realpath the runner cd's into), never passed by the client.
Host-bound sessions are deliberately rejected until code-intel can execute
and preview files on the owning runner/host.

Routes (all under ``/v1``):

* ``GET /sessions/{id}/code-intel/status`` — index status + head/git info
* ``GET /sessions/{id}/code-intel/search?q=...`` — symbol search
* ``GET /sessions/{id}/code-intel/files/{path}`` — read-only repo file content

Both require ``LEVEL_READ`` on the session in multi-user mode and always
return HTTP 200 with a small status envelope (``not_indexed`` and
``engine_unavailable`` are reported as states the UI renders, not as
transport errors). Genuine access/lookup failures still raise
:class:`GoalrailError`.
"""

from __future__ import annotations

import asyncio
from pathlib import Path
from typing import Any

from fastapi import APIRouter, Request

from goalrail.code_intel import (
    CodeIntelClient,
    CodeIntelNotIndexedError,
    CodeIntelNotInstalledError,
    CodeIntelTimeoutError,
    resolve_repo_root,
)
from goalrail.errors import ErrorCode, GoalrailError
from goalrail.server.auth import LEVEL_READ, AuthProvider
from goalrail.server.routes._auth_helpers import get_user_id, require_access
from goalrail.stores import ConversationStore
from goalrail.stores.permission_store import PermissionStore
from goalrail.tools.base import ToolContext

# Hard cap on hits returned to the UI, mirroring the code_search builtin.
_SEARCH_LIMIT_DEFAULT = 20
_SEARCH_LIMIT_MAX = 50
_FILE_READ_LIMIT_BYTES = 256 * 1024


def _head_info(raw: dict[str, Any]) -> dict[str, Any] | None:
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


def _resolve_repo_file(repo_root: Path, path: str) -> Path:
    """Resolve a repo-relative file path without allowing workspace escape.

    :param repo_root: Canonical repository root.
    :param path: Repo-relative file path from the URL.
    :returns: Absolute resolved path inside ``repo_root``.
    :raises GoalrailError: 400 for invalid/escaping paths, 404 for
        missing files.
    """
    rel = Path(path)
    if not path.strip() or rel.is_absolute():
        raise GoalrailError("path must be repo-relative", code=ErrorCode.INVALID_INPUT)
    root = repo_root.resolve()
    candidate = (root / rel).resolve()
    if not candidate.is_relative_to(root):
        raise GoalrailError("path escapes repository root", code=ErrorCode.INVALID_INPUT)
    if not candidate.exists():
        raise GoalrailError("File not found", code=ErrorCode.NOT_FOUND)
    if not candidate.is_file():
        raise GoalrailError("path is not a file", code=ErrorCode.INVALID_INPUT)
    return candidate


def create_code_intel_router(
    conversation_store: ConversationStore,
    auth_provider: AuthProvider | None = None,
    permission_store: PermissionStore | None = None,
    client: CodeIntelClient | None = None,
) -> APIRouter:
    """Build the code-intelligence router.

    :param conversation_store: Used to read the session's canonical
        ``workspace`` (the server-side repo root).
    :param auth_provider: Identifies the requesting user. ``None`` in
        single-user mode.
    :param permission_store: Enforces session-level access. ``None``
        disables permission checks (single-user mode).
    :param client: Override for the engine client (tests). Defaults to a
        process-wide :class:`CodeIntelClient`.
    :returns: A configured :class:`APIRouter`.
    """
    router = APIRouter()
    engine = client or CodeIntelClient()

    async def _resolve_repo(session_id: str, request: Request) -> Path:
        """Authorize the caller and resolve the session's repo root.

        :raises GoalrailError: 401/403 on access denial, 404 if the
            session is unknown, 409 if it has no workspace recorded.
        """
        user_id = get_user_id(request, auth_provider)
        if permission_store is not None:
            await require_access(
                user_id, session_id, LEVEL_READ, permission_store, conversation_store
            )
        conv = await asyncio.to_thread(conversation_store.get_conversation, session_id)
        if conv is None:
            raise GoalrailError("Session not found", code=ErrorCode.NOT_FOUND)
        if conv.host_id:
            raise GoalrailError(
                "Code intelligence is only available for local server workspaces",
                code=ErrorCode.CONFLICT,
            )
        if not conv.workspace:
            raise GoalrailError(
                "Session has no workspace on disk to index",
                code=ErrorCode.CONFLICT,
            )
        # The agent never supplies a path; the boundary is the session's
        # canonical workspace. resolve_repo_root re-canonicalizes and will
        # not climb above it.
        ctx = ToolContext(task_id="", agent_id="", sandbox_root=Path(conv.workspace))
        return resolve_repo_root(ctx)

    @router.get("/sessions/{session_id}/code-intel/status")
    async def code_intel_status(request: Request, session_id: str) -> dict[str, Any]:
        """Return the index status and head info for the session's repo.

        :returns: ``{repo_root, indexed, status, nodes, edges, head,
            project, message}``. ``status`` is ``"ready"``,
            ``"not_indexed"``, or ``"engine_unavailable"``.
        """
        repo_root = await _resolve_repo(session_id, request)
        try:
            status = await asyncio.to_thread(engine.index_status, repo_root)
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
            "head": _head_info(status.raw),
            "project": status.project,
            "message": None,
        }

    @router.get("/sessions/{session_id}/code-intel/search")
    async def code_intel_search(
        request: Request,
        session_id: str,
        q: str,
        limit: int = _SEARCH_LIMIT_DEFAULT,
    ) -> dict[str, Any]:
        """Search the session repo's knowledge graph for symbols.

        :param q: Name pattern to match against symbol names.
        :param limit: Max hits (clamped to ``_SEARCH_LIMIT_MAX``).
        :returns: ``{repo_root, query, status, total, results, message}``.
            ``status`` is ``"ok"``, ``"not_indexed"``, or
            ``"engine_unavailable"``.
        """
        repo_root = await _resolve_repo(session_id, request)
        if not q.strip():
            raise GoalrailError("q is required", code=ErrorCode.INVALID_INPUT)
        capped = min(limit, _SEARCH_LIMIT_MAX) if limit > 0 else _SEARCH_LIMIT_DEFAULT
        try:
            results = await asyncio.to_thread(engine.search, repo_root, q, limit=capped)
        except CodeIntelNotIndexedError as exc:
            return {
                "repo_root": str(repo_root),
                "query": q,
                "status": "not_indexed",
                "total": 0,
                "results": [],
                "message": str(exc),
            }
        except (CodeIntelNotInstalledError, CodeIntelTimeoutError) as exc:
            return {
                "repo_root": str(repo_root),
                "query": q,
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

    @router.get("/sessions/{session_id}/code-intel/files/{path:path}")
    async def code_intel_file(
        request: Request,
        session_id: str,
        path: str,
    ) -> dict[str, Any]:
        """Return read-only source text for a repo-relative path.

        This deliberately uses the same session workspace boundary as
        status/search instead of the runner filesystem resource API. The
        Code tab can therefore preview indexed files for local sessions
        that are not currently bound to a runner.
        """
        repo_root = await _resolve_repo(session_id, request)
        file_path = _resolve_repo_file(repo_root, path)
        size = file_path.stat().st_size
        with file_path.open("rb") as fh:
            raw = fh.read(_FILE_READ_LIMIT_BYTES + 1)
        truncated = len(raw) > _FILE_READ_LIMIT_BYTES
        if truncated:
            raw = raw[:_FILE_READ_LIMIT_BYTES]
        return {
            "repo_root": str(repo_root),
            "path": str(file_path.relative_to(repo_root.resolve())),
            "size_bytes": size,
            "truncated": truncated,
            "content": raw.decode("utf-8", errors="replace"),
        }

    return router
