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
