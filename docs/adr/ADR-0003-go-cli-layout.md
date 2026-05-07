# ADR-0003 — Go CLI layout and canonical binary

Status: accepted
Date: 2026-04-25

## Context

Goalrail needs its first bounded CLI architecture slice for delivery/runtime-facing workflows.
The repository is still documentation-first, and this slice must not imply that Goalrail has a production backend, hosted execution plane, gate implementation, or server-generated proof.

The CLI must provide a small Go-idiomatic foundation for the current product surfaces:
- Delivery / Readiness
- Contract
- Proof

## Decision

Goalrail CLI implementation lives under `apps/cli` as a separate Go module.

The canonical binary name is `goalrail`.

### Layout

The CLI uses idiomatic Go application layout:
- `apps/cli/cmd/goalrail` for the executable entrypoint
- `apps/cli/internal/*` for private CLI packages

The first slice keeps `cmd/goalrail/main.go` thin and routes business/domain behavior into internal packages.

### Naming

- `goalrail` is the canonical binary name.
- `gr` may be introduced later as an optional short alias.
- `gls`, `glr`, and `gor` are not canonical names.

### Scope of this slice

This slice may include local/demo command boundaries for:
- `goalrail init`
- `goalrail readiness scan`
- `goalrail contract validate`
- `goalrail proof show`
- `goalrail version`

It does not implement production server integration, hosted execution, production repository authorization, gate logic, or proof generation.

## Consequences

### Positive

- The CLI has a real compiling foundation without framework sprawl.
- Command boundaries match the current product surfaces.
- The repository can test local readiness, contract validation, and proof rendering behavior without claiming production runtime capability.

### Negative

- The CLI is not yet connected to server-side state.
- `init` can only emit a repo binding draft until server-side key provisioning and RepoBinding sync exist.
- `proof show` can only render provided proof JSON until server proof retrieval exists.

## Not now

This ADR does not imply:
- production repo auth or deploy-key provisioning in the CLI
- a Goalrail server API
- hosted execution
- final gate verdict writing
- server-generated proof
- `gr` alias support in this slice

## Update - 2026-05-07

The original 2026-04-25 slice described `goalrail init` as local/demo and
draft-only. Current implementation has moved beyond that initial slice:
server-backed `goalrail init` now initializes repository context through the
authenticated server API, writes a non-secret `.goalrail/project.yml` marker,
ensures `.goalrail/.gitignore`, and runs a local Project Scan cache write.
Plain init also records a metadata-only repository context snapshot.

This update preserves the historical ADR context; it does not broaden MVP
scope. The current init lifecycle, trust boundary, and partial-failure recovery
direction are documented in `docs/ops/INIT_LIFECYCLE.md`. The local
RepositoryBaselineProfile / WorkspaceOverlay lifecycle is governed by
`docs/adr/ADR-0025-repository-baseline-profile-lifecycle.md`.

The updated init path still does not imply production repository
authorization, checkout, server clone, raw source upload by default, hosted
execution, runner integration, gate logic, or proof generation.
