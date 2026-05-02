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
  - docs/adr/ADR-0019-workitem-planning-controller-runner-boundary.md
  - docs/adr/ADR-0020-public-contract-identity-boundary.md
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

Repo binding is initially manual or dev-seeded. GitHub/GitLab/Bitbucket
integration comes later and is not required for the first server persistence
slice.

## 2. Architectural principles

1. One spine, two planes, one outcome.
2. Approved contract is immutable.
3. Runtime may execute; gate decides; proof preserves.
4. Execution setup is user-owned; verification and proof are Goalrail-owned.
5. Append-only events + materialized views.
6. Canonical objects and derived views stay explicit.
7. Runtime capabilities are wrapped behind adapters.
8. CLI / subscription-backed runtimes are first-class integration targets.
9. One Contract = 1 RepoBinding.
10. Task scope must be a subset of contract scope.
11. Final verdict is written only by gate.
12. Parallelism is explicit and scheduler-controlled.
13. One task, one run, one workspace lineage.
14. One writable run uses one primary writer runtime.
15. One task may use many advisory runtimes.
16. Sensitive policy may override risk-based fan-out.
17. API server owns canonical state, but does not clone repositories or run
    checks in-process.
18. API server validates and persists canonical planning state, but repo-aware
    WorkItem planning computation belongs behind worker / controller / runner
    boundaries.

## 3. Layer map

### Layer 1 — Spine / State Truth
Canonical objects:
- User
- Organization
- OrganizationMembership
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

Initial Project / repo context:
- Goalrail Organization is an internal SaaS tenant/workspace, not a GitHub
  Organization, GitLab Group, or Bitbucket Workspace
- Project is a delivery container inside an Organization
- Project is not a repository
- RepoBinding stores the repository reference directly in the MVP
- RepositoryRecord is deferred until repository catalog, repo-level policy, or
  independent provider sync is needed
- VcsConnection is a future provider connection layer, not required for the
  first code slice

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

Public Contract identity:
- public product/control API should expose one stable `Contract` lifecycle
  aggregate and one stable public `contract_id`
- `ContractSeed`, `ContractDraft`, and `ApprovedContract` remain precise
  internal lifecycle records for bridge, draft, and immutable approved snapshot
  semantics
- public API uses `contracts/{id}` and contract subresources for
  contract creation, draft updates, submission, approval, tasks, plans, and
  proposals rather than exposing `contract-seeds`, `contract-drafts`, or
  `approved-contracts` as product-facing resource names
- the current server implements the smallest stable `contract_id` aggregate
  boundary and public `/v1/contracts` lifecycle façade routes; internal
  `ContractSeed`, `ContractDraft`, and `ApprovedContract` records still carry
  lifecycle precision behind the public Contract view

Control-plane split:
- the API server is the canonical state machine for planning transitions,
  validation, persistence, and event append
- repo-aware task decomposition is not performed in-process by the API server
- workers / controllers may reconcile planning requests but do not own canonical
  truth or write directly to the database
- runners / planners may compute WorkItem plan proposals using bounded
  repo/context/knowledge inputs behind the runner boundary
- canonical `WorkItem(planned)` records are created only through API-server
  acceptance of validated planning output

Conceptual rich planning flow:

```text
ApprovedContract(approved)
  -> WorkItemPlanningRequest(queued)
  -> worker/controller leases or reconciles request
  -> runner/planner obtains bounded repo/context/knowledge snapshot
  -> WorkItemPlanProposal(submitted)
  -> API server validates and stores proposal
  -> explicit acceptance creates WorkItem(planned) records
```

This flow is a target architecture direction, not an implemented MVP surface.
The current direct one-WorkItem planning endpoint remains simple prototype v0
behavior and must not be expanded into rich repo-aware planning inside the API
server.
If this target flow later becomes public REST API, the public URL vocabulary
should use product-facing resources such as `contracts`, `plans`, `proposals`,
and `tasks`; internal domain names such as `ApprovedContract`, `WorkItem`, and
`WorkItemPlanProposal` do not need to appear verbatim in public paths.

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
Organization
  -> Project
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

Runtime adapters must stay execution-neutral.

They may pass bounded task packets, launch or register runs, collect receipts,
capture artifacts, and report runtime capabilities. They must not encode
provider-specific prompt doctrine, manage user skills, or make the kernel depend
on one provider's agent configuration model.

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
- full SaaS onboarding/auth as the first persistence slice
- GitHub/GitLab/Bitbucket implementation before manual/dev-seeded RepoBinding
- repository checkout in the API server
