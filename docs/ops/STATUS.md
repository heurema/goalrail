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

Last updated: 2026-04-26
Status: planning / product canon and pilot frame active; first local Go CLI and Go server intent-plane / ContractSeed / ContractDraft prototypes exist
Owner: Vitaly

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
- fifteen kernel/CLI/server/domain boundary ADRs
- ops rails
- repo-tracked Goalrail and Punk overlay surfaces
- planned flow / eval structure
- reference screens
- shared web stack rules under `apps/web/`
- empty real console shells under `apps/web/console` and `apps/web/console-ru`
- local change-packet demo prototypes under `apps/web/demo-change-packet` and `apps/web/demo-change-packet-ru`
- a local RU pilot-intake landing prototype under `apps/web/pilot-intake-ru`
- an open-source community baseline (`LICENSE`, `NOTICE`, contributor docs, issue forms, `CODEOWNERS`)
- a Go server bootstrap under `apps/server` with Postgres-backed source-neutral intake, Goal promotion, Goal readiness state, intake/goal EventLog persistence, and transactional canonical write + event append hardening when DB is configured, plus in-memory ClarificationRequest, ClarificationAnswer recording, answer application, explicit re-check-after-applied-answers, ContractSeed creation, and ContractDraft creation prototypes

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
- D-0041 documents transactional Postgres-backed intake create, Goal promotion,
  and Goal readiness write/event boundaries without adding queue, outbox, or
  Unit of Work framework semantics

### Delivery model
- roadmap phases defined
- checkpoint model defined
- bounded slice workflow defined
- implementation discipline fixed: `punk`
- execution parallelism and advisory parallelism are separated conceptually
- kernel schema note and fifteen boundary ADRs exist

### Repo structure
- the repo now mirrors `punk`-style planning boundaries
- `.goalrail/work/` is reserved for goals, reports, and bounded planning artifacts such as demo-planning packs
- `.goalrail/knowledge/` is reserved for advisory research and idea backlog
- `.punk/publishing/` is reserved for public narrative drafts, receipts, and manual metrics owned by the Punk publishing layer
- `.goalrail/flows/` and `.goalrail/evals/` exist as planned future structure, not executable product surfaces
- `apps/web/` is now the shared namespace for frontend resources and stack rules
- `apps/web/console` is the empty real console shell with three left-nav surfaces: Contracts, Delivery Readiness, and Proof
- `apps/web/demo-change-packet` is the current React + Vite + Mantine EN change-packet demo prototype, deployed through standalone infra at `demo.goalrail.dev`
- `apps/web/demo-change-packet-ru` is the separate RU copy of the change-packet demo prototype, deployed through standalone infra at `demo.goalrail.ru` rather than in-app i18n
- `apps/web/console-ru` is the separate Russian console shell for `console.goalrail.ru` with the same empty-surface boundary
- `apps/web/pilot-intake-ru` is the current local React + Vite + Mantine RU pilot-intake landing prototype
- `apps/cli` is the first stdlib-only Go CLI bootstrap with canonical binary entrypoint `cmd/goalrail`
- local/demo CLI commands now exist for `version`, `init`, `readiness scan`, `contract validate`, and `proof show`
- `apps/server` is the first Go HTTP server bootstrap with canonical binary entrypoint `cmd/goalrail-server`
- server endpoints include `GET /livez`, `GET /readyz`, `GET /version`, `POST /v1/intake`, `GET /v1/intake/{id}`, `POST /v1/intake/{id}/promote`, `POST /v1/goals/{id}/readiness`, `POST /v1/goals/{id}/clarification-requests`, `POST /v1/clarification-requests/{id}/answers`, `POST /v1/clarification-answers/{id}/apply`, `POST /v1/goals/{id}/contract-seed`, and `POST /v1/contract-seeds/{id}/contract-draft`
- `apps/server` now has a Postgres persistence foundation for the Organization / Project / RepoBinding context plus IntakeRecord, Goal, and intake/goal EventLog state
- server config accepts `GOALRAIL_DATABASE_DSN`
- `goalrail-server migrate up` applies the editable pre-production init migration
- `goalrail-server seed dev` applies the idempotent dev seed
- the init migration creates `users`, `organizations`, `organization_memberships`, `projects`, and `repo_bindings` with UUID persisted ID columns
- the dev seed creates deterministic UUIDv7 IDs: `018f0000-0000-7000-8000-000000000001`, `018f0000-0000-7000-8000-000000000002`, `018f0000-0000-7000-8000-000000000005`, `018f0000-0000-7000-8000-000000000003`, and `018f0000-0000-7000-8000-000000000004`
- the project-context store builds runtime SQL with Squirrel and executes upserts and repo-binding context lookups through pgx/pgxpool
- the source-neutral intake API now requires `project_id` and `repo_binding_id`, validates the repo binding against the persisted Project / RepoBinding context when DB is configured, derives `organization_id`, stores `IntakeRecord` in Postgres when DB is configured, and appends a durable `intake.received` event with context fields
- Goal promotion stores `Goal` as non-executable normalized intent in Postgres when DB is configured, carries `organization_id`, `project_id`, and `repo_binding_id` from the IntakeRecord, prevents duplicate promotion through the persisted `intake_id` uniqueness boundary, and appends durable `goal.created` and `intake.promoted_to_goal` events with context fields
- Goal readiness updates persisted `Goal` state and readiness reason codes when DB is configured, returns reason codes, appends durable readiness transition events, and can be explicitly re-run after answer application
- Postgres-backed intake create, Goal promotion, and Goal readiness writes now share a transaction with their expected event appends, so the durable canonical write does not commit without its audit events
- ClarificationRequest creation stores an open request only as an in-memory prototype, generates deterministic questions from Goal readiness reason codes, and appends `clarification.requested` through the configured EventLog
- ClarificationAnswer recording stores canonical answer evidence only as an in-memory prototype, requires all questions answered, transitions the request from `open` to `answered`, and appends `clarification.answer_recorded` and `clarification.request_answered` through the configured EventLog
- answer application remains part of the in-memory clarification prototype for request/answer state; when DB is configured it updates persisted Goal intent-plane hints, rejects unsupported raw-text `goal.intent_owner` mapping, guards repeated application with `409 already_applied`, and appends events through the configured EventLog; it does not call readiness automatically
- ContractSeed creation stores a `ContractSeed(created)` snapshot only as an in-memory prototype from `Goal(ready_for_contract_seed)`, guards repeated creation with `409 already_seeded`, appends `contract_seed.created`, and does not mutate Goal state
- ContractDraft creation stores a `ContractDraft(draft)` only as an in-memory prototype from `ContractSeed(created)`, guards repeated creation with `409 already_drafted`, appends `contract_draft.created`, and does not mutate ContractSeed
- the runner / repository checkout boundary is documented in ADR-0008, but no runner implementation exists yet
- the `ClarificationAnswer` boundary is documented in ADR-0009; the answer application to Goal hints boundary is documented in ADR-0011 and still keeps clarification request/answer state in-memory
- the explicit readiness re-check after applied answers boundary is documented in ADR-0012, and the existing readiness endpoint is verified to move an applied-answer Goal to `ready_for_contract_seed` without creating contract/work/gate/proof artifacts
- the `ContractSeed` boundary is documented in ADR-0013 and implemented as an in-memory snapshot prototype only; it does not create `ContractDraft`, approved Contract, `WorkItem`, `GateDecision`, or `Proof`
- the `ContractDraft` boundary is documented in ADR-0014 and implemented as an in-memory draft prototype only; it does not create approved Contract, `WorkItem`, `GateDecision`, or `Proof`
- the `ContractDraft` review/update boundary is documented in ADR-0015, but no update implementation exists yet
- the Organization / Project / RepoBinding and persistence bootstrap boundary is documented in ADR-0010, and the first server-local Postgres foundation exists
- `.github/` now contains real contributor/community health surfaces and the docs-check workflow
- `scripts/` remains parked for future bounded implementation slices

## What is not real yet

- no standalone Project Spine schema package beyond CLI/server-local DTO subsets
- no runtime registry implementation
- no production runtime CLI beyond the local/demo `apps/cli` command foundation
- no server integration for the CLI
- no server-owned canonical domain implementation beyond the persisted `IntakeRecord` / `Goal` slice and in-memory `ClarificationRequest` / `ClarificationAnswer` / `ContractSeed` / `ContractDraft` prototypes yet
- no durable server storage for clarification request/answer, ContractSeed, or ContractDraft state yet
- no automatic readiness re-check after answer application
- no approved Contract, contract approval, WorkItem, GateDecision, or Proof yet
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
- no runner registration, runner assignment, checkout request, checkout receipt, or worker implementation yet
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
- no production landing deployment or backend lead-capture integration yet
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
- `apps/server` provides a verified Go server bootstrap plus Postgres-backed source-neutral intake with Project / RepoBinding context validation, Goal promotion, deterministic Goal readiness state, intake/goal EventLog persistence, transactional canonical write + event append hardening, explicit re-check-after-applied-answers, and in-memory ContractSeed / ContractDraft creation when DB is configured; it creates `IntakeRecord`, non-executable `Goal`, open in-memory `ClarificationRequest`, recorded in-memory `ClarificationAnswer`, in-memory `ContractSeed(created)`, and in-memory `ContractDraft(draft)` only, updates Goal readiness state, request answered state, and Goal intent-plane hints only, and does not claim durable clarification, ContractSeed, or ContractDraft storage, automatic readiness re-check, approved Contract, contract approval, work item creation, gate, proof, repo readiness, auth, workers, or repository checkout
- `apps/web/pilot-intake-ru` provides a verified local RU pilot-intake landing prototype for the pilot-first public entry
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
