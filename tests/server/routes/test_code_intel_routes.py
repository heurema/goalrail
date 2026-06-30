"""Tests for the session-scoped code-intelligence routes.

The engine client is faked so the route logic (repo resolution from the
session workspace, status/search envelopes, error→state mapping) is
exercised without a real engine binary or index. A real
``SqlAlchemyConversationStore`` provides the server-side workspace.
"""

from __future__ import annotations

from pathlib import Path
from types import SimpleNamespace

import httpx
import pytest
from fastapi import FastAPI
from fastapi.responses import JSONResponse

from goalrail.code_intel import (
    CodeIntelNotIndexedError,
    CodeIntelNotInstalledError,
    IndexStatus,
    SearchHit,
    SearchResults,
)
from goalrail.errors import GoalrailError
from goalrail.server.routes.code_intel import create_code_intel_router
from goalrail.stores.conversation_store.sqlalchemy_store import (
    SqlAlchemyConversationStore,
)
from goalrail.stores.host_store import HostStore

pytestmark = pytest.mark.asyncio


class _FakeEngine:
    """Duck-typed stand-in for CodeIntelClient with scripted responses."""

    def __init__(
        self,
        *,
        status: IndexStatus | None = None,
        results: SearchResults | None = None,
        raises: Exception | None = None,
    ) -> None:
        self._status = status
        self._results = results
        self._raises = raises
        self.last_search_limit: int | None = None
        self.index_status_calls = 0
        self.search_calls = 0

    def index_status(self, repo_root: Path) -> IndexStatus:
        self.index_status_calls += 1
        if self._raises is not None:
            raise self._raises
        assert self._status is not None
        return self._status

    def search(
        self, repo_root: Path, query: str, *, limit: int = 20, label: str | None = None
    ) -> SearchResults:
        self.search_calls += 1
        self.last_search_limit = limit
        if self._raises is not None:
            raise self._raises
        assert self._results is not None
        return self._results


def _build_app(
    conv_store: SqlAlchemyConversationStore,
    engine: _FakeEngine,
    host_registry: object | None = None,
) -> FastAPI:
    """Mount the code-intel router with a minimal GoalrailError handler."""
    app = FastAPI()

    @app.exception_handler(GoalrailError)
    async def _handle(request: httpx.Request, exc: GoalrailError) -> JSONResponse:
        return JSONResponse(
            status_code=exc.http_status,
            content={"error": {"code": exc.code, "message": exc.message}},
        )

    app.include_router(
        create_code_intel_router(
            conv_store,
            client=engine,  # type: ignore[arg-type]
            host_registry=host_registry,  # type: ignore[arg-type]
        ),
        prefix="/v1",
    )
    return app


class _FakeHostRegistry:
    """Minimal host registry: ``get`` returns a connection or ``None``."""

    def __init__(
        self,
        *,
        online: bool,
        code_intel_file: bool | None = True,
        code_intel_status: bool | None = True,
        code_intel_search: bool | None = True,
    ) -> None:
        features: dict[str, bool] | None = None
        if (
            code_intel_file is not None
            or code_intel_status is not None
            or code_intel_search is not None
        ):
            features = {}
            if code_intel_file is not None:
                features["code_intel_file"] = code_intel_file
            if code_intel_status is not None:
                features["code_intel_status"] = code_intel_status
            if code_intel_search is not None:
                features["code_intel_search"] = code_intel_search
        self._conn = SimpleNamespace(hello=SimpleNamespace(features=features)) if online else None

    def get(self, host_id: str) -> object | None:
        return self._conn


async def _client(app: FastAPI) -> httpx.AsyncClient:
    transport = httpx.ASGITransport(app=app)
    return httpx.AsyncClient(transport=transport, base_url="http://test")


@pytest.fixture()
def conv_store(db_uri: str) -> SqlAlchemyConversationStore:
    """A real conversation store backed by the per-test database."""
    return SqlAlchemyConversationStore(db_uri)


def _session_with_workspace(conv_store: SqlAlchemyConversationStore, workspace: Path) -> str:
    """Create a conversation whose workspace is set, return its id."""
    return conv_store.create_conversation(workspace=str(workspace)).id


def _register_host(db_uri: str, host_id: str = "host_remote") -> str:
    """Create a host row so host-bound conversations satisfy the FK."""
    HostStore(db_uri).upsert_on_connect(host_id, "remote", "local")
    return host_id


# ── status ────────────────────────────────────────────────


async def test_status_ready(conv_store: SqlAlchemyConversationStore, tmp_path: Path) -> None:
    """A ready index returns indexed=True with counts and head info."""
    session_id = _session_with_workspace(conv_store, tmp_path)
    engine = _FakeEngine(
        status=IndexStatus(
            repo_root=str(tmp_path),
            status="ready",
            project="proj",
            nodes=42,
            edges=99,
            raw={"git": {"branch": "main", "head_sha": "abc", "base_sha": "def"}},
        )
    )
    async with await _client(_build_app(conv_store, engine)) as c:
        resp = await c.get(f"/v1/sessions/{session_id}/code-intel/status")
    assert resp.status_code == 200
    body = resp.json()
    assert body["indexed"] is True
    assert body["status"] == "ready"
    assert body["nodes"] == 42
    assert body["head"] == {"branch": "main", "head_sha": "abc", "base_sha": "def"}


async def test_status_not_indexed(conv_store: SqlAlchemyConversationStore, tmp_path: Path) -> None:
    """A not-indexed repo reports the state with indexed=False."""
    session_id = _session_with_workspace(conv_store, tmp_path)
    engine = _FakeEngine(status=IndexStatus(repo_root=str(tmp_path), status="not_indexed"))
    async with await _client(_build_app(conv_store, engine)) as c:
        resp = await c.get(f"/v1/sessions/{session_id}/code-intel/status")
    assert resp.status_code == 200
    body = resp.json()
    assert body["indexed"] is False
    assert body["status"] == "not_indexed"
    assert body["head"] is None


async def test_status_engine_unavailable(
    conv_store: SqlAlchemyConversationStore, tmp_path: Path
) -> None:
    """A missing engine binary maps to the engine_unavailable state."""
    session_id = _session_with_workspace(conv_store, tmp_path)
    engine = _FakeEngine(raises=CodeIntelNotInstalledError("not installed"))
    async with await _client(_build_app(conv_store, engine)) as c:
        resp = await c.get(f"/v1/sessions/{session_id}/code-intel/status")
    assert resp.status_code == 200
    assert resp.json()["status"] == "engine_unavailable"


async def test_status_no_workspace_conflicts(
    conv_store: SqlAlchemyConversationStore,
) -> None:
    """A session without a workspace yields 409."""
    session_id = conv_store.create_conversation().id
    engine = _FakeEngine(status=IndexStatus(repo_root="/x", status="ready"))
    async with await _client(_build_app(conv_store, engine)) as c:
        resp = await c.get(f"/v1/sessions/{session_id}/code-intel/status")
    assert resp.status_code == 409


async def test_status_host_bound_returns_unsupported_without_fs(
    conv_store: SqlAlchemyConversationStore,
    db_uri: str,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """Host-bound status reports host_unsupported and never resolves a path.

    Monkeypatching ``resolve_repo_root`` to detonate proves the server
    does not interpret the host workspace as a local path.
    """

    def _boom(*_a: object, **_k: object) -> Path:
        raise AssertionError("server resolved a host workspace path")

    monkeypatch.setattr("goalrail.server.routes.code_intel.resolve_repo_root", _boom)
    session_id = conv_store.create_conversation(
        host_id=_register_host(db_uri),
        workspace="/host/only/repo",
    ).id
    engine = _FakeEngine(status=IndexStatus(repo_root="/x", status="ready"))

    async with await _client(_build_app(conv_store, engine)) as c:
        resp = await c.get(f"/v1/sessions/{session_id}/code-intel/status")

    assert resp.status_code == 200
    body = resp.json()
    assert body["status"] == "host_unsupported"
    assert body["indexed"] is False
    # Host workspace path must not leak back to the client.
    assert body["repo_root"] == ""
    assert engine.index_status_calls == 0


async def test_search_host_bound_returns_unsupported_without_fs(
    conv_store: SqlAlchemyConversationStore,
    db_uri: str,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """Host-bound search reports host_unsupported and never resolves a path."""

    def _boom(*_a: object, **_k: object) -> Path:
        raise AssertionError("server resolved a host workspace path")

    monkeypatch.setattr("goalrail.server.routes.code_intel.resolve_repo_root", _boom)
    session_id = conv_store.create_conversation(
        host_id=_register_host(db_uri),
        workspace="/host/only/repo",
    ).id
    engine = _FakeEngine(results=SearchResults(repo_root="/x", query="x", total=0, hits=[]))

    async with await _client(_build_app(conv_store, engine)) as c:
        resp = await c.get(f"/v1/sessions/{session_id}/code-intel/search?q=widget")

    assert resp.status_code == 200
    body = resp.json()
    assert body["status"] == "host_unsupported"
    assert body["results"] == []
    assert body["repo_root"] == ""
    assert engine.search_calls == 0


async def test_status_unknown_session_404(
    conv_store: SqlAlchemyConversationStore,
) -> None:
    """An unknown session id yields 404."""
    engine = _FakeEngine(status=IndexStatus(repo_root="/x", status="ready"))
    async with await _client(_build_app(conv_store, engine)) as c:
        resp = await c.get("/v1/sessions/conv_nope/code-intel/status")
    assert resp.status_code == 404


# ── search ────────────────────────────────────────────────


async def test_search_ok(conv_store: SqlAlchemyConversationStore, tmp_path: Path) -> None:
    """A search over an indexed repo returns mapped hits."""
    session_id = _session_with_workspace(conv_store, tmp_path)
    engine = _FakeEngine(
        results=SearchResults(
            repo_root=str(tmp_path),
            query="widget",
            total=1,
            hits=[
                SearchHit(
                    name="widget",
                    qualified_name="pkg.widget",
                    label="Function",
                    file="pkg/mod.py",
                    signature="()",
                    return_type="int",
                )
            ],
        )
    )
    async with await _client(_build_app(conv_store, engine)) as c:
        resp = await c.get(f"/v1/sessions/{session_id}/code-intel/search?q=widget")
    assert resp.status_code == 200
    body = resp.json()
    assert body["status"] == "ok"
    assert body["total"] == 1
    assert body["results"][0]["qualified_name"] == "pkg.widget"
    assert body["results"][0]["file"] == "pkg/mod.py"


async def test_search_not_indexed(conv_store: SqlAlchemyConversationStore, tmp_path: Path) -> None:
    """Searching an unindexed repo reports not_indexed with no hits."""
    session_id = _session_with_workspace(conv_store, tmp_path)
    engine = _FakeEngine(raises=CodeIntelNotIndexedError("not indexed"))
    async with await _client(_build_app(conv_store, engine)) as c:
        resp = await c.get(f"/v1/sessions/{session_id}/code-intel/search?q=widget")
    assert resp.status_code == 200
    body = resp.json()
    assert body["status"] == "not_indexed"
    assert body["results"] == []


async def test_search_requires_q(conv_store: SqlAlchemyConversationStore, tmp_path: Path) -> None:
    """A blank query is rejected with 400."""
    session_id = _session_with_workspace(conv_store, tmp_path)
    engine = _FakeEngine(
        results=SearchResults(repo_root=str(tmp_path), query="", total=0, hits=[])
    )
    async with await _client(_build_app(conv_store, engine)) as c:
        resp = await c.get(f"/v1/sessions/{session_id}/code-intel/search?q=%20")
    assert resp.status_code == 400


async def test_search_caps_limit(conv_store: SqlAlchemyConversationStore, tmp_path: Path) -> None:
    """An oversized limit is capped before reaching the engine."""
    session_id = _session_with_workspace(conv_store, tmp_path)
    engine = _FakeEngine(
        results=SearchResults(repo_root=str(tmp_path), query="x", total=0, hits=[])
    )
    async with await _client(_build_app(conv_store, engine)) as c:
        await c.get(f"/v1/sessions/{session_id}/code-intel/search?q=x&limit=999")
    assert engine.last_search_limit == 50


# ── file preview ──────────────────────────────────────────


async def test_file_preview_reads_repo_relative_file(
    conv_store: SqlAlchemyConversationStore, tmp_path: Path
) -> None:
    """The Code tab can preview files through the session workspace."""
    source = tmp_path / "pkg" / "mod.py"
    source.parent.mkdir()
    source.write_text("class Widget:\n    pass\n", encoding="utf-8")
    session_id = _session_with_workspace(conv_store, tmp_path)
    engine = _FakeEngine(status=IndexStatus(repo_root=str(tmp_path), status="ready"))

    async with await _client(_build_app(conv_store, engine)) as c:
        resp = await c.get(f"/v1/sessions/{session_id}/code-intel/files/pkg/mod.py")

    assert resp.status_code == 200
    body = resp.json()
    assert body["path"] == "pkg/mod.py"
    assert body["content"] == "class Widget:\n    pass\n"
    assert body["truncated"] is False


async def test_file_preview_rejects_repo_escape(
    conv_store: SqlAlchemyConversationStore, tmp_path: Path
) -> None:
    """Encoded ``..`` segments cannot escape the session workspace."""
    (tmp_path.parent / "secret.py").write_text("secret\n", encoding="utf-8")
    session_id = _session_with_workspace(conv_store, tmp_path)
    engine = _FakeEngine(status=IndexStatus(repo_root=str(tmp_path), status="ready"))

    async with await _client(_build_app(conv_store, engine)) as c:
        resp = await c.get(f"/v1/sessions/{session_id}/code-intel/files/%2E%2E/secret.py")

    assert resp.status_code == 400


async def test_file_preview_rejects_host_bound_workspace_before_local_read(
    conv_store: SqlAlchemyConversationStore,
    db_uri: str,
) -> None:
    """A host workspace like "/" must not become the server filesystem root."""
    session_id = conv_store.create_conversation(host_id=_register_host(db_uri), workspace="/").id
    engine = _FakeEngine(status=IndexStatus(repo_root="/", status="ready"))

    async with await _client(_build_app(conv_store, engine)) as c:
        resp = await c.get(f"/v1/sessions/{session_id}/code-intel/files/etc/passwd")

    assert resp.status_code == 409
    assert engine.index_status_calls == 0


async def test_file_preview_host_remote_returns_payload_without_server_fs(
    conv_store: SqlAlchemyConversationStore,
    db_uri: str,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """An online host previews a remote file; the server never resolves FS."""

    def _boom(*_a: object, **_k: object) -> Path:
        raise AssertionError("server resolved a host workspace path")

    async def _fake_file(registry: object, conn: object, workspace: str, path: str) -> dict:
        assert workspace == "/host/only/repo"
        assert path == "pkg/mod.py"
        return {
            "repo_root": "",
            "path": "pkg/mod.py",
            "size_bytes": 14,
            "truncated": False,
            "content": "class Widget:\n",
        }

    monkeypatch.setattr("goalrail.server.routes.code_intel.resolve_repo_root", _boom)
    monkeypatch.setattr("goalrail.server.routes.code_intel.request_code_intel_file", _fake_file)
    session_id = conv_store.create_conversation(
        host_id=_register_host(db_uri), workspace="/host/only/repo"
    ).id
    engine = _FakeEngine(status=IndexStatus(repo_root="/x", status="ready"))

    async with await _client(
        _build_app(conv_store, engine, host_registry=_FakeHostRegistry(online=True))
    ) as c:
        resp = await c.get(f"/v1/sessions/{session_id}/code-intel/files/pkg/mod.py")

    assert resp.status_code == 200
    body = resp.json()
    assert body["path"] == "pkg/mod.py"
    assert body["content"] == "class Widget:\n"
    assert engine.index_status_calls == 0


async def test_file_preview_host_without_feature_falls_back_without_rpc(
    conv_store: SqlAlchemyConversationStore,
    db_uri: str,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """A host that doesn't advertise file preview support is rejected."""

    async def _unexpected_file(registry: object, conn: object, workspace: str, path: str) -> dict:
        raise AssertionError("file RPC should be gated by host feature support")

    monkeypatch.setattr(
        "goalrail.server.routes.code_intel.request_code_intel_file",
        _unexpected_file,
    )
    session_id = conv_store.create_conversation(
        host_id=_register_host(db_uri), workspace="/host/only/repo"
    ).id
    engine = _FakeEngine(status=IndexStatus(repo_root="/x", status="ready"))
    registry = _FakeHostRegistry(online=True, code_intel_file=False)

    async with await _client(_build_app(conv_store, engine, host_registry=registry)) as c:
        resp = await c.get(f"/v1/sessions/{session_id}/code-intel/files/pkg/mod.py")

    assert resp.status_code == 409
    assert engine.index_status_calls == 0


# ── host remote status RPC (Phase 2c) ─────────────────────


async def test_status_host_remote_returns_envelope_without_server_fs(
    conv_store: SqlAlchemyConversationStore,
    db_uri: str,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """An online host returns its remote status; the server never resolves FS."""

    def _boom(*_a: object, **_k: object) -> Path:
        raise AssertionError("server resolved a host workspace path")

    async def _fake_request(registry: object, conn: object, workspace: str) -> dict:
        return {
            "repo_root": "",
            "indexed": True,
            "status": "ready",
            "nodes": 5,
            "edges": 7,
            "head": None,
            "project": "remote",
            "message": None,
        }

    monkeypatch.setattr("goalrail.server.routes.code_intel.resolve_repo_root", _boom)
    monkeypatch.setattr(
        "goalrail.server.routes.code_intel.request_code_intel_status", _fake_request
    )
    session_id = conv_store.create_conversation(
        host_id=_register_host(db_uri), workspace="/host/only/repo"
    ).id
    engine = _FakeEngine(status=IndexStatus(repo_root="/x", status="ready"))

    async with await _client(
        _build_app(conv_store, engine, host_registry=_FakeHostRegistry(online=True))
    ) as c:
        resp = await c.get(f"/v1/sessions/{session_id}/code-intel/status")

    assert resp.status_code == 200
    body = resp.json()
    assert body["status"] == "ready"
    assert body["project"] == "remote"
    assert engine.index_status_calls == 0


async def test_status_host_remote_failure_falls_back(
    conv_store: SqlAlchemyConversationStore,
    db_uri: str,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """A host RPC failure degrades to the host_unsupported envelope."""
    from goalrail.server.host_registry import HostCodeIntelError

    async def _fail(registry: object, conn: object, workspace: str) -> dict:
        raise HostCodeIntelError("host timed out")

    monkeypatch.setattr("goalrail.server.routes.code_intel.request_code_intel_status", _fail)
    session_id = conv_store.create_conversation(
        host_id=_register_host(db_uri), workspace="/host/only/repo"
    ).id
    engine = _FakeEngine(status=IndexStatus(repo_root="/x", status="ready"))

    async with await _client(
        _build_app(conv_store, engine, host_registry=_FakeHostRegistry(online=True))
    ) as c:
        resp = await c.get(f"/v1/sessions/{session_id}/code-intel/status")

    assert resp.status_code == 200
    assert resp.json()["status"] == "host_unsupported"


async def test_status_host_without_code_intel_feature_falls_back_without_rpc(
    conv_store: SqlAlchemyConversationStore,
    db_uri: str,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """A legacy/unsupported host must not receive an unknown status frame."""

    async def _unexpected_request(registry: object, conn: object, workspace: str) -> dict:
        raise AssertionError("status RPC should be gated by host feature support")

    monkeypatch.setattr(
        "goalrail.server.routes.code_intel.request_code_intel_status",
        _unexpected_request,
    )
    session_id = conv_store.create_conversation(
        host_id=_register_host(db_uri), workspace="/host/only/repo"
    ).id
    engine = _FakeEngine(status=IndexStatus(repo_root="/x", status="ready"))

    async with await _client(
        _build_app(
            conv_store,
            engine,
            host_registry=_FakeHostRegistry(online=True, code_intel_status=None),
        )
    ) as c:
        resp = await c.get(f"/v1/sessions/{session_id}/code-intel/status")

    assert resp.status_code == 200
    assert resp.json()["status"] == "host_unsupported"
    assert engine.index_status_calls == 0


async def test_status_host_offline_falls_back(
    conv_store: SqlAlchemyConversationStore,
    db_uri: str,
) -> None:
    """An offline host (no live connection) reports host_unsupported."""
    session_id = conv_store.create_conversation(
        host_id=_register_host(db_uri), workspace="/host/only/repo"
    ).id
    engine = _FakeEngine(status=IndexStatus(repo_root="/x", status="ready"))

    async with await _client(
        _build_app(conv_store, engine, host_registry=_FakeHostRegistry(online=False))
    ) as c:
        resp = await c.get(f"/v1/sessions/{session_id}/code-intel/status")

    assert resp.status_code == 200
    assert resp.json()["status"] == "host_unsupported"


# ── host remote search RPC (Phase 2d) ─────────────────────


async def test_search_host_remote_returns_envelope_without_server_fs(
    conv_store: SqlAlchemyConversationStore,
    db_uri: str,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """An online host returns remote search; the server never resolves FS."""

    def _boom(*_a: object, **_k: object) -> Path:
        raise AssertionError("server resolved a host workspace path")

    async def _fake_search(
        registry: object, conn: object, workspace: str, query: str, limit: int
    ) -> dict:
        return {
            "repo_root": "",
            "query": query,
            "status": "ok",
            "total": 1,
            "results": [
                {
                    "name": "widget",
                    "qualified_name": "pkg.widget",
                    "label": "Function",
                    "file": "pkg/mod.py",
                    "signature": None,
                    "return_type": None,
                }
            ],
            "message": None,
        }

    monkeypatch.setattr("goalrail.server.routes.code_intel.resolve_repo_root", _boom)
    monkeypatch.setattr(
        "goalrail.server.routes.code_intel.request_code_intel_search", _fake_search
    )
    session_id = conv_store.create_conversation(
        host_id=_register_host(db_uri), workspace="/host/only/repo"
    ).id
    engine = _FakeEngine(results=SearchResults(repo_root="/x", query="x", total=0, hits=[]))

    async with await _client(
        _build_app(conv_store, engine, host_registry=_FakeHostRegistry(online=True))
    ) as c:
        resp = await c.get(f"/v1/sessions/{session_id}/code-intel/search?q=widget")

    assert resp.status_code == 200
    body = resp.json()
    assert body["status"] == "ok"
    assert body["results"][0]["qualified_name"] == "pkg.widget"
    assert engine.search_calls == 0


async def test_search_host_remote_failure_falls_back(
    conv_store: SqlAlchemyConversationStore,
    db_uri: str,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """A host search RPC failure degrades to host_unsupported."""
    from goalrail.server.host_registry import HostCodeIntelError

    async def _fail(
        registry: object, conn: object, workspace: str, query: str, limit: int
    ) -> dict:
        raise HostCodeIntelError("host timed out")

    monkeypatch.setattr("goalrail.server.routes.code_intel.request_code_intel_search", _fail)
    session_id = conv_store.create_conversation(
        host_id=_register_host(db_uri), workspace="/host/only/repo"
    ).id
    engine = _FakeEngine(results=SearchResults(repo_root="/x", query="x", total=0, hits=[]))

    async with await _client(
        _build_app(conv_store, engine, host_registry=_FakeHostRegistry(online=True))
    ) as c:
        resp = await c.get(f"/v1/sessions/{session_id}/code-intel/search?q=widget")

    assert resp.status_code == 200
    assert resp.json()["status"] == "host_unsupported"


async def test_search_host_without_feature_falls_back(
    conv_store: SqlAlchemyConversationStore,
    db_uri: str,
) -> None:
    """A host that doesn't advertise search support reports host_unsupported."""
    session_id = conv_store.create_conversation(
        host_id=_register_host(db_uri), workspace="/host/only/repo"
    ).id
    engine = _FakeEngine(results=SearchResults(repo_root="/x", query="x", total=0, hits=[]))

    registry = _FakeHostRegistry(online=True, code_intel_search=False)
    async with await _client(_build_app(conv_store, engine, host_registry=registry)) as c:
        resp = await c.get(f"/v1/sessions/{session_id}/code-intel/search?q=widget")

    assert resp.status_code == 200
    assert resp.json()["status"] == "host_unsupported"


async def test_search_host_blank_query_rejected(
    conv_store: SqlAlchemyConversationStore,
    db_uri: str,
) -> None:
    """A blank query is rejected with 400 before any host RPC."""
    session_id = conv_store.create_conversation(
        host_id=_register_host(db_uri), workspace="/host/only/repo"
    ).id
    engine = _FakeEngine(results=SearchResults(repo_root="/x", query="x", total=0, hits=[]))

    async with await _client(
        _build_app(conv_store, engine, host_registry=_FakeHostRegistry(online=True))
    ) as c:
        resp = await c.get(f"/v1/sessions/{session_id}/code-intel/search?q=%20")

    assert resp.status_code == 400
