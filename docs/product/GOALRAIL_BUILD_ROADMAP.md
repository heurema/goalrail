---
id: goalrail_build_roadmap
title: Goalrail Build Roadmap
kind: ops_plan
authority: operational
status: current
owner: delivery
truth_surfaces:
  - build_sequence
  - checkpoints
lifecycle: active-core
review_after: 2026-07-19
supersedes: []
superseded_by: null
related_docs:
  - docs/product/GOALRAIL_MVP_BLUEPRINT.md
  - docs/product/GOALRAIL_IMPLEMENTATION_GUIDE.md
  - docs/ops/STATUS.md
---
# Goalrail Build Roadmap

> Phase roadmap with checkpoints and proof-oriented exits.

## 1. Roadmap rules

### Rule 1 — one bounded slice at a time
One main slice is active at a time at roadmap level.
Parallel work is allowed only inside the active phase boundary.

All implementation slices in this repo proceed through `punk`.

### Rule 2 — phase exits require checkpoints
A phase is complete only when its checkpoint is satisfied.

### Rule 3 — no silent expansion
If a new major area appears:
- cut it into a later phase
- or update the roadmap and decision log explicitly

### Rule 4 — every phase ends with proof
Minimum acceptable phase proof:
- runnable demo
- deterministic CLI flow
- inspectable web artifact
- or clear verification/proof output

## 2. Phase order

```text
Phase 0  -> Project rails and docs
Phase 1  -> Spine / schemas / events
Phase 2  -> Intent Plane
Phase 3  -> Contract, Risk & Task Shaping
Phase 4  -> Local Runtime
Phase 4B -> Parallel Task Execution & Isolation
Phase 5  -> Gate / Verify / Proof
Phase 6  -> Web + CLI coherence
Phase 7  -> Pilot packaging + tracker sync
Phase 8  -> Advisory panels + review adapters + external checks
```

## Phase 0 — Project rails and docs

### Goal
Make the project governable before implementation grows.

### Deliverables
- docs index
- product brief
- blueprint
- roadmap
- ops docs

### Checkpoint
**C0 — Project rails are real**

Done means:
- reading order is canonical
- current status exists
- next bounded slices exist
- component map exists
- first implementation phase is selected

## Phase 1 — Spine / schemas / events

### Goal
Build the minimum trusted core.

### In scope
- canonical objects
- canonical vs derived views
- IDs and relations
- event envelope
- artifact refs
- minimal materialized views
- recovery and routing records

### Checkpoint
**C1 — Core objects compile and persist**

Done means:
- core domain types compile
- event envelope exists
- serialization and validation tests exist
- canonical vs derived state is explicit
- a WorkLedgerView-style current-state projection is specified

## Phase 2 — Intent Plane

### Goal
Turn vague requests into structured delivery input.

### In scope
- Goal Packet
- Contract Seed
- clarification rules
- constraints and glossary extraction
- readiness check for handoff

### Checkpoint
**C2 — Goal can become a handoff packet**

Done means:
- one raw goal can be clarified deterministically
- Goal Packet is persisted
- Contract Seed is generated
- handoff readiness is visible

## Phase 3 — Contract, Risk & Task Shaping

### Goal
Turn handoff input into an approved bounded contract and a task plan.

### In scope
- repo binding selection
- contract drafting
- contract validation
- approval flow
- task decomposition
- task risk assessment
- routing defaults by task profile
- execution packet redaction rules

### Checkpoint
**C3 — Approved contract produces executable tasks**

Done means:
- contract versioning exists
- contract validation rejects bad scope or check sets
- one approved contract produces a bounded task plan
- each task has a visible risk level and routing recommendation

## Phase 4 — Local Runtime

### Goal
Execute one bounded task in one isolated workspace.

### In scope
- local runner CLI
- runtime registry and auth discovery
- workspace preparation
- execution packet builder
- one primary runtime adapter
- baseline capture
- receipt writing

### Checkpoint
**C4 — One task becomes one run with one receipt**

Done means:
- task can be leased and executed
- runtime availability and auth status are inspectable
- workspace lineage is recorded
- changed files are explicit
- checks run and a receipt is written
- baseline state is preserved for later verification

## Phase 4B — Parallel Task Execution & Isolation

### Goal
Run independent tasks concurrently without unsafe shared writes.

### In scope
- execution groups
- dependency typing
- scheduler rules
- isolation resolver
- worktree parallel path
- docker isolation path
- fan-in barrier
- group summary

### Checkpoint
**C4B — Independent tasks run in parallel safely**

Done means:
- disjoint tasks can run concurrently in separate worktrees
- overlapping or uncertain tasks are forced into stronger isolation or serialization
- blocking and non-blocking work are distinguished explicitly
- every execution group ends with a barrier before downstream verification

## Phase 5 — Gate / Verify / Proof

### Goal
Produce final trustworthy verdicts and proof.

### In scope
- frozen verification bundle
- baseline snapshot
- scope / target / integrity / policy evaluation
- repo invariants
- holdout checks
- deterministic decision synthesis
- proof creation

### Checkpoint
**C5 — Run becomes decision and proof**

Done means:
- a run can be gated
- baseline distinguishes regressions from pre-existing failures
- final verdict is machine-readable
- proof is inspectable in human and machine form

## Phase 6 — Web + CLI coherence

### Goal
Make the product usable by both planning and engineering roles.

### In scope
- project home
- goal workspace
- contract review view
- delivery board
- gate view
- proof view
- CLI flow coherence

### Checkpoint
**C6 — One visible product loop exists**

Done means:
- the same project can be followed from goal to proof in the product
- web and CLI share the same canonical objects

## Phase 7 — Pilot packaging + tracker sync

### Goal
Make the product sellable in a pilot.

### In scope
- pilot flow
- repo connection
- basic tracker import/export
- onboarding docs
- proof sharing

### Checkpoint
**C7 — Pilot workflow is demoable and repeatable**

Done means:
- a team can see one full end-to-end flow
- the pilot setup is bounded and repeatable

## Phase 8 — Advisory panels + review adapters + external checks

### Goal
Add stronger multi-runtime review and policy surfaces without widening the kernel.

### In scope
- advisory panel protocols (`panel`, `quorum`, `verify`, `diverge`)
- review-only runtime adapters
- multi-runtime security / performance / architecture critique paths
- external policy check interface

### Checkpoint
**C8 — Advisory panels and external checks extend the trust model cleanly**

Done means:
- advisory panels can run against bounded or frozen inputs
- risk and policy determine review depth cleanly
- final authority still belongs to Gate
- external checks enrich trust without widening the kernel
