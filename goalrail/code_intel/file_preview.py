"""Shared file-preview helpers for Goalrail code intelligence."""

from __future__ import annotations

from pathlib import Path

from goalrail.errors import ErrorCode, GoalrailError

FILE_READ_LIMIT_BYTES = 256 * 1024


def resolve_repo_file(repo_root: Path, path: str) -> Path:
    """Resolve a repo-relative file path without allowing workspace escape."""
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


def read_repo_file(
    repo_root: Path,
    path: str,
    *,
    limit_bytes: int = FILE_READ_LIMIT_BYTES,
) -> dict[str, object]:
    """Read a bounded UTF-8 preview payload for a repo-relative file."""
    file_path = resolve_repo_file(repo_root, path)
    size = file_path.stat().st_size
    with file_path.open("rb") as fh:
        raw = fh.read(limit_bytes + 1)
    truncated = len(raw) > limit_bytes
    if truncated:
        raw = raw[:limit_bytes]
    return {
        "repo_root": str(repo_root),
        "path": str(file_path.relative_to(repo_root.resolve())),
        "size_bytes": size,
        "truncated": truncated,
        "content": raw.decode("utf-8", errors="replace"),
    }
