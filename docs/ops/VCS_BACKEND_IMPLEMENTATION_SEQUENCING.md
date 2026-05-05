---
id: goalrail_vcs_backend_implementation_sequencing
title: VCS Backend Implementation Sequencing
kind: ops_plan
authority: operational
status: current
owner: architecture
truth_surfaces:
  - vcs_backend_sequence
  - repository_connection_backend_order
  - provider_credential_blockers
lifecycle: active-core
review_after: 2026-06-05
supersedes: []
superseded_by: null
related_docs:
  - docs/INDEX.md
  - docs/product/GOALRAIL_PRODUCT_CONCEPT.md
  - docs/product/GOALRAIL_MVP_BLUEPRINT.md
  - docs/product/GOALRAIL_PROVIDER_BOUNDARIES.md
  - docs/product/GOALRAIL_REPOSITORY_CONNECTION_UX.md
  - docs/research/GOALRAIL_GITLAB_VCS_CONNECTION_RESEARCH.md
  - docs/PROJECT_SPINE_SCHEMA.md
  - docs/adr/ADR-0008-runner-checkout-boundary.md
  - docs/adr/ADR-0010-organization-project-repo-binding-persistence-boundary.md
  - docs/adr/ADR-0022-installation-boundary.md
  - docs/adr/ADR-0023-user-bootstrap-auth-and-cli-login-boundary.md
  - docs/adr/ADR-0024-provider-neutral-vcs-connection-boundary.md
  - docs/ops/STATUS.md
  - docs/ops/NEXT.md
  - docs/ops/DECISIONS.md
  - docs/ops/COMPONENTS.yaml
---
# VCS Backend Implementation Sequencing

## Purpose

This document defines the safe backend implementation order for provider-neutral
VCS / repository connection work after the accepted prerequisite docs:

- `docs/adr/ADR-0024-provider-neutral-vcs-connection-boundary.md`
- `docs/research/GOALRAIL_GITLAB_VCS_CONNECTION_RESEARCH.md`
- `docs/product/GOALRAIL_REPOSITORY_CONNECTION_UX.md`

It is an operational sequencing plan only. It does not implement schema, API
routes, OAuth, provider token storage, GitLab client code, repository metadata
listing, checkout, runner, gate, or proof behavior.

## Prerequisite confirmation

The prerequisite docs are present in the current indexed doc set:

- ADR-0024 is accepted architecture canon for the provider-neutral
  `VcsConnection` boundary.
- GitLab VCS connection research is advisory input only and does not authorize
  implementation.
- Repository connection UX is canonical product-shape guidance for future
  Settings -> Integrations -> Repositories behavior.

Implementation must follow canon in this order: product docs first, architecture
ADR support second, ops sequencing third, research as advisory input only.

## Current Backend Inventory

### Existing Auth / Login Foundation

Current backend auth is enough to authenticate Goalrail users and resolve their
server-side Organization context, but it is not a provider credential boundary.

Existing pieces:

- `user_password_credentials` keeps password material separate from `users`.
- `user_sessions` stores opaque refresh-token session state.
- `cli_auth_codes` stores short-lived, hashed CLI auth codes with S256 verifier
  challenge metadata.
- `apps/server/internal/auth` implements email/password login, access-token
  signing, refresh without refresh-token rotation, password change, bearer-token
  logout, `/v1/me`, and browser-loopback CLI login code exchange.
- `apps/web/console` consumes existing auth endpoints with access and refresh
  tokens in React memory only.

Not present:

- provider OAuth grant model
- provider token encryption / redaction / refresh / revocation / audit model
- provider credential store
- provider account consent state
- provider instance identity model
- keychain integration
- Organization / Project / RepoBinding profile selection in CLI or console

### Existing Installation / Organization / Project / RepoBinding Persistence

The current server has a real Postgres-backed project context foundation:

- `installations`
- `organizations`
- `organization_memberships`
- `projects`
- `repo_bindings`
- `repository_context_snapshots`

This foundation follows ADR-0010 and ADR-0022:

- `Installation` is the running Goalrail control plane.
- Goalrail `Organization` is an internal tenant/workspace, not a provider
  group, organization, workspace, namespace, owner, or account.
- Goalrail `Project` is a delivery container, not a repository.
- `RepoBinding` identifies which repository a Goalrail Project works with.

### Existing Authenticated Repository-Context Init

`POST /v1/init/repository-context` exists as an authenticated repository-context
bootstrap path.

Current behavior:

- resolves the authenticated user's active Goalrail Organization from
  server-side membership state
- creates or reuses one repo-backed Goalrail Project for the repository inside
  that Organization
- creates or reuses a metadata-only RepoBinding for the repository
- records only repository identity and local metadata context
- does not accept `organization_id` from the request body
- does not map provider owners/groups/workspaces to Goalrail Organization

This endpoint is not provider connection, OAuth, provider account sync,
repository picker, repository scan, checkout, runner, gate, or proof.

### Existing Metadata-Only RepoBinding Init

`POST /v1/projects/{project_id}/repo-bindings/init` exists as the low-level
authenticated Project-scoped RepoBinding init path.

Current behavior:

- derives Organization from Project
- initializes one active metadata-only RepoBinding for the Project
- uses `metadata_only` access mode
- writes `repo_binding.initialized` event on create
- blocks a different active repository binding for the same Project
- blocks the same active repository binding from being duplicated in another
  Project inside the same Organization

This endpoint does not grant checkout/read/write access and does not create a
provider connection.

### Existing `repo_bindings.vcs_connection_id` Placeholder

`repo_bindings.vcs_connection_id` exists as a text placeholder with an empty
default. Server DTOs and the Postgres store carry it through as a string.

This is not a real `VcsConnection` implementation. It has no type, table,
foreign key, lifecycle state, API, provider account identity, credential
reference, or provider metadata refresh behavior.

### Explicitly Missing Backend Pieces

The backend does not currently have:

- real `VcsConnection` type, table, store, service, or API
- `RepositoryRecord`
- provider credential or token storage
- token encryption, redaction, refresh, revocation, or audit
- provider OAuth routes or callback handling
- GitLab OAuth application configuration
- GitLab provider client
- GitHub, Bitbucket, self-managed, or custom provider client
- provider-neutral repository metadata listing/search API
- provider repository candidate persistence
- provider sync job
- repository metadata refresh worker
- checkout eligibility API
- checkout credentials
- repository clone or Repository Files API use
- runner, checkout job, checkout receipt, gate, or proof implementation for
  this boundary

## Non-Negotiable Boundaries

These boundaries apply to every backend phase below.

1. `VcsConnection` is not a checkout credential.
2. Raw provider tokens are secrets, not domain objects.
3. `RepoBinding` is not permission to clone, read files, write branches, create
   commits, open merge requests, open pull requests, run checks, gate, or proof.
4. `RepositoryRecord` remains deferred unless explicit trigger conditions are
   met: repository catalog/search independent of Project binding, provider sync
   independent of a Project, multi-Project repository reuse semantics, repo-level
   policy, or repository lifecycle tracking.
5. Goalrail `Organization` is not GitLab Group, GitHub Organization, Bitbucket
   Workspace, provider account, namespace, owner, or installation.
6. Goalrail `Project` is not a GitLab Project, GitHub repository, Bitbucket
   repository, or provider project.
7. The API server must not clone repositories or run checks in-process.
8. GitLab `read_api` risk must be addressed before implementation because
   advisory research found that it can also allow Repository Files API reads.
9. Provider tokens must not be stored in browser storage, repo files, local
   project markers, logs, docs, generated fixtures, or unredacted test output.
10. GitLab is a first provider candidate only. GitLab terms must stay in a
    provider adapter or adapter-owned metadata, not Goalrail core.
11. GitLab metadata discovery must not call the Repository Files API.
12. Clone URLs are metadata only until a separate runner checkout boundary
    issues bounded checkout instructions.
13. Checkout eligibility is a later policy/runner decision, not a property of
    `VcsConnection`, `RepositoryRecord`, or `RepoBinding`.
14. Customer-hosted runner compatibility must remain preserved. Goalrail must
    still work when repository credentials and contents stay in customer
    infrastructure.

## Backend Implementation Sequence

### Phase D1 — Provider Credential / Token Storage Boundary ADR

Goal:
- Define the provider credential storage and security boundary before any
  provider OAuth, token storage, or live provider metadata implementation.

Scope:
- Docs-only ADR or security decision for provider OAuth grants, access tokens,
  refresh tokens, encryption, key ownership, redaction, logging, audit events,
  rotation, revocation, deletion, retention, operator recovery, and testing.
- Explicit treatment of GitLab `read_api` residual risk.
- Explicit treatment of GitLab.com versus self-managed GitLab instance identity.
- Decision on whether credential storage is server-side, runtime-side,
  customer-hosted, or split by deployment mode.
- Decision on whether a credentialless `pending_setup` `VcsConnection` record is
  allowed before OAuth credentials exist.

Out of scope:
- schema migrations
- token tables
- OAuth routes
- provider clients
- live provider calls
- repository metadata APIs
- checkout / runner / gate / proof

Expected files:
- `docs/adr/ADR-0025-provider-credential-storage-boundary.md` or the next
  available ADR number
- `docs/ops/DECISIONS.md`
- `docs/ops/NEXT.md`
- `docs/ops/STATUS.md`, only if the operational state changes
- `docs/ops/COMPONENTS.yaml`, only to keep docs-only/planned status honest

Required validation:
- docs-check fixture self-test
- docs-check changed-files mode
- `scripts/check-staged.sh`

Required prerequisite docs/decisions:
- `docs/adr/ADR-0024-provider-neutral-vcs-connection-boundary.md`
- `docs/product/GOALRAIL_REPOSITORY_CONNECTION_UX.md`
- `docs/research/GOALRAIL_GITLAB_VCS_CONNECTION_RESEARCH.md`

Merge readiness criteria:
- Accepts a concrete credential storage boundary or explicitly blocks token
  persistence.
- Defines encryption/redaction/refresh/revocation/audit minimums.
- Defines what is safe to log and what must never be logged.
- States whether `VcsConnection` can be created without credentials.
- Keeps checkout credentials separate from provider OAuth credentials.

Scope creep:
- adding migrations, env vars, secrets, OAuth routes, provider clients,
  repository metadata listing, checkout, runner, queue, gate, or proof

### Phase D2 — Provider-Neutral VcsConnection Schema / API Skeleton, No Credentials

Goal:
- Add the smallest provider-neutral server model for `VcsConnection` only after
  D1 decides whether a credentialless connection skeleton is allowed.

Scope:
- Server-local type, store, migration, service, and authenticated API skeleton
  for provider-neutral connection state.
- No provider secrets.
- No provider OAuth.
- No live provider calls.
- No repository listing/search.
- No checkout eligibility.
- One Organization-scoped connection boundary inside an Installation.
- Provider-neutral fields only, such as `provider_kind`, provider instance
  identity, `provider_account_ref`, optional `provider_namespace_ref`,
  connection state, metadata scope summary, timestamps, and adapter metadata
  reference.

Out of scope:
- access tokens
- refresh tokens
- OAuth authorization URL generation
- OAuth callback
- token exchange
- GitLab client
- GitLab-specific core columns such as `gitlab_group_id` or
  `gitlab_project_id`
- repository candidate listing
- binding UI
- checkout / runner / gate / proof

Expected files:
- `apps/server/internal/spine/...`
- `apps/server/internal/store/...`
- `apps/server/internal/httpserver/...`
- `apps/server/internal/postgres/migrations/00001_init.sql`
- `docs/ops/COMPONENTS.yaml`
- `docs/ops/STATUS.md`
- tests next to touched server packages

Required validation:
- Go tests for new server packages
- migration tests
- docs-check changed-files mode
- `scripts/check-staged.sh`

Required prerequisite docs/decisions:
- D1 credential/token storage boundary accepted
- ADR-0024 accepted

Merge readiness criteria:
- `VcsConnection` is represented as provider-neutral state only.
- No raw provider token can be persisted through the new model.
- API responses do not imply live provider access unless backed by state.
- `RepoBinding` remains usable without `VcsConnection`.
- `repo_bindings.vcs_connection_id` placeholder is either still inert or linked
  only through an explicitly accepted safe relation.

Scope creep:
- adding OAuth, provider tokens, GitLab API calls, repository listing/search,
  RepositoryRecord, checkout eligibility, runner jobs, queue, gate, or proof

### Phase D3 — Provider-Neutral Repository Metadata Contract, No Provider Client

Goal:
- Define the provider-neutral contract for repository metadata candidates before
  implementing a provider adapter.

Scope:
- DTO/service/API contract or docs-backed API contract for repository candidate
  metadata.
- Fields enough for human repository selection: provider kind, provider
  instance identity, repository external id where available, full name, URL,
  default branch, visibility if safely returned, archived/unavailable state if
  safely returned, and metadata freshness.
- Explicit warning that clone URLs are metadata only.
- Fake/test fixtures only.
- No provider HTTP client.

Out of scope:
- GitLab OAuth
- GitLab live API calls
- Repository Files API
- source code, blobs, file trees, diffs, commits-as-content, or package data
- RepositoryRecord persistence unless a separate ADR says the trigger is met
- binding mutation from picker
- checkout / runner / gate / proof

Expected files:
- `apps/server/internal/spine/...`
- `apps/server/internal/httpserver/...` if an API contract is implemented
- `apps/server/internal/store/...` only if a non-persistent or bounded state
  contract requires it
- fixture files that contain metadata only and no secrets
- `docs/ops/COMPONENTS.yaml`
- `docs/ops/STATUS.md`

Required validation:
- unit tests proving repository metadata responses do not contain checkout
  credentials or source contents
- fixture scan for token-looking values if fixtures are added
- docs-check changed-files mode
- `scripts/check-staged.sh`

Required prerequisite docs/decisions:
- ADR-0024 accepted
- D1 accepted if the contract references connection credential state
- D2 accepted if the contract references persisted `VcsConnection`

Merge readiness criteria:
- Provider-neutral metadata contract exists without a provider client.
- Metadata state cannot be confused with checkout permission.
- RepositoryRecord remains deferred unless its trigger is explicitly met and
  documented.
- Clone URLs are marked metadata-only in responses or docs.

Scope creep:
- live provider calls, OAuth, token storage, provider sync workers,
  RepositoryRecord catalog, repository content reads, checkout, runner, queue,
  gate, or proof

### Phase D4 — GitLab Metadata Adapter Mapping Skeleton With Fake/Test Fixtures Only

Goal:
- Prove GitLab response mapping into provider-neutral repository metadata
  without live GitLab calls or credentials.

Scope:
- GitLab adapter package or internal mapping tests that translate fixture
  shapes into provider-neutral repository metadata.
- Metadata-only fields from GitLab Projects / Groups research.
- Explicit handling of GitLab.com versus self-managed instance identity in
  fixture inputs.
- Pagination and error mapping as pure logic if included.
- Strict endpoint allowlist documented in tests or adapter contract.

Out of scope:
- OAuth
- access tokens or refresh tokens
- live HTTP client
- Repository Files API calls
- Git-over-HTTP
- checkout credentials
- provider sync worker
- repository listing API backed by live GitLab
- binding UI
- checkout / runner / gate / proof

Expected files:
- future adapter path under `apps/server/internal/...` only after component
  ownership is documented
- metadata-only test fixtures
- tests for mapping, pagination metadata, and provider-neutral error mapping
- `docs/ops/COMPONENTS.yaml`
- `docs/ops/STATUS.md`

Required validation:
- tests prove fixture mapping does not expose source file contents
- tests prove clone URLs remain metadata fields only
- tests or docs prove Repository Files API is not in the metadata adapter
  allowlist
- Go tests
- docs-check changed-files mode
- `scripts/check-staged.sh`

Required prerequisite docs/decisions:
- D3 repository metadata contract accepted
- D1 accepted if any connection state or authorization posture is referenced
- GitLab research remains advisory; ADR-0024 wins on conflicts

Merge readiness criteria:
- GitLab-specific names are isolated to the adapter and fixtures.
- Core DTOs remain provider-neutral.
- No live network, env var, secret, token, or OAuth dependency exists.
- Metadata-only discovery excludes Repository Files API.

Scope creep:
- adding a GitLab HTTP client, OAuth routes, token storage, live provider
  calls, repository file/blob/tree reads, checkout, runner, queue, gate, proof,
  branch creation, commits, merge requests, or pull requests

### Phase D5 — GitLab OAuth / Token Implementation After Credential Boundary

Goal:
- Implement GitLab provider authorization only after D1 has accepted the
  provider credential/token boundary and earlier neutral contracts exist.

Scope:
- GitLab OAuth authorization code with PKCE unless a later ADR changes it.
- Instance-aware GitLab base URL handling for GitLab.com and self-managed
  instances.
- Server-side state and redirect validation.
- Token exchange, refresh, revocation, redaction, and audit exactly as D1
  permits.
- Strict provider API allowlist for metadata discovery.
- User-facing consent text that explains GitLab `read_api` can be broader than
  Goalrail metadata adapter behavior.

Out of scope:
- `read_repository`, `write_repository`, or `api` scopes unless a later ADR
  explicitly accepts them
- Repository Files API
- repository clone or Git-over-HTTP
- deriving checkout credentials from `VcsConnection`
- branch, commit, merge request, or pull request creation
- checkout / runner / gate / proof

Expected files:
- server provider/OAuth packages under `apps/server/internal/...`
- token store packages only where D1 assigns ownership
- migration updates only where D1 authorizes token persistence
- HTTP routes only for the accepted OAuth lifecycle
- tests for state, PKCE, token redaction, refresh/revocation, and API allowlist
- docs updates for public surface and component status

Required validation:
- Go tests
- OAuth state/PKCE tests
- token redaction tests
- refresh/revocation race tests where token refresh rotates credentials
- docs-check changed-files mode
- `scripts/check-staged.sh`

Required prerequisite docs/decisions:
- D1 credential/token storage boundary accepted
- D2 `VcsConnection` skeleton accepted
- D3 repository metadata contract accepted
- D4 GitLab mapping skeleton accepted, if GitLab is still the first provider

Merge readiness criteria:
- Provider tokens are encrypted/stored/redacted according to D1.
- Browser storage, repo files, local project markers, logs, docs, and fixtures
  contain no provider tokens.
- GitLab instance identity is explicit.
- `read_api` residual risk is documented in consent and internal allowlist
  behavior.
- Metadata discovery does not use Repository Files API.
- Checkout remains blocked and separate.

Scope creep:
- checkout credentials, repository clone, runner jobs, arbitrary provider API
  surface, repository write operations, generic queue, gate, or proof

### Phase D6 — Console Repository Settings UI Phases

Goal:
- Keep console repository UI honest while backend state is introduced.

Scope:
- D6a static placeholder may proceed before backend APIs if it only shows
  Settings -> Integrations -> Repositories, no-provider-connected state,
  disabled provider candidate, and required warning copy from
  `docs/product/GOALRAIL_REPOSITORY_CONNECTION_UX.md`.
- D6b API-backed provider connection status may proceed only after D2/D5 expose
  real server state.
- D6c repository metadata picker may proceed only after D3 plus a live provider
  adapter expose real server-returned metadata.
- D6d RepoBinding mutation through UI may proceed only after authorized
  server-backed binding APIs exist.

Out of scope:
- fake provider accounts
- fake repositories
- fake connection timestamps
- browser provider clients
- browser token storage
- localStorage/sessionStorage/cookies/IndexedDB for provider state
- scan, clone, checkout, runner, gate, proof, readiness, pass/fail, or branch
  analysis claims

Expected files:
- `apps/web/console/...` only in future frontend slices
- frontend tests
- docs updates if visible behavior changes

Required validation:
- frontend tests/build for console
- browser/storage audit where relevant
- required warning copy present
- docs-check changed-files mode
- `scripts/check-staged.sh`

Required prerequisite docs/decisions:
- Repository connection UX doc for D6a
- D2/D5 for live provider connection status
- D3 plus live adapter for metadata picker

Merge readiness criteria:
- Static placeholder has no enabled Connect, Reconnect, Select, Bind, Scan, or
  Clone action.
- API-backed UI displays only server-returned state.
- No provider token or provider state is stored in browser storage.

Scope creep:
- live provider behavior before backend state, fake data, browser-only provider
  clients, checkout, runner, gate, proof, analytics, or product-loop claims

## Separate Later Tracks

Checkout, runner, gate, and proof must remain separate later tracks.

Later work may reference `VcsConnection` or provider metadata only through
explicit policy and runner boundaries:

- checkout eligibility must be evaluated separately
- hosted runner checkout must use bounded runner instructions
- customer-hosted runner paths must remain first-class
- gate reads frozen receipts and artifacts, not mutable live repository state
- proof is produced by the proof/gate contour, not by repository connection

No phase above authorizes branch creation, commit creation, merge request,
pull request, repository write behavior, generic queue behavior, or hosted
execution platform behavior.

## First Implementation Recommendation

The next backend implementation-agent task should be another docs-only ADR:

```text
Phase D1 — provider credential / token storage boundary ADR
```

That is safer than starting with schema or OAuth because every live provider
slice depends on unresolved credential questions: encryption, redaction,
refresh, revocation, audit, retention, GitLab `read_api` risk, and GitLab.com
versus self-managed instance identity.

A small code skeleton is only safe after D1 explicitly decides that
credentialless `pending_setup` `VcsConnection` records are allowed. If D1 does
not decide that, D2 should remain blocked.

Console Phase D6a static placeholder can proceed before backend APIs only if it
stays non-live, disabled, and warning-copy-only. It should not be treated as a
backend implementation substitute.

## Blockers and Open Questions

Hard blockers before live provider implementation:

- Provider credential/token storage boundary.
- Token encryption / key ownership / redaction / refresh / revocation / audit.
- Token retention and deletion after disconnect or revocation.
- GitLab `read_api` broader-than-metadata risk.
- Internal adapter allowlist that excludes Repository Files API for metadata
  discovery.
- GitLab.com versus self-managed provider instance identity.
- OAuth application ownership mode for GitLab: user-owned, group-owned,
  customer-provided instance-wide, or some narrower first slice.
- Whether `RepositoryRecord` remains deferred for the first implementation
  slice.
- Whether `VcsConnection` can be created without credentials in the first code
  slice.
- Whether console Phase 2 static placeholder should proceed before backend APIs.
- Whether repository metadata candidates should be transient API responses or
  persisted snapshots before binding.
- How provider metadata retention works after provider revocation.
- How provider error states map to provider-neutral connection, metadata, and
  RepoBinding states.

Open implementation questions:

- Should the first provider-neutral repository metadata API be read-only and
  transient before any picker/bind mutation exists?
- Should `repo_bindings.vcs_connection_id` remain a string placeholder until the
  first real `VcsConnection` migration, or should D2 replace it with a typed
  relation?
- Which state names from ADR-0024 become actual enums first, and which remain
  conceptual?
- What minimum audit events are required for provider connect, refresh,
  reconnect, revoke, disable, and metadata refresh?
- Should protected branch metadata be deferred until runner/checkout policy
  design?

## Next Prompt Handoff

A follow-up implementation prompt can start with:

```text
Create a docs-only ADR for provider credential/token storage before any VCS
schema/API/provider/OAuth work. It must answer encryption, redaction, refresh,
revocation, audit, retention, GitLab read_api risk, GitLab.com versus
self-managed instance identity, and whether credentialless pending_setup
VcsConnection records are allowed. Do not add code, migrations, API routes,
OAuth, provider clients, token storage, checkout, runner, gate, or proof.
```
