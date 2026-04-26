# ADR-0017 — Contract approval boundary

Status: accepted
Date: 2026-04-26

## Context

Goalrail now has a bounded server-owned contract preparation flow:

- `ContractSeed(created)` can be created from a ready Goal.
- `ContractDraft(draft)` can be created from that seed.
- `ContractDraft(draft)` can be reviewed and updated through proposed fields.
- `ContractDraft(draft)` can be marked `ready_for_approval` after
  completeness checks.

`ready_for_approval` is not approval. The `marked_by` actor in that transition
is audit identity only, not approval authority.

The next boundary is explicit contract approval. Approval accepts the ready draft
terms and creates a canonical approved Contract snapshot: `ApprovedContract`.

This boundary remains before `WorkItem` creation, task planning, execution,
`GateDecision`, and `Proof`.

ADR-0010 defines the Organization / Project / RepoBinding / Postgres persistence
foundation. This ADR defines a domain boundary, not a storage backend or
migration requirement.

## Decision

The server may approve `ContractDraft(ready_for_approval)`.

Approval is explicit, server-owned, and evented. CLI, skills, web resources, and
integrations may request or display approval state, but they do not own canonical
approval truth.

Approval creates `ApprovedContract` as the canonical approved snapshot of the
current ready draft terms.

Approval requires an `approved_by` actor. Unlike `marked_by` in the
`ready_for_approval` boundary, `approved_by` represents the approval actor. It
is still not an authorization policy by itself; policy and authorization checks
may be defined by a later boundary.

Approval does not:

- create `WorkItem` or task
- plan work
- make work executable
- start execution
- create or write `GateDecision`
- create `Proof`

WorkItem planning remains a later explicit boundary and should consume
`ApprovedContract`, not `ContractDraft`.

## Proposed object model

Conceptual `ApprovedContract` fields:

- `id`
- `contract_draft_id`
- `contract_seed_id`
- `goal_id`
- `repo_binding_id`
- `title`
- `intent_summary`
- `scope`
- `non_goals`
- `constraints`
- `acceptance_criteria`
- `expected_checks`
- `proof_expectations`
- `risk_hints`
- `approved_by`
- `approved_at`
- `source_refs`
- `state`

Recommended v0 state:

- `approved`

Field semantics:

- `scope` is approved scope copied from `proposed_scope`.
- `non_goals` are approved non-goals copied from `proposed_non_goals`.
- `constraints` are approved constraints copied from `proposed_constraints`.
- `acceptance_criteria` are approved acceptance criteria copied from
  `proposed_acceptance_criteria`.
- `expected_checks` are approved expected checks copied from
  `proposed_expected_checks`.
- `proof_expectations` are approved proof expectations copied from
  `proposed_proof_expectations`.
- `risk_hints` remain risk hints on the approved snapshot unless a later policy
  boundary upgrades them into enforced risk policy.

`ApprovedContract` is not `WorkItem`. It is not a task plan, execution run,
`GateDecision`, or `Proof`.

## Approval rules

Minimum rules:

- `ContractDraft` must exist.
- `ContractDraft.state` must be `ready_for_approval`.
- `approved_by.kind` is required.
- `approved_by.id` is required.
- The draft must still have non-empty required proposed fields:
  - `proposed_scope`
  - `proposed_acceptance_criteria`
  - `proposed_proof_expectations`
- Approval snapshots current draft terms into a new `ApprovedContract`.
- Approval must not create `WorkItem`.
- Approval must not start execution.
- Approval must not write `GateDecision` or `Proof`.

Recommended v0 duplicate behavior:

- Do not create multiple active `ApprovedContract` snapshots for the same
  `ContractDraft`.
- Repeated approval returns `409 already_approved`.

## Proposed events

Recommended event:

- `contract.approved`

Event payload should include:

- `approved_contract_id`
- `contract_draft_id`
- `approved_by`
- `approved_at`
- `source_refs`
- optionally `previous_draft_state`

Do not introduce these events in this boundary:

- `work_item.created`
- `run.started`
- `gate.decision_written`
- `proof.created`

## Relationship to ContractDraft

`ApprovedContract` reads the current `ContractDraft(ready_for_approval)`
snapshot and copies approved terms from its proposed fields.

`ContractDraft` remains historical draft material. Recommended v0 behavior is to
not mutate `ContractDraft.state` during approval. This keeps the approved truth
as a separate snapshot and preserves draft history.

Approval may mark or link the draft only if a later boundary explicitly defines
that state transition. This ADR does not require draft mutation.

If draft terms change after approval, a later supersession boundary must define
how a new draft or contract replaces the old approved snapshot.

## Relationship to WorkItems

WorkItems are created later from `ApprovedContract`, not from `ContractDraft`.

Approval does not:

- decompose work
- plan tasks
- create `WorkItem`
- assign executors
- make anything runnable by itself

WorkItem planning is a separate explicit boundary after approval.

## Relationship to Gate / Proof

Contract approval is pre-execution agreement acceptance.

`GateDecision` is a post-execution delivery decision. `Proof` is a
post-execution evidence and decision artifact.

Approval must not be conflated with gate. Approval does not verify completed
work, accept delivery evidence, write `GateDecision`, or create `Proof`.

## Relationship to policy / auth

`approved_by` is required and represents the approval actor.

`approved_by` must not be inferred from `marked_by`. Marking a draft ready and
approving a contract are separate actions with different semantics.

Authorization and approval policy are not silently implied by this ADR. If v0
does not enforce approver authorization, it must state that policy checks are
deferred and still record `approved_by`.

Recommended v0 direction:

- record `approved_by`
- defer authorization policy to a future policy/auth boundary
- do not treat `marked_by` from `ready_for_approval` as approver

## Relationship to ADR-0010 persistence

This ADR defines a domain boundary, not a storage backend.

Storage backend remains governed by ADR-0010 and later persistence decisions.
Current implementation may remain in-memory or use existing persistence,
depending on a later bounded implementation slice.

This ADR does not require a new migration, table, queue, outbox, or event bus.
It does not expand Organization / Project / RepoBinding persistence scope.

## Rejected alternatives

### ready_for_approval automatically approves

Rejected. Readiness for approval is a handoff state, not approval authority.

### Updating draft automatically approves

Rejected. Draft update changes proposed fields only. Approval must be explicit
and evented.

### Approval creates WorkItems

Rejected. Approval accepts contract terms. Task planning and WorkItem creation
are separate boundaries after approval.

### Approval starts execution

Rejected. Execution requires later work planning and runner/execution
boundaries.

### Approval writes GateDecision or Proof

Rejected. Gate and proof are post-execution verification boundaries.

### CLI, skill, web, or integration approves locally as canonical truth

Rejected. External surfaces may request approval or display approval state, but
canonical approval truth is server-owned.

### ApprovedContract is just a state change on draft with no snapshot

Rejected for v0 direction. A separate snapshot preserves draft history and makes
the approved contract terms explicit.

### Approval and task planning in the same boundary

Rejected. Combining approval with WorkItem planning would make agreement and
execution planning indistinguishable.

### Approval and gate decision are the same thing

Rejected. Approval accepts proposed contract terms before execution. Gate decides
whether delivered work satisfies the contract after execution and evidence.

## Non-goals

This ADR does not define or implement:

- code implementation
- endpoint finalization as public API canon
- `WorkItem` creation
- task planning
- execution
- runner checkout
- receipt submission
- verification bundle
- `GateDecision`
- `Proof`
- durable storage changes
- migrations
- CLI integration
- web UI

## Implementation implications

A later implementation slice may add:

- `ApprovedContract` DTO
- in-memory `ApprovedContractStore`
- approval service
- endpoint candidate: `POST /v1/contract-drafts/{id}/approve`
- required `approved_by` actor
- event: `contract.approved`
- duplicate approval guard by `contract_draft_id`

Endpoint choice is not final product API canon.

The immediate implementation slice must not:

- create `WorkItem`
- plan tasks
- start execution
- write `GateDecision` or `Proof`
- modify ADR-0010 persistence scope unless a separate storage decision requires
  it

## Open questions

- Should approval mutate `ContractDraft.state` or only create `ApprovedContract`?
- Should v0 enforce approver authorization or only record `approved_by`?
- Should repeated approval return the existing `ApprovedContract` or
  `409 already_approved`?
- Should WorkItem planning be the immediate next boundary after approval?
- Should `ApprovedContract` be persisted immediately, or remain prototype
  depending on current storage state?

Recommended initial direction:

- create `ApprovedContract` as a separate snapshot
- do not mutate `ContractDraft` in v0
- repeated approval returns `409 already_approved`
- record `approved_by`
- defer authorization policy to a later boundary
- keep WorkItem planning as a separate next boundary
