"""Subprocess client for the code-intel-memory CLI JSON contract.

This module is the single place in Goalrail that knows how to talk to
the engine binary. Everything above it (builtin tools, future API
routes) goes through :class:`CodeIntelClient` and the typed result
models / exceptions defined here — never by shelling out directly.

CLI contract
------------
The engine exposes every tool through one invocation shape::

    <binary> cli <tool> '<json-args>'

Target contract (what the engine should converge to, per the Phase 1
decision):

* **success** -> JSON object on **stdout**, exit code **0**;
* **error** -> JSON object on **stderr**, **non-zero** exit code;
* **logs** stay on stderr and must never break JSON parsing.

Tolerated legacy behavior (current engine): a *logical* error (e.g.
"project not indexed") is written as a JSON envelope on **stderr**
while the process still exits **0** with an empty stdout. We tolerate
this for compatibility but do not treat it as the desired end state —
:meth:`CodeIntelClient._run` normalizes both shapes into a
:class:`CodeIntelToolError`.
"""

from __future__ import annotations

import json
import logging
import os
import shutil
import subprocess
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Protocol, TypeAlias

_logger = logging.getLogger(__name__)

# Binary names to probe, in preference order. The engine is being
# renamed ``codebase-memory-mcp`` -> ``code-intel-memory``; both are
# accepted so the integration works before, during, and after the
# rename lands.
BINARY_NAMES: tuple[str, ...] = ("code-intel-memory", "codebase-memory-mcp")

# Env override pointing at an explicit binary path. Highest priority
# after a constructor-supplied path; used by tests and by advanced
# users with a non-PATH install.
_BINARY_ENV_VAR = "GOALRAIL_CODE_INTEL_BIN"

# Default subprocess wallclock cap. Index *reads* (status/search) are
# sub-millisecond on the engine; 30s is generous headroom that still
# fails fast if the binary hangs.
_DEFAULT_TIMEOUT_S = 30.0

# A JSON object as returned by the engine CLI. Engine payloads are
# free-form and differ per tool, so the value type is genuinely dynamic
# — this is the one place we accept ``Any`` (documented escape hatch).
_Json: TypeAlias = dict[str, Any]  # type: ignore[explicit-any]  # free-form engine JSON


class _SessionContext(Protocol):
    """Minimal session context needed to resolve a repo root.

    Structurally satisfied by :class:`~goalrail.tools.base.ToolContext`;
    declared as a Protocol so this module does not depend on the tool
    layer. ``sandbox_root`` is a read-only property so a frozen
    dataclass attribute satisfies it.
    """

    @property
    def sandbox_root(self) -> Path | None: ...


# ── Errors ────────────────────────────────────────────────


class CodeIntelError(Exception):
    """Base class for every code-intel integration failure."""


class CodeIntelNotInstalledError(CodeIntelError):
    """The engine binary could not be found on PATH or via override."""


class CodeIntelTimeoutError(CodeIntelError):
    """The engine subprocess exceeded the configured timeout."""


class CodeIntelProtocolError(CodeIntelError):
    """The engine produced output that is not parseable as JSON.

    Indicates a contract violation (garbage on the expected stream, or
    a crash with no JSON envelope) rather than a normal tool-level
    error.
    """


class CodeIntelToolError(CodeIntelError):
    """The engine returned a structured error envelope.

    :param message: Human-readable message from the envelope.
    :param code: Machine-readable error code when present (the engine
        uses the ``error`` field as the code today).
    :param payload: The full parsed error envelope, for callers that
        want hints / available_projects / etc.
    """

    def __init__(
        self,
        message: str,
        *,
        code: str | None = None,
        payload: _Json | None = None,
    ) -> None:
        super().__init__(message)
        self.message = message
        self.code = code
        self.payload = payload or {}


class RepoBoundaryError(CodeIntelError):
    """A requested repo root escaped the allowed session boundary.

    Raised when an explicit root (e.g. a future ``repo_id`` resolution
    path) points outside the session/workspace directory. The agent
    never supplies a path, so this guards internal callers and the
    later multi-repo flow.
    """


# ── Result models ─────────────────────────────────────────


@dataclass(frozen=True)
class IndexStatus:
    """Index status for one repository.

    :param repo_root: Absolute repo root Goalrail resolved server-side.
    :param status: ``"ready"`` when indexed and queryable,
        ``"not_indexed"`` when the engine has no project for this root,
        or whatever status string the engine reports otherwise.
    :param project: Engine project name (slug) when known, else ``None``.
    :param nodes: Node count in the knowledge graph, when reported.
    :param edges: Edge count in the knowledge graph, when reported.
    :param raw: The full parsed engine response (empty for the
        synthesized ``not_indexed`` result).
    """

    repo_root: str
    status: str
    project: str | None = None
    nodes: int | None = None
    edges: int | None = None
    raw: _Json = field(default_factory=dict)


# ── Repo resolution (server-side) ─────────────────────────


def _is_within(child: Path, parent: Path) -> bool:
    """Return ``True`` if ``child`` is ``parent`` or nested under it."""
    return child == parent or parent in child.parents


def _discover_git_root(start: Path, *, boundary: Path) -> Path | None:
    """Find a git root at or above ``start`` without leaving ``boundary``.

    Walks upward from ``start`` looking for a ``.git`` entry (directory
    for a normal checkout, file for a worktree/submodule), but never
    ascends past ``boundary``. When ``start == boundary`` this only
    inspects the boundary itself — by design we do **not** climb above
    the trusted sandbox root to discover a parent repo's ``.git``.

    :returns: The git root within the boundary, or ``None`` if none.
    """
    for candidate in (start, *start.parents):
        if not _is_within(candidate, boundary):
            break
        if (candidate / ".git").exists():
            return candidate
    return None


def resolve_repo_root(ctx: _SessionContext, requested_root: str | None = None) -> Path:
    """Resolve the repository root for a code-intel call, server-side.

    The agent never passes a filesystem path. Goalrail derives the root
    from the session's **authoritative sandbox root**
    (:attr:`ToolContext.sandbox_root` — the runner workspace / repo the
    session operates in), falling back to the process cwd when that is
    unset. The per-conversation :attr:`ToolContext.workspace` is
    deliberately *not* used: for runner-local Python tools it is a
    scratch subdirectory (``runner_workspace / conversation_id``), not
    the repo root.

    Decision (repo-root resolution): the sandbox root is the **trusted
    boundary**. We recognize a ``.git`` only at the sandbox root itself
    and never climb above it — indexing a parent repo would read code
    outside the session's trusted boundary. If the sandbox root sits
    inside a larger checkout (``.git`` only above it), we deliberately
    resolve to the sandbox root, not the enclosing repo.

    ``requested_root`` is reserved for internal/UI callers and the
    future multi-repo ``repo_id`` flow; when supplied it must resolve
    to a path inside the boundary (climbing *within* the boundary to a
    ``.git`` is allowed) or :class:`RepoBoundaryError` is raised.

    :param ctx: Anything exposing an optional ``sandbox_root`` path
        (e.g. :class:`~goalrail.tools.base.ToolContext`).
    :param requested_root: Optional explicit root from a trusted
        internal caller; ``None`` for agent-driven calls.
    :returns: Absolute, canonicalized repo root.
    :raises RepoBoundaryError: If ``requested_root`` escapes the
        boundary.
    """
    sandbox_root = ctx.sandbox_root
    boundary = (Path(sandbox_root) if sandbox_root else Path.cwd()).resolve()

    if requested_root is not None:
        candidate = Path(requested_root).expanduser().resolve()
        if not _is_within(candidate, boundary):
            raise RepoBoundaryError(
                f"requested repo root {candidate} is outside the session boundary {boundary}"
            )
        base = candidate
    else:
        base = boundary

    return _discover_git_root(base, boundary=boundary) or base


# ── Client ────────────────────────────────────────────────


def _first_json_object(text: str) -> _Json | None:
    """Extract the first line of ``text`` that parses as a JSON object.

    The engine prefixes some output with ``level=info ...`` log lines on
    stderr; the JSON envelope itself is emitted as a single compact
    line. Scanning line-by-line isolates the envelope from the logs.

    :returns: The parsed object, or ``None`` if no line parses as a
        JSON object.
    """
    for line in text.splitlines():
        stripped = line.strip()
        if not stripped.startswith("{"):
            continue
        try:
            parsed = json.loads(stripped)
        except json.JSONDecodeError:
            continue
        if isinstance(parsed, dict):
            return parsed
    return None


class CodeIntelClient:
    """Typed, timeout-bounded wrapper around the engine CLI.

    :param binary: Explicit path to the engine binary. When ``None``,
        discovery falls back to the :data:`_BINARY_ENV_VAR` env var,
        then to ``PATH`` (probing :data:`BINARY_NAMES`).
    :param timeout_s: Per-invocation wallclock cap in seconds.
    """

    def __init__(
        self,
        *,
        binary: str | Path | None = None,
        timeout_s: float = _DEFAULT_TIMEOUT_S,
    ) -> None:
        self._explicit_binary = Path(binary) if binary is not None else None
        self._timeout_s = timeout_s

    # -- binary discovery --

    def _find_binary(self) -> Path | None:
        """Locate the engine binary, or return ``None`` if absent."""
        if self._explicit_binary is not None:
            return self._explicit_binary if self._explicit_binary.exists() else None
        override = os.environ.get(_BINARY_ENV_VAR)
        if override:
            path = Path(override)
            return path if path.exists() else None
        for name in BINARY_NAMES:
            found = shutil.which(name)
            if found:
                return Path(found)
        return None

    def _resolve_binary(self) -> Path:
        """Locate the engine binary or raise :class:`CodeIntelNotInstalledError`."""
        found = self._find_binary()
        if found is None:
            raise CodeIntelNotInstalledError(
                "code-intel engine binary not found. Looked for "
                f"{', '.join(BINARY_NAMES)} on PATH and "
                f"${_BINARY_ENV_VAR}. Install it or set {_BINARY_ENV_VAR}."
            )
        return found

    def is_installed(self) -> bool:
        """Return ``True`` if the engine binary is discoverable."""
        return self._find_binary() is not None

    def version(self) -> str | None:
        """Return the engine version string, or ``None`` on any failure."""
        try:
            binary = self._resolve_binary()
        except CodeIntelNotInstalledError:
            return None
        try:
            proc = subprocess.run(
                [str(binary), "--version"],
                capture_output=True,
                text=True,
                timeout=self._timeout_s,
                check=False,
            )
        except (subprocess.TimeoutExpired, OSError):
            return None
        out = proc.stdout.strip() or proc.stderr.strip()
        return out or None

    # -- core invocation --

    def _run(self, tool: str, payload: _Json) -> _Json:
        """Run one engine tool and return its success envelope.

        :param tool: Engine tool name, e.g. ``"index_status"``.
        :param payload: JSON-serializable arguments for the tool.
        :returns: The parsed success JSON object.
        :raises CodeIntelNotInstalledError: Binary missing.
        :raises CodeIntelTimeoutError: Subprocess exceeded the timeout.
        :raises CodeIntelToolError: Engine returned an error envelope.
        :raises CodeIntelProtocolError: Output was not parseable JSON.
        """
        binary = self._resolve_binary()
        argv = [str(binary), "cli", tool, json.dumps(payload)]
        try:
            proc = subprocess.run(
                argv,
                capture_output=True,
                text=True,
                timeout=self._timeout_s,
                check=False,
            )
        except subprocess.TimeoutExpired as exc:
            raise CodeIntelTimeoutError(
                f"code-intel '{tool}' timed out after {self._timeout_s}s"
            ) from exc
        except OSError as exc:  # binary vanished / not executable
            raise CodeIntelNotInstalledError(
                f"failed to execute code-intel binary {binary}: {exc}"
            ) from exc

        return self._parse(tool, proc.returncode, proc.stdout, proc.stderr)

    def _parse(
        self,
        tool: str,
        returncode: int,
        stdout: str,
        stderr: str,
    ) -> _Json:
        """Normalize an engine invocation into a success dict or raise.

        Implements the target contract while tolerating the legacy
        "logical error on stderr with exit 0" shape (see module docstring).
        """
        # Success path: exit 0 with a JSON object on stdout that is not
        # itself an error envelope.
        if returncode == 0:
            obj = _first_json_object(stdout)
            if obj is not None and "error" not in obj:
                return obj
            # Tolerated legacy shape: exit 0 but the real payload is an
            # error envelope on stderr (or, defensively, on stdout).
            err = obj if (obj and "error" in obj) else _first_json_object(stderr)
            if err is not None and "error" in err:
                self._raise_tool_error(tool, err)
            # Exit 0 but nothing parseable — contract violation.
            raise CodeIntelProtocolError(
                f"code-intel '{tool}' exited 0 with no parseable JSON "
                f"(stdout={stdout[:200]!r} stderr={stderr[:200]!r})"
            )

        # Non-zero exit: an error envelope should be on stderr (target
        # contract), but accept it on stdout too for robustness.
        err = _first_json_object(stderr) or _first_json_object(stdout)
        if err is not None and "error" in err:
            self._raise_tool_error(tool, err)
        raise CodeIntelProtocolError(
            f"code-intel '{tool}' exited {returncode} without a JSON error "
            f"envelope (stdout={stdout[:200]!r} stderr={stderr[:200]!r})"
        )

    @staticmethod
    def _raise_tool_error(tool: str, envelope: _Json) -> None:
        """Raise :class:`CodeIntelToolError` from an engine error envelope."""
        code = envelope.get("error")
        message = envelope.get("message") or (
            code if isinstance(code, str) else f"code-intel '{tool}' failed"
        )
        raise CodeIntelToolError(
            str(message),
            code=code if isinstance(code, str) else None,
            payload=envelope,
        )

    # -- typed operations --

    def list_projects(self) -> list[_Json]:
        """Return all indexed projects (``name`` + ``root_path`` + meta)."""
        result = self._run("list_projects", {})
        projects = result.get("projects", [])
        return projects if isinstance(projects, list) else []

    def find_project_by_root(self, repo_root: Path) -> _Json | None:
        """Find the indexed project whose ``root_path`` matches ``repo_root``.

        Matches by canonicalized path rather than reimplementing the
        engine's path-to-slug rule, so the two stay in sync by
        construction.
        """
        target = repo_root.resolve()
        for project in self.list_projects():
            root = project.get("root_path")
            if root and Path(root).resolve() == target:
                return project
        return None

    def index_status(self, repo_root: Path) -> IndexStatus:
        """Return the index status for ``repo_root``.

        Resolves the project by ``root_path`` first; when no project
        matches, returns a synthesized ``not_indexed`` status instead of
        relying on the engine's error envelope for the common
        "never indexed" case.
        """
        project = self.find_project_by_root(repo_root)
        if project is None:
            return IndexStatus(
                repo_root=str(repo_root),
                status="not_indexed",
                project=None,
            )
        result = self._run("index_status", {"project": project["name"]})
        return IndexStatus(
            repo_root=str(repo_root),
            status=str(result.get("status", "unknown")),
            project=result.get("project", project.get("name")),
            nodes=result.get("nodes"),
            edges=result.get("edges"),
            raw=result,
        )
