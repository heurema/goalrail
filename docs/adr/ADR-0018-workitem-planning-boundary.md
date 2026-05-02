# ADR-0018 — WorkItem planning boundary

Status: accepted
Date: 2026-04-27

## Context

Goalrail now has a bounded server-owned path to approved contract truth:

- `ContractSeed(created)` can be created from a ready Goal.
- `ContractDraft(draft)` can be created from that seed.
- `ContractDraft(draft)` can be reviewed and updated through proposed fields.
- `ContractDraft(draft)` can be marked `ready_for_approval`.
- `ContractDraft(ready_for_approval)` can be approved into
  `ApprovedContract(approved)`.

`ApprovedContract` is the approved agreement snapshot. Approval does not create
WorkItems today. It does not start execution, create a run, submit receipts,
write `GateDecision`, or create `Proof`.

The next boundary is explicit WorkItem planning:

```text
ApprovedContract(approved) -> WorkItem(planned)
```

This boundary remains before assignment, claiming, execution, runner checkout,
`Run`, receipt submission, `GateDecision`, and `Proof`.

ADR-0010 defines the Organization / Project / RepoBinding / Postgres persistence
foundation. This ADR defines a domain boundary, not a storage backend or
migration requirement.

## Decision

The server may create planned `WorkItem` records from
`ApprovedContract(approved)`.

WorkItems are canonical server-owned planning units. CLI, skills, web resources,
and integrations may request or display WorkItem state, but they do not own
canonical WorkItem truth.

WorkItems are derived from approved scope, approved acceptance criteria, and
approved proof expectations.

WorkItem planning is explicit and evented.

WorkItem planning does not:

- start execution
- create `Run`
- checkout a repository
- assign work
- claim work
- submit a receipt
- write `GateDecision`
- create `Proof`

Runtime task packets, assignment / claiming, runner checkout, execution, receipt
submission, gate decisions, and proof are later explicit boundaries.

## Proposed object model

Conceptual `WorkItem` fields:

- `id`
- `approved_contract_id`
- `repo_binding_id`
- `title`
- `summary`
- `scope`
- `acceptance_refs`
- `proof_expectation_refs`
- `status`
- `owner_hint` optional
- `order_index` optional
- `source_refs`
- `created_at`

Recommended v0 status:

- `planned`

Field semantics:

- `scope` is WorkItem scope derived from approved contract scope.
- `acceptance_refs` point to approved acceptance criteria on the
  `ApprovedContract` snapshot.
- `proof_expectation_refs` point to approved proof expectations on the
  `ApprovedContract` snapshot.
- `owner_hint` is advisory metadata only; it is not assignment.
- `order_index` is planning order only; it is not execution order.

`WorkItem` is not `Run`. It is not execution, checkout, receipt,
`GateDecision`, or `Proof`.

## Planning rules

Minimum rules:

- `ApprovedContract` must exist.
- `ApprovedContract.state` must be `approved`.
- `ApprovedContract.scope` must be non-empty.
- `ApprovedContract.acceptance_criteria` must be non-empty.
- `ApprovedContract.proof_expectations` must be non-empty.
- WorkItem planning must not start execution.
- WorkItem planning must not create `Run`.
- WorkItem planning must not create a receipt.
- WorkItem planning must not write `GateDecision` or `Proof`.

Recommended v0 behavior:

- create one planned `WorkItem` for the `ApprovedContract`
- `WorkItem.title` may come from `ApprovedContract.title`
- `WorkItem.summary` may come from `ApprovedContract.intent_summary`
- `WorkItem.scope` may include all approved scope entries
- `acceptance_refs` may reference all approved acceptance criteria
- `proof_expectation_refs` may reference all approved proof expectations
- more detailed decomposition is a future boundary
- repeated planning returns `409 already_planned`
- `WorkItem.status = planned`

## Proposed events

Recommended event:

- `work_item.created`

Event payload should include:

- `work_item_id`
- `approved_contract_id`
- `repo_binding_id`
- `status=planned`
- `source_refs`
- `created_at`

Optional future WorkItem events:

- `work_item.split`
- `work_item.assigned`
- `work_item.claimed`
- `work_item.cancelled`
- `work_item.superseded`

Do not introduce these events in this boundary:

- `run.started`
- `receipt.submitted`
- `gate.decision_written`
- `proof.created`

## Relationship to ApprovedContract

WorkItems read the `ApprovedContract(approved)` snapshot.

`ApprovedContract` is not mutated by WorkItem planning. The approved agreement
remains a historical approved snapshot.

WorkItems should preserve `source_refs` to `ApprovedContract`, `ContractDraft`,
`ContractSeed`, and Goal where available.

If an `ApprovedContract` is superseded later, WorkItem supersession or
cancellation must be explicit in a future boundary.

## Relationship to assignment / claiming

`WorkItem.status = planned` is not assignment.

`owner_hint` is advisory only. It may help a UI or operator understand likely
ownership, but it does not reserve, claim, or assign the WorkItem.

Assignment and claiming are later explicit boundaries.

A planned WorkItem may be visible on a board, but visibility does not mean
execution started.

## Relationship to execution / Run

`WorkItem` is not `Run`.

WorkItem planning does not checkout a repository, start agent or human
execution, or produce a receipt.

Runtime task packets, runner checkout, `Run`, and receipt submission are later
delivery boundaries.

## Relationship to Gate / Proof

`GateDecision` and `Proof` are post-execution verification boundaries.

WorkItem planning references approved proof expectations, but it does not
satisfy them.

WorkItem planning does not create `Proof` and does not write `GateDecision`.

## Relationship to ADR-0010 persistence

This ADR defines a domain boundary, not a storage backend.

Current implementation may use in-memory stores or existing persistence,
depending on a later implementation slice. This ADR introduces no new migration,
table, queue, outbox, or event bus requirement.

Storage backend remains governed by ADR-0010 and later persistence decisions.
This ADR does not expand Organization / Project / RepoBinding persistence scope.

## Rejected alternatives

### Approval automatically creates WorkItems

Rejected. Approval accepts contract terms. WorkItem planning is a separate
explicit boundary.

### WorkItem planning starts execution

Rejected. Planning produces non-executable WorkItems only. Execution requires
later runtime and runner boundaries.

### WorkItem is treated as Run

Rejected. `WorkItem` is planning state. `Run` is an execution attempt and belongs
to a later delivery boundary.

### WorkItem is treated as receipt

Rejected. Receipts are evidence from execution. WorkItem planning happens before
execution.

### WorkItem writes Proof

Rejected. Proof is a post-execution evidence and decision artifact.

### WorkItem writes GateDecision

Rejected. Gate decisions evaluate completed delivery evidence after execution.

### CLI, skill, web, or integration creates WorkItems locally as canonical truth

Rejected. External surfaces may request planning or display WorkItems, but the
server owns canonical WorkItem truth.

### Planning decomposes complex tasks with LLM in v0

Rejected. The first boundary should create one planned WorkItem per approved
contract. LLM decomposition and multi-item planning can be a future boundary.

### WorkItem creation and runner checkout happen in the same boundary

Rejected. Checkout belongs behind runner boundaries and must not be hidden inside
planning.

### WorkItem assignment / claiming happens in the same boundary

Rejected. Assignment and claiming change operational ownership and should be
explicit later transitions.

### WorkItem creation mutates ApprovedContract

Rejected. `ApprovedContract` is the approved agreement snapshot. WorkItem
planning should create derived planning records without mutating it.

## Non-goals

This ADR does not define or implement:

- code implementation
- endpoint finalization as public API canon
- multi-item decomposition beyond the simple v0 direction
- assignment
- claiming
- runner checkout
- execution
- `Run`
- receipt submission
- verification bundle
- `GateDecision`
- `Proof`
- durable storage changes
- migrations
- LLM decomposition
- CLI integration
- web UI

## Implementation implications

A later implementation slice may add:

- `WorkItem` DTO
- in-memory `WorkItemStore`
- planning service
- endpoint candidate: `POST /v1/contracts/{id}/tasks`
- one-WorkItem planning v0
- duplicate guard by `approved_contract_id`
- event: `work_item.created`

Endpoint choice is not final product API canon.

The immediate implementation slice must not:

- start execution
- create `Run`
- assign or claim `WorkItem`
- submit receipt
- write `GateDecision` or `Proof`
- modify ADR-0010 persistence scope

## Open questions

- Should v0 create one `WorkItem` or split by approved scope entries?
- Should `owner_hint` come from contract metadata or remain empty?
- Should `WorkItem` be assignable immediately or only after a claim boundary?
- Should repeated planning return `409 already_planned` or existing WorkItems?
- Should WorkItem persistence be added immediately, or remain prototype depending
  on current storage state?
- Should WorkItem statuses include only `planned` in v0, or also `cancelled` and
  `superseded`?

Recommended initial direction:

- create one planned `WorkItem` per `ApprovedContract`
- keep `owner_hint` optional / empty in v0
- use `status=planned` only in v0
- keep assignment and claiming as later boundaries
- repeated planning returns `409 already_planned`
- let persistence depend on the later implementation slice, with no new
  migration requirement in this ADR
