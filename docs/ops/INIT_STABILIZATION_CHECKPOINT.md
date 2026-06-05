---
id: goalrail_init_stabilization_checkpoint
title: Goalrail Init Stabilization Checkpoint
kind: ops_status
authority: operational
status: current
owner: ops
truth_surfaces:
  - init_stabilization
  - init_known_risks
  - init_next_safe_slices
lifecycle: active-core
review_after: 2026-08-08
supersedes: []
superseded_by: null
related_docs:
  - docs/INDEX.md
  - docs/product/GOALRAIL_MVP_BLUEPRINT.md
  - docs/product/GOALRAIL_PROJECT_SCAN_AND_CONTEXT_PACK_V0.md
  - docs/ops/INIT_LIFECYCLE.md
  - docs/ops/SNAPSHOT_SCAN_SHARED_SHAPE.md
  - docs/ops/STATUS.md
  - docs/ops/NEXT.md
  - docs/ops/COMPONENTS.yaml
---
# Goalrail Init Stabilization Checkpoint

## Scope

This checkpoint records the completed `goalrail init` stabilization sequence.
It is limited to init bootstrap behavior, local marker safety, advisory
snapshot / Project Scan handling, and shared repository-shape signal guardrails.

It is not a broad platform checkpoint and does not expand Goalrail runtime,
server, Project Scan, runner, gate, proof, checkout, provider, source upload, or
database scope.

## Completed slices

| Slice | Result |
| --- | --- |
| INIT-00-DOCS | Documented the current `goalrail init` lifecycle and recovery semantics. |
| INIT-01-HTTP | Bounded server-backed init HTTP requests with timeout / retry behavior. |
| INIT-02-MARKER | Hardened `.goalrail/project.yml` parsing and writes. |
| INIT-03-STEP-RESULTS | Added init `status` / `steps`, moved snapshot after marker, and mapped recovery states. |
| INIT-03A-RETRY-COMMAND-CONTEXT | Preserved effective `--repo`, `--base`, and `--project` context in retry hints. |
| INIT-04-RESCAN-ALIAS | Added `goalrail project scan --refresh` as the refresh alias. |
| INIT-05-SNAPSHOT-SCAN-PARITY | Added snapshot / Project Scan shared-signal parity tests and documented accepted path-model divergence. |
| INIT-06-SNAPSHOT-SCAN-SHARED-SHAPE-DESIGN | Documented the shared repository-shape direction. |
| INIT-07-SHARED-SHAPE-SIGNAL-TABLE | Added `apps/cli/internal/reposhape` and routed snapshot inventory plus Project Scan detection through shared signal definitions. |
| INIT-08-PROJECT-SCAN-SUMMARY | Added compact human Project Scan summary to successful server-backed `goalrail init` output. |

## Stabilized behavior

- Server-backed init has bounded HTTP behavior.
- Local marker read/write handling is hardened.
- Server-backed init output has machine-readable `status` and `steps`.
- Default `goalrail init` writes or verifies the local marker before advisory
  snapshot work.
- Advisory snapshot and Project Scan failures after the marker are represented
  as warnings rather than marker-less bootstrap failures.
- Retry hints preserve the effective init context.
- `goalrail project scan --refresh` exists as the preferred local rescan alias.
- Snapshot and Project Scan now have parity guardrails and shared signal
  definitions for metadata-only repository-shape signals.
- Successful server-backed `goalrail init` prints compact Project Scan
  baseline / overlay / repository-shape / partiality / freshness facts from the
  existing best-effort local scan path.

## Remaining known risks

- There is no dedicated marker repair command yet.
- Directory fsync portability may need best-effort handling on unsupported
  platforms.
- Cancellation semantics for advisory snapshot / Project Scan work may need a
  narrow cleanup.
- The repository context snapshot remains client-advisory metadata, not trusted
  server truth.
- The shared `reposhape` package must stay bounded and metadata-only.
- There is no full collector unification yet.

## Explicit non-goals

This checkpoint does not add or approve:

- server clone;
- raw source upload by default;
- runner, gate, proof, checkout, provider integration, or runtime execution
  behavior;
- server snapshot sync from Project Scan;
- a repair command;
- Project Scan schema changes;
- snapshot schema changes;
- server API changes;
- database changes.

## Recommended next safe slices

1. Narrow cancellation semantics cleanup for advisory snapshot / Project Scan,
   if current behavior proves confusing or leaky.
2. Marker repair design note before any repair command implementation.
3. Optional small collector spike only after deciding whether the MVP needs
   collector unification beyond the current shared signal definitions.
