---
id: goalrail_mvp_blueprint
title: Goalrail MVP Blueprint
kind: architecture_canon
authority: canonical
status: current
owner: architecture
truth_surfaces:
  - mvp_architecture
  - project_spine
  - runtime_model
  - verification_model
lifecycle: active-core
review_after: 2026-07-19
supersedes: []
superseded_by: null
related_docs:
  - docs/product/GOALRAIL_PRODUCT_CONCEPT.md
  - docs/product/GOALRAIL_OPERATING_MODEL.md
  - docs/PROJECT_SPINE_SCHEMA.md
  - docs/product/GOALRAIL_BUILD_ROADMAP.md
---
# Goalrail MVP Blueprint

> Рабочий blueprint первого продаваемого продукта.

## 0. Working docs

Read together with:
- `docs/INDEX.md`
- `docs/product/GOALRAIL_PRODUCT_BRIEF.md`
- `docs/product/GOALRAIL_BUILD_ROADMAP.md`
- `docs/product/GOALRAIL_PARALLEL_EXECUTION_MODEL.md`
- `docs/ops/STATUS.md`
- `docs/ops/NEXT.md`

## 1. Product surfaces v0

### Surface A — Web: Intent & Oversight
For PM / analyst / tech lead.

Contains:
- goals
- clarification
- constraints
- glossary
- contract review
- delivery board
- gate view
- proof feed

### Surface B — CLI: Delivery Runtime
For dev / tech lead / CI.

Contains:
- task pull
- task show
- run start
- run status
- run submit
- proof show

### Surface C — Integrations
Thin settings surface for:
- repo binding
- tracker binding
- runtime registry and auth status
- sync status
- future external checks

## 2. Architectural principles

1. One spine, two planes, one outcome.
2. Approved contract is immutable.
3. Runtime may execute; gate decides; proof preserves.
4. Append-only events + materialized views.
5. Canonical objects and derived views stay explicit.
6. Runtime capabilities are wrapped behind adapters.
7. CLI / subscription-backed runtimes are first-class integration targets.
8. One Contract = 1 RepoBinding.
9. Task scope must be a subset of contract scope.
10. Final verdict is written only by gate.
11. Parallelism is explicit and scheduler-controlled.
12. One task, one run, one workspace lineage.
13. One writable run uses one primary writer runtime.
14. One task may use many advisory runtimes.
15. Sensitive policy may override risk-based fan-out.

## 3. Layer map

### Layer 1 — Spine / State Truth
Canonical objects:
- Project
- RepoBinding
- Goal
- Constraint
- GlossaryTerm
- Contract
- Task
- TaskRiskAssessment
- TaskRoutingPolicy
- Run
- AdvisoryPanel
- ConsensusRecord
- Decision
- Proof
- RecoveryRecord
- Learning
- Artifact
- Event

Derived views:
- WorkLedgerView
- GroupSummary
- PanelSummary
- RoutingRecommendation

Storage:
- Postgres for canonical object state
- object storage for artifacts
- append-only event log
- materialized views for current-state projections

### Layer 2 — Intent Plane
Produces:
- Goal Packet
- Contract Seed
- Handoff Record

Responsibilities:
- clarification
- scope in/out
- constraints
- glossary
- acceptance criteria

### Layer 3 — Contract, Risk & Task Shaping
Produces:
- approved Delivery Contract
- Task Plan
- TaskRiskAssessment
- Routing Recommendation

Responsibilities:
- repo selection
- scope selection
- checks
- risks
- policy defaults
- approval
- task decomposition
- routing defaults by task profile

### Layer 4 — Delivery Runtime
Produces:
- Run
- Receipt
- execution artifacts
- baseline snapshot

Responsibilities:
- runtime discovery and auth status
- workspace preparation
- bounded execution packet
- primary runtime adapter execution
- checks
- receipt writing

### Layer 4B — Parallel Task Execution
Produces:
- Execution Group
- Execution Group Plan
- IsolationDecision
- BarrierRecord

Responsibilities:
- group independent tasks
- classify blocking vs non-blocking work
- choose worktree or docker isolation
- enforce fan-in barrier before downstream verification

### Layer 4C — Advisory Panels & Consensus
Produces:
- AdvisoryPanel
- ConsensusRecord
- comparative evaluation artifacts

Responsibilities:
- route frozen bundles or bounded questions to multiple runtimes
- run `panel`, `quorum`, `verify`, or `diverge` protocols
- collect split/consensus signals
- keep all outputs advisory to gate

### Layer 5 — Gate / Verify / Proof
Produces:
- Decision
- Proof

Responsibilities:
- load frozen bundle
- account for baseline vs regression
- validate scope
- evaluate target / integrity / policy lanes
- synthesize final verdict
- build proof

### Layer 6 — Product Surfaces
Responsibilities:
- Web coherence
- CLI coherence
- pilot readiness
- role-specific views

## 4. Canonical object chain

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

Supporting objects:
- Constraint
- GlossaryTerm
- Artifact
- Event
- AdvisoryPanel
- ConsensusRecord
- RecoveryRecord

Derived views:
- WorkLedgerView
- GroupSummary
- PanelSummary
- RoutingRecommendation

## 5. Runtime model

Goalrail is runtime-neutral.

Rules:
- CLI / subscription-backed developer runtimes are the default integration model
- local and open-source runtimes must fit through the same adapter boundary
- raw API adapters are optional later extensions, not the default assumption
- one writable run uses one primary execution adapter
- one task may use one or more advisory runtimes in `panel`, `quorum`, `verify`, or `diverge` mode
- runtime-specific logic must stay outside the kernel
- sensitive work may restrict external or multi-vendor exposure

Initial adapter targets:
- Codex CLI
- Claude Code / Cloud Code
- Gemini CLI
- local / open-source runtimes

## 6. Risk and routing model

Risk levels:
- `low`
- `medium`
- `high`
- `critical`

Default routing posture:
- `low` -> one primary writer and one review lane by default
- `medium` -> one primary writer and two advisory reviews, or one review plus escalation-on-split
- `high` -> one primary writer and deeper advisory review, including `quorum`, `verify`, or `diverge` where justified
- `critical` -> policy-driven path, potentially `single-vendor-only`, `local-only`, or `human-signoff-required`

Rules:
- risk controls review depth, not authorship count inside one writable run
- policy may narrow exposure beyond what risk alone would suggest
- routing recommendations are inspectable and revisable

## 7. Verification model

Inputs to gate:
- approved contract
- frozen verification bundle
- baseline snapshot
- receipt and execution artifacts
- advisory panel outputs when present
- external check results when present

Lanes:
- scope
- target
- integrity
- policy

Verdicts:
- `accept`
- `block`
- `escalate`

Rules:
- `accept` is impossible if scope fails
- `accept` is impossible if integrity fails
- baseline distinguishes pre-existing failures from regressions
- advisory consensus may strengthen evidence but may not override scope or policy failures
- incomplete evidence should escalate or block, never silently pass
- holdout checks may exist outside the primary execution packet

## 8. MVP thin slice

The first sellable MVP must support:
1. create goal
2. clarify goal
3. review and approve contract
4. assign task risk and a default runtime route
5. execute one bounded run through one primary runtime
6. optionally run one advisory review lane on the frozen bundle
7. verify and produce proof
8. inspect the result in web and CLI

## 9. What is explicitly out of scope for MVP

- tracker replacement
- broad admin analytics
- large hosted execution plane
- full enterprise governance suite
- unrestricted multi-agent autonomy
- opaque internal memory platform
- API-first vendor orchestration as the default path
