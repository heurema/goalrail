"""Unit tests for the server→host code-intel status RPC.

Exercises ``request_code_intel_status`` directly with lightweight fakes
so the request/await/timeout plumbing is covered without a real tunnel.
"""

from __future__ import annotations

import asyncio
import json
from types import SimpleNamespace
from typing import Any

import pytest

from goalrail.server.host_registry import (
    HostCodeIntelError,
    request_code_intel_file,
    request_code_intel_search,
    request_code_intel_status,
)

pytestmark = pytest.mark.asyncio


class _FakeRegistry:
    """Resolves the pending future inline to simulate a host reply."""

    def __init__(
        self,
        reply: dict[str, Any] | None,
        pending_attr: str = "pending_code_intel_status",
    ) -> None:
        self._reply = reply
        self._pending_attr = pending_attr
        self.sent: list[str] = []

    def send_text(self, conn: Any, data: str) -> None:
        self.sent.append(data)
        if self._reply is None:
            return  # simulate a host that never answers (timeout path)
        request_id = json.loads(data)["request_id"]
        future = getattr(conn, self._pending_attr)[request_id]
        if not future.done():
            future.set_result(self._reply)


def _fake_conn() -> Any:
    return SimpleNamespace(
        host_id="host_1",
        pending_code_intel_files={},
        pending_code_intel_status={},
        pending_code_intel_search={},
    )


async def test_request_status_happy_returns_envelope() -> None:
    envelope = {"repo_root": "/repo", "indexed": True, "status": "ready"}
    registry = _FakeRegistry({"status": "ok", "envelope": envelope, "error": None})
    conn = _fake_conn()

    result = await request_code_intel_status(registry, conn, "~/repo")

    assert result == envelope
    # The request frame carried the workspace as an opaque ref.
    assert json.loads(registry.sent[0])["workspace"] == "~/repo"
    # Pending entry is cleaned up.
    assert conn.pending_code_intel_status == {}


async def test_request_status_host_failure_raises() -> None:
    registry = _FakeRegistry({"status": "failed", "envelope": None, "error": "boom"})
    conn = _fake_conn()

    with pytest.raises(HostCodeIntelError, match="boom"):
        await request_code_intel_status(registry, conn, "/repo")
    assert conn.pending_code_intel_status == {}


async def test_request_status_missing_envelope_raises() -> None:
    registry = _FakeRegistry({"status": "ok", "envelope": None, "error": None})
    conn = _fake_conn()

    with pytest.raises(HostCodeIntelError):
        await request_code_intel_status(registry, conn, "/repo")


async def test_request_status_timeout_raises() -> None:
    registry = _FakeRegistry(None)  # never replies
    conn = _fake_conn()

    with pytest.raises(HostCodeIntelError, match="did not respond"):
        await request_code_intel_status(registry, conn, "/repo", timeout=0.05)
    assert conn.pending_code_intel_status == {}


async def test_request_status_connection_lost_raises() -> None:
    class _DeadRegistry:
        def send_text(self, conn: Any, data: str) -> None:
            raise ConnectionError("gone")

    conn = _fake_conn()
    with pytest.raises(HostCodeIntelError, match="connection lost"):
        await request_code_intel_status(_DeadRegistry(), conn, "/repo")
    assert conn.pending_code_intel_status == {}


async def test_request_status_does_not_block_event_loop() -> None:
    """A pending request leaves the loop free until the reply arrives."""
    registry = _FakeRegistry(None)
    conn = _fake_conn()
    task = asyncio.ensure_future(request_code_intel_status(registry, conn, "/repo", timeout=5))
    await asyncio.sleep(0)  # let it register the future + send
    assert len(conn.pending_code_intel_status) == 1
    # Now resolve it as the tunnel would.
    request_id = json.loads(registry.sent[0])["request_id"]
    conn.pending_code_intel_status[request_id].set_result(
        {"status": "ok", "envelope": {"status": "ready"}, "error": None}
    )
    assert (await task) == {"status": "ready"}


# ── search RPC ────────────────────────────────────────────


async def test_request_search_happy_returns_envelope() -> None:
    envelope = {"query": "widget", "status": "ok", "total": 1, "results": [{}]}
    registry = _FakeRegistry(
        {"status": "ok", "envelope": envelope, "error": None},
        pending_attr="pending_code_intel_search",
    )
    conn = _fake_conn()

    result = await request_code_intel_search(registry, conn, "~/repo", "widget", 25)

    assert result == envelope
    sent = json.loads(registry.sent[0])
    assert sent["query"] == "widget"
    assert sent["limit"] == 25
    assert sent["workspace"] == "~/repo"
    assert conn.pending_code_intel_search == {}


async def test_request_search_host_failure_raises() -> None:
    registry = _FakeRegistry(
        {"status": "failed", "envelope": None, "error": "boom"},
        pending_attr="pending_code_intel_search",
    )
    conn = _fake_conn()

    with pytest.raises(HostCodeIntelError, match="boom"):
        await request_code_intel_search(registry, conn, "/repo", "widget", 20)
    assert conn.pending_code_intel_search == {}


async def test_request_search_timeout_raises() -> None:
    registry = _FakeRegistry(None, pending_attr="pending_code_intel_search")
    conn = _fake_conn()

    with pytest.raises(HostCodeIntelError, match="did not respond"):
        await request_code_intel_search(registry, conn, "/repo", "widget", 20, timeout=0.05)
    assert conn.pending_code_intel_search == {}


# ── file preview RPC ──────────────────────────────────────


async def test_request_file_happy_returns_payload() -> None:
    payload = {"path": "pkg/mod.py", "content": "class Widget:\n"}
    registry = _FakeRegistry(
        {"status": "ok", "file": payload, "error": None},
        pending_attr="pending_code_intel_files",
    )
    conn = _fake_conn()

    result = await request_code_intel_file(registry, conn, "~/repo", "pkg/mod.py")

    assert result == payload
    sent = json.loads(registry.sent[0])
    assert sent["workspace"] == "~/repo"
    assert sent["path"] == "pkg/mod.py"
    assert conn.pending_code_intel_files == {}


async def test_request_file_host_failure_raises() -> None:
    registry = _FakeRegistry(
        {"status": "failed", "file": None, "error": "boom"},
        pending_attr="pending_code_intel_files",
    )
    conn = _fake_conn()

    with pytest.raises(HostCodeIntelError, match="boom"):
        await request_code_intel_file(registry, conn, "/repo", "pkg/mod.py")
    assert conn.pending_code_intel_files == {}


async def test_request_file_timeout_raises() -> None:
    registry = _FakeRegistry(None, pending_attr="pending_code_intel_files")
    conn = _fake_conn()

    with pytest.raises(HostCodeIntelError, match="did not respond"):
        await request_code_intel_file(registry, conn, "/repo", "pkg/mod.py", timeout=0.05)
    assert conn.pending_code_intel_files == {}
