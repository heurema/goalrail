# Goalrail Server

This server is still an early prototype. Existing intake, Goal readiness, and
clarification flows remain in-memory.

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

The dev seed writes one deterministic user, organization, owner membership,
project, and repo binding. It is not auth, onboarding, or production data.

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
    "project_id": "prj_dev_default",
    "repo_binding_id": "rpb_dev_default",
    "source": {"kind": "manual"},
    "title": "Improve billing error handling",
    "body": "We need clearer error behavior around failed invoice payment.",
    "request_author": {"kind": "user", "id": "usr_dev_owner"}
  }'
```

Then promote and check readiness:

```bash
curl -sS -X POST http://localhost:8080/v1/intake/{intake_id}/promote
curl -sS -X POST http://localhost:8080/v1/goals/{goal_id}/readiness
```

Intake and Goal state remain in-memory prototypes. Project/RepoBinding
validation uses Postgres only to derive `organization_id` from the seeded
context, and intake still does not create executable work.
