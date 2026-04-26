# Goalrail Server

This server is still an early prototype. Existing intake, Goal readiness,
ContractSeed creation, ContractDraft creation, and event log flows use Postgres
when `GOALRAIL_DATABASE_DSN` is configured. Clarification request and answer
state remain in-memory.

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
curl -sS http://localhost:8080/v1/intake \
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
curl -sS -X POST http://localhost:8080/v1/intake/{intake_id}/promote
curl -sS -X POST http://localhost:8080/v1/goals/{goal_id}/readiness
```

With Postgres configured, `IntakeRecord`, `Goal`, `ContractSeed`,
`ContractDraft`, and their events are durable and survive server restarts.
Project/RepoBinding validation uses Postgres to derive `organization_id` from
the seeded context. Intake creation, Goal promotion, Goal readiness,
ContractSeed creation, and ContractDraft creation writes share a transaction
with their expected event appends.

After clarification answers are applied and an explicit readiness re-check marks
the Goal `ready_for_contract_seed`, create a seed snapshot:

```bash
curl -sS -X POST http://localhost:8080/v1/goals/{goal_id}/contract-seed
```

Then create a draft from the seed:

```bash
curl -sS -X POST http://localhost:8080/v1/contract-seeds/{contract_seed_id}/contract-draft
```

This flow still does not create executable work, approved Contract, gate
decisions, proof, runner jobs, or VCS integration. Clarification request and
answer state is still prototype/in-memory. ContractSeed creation does not
create `ContractDraft`, `WorkItem`, approved Contract, `GateDecision`, `Proof`,
or executable work. ContractDraft creation does not approve Contract, create
`WorkItem`, write `GateDecision`, or create `Proof`.
