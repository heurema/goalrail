"""Unit tests for CLI OIDC token storage (goalrail/cli_auth.py).

Tests the store/load/clear lifecycle for session tokens persisted
by ``goalrail login``.
"""

from __future__ import annotations

import time
from pathlib import Path

import pytest


@pytest.fixture()
def token_dir(tmp_path, monkeypatch):
    """Redirect the token file to a temp directory.

    Patches ``_token_file_path`` so tests don't touch the real
    runtime data home.

    :param tmp_path: Pytest temp directory.
    :param monkeypatch: Pytest monkeypatch fixture.
    :returns: The temp directory path.
    """
    monkeypatch.setattr(
        "goalrail.cli_auth._token_file_path",
        lambda: tmp_path / "auth_tokens.json",
    )
    return tmp_path


def test_store_and_load_token(token_dir) -> None:
    """A stored token can be loaded back by server URL.

    This is the happy path: ``goalrail login`` stores a token,
    ``goalrail run --server`` loads it.
    """
    from goalrail.cli_auth import load_token, store_token

    store_token(
        server_url="http://localhost:8000",
        token="jwt-abc",
        user_id="alice@example.com",
        expires_at=time.time() + 3600,
    )

    result = load_token("http://localhost:8000")
    # Token must be the exact value stored.
    assert result == "jwt-abc", f"Expected 'jwt-abc', got {result!r}."


def test_store_token_uses_goalrail_data_dir(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """Auth token storage writes under the effective runtime data home."""
    from goalrail.cli_auth import load_token, store_token

    data_home = tmp_path / "goalrail-data"
    legacy_home = tmp_path / "home"
    monkeypatch.setenv("GOALRAIL_DATA_DIR", str(data_home))
    monkeypatch.setenv("HOME", str(legacy_home))

    store_token(
        server_url="http://localhost:8000",
        token="jwt-abc",
        user_id="alice@example.com",
        expires_at=time.time() + 3600,
    )

    token_file = data_home / "auth_tokens.json"
    assert token_file.is_file()
    assert load_token("http://localhost:8000") == "jwt-abc"
    assert not (legacy_home / ".goalrail" / "auth_tokens.json").exists()


def test_load_returns_none_when_no_file(token_dir) -> None:
    """load_token returns None when no token file exists.

    The first time a user runs ``goalrail run --server`` without
    having run ``goalrail login``, there should be no crash.
    """
    from goalrail.cli_auth import load_token

    assert load_token("http://localhost:8000") is None


def test_load_returns_none_for_unknown_server(token_dir) -> None:
    """load_token returns None for a server with no stored token.

    A token stored for one server must not leak to another.
    """
    from goalrail.cli_auth import load_token, store_token

    store_token(
        server_url="http://localhost:8000",
        token="jwt-abc",
        user_id="alice@example.com",
        expires_at=time.time() + 3600,
    )

    assert load_token("http://other-server:9000") is None


def test_load_returns_none_for_expired_token(token_dir) -> None:
    """load_token returns None when the stored token has expired.

    Expired tokens must not be used — the user needs to re-run
    ``goalrail login``.
    """
    from goalrail.cli_auth import load_token, store_token

    store_token(
        server_url="http://localhost:8000",
        token="jwt-expired",
        user_id="alice@example.com",
        expires_at=time.time() - 1,  # Already expired.
    )

    assert load_token("http://localhost:8000") is None


def test_clear_token(token_dir) -> None:
    """clear_token removes a stored token for a server.

    After clearing, load_token must return None.
    """
    from goalrail.cli_auth import clear_token, load_token, store_token

    store_token(
        server_url="http://localhost:8000",
        token="jwt-abc",
        user_id="alice@example.com",
        expires_at=time.time() + 3600,
    )
    clear_token("http://localhost:8000")

    assert load_token("http://localhost:8000") is None


def test_trailing_slash_normalization(token_dir) -> None:
    """Server URLs are normalized (trailing slash stripped).

    ``http://localhost:8000/`` and ``http://localhost:8000`` must
    resolve to the same stored token.
    """
    from goalrail.cli_auth import load_token, store_token

    store_token(
        server_url="http://localhost:8000/",
        token="jwt-slash",
        user_id="alice@example.com",
        expires_at=time.time() + 3600,
    )

    # Load without trailing slash.
    assert load_token("http://localhost:8000") == "jwt-slash"


def test_file_permissions(token_dir) -> None:
    """Token file is created with 0o600 (user-only read/write).

    Tokens are sensitive — they must not be world-readable.
    """

    from goalrail.cli_auth import store_token

    store_token(
        server_url="http://localhost:8000",
        token="jwt-abc",
        user_id="alice@example.com",
        expires_at=time.time() + 3600,
    )

    path = token_dir / "auth_tokens.json"
    mode = path.stat().st_mode & 0o777
    # 0o600 = user read + write only.
    assert mode == 0o600, (
        f"Token file should have 0o600 permissions, got {oct(mode)}. "
        f"This means the token could be readable by other users."
    )


def test_store_overwrites_existing(token_dir) -> None:
    """Storing a token for the same server overwrites the old one.

    Re-running ``goalrail login`` should update the token, not
    append.
    """
    from goalrail.cli_auth import load_token, store_token

    store_token(
        server_url="http://localhost:8000",
        token="old-token",
        user_id="alice@example.com",
        expires_at=time.time() + 3600,
    )
    store_token(
        server_url="http://localhost:8000",
        token="new-token",
        user_id="alice@example.com",
        expires_at=time.time() + 3600,
    )

    assert load_token("http://localhost:8000") == "new-token"


def test_multiple_servers(token_dir) -> None:
    """Tokens for different servers are stored independently.

    A user may have accounts on multiple servers.
    """
    from goalrail.cli_auth import load_token, store_token

    store_token(
        server_url="http://localhost:8000",
        token="token-a",
        user_id="alice@example.com",
        expires_at=time.time() + 3600,
    )
    store_token(
        server_url="https://prod.example.com",
        token="token-b",
        user_id="alice@example.com",
        expires_at=time.time() + 3600,
    )

    assert load_token("http://localhost:8000") == "token-a"
    assert load_token("https://prod.example.com") == "token-b"
