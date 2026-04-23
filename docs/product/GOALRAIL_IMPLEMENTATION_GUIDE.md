---
id: goalrail_implementation_guide
title: Goalrail Implementation Guide
kind: architecture_canon
authority: operational
status: current
owner: architecture
truth_surfaces:
  - implementation_discipline
  - bounded_slices
lifecycle: active-core
review_after: 2026-07-19
supersedes: []
superseded_by: null
related_docs:
  - docs/product/GOALRAIL_BUILD_ROADMAP.md
  - docs/PROJECT_SPINE_SCHEMA.md
  - docs/ops/STATUS.md
---
# Goalrail Implementation Guide

> How to advance the project without scope drift.

## 1. Purpose

This guide defines how to build Goalrail through bounded slices.
It is intentionally operational rather than aspirational.
Goalrail implementation proceeds through `punk` as the repo delivery discipline.

## 2. Reading order

Always work in this order:
1. `docs/INDEX.md`
2. `docs/product/GOALRAIL_PRODUCT_BRIEF.md`
3. `docs/product/GOALRAIL_MVP_BLUEPRINT.md`
4. `docs/product/GOALRAIL_BUILD_ROADMAP.md`
5. `docs/PROJECT_SPINE_SCHEMA.md`
6. `docs/product/GOALRAIL_PARALLEL_EXECUTION_MODEL.md`
7. `docs/adr/ADR-0001-runtime-neutral-cli-first.md`
8. `docs/adr/ADR-0002-single-writer-and-advisory-panels.md`
9. `docs/ops/STATUS.md`
10. `docs/ops/NEXT.md`
11. `docs/ops/DECISIONS.md`
12. `docs/ops/COMPONENTS.yaml`

## 2A. Repo workspace shape

Use the repo with these boundaries:
- `docs/product/` — canonical product truth
- `docs/PROJECT_SPINE_SCHEMA.md` and `docs/adr/` — kernel support truth
- `docs/ops/` — current operating layer
- `work/` — repo-tracked goals, reports, and bounded slice memory
- `knowledge/` — advisory research and idea backlog
- `publishing/` — public narrative drafts, receipts, and manual metrics
- `flows/` — planned flow/spec boundary for future runtime semantics
- `evals/` — planned eval/spec boundary for future verification semantics
- `apps/`, `scripts/`, `.github/` — parked implementation surfaces until a bounded slice makes them real

These boundaries mirror the repo discipline used in `punk` without implying that Goalrail already has a working runtime.

## 3. Core project laws

1. One spine, two planes, one outcome.
2. Approved contracts are immutable.
3. Runtime may execute; gate decides; proof preserves.
4. One contract maps to one repo binding.
5. Task scope must stay inside contract scope.
6. Final verdict is written only by gate.
7. Proof is immutable.
8. Parallel execution is explicit and scheduler-controlled.
9. Runtime neutrality is CLI-first by default.
10. One writable run uses one primary writer runtime.
11. Advisory panels are separate from task execution groups.
12. Real implementation work proceeds through `punk`.

## 4. Bounded slice rule

Each implementation slice must answer:
- what is the exact goal?
- which phase/checkpoint does it belong to?
- what is in scope?
- what is explicitly out of scope?
- what proof will show that it is done?

If a slice cannot answer those questions, it is too vague.

## 5. Required updates after a completed slice

Every completed slice should update:
- `docs/ops/STATUS.md`
- `docs/ops/NEXT.md`
- `docs/ops/DECISIONS.md` when a stable rule changed
- `docs/ops/COMPONENTS.yaml` when the component map changed

## 6. How to choose the next slice

Use this filter:
1. Does it move the active phase toward its checkpoint?
2. Does it strengthen boundedness, reliability, inspectability, or product usability?
3. Can it leave behind visible proof?

If not, cut it or defer it.

## 7. Research-to-punk loop

Every implementation slice should pass through this loop before `punk` starts building:
1. select one roadmap slice or checkpoint target
2. decompose it into one bounded research question
3. inspect donor patterns, risks, and options only for that bounded question
4. write the exact handoff artifact needed for the slice, usually a schema note, ADR, or exact task brief
5. hand `punk` a precise goal, scope, non-goals, and proof target

Rule:
- roadmap items are too coarse to hand directly to `punk`
- each roadmap step should first be decompressed into a small, explicit delivery target
- if the exact goal and proof cannot be written down, stop and clarify before implementation

## 8. Editing rules

### Rule 1 — brief first, blueprint second
If product thesis changes:
1. update `GOALRAIL_PRODUCT_BRIEF.md`
2. update `GOALRAIL_MVP_BLUEPRINT.md`
3. update roadmap or ops docs if necessary

### Rule 2 — no silent architecture drift
If boundaries between intent, runtime, gate, proof, advisory panels, or parallel scheduling change:
- update the blueprint in the same change
- update decisions if the rule is stable

### Rule 3 — roadmap and next are different
- roadmap = phases and checkpoints
- next = immediate bounded slices only

### Rule 4 — keep names neutral and product-owned
Use Goalrail-native naming.
Avoid accidental carry-over naming from reference implementations.

## 9. Working style

Recommended working cadence:
- choose one bounded slice
- implement through `punk`
- test / demo / inspect
- update ops docs
- only then pick the next slice
