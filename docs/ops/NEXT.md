# Goalrail Next

> Only the next bounded slices. Keep this file short.

## Active phase

- **Phase 1 — Spine / schemas / events**

## Next bounded slices

### Slice 1 — Spine package bootstrap
Goal:
- create first implementation package for core domain types and events

Done means:
- IDs, enums, object skeletons, and event envelope compile
- basic serialization / validation tests exist

### Slice 2 — Work ledger and routing schemas
Goal:
- define exact shapes for `WorkLedgerView`, `TaskRiskAssessment`, and `TaskRoutingPolicy`

Done means:
- canonical vs derived boundaries stay explicit
- risk and routing records are precise enough for code generation

### Slice 3 — Intent output schemas
Goal:
- define exact `Goal Packet` and `Contract Seed` JSON schemas

Done means:
- intent output can be validated deterministically

### Slice 4 — Runtime registry kernel note
Goal:
- define the minimum runtime registry and capability schema needed by Phase 4

Done means:
- runtime discovery, auth status, and capability fields are explicit
- no provider-specific behavior leaks into the kernel schema

## Deferred until later

- hosted execution
- tracker integrations
- multi-runtime advisory implementation
- external checks implementation
- analytics / admin product
