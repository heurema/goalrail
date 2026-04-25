# ADR-0004 — Go server boundary and selected stack

Status: accepted
Date: 2026-04-25

## Context

Goalrail needs its first bounded server architecture slice so canonical state can later move out of local/demo helper surfaces and into a server-owned source of truth.

The repository already has a Go CLI under `apps/cli`. The CLI is an adapter/helper surface for delivery workflows and local inspection. It is not the canonical backend and must not become the hidden owner of Goalrail state.

This ADR defines the server boundary and the first stack choice without claiming intake, contract composition, durable storage, gate, proof, runtime execution, or background workers.

## Decision

The Goalrail server implementation lives under `apps/server` as a separate Go module.

The canonical server binary name is `goalrail-server`.

The server is the future owner of canonical Goalrail state, including:
- `IntakeRecord`
- `Goal`
- clarification state
- `ContractDraft`
- `ApprovedContract`
- `WorkItems` / `Tasks`
- `Events`
- `GateDecision` / `Decision`
- `Proof`

CLI, skills, web resources, and integrations are adapters or helper surfaces. They may submit, inspect, or render server state later, but they are not canonical state owners.

### Selected initial stack

The first server slice uses:
- `net/http` with `http.NewServeMux`
- `encoding/json`
- `log/slog`
- `github.com/caarlos0/env/v11`
- manual wiring through constructors
- standard library `testing` and `net/http/httptest`

### Scope of the first server slice

The first code slice is health/version only:
- `GET /livez`
- `GET /readyz`
- `GET /version`

`/livez` and `/readyz` are the Kubernetes-facing health endpoints. Readiness is always ok for now because the skeleton has no database or external dependency.

Source-neutral intake is the next meaningful server domain, but there is no intake endpoint in this PR.

## Explicitly not in this PR

This server bootstrap does not implement:
- intake API
- database
- event log persistence
- contract composer
- gate
- proof generation
- runtime execution
- background workers
- server-side repository clone/readiness
- authentication

## Future storage path

When durable canonical state starts, the preferred path is:
- `pgx/v5`
- `sqlc`
- `goose`
- Postgres event table and projection tables
- local filesystem or object storage later for artifacts
- Postgres outbox or `LISTEN` / `NOTIFY` before any external broker

## Rejected for this stage

The first skeleton deliberately rejects:
- web frameworks
- ORMs
- dependency-injection containers
- brokers
- protobuf
- validation frameworks
- metrics or tracing libraries
- database clients and migrations

## Consequences

### Positive

- Goalrail now has a real server boundary without framework sprawl.
- The CLI remains an adapter/helper surface rather than a backend substitute.
- Health and version behavior can be tested without pretending canonical domain state exists.
- Future intake work has a clear home under `apps/server`.

### Negative

- The server is not useful as a product API yet.
- Readiness cannot validate dependencies until dependencies actually exist.
- Storage and eventing remain architectural decisions for a later bounded slice.
