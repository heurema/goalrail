# ADR-0013 — ContractSeed boundary

Status: accepted
Date: 2026-04-26

## Context

Goalrail now has a bounded server-owned intent path that can reach a
readiness-checked Goal:

- `POST /v1/intake`
- `GET /v1/intake/{id}`
- `POST /v1/intake/{id}/promote`
- `POST /v1/goals/{id}/readiness`
- `POST /v1/goals/{id}/clarification-requests`
- `POST /v1/clarification-requests/{id}/answers`
- `POST /v1/clarification-answers/{id}/apply`
- explicit `POST /v1/goals/{id}/readiness` after applied answers

The current flow can move a Goal to `ready_for_contract_seed`. That state means
only that the intent-plane information is normalized enough for a later contract
seed boundary.

A Goal is still not a contract. Goal hints are still intent-plane hints, not
final contract scope or acceptance criteria. The next boundary is creating a
server-owned `ContractSeed` from a Goal that is already
`ready_for_contract_seed`.

This boundary remains before `ContractDraft`, approval, `WorkItem`, execution,
`GateDecision`, and `Proof`.

ADR-0010 defines the Organization / Project / RepoBinding / Postgres persistence
foundation. This ADR defines a domain boundary and does not change storage or
persistence requirements.

## Decision

The server may create a `ContractSeed` only from a Goal whose state is
`ready_for_contract_seed`.

`ContractSeed` is server-owned canonical state. It is a bridge from normalized,
readiness-checked Goal intent into future contract drafting. It captures the
intent inputs needed for a later `ContractDraft` boundary.

`ContractSeed` creation is explicit. Readiness re-check must not automatically
create a seed.

`ContractSeed` does not:

- create `ContractDraft`
- create `ApprovedContract`
- create `WorkItem` or task
- approve anything
- make work executable
- start execution
- create `GateDecision`
- create `Proof`

`ContractDraft` generation remains a later explicit boundary. Work item creation
remains later and only after an approved contract boundary.

## Proposed object model

### ContractSeed

Minimal conceptual fields:

| Field | Intent |
| --- | --- |
| `id` | Server-owned stable seed ID. |
| `goal_id` | Goal used as seed input. |
| `repo_binding_id` | Repository binding context carried by the Goal. |
| `title` | Seed title copied from `Goal.title`. |
| `intent_summary` | Normalized intent summary copied from `Goal.summary`. |
| `intent_owner` | Actor responsible for the intent, copied from `Goal.intent_owner`. |
| `scope_hint` | Intent-plane scope hint copied from `Goal.scope_hint`. |
| `acceptance_hint` | Intent-plane outcome hint copied from `Goal.acceptance_hint`. |
| `source_refs` | Source references such as Goal ID and original IntakeRecord ref when available. |
| `state` | Seed lifecycle state. |
| `created_at` | Server timestamp for seed creation. |

Recommended v0 state:

- `created`

Field meanings:

- `title` comes from `Goal.title`.
- `intent_summary` comes from `Goal.summary`.
- `intent_owner` comes from `Goal.intent_owner`.
- `scope_hint` comes from `Goal.scope_hint`.
- `acceptance_hint` comes from `Goal.acceptance_hint`.
- `source_refs` include the Goal ID and original IntakeRecord reference if the
  server has it.

Important constraints:

- `intent_summary` is not contract text.
- `scope_hint` is not final contract scope.
- `acceptance_hint` is not acceptance criteria.
- `ContractSeed` is not `ContractDraft`.
- `ContractSeed` is not approval.
- `ContractSeed` is not executable work.

## Creation rules

Minimum creation rules:

- Goal must exist.
- `Goal.state` must be `ready_for_contract_seed`.
- Goal must have `repo_binding_id`.
- Goal must have `title`.
- Goal must have `summary`.
- Goal must have `intent_owner`.
- Goal must have `scope_hint`.
- Goal must have `acceptance_hint`.
- Repeated seed creation for the same Goal should return `409 already_seeded`
  in v0.
- Successful seed creation must not create `ContractDraft`, `WorkItem`,
  `GateDecision`, or `Proof`.

A seed should be created from the current Goal fields at the time of the
explicit seed request.

## Events

Recommended event:

- `contract_seed.created`

Event meaning:

- `contract_seed.created`: canonical seed state was created from a
  `ready_for_contract_seed` Goal. Payload should include `seed_id`, `goal_id`,
  `repo_binding_id`, copied intent fields or safe references to them,
  `source_refs`, and timestamp.

Do not introduce these events in this boundary:

- `contract.draft_created`
- `contract.approved`
- `work_item.created`
- `run.started`
- `gate.decision_written`
- `proof.created`

## Relationship to Goal

`ContractSeed` reads the current Goal fields and captures them as a snapshot.
It does not mutate the Goal.

The Goal remains in `ready_for_contract_seed` unless a later boundary defines an
explicit Goal state transition.

If Goal fields change later, the existing `ContractSeed` must not silently track
those changes as a live view. Seed supersession, regeneration, or replacement
must be explicit in a future boundary.

The audit trail links the seed to its source Goal through `goal_id`,
`source_refs`, and the `contract_seed.created` event.

## Relationship to ContractDraft

`ContractDraft` is a later boundary.

A future `ContractDraft` generator may use `ContractSeed` as input, but
`ContractSeed` itself has no final contract authority. It must not be treated as
approved scope, final acceptance criteria, execution instructions, or a work
packet.

`ContractSeed` exists to preserve a canonical bridge between the intent plane
and future contract drafting without collapsing those stages into one hidden
transition.

## Relationship to ADR-0010 persistence

This ADR defines a domain boundary, not a storage backend.

The current prototype may stay in-memory unless a later storage slice says
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

### Goal readiness automatically creates ContractSeed

Rejected. `ready_for_contract_seed` is only a Goal state. Seed creation must be
an explicit server-owned transition.

### Readiness re-check automatically creates ContractSeed

Rejected. Readiness re-check answers whether the Goal is ready for seeding. It
must not create contract artifacts as a hidden side effect.

### ContractSeed directly creates ContractDraft

Rejected. Contract drafting needs a separate boundary, validation rules, and
audit events.

### ContractSeed directly creates WorkItem

Rejected. Work items are executable delivery objects and belong after contract
shaping and approval.

### CLI, skill, web, or integration owns ContractSeed truth

Rejected. External surfaces may request or display seed state, but the server
owns canonical `ContractSeed` state.

### ContractSeed is only a derived view with no canonical event

Rejected. The bridge from intent to contract drafting must be inspectable and
auditable through canonical state and event history.

### Goal becomes ContractDraft directly

Rejected. Direct Goal-to-draft conversion would erase the bridge/snapshot stage
and make it harder to audit what readiness-checked intent was used for drafting.

### ContractSeed contains final contract scope or acceptance criteria

Rejected. The seed carries intent-plane hints. Final contract scope and
acceptance criteria belong to a later contract drafting boundary.

### ContractSeed creates GateDecision or Proof

Rejected. Gate decision and proof belong to later verification boundaries after
contract shaping, execution, and evidence collection.

## Non-goals

This ADR does not define or implement:

- code implementation
- endpoint finalization as public API canon
- `ContractDraft`
- `ApprovedContract`
- approval policy
- risk assessment
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

The next implementation slice may add:

- `ContractSeed` DTO / value types
- in-memory `ContractSeedStore`
- contract seed creation service
- endpoint candidate: `POST /v1/goals/{id}/contract-seed`
- duplicate guard by `goal_id`
- event: `contract_seed.created`
- tests proving seed creation only from `Goal(ready_for_contract_seed)`

Endpoint choice is not final public API canon.

The immediate implementation slice must not:

- create `ContractDraft`
- create `ApprovedContract`
- create `WorkItem`
- approve anything
- start execution
- write `GateDecision`
- create `Proof`
- modify ADR-0010 persistence scope

## Open questions

1. Should repeated seed creation return `409 already_seeded`, or return the
   existing seed?
2. Should `ContractSeed` always be a snapshot, or should a later projection also
   expose a live view over the Goal?
3. Should `ContractSeed` include `intake_id` directly, or only include source
   references that can point to the original intake when available?
4. Should `ContractDraft` generation be an explicit endpoint after
   `ContractSeed`?
5. Should `ContractSeed` creation require human confirmation?

Recommended initial direction:

- repeated creation returns `409 already_seeded`
- `ContractSeed` is a snapshot
- `source_refs` include Goal and Intake references where available
- `ContractDraft` generation is a later explicit endpoint and boundary
- no human approval is required at the seed stage
