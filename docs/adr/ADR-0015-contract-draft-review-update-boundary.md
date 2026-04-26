# ADR-0015 — ContractDraft review/update boundary

Status: accepted
Date: 2026-04-26

## Context

Goalrail now has a bounded server-owned flow from intake to draft creation:

- `IntakeRecord`
- `Goal`
- Goal readiness check
- `ClarificationRequest`
- `ClarificationAnswer`
- answer application to Goal hints
- explicit readiness re-check
- `ContractSeed(created)`
- `ContractDraft(draft)`

`ContractDraft` contains proposed terms only. Its fields may look like contract
terms, but they are not approved scope, final acceptance criteria, final checks,
proof obligations, runnable work, or gate decisions.

Draft terms may need human review or edits before any approval boundary. The
next boundary is explicit review/update of the draft's proposed fields while the
`ContractDraft` remains in `draft` state.

This boundary remains before approved Contract, approval policy, `WorkItem`,
execution, `GateDecision`, and `Proof`.

ADR-0010 defines the Organization / Project / RepoBinding / Postgres persistence
foundation. This ADR defines a domain boundary and does not change storage or
persistence requirements. The existing persisted Intake / Goal / EventLog
foundation does not make `ContractDraft` persistence required in this boundary.

## Decision

The server may update `ContractDraft` proposed fields through an explicit
review/update transition.

`ContractDraft` review/update is canonical server-owned state change. CLI,
skills, web resources, and integrations may request or display updates, but they
do not own canonical `ContractDraft` truth.

Updates write explicit audit events. In this boundary, the `ContractDraft` state
remains `draft`.

`ContractDraft` review/update does not:

- approve `ContractDraft`
- create an approved Contract
- create `WorkItem` or task
- make work executable
- start execution
- create or write `GateDecision`
- create `Proof`

Contract approval remains a later explicit boundary. WorkItem creation remains
later and only after an approved Contract boundary.

## Review/update actor

The review/update transition must record who requested the change with an
explicit `updated_by` actor-shaped value.

Recommended v0 semantics:

- require `updated_by.kind`.
- require `updated_by.id`.
- preserve `updated_by.display_name` when supplied.
- do not infer an actor from raw text.
- do not introduce an actor directory or actor resolution.
- do not introduce auth, roles, or approval policy in this boundary.

Until a later auth / policy boundary exists, the server records the update actor
for audit but does not treat `updated_by` as approval authority.

## Editable fields

This boundary allows updates only to proposed draft fields:

- `title`
- `intent_summary`
- `proposed_scope`
- `proposed_non_goals`
- `proposed_constraints`
- `proposed_acceptance_criteria`
- `proposed_expected_checks`
- `proposed_proof_expectations`
- `risk_hints`

These fields are not editable in this boundary:

- `id`
- `contract_seed_id`
- `goal_id`
- `repo_binding_id`
- `source_refs`
- `created_at`
- `state`, unless a later ADR defines a state transition

Field meanings and constraints after update:

- `proposed_scope` remains proposed scope, not approved scope.
- `proposed_non_goals` remain proposed non-goals, not final exclusions.
- `proposed_constraints` remain proposed constraints, not final policy.
- `proposed_acceptance_criteria` remain proposed acceptance criteria, not final
  acceptance criteria.
- `proposed_expected_checks` remain proposed checks, not a verification bundle.
- `proposed_proof_expectations` remain proposed proof expectations, not final
  proof obligations.
- `risk_hints` remain advisory until a later review or approval boundary.
- `intent_summary` remains draft context, not approved contract text.
- `ContractDraft` remains not approved and not executable.

## Proposed events

Recommended event:

- `contract_draft.updated`

Event payload should include:

- `contract_draft_id`
- `changed_fields`
- `updated_by`
- previous values where safe
- new values
- `updated_at`

Optional future events:

- `contract_draft.superseded`
- `contract_draft.cancelled`
- `contract_draft.marked_ready_for_approval`

Do not introduce these events in this boundary:

- `contract.approved`
- `work_item.created`
- `run.started`
- `gate.decision_written`
- `proof.created`

## Validation rules

Minimum validation rules:

- `ContractDraft` must exist.
- `ContractDraft.state` must be `draft`.
- update request must include at least one editable field.
- unknown fields must be rejected.
- non-editable fields must be rejected.
- `updated_by.kind` is required.
- `updated_by.id` is required.
- update must not mutate `source_refs`.
- update must not mutate identity fields.
- update must not approve or create work.

Recommended v0 behavior:

- use full-field replacement for allowed array fields.
- allow partial update over allowed fields.
- do not implement JSON Patch or patch operation languages.
- preserve empty slices as empty slices when an allowed array field is set to
  empty.
- allow empty `proposed_non_goals`, `proposed_constraints`,
  `proposed_expected_checks`, and `risk_hints`.
- allow empty `proposed_scope`, `proposed_acceptance_criteria`, or
  `proposed_proof_expectations` only if the implementation makes that choice
  explicit in validation tests; otherwise keep them non-empty until a later ADR
  relaxes the invariant.

## Relationship to approval

Approval is a later boundary.

A reviewed or updated draft is still not approved. This ADR does not introduce
approved Contract state, approval policy, approval events, or approval rules.

No `WorkItem` can be created from a `ContractDraft`. WorkItem creation remains
later and only after an approved Contract boundary.

Recommended direction:

- do not introduce `ready_for_approval` in this boundary.
- keep this ADR focused on update while `state=draft`.
- define `ready_for_approval` in a later explicit boundary if needed.

## Relationship to ContractSeed

Updating `ContractDraft` does not mutate `ContractSeed`.

`ContractSeed` remains a historical snapshot. `ContractDraft` keeps
`source_refs` back to the seed and Goal.

If a seed changes, regenerates, or becomes superseded later, any draft
supersession or replacement must be explicit in a future boundary. A draft must
not silently track a changed seed as a live view.

## Relationship to ADR-0010 persistence

This ADR defines a domain boundary, not a storage backend.

The current prototype may remain in-memory unless a later storage slice says
otherwise. Persisted Intake / Goal / EventLog foundation does not require
`ContractDraft` persistence in this ADR.

This ADR does not introduce:

- new Postgres tables
- migrations
- durable `ContractDraft` storage requirements
- durable event-log changes
- Organization / Project / RepoBinding changes
- `RepoBinding` persistence changes

## Rejected alternatives

### Updating draft automatically approves it

Rejected. Editing proposed terms is not approval. Approval requires a later
explicit boundary with its own rules and events.

### Updating draft automatically creates WorkItem

Rejected. Work items are executable delivery objects and must come only after an
approved Contract boundary.

### Updating draft writes GateDecision or Proof

Rejected. Gate decision and proof belong to later verification boundaries after
approved contract, execution, and evidence collection.

### CLI, skill, web, or integration updates draft as canonical local truth

Rejected. External surfaces may transport update requests or render draft state,
but the server owns canonical `ContractDraft` truth.

### LLM silently rewrites draft as canonical final contract

Rejected. LLM rewrite or enrichment, if introduced later, may propose changes.
It must not silently create final contract authority or approval.

### Draft update mutates ContractSeed

Rejected. `ContractSeed` is a historical snapshot. Draft review/update changes
only the draft.

### Draft update changes identity or source fields

Rejected. `id`, `contract_seed_id`, `goal_id`, `repo_binding_id`, `source_refs`,
`created_at`, and `state` are outside this boundary's editable field set.

### Draft update introduces ready_for_approval and approval in one boundary

Rejected. Readiness for approval and approval are separate decisions. This ADR
keeps update/review focused on proposed fields while state remains `draft`.

## Non-goals

This ADR does not define or implement:

- code implementation
- endpoint finalization as public API canon
- approved Contract
- approval policy
- `ready_for_approval` transition except as future-only discussion
- human approval workflow
- task planning
- `WorkItem`
- execution
- receipt submission
- verification bundle
- `GateDecision`
- `Proof`
- durable storage changes
- Postgres migration
- LLM drafting or rewrite
- policy engine
- actor directory or actor resolution
- auth, roles, or user model
- CLI integration
- web UI

## Implementation implications

A later bounded implementation slice may add:

- update service for `ContractDraft`
- endpoint candidate: `POST /v1/contract-drafts/{id}/updates`
- alternative endpoint candidate: `PATCH /v1/contract-drafts/{id}`
- full-field replacement for allowed proposed fields
- partial update over allowed fields
- required `updated_by` actor
- event: `contract_draft.updated`

Endpoint choice is not final public API canon.

Recommended initial direction:

- use `POST /v1/contract-drafts/{id}/updates` for transition clarity.
- allow partial field update over allowed fields.
- require `updated_by.kind` and `updated_by.id`.
- reject unknown and non-editable fields.
- keep `state=draft`.
- do not add revision numbers in v0 unless already easy and clearly bounded.

The immediate implementation slice must not:

- approve `ContractDraft`
- create approved Contract
- create `WorkItem`
- start execution
- write `GateDecision`
- create `Proof`
- modify ADR-0010 persistence code

## Open questions

Open decision questions for later slices:

1. Should v0 endpoint use `POST /v1/contract-drafts/{id}/updates` or
   `PATCH /v1/contract-drafts/{id}`?
2. Should draft update allow partial field updates or require full replacement
   payload?
3. Should empty `proposed_scope` / `proposed_acceptance_criteria` be allowed?
4. Should empty `proposed_proof_expectations` be allowed?
5. Should `ready_for_approval` be a separate boundary after update?
6. Should updates be versioned with revision numbers in v0?
7. Should `ContractDraft` persistence wait until a storage slice?

Recommended initial direction:

- use `POST /v1/contract-drafts/{id}/updates` for transition clarity
- allow partial field update over allowed fields
- require `updated_by`
- reject non-editable fields
- keep `state=draft`
- no revision numbers in v0 unless already easy
- `ready_for_approval` is a later boundary
- persistence remains separate
