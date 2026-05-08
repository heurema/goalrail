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

Last updated: 2026-05-08
Status: planning / product canon and pilot frame active; first local Go CLI with
local Project Scan baseline / overlay commands and Go server intent-plane /
public Contract aggregate and `/v1/contracts` lifecycle façade / read-only
`/v1/qualification-feed` / authenticated read-only Contract discovery /
authenticated current ContractDraft detail /
ContractSeed / ContractDraft / ApprovedContract /
WorkItem persistence plus `plans` / `proposals` / `acceptance` WorkItem
planning control-plane flow exists; public
Contract aggregate identity is implemented as
a stable `contract_id` boundary and transitional public
seed/draft/approval/direct-task routes are removed; the typed WorkItemPlan pull
lease API from ADR-0021 is implemented without adding a generic queue;
ADR-0024's minimal `goalrail-worker` polling loop now exists as an API-only
prototype under `apps/worker`, and agent-facing proposal review / acceptance
now exists for explicit WorkItemPlanProposal acceptance into planned WorkItems;
compact CLI and server smoke coverage now pins the pull-loop through
`WorkItem(planned)`, and H1+ smoke coverage now pins checkout preparation
through runner checkout lease and `CheckoutReceipt`; H1 checkout preparation
now adds a server-owned checkout job / instruction, a minimal API-only
`goalrail-runner` checkout receipt loop under `apps/runner`, and bounded
runner-submitted workspace receipts without assignment, claiming, actual
clone/fetch, execution, `Run`, gate, or proof;
ADR-0029 now defines the H2 boundary, H2.1 implements the first code slice:
`ExecutionJob(queued)` can be created or returned from a planned WorkItem plus
CheckoutReceipt, and H2.1+ smoke coverage pins that preparation path while
H2.2 adds runner-scoped execution leases plus explicit `Run(started)` creation
with lease proof; H2.2+ smoke coverage now pins execution lease acquisition
through `Run(started)`; H2.3 now adds metadata-only `ExecutionReceipt`
submission with lease/run proof and no command execution, and H2.3+ smoke
coverage pins that receipt path; ADR-0030 defines the bounded command
execution boundary, and H2.4.1 now implements only the fixed
`builtin_diagnostic/workspace_status` command-plan plus command-metadata
receipt path while arbitrary shell, project commands, gate, and proof remain
deferred, and H2.4.1+ smoke coverage pins that builtin diagnostic receipt path
without changing the boundary; ADR-0031 now defines the H2.5 project command
execution boundary as typed, allowlisted, server-owned project command plans
only, with no shell, no arbitrary command strings, no user-provided argv, no
project test execution, one command receipt per Run, and Gate / Proof still
deferred;
runner-facing checkout and execution lease routes are bearer-authenticated
through the current active OrganizationMembership boundary, and lease
acquisition is scoped by requested project / repo binding before any job is
leased;
source-level core server CORS allowlist support exists through exact
`GOALRAIL_HTTP_CORS_ALLOWED_ORIGINS` values, with CORS disabled when unset and
wildcard origins rejected; the main console/API
deployment is now live through `11me/infra` Flux GitOps at
`https://goalrail.dev` and `https://api.goalrail.dev`, with Flux revision
`main@sha1:918c12936b03b469e3cb014a2c0ab119a850563e`, Kustomization
`flux-system/apps-personal` Ready=True, console/server rollouts successful,
public DNS/TLS ready, frontend/API smoke passed, `/start` live with SPA
fallback, and the console bundle built against `https://api.goalrail.dev`; the
same-origin `POST /api/start-chat` route is live through the separate
Cloudflare Worker from `apps/workers/start-assistant`, with public KB revision
`263075db460d762fe7fa1f09d30709bc68e8eb5c` and an operator-triggered public KB
sync workflow for future revisions; live API CORS is temporarily handled by
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
database, generic queue, repo-aware planning worker implementation, LLM/API
outside the bounded start-assistant Worker, repo integration, runtime
execution, gate, proof, or broad backend platform behavior.

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
- thirty-one kernel/CLI/server/domain boundary ADRs
- ops rails
- repo-tracked Goalrail and Punk overlay surfaces
- planned flow / eval structure
- reference screens
- shared web stack rules under `apps/web/`
- one canonical multilingual console source under `apps/web/console` with EN/RU static i18next resources, existing server auth endpoints for login, optional first-login password change, `/v1/me`, logout, in-memory tokens only, no cookies or token/profile/session browser-storage persistence, `goalrail.console.theme` as the only browser-storage key, no locale persistence, a Contracts surface that consumes read-only `GET /v1/contracts?limit=50` discovery by default, renders a compact contract rail/list with state and repo-binding filtering, manual refresh, selected aggregate detail through `GET /v1/contracts/{id}` plus current draft body through `GET /v1/contracts/{id}/current-draft` when linked, secondary manual contract-by-ID lookup, and a read-only Organization / Project / Repository context panel from `GET /v1/organizations/{organization_id}/repository-context` that prefers the selected Contract `repo_binding_id` match or otherwise shows the first Organization repository context / honest empty or missing-binding state, a Delivery Readiness surface polling read-only `GET /v1/qualification-feed` into Qualification / Clarification / Contract / Blocked lanes with one primary status per card, D-0091 display priority, calm browser-local timestamps, read-only clarification question text/context, no Goal/Contract workflow mutation controls, and linked-contract `Open contract` navigation through `GET /v1/contracts/{id}`, structured empty Proof surface, bottom-left Settings utility, Appearance theme picker, public English `/start`, API-backed Organization Users list/create/edit plus temporary-password reset using `/v1/me` organization context and the ADR-0027 Organization user-management routes, and read-only Settings / Repository metadata using `/v1/me` organization context plus `GET /v1/organizations/{organization_id}/repository-context`; selected Contract detail presents the public aggregate with one lifecycle status, linked ids, calm timestamps, and current draft title/body fields when available, while task, execution, gate, runner, and proof data remain unavailable in that view; temporary passwords are shown only from the immediate create/reset response and are not persisted in browser storage; Repository context data is metadata-only Project / RepoBinding visibility and does not claim provider authorization, checkout, readiness, proof, execution, or runner state; the main deployment is live at `https://goalrail.dev` with API base URL `https://api.goalrail.dev` through `11me/infra` Flux GitOps, while the old `apps/web/console-ru` workspace source has been removed and live `https://console.goalrail.ru/` remains separate
- a separate public-edge start assistant Worker under `apps/workers/start-assistant` owns live same-origin `POST https://goalrail.dev/api/start-chat`; it answers from the public KB revision `263075db460d762fe7fa1f09d30709bc68e8eb5c` through OpenAI Responses API file_search, has an operator-triggered GitHub Actions public KB sync path for future revisions, and keeps repo scan, file upload, code execution, analytics, cookies, sessions, CRM, browser OpenAI keys, chat history, and core `apps/server` ownership out of scope
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
- ADR-0028 documents the concrete runner checkout instruction and workspace
  receipt boundary for the next implementation slice: `WorkItem(planned)` may
  lead to a server-owned checkout job / instruction and a runner-submitted
  checkout receipt, while WorkItem state remains `planned` and assignment,
  claiming, execution, `Run`, gate, and proof remain deferred
- ADR-0029 documents the Run and execution receipt boundary for the next
  runtime slice: a server-owned `ExecutionJob` is the leaseable unit,
  `Run` is created only on runner start with lease proof, and execution
  receipts stay evidence inputs rather than `GateDecision` or `Proof`
- H2.1 implements the preparation-only `ExecutionJob(queued)` bridge from
  `WorkItem(planned)` plus `CheckoutReceipt`; H2.2 implements execution lease
  acquisition and explicit `Run(started)` creation with lease proof. Lease
  acquisition does not create `Run`; H2.2+ smoke coverage pins the
  `ExecutionJob(queued)` -> execution lease -> `Run(started)` path. H2.3 now
  accepts one metadata-only `ExecutionReceipt` for a started Run with explicit
  `lease_id` plus `lease_token` proof, including re-lease recovery for expired
  `run_started` jobs without receipts. H2.3+ smoke coverage pins this
  no-command receipt path. It does not execute commands, decide gates, or create
  proof.
- ADR-0030 documents the H2.4 bounded command execution boundary. H2.4.1
  implements only a server-authorized
  `ExecutionCommandPlan(builtin_diagnostic/workspace_status)` plus
  `ExecutionReceipt(builtin_diagnostic)` command-metadata path. It still keeps
  shell execution, user-provided command strings, project commands, provider
  adapters, LLM coding-agent integration, `GateDecision`, and `Proof` out of
  the current implementation. H2.4.1+ smoke coverage pins this path through a
  persisted command plan and builtin diagnostic receipt while keeping the
  one-receipt-per-run behavior explicit.
- ADR-0031 documents the H2.5 project command execution boundary. The next
  implementation should be `project_probe/detect_declared_test_targets` as a
  typed allowlisted command plan with `working_directory` and `path_scope`, no
  shell, no arbitrary command strings, no user-provided argv, no stdout/stderr
  capture, no artifacts, no changed paths, no project test execution, and one
  command receipt per Run. Project command execution is not implemented yet.
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
  planning boundary from `ApprovedContract(approved)`; the planning flow itself
  does not assign, claim, run checkout/execution, submit runtime receipts, gate,
  or proof
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
- ADR-0024 documents the minimal planning worker loop boundary: the first
  `goalrail-worker` prototype now lives under `apps/worker`, talks only to the
  API server, polls one plan lease, reads one plan, computes one deterministic
  development-mode proposal, and submits it with `lease_id` and `lease_token`.
  It is not a runner: no checkout, execution, direct Postgres writes, WorkItem
  creation, assignment/claiming, queue/outbox/runtime registry, `Run`, receipt,
  `GateDecision`, or `Proof`.
- ADR-0026 now extends the agent-facing pull loop through proposal review and
  explicit acceptance: `goalrail work plan status --plan-id` reads the current
  plan and submitted proposal for review, and
  `goalrail work proposal accept --proposal-id <proposal_id>
  --confirm-user-acceptance` accepts a submitted proposal into
  `WorkItem(planned)`. Acceptance derives `accepted_by` from the authenticated
  user, enforces Organization plus project/repo expectations before mutation,
  and still does not assign, claim, checkout, execute, create `Run`, submit
  receipts, write `GateDecision`, or create `Proof`. Compact smoke coverage
  now pins the happy path through `WorkItem(planned)` before runner work.
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
- The `goalrail init` stabilization sequence through INIT-07 is complete and
  recorded in `docs/ops/INIT_STABILIZATION_CHECKPOINT.md`. This is an
  operational checkpoint for bounded init behavior, marker safety, advisory
  snapshot / Project Scan warnings, retry context, and shared metadata-only
  repository-shape signal guardrails; it does not add server clone, source
  upload, repair command, runner, gate, proof, checkout, provider integration,
  schema/API/DB changes, or runtime execution behavior.
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
  active/inactive updates, last-active-owner protection, and self-action
  safety: self owner demotion, self membership deactivation, and self admin
  temporary-password reset are rejected, while self display-name edits and
  non-self resets for active or inactive users/memberships remain allowed.
  Settings / Users consumes these API-backed records, and there is no
  `goalrail users create` command.
- H1 checkout preparation now has checkout jobs, checkout instructions, and
  bounded checkout receipts. Actual runner clone/fetch, provider credential
  storage, VcsConnection, OAuth, provider client, gate, and proof remain
  unimplemented.
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
  control-plane flow with Postgres persistence when configured, while the
  planning flow itself does not assign, claim, run checkout/execution, submit
  runtime receipts, gate, or proof;
  the previous direct public task creation endpoint is removed

### Delivery model
- roadmap phases defined
- checkpoint model defined
- bounded slice workflow defined
- implementation discipline fixed: `punk`
- execution parallelism and advisory parallelism are separated conceptually
- kernel schema note and thirty-one boundary ADRs exist

### Repo structure
- the repo now mirrors `punk`-style planning boundaries
- `.goalrail/work/` is reserved for goals, reports, and bounded planning artifacts such as demo-planning packs
- `.goalrail/knowledge/` is reserved for advisory research and idea backlog
- `.punk/publishing.toml` remains the repo-local binding
- `.punk/publishing/` legacy directory has been removed from repo after manual copy/verify; the runtime publishing workspace lives in external user/platform-local storage
- `.punk/publishing.local.toml` is the ignored local-only manual-bootstrap pointer; resolver/runtime implementation is pending
- `.goalrail/flows/` and `.goalrail/evals/` exist as planned future structure, not executable product surfaces
- `apps/web/` is now the shared namespace for frontend resources and stack rules
- `apps/web/console` is the canonical multilingual EN/RU console source with real auth API login, optional first-login password change, `/v1/me`, logout, neutral internal role/status/surface IDs, runtime i18next language switching, no locale storage, an ops-style Contracts surface that consumes read-only `GET /v1/contracts?limit=50` discovery by default, renders a compact contract rail/list with state and repo-binding filtering, manual refresh, selected aggregate detail through `GET /v1/contracts/{id}` plus current draft body through `GET /v1/contracts/{id}/current-draft` when linked, and secondary explicit `contract_id` lookup, Delivery Readiness polling read-only `GET /v1/qualification-feed?limit=50` while authenticated and rendering Qualification / Clarification / Contract / Blocked lanes as read-only backend state with one primary status per card, D-0091 display priority, calm browser-local timestamps, read-only clarification question text/context, no Goal/Contract workflow mutation controls, linked-contract `Open contract` navigation through `GET /v1/contracts/{id}`, structured empty Proof surface, bottom-left Settings utility, Appearance theme picker, local-only theme preference under `goalrail.console.theme`, API-backed Organization Users list/create/edit using `/v1/me` organization context plus the ADR-0027 routes, read-only Settings / Repository Project + RepoBinding metadata backed by `GET /v1/organizations/{organization_id}/repository-context`, and public English `/start` backed by static guided fallback plus same-origin start assistant route; selected Contract detail presents the public aggregate with one lifecycle status, linked ids, calm timestamps, and current draft fields when available, not task, execution, gate, runner, or proof data; the main deployment is live at `https://goalrail.dev` and uses `https://api.goalrail.dev` through `11me/infra` Flux GitOps
- `apps/workers/start-assistant` is the separate public-edge Worker package for live `POST /api/start-chat`; it is not a core `apps/server` route and does not own canonical Goalrail product state
- `apps/web/demo-change-packet` is the current React + Vite + Mantine EN change-packet demo prototype, deployed through standalone infra at `demo.goalrail.dev`
- `apps/web/demo-change-packet-ru` is the separate RU copy of the change-packet demo prototype, deployed through standalone infra at `demo.goalrail.ru` rather than in-app i18n
- `apps/web/console-ru` source has been removed. The live `https://console.goalrail.ru/` deployment remains a separate legacy RU static release and is not migrated by the main `goalrail.dev` slice.
- `apps/web/pilot-intake-ru` is the current public React + Vite + Mantine RU business-first pilot landing for `pilot.goalrail.ru` (`ИИ-кодинг без хаоса`, safe 2-week пилот ИИ-разработки, repository readiness, project context, controlled tasks, verified result); it includes a narrow landing-owned Go sidecar under `apps/web/pilot-intake-ru/server` for lead capture and digest source, and it supersedes the previous technical interactive walkthrough as the primary public RU landing per D-0055.
- `apps/cli` is the first stdlib-only Go CLI bootstrap with canonical binary entrypoint `cmd/goalrail`
- CLI commands now exist for `version`, normal server-backed `goalrail init`, optional `goalrail init --base <branch>` workflow base override, low-level `goalrail init --project <project_id>`, explicit auth-free `goalrail init --local-demo`, explicit provider-neutral `goalrail agent install`, local `goalrail project scan/status`, server-backed `goalrail work start`, server-backed `goalrail work continue`, server-backed `goalrail work answer`, server-backed `goalrail work plan`, server-backed `goalrail work plan status`, server-backed `goalrail work proposal accept --confirm-user-acceptance`, server-backed `goalrail work checkout prepare`, server-backed `goalrail contract draft`, server-backed `goalrail contract update`, server-backed `goalrail contract submit`, server-backed `goalrail contract approve --confirm-user-approval`, `readiness scan`, `contract validate`, `proof show`, and the first `goalrail login <server_url>` browser-loopback auth path; normal `goalrail init` uses local Git metadata plus the stored login profile to call the server repository-context init endpoint, records a bounded metadata-only repository context snapshot on the server, writes a non-secret Git-root `.goalrail/project.yml` marker only after server success, ensures `.goalrail/.gitignore` for Goalrail-owned machine-local state, and runs a local Project Scan cache write, while `goalrail init --project <project_id>` remains the low-level Project-scoped RepoBinding init path; `goalrail agent install` writes `.goalrail/agent/GOALRAIL.md` and `.goalrail/agent/commands.json` with `work continue`, `work answer`, `contract draft`, `contract update`, `contract submit`, `contract approve`, `work plan`, `work plan status`, `proposal accept`, `checkout prepare`, question_id-bound answer guidance, structured contract field guidance, explicit user approval/acceptance guidance, checkout-preparation guidance, and local repository receipt guidance, may create a tiny root `AGENTS.md` shim only when missing, never overwrites an existing root `AGENTS.md`, and does not install provider-specific Codex, Claude, Gemini, Cursor, Windsurf, Gravity, gate, proof, readiness, Jira, or Linear automation
- The CLI also now exposes server-backed `goalrail work plan status --plan-id
  <plan_id>` for proposal review and
  `goalrail work proposal accept --proposal-id <proposal_id>
  --confirm-user-acceptance` for explicit proposal acceptance. Agent Pack v0
  includes both commands and tells agents not to infer plan acceptance from
  silence.
- The CLI also now exposes server-backed `goalrail work checkout prepare
  --task-id <task_id>` for checkout-job preparation. Agent Pack v0 includes the
  command and tells agents that `runner_checkout_required` needs a
  runner-submitted workspace receipt before any execution slice exists.
- `apps/server` is the first Go HTTP server bootstrap with canonical binary entrypoint `cmd/goalrail-server`
- server endpoints include `GET /livez`, `GET /readyz`, `GET /version`, `POST /v1/auth/login`, `GET /cli/login`, `POST /cli/login`, `POST /v1/auth/cli/exchange`, `POST /v1/auth/refresh`, `POST /v1/auth/change-password`, `POST /v1/auth/logout`, `GET /v1/me`, `GET /v1/organizations/{organization_id}/repository-context`, `GET /v1/organizations/{organization_id}/users`, `POST /v1/organizations/{organization_id}/users`, `PATCH /v1/organizations/{organization_id}/users/{user_id}`, `POST /v1/init/repository-context`, `POST /v1/repo-bindings/{repo_binding_id}/context-snapshots`, `POST /v1/projects/{project_id}/repo-bindings/init`, `POST /v1/intakes`, `GET /v1/intakes/{id}`, `POST /v1/intakes/{id}/goals`, `POST /v1/goals/{id}/readiness`, `POST /v1/goals/{id}/continuation`, `POST /v1/clarifications/{id}/answers/continuation`, `GET /v1/qualification-feed`, `POST /v1/goals/{id}/clarifications`, `POST /v1/clarifications/{id}/answers`, `POST /v1/answers/{id}/applications`, `POST /v1/contracts`, `GET /v1/contracts`, `GET /v1/contracts/{id}`, `GET /v1/contracts/{id}/current-draft`, `PATCH /v1/contracts/{id}`, `POST /v1/contracts/{id}/submissions`, `POST /v1/contracts/{id}/approvals`, `POST /v1/contracts/{id}/plans`, `GET /v1/plans/{id}`, `POST /v1/plans/leases`, `GET /v1/plans/leases/{id}`, `PATCH /v1/plans/leases/{id}`, `POST /v1/plans/{id}/proposals`, `GET /v1/proposals/{id}`, `POST /v1/proposals/{id}/acceptance`, and `GET /v1/tasks/{id}`; there is no full RepoBinding CRUD endpoint, `GET /v1/intakes`, `GET /v1/goals`, `GET /v1/plans`, `GET /v1/proposals`, `GET /v1/tasks`, or worker lease list/search endpoint, and the previous public `/v1/goals/{id}/contract-seeds`, `/v1/contract-seeds/{id}/contract-drafts`, `/v1/contract-drafts/{id}`, and direct `POST /v1/contracts/{id}/tasks` lifecycle/planning routes are no longer registered
- The server also exposes authenticated `POST /v1/plans/{id}/status` for
  agent-facing plan/proposal review with Organization plus project/repo
  expectation checks; the existing worker-facing `GET /v1/plans/{id}` and
  lease/proposal transport routes keep their current semantics.
- `POST /v1/contracts/{id}/plans` resolves `{id}` as stable public
  `contract_id`, requires bearer auth plus active OrganizationMembership,
  derives `requested_by` from the authenticated user, enforces Contract
  organization ownership and supplied project/repo expectations, requires the
  Contract to be `approved`, and creates or returns one server-owned
  `WorkItemPlan`; newly created plans start `queued`, without creating leases,
  proposals, WorkItems, Run, gate, or proof state; CLI output preserves
  `plan_state` and maps existing plan states to honest unavailable follow-up
  actions rather than always claiming queued planning
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
- `GET /v1/qualification-feed` is an authenticated read-only derived view for
  the intent / qualification stage. It scopes to the caller's active
  OrganizationMembership, supports optional `project_id`, `repo_binding_id`,
  `state` / `goal_state`, and `limit` filters, joins IntakeRecord, Goal,
  RepoBinding, open ClarificationRequest questions, and linked Contract
  id/state, maps items into qualification / clarification / contract /
  blocked lanes with explicit next actions, and does not recompute readiness,
  create clarification requests, create contracts, create plans, or write
  events.
- `GET /v1/contracts` is now an authenticated read-only Contract discovery
  endpoint consumed by the Console Contracts rail/list. It resolves the
  caller's active OrganizationMembership through the auth profile path, scopes
  results to that Organization, supports optional AND-combined `project_id`,
  `repo_binding_id`, `goal_id`, `state`, and `limit` filters, returns
  `{ "contracts": [...], "limit": n }` with default limit 50 and max 100, and
  orders stored Contract aggregates by updated time. It does not create
  ContractSeed, ContractDraft, ApprovedContract, event, readiness, plan, run,
  gate, or proof state.
- `GET /v1/contracts/{id}` is now an authenticated read-only public Contract
  detail endpoint consumed by selected Contract views. It resolves the caller's
  active OrganizationMembership through the auth profile path, requires active
  membership in the Contract's Organization, returns `membership_required` for
  inactive or missing membership, keeps `not_found` for missing Contract IDs,
  returns the existing `forbidden` behavior for other-Organization Contracts,
  and preserves the public `spine.Contract` JSON shape while hiding
  `organization_id` and `project_id`. It does not create or update
  ContractSeed, ContractDraft, ApprovedContract, event, readiness, plan, run,
  execution receipt, gate, or proof state.
- `GET /v1/contracts/{id}/current-draft` is an authenticated read-only
  current ContractDraft detail endpoint consumed by the Console selected
  Contract detail panel when the selected aggregate has `current_draft_id`.
  It resolves the caller's active OrganizationMembership through
  the auth profile path, reads the public Contract first, requires membership
  in the Contract's Organization, requires `current_draft_id`, reads that
  internal draft, verifies the draft belongs to the same Contract and
  Organization, and returns the existing public `spine.ContractDraft` JSON
  shape while hiding `organization_id` and `project_id`. It does not create,
  update, submit, approve, plan, execute, gate, prove, recompute readiness,
  create clarification requests, or write events.
- `apps/web/console` consumes `GET /v1/contracts?limit=50` on authenticated
  Contracts entry, renders compact rows with Contract id/state/Goal/RepoBinding
  and calm updated-time labels, supports an `all` / `draft` /
  `ready_for_approval` / `approved` / `seeded` state filter plus a
  repository-context-backed repo-binding filter and manual refresh, keeps
  existing visible rows on transient discovery errors, keeps selected detail
  visible when active filters exclude it, and keeps manual Contract ID lookup as
  a secondary authenticated, organization-scoped read-only fallback through
  `GET /v1/contracts/{id}`. Selected Contract detail now renders the current draft body through read-only
  `GET /v1/contracts/{id}/current-draft` when `current_draft_id` is present,
  shows "No current draft is linked yet" without calling that endpoint when the
  aggregate has no current draft, preserves the aggregate detail on draft
  `not_found` / `invalid_state`, and keeps the visible draft body on transient
  scheduled draft refresh errors where safe.
- `apps/web/console` consumes that feed from Delivery Readiness only while the
  user is authenticated and the surface is open. The polling path renders the
  stored lane/action snapshot and does not call `POST /v1/goals/{id}/continuation`,
  `POST /v1/clarifications/{id}/answers/continuation`, or `POST /v1/contracts`.
  Open clarification questions are rendered as read-only backend state. Linked
  Contract cards expose `Open contract` navigation only, loading selected
  Contract detail through existing read-only `GET /v1/contracts/{id}`.
- The qualification feed starts from promoted Goals, not received-only intakes.
  If `POST /v1/intakes` succeeds but promotion fails, that orphan IntakeRecord
  is currently treated as a CLI/server failure and is not visible in Console.
- Postgres-backed repository-context Project creation, RepoBinding init create, intake create, Goal promotion, Goal readiness, ClarificationRequest creation, ClarificationAnswer recording, answer application, ContractSeed creation, ContractDraft creation/update, ContractDraft ready_for_approval, and ApprovedContract approval writes now share a transaction with their expected event appends, so the durable canonical write does not commit without its audit events
- ClarificationRequest creation stores an open request durably when DB is configured, returns `503 database_not_configured` through production route wiring without DB, generates deterministic questions from Goal readiness reason codes, guards one open request per Goal, and appends `clarification.requested` transactionally with the request write in the Postgres path
- ClarificationAnswer recording stores canonical answer evidence durably when DB is configured, returns `503 database_not_configured` through production route wiring without DB, requires all questions answered, transitions the request from `open` to `answered`, and appends `clarification.answer_recorded` and `clarification.request_answered` through the configured EventLog
- answer application marks the persisted ClarificationAnswer as applied when DB is configured, updates persisted Goal intent-plane hints, rejects unsupported raw-text `goal.intent_owner` mapping, guards repeated application with `409 already_applied`, and appends events through the configured EventLog; it does not call readiness automatically
- `POST /v1/contracts` creates a public Contract lifecycle view from a ready Goal by creating internal `ContractSeed(created)` and `ContractDraft(draft)` records, returning Contract state `draft`, and not creating approval, tasks, execution, gate, or proof
- `PATCH /v1/contracts/{id}` updates the current internal draft's proposed fields through the public `contract_id`, now requires bearer authentication and server-side Organization ownership before mutation, accepts optional project/repo expectations, derives `updated_by` from the authenticated user for audit identity, rejects empty/blank update values, preserves `ContractDraft.state = draft`, appends `contract_draft.updated`, and does not approve or create tasks
- `POST /v1/contracts/{id}/submissions` transitions the current internal draft to `ready_for_approval`, moves Contract state to `ready_for_approval`, now requires bearer authentication and server-side Organization ownership before mutation, accepts optional project/repo expectations, derives `marked_by` from the authenticated user for audit identity, runs completeness checks, and does not approve Contract, create `WorkItem`, write `GateDecision`, or create `Proof`
- `POST /v1/contracts/{id}/approvals` creates an immutable internal `ApprovedContract(approved)` snapshot from the current ready draft, moves Contract state to `approved` with `approved_snapshot_id`, now requires bearer authentication and server-side Organization ownership before mutation, accepts optional project/repo expectations, derives `approved_by` from the authenticated user for audit identity, guards repeated approval with `409 already_approved`, and does not mutate `ContractDraft`, start execution, write `GateDecision`, or create `Proof`
- WorkItem planning now uses `Plan -> Lease -> Proposal -> Acceptance`: one `WorkItemPlan` per approved public Contract in v0, typed `WorkItemPlanLease(active/completed/expired)` records reserve queued or expired leased plans through `POST /v1/plans/leases`, proposal submission requires `lease_id` plus `lease_token`, explicit authenticated acceptance materializes one or more durable canonical `WorkItem(planned)` records with `plan_id` and `proposal_id`, derives `accepted_by` from the authenticated user, persists the records in Postgres when DB is configured, exposes `GET /v1/tasks/{id}` for single task reads, appends `work_item.created` for each accepted task transactionally with Postgres acceptance, and does not assign, claim, create `Run`, start execution, checkout a repository, submit a receipt, write `GateDecision`, or create `Proof`; workers/planners submit proposals through the API and do not write WorkItems directly to the DB
- the typed WorkItemPlan pull lease API is implemented with `POST /v1/plans/leases`, `GET /v1/plans/leases/{id}`, and `PATCH /v1/plans/leases/{id}`; raw lease tokens are returned only on create, stored only as hashes, and no generic queue implementation exists
- the minimal API-only `goalrail-worker` planning loop over typed leases exists
  under `apps/worker`; it does not imply checkout, execution, direct DB writes,
  WorkItem creation by the worker, assignment/claiming, queue/outbox/runtime
  registry, `Run`, receipt, `GateDecision`, or `Proof`
- the runner / repository checkout boundary is documented in ADR-0008, and
  ADR-0028 defines the concrete checkout instruction / workspace receipt
  boundary now implemented by H1 as a checkout-job plus workspace-receipt
  prototype; actual clone/fetch checkout, project command execution, gate, and
  proof remain deferred
- ADR-0029 defines the Run / execution receipt boundary, and H2.1 implements
  only the server-owned `ExecutionJob(queued)` preparation step from
  `WorkItem(planned)` plus `CheckoutReceipt`; H2.2 implements runner-scoped
  execution leases and explicit `Run(started)` creation with lease proof, with
  H2.2+ smoke coverage pinning that runtime transition. H2.3 implements
  metadata-only `ExecutionReceipt` submission for started Runs, and H2.3+
  smoke coverage pins that no-command receipt transition; receipts remain
  separate from Gate / Proof verdicts and do not claim command execution
- ADR-0030 defines the command execution boundary, and H2.4.1 implements the
  first server-authorized `builtin_diagnostic` command plan plus runner-owned
  `workspace_status` action, with no arbitrary shell or project command
  execution; H2.4.1+ smoke coverage pins this builtin diagnostic receipt path
  before any project command execution design
- ADR-0031 defines the project command execution boundary before H2.5 code:
  first project command execution must be typed, allowlisted, server-planned,
  scoped by `working_directory` / `path_scope`, and receipt-only evidence; shell,
  arbitrary command strings, user-provided argv, project test execution, Gate,
  Proof, WorkItem status transitions, and runner trust hardening stay deferred
- the `ClarificationAnswer` boundary is documented in ADR-0009; the answer application to Goal hints boundary is documented in ADR-0011, and clarification request/answer state is durable with Postgres when configured
- the explicit readiness re-check after applied answers boundary is documented in ADR-0012, and the existing readiness endpoint is verified to move an applied-answer Goal to `ready_for_contract_seed` without creating contract/work/gate/proof artifacts
- the `ContractSeed` boundary is documented in ADR-0013 and implemented as a Postgres-backed internal snapshot when DB is configured; there is no standalone public ContractSeed route, and the public `POST /v1/contracts` façade composes internal seed plus draft creation under one stable `contract_id`; standalone seed creation does not approve Contract, create `WorkItem`, write `GateDecision`, or create `Proof`
- the `ContractDraft` boundary is documented in ADR-0014 and implemented as a Postgres-backed draft creation boundary when DB is configured; it does not create approved Contract, `WorkItem`, `GateDecision`, or `Proof`
- the `ContractDraft` review/update boundary is documented in ADR-0015 and implemented as a draft-only update boundary; it does not introduce `ready_for_approval`, approved Contract, `WorkItem`, `GateDecision`, or `Proof`
- the `ContractDraft ready_for_approval` boundary is documented in ADR-0016 and implemented as an explicit `draft -> ready_for_approval` state transition with completeness checks and `marked_by` audit identity; it is not approval, approved Contract, `WorkItem`, execution, `GateDecision`, or `Proof`
- the Contract approval boundary is documented in ADR-0017 and implemented as `ContractDraft(ready_for_approval) -> ApprovedContract`; approval does not start execution, write `GateDecision`, or create `Proof`
- the WorkItem planning boundary is documented in ADR-0018 and ADR-0019 and implemented as a public `Plan -> Lease -> Proposal -> Acceptance -> WorkItem(planned)` control-plane flow with durable Postgres storage when configured and single task read by ID; a minimal API-only planning worker prototype exists, while worker controller, runner-backed planning, and execution-side implementation remain deferred; WorkItem planning is not assignment, claiming, execution, `Run`, runner checkout, receipt, `GateDecision`, or `Proof`
- the WorkItemPlan pull lease boundary is documented in ADR-0021 and implemented
  as a typed server API/persistence slice; ADR-0024 documents and `apps/worker`
  implements the minimal API-only planning worker loop, but no worker
  controller or runner-backed planning implementation exists yet
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
- no production runtime CLI beyond the `apps/cli` command foundation, first browser-loopback login, normal server-backed `goalrail init`, low-level `goalrail init --project <project_id>`, explicit auth-free `goalrail init --local-demo`, explicit provider-neutral `goalrail agent install`, marker-backed `goalrail work start`, marker-backed `goalrail work continue`, marker-backed `goalrail work answer`, marker-backed `goalrail contract draft`, marker-backed `goalrail contract update`, marker-backed `goalrail contract submit`, marker-backed `goalrail contract approve`, marker-backed `goalrail work plan`, and marker-backed `goalrail work execution prepare`
- no server integration for the CLI beyond `goalrail login <server_url>`, server-backed repository-context init, low-level server-backed RepoBinding init, `goalrail work start` using existing `/v1/me`, `/v1/intakes`, and `/v1/intakes/{id}/goals`, `goalrail work continue` using `/v1/me` plus `/v1/goals/{id}/continuation`, `goalrail work answer` using `/v1/me` plus `/v1/clarifications/{id}/answers/continuation`, `goalrail contract draft` using `/v1/me`, local Project Scan evidence, marker project/repo expectations, and authenticated create-or-return `/v1/contracts`, `goalrail contract update` using `/v1/me`, structured fields JSON, marker project/repo expectations, and authenticated `PATCH /v1/contracts/{id}`, `goalrail contract submit` using `/v1/me`, marker project/repo expectations, and authenticated `POST /v1/contracts/{id}/submissions`, `goalrail contract approve` using `/v1/me`, marker project/repo expectations, explicit `--confirm-user-approval`, and authenticated `POST /v1/contracts/{id}/approvals`, and `goalrail work plan` using `/v1/me`, marker project/repo expectations, and authenticated `POST /v1/contracts/{id}/plans`; `goalrail agent install` is local file installation only
- no server-owned canonical domain implementation beyond RepoBinding init, the persisted `IntakeRecord` / `Goal` / `ClarificationRequest` / `ClarificationAnswer` / read-only qualification feed / public Contract lifecycle façade / internal `ContractSeed` / `ContractDraft creation/update/ready_for_approval` / `ApprovedContract` / WorkItem planning plan/lease/proposal/acceptance slice yet
- no automatic/background readiness re-check outside explicit answer continuation or readiness/continuation calls; readiness reconciliation remains explicit through `goalrail work answer`, the readiness endpoint, or `goalrail work continue`
- no WorkItem assignment/claiming, arbitrary shell/project command execution,
  GateDecision, or Proof yet
- no worker controller, runner-backed planning implementation,
  assignment/claiming, arbitrary shell/project command execution, generic
  queue, outbox, broker, runtime registry, `GateDecision`, or `Proof` yet
- no production repo authorization or deploy-key provisioning in the CLI
- no broad RepoBinding state sync beyond repository-context read/init and explicit metadata-only init
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
- no runner registration, runner assignment, planning controller, actual
  clone/fetch checkout implementation, execution, gate, or proof yet
- no hosted runner pool implementation yet
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
- `apps/web/console` provides the verified canonical multilingual EN/RU console source with existing server login / first-login password change / `/v1/me` / logout plus an ops-style Contracts surface that consumes read-only `GET /v1/contracts?limit=50` discovery by default, renders a compact contract rail/list with state and repo-binding filtering, manual refresh, selected aggregate detail through `GET /v1/contracts/{id}` plus current draft body through `GET /v1/contracts/{id}/current-draft` when linked, and secondary explicit `contract_id` lookup, plus a Delivery Readiness surface that polls read-only `GET /v1/qualification-feed?limit=50` into Qualification / Clarification / Contract / Blocked lanes while authenticated with one primary status per card, D-0091 display priority, calm browser-local timestamps, read-only clarification question text/context, and linked-contract `Open contract` navigation through `GET /v1/contracts/{id}`; selected Contract detail presents the public aggregate with one lifecycle status, linked ids, calm timestamps, and current draft fields when available, while task, execution, gate, runner, and proof data remain unavailable in that view; tokens remain in React memory only, locale is not persisted, `goalrail.console.theme` remains the only browser storage key, Users renders `/v1/me` only, the main `https://goalrail.dev` deployment is live with API base URL `https://api.goalrail.dev`, and legacy `https://console.goalrail.ru/` remains separate; the console does not claim automatic continuation/recheck polling, automatic clarification creation, automatic clarification answer submission, automatic contract draft creation, Delivery Readiness Goal/Contract workflow mutation controls, durable user settings API, analytics, runner, gate, proof, repo integration, or product-loop implementation
- `apps/cli` provides a verified Go CLI bootstrap plus first `goalrail login <server_url>` server auth path with browser loopback, random state, S256 verifier/challenge exchange, normal server-backed `goalrail init` repository-context bootstrap, optional `goalrail init --base <branch>` workflow base override without Git mutation, low-level `goalrail init --project <project_id>`, explicit auth-free `goalrail init --local-demo`, explicit provider-neutral `goalrail agent install`, local `goalrail project scan/status` freshness commands, marker-backed `goalrail work start` with `--body-file <path|->`, marker-backed `goalrail work continue --goal-id <goal_id>`, marker-backed `goalrail work answer --clarification-request-id <id> --answers-file <path|->`, marker-backed `goalrail work plan --contract-id <contract_id>`, marker-backed `goalrail work execution prepare --task-id <task_id> --checkout-receipt-id <checkout_receipt_id>`, marker-backed `goalrail contract draft --goal-id <goal_id>`, marker-backed `goalrail contract update --contract-id <contract_id> --fields-file <path|->`, marker-backed `goalrail contract submit --contract-id <contract_id>`, and marker-backed `goalrail contract approve --contract-id <contract_id> --confirm-user-approval`; normal server-backed init records a bounded server-side metadata inventory snapshot after repository-context init, writes the non-secret Git-root `.goalrail/project.yml` repository marker after server success, ensures `.goalrail/.gitignore` for Goalrail-owned machine-local state, and runs a local Project Scan cache write, while `work start` reads that marker to create an IntakeRecord and Goal through existing server endpoints and returns a `goalrail.cli.v1` JSON envelope with `display.summary` plus an available continuation command. `work continue` reads the same marker plus stored login profile, validates `/v1/me` organization membership before mutation, calls authenticated `/v1/goals/{id}/continuation`, and returns available `draft_contract` for ready Goals, `ask_user` with one open clarification request for incomplete Goals, or `blocked` for rejected/blocked states; the server endpoint also rejects OrganizationMembership / Goal organization mismatches before readiness mutation. `work answer` reads structured `question_id`-bound answer JSON from file/stdin after marker/login/org validation, calls the authenticated clarification continuation endpoint, and returns the next `goalrail.cli.v1` action after server-owned answer recording, allowed Goal hint application, and explicit readiness re-check. `contract draft` reads the same marker plus stored login profile, validates `/v1/me` organization membership before mutation, refreshes local Project Scan baseline/overlay evidence without uploading raw source bodies, sends local marker `project_id` and `repo_binding_id` expectations for server-side Goal context validation, calls authenticated create-or-return `/v1/contracts`, and returns a `goalrail.cli.v1` envelope with `contract_id`, `contract_state`, `local_repo_receipt`, and available `update_contract` only while the returned Contract is still `draft`. `contract update` reads structured proposed fields JSON from file/stdin, validates marker/login/org before mutation, sends marker `project_id` and `repo_binding_id` expectations to authenticated `PATCH /v1/contracts/{id}`, updates only current ContractDraft proposed fields, returns `changed_fields`, and yields `review_contract`. `contract submit` validates marker/login/org before mutation, sends marker project/repo expectations to authenticated `POST /v1/contracts/{id}/submissions`, moves a complete draft to `ready_for_approval`, and yields available `approve_contract`. `contract approve` fails before HTTP without `--confirm-user-approval`, validates marker/login/org when present, sends marker project/repo expectations to authenticated `POST /v1/contracts/{id}/approvals`, creates an ApprovedContract snapshot, and yields available `plan_work`. `work plan` validates marker/login/org before mutation, sends marker project/repo expectations to authenticated `POST /v1/contracts/{id}/plans`, creates or returns one server WorkItemPlan with newly created plans starting queued, preserves returned `plan_state`, and maps queued, leased, proposal_submitted, accepted, and unknown states to honest unavailable follow-up actions. `work execution prepare` validates marker/login/org before mutation, sends marker project/repo expectations plus `checkout_receipt_id` to authenticated `POST /v1/tasks/{id}/execution-jobs`, creates or returns an `ExecutionJob(queued)`, and returns unavailable `runner_execution_required` without creating `Run` or executing commands. A CLI-level ADR-0026 pull-loop smoke fixture now covers `work start` -> `work continue` -> `work answer` -> `contract draft` -> `contract update` -> `contract submit` -> explicit `contract approve` -> `work plan` -> `work plan status` -> explicit `work proposal accept` through `WorkItem(planned)` -> `work checkout prepare` -> `work execution prepare` through `ExecutionJob(queued)` without assignment, claiming, runner lease, command execution, checkout receipt creation, `Run`, execution receipt, gate, or proof side effects. `agent install` writes `.goalrail/agent/GOALRAIL.md` and `.goalrail/agent/commands.json`, may create root `AGENTS.md` only when missing, and is not a provider-specific adapter. The CLI does not claim hosted execution, production repo auth, real gate decisions, Organization selection UX, public Organization creation, broad repo binding sync, context-pack generation, proof retrieval, proof generation, provider-specific shim, Jira/Linear sync, local LLM ownership, runner, or execution automation
- `apps/worker` provides the first minimal API-only `goalrail-worker` planning loop. It polls `POST /v1/plans/leases`, exits cleanly on no-work in `--once` mode, fetches the leased plan through `GET /v1/plans/{id}`, submits one deterministic development-mode proposal with `lease_id` plus `lease_token`, and keeps raw lease tokens out of logs and disk persistence. It uses local API DTOs only and does not import server internals, Postgres stores, or command execution packages. It is not a runner and does not checkout repositories, run commands, accept proposals, create WorkItems directly, assign or claim work, start `Run`, submit receipts, write `GateDecision`, create `Proof`, or add a queue/outbox/worker registry.
- `apps/runner` provides the first minimal API-only `goalrail-runner` checkout receipt loop plus H2.2 execution-start, H2.3 execution-receipt, and H2.4.1 execution-diagnostic loops. It uses `GOALRAIL_RUNNER_BEARER_TOKEN` for the current bearer-authenticated API boundary plus `GOALRAIL_RUNNER_PROJECT_ID` / `GOALRAIL_RUNNER_REPO_BINDING_ID` as an operator-declared lease scope. In default checkout mode it polls `POST /v1/checkout-jobs/leases`, exits cleanly on no-work in `--once` mode, receives a bounded checkout instruction, validates it against the requested repo-binding scope, and submits one lease-qualified workspace receipt through `POST /v1/checkout-jobs/{id}/receipts` with checkout lease proof and `raw_source_uploaded=false`. In `--mode execution-start`, it polls `POST /v1/execution-jobs/leases`, validates the scoped execution lease, and calls `POST /v1/execution-jobs/{id}/runs` to start `Run(started)` with lease proof. In `--mode execution-receipt`, it performs the same lease/run-start path and then submits one metadata-only `ExecutionReceipt` through `POST /v1/runs/{id}/receipts` with explicit `lease_id` plus `lease_token`, `execution_mode=no_command`, `process_status=not_executed`, empty artifact/path lists, and `raw_source_uploaded=false`; expired `run_started` jobs without receipts can be re-leased so receipt submission can recover after a transient runner/API failure. In `--mode execution-diagnostic`, it requests the server-owned `builtin_diagnostic/workspace_status` command plan for the started Run, validates that the plan forbids shell, argv, artifacts, and raw source upload, and submits a command-metadata `ExecutionReceipt` without calling `os/exec` or project commands. H1+ smoke coverage now pins the route-level and CLI-level checkout preparation path through runner lease and persisted `CheckoutReceipt`; H2.3+ smoke coverage pins execution-start plus no-command execution receipt submission without command execution; H2.4.1+ smoke coverage pins the builtin diagnostic command-plan plus receipt path. It uses local API DTOs only and does not import server internals or Postgres stores. It does not clone/fetch repositories in H1/H2.4.1, assign or claim WorkItems, run arbitrary shell or project commands, write `GateDecision`, create `Proof`, or add a queue/outbox/runtime registry.
- `apps/server` provides a verified Go server bootstrap plus authenticated repository-context init, authenticated metadata-only RepositoryContextSnapshot recording, authenticated metadata-only RepoBinding init, Postgres-backed source-neutral intake with Project / RepoBinding context validation, Goal promotion, deterministic Goal readiness state, authenticated bounded Goal continuation reconciliation, authenticated clarification answer continuation, authenticated create-or-return public Contract draft creation through `/v1/contracts`, authenticated read-only Contract discovery through `GET /v1/contracts`, authenticated read-only public Contract detail through `GET /v1/contracts/{id}`, authenticated read-only current draft detail through `GET /v1/contracts/{id}/current-draft`, authenticated public Contract update/submit/approve routes, durable ClarificationRequest / ClarificationAnswer storage, public `/v1/contracts` lifecycle facade, public Contract aggregate persistence, internal ContractSeed creation, internal ContractDraft creation/update/ready_for_approval, internal ApprovedContract approval, WorkItem plan/proposal/acceptance planning storage, ExecutionJob preparation storage, execution lease storage, Run(started) storage, metadata-only ExecutionReceipt storage, server-owned `ExecutionCommandPlan(builtin_diagnostic/workspace_status)` storage, Auth API and CLI code exchange, EventLog persistence, transactional canonical write + event append hardening, and explicit re-check-after-applied-answers when DB is configured; it creates repo-backed `Project(active)`, `RepoBinding(active, metadata_only)`, `RepositoryContextSnapshot`, `IntakeRecord`, non-executable `Goal`, open `ClarificationRequest`, recorded `ClarificationAnswer`, `Contract(seed/draft/ready_for_approval/approved)`, `ContractSeed(created)`, `ContractDraft(draft/ready_for_approval)`, `ApprovedContract(approved)`, `WorkItemPlan`, `WorkItemPlanProposal`, accepted `WorkItem(planned)` records, `ExecutionJob(queued/leased/run_started/receipt_submitted)`, `Run(started/receipt_submitted)`, `ExecutionCommandPlan(planned)`, `ExecutionReceipt(no_command/builtin_diagnostic)`, `UserSession`, and hashed `CLIAuthCode` records with S256 verifier challenge metadata only, updates RepoBinding workflow base branch and metadata, Goal readiness state, request answered state, Goal intent-plane hints, answer applied marker, Contract aggregate state/pointers, ContractDraft proposed fields, ContractDraft readiness state, plan state, proposal acceptance state, checkout job lease/receipt state, execution job preparation/lease/run-start/receipt state, Run receipt state, session state, and one-time CLI code consumption only, exposes single task read by ID plus read-only Contract list/detail discovery, returns or creates exactly one open ClarificationRequest during continuation reconciliation after verifying active OrganizationMembership matches the Goal organization, records/applies clarification answers only after resolving ClarificationRequest -> Goal and verifying active OrganizationMembership matches the Goal organization, creates or returns one draft Contract only after verifying active OrganizationMembership matches a ready Goal organization and local marker expectations match Goal project/repo binding, lists and reads public Contracts only after resolving an active OrganizationMembership and scoping to that Organization, updates/submits/approves public Contracts only after verifying active OrganizationMembership matches the Contract organization and supplied project/repo expectations match the Contract, scopes checkout and execution runner leases by authenticated OrganizationMembership plus requested project/repo binding before leasing, derives update/submit/approval audit actors from the authenticated user on public routes, has an ADR-0026 route-level smoke fixture through `WorkItem(planned)`, an H2.1+ smoke extension through runner checkout lease, persisted `CheckoutReceipt`, and `ExecutionJob(queued)`, an H2.2+ smoke extension through execution lease and `Run(started)`, an H2.3+ route-level smoke extension through metadata-only `ExecutionReceipt`, and an H2.4.1+ smoke extension through `ExecutionCommandPlan(builtin_diagnostic/workspace_status)` plus `ExecutionReceipt(builtin_diagnostic)` while keeping arbitrary shell/project command execution, gate, and proof absent, and does not claim automatic/background readiness re-check outside explicit answer/continuation/readiness calls, repo-aware planning computation, WorkItem assignment/claiming, arbitrary shell/project command execution, gate, proof, repo readiness scoring, worker platforms beyond apps/worker and apps/runner, provider integration, public Organization creation, actual repository clone/fetch, or arbitrary shell/project command execution
- `apps/web/pilot-intake-ru` provides a verified public RU business-first pilot landing for `ИИ-кодинг без хаоса`: it sells a safe 2-week пилот ИИ-разработки on one bounded product area, shows illustrative repository readiness / controlled task / pilot result cards with disclaimers, and keeps lead capture limited to `POST /api/pilot-lead` with local JSONL notification status, retry after `notification_failed`, in-flight `received` / `pending` rows blocked as duplicate submissions, duplicate suppression for notified / legacy processed / in-flight rows, no user-agent/IP/cookie/session/fingerprint tracking, a local JSONL purge command, `mailto:pilot@goalrail.dev` fallback, and visible Telegram channel `@goalrail`. The repo source for that narrow endpoint/digest is a landing-owned Go sidecar under `apps/web/pilot-intake-ru/server`, not the core `apps/server` API. Canonical copy and governance live in `docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md`; D-0055 demotes the previous 5-step technical walkthrough to internal / technical demo or checkpoint status; D-0047 boundaries remain intact except for D-0056's narrow lead-capture exception (no LLM/API, no repo provider integration, no code execution, no analytics or session tracking, no cookies, no sessions, no CRM, no Google Sheets, no broad backend platform, no chat UI, no file upload, no model selector, no real repository scan claim). The active target domain remains `pilot.goalrail.ru` per D-0053 with public path `/`; canonical metadata remains `https://pilot.goalrail.ru/`; SSH static deployment remains the hosting path per D-0051; the timestamped static release has been uploaded and `current` switched on the operator-managed server, live endpoint wiring uses the Go sidecar rather than PHP-FPM, and public DNS / HTTPS / `/api/pilot-lead` smoke passed.
- `apps/web/` remains a shared multi-resource namespace instead of a single runnable app surface
- repository community health and OSS baseline are explicit and inspectable
- next sales-pack, ContractContextPack, and execution-boundary slices remain
  explicit and bounded; local Project Scan baseline / overlay implementation
  exists and H1 checkout job / checkout receipt preparation exists while actual
  clone/fetch checkout, provider integrations, execution, gate, and proof remain
  unimplemented

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
10. project command execution could accidentally turn stdout/stderr, artifacts,
    or multiple command attempts into pseudo-proof unless ADR-0031's typed
    allowlist and one-command / one-receipt / one-Run rule are preserved
