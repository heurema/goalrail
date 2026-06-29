"""Session-scoped code-intelligence routes.

Exposes the native :class:`~goalrail.code_intel.CodeIntelClient` to the
web UI without inventing a second logic layer: for local sessions, the
repository is resolved **server-side** from the session's stored workspace
(the canonical realpath the runner cd's into), never passed by the client.
Host-bound sessions never read a path on the server: status/search report
an honest ``host_unsupported`` state (the Code tab stays visible) and the
file-content route is rejected, until code-intel can execute on the owning
host (Phase 2: host frame RPC).

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
    resolve_repo_root,
    service,
)
from goalrail.entities import Conversation
from goalrail.errors import ErrorCode, GoalrailError
from goalrail.server.auth import LEVEL_READ, AuthProvider
from goalrail.server.routes._auth_helpers import get_user_id, require_access
from goalrail.stores import ConversationStore
from goalrail.stores.permission_store import PermissionStore
from goalrail.tools.base import ToolContext

# Cap on file-preview bytes returned to the UI. Status/search shaping and
# limits live in ``goalrail.code_intel.service`` (shared with the future
# host handler).
_FILE_READ_LIMIT_BYTES = 256 * 1024


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

    async def _load_session(session_id: str, request: Request) -> Conversation:
        """Authorize the caller and load the session.

        Does not touch the filesystem — callers decide whether the
        session is local (resolve a server-side repo root) or host-bound
        (report the unsupported state without reading any path).

        :raises GoalrailError: 401/403 on access denial, 404 if unknown.
        """
        user_id = get_user_id(request, auth_provider)
        if permission_store is not None:
            await require_access(
                user_id, session_id, LEVEL_READ, permission_store, conversation_store
            )
        conv = await asyncio.to_thread(conversation_store.get_conversation, session_id)
        if conv is None:
            raise GoalrailError("Session not found", code=ErrorCode.NOT_FOUND)
        return conv

    def _resolve_local_repo(conv: Conversation) -> Path:
        """Resolve the server-side repo root for a *local* session.

        Host-bound sessions are rejected here as defense-in-depth: the
        server must never resolve or stat a host workspace path. Callers
        that want a graceful UI state check ``conv.host_id`` first and
        return the unsupported envelope instead of calling this.

        :raises GoalrailError: 409 for host-bound or workspace-less
            sessions.
        """
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
            ``"not_indexed"``, ``"engine_unavailable"``, or
            ``"host_unsupported"``.
        """
        conv = await _load_session(session_id, request)
        if conv.host_id:
            return service.host_unsupported_status_envelope()
        repo_root = _resolve_local_repo(conv)
        return await asyncio.to_thread(service.status_envelope, engine, repo_root)

    @router.get("/sessions/{session_id}/code-intel/search")
    async def code_intel_search(
        request: Request,
        session_id: str,
        q: str,
        limit: int = service.SEARCH_LIMIT_DEFAULT,
    ) -> dict[str, Any]:
        """Search the session repo's knowledge graph for symbols.

        :param q: Name pattern to match against symbol names.
        :param limit: Max hits (clamped to ``service.SEARCH_LIMIT_MAX``).
        :returns: ``{repo_root, query, status, total, results, message}``.
            ``status`` is ``"ok"``, ``"not_indexed"``,
            ``"engine_unavailable"``, or ``"host_unsupported"``.
        """
        conv = await _load_session(session_id, request)
        if conv.host_id:
            return service.host_unsupported_search_envelope(q)
        repo_root = _resolve_local_repo(conv)
        if not q.strip():
            raise GoalrailError("q is required", code=ErrorCode.INVALID_INPUT)
        capped = service.clamp_search_limit(limit)
        return await asyncio.to_thread(service.search_envelope, engine, repo_root, q, capped)

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
        that are not currently bound to a runner. Host-bound sessions are
        rejected — the server must never read a host workspace path.
        """
        conv = await _load_session(session_id, request)
        repo_root = _resolve_local_repo(conv)
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
