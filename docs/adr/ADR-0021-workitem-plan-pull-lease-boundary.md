# ADR-0021 — WorkItemPlan pull lease boundary

Status: accepted
Date: 2026-05-03

## Context

ADR-0019 defines the WorkItem planning controller / runner split. The current
server has the public `plans` / `proposals` / `acceptance` control-plane API,
but it does not have workers, controllers, runners, a planning lease protocol,
checkout, execution, assignment, claiming, `Run`, receipt, `GateDecision`, or
`Proof`.

The next planning boundary needs to let planner workers discover available
planning work without writing to Postgres directly and without turning Goalrail
into a generic queue platform. Planning work is already represented by typed
domain state:

```text
ApprovedContract(approved)
  -> WorkItemPlan(queued)
  -> WorkItemPlanProposal(submitted)
  -> WorkItem(planned) after explicit acceptance
```

This ADR documents the intended typed Postgres-backed pull lease queue before
implementation. It is a design decision only.

## Decision

Goalrail will use typed domain state as the planning queue.

Accepted future model:

```text
work_item_plans = canonical typed planning queue
work_item_plan_leases = typed lease/reservation records
```

`WorkItemPlan(state=queued)` is the planning queue item. `WorkItemPlanLease` is
the reservation / lease record for one worker attempt to compute and submit a
planning proposal.

Goalrail will not introduce a generic `queue_jobs`, `jobs`,
`generic_queue_items`, or `work_queue` table for this boundary.

The API server owns scheduling, lease state, validation, persistence, and event
append. Workers poll the API server for the next available planning job. Workers
must not read or write Postgres directly.

The API server atomically grants a lease for the next eligible `WorkItemPlan`.
The worker computes a proposal outside the API server and submits that proposal
back through the API. The API server persists the proposal and later
materializes an accepted proposal into canonical `WorkItem(planned)` records.

This remains planning only. It is not execution.

## Kubernetes analogy

This boundary follows a careful control-plane analogy with Kubernetes:

| Kubernetes concept | Goalrail planning concept |
| --- | --- |
| Pending Pod | `WorkItemPlan(state=queued)` |
| Scheduler / binding | API-server lease selection |
| Kubelet sees assigned work | Worker receives a plan lease |
| Kubelet reports status | Worker submits a `WorkItemPlanProposal` |
| API server stores canonical state | Goalrail API server stores canonical planning state |

This is an architectural analogy, not exact equivalence. Goalrail planning
leases are not Kubernetes Pods, bindings, kubelets, or status subresources.

## Why typed queue, not generic queue

A typed `WorkItemPlan` queue is preferred because planning state is domain state,
not a generic background job payload.

Reasons:

- avoid premature abstraction
- keep domain constraints as columns and foreign keys instead of JSON payload
- avoid a second generic state machine beside `WorkItemPlan`
- avoid scope creep into retries, dead-letter queues, priorities, job kinds,
  worker registry, and broad queue platform behavior
- stay aligned with API-server-owned typed resources
- keep planning state inspectable and product-facing

Rejected for now:

```text
queue_jobs / jobs / generic_queue_items with kind + payload JSONB
```

## Future REST shape

At ADR acceptance time, these routes were future target API shape and were not
implemented in the server.

| Route | Meaning |
| --- | --- |
| `POST /v1/plans/leases` | Creates a lease on the next eligible plan. This is REST-ish because it creates a `PlanLease` resource. It must not be `GET` because leasing mutates state. |
| `GET /v1/plans/leases/{id}` | Reads one lease. It must not return the raw lease token. |
| `PATCH /v1/plans/leases/{id}` | Renews the lease TTL if the lease is active and the token is valid. |
| `POST /v1/plans/{id}/proposals` | Existing proposal route; once the lease protocol is implemented, proposal submission should require `lease_id` and `lease_token`. |

Future lease request body:

```json
{
  "leased_by": {"kind": "worker", "id": "planner-worker-1"},
  "ttl_seconds": 900
}
```

Future response when work exists:

```json
{
  "id": "lease_1",
  "plan_id": "plan_1",
  "contract_id": "contract_1",
  "approved_contract_id": "approved_contract_1",
  "repo_binding_id": "repo_binding_1",
  "state": "active",
  "lease_token": "opaque-token-returned-once",
  "expires_at": "2026-05-02T18:30:00Z",
  "created_at": "2026-05-02T18:15:00Z"
}
```

Future response when no work exists:

```text
204 No Content
```

`lease_token` is returned only once, on successful lease creation. Subsequent
lease reads return lease metadata but not the raw token.

## Future planning flow

```text
POST /v1/contracts/{id}/plans
  -> WorkItemPlan(state=queued)

worker polls:
POST /v1/plans/leases
  -> API server selects next eligible queued or expired-leased plan
  -> creates WorkItemPlanLease(active)
  -> marks WorkItemPlan as leased
  -> returns lease + plan identifiers + raw lease token once

worker computes proposal outside API server

worker submits:
POST /v1/plans/{id}/proposals
  -> requires valid active lease_id + lease_token
  -> stores WorkItemPlanProposal(submitted)
  -> marks lease completed
  -> marks WorkItemPlan proposal_submitted

user/client accepts:
POST /v1/proposals/{id}/acceptance
  -> materializes WorkItem(planned)
```

Workers do not create canonical `WorkItem` records directly.

## Scheduling model v0

The first scheduler should be the smallest possible API-server-owned selection
rule:

- FIFO
- eligible plans are `state = queued` or `state = leased` with an expired lease
- order by `created_at ASC`, then `id ASC`
- one lease per poll request
- no priority
- no fairness
- no worker capability database
- no worker registry
- no repo affinity beyond optional future filters
- no standalone scheduler service

## Postgres implementation direction

Future implementation should use a transaction. Conceptually:

1. Select the candidate plan with row locking, such as `FOR UPDATE SKIP LOCKED`.
2. Mark an existing active lease expired when the selected plan is
   `leased` only because its previous lease expired.
3. Create a new `WorkItemPlanLease(active)` row.
4. Mark the plan `leased`.
5. Return the raw lease token only in the creation response.

The database should store only a lease token hash, not the raw token.

Lazy expiry is enough for v0. No cron is required. A plan with an expired lease
can be leased again by a later poll request.

## Lease object concept

Conceptual `WorkItemPlanLease` fields:

- `id`
- `plan_id`
- `contract_id`
- `approved_contract_id`
- `repo_binding_id`
- `leased_by`
- `state`
- `lease_token_hash`
- `expires_at`
- `created_at`
- `updated_at`

Recommended v0 states:

- `active`
- `completed`
- `expired`

Do not add `released` in the first implementation unless a release endpoint is
explicitly justified. No release endpoint is part of the first lease boundary.

## Plan state update

Current plan states include:

- `queued`
- `proposal_submitted`
- `accepted`

Future lease implementation should add:

- `leased`

Future state flow:

```text
queued -> leased -> proposal_submitted -> accepted
leased -> leased again if previous lease expired and a new worker polls
```

## TTL and renewal

TTL direction:

- default TTL: 15 minutes
- minimum TTL: 30 seconds
- maximum TTL: 60 minutes

Implementation may clamp or reject out-of-range TTL. The implementation PR must
document the exact behavior it chooses.

Future renewal request:

```json
{
  "lease_token": "opaque-token",
  "ttl_seconds": 900
}
```

Renewal rules:

- only an active lease can be renewed
- the token must match the stored token hash
- expired or completed leases return conflict
- renewal sets `expires_at = now + ttl`
- no separate heartbeat endpoint is needed in the first version; renewal is
  enough

## Proposal submission with lease proof

Once this lease protocol is implemented, future
`POST /v1/plans/{id}/proposals` should require lease proof:

```json
{
  "lease_id": "lease_1",
  "lease_token": "opaque-token",
  "submitted_by": {"kind": "worker", "id": "planner-worker-1"},
  "planner": {"kind": "goalrail_worker", "id": "planner-worker-1", "version": "0.1.0"},
  "proposed_tasks": []
}
```

Rules:

- lease must exist
- lease must be active
- lease must not be expired
- lease `plan_id` must match the URL plan id
- token must match stored token hash
- successful proposal submission stores the proposal
- successful proposal submission marks the plan `proposal_submitted`
- successful proposal submission marks the lease `completed`

Workers must not write `WorkItem` records directly.

## Current vs future

Implementation at ADR acceptance:

- has public `Plan` / `Proposal` / `Acceptance`
- has `POST /v1/contracts/{id}/plans`
- has `GET /v1/plans/{id}`
- has `POST /v1/plans/{id}/proposals`
- has `GET /v1/proposals/{id}`
- has `POST /v1/proposals/{id}/acceptance`
- may accept manual proposal submission without lease proof
- does not have leases
- does not have `POST /v1/plans/leases`
- does not have `GET /v1/plans/leases/{id}`
- does not have `PATCH /v1/plans/leases/{id}`

Implementation direction:

- should add typed `WorkItemPlanLease` persistence
- should add future lease routes under `/v1/plans/leases`
- should require lease proof for worker proposal submission
- should keep scheduling and lease state API-server-owned
- should keep workers out of direct Postgres access

## Non-goals

This ADR does not implement or authorize:

- route implementation
- Go code changes
- migrations
- new stores
- generic queue tables
- queue, outbox, broker, or runtime registry behavior
- worker, controller, or runner implementation
- checkout
- execution
- assignment or claiming
- `Run`
- receipt submission
- `GateDecision`
- `Proof`

## Consequences

### Positive

- Keeps planning queue semantics inside typed product state.
- Gives workers a future pull model without direct database access.
- Preserves the API server as the canonical scheduler and state owner.
- Avoids generic queue platform scope while Goalrail is still proving the
  planning boundary.
- Keeps planning work inspectable through domain resources.

### Negative

- Worker/controller/runner-side planning still needs a later implementation.
- A later queue abstraction may be needed if unrelated job families appear.
- FIFO v0 is intentionally simple and may need capability or policy filters
  later.

## Deferred implementation

The next backend implementation slice may add the smallest typed
`WorkItemPlanLease` persistence and route surface around existing
`WorkItemPlan` state.

That implementation slice should not add checkout, execution, assignment,
claiming, generic queue jobs, broad worker registry, runtime registry, `Run`,
receipt, `GateDecision`, or `Proof`.

## Implementation note

As of 2026-05-06, the server implements the narrow typed lease API and
persistence slice described by this ADR:

- `POST /v1/plans/leases`
- `GET /v1/plans/leases/{id}`
- `PATCH /v1/plans/leases/{id}`
- `POST /v1/plans/{id}/proposals` now requires `lease_id` and `lease_token`

Raw lease tokens are returned only on lease creation and stored only as hashes.
No worker, controller, runner, checkout, execution, assignment, claiming,
generic queue, outbox, runtime registry, `Run`, receipt, `GateDecision`, or
`Proof` implementation is added by this server slice.
