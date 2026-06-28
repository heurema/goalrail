"""CLI-side auth storage for ``goalrail login``.

Persists per-server session JWTs from the browser-based OIDC / accounts login
flow in the effective runtime data home, keyed by server URL.

See ``designs/OIDC_AUTH.md`` §CLI Login Flow.
"""

from __future__ import annotations

import json
import logging
import os
import stat
import time
from pathlib import Path

from goalrail._env_compat import data_home_path

_logger = logging.getLogger(__name__)
_TOKEN_FILE_NAME = "auth_tokens.json"


def _token_file_path() -> Path:
    """Return the path to the auth token storage file.

    Uses the shared runtime data directory.

    :returns: Path to ``<data-home>/auth_tokens.json``.
    """
    return data_home_path() / _TOKEN_FILE_NAME


def _normalize_server_url(server_url: str) -> str:
    """Normalize a server URL for use as a dict key.

    Strips trailing slashes so ``http://localhost:6767`` and
    ``http://localhost:6767/`` resolve to the same entry.

    :param server_url: The server URL to normalize.
    :returns: Normalized URL string.
    """
    return server_url.rstrip("/")


def _store_entry(server_url: str, entry: dict[str, str | float]) -> None:
    """Create or update a server's record in the auth-tokens file.

    Writes ``<data-home>/auth_tokens.json`` with user-only
    read/write permissions (``0o600``) — the file may hold session
    JWTs, which are sensitive.

    :param server_url: The server URL the record is keyed by, e.g.
        ``"http://localhost:6767"``.
    :param entry: The record to store, e.g.
        ``{"token": "...", "user_id": "...", "expires_at": 1750000000.0}``.
    """
    path = _token_file_path()
    path.parent.mkdir(parents=True, exist_ok=True)

    data: dict[str, dict[str, str | float]] = {}
    if path.exists():
        try:
            data = json.loads(path.read_text())
        except (json.JSONDecodeError, OSError):
            data = {}

    data[_normalize_server_url(server_url)] = entry

    path.write_text(json.dumps(data, indent=2))
    os.chmod(path, stat.S_IRUSR | stat.S_IWUSR)


def store_token(
    server_url: str,
    token: str,
    user_id: str,
    expires_at: float,
) -> None:
    """Persist a session token for a server.

    :param server_url: The server URL, e.g.
        ``"http://localhost:6767"``.
    :param token: The session JWT string.
    :param user_id: The authenticated user's email, e.g.
        ``"alice@example.com"``.
    :param expires_at: Unix timestamp when the token expires.
    """
    _store_entry(
        server_url,
        {
            "token": token,
            "user_id": user_id,
            "expires_at": expires_at,
        },
    )


def _load_entry(server_url: str) -> dict[str, str | float] | None:
    """Load the raw stored record for a server, if any.

    :param server_url: The server URL, e.g.
        ``"http://localhost:6767"``.
    :returns: The stored record dict, or ``None`` when the file or
        entry is missing/unreadable.
    """
    path = _token_file_path()
    if not path.exists():
        return None

    try:
        data = json.loads(path.read_text())
    except (json.JSONDecodeError, OSError):
        return None

    entry = data.get(_normalize_server_url(server_url))
    return entry if isinstance(entry, dict) else None


def load_token(server_url: str) -> str | None:
    """Load a stored session token for a server.

    Returns ``None`` if no token is stored, the token has expired,
    or the file is unreadable.

    :param server_url: The server URL, e.g.
        ``"http://localhost:6767"``.
    :returns: The session JWT string, or ``None``.
    """
    entry = _load_entry(server_url)
    if entry is None:
        return None

    expires_at = entry.get("expires_at", 0)
    if isinstance(expires_at, (int, float)) and expires_at < time.time():
        _logger.debug("Stored token for %s has expired", _normalize_server_url(server_url))
        return None

    token = entry.get("token")
    return token if isinstance(token, str) else None


def clear_token(server_url: str) -> None:
    """Remove a stored token for a server.

    No-op if no token is stored or the file doesn't exist.

    :param server_url: The server URL, e.g.
        ``"http://localhost:6767"``.
    """
    path = _token_file_path()
    if not path.exists():
        return

    try:
        data = json.loads(path.read_text())
    except (json.JSONDecodeError, OSError):
        return

    key = _normalize_server_url(server_url)
    if key in data:
        del data[key]
        path.write_text(json.dumps(data, indent=2))
