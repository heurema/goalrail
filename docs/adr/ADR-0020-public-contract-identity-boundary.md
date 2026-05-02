# ADR-0020 — Public Contract identity boundary

Status: accepted
Date: 2026-05-02

## Context

Goalrail's product language treats the working contract as the central object
between intent and delivery.

The current server implementation reached that lifecycle through precise
internal records:

- `ContractSeed`
- `ContractDraft`
- `ApprovedContract`

Those records are useful implementation and audit boundaries:

- `ContractSeed` is the bridge from a readiness-checked Goal.
- `ContractDraft` is mutable proposed contract material before approval.
- `ApprovedContract` is the immutable approved snapshot after approval.

PR #49 also shortens public route vocabulary toward product-facing resources,
including `POST /v1/contracts/{id}/tasks`. That spelling is the right public
direction, but the current server implementation still resolves that route
against existing approved contract state internally.

Goalrail needs an explicit public identity boundary so future route and storage
work does not expose every internal lifecycle record as a top-level public
resource.

## Decision

Public API uses one stable `Contract` identity. `ContractSeed`,
`ContractDraft`, and `ApprovedContract` are internal lifecycle records.

`Contract` is the public lifecycle aggregate / envelope for the working
contract. It gives public and control-plane API users one stable `contract_id`
from contract creation through draft review, ready-for-approval, approval, and
later task planning.

Internal lifecycle records remain precise and auditable:

- `ContractSeed` remains the explicit bridge from `Goal(ready_for_contract_seed)`.
- `ContractDraft` remains the mutable proposed terms record before approval.
- `ApprovedContract` remains the immutable approved snapshot.

Future public API should avoid exposing `contract-seeds`, `contract-drafts`, and
`approved-contracts` as product-facing resource names. It should expose
`/v1/contracts/{id}` and subresources around the stable public contract
identity.

The current implementation may keep transitional routes and internal IDs until a
bounded implementation slice creates a public `Contract` aggregate. This ADR
defines direction; it does not claim that aggregate exists today.

## Target public API shape

This target shape is not implemented by this ADR:

| Route | Meaning |
| --- | --- |
| `POST /v1/contracts` | Creates a public Contract aggregate from `goal_id` and creates initial internal seed/draft records as needed by implementation. |
| `GET /v1/contracts/{id}` | Reads the public Contract aggregate / lifecycle view. |
| `PATCH /v1/contracts/{id}` | Updates draft proposed fields only while the Contract is in a draft-compatible state. |
| `POST /v1/contracts/{id}/submissions` | Marks the current draft ready for approval. |
| `POST /v1/contracts/{id}/approvals` | Approves the current ready draft and creates an immutable `ApprovedContract` snapshot internally. |
| `POST /v1/contracts/{id}/tasks` | Simple v0 direct task planning from the approved contract state; later richer planning uses plan/proposal boundaries. |

Planning request/proposal routes, when later implemented, should use:

- `POST /v1/contracts/{id}/plans`
- `GET /v1/plans/{id}`
- `POST /v1/plans/{id}/proposals`
- `GET /v1/proposals/{id}`
- `POST /v1/proposals/{id}/acceptance`

Future public REST paths should not use:

- `/contract-seeds`
- `/contract-drafts`
- `/approved-contracts`
- `/work-items`
- `/work-item-planning-requests`
- `/work-item-plan-proposals`

## Conceptual model

### Public aggregate: `Contract`

Conceptual fields:

- `id`
- `organization_id`
- `project_id`
- `repo_binding_id`
- `goal_id`
- `state`
- `current_seed_id` optional internal reference
- `current_draft_id` optional internal reference
- `approved_snapshot_id` optional internal reference
- `created_at`
- `updated_at`

Initial public states:

- `draft`
- `ready_for_approval`
- `approved`

Additional states require later ADRs.

### Internal lifecycle records

Internal records remain:

- `ContractSeed`
- `ContractDraft`
- `ApprovedContract`

These records should eventually reference public `contract_id`.

The public `Contract.state` reflects lifecycle state. After approval, the
Contract aggregate may point to the immutable `ApprovedContract` snapshot
through `approved_snapshot_id`.

## Relationship to existing ADRs

ADR-0013 through ADR-0017 remain valid as internal lifecycle boundaries.

ADR-0018 and ADR-0019 remain valid for non-executable task planning and the
controller / runner planning split. Future task and planning routes should
attach to the public `Contract` identity, while canonical WorkItems remain
API-server-owned after validation / acceptance.

This ADR qualifies endpoint vocabulary and public identity. It does not rewrite
the historical meaning of the earlier lifecycle records.

## Consequences

### Positive

- Public API is product-facing instead of a direct dump of Go / DB type names.
- Public callers can use one stable `contract_id` across the lifecycle.
- Internal lifecycle records can stay strict, auditable, and implementation
  precise.
- Future plans, proposals, and tasks attach to `contracts/{id}` instead of
  `approved-contracts/{id}` or internal lifecycle IDs.

### Negative

- Initial implementation now introduces a minimal `Contract` aggregate and maps
  it to existing lifecycle records, but public lifecycle façade routes are still
  a later slice.
- Transitional routes and examples need to remain honest about any remaining
  internal lifecycle IDs until the public lifecycle façade exists.
- Some existing endpoint candidates in historical ADRs are now transitional
  implementation details, not final public API shape.

## Non-goals

This ADR did not originally define or implement:

- code changes
- migrations
- public `/v1/contracts` lifecycle façade routes
- endpoint implementation
- route compatibility aliases
- runner, worker, or controller implementation
- checkout
- execution
- queue or outbox
- runtime registry
- `Run`
- receipt submission
- `GateDecision`
- `Proof`
- plans or proposals

## Recommended next slice

The next backend contract slice should design and implement the smallest public
`Contract` aggregate boundary before further route expansion.

It should preserve the existing internal lifecycle records, avoid runner /
execution / gate / proof scope, and keep `ApprovedContract` as the immutable
approved snapshot.
