# ADR-0012 — Explicit readiness re-check after applied answers

Status: accepted
Date: 2026-04-26

## Context

Goalrail now has a bounded server-owned intent path:

- `POST /v1/intakes`
- `GET /v1/intakes/{id}`
- `POST /v1/intakes/{id}/goals`
- `POST /v1/goals/{id}/readiness`
- `POST /v1/goals/{id}/clarifications`
- `POST /v1/clarifications/{id}/answers`
- `POST /v1/answers/{id}/applications`

A Goal can be promoted from an `IntakeRecord`. Goal readiness can identify
missing intent-plane hints and move the Goal to `needs_clarification`.
`ClarificationRequest` asks for missing information. `ClarificationAnswer`
records canonical answer evidence. Answer application can update Goal
intent-plane hints.

Answer application does not automatically re-check Goal readiness. The next
boundary is an explicit server-owned readiness re-check after applied answers.
This boundary is still before contract seed generation, `ContractDraft`,
`WorkItem`, `GateDecision`, or `Proof`.

ADR-0010 defines the Organization / Project / RepoBinding / Postgres persistence
foundation. This ADR is a domain boundary for readiness and does not change
storage or persistence requirements.

## Decision

The server may explicitly re-check Goal readiness after answers have been applied
to Goal hints.

The re-check reuses deterministic Goal readiness logic. It reads the current
Goal state and fields, including any hint updates created by answer application,
and updates the Goal to one readiness outcome:

- `needs_clarification`
- `ready_for_contract_seed`
- `rejected`

Readiness re-check is a server-owned canonical state transition. CLI, skills,
web resources, and integrations may request or display the result, but they do
not own readiness truth.

Readiness re-check writes readiness events and remains explicit. Answer
application must not call readiness re-check implicitly.

Readiness re-check does not:

- create contract seed
- create `ContractDraft`
- create `ApprovedContract`
- create `WorkItem` or task
- approve anything
- make work executable
- create `GateDecision`
- create `Proof`

`ready_for_contract_seed` means only that enough normalized intent exists for a
future contract seed boundary. It is not contract creation.

## API / transition direction

Recommended prototype direction: reuse the existing explicit endpoint:

```text
POST /v1/goals/{id}/readiness
```

The same endpoint may be called after answer application. It may also be called
more than once. Repeated checks are audit events, not hidden automation.

No automatic call is made from:

- `POST /v1/answers/{id}/applications`
- answer recording
- clarification request creation
- Goal promotion

A later orchestration layer may decide when to request the re-check, but the
canonical transition still belongs to the server readiness endpoint or service.

## State semantics

### `needs_clarification`

The Goal is still missing required intent-plane information. It is not ready for
contract seed generation. More clarification may be needed.

### `ready_for_contract_seed`

The Goal has enough normalized intent for a later contract seed boundary.

`ready_for_contract_seed` does not:

- create contract seed
- create `ContractDraft`
- create `ApprovedContract`
- create `WorkItem`
- approve work
- make work executable
- create `GateDecision`
- create `Proof`

### `rejected`

The Goal should not proceed through Goalrail. Rejection requires an explicit
reason and must not be used as the default fallback for missing information.

## Events

This boundary reuses readiness events from the existing readiness prototype and
ADR-0006.

Recommended events:

- `goal.readiness_checked`
- `goal.marked_needs_clarification`
- `goal.marked_ready_for_contract_seed`
- `goal.rejected`

Event meanings:

- `goal.readiness_checked`: an explicit readiness evaluation was requested and
  completed. Payload should include Goal ID, checked fields or reason codes,
  previous state, resulting state, and timestamp.
- `goal.marked_needs_clarification`: Goal moved or remained in
  `needs_clarification` because required intent-plane information is missing.
- `goal.marked_ready_for_contract_seed`: Goal moved to
  `ready_for_contract_seed`.
- `goal.rejected`: Goal moved to `rejected` with an explicit reason.

Repeated checks should at least append `goal.readiness_checked`. Whether an
unchanged state also appends a state event is an implementation detail to keep
bounded in the next slice.

No event in this boundary may create or imply:

- `contract.seed_created`
- `contract.created`
- `work_item.created`
- `gate.decision_written`
- `proof.created`

## Relationship to applied answers

Applied answers may update Goal intent-plane fields such as:

- `summary`
- `scope_hint`
- `acceptance_hint`

Future actor-shaped answer application may update `intent_owner` when the value
is explicit enough to produce an `ActorRef`. Raw text must not be silently
converted into an owner.

Readiness re-check reads the current Goal fields. It does not need to know which
answer caused a hint update. The audit trail links answer application events and
later readiness events by Goal ID, event order, and event payloads.

There is no hidden transition from answer application to readiness. A Goal can
have updated hints and still require an explicit readiness re-check before it is
marked `ready_for_contract_seed`.

## Relationship to ADR-0010 Postgres foundation

This ADR is about the domain readiness boundary, not storage.

The current prototype may continue to use in-memory Goal store and in-memory
event log for the intent-plane flow. Durable storage semantics remain governed
by ADR-0010 and later persistence decisions.

This ADR does not introduce:

- new Postgres tables
- migrations
- durable event-log requirements
- project/repo binding changes
- Organization / Project / RepoBinding implementation changes

## Rejected alternatives

### Answer application automatically triggers readiness re-check

Rejected. Hidden transition chains would collapse answer application,
readiness, and contract-seed eligibility into one operation. The re-check must
remain explicit and auditable.

### Readiness re-check creates contract seed or ContractDraft

Rejected. `ready_for_contract_seed` is only a Goal state. Contract seed and
`ContractDraft` need their own later boundary.

### Readiness re-check creates WorkItem

Rejected. Work items are executable delivery objects and belong after contract
shaping and approval boundaries.

### CLI, skill, or web owns readiness truth

Rejected. Local and external surfaces may request or display readiness, but the
server owns canonical Goal readiness state.

### Readiness re-check is only a hidden derived view

Rejected. Readiness must be an explicit server-owned check with auditable events,
not an unrecorded projection.

### Readiness re-check silently approves work

Rejected. Readiness is not approval and does not make work executable.

### Readiness re-check writes GateDecision or Proof

Rejected. Gate decision and proof belong to later verification boundaries.

## Non-goals

This ADR does not define or implement:

- code implementation
- endpoint finalization as public API canon
- contract seed generation
- `ContractDraft`
- `ApprovedContract`
- approval policy
- `WorkItem` or task planning
- `GateDecision`
- `Proof`
- durable storage changes
- Postgres migration
- LLM readiness assessment
- policy engine
- CLI integration
- web UI

## Implementation implications

The next implementation slice may:

- reuse `POST /v1/goals/{id}/readiness`
- ensure the existing readiness endpoint works after answer application
- ensure a Goal can move from `needs_clarification` to
  `ready_for_contract_seed` after required hints are applied
- append readiness events for explicit re-checks
- add tests for:
  - answer applied
  - explicit readiness re-check requested
  - Goal transitions to `ready_for_contract_seed`
  - no contract seed, `ContractDraft`, `WorkItem`, `GateDecision`, or `Proof`
    is created

The implementation slice must not:

- create contract seed
- create `ContractDraft`
- create `WorkItem`
- call a contract composer
- create `GateDecision`
- create `Proof`
- modify the Postgres foundation
- make answer application call readiness automatically

## Open questions

1. Should repeated readiness checks always append state transition events when
   the state is unchanged, or should `goal.readiness_checked` be the only audit
   event for unchanged results?
2. Should `ready_for_contract_seed` remain Goal state only for now, or should a
   separate readiness result projection appear later?
3. Should contract seed generation be an explicit endpoint after
   `ready_for_contract_seed`?
4. Should readiness re-check require applied answers, or can it be called
   anytime?

Recommended initial direction:

- repeated readiness checks remain audit events through `goal.readiness_checked`
- `ready_for_contract_seed` remains Goal state only for now
- contract seed generation is a later explicit endpoint and boundary
- readiness can be called anytime, but answer application never calls it
  implicitly
