---
id: project_spine_schema_note
title: Project Spine Schema Note
kind: architecture_canon
authority: canonical
status: current
owner: architecture
truth_surfaces:
  - project_spine_schema
  - canonical_objects
lifecycle: active-core
review_after: 2026-07-19
supersedes: []
superseded_by: null
related_docs:
  - docs/product/GOALRAIL_MVP_BLUEPRINT.md
  - docs/product/GOALRAIL_PARALLEL_EXECUTION_MODEL.md
  - docs/adr/ADR-0001-runtime-neutral-cli-first.md
  - docs/adr/ADR-0002-single-writer-and-advisory-panels.md
---
# Project Spine Schema Note

> Compact schema note for the canonical Goalrail kernel.

## 1. Purpose

This note defines the minimum Goalrail spine needed before Phase 1 implementation.
It is not a full database design.
It exists to make canonical objects, derived views, and ownership boundaries explicit before code starts.

Read together with:
- `docs/product/GOALRAIL_MVP_BLUEPRINT.md`
- `docs/product/GOALRAIL_BUILD_ROADMAP.md`
- `docs/product/GOALRAIL_PARALLEL_EXECUTION_MODEL.md`
- `docs/adr/ADR-0001-runtime-neutral-cli-first.md`
- `docs/adr/ADR-0002-single-writer-and-advisory-panels.md`

## 2. Canonical vs derived

### Canonical objects
Canonical objects are persisted truth.
They have stable IDs, explicit ownership, and append-only event history.

### Derived views
Derived views are projections over canonical objects and events.
They may be rebuilt.
They must never become hidden sources of truth.

## 3. Canonical objects

| Object | Purpose | Authoritative writer |
|---|---|---|
| `Project` | top-level delivery container | system / setup flow |
| `RepoBinding` | binds one contract path to one repo | contract shaping |
| `Goal` | normalized business request | intent plane |
| `Constraint` | explicit hard limits or requirements | intent plane |
| `GlossaryTerm` | shared domain vocabulary | intent plane |
| `Contract` | approved bounded delivery agreement | contract shaping |
| `Task` | executable bounded unit from contract | contract shaping |
| `TaskRiskAssessment` | explicit risk level and rationale | contract shaping |
| `TaskRoutingPolicy` | default runtime/review posture for a task | contract shaping / policy engine |
| `Run` | one writable execution attempt | runtime |
| `AdvisoryPanel` | bounded multi-runtime advisory process | advisory layer |
| `ConsensusRecord` | normalized advisory output | advisory layer |
| `Decision` | final machine-readable verdict | gate |
| `Proof` | inspectable acceptance artifact | gate |
| `RecoveryRecord` | blocked / escalated / next-action record | runtime or gate |
| `Learning` | durable lesson or feedback outcome | feedback loop |
| `Artifact` | blob or document reference | producing layer |
| `Event` | append-only state transition envelope | all layers |

## 4. Derived views

| View | Purpose | Derived from |
|---|---|---|
| `WorkLedgerView` | current state, latest refs, next action | Project, Contract, Task, Run, Decision, Proof, Event |
| `GroupSummary` | execution-group summary after barrier | ExecutionGroup-related events and artifacts |
| `PanelSummary` | advisory split/consensus summary | AdvisoryPanel, ConsensusRecord, advisory artifacts |
| `RoutingRecommendation` | human-facing explanation of routing choice | TaskRiskAssessment, TaskRoutingPolicy, policy inputs |

## 5. Core relations

Canonical chain:

```text
Project
  -> RepoBinding
  -> Goal
  -> Contract
  -> Task
  -> TaskRiskAssessment
  -> TaskRoutingPolicy
  -> Run
  -> Decision
  -> Proof
  -> Learning
```

Supporting relations:
- `Goal -> Constraint[]`
- `Goal -> GlossaryTerm[]`
- `Task -> AdvisoryPanel[]`
- `AdvisoryPanel -> ConsensusRecord`
- `Run -> Artifact[]`
- `Decision -> Proof`
- `Run | Decision -> RecoveryRecord[]`
- `Any canonical object -> Event[]`

## 6. Ownership boundaries by phase

### Intent Plane writes
- `Goal`
- `Constraint`
- `GlossaryTerm`
- intent artifacts such as Goal Packet and Contract Seed refs

### Contract, Risk & Task Shaping writes
- `RepoBinding`
- `Contract`
- `Task`
- `TaskRiskAssessment`
- `TaskRoutingPolicy`

### Runtime writes
- `Run`
- runtime receipts and execution artifacts
- baseline snapshot refs
- runtime-originated `RecoveryRecord`

### Advisory layer writes
- `AdvisoryPanel`
- `ConsensusRecord`
- advisory artifacts and summaries

### Gate writes
- `Decision`
- `Proof`
- gate-originated `RecoveryRecord`

### Feedback writes
- `Learning`

Rule:
- no layer may silently overwrite another layer's canonical object
- final verdict is written only by gate

## 7. Minimum event envelope

```json
{
  "id": "evt_123",
  "type": "task.created",
  "project_id": "prj_1",
  "entity_type": "Task",
  "entity_id": "tsk_9",
  "actor": {
    "kind": "system|user|runtime|gate",
    "id": "runtime_codex"
  },
  "timestamp": "2026-04-14T12:00:00Z",
  "payload": {},
  "artifact_refs": ["art_7"],
  "causation_id": "evt_122",
  "correlation_id": "run_33"
}
```

Rules:
- events are append-only
- events reference canonical entities, not replace them
- materialized views rebuild from canonical state plus events

## 8. Minimum artifact ref shape

```json
{
  "id": "art_7",
  "kind": "goal_packet|contract_seed|receipt|baseline|panel_summary|proof",
  "uri": "object://goalrail/artifacts/art_7.json",
  "content_type": "application/json",
  "sha256": "...",
  "created_by": "gate"
}
```

## 9. Kernel rules this schema must preserve

1. approved contracts are immutable
2. one writable run uses one primary writer runtime
3. advisory panels are separate from task execution groups
4. gate reads frozen verification inputs
5. canonical truth stays separate from derived views

## 10. Out of scope for this note

- full relational schema
- database migrations
- API contracts
- UI view models
- hosted queue design
- provider-specific adapter internals
