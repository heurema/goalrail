"""Goalrail-native integration with the code-intel-memory engine.

Goalrail owns the integration end to end: it resolves the target
repository root **server-side** from the session context (never from
an agent-supplied path), runs the engine's stable CLI JSON contract
as a subprocess, parses the result, and surfaces typed errors. The
agent only ever calls Goalrail ``code_*`` builtin tools — it does not
know about the CLI, the binary name, or the on-disk index store.

The engine ships as a single static binary that historically went by
two names; both are supported during the rename (see
:data:`goalrail.code_intel.client.BINARY_NAMES`).
"""

from __future__ import annotations

from goalrail.code_intel.client import (
    BINARY_NAMES,
    CodeIntelClient,
    CodeIntelError,
    CodeIntelNotIndexedError,
    CodeIntelNotInstalledError,
    CodeIntelProtocolError,
    CodeIntelTimeoutError,
    CodeIntelToolError,
    IndexStatus,
    RepoBoundaryError,
    SearchHit,
    SearchResults,
    resolve_repo_root,
)

__all__ = [
    "BINARY_NAMES",
    "CodeIntelClient",
    "CodeIntelError",
    "CodeIntelNotIndexedError",
    "CodeIntelNotInstalledError",
    "CodeIntelProtocolError",
    "CodeIntelTimeoutError",
    "CodeIntelToolError",
    "IndexStatus",
    "RepoBoundaryError",
    "SearchHit",
    "SearchResults",
    "resolve_repo_root",
]
