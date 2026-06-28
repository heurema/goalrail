"""Tests for the code-intel client and the code_index_status builtin.

A fake engine binary (a Python script honoring the same CLI JSON
contract) drives the scenarios deterministically, so the tests never
depend on a real install or a pre-existing index.
"""

from __future__ import annotations

import json
import stat
import sys
from pathlib import Path

import pytest

from goalrail.code_intel import (
    CodeIntelClient,
    CodeIntelNotIndexedError,
    CodeIntelNotInstalledError,
    CodeIntelProtocolError,
    CodeIntelTimeoutError,
    IndexStatus,
    RepoBoundaryError,
    resolve_repo_root,
)
from goalrail.code_intel.client import _BINARY_ENV_VAR
from goalrail.tools.base import ToolContext
from goalrail.tools.builtins.code_intel import CodeIndexStatusTool, CodeSearchTool

# ── Fakes / fixtures ──────────────────────────────────────

# A fake engine honoring `<bin> --version` and `<bin> cli <tool> <json>`.
# Behavior is selected by FAKE_MODE; the indexed project's root_path is
# injected via FAKE_ROOT so happy-path matching is exact.
_FAKE_ENGINE = f"""#!{sys.executable}
import json, os, sys, time

if len(sys.argv) >= 2 and sys.argv[1] == "--version":
    print("code-intel-memory 9.9.9-fake")
    sys.exit(0)

mode = os.environ.get("FAKE_MODE", "happy")
root = os.environ.get("FAKE_ROOT", "")
tool = sys.argv[2] if len(sys.argv) > 2 else ""
args = json.loads(sys.argv[3]) if len(sys.argv) > 3 else {{}}

# Engines emit a log line on stderr; it must not break JSON parsing.
print("level=info msg=mem.init budget_mb=8192", file=sys.stderr)

if mode == "timeout":
    time.sleep(30)
    sys.exit(0)

if mode == "invalid_json":
    print("this is definitely not json")
    sys.exit(0)

if tool == "list_projects":
    if mode == "not_indexed":
        projects = [{{"name": "other", "root_path": "/somewhere/else",
                     "nodes": 1, "edges": 1}}]
    else:
        projects = [{{"name": "proj-slug", "root_path": root,
                     "nodes": 100, "edges": 200}}]
    print(json.dumps({{"schema_version": 1, "projects": projects}}))
    sys.exit(0)

if tool == "index_status":
    print(json.dumps({{"schema_version": 1, "project": args.get("project"),
                      "status": "ready", "nodes": 100, "edges": 200,
                      "root_path": root}}))
    sys.exit(0)

if tool == "search_graph":
    print(json.dumps({{"schema_version": 1, "total": 1, "results": [
        {{"name": args.get("name_pattern", ""),
         "qualified_name": "pkg.mod." + args.get("name_pattern", ""),
         "label": "Function", "file_path": "pkg/mod.py",
         "signature": "()", "return_type": "int"}}]}}))
    sys.exit(0)

# Target error contract: JSON envelope on stderr, non-zero exit.
print(json.dumps({{"schema_version": 1, "error": "unknown_tool",
                  "message": "no such tool"}}), file=sys.stderr)
sys.exit(2)
"""


@pytest.fixture
def fake_engine(tmp_path: Path) -> Path:
    """Write the fake engine script and return its executable path."""
    path = tmp_path / "fake-engine"
    path.write_text(_FAKE_ENGINE)
    path.chmod(path.stat().st_mode | stat.S_IEXEC | stat.S_IXGRP | stat.S_IXOTH)
    return path


@pytest.fixture
def repo(tmp_path: Path) -> Path:
    """A workspace directory standing in for the session repo root."""
    d = tmp_path / "repo"
    d.mkdir()
    return d.resolve()


def _ctx(sandbox_root: Path) -> ToolContext:
    """Build a ToolContext whose authoritative sandbox root is set."""
    return ToolContext(
        task_id="t",
        agent_id="a",
        sandbox_root=sandbox_root,
        conversation_id="c",
    )


# ── resolve_repo_root ─────────────────────────────────────


def test_resolve_repo_root_uses_sandbox_root(repo: Path) -> None:
    """With no git marker the sandbox root itself is the repo root."""
    assert resolve_repo_root(_ctx(repo)) == repo


def test_resolve_repo_root_finds_git_at_sandbox_root(repo: Path) -> None:
    """A ``.git`` at the sandbox root is recognized as the repo root."""
    (repo / ".git").mkdir()
    assert resolve_repo_root(_ctx(repo)) == repo


def test_resolve_repo_root_does_not_climb_above_sandbox_root(repo: Path) -> None:
    """When ``.git`` is only *above* the sandbox root, we stay at the root.

    Decision: the sandbox root is the trusted boundary. A session
    rooted at ``/repo/subdir`` must not be silently resolved to the
    enclosing ``/repo`` checkout even though ``/repo/.git`` exists —
    that would index code outside the boundary.
    """
    (repo / ".git").mkdir()
    subdir = repo / "subdir"
    subdir.mkdir()
    assert resolve_repo_root(_ctx(subdir)) == subdir


def test_resolve_repo_root_rejects_root_outside_boundary(repo: Path) -> None:
    """An explicit root above the session boundary is rejected."""
    with pytest.raises(RepoBoundaryError):
        resolve_repo_root(_ctx(repo), requested_root=str(repo.parent))


def test_resolve_repo_root_accepts_root_inside_boundary(repo: Path) -> None:
    """An explicit root nested inside the boundary is accepted."""
    nested = repo / "pkg"
    nested.mkdir()
    assert resolve_repo_root(_ctx(repo), requested_root=str(nested)) == nested


def test_resolve_repo_root_climbs_within_boundary_to_git(repo: Path) -> None:
    """An explicit nested root climbs *within* the boundary to find ``.git``."""
    (repo / ".git").mkdir()
    nested = repo / "pkg" / "inner"
    nested.mkdir(parents=True)
    # boundary is the sandbox root (repo); climbing up to repo's .git is allowed.
    assert resolve_repo_root(_ctx(repo), requested_root=str(nested)) == repo


# ── CodeIntelClient ───────────────────────────────────────


def test_missing_binary_is_not_installed(tmp_path: Path) -> None:
    """A nonexistent binary path reports not-installed and raises."""
    client = CodeIntelClient(binary=tmp_path / "nope")
    assert client.is_installed() is False
    assert client.version() is None
    with pytest.raises(CodeIntelNotInstalledError):
        client.index_status(tmp_path)


def test_version(fake_engine: Path) -> None:
    """The version string is read from the engine."""
    client = CodeIntelClient(binary=fake_engine)
    assert client.is_installed() is True
    assert client.version() == "code-intel-memory 9.9.9-fake"


def test_timeout(fake_engine: Path, repo: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    """A slow engine call surfaces as CodeIntelTimeoutError."""
    monkeypatch.setenv("FAKE_MODE", "timeout")
    client = CodeIntelClient(binary=fake_engine, timeout_s=0.5)
    with pytest.raises(CodeIntelTimeoutError):
        client.index_status(repo)


def test_invalid_json(fake_engine: Path, repo: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    """Non-JSON output on exit 0 surfaces as CodeIntelProtocolError."""
    monkeypatch.setenv("FAKE_MODE", "invalid_json")
    client = CodeIntelClient(binary=fake_engine)
    with pytest.raises(CodeIntelProtocolError):
        client.index_status(repo)


def test_not_indexed(fake_engine: Path, repo: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    """A repo with no matching project resolves to not_indexed."""
    monkeypatch.setenv("FAKE_MODE", "not_indexed")
    client = CodeIntelClient(binary=fake_engine)
    status = client.index_status(repo)
    assert status == IndexStatus(repo_root=str(repo), status="not_indexed", project=None)


def test_happy_path(fake_engine: Path, repo: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    """A matching, indexed repo returns a ready status with counts."""
    monkeypatch.setenv("FAKE_MODE", "happy")
    monkeypatch.setenv("FAKE_ROOT", str(repo))
    client = CodeIntelClient(binary=fake_engine)
    status = client.index_status(repo)
    assert status.status == "ready"
    assert status.project == "proj-slug"
    assert status.nodes == 100
    assert status.edges == 200


# ── CodeIndexStatusTool (end to end) ──────────────────────


def test_tool_happy_path(fake_engine: Path, repo: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    """The builtin resolves the repo server-side and returns ready status."""
    monkeypatch.setenv(_BINARY_ENV_VAR, str(fake_engine))
    monkeypatch.setenv("FAKE_MODE", "happy")
    monkeypatch.setenv("FAKE_ROOT", str(repo))

    result = json.loads(CodeIndexStatusTool().invoke("{}", _ctx(repo)))
    assert result["status"] == "ready"
    assert result["repo_root"] == str(repo)
    assert result["nodes"] == 100
    assert "raw" not in result


def test_tool_not_installed_returns_error_payload(
    tmp_path: Path, repo: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    """A missing engine yields a structured error, not a crash."""
    monkeypatch.setenv(_BINARY_ENV_VAR, str(tmp_path / "nope"))
    result = json.loads(CodeIndexStatusTool().invoke("{}", _ctx(repo)))
    assert result["error"] == "code_intel_not_installed"
    assert "message" in result


# ── search ────────────────────────────────────────────────


def test_search_happy(fake_engine: Path, repo: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    """Search over an indexed repo returns trimmed hits."""
    monkeypatch.setenv("FAKE_MODE", "happy")
    monkeypatch.setenv("FAKE_ROOT", str(repo))
    client = CodeIntelClient(binary=fake_engine)
    results = client.search(repo, "widget", limit=5)
    assert results.total == 1
    assert results.query == "widget"
    assert len(results.hits) == 1
    hit = results.hits[0]
    assert hit.name == "widget"
    assert hit.qualified_name == "pkg.mod.widget"
    assert hit.label == "Function"
    assert hit.file == "pkg/mod.py"


def test_search_not_indexed_raises(
    fake_engine: Path, repo: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    """Searching an unindexed repo raises CodeIntelNotIndexedError."""
    monkeypatch.setenv("FAKE_MODE", "not_indexed")
    client = CodeIntelClient(binary=fake_engine)
    with pytest.raises(CodeIntelNotIndexedError):
        client.search(repo, "widget")


def test_tool_search_happy(fake_engine: Path, repo: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    """The code_search builtin returns results resolved server-side."""
    monkeypatch.setenv(_BINARY_ENV_VAR, str(fake_engine))
    monkeypatch.setenv("FAKE_MODE", "happy")
    monkeypatch.setenv("FAKE_ROOT", str(repo))

    result = json.loads(CodeSearchTool().invoke(json.dumps({"query": "widget"}), _ctx(repo)))
    assert result["total"] == 1
    assert result["query"] == "widget"
    assert result["results"][0]["qualified_name"] == "pkg.mod.widget"


def test_tool_search_requires_query(repo: Path) -> None:
    """A missing query is rejected before any engine call."""
    result = json.loads(CodeSearchTool().invoke("{}", _ctx(repo)))
    assert result["error"] == "invalid_arguments"


def test_tool_search_not_indexed(
    fake_engine: Path, repo: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    """code_search surfaces not_indexed as a structured payload."""
    monkeypatch.setenv(_BINARY_ENV_VAR, str(fake_engine))
    monkeypatch.setenv("FAKE_MODE", "not_indexed")

    result = json.loads(CodeSearchTool().invoke(json.dumps({"query": "widget"}), _ctx(repo)))
    assert result["error"] == "not_indexed"
