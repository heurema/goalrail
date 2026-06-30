"""Tests for the minimal ``GET /v1/projects`` route."""

from __future__ import annotations

import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient

from goalrail.server.auth import LEVEL_OWNER, UnifiedAuthProvider
from goalrail.server.routes.projects import create_projects_router
from goalrail.stores.agent_store.sqlalchemy_store import SqlAlchemyAgentStore
from goalrail.stores.conversation_store.sqlalchemy_store import SqlAlchemyConversationStore
from goalrail.stores.permission_store.sqlalchemy_store import SqlAlchemyPermissionStore

ALICE = "alice@example.com"
BOB = "bob@example.com"


def _stores(
    db_uri: str,
) -> tuple[SqlAlchemyConversationStore, SqlAlchemyAgentStore, SqlAlchemyPermissionStore]:
    return (
        SqlAlchemyConversationStore(db_uri),
        SqlAlchemyAgentStore(db_uri),
        SqlAlchemyPermissionStore(db_uri),
    )


def _app(conversation_store: SqlAlchemyConversationStore) -> FastAPI:
    app = FastAPI()
    app.include_router(create_projects_router(conversation_store), prefix="/v1")
    return app


def _authed_app(conversation_store: SqlAlchemyConversationStore) -> FastAPI:
    app = FastAPI()
    app.include_router(
        create_projects_router(
            conversation_store,
            auth_provider=UnifiedAuthProvider(source="header"),
        ),
        prefix="/v1",
    )
    return app


def _ensure_agent(agent_store: SqlAlchemyAgentStore) -> None:
    if agent_store.get("ag_projects") is None:
        agent_store.create(
            agent_id="ag_projects",
            name="projects-agent",
            bundle_location="projects-agent/bundle",
        )


def _seed_session(
    stores: tuple[SqlAlchemyConversationStore, SqlAlchemyAgentStore, SqlAlchemyPermissionStore],
    *,
    workspace: str | None,
    owner: str | None = None,
    archived: bool = False,
    kind: str = "default",
) -> str:
    conversation_store, agent_store, permission_store = stores
    _ensure_agent(agent_store)
    conv = conversation_store.create_conversation(
        agent_id="ag_projects",
        workspace=workspace,
        kind=kind,
    )
    if archived:
        conversation_store.update_conversation(conv.id, archived=True)
    if owner is not None:
        permission_store.ensure_user(owner)
        permission_store.grant(owner, conv.id, LEVEL_OWNER)
    return conv.id


def test_list_projects_groups_by_workspace_and_sorts_by_last_activity(
    db_uri: str,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    stores = _stores(db_uri)
    conversation_store, _, _ = stores
    older_repo = _seed_session(stores, workspace="/Users/me/repo")
    newer_repo = _seed_session(stores, workspace="/Users/me/repo/")
    api_repo = _seed_session(stores, workspace="/Users/me/api")
    base_updated_at = max(
        conversation_store.get_conversation(sid).updated_at
        for sid in (older_repo, newer_repo, api_repo)
    )

    monkeypatch.setattr(
        "goalrail.stores.conversation_store.sqlalchemy_store.now_epoch",
        lambda: base_updated_at + 10,
    )
    conversation_store.update_conversation(api_repo, title="api")
    monkeypatch.setattr(
        "goalrail.stores.conversation_store.sqlalchemy_store.now_epoch",
        lambda: base_updated_at + 20,
    )
    conversation_store.update_conversation(older_repo, title="repo older")
    monkeypatch.setattr(
        "goalrail.stores.conversation_store.sqlalchemy_store.now_epoch",
        lambda: base_updated_at + 30,
    )
    conversation_store.update_conversation(newer_repo, title="repo newer")

    with TestClient(_app(conversation_store)) as client:
        resp = client.get("/v1/projects")

    assert resp.status_code == 200
    body = resp.json()
    assert body["object"] == "list"
    projects = body["data"]
    assert [project["workspace"] for project in projects] == ["/Users/me/repo/", "/Users/me/api"]
    assert projects[0]["name"] == "repo"
    assert projects[0]["last_activity_at"] > projects[1]["last_activity_at"]


def test_list_projects_excludes_archived_null_blank_and_child_sessions(db_uri: str) -> None:
    stores = _stores(db_uri)
    conversation_store, _, _ = stores
    _seed_session(stores, workspace="/Users/me/visible")
    _seed_session(stores, workspace="/Users/me/archived", archived=True)
    _seed_session(stores, workspace=None)
    _seed_session(stores, workspace="   ")
    _seed_session(stores, workspace="/Users/me/child", kind="sub_agent")

    with TestClient(_app(conversation_store)) as client:
        resp = client.get("/v1/projects")

    assert resp.status_code == 200
    projects = resp.json()["data"]
    assert [project["workspace"] for project in projects] == ["/Users/me/visible"]


def test_list_projects_scopes_to_authenticated_users_visible_sessions(db_uri: str) -> None:
    stores = _stores(db_uri)
    conversation_store, _, permission_store = stores
    _seed_session(stores, workspace="/Users/me/alice", owner=ALICE)
    _seed_session(stores, workspace="/Users/me/bob", owner=BOB)
    permission_store.ensure_user(ALICE)
    permission_store.ensure_user(BOB)

    with TestClient(_authed_app(conversation_store)) as client:
        resp = client.get("/v1/projects", headers={"X-Forwarded-Email": "alice@example.com"})

    assert resp.status_code == 200
    projects = resp.json()["data"]
    assert [project["workspace"] for project in projects] == ["/Users/me/alice"]


def test_list_projects_returns_minimal_fields_only(db_uri: str) -> None:
    stores = _stores(db_uri)
    conversation_store, _, _ = stores
    _seed_session(stores, workspace="/Users/me/minimal")

    with TestClient(_app(conversation_store)) as client:
        resp = client.get("/v1/projects")

    assert resp.status_code == 200
    [project] = resp.json()["data"]
    assert set(project) == {"id", "object", "name", "workspace", "last_activity_at"}
    forbidden = {
        "people",
        "contributors",
        "sessions",
        "cost",
        "awaiting_input",
        "risk",
        "verification",
        "capture",
        "trend",
        "branch",
    }
    assert forbidden.isdisjoint(project)
