# ADR-0014 — ContractDraft boundary

Status: accepted
Date: 2026-04-26

## Context

Goalrail now has a bounded server-owned flow from intake to a contract-plane
bridge:

- `IntakeRecord`
- `Goal`
- Goal readiness check
- `ClarificationRequest`
- `ClarificationAnswer`
- answer application to Goal hints
- explicit readiness re-check
- `ContractSeed(created)`

`ContractSeed` is a canonical snapshot from `Goal(ready_for_contract_seed)`.
It is still not a contract. It preserves readiness-checked intent material for a
later drafting step, but it does not contain approved contract authority,
executable work, final scope, final acceptance criteria, gate decision, or
proof.

The next boundary is turning seed material into a proposed `ContractDraft`.
This remains before approved Contract, approval policy, `WorkItem`, execution,
`GateDecision`, and `Proof`.

ADR-0010 defines the Organization / Project / RepoBinding / Postgres persistence
foundation. This ADR defines a domain boundary and does not change storage or
persistence requirements.

## Decision

The server may create a `ContractDraft` from `ContractSeed(created)`.

`ContractDraft` is canonical server-owned draft state. It contains proposed
contract terms derived from the seed snapshot. It must be reviewed and approved
by a later explicit boundary before any work becomes executable.

`ContractDraft` creation is explicit. `ContractSeed` creation must not
automatically create a draft.

`ContractDraft` creation does not:

- create an approved Contract
- approve anything
- create `WorkItem` or task
- make work executable
- start execution
- create `GateDecision`
- create `Proof`

`ContractDraft` creation writes explicit events. CLI, skills, web resources, and
integrations may request or display draft state, but they do not own canonical
`ContractDraft` truth.

Contract approval remains a later explicit boundary. WorkItem creation remains
later and only after an approved Contract boundary.

## Proposed object model

### ContractDraft

Minimal conceptual fields:

| Field | Intent |
| --- | --- |
| `id` | Server-owned stable draft ID. |
| `contract_seed_id` | ContractSeed used as draft input. |
| `goal_id` | Source Goal carried through from the seed. |
| `repo_binding_id` | Repository binding context carried through from the seed. |
| `title` | Draft title. |
| `intent_summary` | Intent summary carried from the seed; not contract text. |
| `proposed_scope` | Draft proposed scope. |
| `proposed_non_goals` | Draft proposed non-goals. |
| `proposed_constraints` | Draft proposed constraints. |
| `proposed_acceptance_criteria` | Draft proposed acceptance criteria. |
| `proposed_expected_checks` | Draft proposed checks expected before acceptance. |
| `proposed_proof_expectations` | Draft proposed proof expectations. |
| `risk_hints` | Advisory risk hints for later review and approval. |
| `source_refs` | References to ContractSeed, Goal, IntakeRecord when available, and related inputs. |
| `state` | Draft lifecycle state. |
| `created_at` | Server timestamp for draft creation. |

Recommended v0 state:

- `draft`

Field meanings and constraints:

- `proposed_scope` is draft scope, not approved scope.
- `proposed_non_goals` are draft non-goals, not approved exclusions.
- `proposed_constraints` are draft constraints, not final policy.
- `proposed_acceptance_criteria` are draft acceptance criteria, not final
  acceptance.
- `proposed_expected_checks` are draft check expectations, not a verification
  bundle.
- `proposed_proof_expectations` are draft proof expectations, not final proof
  requirements.
- `risk_hints` are advisory until review and approval.
- `intent_summary` is not final contract text.
- `ContractDraft` is not approved.
- `ContractDraft` is not executable.

## Creation rules

Minimum creation rules:

- `ContractSeed` must exist.
- `ContractSeed.state` must be `created`.
- `ContractSeed` must have `goal_id`.
- `ContractSeed` must have `repo_binding_id`.
- `ContractSeed` must have `title`.
- `ContractSeed` must have `intent_summary`.
- `ContractSeed` must have `intent_owner`.
- `ContractSeed` must have `scope_hint`.
- `ContractSeed` must have `acceptance_hint`.
- `ContractDraft` creation must not create an approved Contract.
- `ContractDraft` creation must not create `WorkItem` or task.
- `ContractDraft` creation must not create `GateDecision` or `Proof`.
- `ContractDraft` creation must not mutate `ContractSeed`.
- Repeated draft creation for the same `ContractSeed` should return
  `409 already_drafted` in v0.

Recommended v0 draft generation:

- `proposed_scope` may be seeded directly from `ContractSeed.scope_hint`.
- `proposed_acceptance_criteria` may be seeded directly from
  `ContractSeed.acceptance_hint`.
- `proposed_non_goals` may be empty initially.
- `proposed_constraints` may be empty initially.
- `proposed_expected_checks` may be empty, or may contain only checks already
  known from existing explicit context.
- `proposed_proof_expectations` should be present but minimal and explicit; it
  must not invent final proof requirements.
- No LLM drafting, rewrite, or enrichment is part of this boundary.

A draft should be created from the current `ContractSeed` snapshot at the time
of the explicit draft request.

## Proposed events

Recommended event:

- `contract_draft.created`

Event payload should include:

- `contract_draft_id`
- `contract_seed_id`
- `goal_id`
- `repo_binding_id`
- `source_refs`
- `created_at`

The payload should be sufficient to link the draft to its seed and source Goal
without implying approval or executable work.

Optional future events:

- `contract_draft.updated`
- `contract_draft.superseded`
- `contract_draft.cancelled`

Do not introduce these events in this boundary:

- `contract.approved`
- `work_item.created`
- `run.started`
- `gate.decision_written`
- `proof.created`

## Relationship to ContractSeed

`ContractDraft` reads the `ContractSeed` snapshot and derives proposed terms
from it. It does not mutate the seed.

`ContractSeed` remains `created` unless a later boundary defines an explicit
seed transition.

If a seed is changed, regenerated, or superseded later, any draft supersession
or replacement must be explicit in a future boundary. A draft must not silently
track a changed seed as a live view.

`ContractDraft` must preserve `source_refs` back to the `ContractSeed` and the
source Goal.

## Relationship to approved Contract

Approved Contract is a later boundary.

A `ContractDraft` requires review and approval before it can become an approved
Contract. Approval may add, confirm, or reject final scope, non-goals,
acceptance criteria, checks, proof expectations, risk policy, ownership, and
execution readiness.

`ContractDraft` does not imply approval. It does not make work executable and
must not be treated as final contract authority.

## Relationship to WorkItems

`WorkItem` objects are not created from `ContractDraft`.

WorkItems are created later from an approved Contract only. A draft must not be
shown or handed off as runnable work.

A future UI may display `ContractDraft` as a draft or review item, but that is a
view concern and does not make the draft executable.

## Relationship to ADR-0010 persistence

This ADR defines a domain boundary, not a storage backend.

The current prototype may remain in-memory unless a later storage slice says
otherwise. Durable storage semantics for Organization / Project / RepoBinding
and broader server persistence remain governed by ADR-0010 and later persistence
decisions.

This ADR does not introduce:

- new Postgres tables
- migrations
- durable event-log requirements
- Organization / Project / RepoBinding changes
- `RepoBinding` persistence changes

## Rejected alternatives

### ContractSeed automatically creates ContractDraft

Rejected. Seed creation and draft creation are separate auditable transitions.
A seed is a snapshot bridge, not a draft.

### ContractDraft is created automatically during readiness re-check

Rejected. Readiness re-check only decides whether a Goal is ready for contract
seed. It must not create contract artifacts as a hidden side effect.

### ContractDraft directly becomes approved Contract

Rejected. Approval requires a separate boundary, review rules, events, and
policy controls.

### ContractDraft directly creates WorkItem

Rejected. Work items are executable delivery objects and must come only after an
approved Contract boundary.

### CLI, skill, web, or integration owns ContractDraft truth

Rejected. External surfaces may transport draft requests or render draft state,
but the server owns canonical draft truth.

### ContractDraft is treated as executable work

Rejected. A draft is proposed agreement text and structured terms. It is not an
execution packet.

### ContractDraft writes GateDecision or Proof

Rejected. Gate decision and proof belong to later verification boundaries after
approved contract, execution, and evidence collection.

### LLM-generated draft is accepted as canonical approval

Rejected. LLM drafting, if introduced later, may propose text, but approval is a
separate server-owned and human/policy-governed boundary.

### ContractSeed is skipped and Goal becomes ContractDraft directly

Rejected. Skipping the seed would collapse intent readiness and drafting into
one transition and erase the canonical snapshot used as draft input.

## Non-goals

This ADR does not define or implement:

- code implementation
- endpoint finalization as public API canon
- approved Contract
- approval policy
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
- LLM contract drafting
- policy engine
- CLI integration
- web UI

## Implementation implications

A later bounded implementation slice may add:

- `ContractDraft` DTO
- in-memory `ContractDraftStore`
- `contractdraft` service
- endpoint candidate: `POST /v1/contract-seeds/{id}/contract-draft`
- duplicate guard by `contract_seed_id`
- event: `contract_draft.created`

The endpoint candidate is not final public API canon.

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

- Should repeated draft creation return `409 already_drafted` or return the
  existing draft?
- Should `proposed_expected_checks` be empty in v0 or seeded from repo readiness
  context when that context exists?
- Should `proposed_proof_expectations` be required in v0?
- Should draft generation require human confirmation at creation time?
- Should `ContractDraft` support update or supersession in v0?
- Should the approval boundary be next after draft creation, or should draft
  review / update come first?

Recommended initial direction:

- repeated draft creation returns `409 already_drafted`
- `proposed_expected_checks` may be empty in v0 unless existing explicit context
  is already available
- `proposed_proof_expectations` should be present but can be minimal
- no approval at draft creation
- draft update / review is a later boundary
- approval boundary comes after draft creation / review
