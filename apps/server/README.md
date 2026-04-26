# Goalrail Server

This server is still an early prototype. Existing intake, Goal readiness,
clarification, answer application, and ContractSeed flows remain in-memory.

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

After clarification answers are applied and an explicit readiness re-check marks
the Goal `ready_for_contract_seed`, create a seed snapshot:

```bash
curl -sS -X POST http://localhost:8080/v1/goals/{goal_id}/contract-seed
```

Intake and Goal state remain in-memory prototypes. Project/RepoBinding
validation uses Postgres only to derive `organization_id` from the seeded
context. ContractSeed creation does not create `ContractDraft`, `WorkItem`,
approved Contract, `GateDecision`, `Proof`, or executable work.
