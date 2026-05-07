---
id: goalrail_status
title: Goalrail Status
kind: ops_status
authority: operational
status: current
owner: ops
truth_surfaces:
  - current_state
  - implementation_status_summary
lifecycle: active-core
review_after: 2026-07-19
supersedes: []
superseded_by: null
related_docs:
  - docs/INDEX.md
  - docs/product/GOALRAIL_BUILD_ROADMAP.md
  - docs/product/GOALRAIL_IMPLEMENTATION_GUIDE.md
  - docs/ops/COMPONENTS.yaml
  - docs/ops/CONSOLE_MAIN_DEPLOYMENT_WIRING.md
---
# Goalrail Status

Last updated: 2026-05-07
Status: planning / product canon and pilot frame active; first local Go CLI with
local Project Scan baseline / overlay commands and Go server intent-plane /
public Contract aggregate and `/v1/contracts` lifecycle façade / ContractSeed /
ContractDraft / ApprovedContract / WorkItem persistence plus `plans` /
`proposals` / `acceptance` WorkItem planning control-plane flow exists; public
Contract aggregate identity is implemented as
a stable `contract_id` boundary and transitional public
seed/draft/approval/direct-task routes are removed; the typed WorkItemPlan pull
lease API from ADR-0021 is implemented without adding a generic queue;
ADR-0024 accepts a future minimal `goalrail-worker` polling loop over that API,
but no worker binary exists yet;
source-level core server CORS allowlist support exists through exact
`GOALRAIL_HTTP_CORS_ALLOWED_ORIGINS` values, with CORS disabled when unset and
wildcard origins rejected; the main console/API
deployment is now live through `11me/infra` Flux GitOps at
`https://goalrail.dev` and `https://api.goalrail.dev`, with Flux revision
`main@sha1:f4cb3db22853d0d92291f37acb055cd28e8abec7`, Kustomization
`flux-system/apps-personal` Ready=True, console/server rollouts successful,
public DNS/TLS ready, frontend/API smoke passed, and the console bundle built
against `https://api.goalrail.dev`; live API CORS is temporarily handled by
nginx ingress annotations allowing `https://goalrail.dev` because the deployed
server image still predates the app-level CORS implementation from Goalrail
PR #120; pilot-intake-ru is now a business-first public RU pilot landing per
D-0055 (`ИИ-кодинг без хаоса`, safe 2-week пилот ИИ-разработки, repository
readiness, project context, controlled tasks, verified result) rather than the
previous technical interactive walkthrough; active target domain remains
`pilot.goalrail.ru` per D-0053, canonical metadata in
`apps/web/pilot-intake-ru/index.html` remains `https://pilot.goalrail.ru/`,
the static hosting path remains operator-managed SSH static server per D-0051,
server upload, operator-managed Go sidecar migration from previous PHP-FPM
wiring, server-side TLS provisioning, server-local HTTPS smoke, public DNS
verification, public HTTPS smoke, public `/api/pilot-lead` smoke, and D-0058
digest dry-run are complete; `apps/web/console` is now the single canonical
multilingual EN/RU console source with the existing server auth API login /
optional first-login password change / `/v1/me` / logout flow, static i18next
resources, in-memory tokens only, `goalrail.console.theme` as the only
browser-storage key, and no locale persistence; live `https://console.goalrail.ru/`
remains a legacy/static RU console deployment separate from the new main
`goalrail.dev` route; and D-0047 boundaries remain intact except for the narrow
D-0056 lead-capture endpoint, D-0058 daily digest, and D-0059 Resend mail
transport on the pilot surface (no analytics, tracking, CRM, Google Sheets,
cookies, sessions, LLM/API calls, repo integration, runtime execution, broad
backend platform, chat UI, file upload, or model selector).
Owner: Vitaly

Installation/auth boundary note: ADR-0022 documents `Installation` as the
running Goalrail control-plane boundary above `Organization`, with
`self_hosted` and `saas` as the only deployment modes. The smallest server
schema foundation now exists: `installations` stores mode and
`public_base_url`, `organizations.installation_id` links Organizations to an
Installation, and organization slugs are installation-scoped. ADR-0023
documents the self-hosted user bootstrap, auth token, and browser-loopback CLI
login direction. The smallest auth credential foundation now exists:
`user_password_credentials`, `user_sessions`, server-local auth spine types, a
Squirrel-backed auth store, short-lived hashed CLI auth codes, and Argon2id
PHC-style password hashing / verification. `goalrail-server bootstrap owner`
now creates or reuses the
self-hosted Installation, primary Organization, first owner User, owner
membership, and first temporary password credential without rotating existing
credentials. The smallest server auth API lifecycle now exists:
`POST /v1/auth/login`, `POST /v1/auth/refresh`,
`POST /v1/auth/change-password`, `POST /v1/auth/logout`, `GET /v1/me`,
`GET /cli/login`, `POST /cli/login`, and `POST /v1/auth/cli/exchange`.
It verifies existing `user_password_credentials`, creates server-owned
`user_sessions` refresh-token state, signs short-lived JWT access tokens with
`GOALRAIL_AUTH_JWT_SECRET`, refreshes access tokens from opaque DB-backed
refresh-token state without refresh-token rotation, revokes the current
session on bearer-token logout, and resolves current user membership
server-side instead of trusting role claims in JWTs. ADR-0027 documents the
Organization user-management boundary as Console-backed server API routes, not
CLI user creation; the backend admin user API exists, and the canonical
Console Users UI now consumes it for Organization user
list/create/patch/temporary-password reset.
`goalrail login
<server_url>` now starts a localhost loopback listener, opens or prints the
server CLI login URL, exchanges a one-time code for tokens, and stores token
metadata in a local 0600 auth file. `GET /cli/login` and `POST /cli/login`
currently use a minimal server-rendered HTML page as a temporary CLI auth bridge
only; it is not the product web console login UI. `apps/web/console` now has
the bounded multilingual React auth flow for the existing server endpoints, and
the main `https://goalrail.dev` deployment routes it to
`https://api.goalrail.dev` through the `11me/infra` Flux GitOps path. The
legacy `https://console.goalrail.ru/` deployment remains separate. SaaS
onboarding, organization creation API, public registration, keychain
integration, Organization / Project / RepoBinding profile selection, and CLI
user creation remain unimplemented.

Current risk note: the stabilization tranche is complete repo-side through
D-0065, the operator-managed Go sidecar deployment plus public DNS/live smoke
slice has passed, and the main console/API Flux GitOps deployment has passed
public smoke. The public RU pilot surface is live through the operator-managed
SSH static server, with `/api/pilot-lead` routed to the Go sidecar rather than
the previous PHP-FPM wiring. The RU console shell is also live through the same
operator-managed static-server pattern, but remains a static visual shell only
and is not the main `goalrail.dev` deployment. Core server app-level CORS is
source-level configuration only until a post-PR-#120 server image is pinned in
infra; current live API CORS is a temporary nginx ingress bridge for
`https://goalrail.dev`. This status does not claim committed server config,
required human review, signed-commit enforcement, real-device mobile QA, or
native-speaker copy proofread. It also does not approve analytics, CRM,
database, generic queue, planning worker implementation, LLM/API, repo
integration, runtime execution, gate, proof, or broad backend platform
behavior.

## Current state

The project currently has:
- product concept canon
- operating model
- deployment model
- pilot model
- GTM model
- ICP
- decision recommendations
- product brief and supporting positioning docs
- MVP blueprint
- build roadmap
- parallel execution model
- implementation guide
- project spine schema note
- twenty-seven kernel/CLI/server/domain boundary ADRs
- ops rails
- repo-tracked Goalrail and Punk overlay surfaces
- planned flow / eval structure
- reference screens
- shared web stack rules under `apps/web/`
- one canonical multilingual console source under `apps/web/console` with EN/RU static i18next resources, existing server auth endpoints for login, optional first-login password change, `/v1/me`, logout, in-memory tokens only, no cookies or token/profile/session browser-storage persistence, `goalrail.console.theme` as the only browser-storage key, no locale persistence, three structured empty product surfaces, bottom-left Settings utility, Appearance theme picker, and API-backed Organization Users list/create/edit plus temporary-password reset using `/v1/me` organization context and the ADR-0027 Organization user-management routes; temporary passwords are shown only from the immediate create/reset response and are not persisted in browser storage; the main deployment is live at `https://goalrail.dev` with API base URL `https://api.goalrail.dev` through `11me/infra` Flux GitOps, while the old `apps/web/console-ru` workspace source has been removed and live `https://console.goalrail.ru/` remains separate
- local change-packet demo prototypes under `apps/web/demo-change-packet` and `apps/web/demo-change-packet-ru`
- a business-first RU pilot landing under `apps/web/pilot-intake-ru` for `ИИ-кодинг без хаоса`: a mostly static Founding Pilot page for a safe 2-week пилот ИИ-разработки on one product area, with illustrative repository readiness / controlled task / pilot result cards, a D-0056 minimal `POST /api/pilot-lead` email lead endpoint with local JSONL notification status, retry after `notification_failed`, in-flight `received` / `pending` rows blocked as duplicate submissions, duplicate suppression for successfully notified, legacy processed, and in-flight rows, no user-agent storage for new lead records, a landing-owned repo-side Go sidecar for the endpoint/digest/purge command under `apps/web/pilot-intake-ru/server`, server-installed daily previous-day digest at 07:00 GMT+3 when leads exist plus direct mailto fallback, no analytics, no tracking, no IP logging, no cookies, no sessions, no fingerprinting, no CRM, no Google Sheets, no repo integration, no runtime execution, no persistence beyond local JSONL lead log, no chat UI, no file upload, and no model selector; the previous 5-step technical walkthrough is demoted to internal / technical demo or checkpoint status in git history per D-0055.
- an open-source community baseline (`LICENSE`, `NOTICE`, contributor docs, issue forms, `CODEOWNERS`)
- a Go server bootstrap under `apps/server` with authenticated repository-context init, authenticated metadata-only RepoBinding init, Postgres-backed source-neutral intake, Goal promotion, Goal readiness state, ClarificationRequest / ClarificationAnswer storage, authenticated clarification answer continuation, ContractSeed creation, ContractDraft creation/update/ready_for_approval, ApprovedContract approval, WorkItem `plans` / `proposals` / `acceptance` planning control-plane flow, planned task read-by-ID, EventLog persistence, auth credential/session persistence primitives, Argon2id password hashing primitives, `goalrail-server bootstrap owner`, transactional canonical write + event append hardening when DB is configured, production product/auth route wiring that returns `503 database_not_configured` instead of falling back to in-memory state when DB config is absent, no remaining map-backed product store implementations under `apps/server/internal/store`, and no production-looking in-memory event log helper

## What is real now

### Product
- thesis fixed: **от бизнес-цели до проверенного изменения в коде**
- Goalrail is framed as a **productized operating layer** for AI-assisted delivery
- two-plane model fixed:
  - Intent / Planning
  - Delivery / Execution
- working contract is the central working object
- verify / proof is part of the core product contour
- fixed core vs configurable knobs is explicit in the canon

### Commercial and deployment model
- managed rollout is the default early rollout motion
- pilot-first entry model is explicit
- free qualification + paid pilot is explicit
- RU-first, founder-led GTM is explicit
- ICP is separate from GTM copy and landing wording
- the product is positioned as a supplement layer over existing tools, not a bespoke process redesign

### Architecture
- Project Spine remains the canonical center
- runtime-neutral, CLI-first posture is explicit
- one primary writer runtime per writable run is explicit
- advisory panels are distinct from task execution groups
- risk- and policy-driven routing is part of the model
- frozen verification inputs and baseline-aware verification are explicit
- canonical objects vs derived views are explicit
- roadmap-to-research-to-punk loop is explicit
- runner and repository checkout boundary is documented in ADR-0008
- repository checkout and check execution must happen behind runners, not inside the API server
- customer-hosted runners are first-class in the architecture model
- ADR-0008 separates VCS discovery, repository binding, and checkout access
- customer-hosted runners may operate without Goalrail cloud-side clone access
- ADR-0008 documents the first runner prototype direction as hosted-only read-only ephemeral checkout
- hosted runner workers are expected to use pull-based / poll-based job leasing
- customer-hosted runner remains documented but unimplemented
- ADR-0010 documents the MVP Organization / Project / RepoBinding and
  persistence bootstrap boundary
- MVP will use direct `RepoBinding` before `RepositoryRecord`
- persistence direction is pgx + Squirrel + goose, with no `sqlc` and no ORM
- ADR-0011 documents answer application to Goal hints as a server-owned
  transition with no readiness re-check, contract seed, work item, gate, or proof
- ADR-0012 documents explicit readiness re-check after applied answers as a
  separate server-owned transition before any contract seed boundary
- ADR-0013 documents `ContractSeed` as an explicit server-owned canonical
  bridge from `Goal(ready_for_contract_seed)` to future contract drafting
- ADR-0014 documents `ContractDraft(draft)` as an explicit server-owned draft
  boundary from `ContractSeed(created)`, before approval, work item, gate, or
  proof
- ADR-0015 documents `ContractDraft` review/update as an explicit server-owned
  draft-only boundary for proposed fields, before `ready_for_approval`, approval,
  work item, gate, or proof
- ADR-0016 documents `ContractDraft(ready_for_approval)` as an explicit
  server-owned state transition with completeness checks and `marked_by` audit
  identity, before approval, approved Contract, work item, execution, gate, or
  proof
- ADR-0017 documents `ApprovedContract` as a separate server-owned approval
  snapshot from `ContractDraft(ready_for_approval)`, before work item planning,
  execution, gate, or proof
- ADR-0018 documents `WorkItem(planned)` as a server-owned non-executable
  planning boundary from `ApprovedContract(approved)`, before assignment,
  claiming, execution, runner checkout, receipt, gate, or proof
- ADR-0019 qualifies WorkItem planning with a Kubernetes-style control-plane
  split: the API server owns canonical state, validation, persistence, events,
  and accepted WorkItems, while repo-aware planning computation belongs behind
  worker / controller / runner boundaries through planning request, proposal,
  and acceptance states
- ADR-0020 documents public `Contract` as the stable product-facing lifecycle
  identity across seed, draft, approval, and later planning, while
  `ContractSeed`, `ContractDraft`, and `ApprovedContract` remain internal
  lifecycle records; the server now implements the smallest stable
  `contract_id` aggregate boundary and the smallest public `/v1/contracts`
  lifecycle façade routes
- ADR-0021 documents the typed WorkItemPlan pull lease boundary: the accepted
  direction is `WorkItemPlan(state=queued)` as the canonical typed
  planning queue item, `WorkItemPlanLease` as typed reservation state, API
  server-owned pull leasing through `POST /v1/plans/leases`, FIFO v0
  scheduling with lazy expiry, and no generic `queue_jobs` / `jobs` /
  `work_queue` table for this boundary
- ADR-0024 documents the minimal planning worker loop boundary: the accepted
  future first worker is a separate thin `goalrail-worker` process that talks
  only to the API server, polls one plan lease, reads one plan, renews while
  working when needed, computes or collects one proposal, submits it with
  `lease_id` and `lease_token`, and repeats. It is not a runner and remains
  unimplemented: no checkout, execution, direct Postgres writes, WorkItem
  creation, assignment/claiming, queue/outbox/runtime registry, `Run`, receipt,
  `GateDecision`, or `Proof`.
- ADR-0022 documents the Installation boundary above Organization:
  `Installation` is the concrete running Goalrail control plane / instance,
  Organization remains the tenant/workspace boundary, `self_hosted` and `saas`
  are the only deployment modes, MVP starts with one bootstrapped primary
  Organization in `self_hosted` mode, the backend must remain
  organization-aware, and `public_base_url` belongs to Installation
- ADR-0023 documents the user bootstrap, auth, and CLI login boundary:
  self-hosted bootstrap creates the first product super admin as
  `OrganizationMembership(owner)`, there is no public registration in the MVP,
  admins create users with backend-generated temporary passwords, first-login
  password change is required, email invite/reset delivery is deferred,
  password credentials should live outside `users`, access tokens should be
  short-lived JWTs, refresh tokens should be opaque DB-backed server state, role
  checks should use server-side `OrganizationMembership`, and `goalrail login`
  uses explicit `server_url` plus browser localhost loopback. The first
  credential/session foundation now exists as schema, server-local types,
  Squirrel-backed store primitives, and password hashing/verification, the
  server now has the smallest self-hosted `goalrail-server bootstrap owner`
  command for initial owner credentials, the smallest server auth API exists
  for login, refresh, password change, logout, current-user profile, and CLI
  code exchange, and the CLI now stores token metadata locally after browser
  loopback login. `apps/web/console` now implements a bounded multilingual React
  auth flow over those endpoints for login, first-login password change,
  current profile lookup, and logout. Organization / Project / RepoBinding
  profile selection remains unimplemented; the current `/cli/login` HTML page
  is a temporary CLI auth bridge only.
- Repository access MVP is reset to runner-owned credentials plus RepoBinding
  context. Existing RepoBinding / repository-context init remains real:
  RepoBinding identifies which repository a Project works with, remains
  metadata/context only, and does not grant checkout permission.
- The intended next repository-access direction is API-issued checkout
  instructions derived from WorkItem / RepoBinding context and runner-owned
  local credentials, with no repository secrets stored by the API server in the
  MVP.
- ADR-0025 documents the accepted local Project Scan and repository baseline
  lifecycle boundary: `RepositoryBaselineProfile` is an immutable committed
  repository-shape profile keyed by RepoBinding, canonical repo root, HEAD SHA,
  and scanner schema; `WorkspaceOverlay` separately records dirty, unmerged,
  partial, submodule, sparse-checkout, shallow-clone, and worktree-specific
  state; and future `ContractContextPack` records are task-specific cuts that
  reference exact baseline and overlay versions. Background scans are best-effort
  only and cannot bypass synchronous freshness checks. The API server remains
  outside repository clone, source upload by default, in-process checks, gate,
  and proof for this boundary.
- The CLI now implements the first local Project Scan v0 foundation:
  `apps/cli/internal/projectscan` builds and caches immutable
  `RepositoryBaselineProfile` JSON plus cheap `WorkspaceOverlay` JSON under the
  user cache directory, `goalrail project scan/status` report freshness, and
  server-backed `goalrail init` runs a best-effort quick local Project Scan
  after `.goalrail/project.yml` is written or verified. `.goalrail/project.yml`
  remains the committed repository/team marker, while `.goalrail/.gitignore`
  keeps Goalrail-owned machine-local `.goalrail/local`, `.goalrail/cache`,
  `.goalrail/state`, `.goalrail/tmp`, and `*.local.*` files out of Git. This
  remains local-only and does not add server baseline persistence, server clone,
  source upload, background daemon, runner, context-pack generation, gate, or
  proof.
- ADR-0027 documents the Organization user management boundary:
  regular Organization users are created through Console UI backed by
  server API, not through CLI user creation; canonical identity is `User`,
  access is `OrganizationMembership`, password credentials stay separate from
  `users`, temporary passwords are backend-generated and shown once, v0
  user-management authorization is owner-only, role checks load current
  membership server-side, cross-organization attempts are rejected, and the
  last active owner cannot be disabled or demoted. The backend now implements
  `GET /v1/organizations/{organization_id}/users`,
  `POST /v1/organizations/{organization_id}/users`, and
  `PATCH /v1/organizations/{organization_id}/users/{user_id}`, plus
  `POST /v1/organizations/{organization_id}/users/{user_id}/temporary-password-resets`
  with owner-only v0 authorization, one-time temporary password return for newly
  created users and reset rotations, reset-side active session revocation,
  safe attachment of existing active users that are not yet members of the
  target Organization without credential rotation, membership-scoped
  active/inactive updates, and last-active-owner protection.
  Settings / Users consumes these API-backed records, and there is no
  `goalrail users create` command.
- No checkout job, checkout instruction, checkout receipt, runner clone/fetch,
  mounted-workspace checkout flow, provider credential storage, VcsConnection,
  OAuth, provider client, gate, or proof exists yet.
- D-0041 documents transactional Postgres-backed intake create, Goal promotion,
  and Goal readiness write/event boundaries without adding queue, outbox, or
  Unit of Work framework semantics
- D-0042 documents transactional Postgres-backed ContractSeed and ContractDraft
  creation with durable EventLog appends while leaving approval, work items,
  runner, gate, and proof as later boundaries
- D-0045 documents transactional Postgres-backed ApprovedContract approval with
  durable `contract.approved` while leaving runner, gate, and proof as later
  boundaries; WorkItem planning is now the separate ADR-0018 boundary
- D-0046 documents WorkItem planning as a non-executable boundary; the server
  now implements ADR-0019's public `plans` / `proposals` / `acceptance`
  control-plane flow with Postgres persistence when configured, without
  assignment, claiming, execution, runner checkout, receipt, gate, or proof;
  the previous direct public task creation endpoint is removed

### Delivery model
- roadmap phases defined
- checkpoint model defined
- bounded slice workflow defined
- implementation discipline fixed: `punk`
- execution parallelism and advisory parallelism are separated conceptually
- kernel schema note and twenty-seven boundary ADRs exist

### Repo structure
- the repo now mirrors `punk`-style planning boundaries
- `.goalrail/work/` is reserved for goals, reports, and bounded planning artifacts such as demo-planning packs
- `.goalrail/knowledge/` is reserved for advisory research and idea backlog
- `.punk/publishing.toml` remains the repo-local binding
- `.punk/publishing/` legacy directory has been removed from repo after manual copy/verify; the runtime publishing workspace lives in external user/platform-local storage
- `.punk/publishing.local.toml` is the ignored local-only manual-bootstrap pointer; resolver/runtime implementation is pending
- `.goalrail/flows/` and `.goalrail/evals/` exist as planned future structure, not executable product surfaces
- `apps/web/` is now the shared namespace for frontend resources and stack rules
- `apps/web/console` is the canonical multilingual EN/RU console source with real auth API login, optional first-login password change, `/v1/me`, logout, neutral internal role/status/surface IDs, runtime i18next language switching, no locale storage, an ops-style Contracts surface that can load one real public Contract aggregate by explicit `contract_id` through `GET /v1/contracts/{id}`, structured empty Delivery Readiness and Proof surfaces, bottom-left Settings utility, Appearance theme picker, local-only theme preference under `goalrail.console.theme`, and API-backed Organization Users list/create/edit using `/v1/me` organization context plus the ADR-0027 routes; the main deployment is live at `https://goalrail.dev` and uses `https://api.goalrail.dev` through `11me/infra` Flux GitOps
- `apps/web/demo-change-packet` is the current React + Vite + Mantine EN change-packet demo prototype, deployed through standalone infra at `demo.goalrail.dev`
- `apps/web/demo-change-packet-ru` is the separate RU copy of the change-packet demo prototype, deployed through standalone infra at `demo.goalrail.ru` rather than in-app i18n
- `apps/web/console-ru` source has been removed. The live `https://console.goalrail.ru/` deployment remains a separate legacy RU static release and is not migrated by the main `goalrail.dev` slice.
- `apps/web/pilot-intake-ru` is the current public React + Vite + Mantine RU business-first pilot landing for `pilot.goalrail.ru` (`ИИ-кодинг без хаоса`, safe 2-week пилот ИИ-разработки, repository readiness, project context, controlled tasks, verified result); it includes a narrow landing-owned Go sidecar under `apps/web/pilot-intake-ru/server` for lead capture and digest source, and it supersedes the previous technical interactive walkthrough as the primary public RU landing per D-0055.
- `apps/cli` is the first stdlib-only Go CLI bootstrap with canonical binary entrypoint `cmd/goalrail`
- CLI commands now exist for `version`, normal server-backed `goalrail init`, optional `goalrail init --base <branch>` workflow base override, low-level `goalrail init --project <project_id>`, explicit auth-free `goalrail init --local-demo`, explicit provider-neutral `goalrail agent install`, local `goalrail project scan/status`, server-backed `goalrail work start`, server-backed `goalrail work continue`, server-backed `goalrail work answer`, server-backed `goalrail contract draft`, server-backed `goalrail contract update`, server-backed `goalrail contract submit`, server-backed `goalrail contract approve --confirm-user-approval`, `readiness scan`, `contract validate`, `proof show`, and the first `goalrail login <server_url>` browser-loopback auth path; normal `goalrail init` uses local Git metadata plus the stored login profile to call the server repository-context init endpoint, records a bounded metadata-only repository context snapshot on the server, writes a non-secret Git-root `.goalrail/project.yml` marker only after server success, ensures `.goalrail/.gitignore` for Goalrail-owned machine-local state, and runs a local Project Scan cache write, while `goalrail init --project <project_id>` remains the low-level Project-scoped RepoBinding init path; `goalrail agent install` writes `.goalrail/agent/GOALRAIL.md` and `.goalrail/agent/commands.json` with `work continue`, `work answer`, `contract draft`, `contract update`, `contract submit`, `contract approve`, question_id-bound answer guidance, structured contract field guidance, explicit user approval guidance, and local repository receipt guidance, may create a tiny root `AGENTS.md` shim only when missing, never overwrites an existing root `AGENTS.md`, and does not install provider-specific Codex, Claude, Gemini, Cursor, Windsurf, Gravity, runner, gate, proof, readiness, Jira, or Linear automation
- `apps/server` is the first Go HTTP server bootstrap with canonical binary entrypoint `cmd/goalrail-server`
- server endpoints include `GET /livez`, `GET /readyz`, `GET /version`, `POST /v1/auth/login`, `GET /cli/login`, `POST /cli/login`, `POST /v1/auth/cli/exchange`, `POST /v1/auth/refresh`, `POST /v1/auth/change-password`, `POST /v1/auth/logout`, `GET /v1/me`, `GET /v1/organizations/{organization_id}/users`, `POST /v1/organizations/{organization_id}/users`, `PATCH /v1/organizations/{organization_id}/users/{user_id}`, `POST /v1/init/repository-context`, `POST /v1/repo-bindings/{repo_binding_id}/context-snapshots`, `POST /v1/projects/{project_id}/repo-bindings/init`, `POST /v1/intakes`, `GET /v1/intakes/{id}`, `POST /v1/intakes/{id}/goals`, `POST /v1/goals/{id}/readiness`, `POST /v1/goals/{id}/continuation`, `POST /v1/clarifications/{id}/answers/continuation`, `POST /v1/goals/{id}/clarifications`, `POST /v1/clarifications/{id}/answers`, `POST /v1/answers/{id}/applications`, `POST /v1/contracts`, `GET /v1/contracts/{id}`, `PATCH /v1/contracts/{id}`, `POST /v1/contracts/{id}/submissions`, `POST /v1/contracts/{id}/approvals`, `POST /v1/contracts/{id}/plans`, `GET /v1/plans/{id}`, `POST /v1/plans/leases`, `GET /v1/plans/leases/{id}`, `PATCH /v1/plans/leases/{id}`, `POST /v1/plans/{id}/proposals`, `GET /v1/proposals/{id}`, `POST /v1/proposals/{id}/acceptance`, and `GET /v1/tasks/{id}`; there is no full RepoBinding CRUD endpoint, `GET /v1/plans`, `GET /v1/proposals`, `GET /v1/tasks`, or worker lease list/search endpoint, and the previous public `/v1/goals/{id}/contract-seeds`, `/v1/contract-seeds/{id}/contract-drafts`, `/v1/contract-drafts/{id}`, and direct `POST /v1/contracts/{id}/tasks` lifecycle/planning routes are no longer registered
- `POST /v1/contracts/{id}/plans` resolves `{id}` as stable public
  `contract_id`, requires the Contract to be `approved`, and creates a
  server-owned `WorkItemPlan(queued)` without creating WorkItems
- `apps/server` now has a Postgres persistence foundation for the Organization / Project / RepoBinding context plus metadata-only RepoBinding init, metadata-only RepositoryContextSnapshot records, IntakeRecord, Goal, public Contract aggregate, ContractSeed, ContractDraft, ApprovedContract, WorkItemPlan lease state, and EventLog state
- server config uses structured Postgres fields
  `GOALRAIL_DATABASE_HOST`, `GOALRAIL_DATABASE_PORT`,
  `GOALRAIL_DATABASE_NAME`, `GOALRAIL_DATABASE_USER`,
  `GOALRAIL_DATABASE_PASSWORD`, and `GOALRAIL_DATABASE_SSLMODE`, plus
  `GOALRAIL_AUTH_JWT_SECRET` and optional exact-origin
  `GOALRAIL_HTTP_CORS_ALLOWED_ORIGINS`; the server may start without database
  configuration for health/version only, product/auth API routes return
  `503 database_not_configured` without DB config, auth endpoints fail with a
  clear auth configuration error when token signing or validation is needed
  without the JWT secret, and CORS is disabled unless explicit origins are
  configured
- `goalrail-server migrate up` applies the editable pre-production init migration
- `goalrail-server seed dev` applies the idempotent dev seed
- `goalrail-server bootstrap owner` applies the smallest self-hosted owner
  bootstrap: explicit flag input, one reused or created `self_hosted`
  Installation with normalized `public_base_url`, one reused or created primary
  Organization, one reused or created owner User, `OrganizationMembership(owner)`,
  and a generated temporary password credential with
  `must_change_password = true`; existing owner credentials are not silently
  rotated
- the init migration creates `users`, `user_password_credentials`, `user_sessions`, `cli_auth_codes`, `installations`, `organizations`, `organization_memberships`, `projects`, `repo_bindings`, `intake_records`, `goals`, `clarification_requests`, `clarification_answers`, `contracts`, `contract_seeds`, `contract_drafts`, `approved_contracts`, `work_item_plans`, `work_item_plan_proposals`, `work_items`, and `events` with UUID persisted ID columns for canonical entities
- `user_password_credentials` stores password hash material outside `users` with first-login `must_change_password` state; `user_sessions` stores opaque refresh-token/session server state with `active`, `revoked`, and `expired` states plus expiry, revocation, and last-used timestamps; `cli_auth_codes` stores only hashed one-time CLI authorization codes with state, localhost callback URI, S256 code challenge metadata, TTL, and consumption timestamp
- `apps/server/internal/auth/password` implements Argon2id password hashing and verification using PHC-style encoded strings with algorithm, version, memory, time, parallelism, salt, and derived key fields; empty passwords and malformed or unsupported hashes return errors
- `apps/server/internal/store/postgres_auth_store.go` provides Squirrel-backed credential/session and CLI auth code upsert/lookup/consume primitives; login business logic stays in `apps/server/internal/auth`
- the dev seed creates deterministic UUIDv7 IDs: `018f0000-0000-7000-8000-000000000001`, `018f0000-0000-7000-8000-000000000006`, `018f0000-0000-7000-8000-000000000002`, `018f0000-0000-7000-8000-000000000005`, `018f0000-0000-7000-8000-000000000003`, and `018f0000-0000-7000-8000-000000000004`
- the dev seed creates one `self_hosted` Installation with `public_base_url = http://localhost:8080` before creating the dev Organization linked to it
- the project-context store builds runtime SQL with Squirrel and executes Installation / Organization / Project / RepoBinding upserts and repo-binding context lookups through pgx/pgxpool
- the source-neutral intake API now requires `project_id` and `repo_binding_id`, validates the repo binding against the persisted Project / RepoBinding context when DB is configured, derives `organization_id`, stores `IntakeRecord` in Postgres when DB is configured, and appends a durable `intake.received` event with context fields
- Goal promotion stores `Goal` as non-executable normalized intent in Postgres when DB is configured, carries `organization_id`, `project_id`, and `repo_binding_id` from the IntakeRecord, prevents duplicate promotion through the persisted `intake_id` uniqueness boundary, and appends durable `goal.created` and `intake.promoted_to_goal` events with context fields
- Goal readiness updates persisted `Goal` state and readiness reason codes when DB is configured, returns reason codes, appends durable readiness transition events, and can be explicitly re-run after answer application
- Postgres-backed repository-context Project creation, RepoBinding init create, intake create, Goal promotion, Goal readiness, ClarificationRequest creation, ClarificationAnswer recording, answer application, ContractSeed creation, ContractDraft creation/update, ContractDraft ready_for_approval, and ApprovedContract approval writes now share a transaction with their expected event appends, so the durable canonical write does not commit without its audit events
- ClarificationRequest creation stores an open request durably when DB is configured, returns `503 database_not_configured` through production route wiring without DB, generates deterministic questions from Goal readiness reason codes, guards one open request per Goal, and appends `clarification.requested` transactionally with the request write in the Postgres path
- ClarificationAnswer recording stores canonical answer evidence durably when DB is configured, returns `503 database_not_configured` through production route wiring without DB, requires all questions answered, transitions the request from `open` to `answered`, and appends `clarification.answer_recorded` and `clarification.request_answered` through the configured EventLog
- answer application marks the persisted ClarificationAnswer as applied when DB is configured, updates persisted Goal intent-plane hints, rejects unsupported raw-text `goal.intent_owner` mapping, guards repeated application with `409 already_applied`, and appends events through the configured EventLog; it does not call readiness automatically
- `POST /v1/contracts` creates a public Contract lifecycle view from a ready Goal by creating internal `ContractSeed(created)` and `ContractDraft(draft)` records, returning Contract state `draft`, and not creating approval, tasks, execution, gate, or proof
- `PATCH /v1/contracts/{id}` updates the current internal draft's proposed fields through the public `contract_id`, now requires bearer authentication and server-side Organization ownership before mutation, accepts optional project/repo expectations, derives `updated_by` from the authenticated user for audit identity, rejects empty/blank update values, preserves `ContractDraft.state = draft`, appends `contract_draft.updated`, and does not approve or create tasks
- `POST /v1/contracts/{id}/submissions` transitions the current internal draft to `ready_for_approval`, moves Contract state to `ready_for_approval`, now requires bearer authentication and server-side Organization ownership before mutation, accepts optional project/repo expectations, derives `marked_by` from the authenticated user for audit identity, runs completeness checks, and does not approve Contract, create `WorkItem`, write `GateDecision`, or create `Proof`
- `POST /v1/contracts/{id}/approvals` creates an immutable internal `ApprovedContract(approved)` snapshot from the current ready draft, moves Contract state to `approved` with `approved_snapshot_id`, now requires bearer authentication and server-side Organization ownership before mutation, accepts optional project/repo expectations, derives `approved_by` from the authenticated user for audit identity, guards repeated approval with `409 already_approved`, and does not mutate `ContractDraft`, start execution, write `GateDecision`, or create `Proof`
- WorkItem planning now uses `Plan -> Lease -> Proposal -> Acceptance`: one `WorkItemPlan` per approved public Contract in v0, typed `WorkItemPlanLease(active/completed/expired)` records reserve queued or expired leased plans through `POST /v1/plans/leases`, proposal submission requires `lease_id` plus `lease_token`, explicit acceptance materializes one or more durable canonical `WorkItem(planned)` records with `plan_id` and `proposal_id`, persists the records in Postgres when DB is configured, exposes `GET /v1/tasks/{id}` for single task reads, appends `work_item.created` for each accepted task transactionally with Postgres acceptance, and does not assign, claim, create `Run`, start execution, checkout a repository, submit a receipt, write `GateDecision`, or create `Proof`; workers/planners submit proposals through the API and do not write WorkItems directly to the DB
- the typed WorkItemPlan pull lease API is implemented with `POST /v1/plans/leases`, `GET /v1/plans/leases/{id}`, and `PATCH /v1/plans/leases/{id}`; raw lease tokens are returned only on create, stored only as hashes, and no generic queue implementation exists
- the accepted next worker direction is a minimal API-only `goalrail-worker`
  planning loop over typed leases; it has not been implemented and does not
  imply checkout, execution, direct DB writes, WorkItem creation by the worker,
  assignment/claiming, queue/outbox/runtime registry, `Run`, receipt,
  `GateDecision`, or `Proof`
- the runner / repository checkout boundary is documented in ADR-0008, but no runner implementation exists yet
- the `ClarificationAnswer` boundary is documented in ADR-0009; the answer application to Goal hints boundary is documented in ADR-0011, and clarification request/answer state is durable with Postgres when configured
- the explicit readiness re-check after applied answers boundary is documented in ADR-0012, and the existing readiness endpoint is verified to move an applied-answer Goal to `ready_for_contract_seed` without creating contract/work/gate/proof artifacts
- the `ContractSeed` boundary is documented in ADR-0013 and implemented as a Postgres-backed internal snapshot when DB is configured; there is no standalone public ContractSeed route, and the public `POST /v1/contracts` façade composes internal seed plus draft creation under one stable `contract_id`; standalone seed creation does not approve Contract, create `WorkItem`, write `GateDecision`, or create `Proof`
- the `ContractDraft` boundary is documented in ADR-0014 and implemented as a Postgres-backed draft creation boundary when DB is configured; it does not create approved Contract, `WorkItem`, `GateDecision`, or `Proof`
- the `ContractDraft` review/update boundary is documented in ADR-0015 and implemented as a draft-only update boundary; it does not introduce `ready_for_approval`, approved Contract, `WorkItem`, `GateDecision`, or `Proof`
- the `ContractDraft ready_for_approval` boundary is documented in ADR-0016 and implemented as an explicit `draft -> ready_for_approval` state transition with completeness checks and `marked_by` audit identity; it is not approval, approved Contract, `WorkItem`, execution, `GateDecision`, or `Proof`
- the Contract approval boundary is documented in ADR-0017 and implemented as `ContractDraft(ready_for_approval) -> ApprovedContract`; approval does not start execution, write `GateDecision`, or create `Proof`
- the WorkItem planning boundary is documented in ADR-0018 and ADR-0019 and implemented as a public `Plan -> Lease -> Proposal -> Acceptance -> WorkItem(planned)` control-plane flow with durable Postgres storage when configured and single task read by ID; the worker/controller/runner execution-side implementation remains deferred; WorkItem planning is not assignment, claiming, execution, `Run`, runner checkout, receipt, `GateDecision`, or `Proof`
- the WorkItemPlan pull lease boundary is documented in ADR-0021 and implemented
  as a typed server API/persistence slice; ADR-0024 documents the accepted
  minimal planning worker loop direction, but no planning worker/controller or
  runner-backed planning implementation exists yet
- the public Contract identity boundary is documented in ADR-0020 and the
  server now exposes the smallest public `/v1/contracts` lifecycle façade while
  keeping `ContractSeed`, `ContractDraft`, and `ApprovedContract` as internal
  lifecycle records linked to the stable `contract_id`
- the Organization / Project / RepoBinding and persistence bootstrap boundary is documented in ADR-0010, and the first server-local Postgres foundation exists
- `.github/` now contains real contributor/community health surfaces, docs-check
  and PR intake gate workflows, and D-0063 repo checks CI for Go + web surfaces
  only; D-0064 branch protection is active as external GitHub configuration for
  `main`, requiring docs-check, pr-intake-gate, the three Go checks, and web
  workspaces before merge
- `scripts/` remains parked for future bounded implementation slices

## What is not real yet

- no standalone Project Spine schema package beyond CLI/server-local DTO subsets
- no runtime registry implementation
- no production runtime CLI beyond the `apps/cli` command foundation, first browser-loopback login, normal server-backed `goalrail init`, low-level `goalrail init --project <project_id>`, explicit auth-free `goalrail init --local-demo`, explicit provider-neutral `goalrail agent install`, marker-backed `goalrail work start`, marker-backed `goalrail work continue`, marker-backed `goalrail work answer`, marker-backed `goalrail contract draft`, marker-backed `goalrail contract update`, marker-backed `goalrail contract submit`, and marker-backed `goalrail contract approve`
- no server integration for the CLI beyond `goalrail login <server_url>`, server-backed repository-context init, low-level server-backed RepoBinding init, `goalrail work start` using existing `/v1/me`, `/v1/intakes`, and `/v1/intakes/{id}/goals`, `goalrail work continue` using `/v1/me` plus `/v1/goals/{id}/continuation`, `goalrail work answer` using `/v1/me` plus `/v1/clarifications/{id}/answers/continuation`, `goalrail contract draft` using `/v1/me`, local Project Scan evidence, marker project/repo expectations, and authenticated create-or-return `/v1/contracts`, `goalrail contract update` using `/v1/me`, structured fields JSON, marker project/repo expectations, and authenticated `PATCH /v1/contracts/{id}`, `goalrail contract submit` using `/v1/me`, marker project/repo expectations, and authenticated `POST /v1/contracts/{id}/submissions`, and `goalrail contract approve` using `/v1/me`, marker project/repo expectations, explicit `--confirm-user-approval`, and authenticated `POST /v1/contracts/{id}/approvals`; `goalrail agent install` is local file installation only
- no server-owned canonical domain implementation beyond RepoBinding init, the persisted `IntakeRecord` / `Goal` / `ClarificationRequest` / `ClarificationAnswer` / public Contract lifecycle façade / internal `ContractSeed` / `ContractDraft creation/update/ready_for_approval` / `ApprovedContract` / WorkItem planning plan/lease/proposal/acceptance slice yet
- no automatic/background readiness re-check outside explicit answer continuation or readiness/continuation calls; readiness reconciliation remains explicit through `goalrail work answer`, the readiness endpoint, or `goalrail work continue`
- no WorkItem assignment/claiming, `Run`, receipt, GateDecision, or Proof yet
- no planning worker/controller, runner-backed planning implementation, lease
  protocol, assignment/claiming, checkout, execution, generic queue, outbox,
  broker, runtime registry, `Run`, receipt, `GateDecision`, or `Proof` yet
- no production repo authorization or deploy-key provisioning in the CLI
- no broad RepoBinding state sync beyond repository-context and explicit metadata-only init
- no production organization/user/provider connection/repository catalog implementation beyond the dev-seeded Installation / Organization / Project / RepoBinding Postgres foundation and metadata-only RepoBinding init yet
- no Installation bootstrap API, setup flow, or public management surface beyond
  the schema foundation and local `goalrail-server bootstrap owner` command yet
- no refresh-token rotation, public registration, SaaS onboarding, organization
  creation API, admin user creation endpoint, keychain integration,
  Organization / Project / RepoBinding CLI profile selection, or data-backed
  Goalrail web product loop yet; the current server-rendered `/cli/login` page
  is only a temporary CLI auth bridge for `goalrail login <server_url>`
- no VcsConnection, provider UI integration, OAuth, provider client, or provider
  credential storage implementation yet
- no `ContractContextPack` implementation yet; the local
  `RepositoryBaselineProfile` / `WorkspaceOverlay` CLI foundation exists, but
  there is still no server baseline persistence, background worker, raw source
  upload, server clone, gate, or proof behavior
- no `RepositoryRecord` implementation; it is intentionally deferred for the MVP
- no `RepositoryRecord.source_kind` implementation
- no `RepoBinding.access_mode` implementation beyond `metadata_only` init
- no CRUD onboarding endpoints yet
- no full manual-declared repository registration flow beyond repository-context and explicit metadata-only RepoBinding init
- no runner-reported repository metadata flow
- no runner registration, runner assignment, checkout request, checkout receipt,
  planning controller, or worker implementation yet
- no hosted runner pool implementation yet
- no checkout job implementation yet
- no customer-hosted runner installer/registration/auth yet
- no checkout receipt trust or attestation implementation yet
- no repository clone/readiness implementation in either hosted or customer-hosted runner mode yet
- no persistent mirrors
- no repository writes
- no executable flow specs yet
- no runnable eval harness yet
- no gate/proof implementation; `proof show` only renders provided local JSON, and the server does not create decisions or proof
- no advisory panel implementation
- no `goalrail users create` command, data-backed Goalrail goal-to-proof web
  UI, or goal-to-proof product loop yet
- RU pilot landing static files are uploaded to the operator-managed SSH server; the repository source for D-0056/D-0057/D-0058/D-0059/D-0061 lead capture and digest is now a narrow landing-owned Go sidecar under `apps/web/pilot-intake-ru/server`, replacing the transitional PHP source in repo; on 2026-04-30 the operator-managed server wiring moved from the earlier PHP-FPM endpoint to the Go sidecar; the server-local Resend HTTPS mail transport uses `skill7.dev` sender and server-local API key, with local sendmail/Postfix fallback where available, server-local direct notification override configured outside the repo, fallback to `pilot@goalrail.dev`, public/manual pilot contact `pilot@goalrail.dev`, visible Telegram channel `@goalrail`, JSONL-based duplicate suppression, daily previous-day digest cron at 07:00 GMT+3 when leads exist, local JSONL lead log with UTC and GMT+3 submission fields for new rows, no user-agent storage for new rows, and D-0061 notification status so failed mail notifications remain retryable while in-flight attempts do not start duplicate mail delivery; D-0065 adds a local dry-run-first purge command for JSONL retention, and reverse-proxy rate limiting is applied as an operator-managed deployment guardrail without committed config; server-local Go sidecar, digest dry-run, purge dry-run, public DNS, public HTTPS, and public `/api/pilot-lead` smoke passed
- no tracker sync
- no proof-producing demo
- no CTO / Head of Engineering deck yet
- `GOALRAIL_LANDING_COPY.md` still reflects older prompt / handoff framing and needs rewrite under the new pilot-first motion
- no DCO enforcement automation or asset provenance inventory yet

## Active checkpoint target

Current implementation target:
- **C1 — Core objects compile and persist**

Current exit condition:
- core domain types compile
- event envelope exists in code
- serialization and validation tests exist
- canonical vs derived state remains explicit in code

Current packaging target:
- ops docs are synchronized with the new concept / deployment canon
- repo overlay boundaries keep Goalrail and Punk working artifacts out of the root
- `GOALRAIL_OFFER.md` exists as the current sellable package source
- `apps/web/demo-change-packet` and `apps/web/demo-change-packet-ru` provide verified frontend change-packet walkthrough prototypes; EN and RU demo domains are wired independently through standalone infra without changing product phase order
- `apps/web/console` provides the verified canonical multilingual EN/RU console source with existing server login / first-login password change / `/v1/me` / logout plus an ops-style Contracts surface that can load one real public Contract aggregate by explicit `contract_id` through `GET /v1/contracts/{id}`; tokens remain in React memory only, locale is not persisted, `goalrail.console.theme` remains the only browser storage key, Users renders `/v1/me` only, the main `https://goalrail.dev` deployment is live with API base URL `https://api.goalrail.dev`, and legacy `https://console.goalrail.ru/` remains separate; the console does not claim contract list/search, draft body/detail reads, durable user settings API, analytics, runner, gate, proof, repo integration, or product-loop implementation
- `apps/cli` provides a verified Go CLI bootstrap plus first `goalrail login <server_url>` server auth path with browser loopback, random state, S256 verifier/challenge exchange, normal server-backed `goalrail init` repository-context bootstrap, optional `goalrail init --base <branch>` workflow base override without Git mutation, low-level `goalrail init --project <project_id>`, explicit auth-free `goalrail init --local-demo`, explicit provider-neutral `goalrail agent install`, local `goalrail project scan/status` freshness commands, marker-backed `goalrail work start` with `--body-file <path|->`, marker-backed `goalrail work continue --goal-id <goal_id>`, marker-backed `goalrail work answer --clarification-request-id <id> --answers-file <path|->`, marker-backed `goalrail contract draft --goal-id <goal_id>`, marker-backed `goalrail contract update --contract-id <contract_id> --fields-file <path|->`, marker-backed `goalrail contract submit --contract-id <contract_id>`, and marker-backed `goalrail contract approve --contract-id <contract_id> --confirm-user-approval`; normal server-backed init records a bounded server-side metadata inventory snapshot after repository-context init, writes the non-secret Git-root `.goalrail/project.yml` repository marker after server success, ensures `.goalrail/.gitignore` for Goalrail-owned machine-local state, and runs a local Project Scan cache write, while `work start` reads that marker to create an IntakeRecord and Goal through existing server endpoints and returns a `goalrail.cli.v1` JSON envelope with `display.summary` plus an available continuation command. `work continue` reads the same marker plus stored login profile, validates `/v1/me` organization membership before mutation, calls authenticated `/v1/goals/{id}/continuation`, and returns available `draft_contract` for ready Goals, `ask_user` with one open clarification request for incomplete Goals, or `blocked` for rejected/blocked states; the server endpoint also rejects OrganizationMembership / Goal organization mismatches before readiness mutation. `work answer` reads structured `question_id`-bound answer JSON from file/stdin after marker/login/org validation, calls the authenticated clarification continuation endpoint, and returns the next `goalrail.cli.v1` action after server-owned answer recording, allowed Goal hint application, and explicit readiness re-check. `contract draft` reads the same marker plus stored login profile, validates `/v1/me` organization membership before mutation, refreshes local Project Scan baseline/overlay evidence without uploading raw source bodies, sends local marker `project_id` and `repo_binding_id` expectations for server-side Goal context validation, calls authenticated create-or-return `/v1/contracts`, and returns a `goalrail.cli.v1` envelope with `contract_id`, `contract_state`, `local_repo_receipt`, and available `update_contract` only while the returned Contract is still `draft`. `contract update` reads structured proposed fields JSON from file/stdin, validates marker/login/org before mutation, sends marker `project_id` and `repo_binding_id` expectations to authenticated `PATCH /v1/contracts/{id}`, updates only current ContractDraft proposed fields, returns `changed_fields`, and yields `review_contract`. `contract submit` validates marker/login/org before mutation, sends marker project/repo expectations to authenticated `POST /v1/contracts/{id}/submissions`, moves a complete draft to `ready_for_approval`, and yields available `approve_contract`. `contract approve` fails before HTTP without `--confirm-user-approval`, validates marker/login/org when present, sends marker project/repo expectations to authenticated `POST /v1/contracts/{id}/approvals`, creates an ApprovedContract snapshot, and yields unavailable planned `plan_work`. `agent install` writes `.goalrail/agent/GOALRAIL.md` and `.goalrail/agent/commands.json`, may create root `AGENTS.md` only when missing, and is not a provider-specific adapter. The CLI does not claim hosted execution, production repo auth, real gate decisions, Organization selection UX, public Organization creation, broad repo binding sync, context-pack generation, proof retrieval, proof generation, provider-specific shim, Jira/Linear sync, local LLM ownership, planning, runner, or execution automation
- `apps/server` provides a verified Go server bootstrap plus authenticated repository-context init, authenticated metadata-only RepositoryContextSnapshot recording, authenticated metadata-only RepoBinding init, Postgres-backed source-neutral intake with Project / RepoBinding context validation, Goal promotion, deterministic Goal readiness state, authenticated bounded Goal continuation reconciliation, authenticated clarification answer continuation, authenticated create-or-return public Contract draft creation through `/v1/contracts`, authenticated public Contract update/submit/approve routes, durable ClarificationRequest / ClarificationAnswer storage, public `/v1/contracts` lifecycle facade, public Contract aggregate persistence, internal ContractSeed creation, internal ContractDraft creation/update/ready_for_approval, internal ApprovedContract approval, WorkItem plan/proposal/acceptance planning storage, Auth API and CLI code exchange, EventLog persistence, transactional canonical write + event append hardening, and explicit re-check-after-applied-answers when DB is configured; it creates repo-backed `Project(active)`, `RepoBinding(active, metadata_only)`, `RepositoryContextSnapshot`, `IntakeRecord`, non-executable `Goal`, open `ClarificationRequest`, recorded `ClarificationAnswer`, `Contract(seed/draft/ready_for_approval/approved)`, `ContractSeed(created)`, `ContractDraft(draft/ready_for_approval)`, `ApprovedContract(approved)`, `WorkItemPlan`, `WorkItemPlanProposal`, accepted `WorkItem(planned)` records, `UserSession`, and hashed `CLIAuthCode` records with S256 verifier challenge metadata only, updates RepoBinding workflow base branch and metadata, Goal readiness state, request answered state, Goal intent-plane hints, answer applied marker, Contract aggregate state/pointers, ContractDraft proposed fields, ContractDraft readiness state, plan state, proposal acceptance state, session state, and one-time CLI code consumption only, exposes single task read by ID, returns or creates exactly one open ClarificationRequest during continuation reconciliation after verifying active OrganizationMembership matches the Goal organization, records/applies clarification answers only after resolving ClarificationRequest -> Goal and verifying active OrganizationMembership matches the Goal organization, creates or returns one draft Contract only after verifying active OrganizationMembership matches a ready Goal organization and local marker expectations match Goal project/repo binding, updates/submits/approves public Contracts only after verifying active OrganizationMembership matches the Contract organization and supplied project/repo expectations match the Contract, derives update/submit/approval audit actors from the authenticated user on public routes, and does not claim automatic/background readiness re-check outside explicit answer/continuation/readiness calls, repo-aware planning computation, WorkItem assignment/claiming, execution, `Run`, receipt, gate, proof, repo readiness scoring, workers, provider integration, public Organization creation, or repository checkout
- `apps/web/pilot-intake-ru` provides a verified public RU business-first pilot landing for `ИИ-кодинг без хаоса`: it sells a safe 2-week пилот ИИ-разработки on one bounded product area, shows illustrative repository readiness / controlled task / pilot result cards with disclaimers, and keeps lead capture limited to `POST /api/pilot-lead` with local JSONL notification status, retry after `notification_failed`, in-flight `received` / `pending` rows blocked as duplicate submissions, duplicate suppression for notified / legacy processed / in-flight rows, no user-agent/IP/cookie/session/fingerprint tracking, a local JSONL purge command, `mailto:pilot@goalrail.dev` fallback, and visible Telegram channel `@goalrail`. The repo source for that narrow endpoint/digest is a landing-owned Go sidecar under `apps/web/pilot-intake-ru/server`, not the core `apps/server` API. Canonical copy and governance live in `docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md`; D-0055 demotes the previous 5-step technical walkthrough to internal / technical demo or checkpoint status; D-0047 boundaries remain intact except for D-0056's narrow lead-capture exception (no LLM/API, no repo provider integration, no code execution, no analytics or session tracking, no cookies, no sessions, no CRM, no Google Sheets, no broad backend platform, no chat UI, no file upload, no model selector, no real repository scan claim). The active target domain remains `pilot.goalrail.ru` per D-0053 with public path `/`; canonical metadata remains `https://pilot.goalrail.ru/`; SSH static deployment remains the hosting path per D-0051; the timestamped static release has been uploaded and `current` switched on the operator-managed server, live endpoint wiring uses the Go sidecar rather than PHP-FPM, and public DNS / HTTPS / `/api/pilot-lead` smoke passed.
- `apps/web/` remains a shared multi-resource namespace instead of a single runnable app surface
- repository community health and OSS baseline are explicit and inspectable
- next sales-pack, ContractContextPack, and runner-boundary slices remain
  explicit and bounded; local Project Scan baseline / overlay implementation
  exists while checkout jobs, checkout instructions, checkout receipts, provider
  integrations, runner implementation, gate, and proof remain unimplemented

## Main current risks

1. ops, offer, deck, and landing assets could drift away from the new concept canon
2. schema work could overgrow before the first compiling package exists
3. runtime adapter model could drift into vendor-specific code
4. execution parallelism and advisory parallelism could still leak into one implementation surface
5. MVP scope could widen into a generic agent or tooling platform too early
6. repository checkout could leak into the API server instead of staying behind runner boundaries
7. customer-hosted runner support could be treated as a late enterprise add-on instead of a first-class architecture mode
8. repository baseline or context-pack work could drift into hidden mutable memory, raw source upload, or background-scan truth
9. reference screenshots or brand assets could be relicensed accidentally without a provenance audit
