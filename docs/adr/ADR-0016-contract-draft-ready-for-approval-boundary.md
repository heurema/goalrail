# ADR-0016 — ContractDraft ready_for_approval boundary

Status: accepted
Date: 2026-04-26

## Context

Goalrail now has a bounded server-owned flow through draft creation and review:

- `IntakeRecord`
- `Goal`
- Goal readiness check
- `ClarificationRequest`
- `ClarificationAnswer`
- answer application to Goal hints
- explicit readiness re-check
- `ContractSeed(created)`
- `ContractDraft(draft)`
- `ContractDraft(draft)` review/update through proposed-field updates

`ContractDraft` can be created from `ContractSeed` and updated while its state
remains `draft`. Those updates affect proposed terms only. They do not approve
anything, create executable work, or write gate/proof state.

A reviewed draft may eventually become complete enough to submit for approval.
The next boundary is marking a draft as `ready_for_approval` through an explicit
server-owned transition.

This boundary remains before approved Contract, approval policy, `WorkItem`,
execution, `GateDecision`, and `Proof`.

ADR-0010 defines the Organization / Project / RepoBinding / Postgres persistence
foundation. The implementation stores `ContractDraft` in memory or in
Postgres-backed mode using the existing `contract_drafts` table. No new table is
introduced for this boundary.

## Decision

The server may transition `ContractDraft` from `draft` to
`ready_for_approval`.

The transition is explicit, server-owned, and evented. CLI, skills, web
resources, and integrations may request or display the transition, but they do
not own canonical `ready_for_approval` truth.

The request must include a `marked_by` actor reference. The draft must exist,
must currently be `draft`, and must pass minimum completeness checks before the
state changes.

The transition requires minimum draft completeness checks before the state can
change.

Marking a draft `ready_for_approval` does not:

- approve `ContractDraft`
- create an approved Contract
- create `WorkItem` or task
- make work executable
- start execution
- create or write `GateDecision`
- create `Proof`

Contract approval remains a later explicit boundary. WorkItem creation remains
later and only after an approved Contract boundary.

## Required completeness checks

Minimum checks before `ready_for_approval`:

- `title` is present.
- `intent_summary` is present.
- `proposed_scope` is non-empty.
- `proposed_acceptance_criteria` is non-empty.
- `proposed_proof_expectations` is non-empty.
- `repo_binding_id` is present.
- `contract_seed_id` is present.
- `goal_id` is present.

Recommended optional checks for v0:

- `proposed_non_goals` may be empty.
- `proposed_constraints` may be empty.
- `proposed_expected_checks` may be empty unless known checks already exist.
- `risk_hints` may be empty.

These checks are draft readiness checks, not approval checks. They only confirm
that enough proposed material exists to submit the draft to a later approval
boundary.

They do not:

- validate technical correctness
- validate delivery feasibility
- validate proof sufficiency
- grant approval authority
- make work executable

## Proposed state transition

`ContractDraft` states in scope for this boundary:

- `draft`
- `ready_for_approval`

Transition in scope:

```text
ContractDraft(draft) -> ContractDraft(ready_for_approval)
```

Do not introduce these states in this boundary:

- `approved`
- `rejected`
- `running`
- `done`
- `proof_pending`

If future edits are needed after a draft is marked `ready_for_approval`, a later
boundary may define revert-to-draft, supersede, cancel, or update-after-ready
behavior. This ADR does not define that transition.

## Proposed events

Recommended event:

- `contract_draft.marked_ready_for_approval`

Event payload should include:

- `contract_draft_id`
- `contract_seed_id`
- `goal_id`
- `marked_by`
- `previous_state=draft`
- `new_state=ready_for_approval`
- `marked_at`

Do not introduce these events in this boundary:

- `contract.approved`
- `work_item.created`
- `run.started`
- `gate.decision_written`
- `proof.created`

## Actor semantics

The transition must record who marked the draft ready with an explicit
`marked_by` actor-shaped value.

Required v0 fields:

- `marked_by.kind`
- `marked_by.id`

`marked_by.display_name` may be preserved when supplied.

`marked_by` is audit identity only. It does not mean approver, approval
authority, authorization policy, role membership, or final contract signoff.
Authorization and approval policy remain later boundaries.

## Relationship to ContractDraft update

`ContractDraft` update remains a separate boundary.

The update boundary keeps `ContractDraft.state = draft` and changes only allowed
proposed fields.

The `ready_for_approval` transition should not change proposed fields. If
proposed fields need changes, those changes should happen through the update
boundary before marking the draft ready.

Marking ready should preserve identity and source fields such as `id`,
`contract_seed_id`, `goal_id`, `repo_binding_id`, `source_refs`, and
`created_at`.

## Relationship to approval

Approval is a later boundary.

`ready_for_approval` means the draft is complete enough to be submitted for
approval. It does not mean the proposed scope, non-goals, constraints,
acceptance criteria, expected checks, proof expectations, or risks are approved.

`ready_for_approval` does not:

- create approved Contract
- grant execution permission
- create `WorkItem`
- create or write `GateDecision`
- create `Proof`

A later approval boundary may decide how a ready draft becomes approved,
rejected, returned for changes, or superseded.

## Relationship to persistence / ADR-0010

This ADR defines a domain boundary, not a storage backend.

Current implementation may use in-memory draft storage or existing
Postgres-backed draft storage. No new table, column, migration, outbox, queue,
or event-bus requirement is introduced by this ADR.

Persistence semantics remain governed by ADR-0010 and later storage decisions.
This ADR does not expand Organization / Project / RepoBinding persistence scope.

## Rejected alternatives

### Updating draft automatically marks ready_for_approval

Rejected. Review/update and readiness-for-approval are separate auditable
transitions. Updating proposed fields must not implicitly change draft state.

### ready_for_approval automatically approves Contract

Rejected. Completeness for approval review is not approval. Approval requires a
later explicit boundary with its own authority, policy, and event semantics.

### ready_for_approval creates WorkItem

Rejected. Work items are executable delivery objects and must come only after an
approved Contract boundary.

### ready_for_approval starts execution

Rejected. Execution belongs after approved Contract and task shaping. A ready
draft is not runnable work.

### ready_for_approval writes GateDecision or Proof

Rejected. Gate decision and proof belong to later verification boundaries after
approved contract, execution, and evidence collection.

### CLI, skill, web, or integration marks ready locally as canonical truth

Rejected. External surfaces may transport a request or display state, but the
server owns canonical `ContractDraft` state and events.

### LLM completeness check alone marks draft ready

Rejected. LLM review, if introduced later, may be advisory. It must not silently
become canonical readiness, approval, or contract authority.

### ready_for_approval and approval are implemented in the same boundary

Rejected. Readiness for approval and approval are separate decisions. Combining
them would erase an auditable review handoff.

### ready_for_approval mutates proposed fields

Rejected. Marking ready validates the current draft and changes state only. Any
field edits must go through the draft update boundary.

## Non-goals

This ADR does not define or implement:

- approved Contract
- contract approval
- approval policy
- approver identity or authorization model
- task planning
- `WorkItem`
- execution
- receipt submission
- verification bundle
- `GateDecision`
- `Proof`
- LLM review or rewrite
- CLI integration
- web UI

## Implementation implications

The implementation slice uses:

- endpoint: `POST /v1/contract-drafts/{id}/approval-submissions`
- service method to validate completeness
- `draft -> ready_for_approval` state transition
- required `marked_by` actor
- event: `contract_draft.marked_ready_for_approval`
- the existing `contract_drafts` table with its state check allowing `draft` and
  `ready_for_approval`

The immediate implementation slice must not:

- approve `ContractDraft`
- create approved Contract
- create `WorkItem`
- start execution
- write `GateDecision` or `Proof`
- modify ADR-0010 persistence scope

## Open questions

- Should `ready_for_approval` require `proposed_non_goals`?
- Should `ready_for_approval` require `proposed_expected_checks`?
- Should `ready_for_approval` require `risk_hints`?
- Should readiness checks be stored as event payload only or a separate
  projection?

Recommended initial direction:

- keep `proposed_non_goals` optional in v0.
- keep `proposed_expected_checks` optional in v0.
- keep `risk_hints` optional in v0.
- store readiness checks in the event payload.
