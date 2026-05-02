# ADR-0007 — Clarification request boundary

Status: accepted
Date: 2026-04-25

## Context

Goalrail now has a bounded server-owned intent path:

- `POST /v1/intakes`
- `GET /v1/intakes/{id}`
- `POST /v1/intakes/{id}/goals`
- `POST /v1/goals/{id}/readiness`

A received `IntakeRecord` can be promoted into a non-executable `Goal`.
The deterministic Goal readiness check can then mark the Goal as
`needs_clarification`, `ready_for_contract_seed`, or `rejected` using explicit
reason codes.

The next boundary is asking for missing information when a Goal is not ready.
That boundary must collect clarification without turning the Goal into a
contract, task, approval, or executable work.

Clarification must stay server-owned and auditable. CLI, skills, web resources,
and integrations may deliver questions and submit answers later, but they do not
own canonical clarification truth.

## Decision

The server may create a `ClarificationRequest` for a `Goal` in
`needs_clarification`.

A `ClarificationRequest` captures the missing information requested from the
best available clarification target. It is canonical server-owned state in the
intent plane.

A `ClarificationAnswer` records submitted answers as canonical server-owned
state. Answers are evidence of clarification; they are not approval and do not
make work executable.

Answers may update intent-plane hints on `Goal` through a server-owned
transition. After answers are recorded and applied, Goal readiness must be
re-checked before any later contract seed boundary can run.

Clarification does not create:

- contract seed
- `ContractDraft`
- `ApprovedContract`
- `WorkItem` or task
- `GateDecision`
- `Proof`

## Proposed object model

### ClarificationRequest

Minimal conceptual fields:

| Field | Intent |
| --- | --- |
| `id` | Server-owned stable request ID. |
| `goal_id` | Goal that needs information. |
| `reason_codes` | Readiness reason codes that caused the request. |
| `questions` | Clarification questions grouped for the target. |
| `target` | Best available clarification target. |
| `state` | Request lifecycle state. |
| `created_at` | Server timestamp for request creation. |

### ClarificationQuestion

Minimal conceptual fields:

| Field | Intent |
| --- | --- |
| `id` | Stable question ID inside the request. |
| `text` | Human-readable question text. |
| `why_needed` | Short explanation of why the answer is needed. |
| `answer_type` | Expected answer shape. |
| `maps_to` | Intent-plane field the answer can help update. |

Allowed `answer_type` values for v0:

- `text`
- `choice`
- `boolean`

Allowed `maps_to` values for v0:

- `goal.summary`
- `goal.intent_owner`
- `goal.scope_hint`
- `goal.acceptance_hint`

`maps_to` is an intent-plane hint mapping, not contract field writing.
`scope_hint` is not contract scope. `acceptance_hint` is not acceptance criteria.

### ClarificationTarget

Minimal conceptual fields:

| Field | Intent |
| --- | --- |
| `role` | Target role for answering. |
| `actor_ref` | Optional concrete actor reference. |
| `preferred_surface` | Optional delivery surface such as web, CLI, tracker, or email later. |

Allowed target roles for v0:

- `request_author`
- `intent_owner`
- `delivery_owner`
- `repo_owner`
- `policy_owner`

### ClarificationAnswer

Minimal conceptual fields:

| Field | Intent |
| --- | --- |
| `id` | Server-owned stable answer ID. |
| `request_id` | Clarification request being answered. |
| `goal_id` | Goal the answer clarifies. |
| `answers` | Answer items keyed by question. |
| `submitted_by` | Actor that submitted the answer. |
| `created_at` | Server timestamp for answer recording. |

### ClarificationAnswerItem

Minimal conceptual fields:

| Field | Intent |
| --- | --- |
| `question_id` | Question being answered. |
| `value` | Submitted value matching the question answer type. |

A `ClarificationAnswer` should be immutable once recorded. If a correction is
needed, record another answer or supersede the request through explicit events.

## Proposed states

`ClarificationRequest` states:

- `open`
- `answered`
- `cancelled`
- `superseded`

State meanings:

- `open`: waiting for an answer.
- `answered`: answer was recorded.
- `cancelled`: request is no longer needed.
- `superseded`: request was replaced by a newer request after Goal or readiness
  state changed.

`ClarificationAnswer` is immutable once recorded and does not need a lifecycle
state in the first boundary.

## Proposed events

Recommended events:

- `clarification.requested`
- `clarification.answered`
- `clarification.cancelled`
- `clarification.superseded`
- `goal.hints_updated`
- `goal.readiness_recheck_requested`

Optional future events:

- `clarification.routed`
- `clarification.escalated`

No event in this boundary may create or imply:

- `contract.created`
- `contract.seed_created`
- `work_item.created`
- `gate.decision_written`
- `proof.created`

## Routing rules

Default routing is deterministic and conceptual for this ADR only:

- If missing information comes from developer-originated intake, target
  `request_author` by default.
- If missing information concerns business intent, target `intent_owner`.
- If missing information concerns repository ownership, target `repo_owner`.
- If missing information concerns policy ownership, target `policy_owner`.
- If target is unknown, target `intent_owner` when present, otherwise
  `request_author`.

This boundary does not implement a policy engine. Future policy may override the
routing defaults with explicit server-owned decisions.

## Answer application rules

A `ClarificationAnswer` is stored as canonical evidence of the answer. Goalrail
must not make `Goal` the only place where answer content lives.

The server may update `Goal` intent-plane hints based on answer mappings:

- `goal.summary`
- `goal.intent_owner`
- `goal.scope_hint`
- `goal.acceptance_hint`

Goal updates must be evented, for example with `goal.hints_updated`.

Updating hints does not create `ContractDraft`, `ApprovedContract`, `WorkItem`,
`Task`, `GateDecision`, or `Proof`.

After answers are applied, Goal readiness must be re-checked through a later
implementation slice or explicit transition before any contract seed boundary.
The first implementation should prefer an explicit re-check over hidden
automatic transitions.

## Rejected alternatives

### ClarificationAnswer directly creates ContractDraft

Rejected. Answers clarify intent. Contract seed generation and ContractDraft
creation require their own later boundary.

### ClarificationRequest becomes a chat thread without canonical state

Rejected. Clarification must remain auditable server-owned state, not an opaque
conversation transcript owned by a surface or integration.

### Local CLI or skill owns clarification truth

Rejected. CLI, skills, web resources, and integrations are adapters. They may
transport questions and answers, but the server owns canonical clarification
state.

### Goal readiness auto-fills missing hints with LLM output as canonical truth

Rejected. The first clarification path must preserve explicit submitted answers
and auditable state. LLM question generation or enrichment can be evaluated later
but must not become hidden canonical truth.

### Clarification becomes generic ticket status workflow

Rejected. Clarification is an intent-plane information request, not a replacement
for Jira, Linear, or customer-specific ticket statuses.

### ClarificationAnswer is treated as approval

Rejected. Answers provide missing information. Approval belongs to a later
contract boundary and must not be inferred from an answer.

## Non-goals

This ADR does not define or implement:

- clarification engine implementation
- endpoint design as final API canon
- LLM question generation
- policy engine
- contract seed
- `ContractDraft`
- `ApprovedContract`
- approval
- `WorkItem` or task
- `GateDecision`
- `Proof`
- durable storage
- CLI integration
- web UI

## Implementation implications

The first bounded implementation slice after this ADR should add only request
creation:

- `ClarificationRequest` DTO
- in-memory `ClarificationStore`
- deterministic request creation from readiness reason codes
- endpoint candidate: `POST /v1/goals/{id}/clarifications`
- event for request creation, such as `clarification.requested`

That first implementation slice must not add `ClarificationAnswer`, answer
application, Goal hint updates, readiness re-check automation, contract seed
generation, ContractDraft, WorkItems, gate/proof, durable storage, CLI
integration, or web UI unless a separate boundary explicitly authorizes them.

Later bounded slices may add:

- `ClarificationAnswer` DTO
- answer endpoint candidate: `POST /v1/clarifications/{id}/answers`
- answer recording events
- Goal hint update events
- explicit readiness re-check request events

Endpoint choices in this ADR are prototype candidates, not final product API
canon.

## Open questions

1. Should `ClarificationAnswer` immediately update Goal hints, or should a
   separate explicit apply step exist?
2. Should one `ClarificationRequest` contain multiple questions, or should there
   be one request per reason code?
3. Should readiness re-check happen automatically after answer submission, or be
   explicit?
4. Should request target be selected by deterministic defaults first, with a
   policy engine later?

Initial direction across clarification implementation slices:

- one `ClarificationRequest` may contain multiple questions
- answer is stored separately
- answer application updates Goal hints through a server-owned transition
- readiness re-check is explicit at first to avoid hidden contract-adjacent
  transitions
