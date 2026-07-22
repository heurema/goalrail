# Decouple Canary Runtime Artifacts from OpenSpec

- **Status:** proposed for owner approval through pull-request merge
- **Date:** 2026-07-22
- **Decision owner:** Goalrail owner

## Context

Archiving `intent-canary-v0` exposed two dependencies on its active OpenSpec change directory: the command defaulted evidence to that directory, and tests treated its frozen manifest as a runtime artifact. OpenSpec is a replaceable development-time compiler/provider, while canary evidence and its frozen measurement contract must remain available after a change is archived.

## Options

1. Read runtime artifacts from the dated OpenSpec archive. This is a small edit, but permanently couples runtime behavior to an OpenSpec lifecycle path and archive date.
2. Give runtime-owned canary artifacts a stable provider-neutral path, while retaining archived OpenSpec artifacts as historical adapter fixtures. This adds one explicit boundary and keeps the archive immutable.
3. Leave the completed changes active. This avoids relocation now, but leaves their lifecycle open and makes future runtime continuity depend on never archiving them.

## Evidence

- `go test ./...` failed after the archive because the default manifest fixture and adapter test still addressed `openspec/changes/intent-canary-v0`.
- The confirmed design requires canonical runtime semantics to remain provider-neutral and describes OpenSpec as replaceable.
- The real canary is still `frozen_not_activated`, so no production evidence needs migration.

## Decision

Use `canary/intent-canary-v0` as the stable location for the frozen manifest and default append-only evidence path. Keep `intent.md` and `proposal.md` in the dated OpenSpec archive and use them only as a historical OpenSpec adapter fixture. Do not resolve runtime paths by scanning archive directories.

The pull request implementing this record is the reversible canary. The decision becomes accepted when the owner merges it.

## Rejected Objections

- Duplicating the frozen manifest between the historical archive and the stable runtime path is acceptable because the archive is evidence of the completed change, while the stable copy is the current runtime contract. Future material rule changes create a new manifest version rather than editing archived history.
- A generic artifact registry is unnecessary for one canary and would add a new abstraction before measured need.

## Rollback

Revert the pull request before activation. No real event migration or external cleanup is required.

## Revisit Condition

Revisit the directory contract before a second canary family, an external repository pilot, or any move from repository-local evidence to durable hosted storage.
