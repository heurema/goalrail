# ADR-0010 — Organization, project, repo binding, and persistence bootstrap boundary

Status: accepted
Date: 2026-04-26

## Context

Goalrail now has a Go server bootstrap with in-memory source-neutral intake,
Goal promotion, Goal readiness, and ClarificationRequest prototypes. The
ClarificationAnswer recording boundary is documented separately. The runner
boundary is also documented separately in ADR-0008: the API server owns
canonical state, while repository checkout and check execution belong behind
runners.

The next MVP need is a durable SaaS-shaped project context before real VCS
integration, runner implementation, contract composition, gate, or proof. We
need a small `Organization -> Project -> RepoBinding` foundation so a Goalrail
Project can say which repository it works with without making the API server
clone code or making GitHub/GitLab/Bitbucket integration a prerequisite.

This stage intentionally avoids full SaaS onboarding, production auth,
invitations, UI setup flows, full CRUD endpoints, GitHub App installation, and
runner jobs. The boundary exists to define the server MVP foundation without
turning Goalrail into a hosted DevOps suite, generic workflow engine, AI IDE, or
universal agent platform.

## Decision

The MVP uses a simplified entity model:

- `User`
- `Organization`
- `OrganizationMembership`
- `Project`
- `RepoBinding`

The MVP does not require:

- `RepositoryRecord`
- `RepositoryEnrollment`
- `RepositoryPolicy`

For the MVP, `RepoBinding` stores the repository reference directly. A separate
`RepositoryRecord` may be extracted later if Goalrail needs a repository
catalog, multi-project repository reuse, repo-level policy, or independent
provider sync.

`VcsConnection` remains a future provider connection layer for GitHub App
installations, GitLab OAuth/token flows, Bitbucket OAuth/token flows, and
self-managed/custom Git support. It is not required for the first code slice.
Manual `RepoBinding` is enough before GitHub integration. Customer-hosted runner
paths must remain possible without Goalrail cloud having VCS provider access.

Server persistence uses:

- `pgx/v5` for PostgreSQL execution, connection pool, and transactions
- `Masterminds/squirrel` for runtime SQL statement construction in Go code
- `goose` for migrations

Goalrail will not use `sqlc` or an ORM for this server persistence foundation.

Dev seed data writes to the database and stays separate from migrations.

## MVP object model

### User

A Goalrail user. For early development flow, a deterministic seed user is
enough. This ADR does not define production auth.

### Organization

A Goalrail `Organization` is an internal SaaS tenant/workspace.

Goalrail `Organization` is not:

- a GitHub Organization
- a GitLab Group
- a Bitbucket Workspace

### OrganizationMembership

Connects a `User` to an `Organization`.

Initial roles:

- `owner`
- `admin`
- `member`
- `viewer`

For the first dev seed, create one owner membership.

### Project

A delivery container inside an `Organization`.

`Project` is not a repository. It is the Goalrail work contour for goals,
contracts, tasks, runs, proof, and related operating state.

### RepoBinding

A project-to-repository reference. For the MVP, `RepoBinding` stores repository
metadata directly.

Conceptual fields:

- `id`
- `organization_id`
- `project_id`
- optional `vcs_connection_id`
- `provider`
- optional `repository_external_id`
- `repository_full_name`
- `repository_url`
- `default_branch`
- `path_scope`
- `access_mode`
- `state`
- timestamps

`RepoBinding` identifies which repository a Project works with. It is not
permission to clone. Checkout authority is determined later by runner mode,
policy, access mode, and bounded checkout instructions.

Future-compatible `access_mode` values:

- `provider_token_checkout`
- `customer_runner_checkout`
- `customer_mounted_workspace`
- `metadata_only`

This ADR documents the values only. It does not implement runtime behavior.

## Relations

Conceptual ownership chain:

```text
User -> OrganizationMembership -> Organization -> Project -> RepoBinding
```

Rules:

- `User` gains organization context through `OrganizationMembership`.
- `Organization` owns Projects.
- `Project` owns RepoBindings.
- `RepoBinding` belongs to exactly one Project.
- For the MVP, one Project should have at most one active primary
  `RepoBinding`, even if the model can later support more.

## Persistence policy

`pgx` executes SQL and manages connection pools and transactions.

Squirrel builds runtime SQL statements in Go code. Squirrel is not an executor.
Execution goes through `pgx`.

Runtime SQL in Go code should be built through Squirrel consistently rather than
mixing ad hoc handwritten runtime SQL with Squirrel builders.

`goose` handles migrations. Migrations are raw SQL files because goose
migrations are schema DDL.

Goalrail will not use `sqlc` or an ORM for this persistence foundation.

## Migration policy

Before production, one editable init migration is allowed.

Schema changes may edit that init migration. When the init migration changes,
developers should reset/recreate the dev database.

After production, migrations become forward-only.

## Dev seed policy

Seed data goes into PostgreSQL.

Seed is separate from migrations:

- migration = schema
- seed = dev data

Seed must be:

- dev-only
- idempotent
- safe to re-run
- not production onboarding
- not auth

The first dev seed should create deterministic records:

- dev owner user `018f0000-0000-7000-8000-000000000001`
- dev organization `018f0000-0000-7000-8000-000000000002`
- owner membership
- dev project `018f0000-0000-7000-8000-000000000003`
- dev repo binding `018f0000-0000-7000-8000-000000000004`

Seed should be run through a later command or tool, for example
`goalrail-server seed dev` or a dedicated dev tool.

The seed does not create `IntakeRecord` or `Goal` by default. Intake and Goal
flow should be exercised through the API.

## Rejected alternatives

### Full CRUD endpoints first

Rejected. The first need is a durable state foundation, not a full onboarding or
admin API surface.

### UI onboarding first

Rejected. UI setup flows would imply more product surface than the server MVP
needs.

### In-memory seed as MVP foundation

Rejected. The next boundary is durable server state. Dev seed should exercise
the real database path.

### RepositoryRecord in MVP

Rejected. A repository catalog is useful later, but direct `RepoBinding` keeps
the first entity model small.

### sqlc

Rejected. The current persistence choice favors explicit runtime SQL builders
without generated query code.

### ORM

Rejected. The server should keep persistence native and explicit.

### Raw handwritten SQL mixed with Squirrel in runtime code

Rejected. Runtime SQL construction should be consistent. Goose migrations remain
raw SQL because they are schema DDL.

### GitHub App integration before project/repo binding foundation

Rejected. Goalrail needs the Project and RepoBinding contour before provider
integration. Manual/dev-seeded RepoBinding is enough for the next slice.

## Non-goals

This ADR does not define or implement:

- auth
- invitations
- billing
- UI
- full CRUD API
- GitHub/GitLab/Bitbucket integration
- `VcsConnection` implementation
- runner implementation
- checkout jobs
- contract/gate/proof
- production onboarding
- multi-tenant permission enforcement beyond conceptual ownership fields

## Implementation implications

Recommended first code slice:

```text
server: add postgres foundation and dev seed
```

In scope:

- pgx pool/config
- goose migration setup
- one editable init migration
- tables for users, organizations, organization_memberships, projects, and
  repo_bindings
- Squirrel-based stores
- idempotent dev seed
- basic persistence tests

Out of scope:

- auth
- UI
- GitHub
- GitLab
- runner
- gate/proof
- full CRUD endpoints

## Open questions

- Seed command shape: `goalrail-server seed dev` vs separate dev tool.
- First seed repo provider: github-looking vs `custom_git`.
- Whether `project_id` should be added to `IntakeRecord` in the same
  implementation slice or the next slice.
- Whether a debug endpoint for dev context is needed.
