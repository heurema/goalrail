# ADR-0006 — Goal clarification readiness boundary

Status: accepted
Date: 2026-04-25

## Context

Goalrail now has a bounded server-owned intake and Goal path:

- `POST /v1/intake`
- `GET /v1/intake/{id}`
- `POST /v1/intake/{id}/promote`

The server stores `IntakeRecord` and non-executable `Goal` as in-memory
prototypes. Promotion appends `goal.created` and `intake.promoted_to_goal`
events, and repeated promotion returns `409 already_promoted`.

ADR-0005 defines `Goal` as normalized server-owned intent. A `Goal` is not a
contract, work item, executable task, approval, gate decision, or proof.

The next domain question is how a created `Goal` becomes ready for the later
contract seed boundary, or how Goalrail decides that more information is needed.
This ADR defines only that readiness decision boundary. It does not implement a
clarification engine.

## Decision

Goalrail introduces a Goal clarification readiness boundary in the intent plane.

A created `Goal` may be evaluated into one of these intent-plane outcomes:

- `needs_clarification`
- `ready_for_contract_seed`
- `rejected`

The readiness decision is a state transition over `Goal`. It is not a contract,
not an approval, not executable work, and not a work item plan.

Goal readiness truth is server-owned canonical state. CLI, skills, web resources,
and integrations may request, render, or submit context for readiness later, but
they do not own the canonical readiness decision.

The first implementation should start as a deterministic server-owned readiness
check and state transition. It should not call an LLM, generate a contract seed,
create tasks, or ask/answer clarification questions automatically.

## Minimum readiness inputs

The readiness check may use the current `Goal` fields:

- `id`
- `intake_id`
- `repo_binding_id`
- `title`
- `summary`
- `source_refs`
- `request_author`
- `intent_owner`
- `state`
- `created_at`

Minimum information needed to mark a Goal `ready_for_contract_seed`:

- `repo_binding_id` is present
- `title` is present
- `summary` is present
- `request_author.kind` and `request_author.id` are present
- `intent_owner.kind` and `intent_owner.id` are present
- at least one source reference points back to the originating intake

If these minimum fields are missing or unusable, the Goal should move to
`needs_clarification` rather than contract generation.

A Goal may move to `rejected` only when there is an explicit reason that the work
should not proceed through Goalrail. Rejection is not the default fallback for
missing information.

## State transitions

Allowed readiness transitions for this boundary:

```text
created -> needs_clarification
created -> ready_for_contract_seed
created -> rejected
needs_clarification -> ready_for_contract_seed
needs_clarification -> rejected
```

Out of scope for this ADR:

- `ready_for_contract_seed -> ContractDraft`
- execution states
- ticket states
- work item states
- gate/proof states

`ready_for_contract_seed` means only that the Goal has enough normalized intent
for a later contract seed step. It does not mean a contract exists or is approved.

## Clarification requests

This ADR does not introduce a durable `ClarificationRequest` object yet.

For the next implementation slice, a readiness result may include lightweight
machine-readable missing-information reasons, for example:

- `missing_summary`
- `missing_repo_binding_id`
- `missing_intent_owner`
- `missing_source_ref`

Those reasons are diagnostic output for the readiness decision. They are not yet
a conversation, question workflow, or approval policy.

A future clarification slice may introduce:

- `ClarificationRequest`
- `ClarificationQuestion`
- `ClarificationAnswer`
- target actor selection
- question lifecycle
- answer validation

That future slice must stay separate from contract generation.

## Proposed events

The readiness boundary writes explicit events:

- `goal.marked_needs_clarification`
- `goal.marked_ready_for_contract_seed`
- `goal.rejected`

Event payloads should include enough information to reconstruct:

- Goal ID
- previous state
- new state
- reason codes, if any
- timestamp

No event in this boundary may create or imply `ContractDraft`, `ApprovedContract`,
`WorkItem`, `Task`, `GateDecision`, or `Proof`.

## API direction for a later implementation slice

A later bounded implementation may add an endpoint such as:

```text
POST /v1/goals/{id}/readiness
```

Recommended behavior for the first prototype:

- load `Goal`
- run deterministic minimum-field checks
- update Goal state to one of the readiness states
- append exactly one readiness event
- return the updated Goal plus readiness reasons

This endpoint choice is not final product API canon. It is a bounded prototype
shape for the next server slice.

## Rejected alternatives

### Generate ContractDraft directly from Goal

Rejected. `ready_for_contract_seed` is a readiness state, not contract creation.
Contract seed generation and ContractDraft creation need their own boundary.

### Create WorkItems from Goal readiness

Rejected. WorkItems are executable units and belong after contract shaping.
Goal readiness is intent-plane state only.

### Use LLM enrichment as the first readiness mechanism

Rejected for this stage. The first server behavior should be deterministic and
inspectable before any composer or enrichment layer is added.

### Introduce ClarificationRequest immediately

Rejected for this step. The system first needs a stable readiness boundary and
reason vocabulary. Clarification objects can be introduced in a later slice when
actual question/answer flow is implemented.

### Local CLI or skill owns readiness truth

Rejected because CLI, skills, web resources, and integrations are adapters. The
server owns canonical Goal readiness state.

### Treat Goal readiness as tracker workflow status

Rejected. Goal states are internal intent-plane states, not replacements for
Jira, Linear, or customer-specific ticket statuses.

## Non-goals

This ADR does not define or implement:

- clarification request lifecycle
- clarification answers
- target actor routing
- contract seed generation
- ContractDraft
- ApprovedContract
- WorkItems or Tasks
- approval policy
- gate/proof
- durable storage
- LLM enrichment
- dedupe
- CLI integration
- frontend updates

## Implementation implications

The next implementation slice may add:

- Goal readiness service
- in-memory Goal state update support
- deterministic readiness reason codes
- readiness events
- `POST /v1/goals/{id}/readiness`
- tests for each transition and non-goal boundary

That implementation slice must not add clarification objects, contract seed,
ContractDraft, WorkItems, gate/proof, durable storage, CLI integration, or web
changes.

## Open questions

1. Should the first readiness endpoint be `POST /v1/goals/{id}/readiness`, or
   should readiness evaluation run automatically after Goal promotion?
2. Should readiness reasons be a small enum from the start, or plain strings
   until the first clarification object exists?
3. Should `rejected` require an explicit user/system actor and reason in the
   first prototype, or be deferred until policy exists?
