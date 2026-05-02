# ADR-0009 — Clarification answer boundary

Status: accepted
Date: 2026-04-26

## Context

Goalrail now has a bounded server-owned intent path:

- `POST /v1/intakes`
- `GET /v1/intakes/{id}`
- `POST /v1/intakes/{id}/promotions`
- `POST /v1/goals/{id}/readiness-checks`
- `POST /v1/goals/{id}/clarification-requests`

A received `IntakeRecord` can be promoted into a non-executable `Goal`.
Goal readiness can mark that Goal as `needs_clarification` using explicit
reason codes. The server can then create a `ClarificationRequest(open)` with
questions generated from those reason codes.

The next boundary is recording answers to an open clarification request.
Answer recording must stay separate from answer application, Goal hint updates,
readiness re-check, contract seed generation, and `ContractDraft` creation.

## Decision

The server may record a `ClarificationAnswer` for an open
`ClarificationRequest`.

A `ClarificationAnswer` is canonical server-owned evidence. It records values
submitted for questions in a clarification request. CLI, skills, web resources,
and integrations may transport answers, but they do not own canonical answer
truth.

Recording a `ClarificationAnswer` does not:

- update Goal hints
- trigger Goal readiness re-check
- create contract seed
- create `ContractDraft`
- create `ApprovedContract`
- create `WorkItem` or task
- create `GateDecision`
- create `Proof`
- approve anything
- make work executable

A `ClarificationRequest` may transition from `open` to `answered` after answer
recording. Answer application to Goal hints is a later explicit server-owned
transition. Goal readiness re-check after answer application is also a later
explicit step.

## Proposed object model

### ClarificationAnswer

Minimal conceptual fields:

| Field | Intent |
| --- | --- |
| `id` | Server-owned stable answer ID. |
| `request_id` | Clarification request being answered. |
| `goal_id` | Goal the answer clarifies. |
| `answers` | Answer items for questions in the request. |
| `submitted_by` | Actor that submitted the answer. |
| `state` | Answer recording state. |
| `created_at` | Server timestamp for answer recording. |

### ClarificationAnswerItem

Minimal conceptual fields:

| Field | Intent |
| --- | --- |
| `question_id` | Question being answered. |
| `value` | Raw submitted answer content. |

`ClarificationAnswerItem.value` is raw submitted answer content. It is not
automatically a Goal field, contract scope, acceptance criterion, approval, or
work item instruction. The relationship between answer values and Goal hints is
handled later by an explicit answer application boundary.

### Submitted actor

`submitted_by` uses the same actor reference shape used elsewhere:

| Field | Intent |
| --- | --- |
| `kind` | Actor kind such as user, integration, or system later. |
| `id` | Actor identifier. |
| `display_name` | Optional display label. |

### States

`ClarificationAnswer` state for the first boundary:

- `recorded`

`ClarificationRequest` transition for this boundary:

- `open` -> `answered`

A recorded answer is immutable in the first boundary. If correction or
supersession is needed later, it must be represented by explicit future state or
events rather than mutating the original answer silently.

## Validation rules

Minimum validation rules for answer recording:

- `ClarificationRequest` must exist.
- `ClarificationRequest` must be `open`.
- Answer must include at least one answer item.
- Each answer item must reference a question in the request.
- Unknown `question_id` must be rejected.
- Duplicate answer for the same `question_id` in one submission must be
  rejected.
- `submitted_by.kind` is required.
- `submitted_by.id` is required.
- Creating an answer for an `answered`, `cancelled`, or `superseded` request
  must fail.
- Successful answer recording must not create `ContractDraft`, `WorkItem`,
  `GateDecision`, or `Proof`.

For the first implementation, one `ClarificationAnswer` should answer all
questions in the request. Partial answers are deferred.

## Proposed events

Recommended events:

- `clarification.answer_recorded`
- `clarification.request_answered`

Event meanings:

- `clarification.answer_recorded`: canonical answer evidence was stored.
- `clarification.request_answered`: request state changed from `open` to
  `answered`.

Optional future events:

- `clarification.answer_rejected`
- `clarification.answer_superseded`

No event in this boundary may create or imply:

- `goal.hints_updated`
- `goal.readiness_recheck_requested`
- `contract.seed_created`
- `contract.created`
- `work_item.created`
- `gate.decision_written`
- `proof.created`

## Request / answer relationship

One `ClarificationRequest` may contain multiple questions. One
`ClarificationAnswer` may answer multiple questions in that request.

Recommended first implementation behavior:

- require answers for all questions in the request
- record exactly one answer for an open request
- transition the request to `answered` after successful answer recording
- return `409 already_answered` on repeated answer submission
- defer partial answer support
- defer answer supersession

This keeps request lifecycle semantics simple until a real user or integration
requires partial answer workflows.

## Answer application boundary

Answer recording, answer application, and readiness re-check are separate
boundaries.

Answer recording preserves canonical evidence. It does not rewrite `Goal`.

A later answer application boundary may map answer items into Goal intent-plane
hints:

- `goal.summary`
- `goal.intent_owner`
- `goal.scope_hint`
- `goal.acceptance_hint`

A later readiness boundary then evaluates the Goal again after hints are
applied.

Recommended later sequence:

```text
ClarificationAnswer(recorded)
  -> AnswerAppliedToGoalHints
  -> Goal readiness re-check
  -> ready_for_contract_seed or needs_clarification
```

`ready_for_contract_seed` still does not create a contract.

## Rejected alternatives

### ClarificationAnswer directly updates Goal hints in the same boundary

Rejected. Answer recording is canonical evidence capture. Updating Goal hints is
a separate server-owned transition that must be evented and reviewable.

### ClarificationAnswer directly creates ContractDraft

Rejected. Answers clarify intent. Contract seed generation and `ContractDraft`
creation require their own later boundary.

### ClarificationAnswer directly creates WorkItem

Rejected. Answers are not executable work and must not bypass contract approval.

### ClarificationAnswer is treated as approval

Rejected. An answer can provide information, but it does not approve scope,
execution, or delivery.

### Local CLI or skill owns answer truth

Rejected. CLI, skills, web resources, and integrations are adapters. They may
transport answers, but the server owns canonical answer truth.

### Answer lives only in chat or comment without canonical server state

Rejected. Clarification must remain auditable and queryable as server-owned
state.

### LLM rewrites answer into canonical truth without explicit server transition

Rejected. LLM normalization may assist later, but canonical answer evidence and
Goal hint updates must remain explicit server-owned transitions.

### Answer submission automatically triggers hidden readiness re-check

Rejected. Hidden readiness transitions make state harder to audit. Re-checking
Goal readiness should remain explicit until the workflow needs automation and
clear event semantics.

## Non-goals

This ADR does not define or implement:

- code changes
- endpoint finalization as public API canon
- answer application
- Goal hint updates
- automatic readiness re-check
- contract seed
- `ContractDraft`
- `ApprovedContract`
- approval policy
- `WorkItem` or task planning
- `GateDecision`
- `Proof`
- durable storage
- LLM answer normalization
- policy engine
- CLI integration
- web UI

## Implementation implications

A later implementation slice may add:

- `ClarificationAnswer` DTO
- in-memory answer store
- answer recording service
- request state transition to `answered`
- events:
  - `clarification.answer_recorded`
  - `clarification.request_answered`
- endpoint candidate:
  - `POST /v1/clarification-requests/{id}/answers`

That endpoint choice is not final product API canon.

The immediate implementation must not:

- apply answers to Goal hints
- re-check readiness
- create contract seed
- create `ContractDraft`
- create `WorkItem`
- create `GateDecision`
- create `Proof`

## Open questions

- Should the first implementation require answers for all questions, or allow
  partial answers?
- Should repeated answer submission return `409 already_answered` or allow
  supersession?
- Should answer values remain raw strings in v0, or support typed values
  immediately?
- Should answer application happen automatically in a later slice, or through an
  explicit endpoint?
- Should readiness re-check happen automatically after answer application, or
  remain explicit?

Recommended initial direction:

- require all questions answered in the first implementation
- repeated answer submission returns `409 already_answered`
- answer values are strings in v0
- answer application is explicit and separate
- readiness re-check remains explicit and separate
