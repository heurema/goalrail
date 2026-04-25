# ADR-0005 — Intake to Goal promotion boundary

Status: accepted
Date: 2026-04-25

## Context

Goalrail now has a bounded Go server prototype under `apps/server`.
The server accepts source-neutral raw intake through:

- `POST /v1/intake`
- `GET /v1/intake/{id}`

That slice creates `IntakeRecord` only, stores it in memory, and appends an
in-memory `intake.received` event. It does not create `Goal`, `ContractDraft`,
`ApprovedContract`, `WorkItem`, `GateDecision`, or `Proof`.

The next domain boundary is deciding how a received `IntakeRecord` becomes a
`Goal`. This boundary is still before clarification, before contract composition,
and before any executable work.

## Decision

The server may promote a received `IntakeRecord` into a `Goal`.

`Goal` is the first normalized intent object in server-owned canonical state.
It captures the intent that Goalrail can clarify and later use as input for
contract seed generation.

`Goal` remains non-executable:

- `Goal` is not an approved `Contract`.
- `Goal` is not a `WorkItem` or task.
- `Goal` does not imply approval.
- `Goal` does not authorize runtime execution.
- `Goal` promotion must not create `ContractDraft`, `ApprovedContract`, or
  `WorkItem` records.

Goal promotion writes explicit events so later storage and projections can keep
promotion history inspectable.

## Proposed object model

Minimal `Goal` fields:

| Field | Intent |
| --- | --- |
| `id` | Server-owned stable Goal ID. |
| `intake_id` | Source `IntakeRecord` that caused the Goal. |
| `repo_binding_id` | Repository binding context carried from intake. |
| `title` | Short normalized goal title; may initially come from intake title. |
| `summary` | Normalized description of intent; not a contract and not acceptance criteria. |
| `source_refs` | References back to intake/source metadata so origin is preserved. |
| `request_author` | Actor who submitted or originated the intake. |
| `intent_owner` | Actor responsible for the goal intent; may default from intake and later be corrected by policy or clarification. |
| `state` | Goal lifecycle state in the intent plane. |
| `created_at` | Server timestamp for Goal creation. |

`summary` should remain intentionally lighter than a contract. It may preserve
raw intake text until a later composer or clarification boundary exists, but it
must not encode scope approval, implementation plan, task decomposition, or gate
expectations.

`source_refs` preserve source-neutral traceability. They should be enough to show
where the Goal came from without making the original tracker, CLI, skill, or
integration the canonical owner of Goal truth.

## Proposed states

The Goal state machine starts minimal:

- `created`
- `needs_clarification`
- `ready_for_contract_seed`
- `rejected`

State meanings:

- `created`: normalized goal exists, but may still need clarification before a
  contract seed can be prepared.
- `needs_clarification`: the goal lacks required information for contract seed
  generation.
- `ready_for_contract_seed`: enough information exists to generate a
  `ContractDraft` in a later slice.
- `rejected`: the intake should not proceed as a Goalrail delivery item.

These are intent-plane states only. They are not execution states, task states,
run states, approval states, or proof states.

For the first implementation slice, it is acceptable to create only `created`
Goals while keeping the broader state names reserved by this ADR.

## Proposed events

Goal promotion should write at least:

- `goal.created`
- `intake.promoted_to_goal`

`goal.created` records the canonical Goal object that was created.

`intake.promoted_to_goal` records the transition from raw intake to normalized
intent. It must not be interpreted as contract creation, work item creation,
approval, or execution authorization.

Future events may include:

- `goal.marked_needs_clarification`
- `goal.marked_ready_for_contract_seed`
- `goal.rejected`

## Validation and promotion rules

Minimum promotion rules:

- `IntakeRecord` must exist.
- `IntakeRecord.state` must be `received`.
- `IntakeRecord.repo_binding_id` must be present.
- `IntakeRecord` must have title or body.
- `request_author` must be present.
- Promotion must create at most a `Goal` plus promotion events.
- Promotion must not create `ContractDraft`, `ApprovedContract`, `WorkItem`,
  `Task`, `GateDecision`, or `Proof`.

If `intent_owner` was defaulted from `request_author` during intake, that remains
an intake default only. Later policy or clarification may correct the owner, but
promotion itself does not define approval policy.

## Rejected alternatives

### Raw intake becomes Contract directly

Rejected because raw intake can be vague, incomplete, or source-shaped. Goalrail
requires a normalized intent boundary before contract composition.

### Raw intake becomes WorkItem directly

Rejected because work items are executable units derived from an approved or
contract-shaped delivery scope. Raw intake is not executable work.

### Goal promotion creates executable tasks

Rejected because task planning belongs after contract shaping. Creating tasks at
promotion time would collapse intent, contract, and execution boundaries.

### Local CLI or skill owns Goal truth

Rejected because CLI, skills, web resources, and integrations are adapters. The
server owns canonical Goalrail state.

### Goal state becomes a ticket/status workflow

Rejected because Goal state is an intent-plane lifecycle, not a replacement for
Jira, Linear, or a team-specific ticket workflow.

## Non-goals

This ADR does not define or implement:

- clarification engine
- contract composer
- approval policy
- task or work item planner
- gate or proof
- durable storage
- LLM enrichment
- dedupe
- tracker or CLI integrations
- endpoint shape as a final API contract
- database schema or migrations

## Implementation implications

A later implementation slice may add:

- `spine.Goal`
- `goal.Service`
- in-memory `GoalStore`
- `goal.created` event
- `intake.promoted_to_goal` event
- a small promotion endpoint such as `POST /v1/intake/{id}/promote` or
  `POST /v1/goals`

That implementation slice must remain bounded to promotion. It must not add
clarification, contract draft generation, work item planning, durable storage,
gate/proof, CLI integration, or frontend changes unless a separate ADR or bounded
slice explicitly authorizes them.

## Open questions

1. Should the first promotion endpoint be `POST /v1/intake/{id}/promote` or
   `POST /v1/goals`?
2. Should `Goal.summary` initially be a rule-free copy from intake text, or a
   small deterministic normalization before a composer exists?
3. Should rejection happen at the intake stage, at the goal stage, or both with
   distinct semantics?
