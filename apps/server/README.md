# Goalrail Server

This server is still an early prototype. Existing intake, Goal readiness,
public Contract lifecycle, ContractSeed creation, ContractDraft
creation/update, ContractDraft ready_for_approval, ApprovedContract approval,
WorkItem planning, and event log flows use Postgres when
`GOALRAIL_DATABASE_DSN` is configured. ClarificationRequest and
ClarificationAnswer state remain in-memory prototypes.

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
events are durable and survive server restarts. Planned WorkItems are also
durable when Postgres is configured.
Project/RepoBinding validation uses Postgres to derive `organization_id` from
the seeded context. Intake creation, Goal promotion, Goal readiness,
ContractSeed creation, ContractDraft creation/update, ContractDraft
ready_for_approval writes, and ApprovedContract approval writes share a
transaction with their expected event appends. The stable `contract_id` is
returned by ContractSeed, ContractDraft, and ApprovedContract responses.

After clarification answers are applied and an explicit readiness re-check marks
the Goal `ready_for_contract_seed`, create the public Contract lifecycle
aggregate. This creates the internal `ContractSeed` and `ContractDraft` records
and returns a public Contract view in `draft` state:

```bash
curl -sS -X POST http://localhost:8080/v1/contracts \
  -H 'Content-Type: application/json' \
  -d '{
    "goal_id": "{goal_id}"
  }'
```

Then update proposed draft fields explicitly:

```bash
curl -sS -X PATCH http://localhost:8080/v1/contracts/{contract_id} \
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
curl -sS -X POST http://localhost:8080/v1/contracts/{contract_id}/submissions \
  -H 'Content-Type: application/json' \
  -d '{
    "marked_by": {"kind": "user", "id": "018f0000-0000-7000-8000-000000000001"}
  }'
```

Then approve the ready draft into an approved contract snapshot:

```bash
curl -sS -X POST http://localhost:8080/v1/contracts/{contract_id}/approvals \
  -H 'Content-Type: application/json' \
  -d '{
    "approved_by": {"kind": "user", "id": "018f0000-0000-7000-8000-000000000001"}
  }'
```

Then create a server-owned planning request for the approved Contract using the
same stable public `contract_id`:

```bash
curl -sS -X POST http://localhost:8080/v1/contracts/{contract_id}/plans \
  -H 'Content-Type: application/json' \
  -d '{
    "requested_by": {"kind": "user", "id": "018f0000-0000-7000-8000-000000000001"}
  }'
```

For now the future worker/planner output can be submitted manually through the
API as a Proposal. The server validates and stores the Proposal but does not
create canonical WorkItems yet:

```bash
curl -sS -X POST http://localhost:8080/v1/plans/{plan_id}/proposals \
  -H 'Content-Type: application/json' \
  -d '{
    "submitted_by": {"kind": "worker", "id": "planner-worker-1"},
    "planner": {"kind": "goalrail_worker", "id": "planner-worker-1", "version": "0.1.0"},
    "source_snapshot_refs": [{"kind": "approved_contract", "id": "{approved_contract_id}"}],
    "rationale": "Why this task decomposition was proposed.",
    "proposed_tasks": [{
      "title": "Refactor CSV export filter builder",
      "summary": "Extract duplicated filter construction logic.",
      "scope": ["Update export filter construction code"],
      "acceptance_refs": ["acceptance_criteria[0]"],
      "proof_expectation_refs": ["proof_expectations[0]"],
      "order_index": 0,
      "source_refs": [{"kind": "approved_contract", "id": "{approved_contract_id}"}]
    }]
  }'
```

Explicitly accept the Proposal to materialize canonical durable
`WorkItem(planned)` records:

```bash
curl -sS -X POST http://localhost:8080/v1/proposals/{proposal_id}/acceptance \
  -H 'Content-Type: application/json' \
  -d '{
    "accepted_by": {"kind": "user", "id": "018f0000-0000-7000-8000-000000000001"}
  }'
```

Read the planned task by its stable task ID:

```bash
curl -sS http://localhost:8080/v1/tasks/{task_id}
```

There is no task list/search endpoint in this slice.

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
Approval creates an immutable `ApprovedContract(approved)` snapshot from a
ready draft, requires `approved_by`, appends `contract.approved`, and does not
mutate `ContractDraft` or create execution, `GateDecision`, or `Proof`.
Planning now uses `Plan -> Proposal -> Acceptance`: one plan per approved
Contract in v0, one proposal per plan in v0, and accepted proposals may create
one or more `WorkItem(planned)` records. Acceptance appends `work_item.created`
for each task and persists the plan/proposal/tasks when Postgres is configured.
Workers/planners do not write WorkItems directly to the DB. This flow does not
assign, claim, create `Run`, checkout a repository, submit a receipt, write
`GateDecision`, or create `Proof`.
