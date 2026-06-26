# 04 Change Log

## 2026-06-26

- Started project-local RDLab run for Omnigent codebase-memory-mcp integration.
- Inspected Omnigent installer, setup command, installer tests, and remote host image behavior.
- Inspected codebase-memory-mcp PyPI wrapper, CLI install/config behavior, README install docs, and install plan support.
- Ran provider router doctor with probes:
  - `claude-sonnet`: ok
  - `claude-haiku`: ok
  - `vibe-default`: ok
  - `agy-default`: failed by returning empty stdout with exit 0
- Ran proposer pass on `claude-sonnet`.
- Ran skeptic pass on `vibe-default`.
- Updated recommendation from "run `install -y` unconditionally" to a safer split:
  - default warning-not-fatal binary install and `auto_index=true` check are acceptable;
  - full `codebase-memory-mcp install -y` needs prompt/guardrails because current CBM install can auto-delete existing indexes under `-y`.

## Snapshot inputs


## Added sources or claims


## Removed sources or claims


## Changed sources or claims


## Unchanged but important


## Failed checks
