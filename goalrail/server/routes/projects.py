"""Read-only project list routes.

The first ``/projects`` web page is intentionally a project picker. The
backend mirrors that product boundary by exposing only a minimal read model
derived from existing sessions: one project per workspace, newest activity
first, with no people, cost, health, or session-count aggregates.
"""

from __future__ import annotations

import asyncio
import hashlib
import posixpath

from fastapi import APIRouter, Request

from goalrail.server.auth import AuthProvider
from goalrail.server.routes._auth_helpers import require_user
from goalrail.server.schemas import ProjectList, ProjectListItem
from goalrail.stores import ConversationStore

_PROJECTS_SCAN_PAGE_SIZE = 1000
_PROJECT_ID_PREFIX = "proj_"
_PROJECT_ID_HASH_LEN = 24


def _normalize_workspace(workspace: str) -> str:
    """Return a stable grouping key for a stored workspace path."""

    return posixpath.normpath(workspace.strip().replace("\\", "/"))


def _project_id(normalized_workspace: str) -> str:
    """Build an opaque, stable id from the normalized workspace key."""

    digest = hashlib.sha256(normalized_workspace.encode("utf-8")).hexdigest()
    return f"{_PROJECT_ID_PREFIX}{digest[:_PROJECT_ID_HASH_LEN]}"


def _project_name(workspace: str, normalized_workspace: str) -> str:
    """Derive a display name from the final path segment."""

    candidate = normalized_workspace.rstrip("/").rsplit("/", 1)[-1]
    if candidate and candidate != ".":
        return candidate
    return workspace


def create_projects_router(
    conversation_store: ConversationStore,
    auth_provider: AuthProvider | None = None,
) -> APIRouter:
    """Build the read-only projects router.

    :param conversation_store: Source of session rows to aggregate.
    :param auth_provider: Optional auth provider. When set, unauthenticated
        callers are rejected and the session list is scoped to that user,
        matching ``GET /v1/sessions``.
    :returns: A configured :class:`APIRouter`.
    """

    router = APIRouter()

    @router.get(
        "/projects",
        response_model=None,
        responses={200: {"model": ProjectList}},
    )
    async def list_projects(request: Request) -> ProjectList:
        """Return minimal project summaries grouped by session workspace."""

        user_id = require_user(request, auth_provider)
        after: str | None = None
        projects_by_workspace: dict[str, ProjectListItem] = {}

        while True:
            page = await asyncio.to_thread(
                conversation_store.list_conversations,
                limit=_PROJECTS_SCAN_PAGE_SIZE,
                after=after,
                has_agent_id=True,
                kind="default",
                order="desc",
                sort_by="updated_at",
                accessible_by=user_id,
                include_archived=False,
            )

            for conv in page.data:
                raw_workspace = conv.workspace
                if raw_workspace is None:
                    continue
                workspace = raw_workspace.strip()
                if not workspace:
                    continue
                normalized_workspace = _normalize_workspace(workspace)
                existing = projects_by_workspace.get(normalized_workspace)
                if existing is not None:
                    if conv.updated_at > existing.last_activity_at:
                        existing.last_activity_at = conv.updated_at
                    continue
                projects_by_workspace[normalized_workspace] = ProjectListItem(
                    id=_project_id(normalized_workspace),
                    name=_project_name(workspace, normalized_workspace),
                    workspace=workspace,
                    last_activity_at=conv.updated_at,
                )

            if not page.has_more or page.last_id is None:
                break
            after = page.last_id

        projects = sorted(
            projects_by_workspace.values(),
            key=lambda project: (project.last_activity_at, project.id),
            reverse=True,
        )
        return ProjectList(data=projects)

    return router
