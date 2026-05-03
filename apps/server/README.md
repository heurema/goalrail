# Goalrail Server

This server is still an early prototype. Existing intake, Goal readiness,
public Contract lifecycle, ContractSeed creation, ContractDraft
creation/update, ContractDraft ready_for_approval, ApprovedContract approval,
WorkItem planning, and event log flows use Postgres when
`GOALRAIL_DATABASE_DSN` is configured. ClarificationRequest and
ClarificationAnswer state also use Postgres when configured, with in-memory
fallback when no database DSN is set.

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

The dev seed writes one deterministic UUIDv7 user, one `self_hosted`
Installation with `public_base_url` set to `http://localhost:8080`, one linked
organization, owner membership, project, and repo binding. It is not auth,
onboarding, or production data.

## Self-hosted owner bootstrap

After applying migrations, create the first self-hosted owner with explicit
flags:

```bash
go run ./cmd/goalrail-server bootstrap owner \
  --email owner@example.com \
  --display-name "Owner User" \
  --organization-slug acme \
  --organization-name "Acme" \
  --public-base-url https://goalrail.example.com
```

The command creates or reuses one `self_hosted` Installation, normalizes
`public_base_url` without a trailing slash, creates or reuses the primary
Organization, creates or reuses the matching User, ensures an
`OrganizationMembership(owner)`, and creates `user_password_credentials` with a
backend-generated temporary password and `must_change_password = true`.

The temporary password is printed to stdout only when a new password credential
is created:

```text
temporary_password=<generated-password>
```

Re-running the command for an owner that already has password credentials does
not rotate the password and prints:

```text
temporary_password_already_exists=true
```

## Auth API

The smallest server-only auth API is available after migrations and owner
bootstrap. Configure JWT signing with an operator-owned secret:

```bash
export GOALRAIL_AUTH_JWT_SECRET='<operator-managed-secret>'
```

Do not commit or auto-generate this secret in the repository. The server can
start without it, but auth endpoints return a clear auth configuration error
when signing or validating JWT access tokens without the secret. Configured
JWT secrets must be at least 32 characters after trimming.

Log in with the bootstrapped owner email and temporary password:

```bash
curl -sS http://localhost:8080/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "owner@example.com",
    "password": "temporary-password"
  }'
```

The response includes a short-lived bearer `access_token`, an opaque
`refresh_token` backed by `user_sessions`, and `must_change_password`.

Refresh the short-lived access token with the opaque refresh token:

```bash
curl -sS -X POST http://localhost:8080/v1/auth/refresh \
  -H 'Content-Type: application/json' \
  -d '{
    "refresh_token": "refresh-token"
  }'
```

The refresh endpoint hashes the supplied opaque token, looks up the
server-owned `user_sessions` row, rejects unknown, expired, revoked, or
inactive sessions, loads the current User and active OrganizationMembership
server-side, updates session `last_used_at`, and returns a new access token
only. It does not rotate the refresh token in this slice.

Change the temporary password with the bearer token:

```bash
curl -sS -X POST http://localhost:8080/v1/auth/change-password \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d '{
    "current_password": "temporary-password",
    "new_password": "new-password"
  }'
```

Read the current authenticated profile:

```bash
curl -sS http://localhost:8080/v1/me \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

`GET /v1/me` loads the current User and active OrganizationMembership
server-side. JWTs carry only identity/session claims, not broad role or
permission claims.

Log out the current bearer-token session:

```bash
curl -sS -X POST http://localhost:8080/v1/auth/logout \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
```

Logout validates the bearer access token, loads the referenced session, and
marks that session revoked with `revoked_at`.

There is still no CLI `goalrail login`, browser loopback, web UI, public
registration, admin user creation endpoint, SaaS onboarding, organization
creation API, password reset flow, email invite/reset delivery, refresh
token rotation, or broader session-management API in this slice.

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

With Postgres configured, `IntakeRecord`, `Goal`, `ClarificationRequest`,
`ClarificationAnswer`, the public `Contract` aggregate, `ContractSeed`,
`ContractDraft`, `ApprovedContract`, and their events are durable and survive
server restarts. Planned WorkItems are also durable when Postgres is configured.
Project/RepoBinding validation uses Postgres to derive `organization_id` from
the seeded context; the seeded Organization is linked to the seeded
Installation. Intake creation, Goal promotion, Goal readiness,
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
jobs, VCS integration, or automatic readiness re-check after answer application.
Clarification does not create contracts, plans, tasks, or work items. There is
no standalone public ContractSeed route; public
`POST /v1/contracts` composes the internal seed and draft records under one
stable `contract_id`. Standalone seed creation does not approve Contract, create
`WorkItem`, write `GateDecision`, create `Proof`, or create executable work.
ContractDraft creation does not approve Contract, create `WorkItem`, write
`GateDecision`, or create `Proof`. ContractDraft updates modify proposed fields
only, keep `ContractDraft.state` as `draft`, and treat `updated_by` as audit
identity only. The ready_for_approval transition changes only
`ContractDraft.state` from `draft` to `ready_for_approval`, requires `marked_by`
as audit identity, runs completeness checks, and does not approve Contract,
create `WorkItem`, write `GateDecision`, or create `Proof`.
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
