# ADR-0019 — WorkItem planning controller / runner boundary

Status: accepted
Date: 2026-05-02

## Context

ADR-0018 defined the first WorkItem planning boundary:

```text
ApprovedContract(approved) -> WorkItem(planned)
```

The current server implementation now replaces the earlier simple direct
planning prototype with public `plans` / `proposals` / `acceptance` API routes.
The API server owns the state transitions and materializes accepted proposals
into canonical `WorkItem(planned)` records, using Postgres when configured with
in-memory fallback otherwise.

It must not become the final architecture for rich repo-aware planning.

Rich planning can require repository context, code inspection, knowledge
retrieval, policy context, risk classification, prior decisions, or other
bounded inputs. ADR-0008 already says repository checkout, workspace
preparation, code inspection, and check execution belong behind runner
boundaries, not inside the primary API server process.

Goalrail needs the same control-plane split for WorkItem planning:

- the API server is the canonical state machine
- Postgres is the canonical state store
- workers / controllers reconcile desired state and coordinate background work
- runners / planners do bounded repo/context/knowledge work
- canonical WorkItems are accepted and persisted through the API server

The analogy is Kubernetes-style control plane architecture:

| Kubernetes concept | Goalrail concept |
| --- | --- |
| API server | Goalrail API server |
| etcd | Postgres / canonical state store |
| controller / operator / reconciler | Goalrail worker / controller |
| kubelet / job runner | Goalrail runner / planner execution side |

## Decision

The Goalrail API server owns canonical state, validation, persistence, event
append, REST surfaces, and transition authority.

WorkItem planning computation that needs repository checkout, repository
context, code inspection, knowledge retrieval, or other bounded execution-side
inputs belongs to a worker / controller / runner boundary, not in the API server
process.

The API server may expose or lease planning work, accept planning proposals,
validate proposals against approved canonical state, persist proposal state, and
create canonical `WorkItem(planned)` records after explicit acceptance.

The API server must not:

- clone repositories
- mount workspaces
- inspect source trees in-process
- run checks or runtime commands
- perform rich repo-aware WorkItem decomposition in-process
- let workers, runners, CLIs, skills, or integrations write canonical WorkItems
  directly to the database

Workers / controllers may observe or lease server state and reconcile planning
requests. They coordinate planning jobs but do not own canonical truth and do
not write directly to the canonical database.

Runner / planner execution-side components may perform bounded
repo/context/knowledge work and compute `WorkItemPlanProposal` records. They
submit those proposals through the API server. They do not create canonical
WorkItems directly, write final `GateDecision`, create `Proof`, or become the
canonical source of truth.

## Target planning flow

The intended rich planning flow is:

```text
ApprovedContract(approved)
  -> WorkItemPlanningRequest(queued)
  -> worker/controller leases or reconciles the request
  -> runner/planner obtains bounded repo/context/knowledge snapshot
  -> WorkItemPlanProposal(submitted)
  -> API server validates and stores the proposal
  -> explicit acceptance creates WorkItem(planned) records
```

PR #53 implements the current public control-plane API for this flow with
short product-facing resources: `POST /v1/contracts/{id}/plans`,
`GET /v1/plans/{id}`, `POST /v1/plans/{id}/proposals`,
`GET /v1/proposals/{id}`, `POST /v1/proposals/{id}/acceptance`, and
`GET /v1/tasks/{id}`. The current implementation names the public queued
planning request `WorkItemPlan`; long internal type names such as
`work-item-planning-requests` or `work-item-plan-proposals` should remain
implementation vocabulary, not URL vocabulary.

Recommended initial posture:

- one accepted proposal may create one or more `WorkItem(planned)` records
- explicit acceptance is preferred before canonical WorkItems are created
- auto-accept may be a later policy, but it must be explicit policy, not an
  implicit side effect
- proposal validation must compare proposals to `ApprovedContract(approved)`
  scope, acceptance criteria, proof expectations, RepoBinding context, and any
  later policy constraints
- proposal submission and acceptance are state transitions through the API
  server, not direct database writes from a worker or runner

## Relationship to ADR-0018

This ADR qualifies ADR-0018.

ADR-0018 remains the source for the non-executable `WorkItem(planned)` concept
and for the earlier simple v0 direct planning prototype. The public
`POST /v1/contracts/{id}/tasks` endpoint has been removed in favor of
`POST /v1/contracts/{id}/plans`, `POST /v1/plans/{id}/proposals`, and
`POST /v1/proposals/{id}/acceptance`.

However, ADR-0018's one-WorkItem direct planning path must not be expanded into
repo-aware decomposition inside the API server. Rich planning belongs behind the
planning request / proposal / acceptance boundary described here.

## Relationship to ADR-0008

ADR-0008 remains the repository checkout boundary.

If WorkItem planning needs repository checkout, repository inspection, file
context, baseline snapshots, or command execution, that work must happen behind
a runner boundary. The API server may issue bounded planning requests and accept
returned proposals, but it must not perform checkout or source-tree inspection
in-process.

## Relationship to assignment / claiming

Assignment and claiming should wait until the planning request / proposal /
acceptance boundary is clarified.

`WorkItem(planned)` remains non-executable. Planning proposal submission is not
assignment, not claiming, not execution, and not a runtime task packet.

## Proposed conceptual objects

These names are conceptual and do not require an implementation in this ADR:

### `WorkItemPlanningRequest`

Represents server-owned desired planning work for an
`ApprovedContract(approved)`.

Recommended conceptual states:

- `queued`
- `leased`
- `proposal_submitted`
- `accepted`
- `rejected`
- `cancelled`

### `WorkItemPlanProposal`

Represents planner-produced proposed WorkItems and supporting rationale.

Potential fields:

- `id`
- `planning_request_id`
- `approved_contract_id`
- `repo_binding_id`
- `proposed_work_items`
- `source_snapshot_refs`
- `planner_identity`
- `rationale`
- `risk_notes`
- `created_at`

Proposal fields are not canonical WorkItems until accepted through the API
server.

### `WorkItemPlanAcceptance`

Represents explicit acceptance of a proposal into canonical
`WorkItem(planned)` records.

## Non-goals

This ADR does not define or implement:

- queue, outbox, broker, or event bus
- runner code
- checkout jobs
- runtime registry
- assignment
- claiming
- execution
- `Run`
- receipt submission
- `GateDecision`
- `Proof`
- web UI
- auto-accept policy

## Consequences

### Positive

- Keeps the API server as a state machine and validation boundary.
- Preserves the API server / runner split from ADR-0008.
- Gives rich planning a path without hiding repository checkout or computation
  inside the main server process.
- Keeps canonical WorkItem truth API-server-owned.
- Prevents workers and runners from becoming shadow databases or hidden sources
  of truth.

### Negative

- Worker/controller/runner-side planning still needs an additional boundary.
- Assignment / claiming should wait for this planning request / proposal model.
- The direct public task creation endpoint is removed; proposal acceptance is
  now the only public path that materializes canonical WorkItems.

## Recommended next slice

The next backend planning slice should design the smallest worker/controller
lease or reconciliation protocol around the existing public plan/proposal API.
The public `plans` / `proposals` / `acceptance` routes exist; the
worker/controller/runner execution-side implementation remains deferred.

It should not add runner checkout, execution, receipt submission,
`GateDecision`, `Proof`, queue, outbox, broad worker platform, or runtime
registry behavior.
