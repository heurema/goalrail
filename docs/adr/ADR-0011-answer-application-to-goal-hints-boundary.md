# ADR-0011 — Answer application to Goal hints boundary

Status: accepted
Date: 2026-04-26

## Context

Goalrail now has a bounded server-owned intent path:

- `POST /v1/intake`
- `GET /v1/intake/{id}`
- `POST /v1/intake/{id}/promote`
- `POST /v1/goals/{id}/readiness`
- `POST /v1/goals/{id}/clarification-requests`
- `POST /v1/clarification-requests/{id}/answers`

Goal readiness can produce missing-information reason codes. A
`ClarificationRequest(open)` can ask questions for those reasons. A
`ClarificationAnswer(recorded)` can store canonical answer evidence for that
request.

Answer recording does not update `Goal` hints. The next boundary is applying a
recorded answer to Goal intent-plane hints. Applying answers is separate from
Goal readiness re-check, contract seed generation, `ContractDraft` creation,
and executable work.

## Decision

The server may apply a recorded `ClarificationAnswer` to Goal intent-plane
hints.

Answer application is a canonical server-owned transition. CLI, skills, web
resources, and integrations may transport requests to apply answers, but they do
not own canonical answer application truth.

Answer application updates only allowed Goal hint fields. It writes explicit
events and preserves the original `ClarificationAnswer` as canonical evidence.

Answer application does not:

- trigger Goal readiness re-check
- create contract seed
- create `ContractDraft`
- create `ApprovedContract`
- create `WorkItem` or task
- create `GateDecision`
- create `Proof`
- approve anything
- make work executable

Goal readiness re-check remains a later explicit boundary. Contract seed and
`ContractDraft` remain later boundaries after readiness says the Goal is ready
for contract seed.

## Allowed mappings

Only these answer mappings may update Goal intent-plane fields in this boundary:

- `goal.summary`
- `goal.intent_owner`
- `goal.scope_hint`
- `goal.acceptance_hint`

Mapping meanings:

| Mapping | Meaning |
| --- | --- |
| `goal.summary` | Normalized intent description. |
| `goal.intent_owner` | Actor responsible for the intent. |
| `goal.scope_hint` | High-level intent-plane scope hint. |
| `goal.acceptance_hint` | High-level intent-plane outcome hint. |

Important constraints:

- `goal.scope_hint` is not contract scope.
- `goal.acceptance_hint` is not acceptance criteria.
- Goal hints are not final contract terms.
- Contract semantics come later in a separate contract seed / `ContractDraft`
  boundary.

## Proposed object / event model

### v0 application record

Do not introduce a separate major canonical object for v0.

The canonical evidence remains `ClarificationAnswer`. The updated canonical
intent object remains `Goal`. Application can be represented by explicit event
payloads plus the updated Goal projection.

A future object may be introduced if application history needs richer querying:

```text
AnswerApplicationRecord
- id
- answer_id
- goal_id
- applied_mappings
- applied_by
- created_at
```

For v0, prefer event-backed application without a separate object unless an
implementation slice discovers that duplicate prevention or audit retrieval needs
a minimal in-memory application record.

### Events

Recommended events:

- `clarification.answer_applied_to_goal`
- `goal.hints_updated`

Event meanings:

- `clarification.answer_applied_to_goal`: the answer was applied to the target
  Goal. Payload links `answer_id`, `goal_id`, applied mappings, and actor.
- `goal.hints_updated`: Goal intent-plane hints changed. Payload includes
  changed fields and old / new values where safe.

No event in this boundary may create or imply:

- `goal.readiness_recheck_requested`
- `contract.seed_created`
- `contract.created`
- `work_item.created`
- `gate.decision_written`
- `proof.created`

## Validation rules

Minimum validation rules for answer application:

- `ClarificationAnswer` must exist.
- `ClarificationAnswer.state` must be `recorded`.
- Related `ClarificationRequest` must be `answered`.
- Related `Goal` must exist.
- Each answer item must correspond to a `ClarificationQuestion` in the request.
- Each referenced question must have a `maps_to` value in the allowed mappings.
- Unsupported `maps_to` must be rejected.
- `applied_by.kind` is required.
- `applied_by.id` is required.
- Application must be idempotency-protected.
- Repeated application should return `409 already_applied` in the first
  implementation.
- Applying an answer must not mutate `ClarificationAnswer` evidence.
- Applying an answer must not create contract, work item, gate decision, or
  proof.
- Applying an answer must not trigger readiness re-check.

Recommended first implementation behavior:

- require `applied_by`
- apply all supported mappings
- reject unsupported mappings
- reject ambiguous raw text for `goal.intent_owner`
- reject empty mapped values
- reject partial application in v0 unless a later ADR explicitly allows it
- return `409 already_applied` on repeated application

## Application semantics

### `goal.summary`

The answer value fills or replaces `Goal.summary`.

For v0, use direct string assignment after trimming surrounding whitespace. Do
not use LLM rewrite, summarization, or enrichment.

### `goal.intent_owner`

The answer value identifies the intent owner.

For v0, do not map arbitrary free-form text into an owner. Apply this mapping
only when the submitted value is explicitly actor-shaped enough to produce an
`ActorRef` with `kind` and `id`. If the first implementation still accepts only
raw string answer values, it should reject or defer `goal.intent_owner`
application unless that string is constrained by the request/question contract to
an explicit actor reference.

Directory lookup, identity verification, `display_name` enrichment, and actor
resolution remain separate identity/user boundaries.

### `goal.scope_hint`

The answer value fills `Goal.scope_hint`.

This remains a high-level intent-plane hint. It is not contract scope and must
not be treated as executable scope.

### `goal.acceptance_hint`

The answer value fills `Goal.acceptance_hint`.

This remains a high-level outcome hint. It is not acceptance criteria and must
not become a gate or proof expectation.

### Explicit non-inference

Answer application must not infer:

- final contract scope
- acceptance criteria
- proof expectations
- task lists
- runtime instructions
- gate decisions

## Rejected alternatives

### Answer recording automatically updates Goal hints

Rejected. Answer recording captures canonical evidence. Applying that evidence
to Goal hints is a separate server-owned transition.

### Answer application automatically triggers readiness re-check

Rejected. Hidden transition chains would collapse answer submission,
application, readiness, and contract-seed eligibility. Readiness re-check remains
a later explicit boundary.

### Answer application creates ContractDraft

Rejected. Applying hints clarifies intent. Contract drafting requires a later
contract seed / `ContractDraft` boundary.

### Answer application creates WorkItem

Rejected. Hints are not executable work. Work items require approved contract and
task-shaping boundaries.

### Local CLI or skill applies answers as canonical truth

Rejected. Local tools may transport answers or request application, but the
server owns canonical application state.

### LLM rewrites answers into canonical hints without explicit transition

Rejected. Any normalization or enrichment must be explicit, inspectable, and
bounded by a later decision. v0 uses deterministic direct assignment.

### Goal hints become the only record of answers

Rejected. `ClarificationAnswer` remains canonical evidence. Applying answers
must not delete, overwrite, or replace the answer evidence trail.

### Applying answers deletes or mutates ClarificationAnswer evidence

Rejected. Recorded answers are evidence. If correction or supersession is needed
later, it must be represented explicitly rather than mutating evidence silently.

## Non-goals

This ADR does not define or implement:

- code implementation
- endpoint finalization as public API canon
- Goal readiness re-check
- contract seed
- `ContractDraft`
- approval policy
- `WorkItem` or task planning
- `GateDecision`
- `Proof`
- durable storage
- LLM normalization
- policy engine
- CLI integration
- web UI

## Implementation implications

A later bounded implementation slice may add:

- endpoint candidate: `POST /v1/clarification-answers/{id}/apply`
- answer application service
- allowed mapping application
- Goal hint update persistence
- `clarification.answer_applied_to_goal` event
- `goal.hints_updated` event
- duplicate application guard

Endpoint choice is not final public API canon.

The immediate implementation must not:

- re-check readiness
- create contract seed
- create `ContractDraft`
- create `WorkItem`
- create `GateDecision`
- create `Proof`

## Open questions

- Should repeated application always return `409 already_applied`, or should a
  later durable implementation return the existing application result?
- Should `applied_by` always be a human/user actor, or can system-applied
  transitions be allowed later?
- Should v0 accept actor-shaped `goal.intent_owner` values, or defer that
  mapping until typed answer values / identity resolution exist?
- Should future application support partial mapping, or continue requiring all
  answer items to be mappable?
- Should readiness re-check remain an explicit endpoint, or become automatic in
  a later slice after application semantics are stable?

Recommended initial direction:

- repeated application -> `409 already_applied`
- `applied_by` required
- v0 rejects ambiguous raw text for `goal.intent_owner`; apply it only from an
  explicit actor-shaped value or defer the mapping
- unsupported mappings rejected
- readiness re-check remains explicit and separate
