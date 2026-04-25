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
