"""Shared helpers for uploaded agent bundles."""

from __future__ import annotations

import hashlib
import tempfile
from pathlib import Path

from omnigent.errors import ErrorCode, OmnigentError
from omnigent.spec import AgentSpec, ExtractionError, load


def validate_agent_bundle(
    bundle_bytes: bytes,
    *,
    enforce_handler_allowlist: bool = True,
) -> AgentSpec:
    """
    Validate an agent bundle and return the parsed spec.

    Extracts the tarball to a temp directory, parses the spec,
    and checks that a name is present.

    This validates bundles uploaded over HTTP, so it always parses with
    ``expand_env=False``: expanding a tenant-supplied ``${VAR}`` against
    the server process environment would leak server-side secrets.
    The author of an HTTP-uploaded spec is not the
    server operator, so the server must never resolve env vars on their
    behalf — operator-authored specs resolve env at the client /
    registration boundary instead (``omnigent.cli._resolve_bundle_env_vars``).

    :param bundle_bytes: Raw bytes of the ``.tar.gz`` bundle.
    :param enforce_handler_allowlist: When ``True`` (the default),
        reject any ``type: function`` policy whose handler is not a
        registered policy handler, before the inner loader
        can resolve and call it. Callers pass ``False`` only for a
        trusted single-user/local server, where ``omnigent run`` uploads
        the operator's own bundle through this same path and custom
        handlers must keep working (the operator already has code
        execution, so the restriction would add no security). See the
        call sites in ``omnigent/server/routes/sessions.py``, which gate
        this on :func:`omnigent.server.auth.local_single_user_enabled`.
    :returns: The validated :class:`AgentSpec`.
    :raises OmnigentError: If the bundle is invalid, the spec is
        missing a name, or (when *enforce_handler_allowlist*) a policy
        names an unregistered handler.
    """
    try:
        with tempfile.TemporaryDirectory() as tmpdir:
            spec = load(
                bundle_bytes,
                dest=Path(tmpdir) / "agent",
                expand_env=False,
                enforce_handler_allowlist=enforce_handler_allowlist,
            )
    except OmnigentError:
        raise
    except ExtractionError as exc:
        raise OmnigentError(str(exc), code=ErrorCode.INVALID_INPUT) from exc
    except Exception as exc:
        # Catch YAML parse errors and other unexpected failures
        # during spec loading so they surface as 400, not 500.
        raise OmnigentError(
            f"invalid agent bundle: {exc}",
            code=ErrorCode.INVALID_INPUT,
        ) from exc

    if spec.name is None:
        raise OmnigentError(
            "agent spec must include a name",
            code=ErrorCode.INVALID_INPUT,
        )

    return spec


def bundle_location(agent_id: str, bundle_bytes: bytes) -> str:
    """
    Compute a content-addressed artifact key for a bundle.

    :param agent_id: The agent's unique identifier,
        e.g. ``"ag_abc123"``.
    :param bundle_bytes: Raw bytes of the bundle.
    :returns: Artifact store key in the form
        ``"{agent_id}/{sha256_hex}"``.
    """
    digest = hashlib.sha256(bundle_bytes).hexdigest()
    return f"{agent_id}/{digest}"
