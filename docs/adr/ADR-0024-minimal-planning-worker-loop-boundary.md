# ADR-0024 — Minimal planning worker loop boundary

Status: accepted
Date: 2026-05-07

## Context

ADR-0019 separates WorkItem planning computation from API-server-owned
canonical state. ADR-0021 then selects typed `WorkItemPlan` pull leases instead
of a generic queue platform.

The current server now implements the typed lease API:

- `POST /v1/plans/leases`
- `GET /v1/plans/leases/{id}`
- `PATCH /v1/plans/leases/{id}`

Proposal submission through `POST /v1/plans/{id}/proposals` requires lease
proof with `lease_id` and `lease_token`.

There is still no worker, controller, runner, checkout, execution, assignment,
claiming, `Run`, receipt, `GateDecision`, or `Proof` implementation. The next
boundary is therefore the smallest process loop that can consume the typed
planning lease API without expanding into a worker platform or runner.

## Decision

The first worker should be a separate thin planning worker binary, likely named
`goalrail-worker`.

The worker is not a runner. It does not checkout repositories, execute code,
start runs, write receipts, decide gates, or create proof. It does not write to
Postgres and does not create canonical `WorkItem` records directly.

The API server remains the canonical state machine. The worker only talks to the
API server.

The first worker loop should process one planning lease at a time:

```text
loop:
  POST /v1/plans/leases

  if 204:
    sleep/backoff
    continue

  GET /v1/plans/{lease.plan_id}
  optionally GET /v1/plans/leases/{lease.id}

  compute proposal outside API server

  POST /v1/plans/{plan_id}/proposals
    with lease_id + lease_token

  repeat
```

The first worker is a narrow planning worker, not a broad worker platform.

## Terminology

- `worker` means the process that polls the API server and coordinates planning
  work.
- `planner` means the capability inside the worker that produces proposed
  WorkItems.
- `runner` means a later execution-side component for repository checkout,
  command execution, receipts, and related runtime work.
- `proposal` means planner output submitted to the API server.
- `WorkItem` means the canonical server-owned planned task materialized only
  after proposal acceptance.

## Future binary direction

The future binary should use minimal startup configuration:

- API server URL
- worker identity
- poll interval
- lease TTL
- optional token/auth placeholder

It should not require:

- repository credentials
- checkout credentials
- runtime credentials
- execution adapters
- worker registry setup
- queue/outbox setup

## Worker responsibilities

A future planning worker may:

- poll `POST /v1/plans/leases`
- renew a lease while planning
- read leased plan details through the API
- compute or collect one proposal
- submit the proposal with lease proof
- log local diagnostics
- use bounded planning logic later

A future planning worker must not:

- access Postgres directly
- create WorkItems directly
- accept proposals
- assign or claim WorkItems
- checkout repositories
- run commands
- start Runs
- submit receipts
- write `GateDecision`
- create `Proof`

## Planner behavior v0

The first planner behavior may initially be deterministic or manual proposal
mode. This is a target boundary, not an implemented behavior in this ADR.

The planner may use approved Contract and Plan metadata fetched from the API.
It should not require repository checkout in the first worker slice. Repo-aware
planning remains a later planner / runner context boundary.

If a dummy or development planner is introduced later, it must be clearly
marked as development-only and not production planning behavior.

## Lease behavior

The worker gets one lease at a time and processes one plan at a time.

If planning takes longer than the lease TTL, the worker renews the lease through
`PATCH /v1/plans/leases/{id}` with the lease token. If the worker crashes, the
lease expires and the API server can re-lease the plan.

The worker must submit proposals with both `lease_id` and `lease_token`. Raw
lease tokens are secrets. They must not be logged, included in diagnostic
output, or persisted casually in local files.

## Error and retry posture v0

Simple v0 behavior:

- `204 No Content`: sleep/backoff and poll later.
- `409 lease_expired`: abandon local work and poll again.
- `409 invalid_lease`: do not retry the same proposal blindly.
- Network failure before proposal submission: the worker may retry lease read or
  renewal if the token is still valid.
- Network failure after proposal submission: the worker should check plan or
  proposal state before retrying later; exact idempotency is a future hardening
  slice.
- No durable local worker queue is required in v0.

## Kubernetes analogy

The API server owns typed state. The worker polls and reconciles
server-owned desired state. A lease is similar to a binding or reservation.

This is not Kubernetes exactly. This decision does not introduce a scheduler,
node registry, controller manager, kubelet-like runtime, generic job API, or
runtime platform.

## Relationship to ADR-0021

ADR-0021 defines the typed `WorkItemPlan` pull lease API and keeps lease
selection API-server-owned. This ADR defines the minimal external planning
worker loop that may consume that API.

The worker loop must preserve ADR-0021's constraints:

- typed `WorkItemPlan` leases, not generic queue jobs
- one lease per poll request
- API-server-owned lease validation and state transitions
- proposal submission through the API with lease proof
- no direct database writes by workers

## Relationship to ADR-0008 and runners

ADR-0008 remains the repository checkout and runner boundary.

This planning worker is not the runner from ADR-0008. If future planning needs
repository checkout, repository inspection, baseline snapshots, or command
execution, that requires a later explicit runner / planner context boundary.
The first planning worker must not smuggle checkout or execution into the
planning loop.

## Non-goals

This ADR does not implement or authorize:

- worker code
- worker binary packaging
- route implementation
- migrations
- new stores
- direct database access from workers
- checkout
- execution
- assignment or claiming
- queue, outbox, broker, worker registry, or runtime registry behavior
- `Run`
- receipt submission
- `GateDecision`
- `Proof`
- repo-aware planning computation
- durable local worker queue
- idempotency hardening beyond the simple v0 retry posture

## Consequences

### Positive

- Gives the implemented typed lease API a minimal consuming process boundary.
- Keeps the API server as the canonical planning state machine.
- Avoids broad worker platform scope.
- Keeps planning separate from runner checkout and execution.
- Preserves explicit proposal acceptance before canonical WorkItem creation.

### Negative

- The first worker may only support deterministic or manual planning behavior.
- Idempotency and retry hardening remain later work.
- Repo-aware planning still needs a separate context / runner boundary.

## Recommended next slice

The next backend / worker implementation slice may add the smallest
`goalrail-worker` loop around the existing lease API:

- start a separate thin binary
- read API URL, worker identity, poll interval, lease TTL, and optional auth
  placeholder from config
- poll for one lease
- fetch one plan
- optionally renew while working
- compute or collect one proposal through a clearly bounded v0 planner mode
- submit that proposal with lease proof
- repeat

That implementation slice must not add checkout, execution, assignment,
claiming, queue, outbox, runtime registry, `Run`, receipt, `GateDecision`, or
`Proof`.
