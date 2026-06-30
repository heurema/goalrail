"""Unit tests for the shared code-intel service (envelope shaping).

These pin the wire shape that both the local route and the future host
handler depend on, so they can't drift.
"""

from __future__ import annotations

from pathlib import Path

from goalrail.code_intel import (
    CodeIntelNotIndexedError,
    CodeIntelNotInstalledError,
    IndexStatus,
    SearchHit,
    SearchResults,
    service,
)


class _Engine:
    """Scripted stand-in for CodeIntelClient."""

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

    def index_status(self, repo_root: Path) -> IndexStatus:
        if self._raises is not None:
            raise self._raises
        assert self._status is not None
        return self._status

    def search(
        self, repo_root: Path, query: str, *, limit: int = 20, label: str | None = None
    ) -> SearchResults:
        if self._raises is not None:
            raise self._raises
        assert self._results is not None
        return self._results


def test_clamp_search_limit() -> None:
    assert service.clamp_search_limit(999) == service.SEARCH_LIMIT_MAX
    assert service.clamp_search_limit(0) == service.SEARCH_LIMIT_DEFAULT
    assert service.clamp_search_limit(5) == 5


def test_head_info_present_and_absent() -> None:
    assert service.head_info({"git": {"branch": "main", "head_sha": "a", "base_sha": "b"}}) == {
        "branch": "main",
        "head_sha": "a",
        "base_sha": "b",
    }
    assert service.head_info({}) is None


def test_status_envelope_ready() -> None:
    engine = _Engine(
        status=IndexStatus(
            repo_root="/repo",
            status="ready",
            project="proj",
            nodes=10,
            edges=20,
            raw={"git": {"branch": "main", "head_sha": "abc", "base_sha": "def"}},
        )
    )
    env = service.status_envelope(engine, Path("/repo"))
    assert env["indexed"] is True
    assert env["status"] == "ready"
    assert env["head"]["branch"] == "main"


def test_status_envelope_engine_unavailable() -> None:
    engine = _Engine(raises=CodeIntelNotInstalledError("missing"))
    env = service.status_envelope(engine, Path("/repo"))
    assert env["status"] == "engine_unavailable"
    assert env["indexed"] is False


def test_search_envelope_ok() -> None:
    engine = _Engine(
        results=SearchResults(
            repo_root="/repo",
            query="widget",
            total=1,
            hits=[
                SearchHit(
                    name="widget",
                    qualified_name="pkg.widget",
                    label="Function",
                    file="pkg/mod.py",
                )
            ],
        )
    )
    env = service.search_envelope(engine, Path("/repo"), "widget", 20)
    assert env["status"] == "ok"
    assert env["results"][0]["qualified_name"] == "pkg.widget"


def test_search_envelope_not_indexed() -> None:
    engine = _Engine(raises=CodeIntelNotIndexedError("not indexed"))
    env = service.search_envelope(engine, Path("/repo"), "widget", 20)
    assert env["status"] == "not_indexed"
    assert env["results"] == []


def test_host_unsupported_envelopes() -> None:
    status = service.host_unsupported_status_envelope()
    assert status["status"] == service.HOST_UNSUPPORTED_STATUS
    assert status["repo_root"] == ""
    search = service.host_unsupported_search_envelope("q")
    assert search["status"] == service.HOST_UNSUPPORTED_STATUS
    assert search["query"] == "q"
    assert search["results"] == []
