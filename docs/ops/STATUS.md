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
---
# Goalrail Status

Last updated: 2026-05-02
Status: planning / product canon and pilot frame active; first local Go CLI and Go server intent-plane / public Contract aggregate and `/v1/contracts` lifecycle façade / ContractSeed / ContractDraft / ApprovedContract / WorkItem persistence plus `plans` / `proposals` / `acceptance` WorkItem planning control-plane flow exists; public Contract aggregate identity is implemented as a stable `contract_id` boundary and transitional public seed/draft/approval/direct-task routes are removed; pilot-intake-ru is now a business-first public RU pilot landing per D-0055 (`ИИ-кодинг без хаоса`, safe 2-week пилот ИИ-разработки, repository readiness, project context, controlled tasks, verified result) rather than the previous technical interactive walkthrough; active target domain remains `pilot.goalrail.ru` per D-0053, canonical metadata in `apps/web/pilot-intake-ru/index.html` remains `https://pilot.goalrail.ru/`, the static hosting path remains operator-managed SSH static server per D-0051, server upload, operator-managed Go sidecar migration from previous PHP-FPM wiring, server-side TLS provisioning, server-local HTTPS smoke, public DNS verification, public HTTPS smoke, public `/api/pilot-lead` smoke, and D-0058 digest dry-run are complete, and D-0047 boundaries remain intact except for the narrow D-0056 lead-capture endpoint, D-0058 daily digest, and D-0059 Resend mail transport (no analytics, tracking, CRM, Google Sheets, cookies, sessions, LLM/API calls, repo integration, runtime execution, broad backend platform, chat UI, file upload, or model selector).
Owner: Vitaly

Current risk note: the stabilization tranche is complete repo-side through
D-0065, and the operator-managed Go sidecar deployment plus public DNS/live
smoke slice has passed. The public RU pilot surface is now live through the
operator-managed SSH static server, with `/api/pilot-lead` routed to the Go
sidecar rather than the previous PHP-FPM wiring. This status does not claim
repo-side deployment automation, committed server config, committed DNS config,
required human review, signed-commit enforcement, real-device mobile QA, or
native-speaker copy proofread. It also does not approve analytics, CRM,
database, queue, LLM/API, repo integration, runtime execution, gate, proof, or
broad backend platform behavior.

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
- twenty kernel/CLI/server/domain boundary ADRs
- ops rails
- repo-tracked Goalrail and Punk overlay surfaces
- planned flow / eval structure
- reference screens
- shared web stack rules under `apps/web/`
- empty real console shells under `apps/web/console` and `apps/web/console-ru`
- local change-packet demo prototypes under `apps/web/demo-change-packet` and `apps/web/demo-change-packet-ru`
- a business-first RU pilot landing under `apps/web/pilot-intake-ru` for `ИИ-кодинг без хаоса`: a mostly static Founding Pilot page for a safe 2-week пилот ИИ-разработки on one product area, with illustrative repository readiness / controlled task / pilot result cards, a D-0056 minimal `POST /api/pilot-lead` email lead endpoint with local JSONL notification status, retry after `notification_failed`, in-flight `received` / `pending` rows blocked as duplicate submissions, duplicate suppression for successfully notified, legacy processed, and in-flight rows, no user-agent storage for new lead records, a landing-owned repo-side Go sidecar for the endpoint/digest/purge command under `apps/web/pilot-intake-ru/server`, server-installed daily previous-day digest at 07:00 GMT+3 when leads exist plus direct mailto fallback, no analytics, no tracking, no IP logging, no cookies, no sessions, no fingerprinting, no CRM, no Google Sheets, no repo integration, no runtime execution, no persistence beyond local JSONL lead log, no chat UI, no file upload, and no model selector; the previous 5-step technical walkthrough is demoted to internal / technical demo or checkpoint status in git history per D-0055.
- an open-source community baseline (`LICENSE`, `NOTICE`, contributor docs, issue forms, `CODEOWNERS`)
- a Go server bootstrap under `apps/server` with Postgres-backed source-neutral intake, Goal promotion, Goal readiness state, ContractSeed creation, ContractDraft creation/update/ready_for_approval, ApprovedContract approval, WorkItem `plans` / `proposals` / `acceptance` planning control-plane flow, planned task read-by-ID, EventLog persistence, and transactional canonical write + event append hardening when DB is configured, plus in-memory ClarificationRequest, ClarificationAnswer recording, answer application, and explicit re-check-after-applied-answers prototypes

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
- managed deployment is the default early deployment mode
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
- kernel schema note and twenty boundary ADRs exist

### Repo structure
- the repo now mirrors `punk`-style planning boundaries
- `.goalrail/work/` is reserved for goals, reports, and bounded planning artifacts such as demo-planning packs
- `.goalrail/knowledge/` is reserved for advisory research and idea backlog
- `.punk/publishing.toml` remains the repo-local binding
- `.punk/publishing/` legacy directory has been removed from repo after manual copy/verify; the runtime publishing workspace lives in external user/platform-local storage
- `.punk/publishing.local.toml` is the ignored local-only manual-bootstrap pointer; resolver/runtime implementation is pending
- `.goalrail/flows/` and `.goalrail/evals/` exist as planned future structure, not executable product surfaces
- `apps/web/` is now the shared namespace for frontend resources and stack rules
- `apps/web/console` is the empty real console shell with three left-nav surfaces: Contracts, Delivery Readiness, and Proof
- `apps/web/demo-change-packet` is the current React + Vite + Mantine EN change-packet demo prototype, deployed through standalone infra at `demo.goalrail.dev`
- `apps/web/demo-change-packet-ru` is the separate RU copy of the change-packet demo prototype, deployed through standalone infra at `demo.goalrail.ru` rather than in-app i18n
- `apps/web/console-ru` is the separate Russian console shell for `console.goalrail.ru` with the same empty-surface boundary
- `apps/web/pilot-intake-ru` is the current public React + Vite + Mantine RU business-first pilot landing for `pilot.goalrail.ru` (`ИИ-кодинг без хаоса`, safe 2-week пилот ИИ-разработки, repository readiness, project context, controlled tasks, verified result); it includes a narrow landing-owned Go sidecar under `apps/web/pilot-intake-ru/server` for lead capture and digest source, and it supersedes the previous technical interactive walkthrough as the primary public RU landing per D-0055.
- `apps/cli` is the first stdlib-only Go CLI bootstrap with canonical binary entrypoint `cmd/goalrail`
- local/demo CLI commands now exist for `version`, `init`, `readiness scan`, `contract validate`, and `proof show`
- `apps/server` is the first Go HTTP server bootstrap with canonical binary entrypoint `cmd/goalrail-server`
- server endpoints include `GET /livez`, `GET /readyz`, `GET /version`, `POST /v1/intakes`, `GET /v1/intakes/{id}`, `POST /v1/intakes/{id}/goals`, `POST /v1/goals/{id}/readiness`, `POST /v1/goals/{id}/clarifications`, `POST /v1/clarifications/{id}/answers`, `POST /v1/answers/{id}/applications`, `POST /v1/contracts`, `GET /v1/contracts/{id}`, `PATCH /v1/contracts/{id}`, `POST /v1/contracts/{id}/submissions`, `POST /v1/contracts/{id}/approvals`, `POST /v1/contracts/{id}/plans`, `GET /v1/plans/{id}`, `POST /v1/plans/{id}/proposals`, `GET /v1/proposals/{id}`, `POST /v1/proposals/{id}/acceptance`, and `GET /v1/tasks/{id}`; there is no `GET /v1/plans`, `GET /v1/proposals`, or `GET /v1/tasks` list endpoint, and the previous public `/v1/goals/{id}/contract-seeds`, `/v1/contract-seeds/{id}/contract-drafts`, `/v1/contract-drafts/{id}`, and direct `POST /v1/contracts/{id}/tasks` lifecycle/planning routes are no longer registered
- `POST /v1/contracts/{id}/plans` resolves `{id}` as stable public
  `contract_id`, requires the Contract to be `approved`, and creates a
  server-owned `WorkItemPlan(queued)` without creating WorkItems
- `apps/server` now has a Postgres persistence foundation for the Organization / Project / RepoBinding context plus IntakeRecord, Goal, public Contract aggregate, ContractSeed, ContractDraft, ApprovedContract, and EventLog state
- server config accepts `GOALRAIL_DATABASE_DSN`
- `goalrail-server migrate up` applies the editable pre-production init migration
- `goalrail-server seed dev` applies the idempotent dev seed
- the init migration creates `users`, `organizations`, `organization_memberships`, `projects`, `repo_bindings`, `intake_records`, `goals`, `contracts`, `contract_seeds`, `contract_drafts`, `approved_contracts`, `work_item_plans`, `work_item_plan_proposals`, `work_items`, and `events` with UUID persisted ID columns for canonical entities
- the dev seed creates deterministic UUIDv7 IDs: `018f0000-0000-7000-8000-000000000001`, `018f0000-0000-7000-8000-000000000002`, `018f0000-0000-7000-8000-000000000005`, `018f0000-0000-7000-8000-000000000003`, and `018f0000-0000-7000-8000-000000000004`
- the project-context store builds runtime SQL with Squirrel and executes upserts and repo-binding context lookups through pgx/pgxpool
- the source-neutral intake API now requires `project_id` and `repo_binding_id`, validates the repo binding against the persisted Project / RepoBinding context when DB is configured, derives `organization_id`, stores `IntakeRecord` in Postgres when DB is configured, and appends a durable `intake.received` event with context fields
- Goal promotion stores `Goal` as non-executable normalized intent in Postgres when DB is configured, carries `organization_id`, `project_id`, and `repo_binding_id` from the IntakeRecord, prevents duplicate promotion through the persisted `intake_id` uniqueness boundary, and appends durable `goal.created` and `intake.promoted_to_goal` events with context fields
- Goal readiness updates persisted `Goal` state and readiness reason codes when DB is configured, returns reason codes, appends durable readiness transition events, and can be explicitly re-run after answer application
- Postgres-backed intake create, Goal promotion, Goal readiness, ContractSeed creation, ContractDraft creation/update, ContractDraft ready_for_approval, and ApprovedContract approval writes now share a transaction with their expected event appends, so the durable canonical write does not commit without its audit events
- ClarificationRequest creation stores an open request only as an in-memory prototype, generates deterministic questions from Goal readiness reason codes, and appends `clarification.requested` through the configured EventLog
- ClarificationAnswer recording stores canonical answer evidence only as an in-memory prototype, requires all questions answered, transitions the request from `open` to `answered`, and appends `clarification.answer_recorded` and `clarification.request_answered` through the configured EventLog
- answer application remains part of the in-memory clarification prototype for request/answer state; when DB is configured it updates persisted Goal intent-plane hints, rejects unsupported raw-text `goal.intent_owner` mapping, guards repeated application with `409 already_applied`, and appends events through the configured EventLog; it does not call readiness automatically
- `POST /v1/contracts` creates a public Contract lifecycle view from a ready Goal by creating internal `ContractSeed(created)` and `ContractDraft(draft)` records, returning Contract state `draft`, and not creating approval, tasks, execution, gate, or proof
- `PATCH /v1/contracts/{id}` updates the current internal draft's proposed fields through the public `contract_id`, requires `updated_by` as audit identity, preserves `ContractDraft.state = draft`, appends `contract_draft.updated`, and does not approve or create tasks
- `POST /v1/contracts/{id}/submissions` transitions the current internal draft to `ready_for_approval`, moves Contract state to `ready_for_approval`, requires `marked_by` as audit identity only, runs completeness checks, and does not approve Contract, create `WorkItem`, write `GateDecision`, or create `Proof`
- `POST /v1/contracts/{id}/approvals` creates an immutable internal `ApprovedContract(approved)` snapshot from the current ready draft, moves Contract state to `approved` with `approved_snapshot_id`, requires `approved_by`, guards repeated approval with `409 already_approved`, and does not mutate `ContractDraft`, start execution, write `GateDecision`, or create `Proof`
- WorkItem planning now uses `Plan -> Proposal -> Acceptance`: one `WorkItemPlan` per approved public Contract in v0, one `WorkItemPlanProposal` per plan in v0, explicit acceptance materializes one or more durable canonical `WorkItem(planned)` records with `plan_id` and `proposal_id`, persists the records in Postgres when DB is configured, exposes `GET /v1/tasks/{id}` for single task reads, appends `work_item.created` for each accepted task transactionally with Postgres acceptance, and does not assign, claim, create `Run`, start execution, checkout a repository, submit a receipt, write `GateDecision`, or create `Proof`; workers/planners submit proposals through the API and do not write WorkItems directly to the DB
- the runner / repository checkout boundary is documented in ADR-0008, but no runner implementation exists yet
- the `ClarificationAnswer` boundary is documented in ADR-0009; the answer application to Goal hints boundary is documented in ADR-0011 and still keeps clarification request/answer state in-memory
- the explicit readiness re-check after applied answers boundary is documented in ADR-0012, and the existing readiness endpoint is verified to move an applied-answer Goal to `ready_for_contract_seed` without creating contract/work/gate/proof artifacts
- the `ContractSeed` boundary is documented in ADR-0013 and implemented as a Postgres-backed internal snapshot when DB is configured; there is no standalone public ContractSeed route, and the public `POST /v1/contracts` façade composes internal seed plus draft creation under one stable `contract_id`; standalone seed creation does not approve Contract, create `WorkItem`, write `GateDecision`, or create `Proof`
- the `ContractDraft` boundary is documented in ADR-0014 and implemented as a Postgres-backed draft creation boundary when DB is configured; it does not create approved Contract, `WorkItem`, `GateDecision`, or `Proof`
- the `ContractDraft` review/update boundary is documented in ADR-0015 and implemented as a draft-only update boundary; it does not introduce `ready_for_approval`, approved Contract, `WorkItem`, `GateDecision`, or `Proof`
- the `ContractDraft ready_for_approval` boundary is documented in ADR-0016 and implemented as an explicit `draft -> ready_for_approval` state transition with completeness checks and `marked_by` audit identity; it is not approval, approved Contract, `WorkItem`, execution, `GateDecision`, or `Proof`
- the Contract approval boundary is documented in ADR-0017 and implemented as `ContractDraft(ready_for_approval) -> ApprovedContract`; approval does not start execution, write `GateDecision`, or create `Proof`
- the WorkItem planning boundary is documented in ADR-0018 and ADR-0019 and implemented as a public `Plan -> Proposal -> Acceptance -> WorkItem(planned)` control-plane flow with durable Postgres storage when configured and single task read by ID; the worker/controller/runner execution-side implementation remains deferred; WorkItem planning is not assignment, claiming, execution, `Run`, runner checkout, receipt, `GateDecision`, or `Proof`
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
- no production runtime CLI beyond the local/demo `apps/cli` command foundation
- no server integration for the CLI
- no server-owned canonical domain implementation beyond the persisted `IntakeRecord` / `Goal` / public Contract lifecycle façade / internal `ContractSeed` / `ContractDraft creation/update/ready_for_approval` / `ApprovedContract` / WorkItem planning plan/proposal/acceptance slice and in-memory `ClarificationRequest` / `ClarificationAnswer` prototypes yet
- no durable server storage for clarification request/answer state yet
- no automatic readiness re-check after answer application
- no WorkItem assignment/claiming, `Run`, receipt, GateDecision, or Proof yet
- no planning worker/controller, runner-backed planning implementation, lease
  protocol, assignment/claiming, checkout, execution, queue, outbox, broker, or
  runtime registry yet
- no production repo authorization or deploy-key provisioning in the CLI
- no real RepoBinding state sync
- no production organization/user/VCS connection/repository catalog implementation beyond the dev-seeded Organization / Project / RepoBinding Postgres foundation yet
- no `VcsConnection` implementation yet
- no `RepositoryRecord` implementation; it is intentionally deferred for the MVP
- no `RepositoryRecord.source_kind` implementation
- no `RepoBinding.access_mode` implementation
- no CRUD onboarding endpoints yet
- no manual-declared repository registration
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
- no data-backed Goalrail web UI or goal-to-proof product loop yet
- RU pilot landing static files are uploaded to the operator-managed SSH server; the repository source for D-0056/D-0057/D-0058/D-0059/D-0061 lead capture and digest is now a narrow landing-owned Go sidecar under `apps/web/pilot-intake-ru/server`, replacing the transitional PHP source in repo; on 2026-04-30 the operator-managed server wiring moved from the earlier PHP-FPM endpoint to the Go sidecar; the server-local Resend HTTPS mail transport uses `skill7.dev` sender and server-local API key, with local sendmail/Postfix fallback where available, server-local direct notification override configured outside the repo, fallback to `hello@goalrail.dev`, JSONL-based duplicate suppression, daily previous-day digest cron at 07:00 GMT+3 when leads exist, local JSONL lead log with UTC and GMT+3 submission fields for new rows, no user-agent storage for new rows, and D-0061 notification status so failed mail notifications remain retryable while in-flight attempts do not start duplicate mail delivery; D-0065 adds a local dry-run-first purge command for JSONL retention, and reverse-proxy rate limiting is applied as an operator-managed deployment guardrail without committed config; server-local Go sidecar, digest dry-run, purge dry-run, public DNS, public HTTPS, and public `/api/pilot-lead` smoke passed
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
- `apps/web/console` and `apps/web/console-ru` provide verified empty console shells only; they do not claim backend, server, auth, data, or product-loop implementation
- `apps/cli` provides a verified local/demo Go CLI bootstrap only; it does not claim server integration, hosted execution, production repo auth, real gate decisions, or proof generation
- `apps/server` provides a verified Go server bootstrap plus Postgres-backed source-neutral intake with Project / RepoBinding context validation, Goal promotion, deterministic Goal readiness state, public `/v1/contracts` lifecycle façade, public Contract aggregate persistence, internal ContractSeed creation, internal ContractDraft creation/update/ready_for_approval, internal ApprovedContract approval, WorkItem plan/proposal/acceptance planning storage, EventLog persistence, transactional canonical write + event append hardening, explicit re-check-after-applied-answers, and in-memory clarification request/answer prototypes when DB is configured; it creates `IntakeRecord`, non-executable `Goal`, open in-memory `ClarificationRequest`, recorded in-memory `ClarificationAnswer`, `Contract(seed/draft/ready_for_approval/approved)`, `ContractSeed(created)`, `ContractDraft(draft/ready_for_approval)`, `ApprovedContract(approved)`, `WorkItemPlan`, `WorkItemPlanProposal`, and accepted `WorkItem(planned)` records only, updates Goal readiness state, request answered state, Goal intent-plane hints, Contract aggregate state/pointers, ContractDraft proposed fields, ContractDraft readiness state, plan state, and proposal acceptance state only, exposes single task read by ID, and does not claim durable clarification storage, automatic readiness re-check, repo-aware planning computation, WorkItem assignment/claiming, execution, `Run`, receipt, gate, proof, repo readiness, auth, workers, or repository checkout
- `apps/web/pilot-intake-ru` provides a verified public RU business-first pilot landing for `ИИ-кодинг без хаоса`: it sells a safe 2-week пилот ИИ-разработки on one bounded product area, shows illustrative repository readiness / controlled task / pilot result cards with disclaimers, and keeps lead capture limited to `POST /api/pilot-lead` with local JSONL notification status, retry after `notification_failed`, in-flight `received` / `pending` rows blocked as duplicate submissions, duplicate suppression for notified / legacy processed / in-flight rows, no user-agent/IP/cookie/session/fingerprint tracking, a local JSONL purge command, plus `mailto:` fallback. The repo source for that narrow endpoint/digest is a landing-owned Go sidecar under `apps/web/pilot-intake-ru/server`, not the core `apps/server` API. Canonical copy and governance live in `docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md`; D-0055 demotes the previous 5-step technical walkthrough to internal / technical demo or checkpoint status; D-0047 boundaries remain intact except for D-0056's narrow lead-capture exception (no LLM/API, no repo provider integration, no code execution, no analytics or session tracking, no cookies, no sessions, no CRM, no Google Sheets, no broad backend platform, no chat UI, no file upload, no model selector, no real repository scan claim). The active target domain remains `pilot.goalrail.ru` per D-0053 with public path `/`; canonical metadata remains `https://pilot.goalrail.ru/`; SSH static deployment remains the hosting path per D-0051; the timestamped static release has been uploaded and `current` switched on the operator-managed server, live endpoint wiring uses the Go sidecar rather than PHP-FPM, and public DNS / HTTPS / `/api/pilot-lead` smoke passed.
- `apps/web/` remains a shared multi-resource namespace instead of a single runnable app surface
- repository community health and OSS baseline are explicit and inspectable
- next sales-pack, VCS-boundary, and runner-boundary slices are explicit and bounded

## Main current risks

1. ops, offer, deck, and landing assets could drift away from the new concept canon
2. schema work could overgrow before the first compiling package exists
3. runtime adapter model could drift into vendor-specific code
4. execution parallelism and advisory parallelism could still leak into one implementation surface
5. MVP scope could widen into a generic agent or tooling platform too early
6. repository checkout could leak into the API server instead of staying behind runner boundaries
7. customer-hosted runner support could be treated as a late enterprise add-on instead of a first-class architecture mode
8. reference screenshots or brand assets could be relicensed accidentally without a provenance audit
