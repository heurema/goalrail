# 05 Contradictions

## C1: "Install immediately" vs. "Do not mutate hidden external config"

- Initial desire: Goalrail install should immediately make codebase-memory-mcp available.
- Evidence: `codebase-memory-mcp install` writes MCP configs, instruction files, skills, and hooks for detected agents.
- Resolution: installing the companion binary is safe as an explicit installer step; agent-config repair must be visible, promptable, and warning-not-fatal.

## C2: "`install -y` is convenient" vs. "`install -y` can delete indexes"

- Initial command: `codebase-memory-mcp install -y`.
- Evidence: current CBM install prompts to delete existing indexes and `-y` auto-answers yes.
- Resolution: do not run full `install -y` blindly in the first Goalrail PR. Prefer:
  - run `install --plan` for visibility;
  - in interactive installer/setup, ask before full repair and mention existing index rebuild risk;
  - in non-interactive installer, skip full repair and print the command;
  - or wait for an upstream non-destructive repair mode.

## C3: PyPI dependency looks simple vs. wheel install has no safe post-install hook

- Dependency argument: adding `codebase-memory-mcp` to `pyproject.toml` would install the wrapper.
- Evidence: package install would not safely run `config set auto_index` or agent config repair, and wheels do not provide a suitable post-install hook.
- Resolution: no Goalrail `pyproject.toml` dependency for this integration.

## C4: Local installer vs. remote sandbox parity

- Local path: install codebase-memory-mcp via `uv tool`.
- Remote path: Goalrail sandbox host image overlays wheels with `pip --no-deps`.
- Resolution: remote support is separate host-image work near `DEFAULT_HOST_IMAGE` and Docker image build, not part of the installer PR.

## C5: Hard fail vs. warning-not-fatal

- Skeptic position: hard-fail platform/download mismatches for clarity.
- Product constraint: Goalrail must install even if CBM GitHub release/download is unavailable.
- Resolution: warning-not-fatal for all companion failures, but warnings must be explicit and next steps must state how to retry or skip.

## Critical


## Medium


## Minor


## Missing searches


## Claims to weaken or remove
