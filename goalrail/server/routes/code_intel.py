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
from goalrail.code_intel.file_preview import read_repo_file
from goalrail.entities import Conversation
from goalrail.errors import ErrorCode, GoalrailError
from goalrail.host.frames import (
    HOST_FEATURE_CODE_INTEL_FILE,
    HOST_FEATURE_CODE_INTEL_SEARCH,
    HOST_FEATURE_CODE_INTEL_STATUS,
)
from goalrail.server.auth import LEVEL_READ, AuthProvider
from goalrail.server.host_registry import (
    HostCodeIntelError,
    HostRegistry,
    request_code_intel_file,
    request_code_intel_search,
    request_code_intel_status,
)
from goalrail.server.routes._auth_helpers import get_user_id, require_access
from goalrail.stores import ConversationStore
from goalrail.stores.permission_store import PermissionStore
from goalrail.tools.base import ToolContext


def create_code_intel_router(
    conversation_store: ConversationStore,
    auth_provider: AuthProvider | None = None,
    permission_store: PermissionStore | None = None,
    client: CodeIntelClient | None = None,
    host_registry: HostRegistry | None = None,
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
    :param host_registry: Live host connections. When provided, status,
        search, and file preview for a host-bound session are fetched from
        the owning host via code-intel host frames;
        without it (or when the host is offline / lacks the feature /
        is unresponsive) the route falls back to the ``host_unsupported``
        envelope.
    :returns: A configured :class:`APIRouter`.
    """
    router = APIRouter()
    engine = client or CodeIntelClient()

    def _host_supports(host_conn: object, feature: str) -> bool:
        """Return whether a connected host advertised a code-intel feature."""
        hello = getattr(host_conn, "hello", None)
        features = getattr(hello, "features", None)
        return isinstance(features, dict) and features.get(feature) is True

    async def _host_status_envelope(conv: Conversation) -> dict[str, Any]:
        """Fetch status from the owning host, or fall back to unsupported.

        The server never reads the host workspace path; it asks the host
        (which owns resolution) over the tunnel. Any failure — no
        registry, host offline, timeout, host-side error — degrades to
        the honest ``host_unsupported`` envelope.
        """
        if host_registry is None or not conv.workspace:
            return service.host_unsupported_status_envelope()
        host_conn = host_registry.get(conv.host_id) if conv.host_id else None
        if host_conn is None:
            return service.host_unsupported_status_envelope()
        if not _host_supports(host_conn, HOST_FEATURE_CODE_INTEL_STATUS):
            return service.host_unsupported_status_envelope()
        try:
            return await request_code_intel_status(host_registry, host_conn, conv.workspace)
        except HostCodeIntelError:
            return service.host_unsupported_status_envelope()

    async def _host_search_envelope(conv: Conversation, query: str, limit: int) -> dict[str, Any]:
        """Fetch search results from the owning host, or fall back.

        Mirrors :func:`_host_status_envelope`: no registry, host offline,
        missing feature, timeout, or host-side error all degrade to the
        honest ``host_unsupported`` search envelope.
        """
        if host_registry is None or not conv.workspace:
            return service.host_unsupported_search_envelope(query)
        host_conn = host_registry.get(conv.host_id) if conv.host_id else None
        if host_conn is None:
            return service.host_unsupported_search_envelope(query)
        if not _host_supports(host_conn, HOST_FEATURE_CODE_INTEL_SEARCH):
            return service.host_unsupported_search_envelope(query)
        try:
            return await request_code_intel_search(
                host_registry, host_conn, conv.workspace, query, limit
            )
        except HostCodeIntelError:
            return service.host_unsupported_search_envelope(query)

    async def _host_file_payload(conv: Conversation, path: str) -> dict[str, Any]:
        """Fetch a file preview from the owning host.

        The server never resolves the host workspace path. Host-side failures
        surface as a conflict so the UI sees a failed preview instead of a
        misleading server-filesystem result.
        """
        if host_registry is None or not conv.workspace:
            raise GoalrailError(
                "Code intelligence file preview is unavailable for this host",
                code=ErrorCode.CONFLICT,
            )
        host_conn = host_registry.get(conv.host_id) if conv.host_id else None
        if host_conn is None or not _host_supports(host_conn, HOST_FEATURE_CODE_INTEL_FILE):
            raise GoalrailError(
                "Code intelligence file preview is unavailable for this host",
                code=ErrorCode.CONFLICT,
            )
        try:
            return await request_code_intel_file(host_registry, host_conn, conv.workspace, path)
        except HostCodeIntelError as exc:
            raise GoalrailError(
                "Code intelligence file preview is unavailable for this host",
                code=ErrorCode.CONFLICT,
            ) from exc

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
            return await _host_status_envelope(conv)
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
        :param limit: Max hits, clamped to the configured search limit.
        :returns: ``{repo_root, query, status, total, results, message}``.
            ``status`` is ``"ok"``, ``"not_indexed"``,
            ``"engine_unavailable"``, or ``"host_unsupported"``.
        """
        conv = await _load_session(session_id, request)
        if not q.strip():
            raise GoalrailError("q is required", code=ErrorCode.INVALID_INPUT)
        capped = service.clamp_search_limit(limit)
        if conv.host_id:
            return await _host_search_envelope(conv, q, capped)
        repo_root = _resolve_local_repo(conv)
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
        served by the owning host over the host tunnel; the server must
        never read a host workspace path.
        """
        conv = await _load_session(session_id, request)
        if conv.host_id:
            return await _host_file_payload(conv, path)
        repo_root = _resolve_local_repo(conv)
        return read_repo_file(repo_root, path)

    return router
