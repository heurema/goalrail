# Goalrail Next

> Only the next bounded slices. Keep this file short.

## Active phase

- **Phase 1 canonical multilingual console source integration complete; main console/API routing is live via infra; post-PR-#120 CORS cleanup remains**
- product and deployment canon is now in place
- repo overlay structure now keeps Goalrail artifacts in `.goalrail/` and Punk publishing artifacts in `.punk/`
- `apps/web/` now exists as the shared namespace for frontend resources
- `apps/web/console` is now the single canonical multilingual EN/RU console source with static i18next resources, existing server login / optional password-change / `/v1/me` / logout endpoints, in-memory tokens only, no locale storage, and `goalrail.console.theme` as the only browser-storage key; the main deployment is live at `https://goalrail.dev` with API base URL `https://api.goalrail.dev` through `11me/infra` Flux GitOps, while the old `apps/web/console-ru` workspace source is removed and live `console.goalrail.ru` remains separate
- `apps/web/demo-change-packet` and `apps/web/demo-change-packet-ru` are separate EN/RU demo resources with independent domains; future web work should follow `apps/web/<resource>`
- `apps/web/pilot-intake-ru` now targets a business-first RU pilot landing for `ИИ-кодинг без хаоса`: a mostly static Founding Pilot page for a safe 2-week пилот ИИ-разработки on one bounded product area, with repository readiness, project context, controlled tasks, verified result, a D-0056 minimal `POST /api/pilot-lead` endpoint with duplicate suppression, D-0059 Resend HTTPS notification transport when configured, and direct `mailto:` fallback. D-0055 supersedes the previous technical interactive walkthrough as the primary public RU landing; that walkthrough is demoted to internal / technical demo or checkpoint status in git history. D-0047 boundaries remain in full except for the narrow D-0056 lead-capture endpoint (no analytics, tracking, CRM, Google Sheets, cookies, sessions, LLM/API, repo integration, code execution, broad backend platform, chat UI, file upload, model selector, or real repository scan claim). Active target domain remains `pilot.goalrail.ru` per D-0053; SSH static hosting remains the path per D-0051; server upload, operator-managed Go sidecar endpoint wiring, server-side TLS provisioning, public DNS verification, public HTTPS smoke, and public `/api/pilot-lead` smoke are complete.
- `apps/server` now exists as a Go server bootstrap with health/version endpoints plus authenticated repository-context init, authenticated metadata-only RepoBinding init, Postgres-backed source-neutral intake, Project / RepoBinding context validation for intake, Goal promotion, Goal readiness state, ClarificationRequest / ClarificationAnswer storage, ContractSeed creation, ContractDraft creation/update/ready_for_approval, ApprovedContract approval, WorkItem plan/proposal/acceptance planning storage, durable EventLog persistence, transactional canonical write + event append hardening, explicit re-check, and exact-origin CORS allowlist support for the `goalrail.dev` -> `api.goalrail.dev` browser API split; the live server image still predates that app-level CORS code, so infra currently keeps nginx ingress CORS as a temporary bridge; future server work should stay bounded and avoid fake canonical state claims
- ADR-0008 now defines the runner and repository checkout boundary; future repository checkout/check work must happen behind runners, not inside the API server
- ADR-0009 now defines the ClarificationAnswer recording boundary; future answer work must record evidence before Goal hint application or readiness re-check
- ADR-0010 now defines the MVP Organization / Project / RepoBinding and persistence bootstrap boundary; future persistence work should keep direct RepoBinding before RepositoryRecord
- ADR-0011 now defines answer application to Goal hints; the server keeps readiness re-check separate and persists clarification request/answer state with Postgres when configured
- ADR-0012 defines explicit readiness re-check after applied answers, and the server verifies that the existing readiness endpoint can move an applied-answer Goal to `ready_for_contract_seed` without creating contract seed
- ADR-0013 now defines the `ContractSeed` boundary, and the server persists `ContractSeed(created)` in Postgres when DB is configured; future contract work must keep approval, work item, gate, and proof as later boundaries
- ADR-0014 now defines the `ContractDraft` boundary, and the server persists `ContractDraft(draft)` creation in Postgres when DB is configured; future contract work must keep approval, work item, gate, and proof as later boundaries
- ADR-0015 now defines the `ContractDraft` review/update boundary, and the server can update proposed draft fields while keeping state `draft`; approval remains a later boundary
- ADR-0016 now defines the `ContractDraft ready_for_approval` boundary, and the server implements it as an explicit `draft -> ready_for_approval` transition with completeness checks and `marked_by` audit identity; approval, approved Contract, work item, gate, and proof remain later boundaries
- ADR-0017 now defines the Contract approval boundary from `ContractDraft(ready_for_approval)` to `ApprovedContract`; the server implements it as explicit ApprovedContract snapshot creation with `approved_by` and `contract.approved`; approval does not start execution, gate, or proof
- ADR-0018 now defines the WorkItem planning boundary from `ApprovedContract(approved)` to `WorkItem(planned)`; WorkItems remain non-executable while assignment, claiming, execution, Run, receipt, gate, and proof remain later boundaries
- ADR-0019 now qualifies WorkItem planning with a Kubernetes-style control-plane split: the API server owns canonical state and accepted WorkItems, while repo-aware planning computation belongs behind worker / controller / runner boundaries; the public `plans` / `proposals` / `acceptance` API has landed, while worker/controller/runner execution-side implementation remains deferred
- ADR-0020 now defines the public Contract identity boundary: public API should use one stable `Contract` aggregate and `contract_id`, while `ContractSeed`, `ContractDraft`, and `ApprovedContract` remain internal lifecycle records; the server now implements the smallest aggregate/store/linkage boundary and public `/v1/contracts` lifecycle façade routes
- ADR-0021 now defines the typed WorkItemPlan pull lease boundary: future planning workers should create `WorkItemPlanLease` reservations through the API server using `POST /v1/plans/leases`; `WorkItemPlan(state=queued)` remains the typed planning queue item, no generic queue platform is accepted, and the lease API is not implemented yet
- ADR-0022 now defines the Installation boundary above Organization:
  `Installation` is the concrete running Goalrail control plane / instance,
  Organization remains the tenant/workspace boundary, `self_hosted` and `saas`
  are the only deployment modes, and the server now has the smallest
  Installation schema foundation: `installations`, installation-scoped
  organization slugs, and a dev `self_hosted` Installation linked to the dev
  Organization. The later ADR-0023 auth API slices now add login, refresh,
  password change, logout, `/v1/me`, and the smallest CLI browser-loopback
  login code exchange; SaaS onboarding and organization creation API remain
  unimplemented. The current server-rendered `/cli/login` page is a temporary
  CLI auth bridge only, not the product web console login UI; the canonical
  `apps/web/console` source now consumes the server auth API directly, and the
  main `https://goalrail.dev` deployment routes it to
  `https://api.goalrail.dev`; legacy static deployments remain separate.
- ADR-0023 now defines the user bootstrap, auth, and CLI login boundary:
  self-hosted bootstrap creates the first product super admin as
  `OrganizationMembership(owner)`, public registration is out of MVP, admins
  create users with backend-generated temporary passwords, first-login password
  change is required, email invite/reset delivery is deferred, password
  credentials should live outside `users`, access tokens should be short-lived
  JWTs, refresh tokens should be opaque DB-backed server state, JWTs should not
  carry broad/stale permission state, server-side role checks use
  `OrganizationMembership`, and `goalrail login` uses explicit `server_url`
  plus browser localhost loopback. The smallest credential/session foundation
  now exists as schema, server-local types, Squirrel-backed store primitives,
  and Argon2id PHC-style password hashing/verification, and
  `goalrail-server bootstrap owner` now implements the smallest self-hosted
  owner bootstrap command. The smallest auth API lifecycle now exists for
  login, refresh, password change, logout, current-user profile, and CLI code
  exchange with S256 verifier/challenge binding; the CLI now stores token
  metadata locally after browser-loopback login. `apps/web/console` now
  implements multilingual login, first-login password change, current profile
  lookup, and logout over those existing endpoints. Organization / Project /
  RepoBinding profile selection remains unimplemented; the server-rendered CLI
  auth bridge remains separate.
- ADR-0027 defines the Organization user management boundary:
  regular Organization users are a Console-backed server API workflow,
  not CLI user creation; canonical identity is `User`, Organization access is
  `OrganizationMembership`, v0 user management is owner-only, temporary
  passwords are backend-generated and shown once, first-login password change
  remains mandatory, roles are `owner`, `admin`, `member`, and `viewer`, and
  there is no separate CLI-user entity or `goalrail users create` command. The
  backend now implements `GET /v1/organizations/{organization_id}/users`,
  `POST /v1/organizations/{organization_id}/users`, and
  `PATCH /v1/organizations/{organization_id}/users/{user_id}` with owner-only
  v0 authorization, server-side membership loading, cross-organization
  rejection, one-time temporary password generation for newly created users,
  safe attachment of already existing active users that are not yet members of
  the target Organization without credential rotation, membership-scoped
  active/inactive updates, and last-active-owner protection.
- Next bounded Organization user-management implementation slices:
  1. console Users API-backed persistence replacing component-state Users
     data, without storing temporary passwords or tokens in browser storage
  2. ops docs update after console persistence lands to align STATUS, NEXT,
     COMPONENTS.yaml, and console docs with actual code reality
- Repository access MVP is reset to RepoBinding context plus runner-owned
  local credentials. RepoBinding remains canonical repository context and not
  permission to clone; the API server stores no repository secrets in the MVP.
- Next bounded docs slice: runner-owned repository checkout credential
  boundary. It should define runner startup flags, API-issued
  `CheckoutInstruction`, `CheckoutReceipt`, and supported credential modes:
  Git HTTPS token file, SSH key file, and mounted workspace.
- This cleanup adds no provider OAuth, VcsConnection, token storage, provider
  clients, live metadata listing, checkout implementation, runner
  implementation, gate, or proof.
- the next slices should use those overlay boundaries instead of adding ad hoc top-level storage
- `apps/server` product/auth APIs now require structured Postgres database
  configuration for durable state; health/version stay available without DB,
  while product/auth routes return `503 database_not_configured` when DB config
  is absent. Postgres-backed stores are the only real server persistence
  implementation; obsolete map-backed server stores were removed from
  `apps/server/internal/store`, the old in-memory event log helper was removed,
  and tests use package-local fakes where needed.
- `GOALRAIL_HTTP_CORS_ALLOWED_ORIGINS` is the source-level CORS knob for
  browser access from exact frontend origins such as `https://goalrail.dev` or
  local `http://localhost:5173`; empty means disabled and wildcard origins are
  rejected. The main deployment currently uses nginx ingress CORS as a
  temporary bridge because the pinned live server image predates the app-level
  CORS code. The cleanup slice should pin a post-PR-#120 server image, enable
  app-level CORS, and remove nginx ingress CORS annotations in the same infra
  PR.

## Stabilization tranche — source-of-truth and public-surface hardening

Status: **COMPLETE repo-side through D-0065**.

This tranche was the immediate stabilization priority before new feature
slices. It was not a product expansion, did not change MVP scope, and did not
approve new runtime, analytics, CRM, repo integration, LLM/API, runner, gate,
proof, or broad backend claims.

Completed slices:

A. Documentation source-of-truth alignment — complete; landing canon,
   stabilization decision, AGENTS, README, STATUS, and NEXT are aligned.
B. Pilot lead capture reliability patch — D-0061 complete; local JSONL
   notification status keeps retry-after-failure possible without concurrent
   duplicate notification attempts.
C. Pilot lead runtime migration to Go sidecar — D-0062 complete; transitional
   PHP source was removed from active repo runtime under
   `apps/web/pilot-intake-ru/server` without changing the endpoint path, MVP
   scope, or public landing boundary.
D. Repo checks CI for Go + web surfaces only — D-0063 complete; PR checks cover
   current Go modules and `apps/web` workspaces only, with no PHP and no
   deployment automation.
E. Branch protection / PR-only governance checklist — D-0064 complete; `main`
   requires current repo checks through verified GitHub branch protection, with
   no required human review or signed-commit requirement added in this slice.
F. PII / abuse / retention guardrails for pilot lead capture — D-0065
   complete; new rows omit user-agent, local JSONL purge is available with
   dry-run default, and reverse-proxy rate limiting remains operator-managed.
G. Root onboarding surface map cleanup — complete / no further action from this
   tranche; README, README.ru, and AGENTS already reflect the current repo
   surfaces and non-implemented boundaries.

## Completed deployment/live slice

### Operator-managed Go sidecar deployment and public DNS/live smoke

Status: **DONE — LIVE VIA SSH STATIC SERVER / SMOKE PASSED.**

Goal:
- migrate the operator-managed public RU pilot lead endpoint from the earlier
  PHP-FPM wiring to the repo-side Go sidecar, then correct public DNS and run
  HTTPS plus `/api/pilot-lead` live smoke.

Done means:
- the Go sidecar is deployed and wired by the operator outside repo source
  control; no server hostnames, IPs, ports, credentials, concrete
  reverse-proxy config, deployment scripts, or secrets are committed
- public DNS for `pilot.goalrail.ru` reaches the operator-managed server
- public HTTPS smoke and public `/api/pilot-lead` smoke pass
- ops docs are updated only after verification

Current truth:
- repo-side Go sidecar source exists under `apps/web/pilot-intake-ru/server`
- operator-managed public server migration to the Go sidecar is complete
  outside repo source control
- public DNS, HTTPS smoke, and public `/api/pilot-lead` smoke passed
- no server config, deployment scripts, secrets, hostnames, IPs, ports,
  usernames, key paths, or DNS provider credentials were committed

### Main console/API Flux GitOps deployment and public smoke

Status: **DONE — LIVE VIA `11me/infra` FLUX GITOPS / SMOKE PASSED.**

Goal:
- route the canonical `apps/web/console` frontend to `https://goalrail.dev`
  and the `apps/server` API to `https://api.goalrail.dev` through the external
  `11me/infra` Flux GitOps path.

Done means:
- Flux reconciled infra revision
  `main@sha1:f4cb3db22853d0d92291f37acb055cd28e8abec7`
- Flux Kustomization `flux-system/apps-personal` reported `Ready=True`
- `goalrail-console` and `goalrail-server` rollouts completed successfully
- `goalrail.dev` and `api.goalrail.dev` resolved publicly
- `goalrail-dev-tls` and `api-goalrail-dev-tls` reported `Ready=True`
- frontend HTTP 200 smoke passed with no `Set-Cookie`
- API `/livez`, `/readyz`, and `/version` smoke passed
- frontend HTML/bundle contained no `console.goalrail.dev`, and the bundle
  contained `https://api.goalrail.dev`
- API CORS preflight for `Origin: https://goalrail.dev` returned HTTP 204 with
  allowed methods `GET, POST, PATCH, OPTIONS` and headers
  `Authorization, Content-Type`

Current truth:
- live status and smoke evidence are recorded in
  `docs/ops/CONSOLE_MAIN_DEPLOYMENT_WIRING.md`
- console source remains `apps/web/console`; API source remains `apps/server`;
  deployment source of truth remains the external `11me/infra` repo
- demo sandbox `https://demo.goalrail.dev` remains separate
- legacy `https://console.goalrail.ru/` remains separate and is not migrated by
  this slice
- current live API CORS is a temporary nginx ingress bridge because the pinned
  live `goalrail-server` image predates the app-level CORS implementation from
  Goalrail PR #120
- the later cleanup is to pin a post-PR-#120 server image, enable
  `GOALRAIL_HTTP_CORS_ALLOWED_ORIGINS=https://goalrail.dev`, and remove nginx
  ingress CORS annotations in the same infra change
- no kubeconfig, secrets, credentials, private hosts/IPs, SSH details, concrete
  reverse-proxy snippets, deployment scripts, analytics, CRM, runner, gate,
  proof, or product data loop were introduced or recorded

## Completed web bounded slice

### Console RU static deployment and public HTTPS smoke

Status: **DONE — LIVE VIA SSH STATIC SERVER / SMOKE PASSED.**

Goal:
- publish the static RU console shell at `https://console.goalrail.ru/` next to
  the existing RU pilot static surface without adding backend behavior.

Done means:
- `apps/web/console-ru` was tested and built locally
- `dist/` was uploaded to `/srv/goalrail/console-ru/releases` through the
  operator-provided deploy SSH target
- `/srv/goalrail/console-ru/current` was switched to the timestamped release
- a static-only Nginx vhost and server-side TLS were configured outside repo
  source control
- public HTTPS smoke passed for `https://console.goalrail.ru/`
- console-specific Certbot renewal dry-run passed
- no server config, deployment scripts, secrets, hostnames, IPs, ports,
  usernames, key paths, DNS provider credentials, backend routes, analytics,
  cookies, sessions, LLM/API, repo integration, runtime execution, gate, proof,
  or product data loop were committed or introduced

Current truth:
- live status and smoke evidence are recorded in
  `docs/ops/CONSOLE_RU_DEPLOYMENT_WIRING.md`
- the live console remains a static visual shell with no `/v1/*` backend route;
  canonical repo source now lives in `apps/web/console`, and deployed auth
  still needs a separate Phase 3 routing/proxy/CORS plus deployment migration
  slice
- Users changes are component state only and are not persisted
- a whole-host Certbot renewal dry-run surfaced an unrelated
  `pilot.goalrail.ru` renewal dry-run failure; console-specific renewal passed,
  and pilot renewal should be handled as a separate operator follow-up

## Completed frontend bounded slice

### Canonical console multilingual repo-side auth flow

Status: **DONE — SOURCE ONLY / NOT LIVE-ROUTED.**

Goal:
- consolidate the split EN/RU console sources into `apps/web/console`, replacing
  fake local login with a bounded frontend auth flow over the existing
  `apps/server` auth API.

Done means:
- `apps/web/console` has a typed fetch auth client for
  `POST /v1/auth/login`, `POST /v1/auth/change-password`, `GET /v1/me`, and
  `POST /v1/auth/logout`
- EN/RU resources are statically bundled through `react-i18next` / `i18next`,
  with runtime language switching and no locale browser storage
- `apps/web/console-ru` has been removed as a workspace source
- successful login enters the console only after `/v1/me` succeeds
- bootstrapped users with `must_change_password = true` are routed through a
  password-change form before console entry
- logout calls the server and clears local in-memory auth state even if the
  request fails
- access and refresh tokens are kept in React memory only; no cookies, token
  `localStorage`, token `sessionStorage`, or profile browser persistence exists
- `goalrail.console.theme` remains the only accepted localStorage key
- Settings -> Users remains component-state only and does not call the backend
  admin user-management API
- no public registration, signup, SSO, invite/reset email, password reset,
  SaaS onboarding, organization creation API, analytics, repo integration,
  runner, gate, proof, CORS, deployment
  config, hostnames, IPs, ports, credentials, reverse-proxy snippets, or
  secrets were added
- live `https://goalrail.dev` now uses this repo-side auth source through the
  external `11me/infra` Flux GitOps path; live `https://console.goalrail.ru/`
  remains a separate legacy RU static release

## Completed backend bounded slice

### Self-hosted bootstrap owner command

Status: **DONE — smallest ADR-0023 owner bootstrap command exists.**

Goal:
- implement the smallest flag-based self-hosted owner bootstrap path on top of
  the Installation and auth credential foundations.

Done means:
- ✅ `goalrail-server bootstrap owner` exists
- ✅ required flag input covers owner email/display name, organization slug/name,
  and public base URL
- ✅ one `self_hosted` Installation is created or reused with normalized
  `public_base_url`
- ✅ one primary Organization is created or reused under that Installation
- ✅ the first owner User is created or reused
- ✅ `OrganizationMembership(owner)` is created or updated for that User
- ✅ a temporary password is generated with `crypto/rand`, hashed through the
  existing Argon2id package, stored in `user_password_credentials`, and marked
  `must_change_password = true`
- ✅ existing owner password credentials are not silently rotated
- no login endpoint
- no JWT implementation
- no refresh/logout endpoint
- no CLI `goalrail login`
- no web UI
- no SaaS onboarding
- no organization creation API
- no billing
- no SSO/OIDC
- no runner, gate, proof, or generic queue work

## Completed backend bounded slice

### Auth credential foundation

Status: **DONE — smallest ADR-0023 credential/session foundation exists.**

Goal:
- implement the smallest server persistence foundation for the documented
  ADR-0023 auth/bootstrap boundary after the Installation schema foundation,
  plus password hashing/verification primitives explicitly scoped to this
  slice.

Done means:
- ✅ `user_password_credentials` exists as a dedicated password credential
  table outside `users`
- ✅ `user_sessions` exists as an opaque DB-backed refresh-token/session store
- ✅ first-login password-change state is representable
- ✅ refresh token/session state is server-owned and revocable
- ✅ server-local password credential, user session, and session-state types
  exist
- ✅ Argon2id PHC-style password hashing and verification primitives exist
- ✅ Squirrel-backed store primitives can upsert and look up credentials and
  sessions
- role checks remain server-side through `OrganizationMembership` for later
  auth API work
- no login endpoints
- no JWT implementation
- no CLI changes
- no web UI
- no SaaS onboarding
- no organization creation API
- no billing
- no SSO/OIDC
- no runner, gate, proof, or generic queue work

## Completed backend bounded slice

### CLI browser-loopback login slice

Status: **DONE — smallest `goalrail login <server_url>` auth path exists.**

Goal:
- implement the first real CLI login path on top of the existing server auth
  lifecycle and Postgres-only persistence boundary, without adding a general
  web UI or selected Organization / Project / RepoBinding profile.

Done means:
- ✅ `goalrail login <server_url>` validates an explicit http(s) server URL
- ✅ the CLI starts a `127.0.0.1` loopback callback listener on a random port
- ✅ the CLI generates random `state` and `code_verifier`
- ✅ the CLI opens the server `/cli/login` URL with an S256 `code_challenge` or
  prints it with `--no-browser`
- ✅ `GET /cli/login` renders a minimal CLI-only HTML login page
- ✅ `POST /cli/login` verifies existing email/password credentials through the
  auth service, rejects credentials still marked `must_change_password`,
  validates localhost loopback redirect targets, creates a short-lived one-time
  auth code stored by hash with S256 challenge metadata, and redirects with only
  `code` and `state`
- ✅ the `/cli/login` HTML page is a temporary server-rendered CLI auth bridge
  only; it is not the product web console login UI
- ✅ `POST /v1/auth/cli/exchange` requires the matching `code_verifier`,
  consumes a valid unused code once, and returns normal access/refresh token
  metadata
- ✅ the CLI stores `server_url`, `access_token`, `refresh_token`,
  `access_token_expires_at`, and `token_type` in a local auth JSON file with
  0600 permissions
- ✅ normal `goalrail init` calls the server repository-context init endpoint
  using local Git metadata and the stored login profile, then prints the
  server-owned Project and RepoBinding context
- ✅ normal `goalrail init` records a bounded metadata-only repository context
  snapshot on the server after repository-context init; this is inventory only,
  not readiness/audit scoring
- ✅ `goalrail init --base <branch>` can set `workflow_base_branch` explicitly
  without creating branches or changing Git state
- ✅ low-level `goalrail init --project <project_id>` still calls the
  Project-scoped RepoBinding init endpoint
- ✅ explicit `goalrail init --local-demo` preserves the auth-free local/demo
  draft and writes no files
- ✅ after successful server-backed init, the CLI writes the non-secret
  Git-root `.goalrail/project.yml` repository marker with server/project/repo
  binding identity only and ensures `.goalrail/.gitignore` ignores
  Goalrail-owned machine-local state directories/files
- ✅ server-backed init preflights an existing `.goalrail/project.yml` before
  the server call and fails locally on server/project/repo/base conflicts
- ✅ `goalrail work start --title <title> [--body <body> | --body-file
  <path|->]` reads the Git-root marker plus stored login profile, calls
  `/v1/me`, creates `/v1/intakes`, and promotes through
  `/v1/intakes/{id}/goals`
- ✅ `goalrail agent install` explicitly installs provider-neutral repo-local
  Agent Pack v0 files under `.goalrail/agent/` for local coding agents and may
  create a tiny root `AGENTS.md` shim only when missing; it does not overwrite
  existing root agent instructions and does not install Claude, Gemini, Cursor,
  Windsurf, Gravity, or other provider-specific adapters
- ✅ `goalrail work start --body-file <path|->` supports agent-friendly task
  bodies from a file or stdin while returning a `goalrail.cli.v1` JSON envelope
  with `display.summary` and a planned unavailable Slice B continuation action
- no keychain integration
- no Organization selection UX or public Organization creation
- no auth token, contract, work item, audit, proof, diff, memory, or runtime
  cache storage in `.goalrail/project.yml`
- no root `.gitignore` mutation for Goalrail local-state ignores
- no audit/hook/branch/verification setup from init
- no WorkItem, Contract, audit request, Run, receipt, gate, proof, provider
  integration, provider shim, branch, PR, hook, clone, deploy-key setup,
  readiness reconciliation, `work continue`, `work answer`, or contract draft
  CLI from `work start` or `agent install`
- no proof retrieval
- no public registration
- no admin user creation endpoint
- no SaaS onboarding
- no organization creation API
- this CLI slice did not implement product web console login UI; the canonical
  `apps/web/console` source now consumes the server auth API directly while the
  server-rendered CLI auth bridge remains separate
- no runner, gate, proof, or generic queue work

## Completed backend bounded slice

### Auth API/login slice

Status: **DONE — smallest server-only auth API/login slice exists.**

Goal:
- implement the next narrow auth API boundary on top of the credential/session
  primitives and completed owner bootstrap command, without broadening into CLI
  or web UI.

Done means:
- ✅ `POST /v1/auth/login` accepts email/password, verifies the existing
  Argon2id password credential, rejects inactive users, requires an active
  OrganizationMembership, creates server-owned `user_sessions` refresh-token
  state, and returns a short-lived JWT access token plus an opaque refresh
  token
- ✅ login response includes `must_change_password`
- ✅ `POST /v1/auth/change-password` requires a valid bearer access token,
  verifies `current_password`, stores the new password hash, sets
  `must_change_password = false`, and sets `password_changed_at`
- ✅ `GET /v1/me` requires a valid bearer access token and loads the current
  User plus OrganizationMembership server-side
- ✅ JWT access tokens carry narrow identity/session claims only, not broad
  role or permission claims
- ✅ `GOALRAIL_AUTH_JWT_SECRET` config exists; the server may start without it,
  but auth endpoints fail clearly when token signing/validation is attempted
  without it
- no public registration
- no CLI `goalrail login`
- no browser loopback
- no web UI
- no admin user creation endpoint
- no SaaS onboarding
- no organization creation API
- no billing
- no SSO/OIDC
- no password reset or email invite/reset delivery
- no runner, gate, proof, or generic queue work

## Completed backend bounded slice

### Auth refresh/logout API slice

Status: **DONE — server-only refresh/logout lifecycle exists.**

Goal:
- implement the next narrow session lifecycle boundary around existing
  `user_sessions` and opaque refresh-token storage, without broadening into CLI
  login, browser loopback, web UI, public registration, admin user creation, or
  SaaS onboarding.

Done means:
- ✅ `POST /v1/auth/refresh` accepts an opaque refresh token, hashes it with
  the same opaque-token hash strategy used by login, looks up
  `user_sessions.refresh_token_hash`, rejects unknown, revoked, expired, or
  inactive sessions, rejects inactive users, requires an active
  OrganizationMembership, updates `last_used_at` / `updated_at`, and returns a
  new short-lived JWT access token only
- ✅ refresh keeps the existing refresh token valid until expiry; this slice
  does not implement refresh-token rotation
- ✅ `POST /v1/auth/logout` validates the bearer access token, loads the
  referenced session, marks that session revoked, and sets `revoked_at` /
  `updated_at`
- ✅ missing, invalid, or expired bearer tokens return unauthorized for logout
- ✅ missing or weak `GOALRAIL_AUTH_JWT_SECRET` returns `auth_not_configured`
  when refresh needs JWT signing
- no CLI `goalrail login`
- no browser loopback
- no web UI
- no public registration
- no admin user creation endpoint
- no SaaS onboarding
- no organization creation API
- no password reset or email invite/reset delivery
- no runner, gate, proof, or generic queue work

## Next public-surface bounded slice

### Pilot intake RU post-live mobile and copy QA

Goal:
- run real-device iOS Safari / Android Chrome smoke and native-speaker Russian
  copy proofread against the live public surface.

Done means:
- mobile rendering and lead form behavior are checked on real devices
- any copy changes land in lock-step with
  `docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md`
- no analytics, tracking, cookies, sessions, CRM, repo integration, LLM/API,
  runtime execution, gate, proof, or broad backend platform is added

## Completed backend bounded slice

### Installation schema foundation

Status: **DONE — smallest ADR-0022 schema foundation exists.**

Goal:
- implement the smallest server persistence foundation for the documented
  Installation boundary before auth, CLI login, or SaaS onboarding.

Done means:
- ✅ `installations` table exists
- ✅ `organizations.installation_id` exists
- ✅ organization slugs are unique within an Installation rather than globally
- ✅ installation mode enum/check accepts only `self_hosted` and `saas`
- ✅ `public_base_url` is stored on Installation
- ✅ dev seed creates one `self_hosted` Installation with explicit
  `http://localhost:8080` public base URL before the dev Organization
- `public_base_url` production bootstrap direction requires a normalized value without a
  trailing slash, with HTTPS except localhost/dev
- ✅ backend paths remain organization-aware and do not bypass `organization_id`
- auth credential/session primitives are now covered by the separate completed
  ADR-0023 credential foundation slice
- no JWT implementation
- no CLI login implementation
- no SaaS onboarding
- no organization creation API
- no web UI

## Next bounded slices

### Slice 1 — CTO deck outline
Goal:
- create a 6–8 slide outline for CTO / Head of Engineering conversations

Done means:
- problem, product, operating model, deployment, pilot, outputs, and next step are sequenced clearly
- the outline is derived from the current canon rather than ad hoc pitch copy

### Slice 2 — Landing copy rewrite
Goal:
- rewrite `docs/product/GOALRAIL_LANDING_COPY.md` for pilot-first, contract-centered motion

Done means:
- prompt-export framing is removed
- CTA is aligned to pilot qualification / task review
- public flow matches `GOALRAIL_DESIGN_DECISIONS.md` and `GOALRAIL_GTM_MODEL.md`

### Slice 3 — Spine package bootstrap
Goal:
- create first implementation package for core domain types and events

Done means:
- IDs, enums, object skeletons, and event envelope compile
- basic serialization / validation tests exist
- implementation starts from the updated canon rather than the older docs baseline

### Slice 4 — Open-source asset provenance audit
Goal:
- classify reference screens, mascot/brand assets, and any third-party materials before broader public OSS reuse

Done means:
- `docs/reference/design/reference_screens/` usage and licensing status are documented
- any exclusions or attribution needs are explicit
- repo-level OSS policy stays aligned with actual asset rights

### Slice — Pilot intake RU hosting provider selection (blocker)
Status: DONE — provider has been chosen and then changed. D-0050 selected Cloudflare Pages Direct Upload but is now `superseded by D-0051 for hosting provider and deployment mode`. D-0051 selects an **operator-managed SSH static server** as the hosting path: manual rsync/scp upload, atomic release directory + `current` symlink, server-managed HTTPS, externally-managed DNS, no Git integration, no automatic redeploys, no Cloudflare Pages / Workers / Functions / KV / R2 / D1 / Durable Objects / Queues / proxy/CDN / Web Analytics for this surface. No further follow-up at the provider-decision level.

### Slice — Pilot intake RU pre-publish hygiene patch
Status: DONE (Phase 8E, then realigned in Phase 8K). Phase 8E updated `apps/web/pilot-intake-ru/index.html` `<link rel="canonical">` to `https://pilot.goalrail.dev/` (the then-active D-0049 value); Phase 8K realigned it to `https://pilot.goalrail.ru/` (the active D-0053 value, which supersedes D-0049 for target domain and canonical public URL); built `dist/index.html` reflects the active `.ru` canonical; typecheck / test / build / preview smoke / boundary scan all PASS; no other app source / tests / styles / package files were touched. No follow-up needed.

### Slice — Pilot intake RU target-domain realignment to .ru
Status: DONE (Phase 8K). `docs/ops/DECISIONS.md` D-0053 records the target-domain change from `pilot.goalrail.dev` to `pilot.goalrail.ru`; D-0049 is marked superseded by D-0053 for target domain and canonical public URL; `apps/web/pilot-intake-ru/index.html` canonical and built `dist/index.html` canonical now point to `https://pilot.goalrail.ru/`; STATUS / NEXT / WIRING / COMPONENTS reflect the active `.ru` target; D-0047 / D-0048 / D-0051 boundaries fully intact; D-0050 remains superseded by D-0051; no deployment, no DNS, no TLS provisioning, no backend, no analytics, no email backend, no persistence, no LLM/repo/runtime integration, no new scenarios, no new outcome tones, and no product behavior changes were introduced. The `.dev` domain remains reserved for a later global-market rollout.

### Slice — Pilot intake RU business-first landing validation and SSH static deployment
Goal:
- validate the rewritten business-first `apps/web/pilot-intake-ru` landing locally, then publish it at `https://pilot.goalrail.ru/` on the operator-managed SSH static server per D-0055 + D-0053 + D-0051, without expanding product scope and without weakening D-0047

Status: **LIVE VIA SSH STATIC SERVER — SMOKE PASSED.** The business-first landing rewrite and D-0056 lead-capture patch passed local typecheck / test / build / boundary audit / local preview smoke. The operator-managed SSH server was bootstrapped, Nginx static serving was configured, timestamped releases were uploaded with `rsync`, and `/srv/goalrail/pilot/current` was atomically switched to the latest release. The earlier server install used a narrow PHP-FPM `/api/pilot-lead` endpoint; live endpoint wiring has now migrated to the D-0062 Go sidecar outside repo source control. D-0059 Resend HTTPS transport is configured with `skill7.dev` sender and server-local API key, local Postfix/sendmail remains a fallback where available, server-local direct recipient override is configured outside the repo, lead JSONL append and duplicate suppression passed, the daily digest cron is installed for 07:00 GMT+3 previous-day summaries, digest dry-run and purge dry-run smoke passed, and server-local endpoint smoke passed. Server-side TLS provisioning, renew dry-run, server-local HTTPS smoke, public DNS verification, public HTTPS smoke, and public `/api/pilot-lead` smoke passed.

Immediate next action:
- perform real-device iOS Safari / Android Chrome smoke and native-speaker Russian copy proofread against the live public surface; file any fixes as separate small patches.

Pre-requisites (base gating satisfied; static upload and live smoke complete):
- the target-domain decision is recorded (D-0053: `pilot.goalrail.ru` / `/` / `candidate-public`, supersedes D-0049 for target domain and canonical public URL; D-0051 hosting/deployment mode preserved)
- the canonical-link metadata fix has landed (Phase 8E — DONE)
- the hosting-path decision is recorded (D-0051: operator-managed SSH static server; D-0050 is `superseded by D-0051 for hosting provider and deployment mode`)
- the current business-first rewrite has fresh typecheck / test / build / boundary scan / local preview smoke results recorded in `docs/ops/PILOT_INTAKE_RU_DEPLOYMENT_WIRING.md`

Done means:
- the operator-managed SSH static server was accessed through operator-provided SSH targets; host, IP, SSH port, SSH user, credentials, and reverse-proxy details remain out of repo; the planned release root and current symlink layout are confirmed in `docs/ops/PILOT_INTAKE_RU_DEPLOYMENT_WIRING.md` without leaking server identifiers
- the web server / reverse-proxy posture is confirmed (e.g. existing nginx/Caddy/Apache/etc.) without committing reverse-proxy config to this repo unless a separate explicit decision authorises it
- production build is run locally (`npm run pilot-intake-ru:typecheck`, `npm run pilot-intake-ru:test`, `npm run pilot-intake-ru:build`) and `apps/web/pilot-intake-ru/dist/` is the upload payload
- the manual upload method has been exercised: `rsync` from the local machine to a timestamped release directory on the SSH server, followed by an atomic `current` symlink switch; no previous non-placeholder release existed during this run, so the placeholder remains the only rollback anchor until the next real release
- no SSH keys, tokens, hostnames, IP addresses, usernames, ports, paths, deploy scripts, server config, or other server identifiers are committed to the repository
- the rollback method is documented: switching the `current` symlink back to a previously-known-good release directory, with no repo-side state changes
- because `PUBLIC_PATH` is `/`, no `vite.config.ts` `base` adjustment is required; root-path behavior and static asset paths are explicitly verified on the deployed surface
- no env vars, no secrets, and no runtime configuration are introduced anywhere — neither in repo nor in the deployed assets
- a local `vite preview` smoke check (`npm run preview --workspace @goalrail/pilot-intake-ru-web` from `apps/web`) is run before upload; visually verifies the business-first landing sections, CTA focus, lead form visibility, mailto fallback, canonical URL, and no non-static network requests on load
- a server-local static smoke check passed against the manually-uploaded release; server-side TLS provisioning, server-local HTTPS smoke, public `https://pilot.goalrail.ru/` smoke, and public `/api/pilot-lead` smoke passed
- the canonical URL in the deployed `index.html` is verified to be `https://pilot.goalrail.ru/`
- canonical copy parity is re-confirmed against `docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md`; any drift is reconciled in a follow-up patch
- the lead form posts only to same-origin `/api/pilot-lead`, the fallback still resolves to `mailto:pilot@goalrail.dev`, the Telegram channel resolves to `https://t.me/goalrail`, and the primary CTA still only focuses the email input with the temporary local highlight
- DNS now reaches the operator-managed SSH server per D-0051 / D-0053, and public resolver comparison passed without recording server IPs in repo docs. If the DNS zone is in Cloudflare, the record remains outside Cloudflare Pages, Workers, Web Analytics, and repo-side infrastructure config unless a separate decision changes that.
- server-side HTTPS provisioning is installed on the operator-managed server, and public HTTPS at `https://pilot.goalrail.ru/` is verified live
- a real-device pass is performed on iOS Safari and Android Chrome against the live URL; any blockers are filed as separate small patches
- a native-speaker proofread of canonical Russian copy is performed as a post-live QA slice; any wording changes land in lock-step in `App.tsx` and `docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md`
- D-0047 boundary is re-confirmed in the deployed context with the D-0056 exception: no LLM/API, no repo provider integration, no code execution, no persistence beyond local JSONL lead log, no analytics or session tracking, no cookies, no sessions, no CRM, no Google Sheets, no broad backend platform, no chat UI, no file upload, no model selector; no non-static network requests are observed during page load, and submit only calls same-origin `/api/pilot-lead`
- the result is recorded in `docs/ops/PILOT_INTAKE_RU_DEPLOYMENT_WIRING.md` as `LIVE VIA SSH STATIC SERVER — SMOKE PASSED`
- if the chosen target domain or public path ever changes, that change is recorded as a separate explicit decision in `docs/ops/DECISIONS.md` before it is implemented; if the deployment mode is later changed (e.g. CI deploy, automatic deploys, repo-side server config), that also requires its own separate decision per D-0051

Out of immediate scope (do not introduce without a new decision):
- backend beyond D-0056 `POST /api/pilot-lead`, D-0058 daily digest, D-0059 Resend mail transport, or any other form submission
- analytics or session tracking
- LLM / AI API calls
- repo provider integration
- real execution / runner / sandbox
- persistence of user input
- Cloudflare Pages, Cloudflare Workers, Cloudflare Functions, Cloudflare KV / R2 / D1 / Durable Objects / Queues, Cloudflare proxy / CDN, Cloudflare Web Analytics
- automatic deploys (CI / Git-triggered / cron-driven deploy / file-watch / etc.)
- storing SSH keys, server credentials, hostnames, IP addresses, ports, usernames, deploy scripts, or reverse-proxy config in the repository
- restoring or expanding the old technical interactive walkthrough without a separate bounded decision
- business-first copy rewrites beyond findings reconciled with the canonical doc
- email lead capture beyond the D-0056 `POST /api/pilot-lead` endpoint plus `mailto:` fallback

### Slice 5 — Publishing Boundary Migration
Goal:
- establish the thin binding manifest and perform repository cleanup

Done means:
- ✅ `.punk/publishing.toml` exists as the committed binding manifest
- ✅ resolver contract `punk publishing locate --project-root . --json` is documented
- ✅ existing content in `.punk/publishing/` has been inventoried and classified
- ✅ legacy repo-local directory `.punk/publishing/` has been removed from the repository
- `AGENTS.md` uses the manual bootstrap fallback/resolver logic
- `.gitignore` blocks secrets/sessions and the legacy directory path
- Next: implement or verify the resolver; perform semantic review of styles/prompts in the external workspace

### Slice — Project Scan / Repository Baseline lifecycle docs
Status: DONE — architecture boundary recorded.

Goal:
- document the local Project Scan, immutable `RepositoryBaselineProfile`,
  separate `WorkspaceOverlay`, and future task-specific `ContractContextPack`
  freshness model before implementation.

Done means:
- ADR-0025 defines baseline lifecycle, rebuild triggers, overlay handling,
  partiality states, background-scan limits, and the server/no-clone boundary
- `docs/product/GOALRAIL_PROJECT_SCAN_AND_CONTEXT_PACK_V0.md` defines Project
  Scan v0, `RepositoryBaselineProfile`, `WorkspaceOverlay`,
  `ContractContextPack`, freshness gates, and v0 non-goals
- `docs/INDEX.md`, `docs/ops/DECISIONS.md`, and `docs/ops/NEXT.md` are aligned
- no implementation code, server clone, provider OAuth, runner checkout,
  watcher/daemon, embeddings, raw source upload, gate, or proof is added in this
  docs slice

### Architecture follow-up slices

1. Project Scan v0 implementation boundary
   - Status: DONE — local CLI baseline / overlay foundation exists.
   - `apps/cli/internal/projectscan` builds immutable local
     `RepositoryBaselineProfile` JSON for committed HEAD, refreshes cheap
     `WorkspaceOverlay` JSON, evaluates freshness, and writes cache artifacts
     under the user cache directory
   - `goalrail project scan/status` are the explicit local freshness commands
   - server-backed `goalrail init` runs the quick local Project Scan after the
     non-secret `.goalrail/project.yml` marker is written or verified
   - start from ADR-0025 and `GOALRAIL_PROJECT_SCAN_AND_CONTEXT_PACK_V0.md`
   - keep scanning local CLI / runner owned and deterministic
   - persist summaries/receipts only, not raw source bodies by default
   - separate `RepositoryBaselineProfile` from `WorkspaceOverlay`
   - do not add server-side clone, provider OAuth, runner checkout,
     watcher/daemon, embeddings, ContractContextPack generation, gate, or proof
2. Runner-owned repository checkout credential boundary
   - define runner startup flags for Goalrail connection and local credential
     file paths only
   - define API-issued `CheckoutInstruction` fields, including
     `repo_binding_id`, `repository_url`, `ref`, `path_scope`, and optional auth
     hint
   - define `CheckoutReceipt` / bounded metadata snapshot fields returned by
     the runner
   - define supported credential modes: Git HTTPS token file, SSH key file, and
     mounted workspace
   - add no provider OAuth, VcsConnection, token storage, provider clients, live
     metadata listing, checkout implementation, runner implementation, gate, or
     proof
3. Organization / project / repo binding persistence boundary
   - ADR-0010 documents Goalrail `Organization`, `User`, `OrganizationMembership`, `Project`, `RepoBinding`, and `RepoBinding.access_mode`
   - direct `RepoBinding` stores repository reference in the MVP
   - `RepositoryRecord` and `RepositoryEnrollment` are deferred
   - normal CLI repository-context init and metadata-only RepoBinding init
     remain valid
   - support the runner-owned credential path without requiring GitHub App,
     GitLab, or Bitbucket cloud connection
4. Runner checkout prototype boundary
   - start with a universal runner as a separate binary/process
   - use pull-based / poll-based job leasing from the API server
   - perform read-only ephemeral checkout or use a mounted workspace and
     produce a checkout receipt with minimum evidence fields
   - do not implement provider OAuth, token storage, provider clients,
     persistent mirrors, repository writes, arbitrary command execution, gate,
     or proof
5. Customer-hosted runner protocol boundary
   - define later customer-hosted runner protocol, registration/auth, and customer-owned repository credential flow
   - keep clone access inside customer infrastructure and return bounded artifacts only
   - leave optional attestation or receipt signatures for a later trust-hardening slice

### CLI follow-up slices

1. Server-side repo key provisioning API/client
   - define the smallest server-owned provisioning boundary for repo access
   - keep production private-key generation and storage outside the local CLI
2. Marker-backed work command hardening
   - decide whether later work-start UX needs a server-owned composite endpoint
     for Intake + Goal atomicity before adding audit
   - keep Contract, WorkItem, audit, runner, gate, and proof deferred
3. Contract draft/approval flow integration
   - connect `goalrail contract validate` to real contract draft and approval state
   - preserve field-level validation findings
4. Proof retrieval from server
   - add `proof show` support for fetching stored proof by ID when server proof state exists
   - do not generate final gate verdicts in the CLI

### Server follow-up slices

1. Typed WorkItemPlan pull lease boundary implementation
   - implement the smallest server-owned typed planning lease protocol around
     existing `WorkItemPlan` / `WorkItemPlanProposal` state
   - keep repo-aware planning computation behind worker / controller / runner
     boundaries
   - do not implement checkout, execution, receipt, generic queue, outbox, gate,
     proof, runtime registry, assignment, or claiming
2. WorkItem assignment/claiming boundary design
   - define the smallest explicit transition after the accepted-proposal
     planning boundary
   - keep runner, execution, receipt, gate, and proof as later boundaries
   - do not start execution from assignment/claiming
   - preserve `owner_hint` as advisory unless a later boundary upgrades it
3. Runner boundary design
   - define the hosted runner protocol and checkout boundary before any code
     execution work
   - keep execution, gate, and proof deferred
5. CLI-to-server intake submit integration
   - submit intake from the CLI to the server once the API boundary exists
   - keep the CLI as an adapter, not a canonical state owner

## Deferred until later

- hosted execution implementation beyond bounded runner prototypes
- tracker integrations
- multi-runtime advisory implementation
- external checks implementation
- analytics / console product features
- Goalrail-specific web product features beyond the current change-packet demo prototypes
- persistent repository mirrors
- repository write operations such as branch creation, commits, or pull requests
