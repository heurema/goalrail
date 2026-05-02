# Goalrail Server

This server is still an early prototype. Existing intake, Goal readiness,
Contract aggregate creation, ContractSeed creation, ContractDraft
creation/update, ContractDraft ready_for_approval, ApprovedContract approval,
and event log flows use Postgres when `GOALRAIL_DATABASE_DSN` is configured.
WorkItem planning, ClarificationRequest, and ClarificationAnswer state remain
in-memory prototypes.

## Local Postgres foundation

Configure Postgres with:

```bash
export GOALRAIL_DATABASE_DSN='postgres://goalrail:goalrail@localhost:5432/goalrail?sslmode=disable'
```

Apply the editable pre-production init migration:

```bash
go run ./cmd/goalrail-server migrate up
```

Apply the idempotent dev seed:

```bash
go run ./cmd/goalrail-server seed dev
```

The dev seed writes one deterministic UUIDv7 user, organization, owner
membership, project, and repo binding. It is not auth, onboarding, or
production data.

## Dev intake flow

After migration and dev seed:

```bash
go run ./cmd/goalrail-server
```

Submit intake with the seeded Project and RepoBinding context:

```bash
curl -sS http://localhost:8080/v1/intakes \
  -H 'Content-Type: application/json' \
  -d '{
    "project_id": "018f0000-0000-7000-8000-000000000003",
    "repo_binding_id": "018f0000-0000-7000-8000-000000000004",
    "source": {"kind": "manual"},
    "title": "Improve billing error handling",
    "body": "We need clearer error behavior around failed invoice payment.",
    "request_author": {"kind": "user", "id": "018f0000-0000-7000-8000-000000000001"}
  }'
```

Then promote and check readiness:

```bash
curl -sS -X POST http://localhost:8080/v1/intakes/{intake_id}/goals
curl -sS -X POST http://localhost:8080/v1/goals/{goal_id}/readiness
```

With Postgres configured, `IntakeRecord`, `Goal`, the public `Contract`
aggregate, `ContractSeed`, `ContractDraft`, `ApprovedContract`, and their
events are durable and survive server restarts.
WorkItems are currently planned through an in-memory prototype store; no
`work_items` table or migration exists in this slice.
Project/RepoBinding validation uses Postgres to derive `organization_id` from
the seeded context. Intake creation, Goal promotion, Goal readiness,
ContractSeed creation, ContractDraft creation/update, ContractDraft
ready_for_approval writes, and ApprovedContract approval writes share a
transaction with their expected event appends. The stable `contract_id` is
returned by ContractSeed, ContractDraft, and ApprovedContract responses.

After clarification answers are applied and an explicit readiness re-check marks
the Goal `ready_for_contract_seed`, create a seed snapshot:

```bash
curl -sS -X POST http://localhost:8080/v1/goals/{goal_id}/contract-seeds
```

Then create a draft from the seed:

```bash
curl -sS -X POST http://localhost:8080/v1/contract-seeds/{contract_seed_id}/contract-drafts
```

Then update proposed draft fields explicitly:

```bash
curl -sS -X PATCH http://localhost:8080/v1/contract-drafts/{contract_draft_id} \
  -H 'Content-Type: application/json' \
  -d '{
    "updated_by": {"kind": "user", "id": "018f0000-0000-7000-8000-000000000001"},
    "changes": {
      "proposed_scope": ["Reviewed proposed scope"],
      "proposed_acceptance_criteria": ["Reviewed proposed acceptance criteria"]
    }
  }'
```

Then mark a complete draft ready for approval:

```bash
curl -sS -X POST http://localhost:8080/v1/contract-drafts/{contract_draft_id}/submissions \
  -H 'Content-Type: application/json' \
  -d '{
    "marked_by": {"kind": "user", "id": "018f0000-0000-7000-8000-000000000001"}
  }'
```

Then approve the ready draft into an approved contract snapshot:

```bash
curl -sS -X POST http://localhost:8080/v1/contract-drafts/{contract_draft_id}/approvals \
  -H 'Content-Type: application/json' \
  -d '{
    "approved_by": {"kind": "user", "id": "018f0000-0000-7000-8000-000000000001"}
  }'
```

Then plan one non-executable WorkItem from the approved Contract using the
stable public `contract_id` returned earlier:

```bash
curl -sS -X POST http://localhost:8080/v1/contracts/{contract_id}/tasks
```

This route resolves `{contract_id}` through the public Contract aggregate,
requires the Contract to be `approved`, and then uses the internal immutable
ApprovedContract snapshot to create the simple v0 planned WorkItem. The server
does not expose `/v1/contracts` lifecycle façade routes yet.

This flow still does not create executable work, gate decisions, proof, runner
jobs, or VCS integration. Clarification request and
answer state is still prototype/in-memory. ContractSeed creation does not
create `ContractDraft`, `WorkItem`, approved Contract, `GateDecision`, `Proof`,
or executable work. ContractDraft creation does not approve Contract, create
`WorkItem`, write `GateDecision`, or create `Proof`. ContractDraft updates
modify proposed fields only, keep `ContractDraft.state` as `draft`, and treat
`updated_by` as audit identity only. The ready_for_approval transition changes
only `ContractDraft.state` from `draft` to `ready_for_approval`, requires
`marked_by` as audit identity, runs completeness checks, and does not approve
Contract, create `WorkItem`, write `GateDecision`, or create `Proof`.
Approval creates an immutable `ApprovedContract(approved)` snapshot from a ready
draft, requires `approved_by`, appends `contract.approved`, and does not mutate
`ContractDraft` or create execution, `GateDecision`, or `Proof`. WorkItem
planning creates exactly one in-memory `WorkItem(planned)` per approved
contract in v0, appends `work_item.created`, guards repeated planning with
`409 already_planned`, and does not assign, claim, create `Run`, checkout a
repository, submit a receipt, write `GateDecision`, or create `Proof`.
