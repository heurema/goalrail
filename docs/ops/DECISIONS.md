# Goalrail Decisions

## D-0001 — Goalrail uses a dual-plane product model
Date: 2026-04-13
Status: accepted

Decision:
- product has two main planes:
  - Intent / Planning
  - Delivery / Execution
- both are connected through one Project Spine

## D-0002 — Goalrail is not a tracker replacement in v1
Date: 2026-04-13
Status: accepted

Decision:
- Goalrail acts as an intent-to-delivery layer
- external trackers remain systems of record where needed

## D-0003 — Runtime neutrality is explicit and CLI-first
Date: 2026-04-14
Status: accepted

Decision:
- Goalrail is runtime-neutral
- first-class integrations target authenticated developer runtimes such as CLIs and local tools
- raw API adapters are optional later extensions, not the default assumption
- runtime-specific logic must stay behind adapters

## D-0004 — Runtime may execute; gate decides; proof preserves
Date: 2026-04-13
Status: accepted

Decision:
- execution truth and final trust decision remain separate
- final verdict is written only by gate
- proof is immutable and linked to decision

## D-0005 — Parallel task execution uses execution groups
Date: 2026-04-14
Status: accepted

Decision:
- parallel work over different tasks is expressed through Execution Groups
- disjoint writable tasks may run in parallel
- overlap or uncertainty forces stronger isolation or serialization
- every multi-run group ends with a barrier before final downstream verification

## D-0006 — Goalrail implementation proceeds through punk
Date: 2026-04-13
Status: accepted

Decision:
- Goalrail implementation proceeds through `punk`
- work advances in bounded slices rather than broad scaffold dumps
- implementation posture must preserve explicit verification and proof discipline

## D-0007 — One writable run has one primary writer; advisory panels are separate
Date: 2026-04-14
Status: accepted

Decision:
- one writable run uses one primary writer runtime
- the same task may use multiple advisory runtimes in parallel
- advisory panels are non-authoritative inputs to gate, not replacements for gate

## D-0008 — Task routing is risk- and policy-driven
Date: 2026-04-14
Status: accepted

Decision:
- tasks carry an explicit risk level
- risk affects review depth and advisory fan-out
- policy may narrow runtime exposure beyond what risk alone would suggest
- sensitive tasks may require `single-vendor-only`, `local-only`, or human signoff

## D-0009 — Gate reads frozen verification inputs
Date: 2026-04-14
Status: accepted

Decision:
- gate evaluates frozen bundles, receipts, baseline snapshots, and persisted advisory outputs
- gate does not trust a live mutable workspace as the final verification source
- holdout checks may exist outside the primary execution packet

## D-0010 — Baselines and repo invariants are first-class verification inputs
Date: 2026-04-14
Status: accepted

Decision:
- pre-existing failures must be distinguished from regressions
- repo-level invariants may block acceptance even when task-specific checks pass
- verification must preserve enough evidence to explain that distinction

## D-0011 — Canonical objects and derived views stay explicit
Date: 2026-04-14
Status: accepted

Decision:
- canonical objects remain the source of truth
- views such as WorkLedgerView, GroupSummary, and PanelSummary are derived projections
- UX and helper flows must not become hidden truth stores

## D-0012 — Goalrail is a productized operating layer
Date: 2026-04-15
Status: accepted

Decision:
- Goalrail is designed and sold as a productized operating layer for AI-assisted delivery
- the core promise is contract -> execution -> verify -> proof, not generic agent autonomy
- Goalrail supplements existing tools rather than replacing the stack

## D-0013 — Goalrail keeps a fixed operating core with configurable knobs
Date: 2026-04-15
Status: accepted

Decision:
- the fixed core includes contract-first logic, bounded execution, one primary writer per writable run, and proof as required output
- organization-specific differences are handled through limited knobs such as tracker binding, runtime binding, policy profile, review depth, terminology mapping, approval profile, proof strictness, and scope templates
- configuration must not break the fixed operating core

## D-0014 — Early rollout is managed-first
Date: 2026-04-15
Status: accepted

Decision:
- early Goalrail rollout defaults to managed rollout
- guided rollout comes later after the playbook stabilizes
- Goalrail should not enter as bespoke process redesign per customer

## D-0015 — Commercial entry is free qualification plus paid pilot
Date: 2026-04-15
Status: accepted

Decision:
- the early commercial flow is fit check / qualification first, then a paid pilot
- the first sellable object is a bounded pilot for one team, one repo, and one visible task-to-proof loop
- the pilot ends with an explicit expand / stabilize / stop decision

## D-0016 — Early GTM is RU-first and founder-led
Date: 2026-04-15
Status: accepted

Decision:
- initial market entry is Russian-speaking
- the early sales motion is founder-led and pilot-first
- landing and outreach should be lead-capture and contract-centered, not prompt-tool centric

## D-0017 — Goalrail adopts overlay workspace boundaries and publishing thin binding
Date: 2026-04-20
Status: updated (2026-04-27)

Decision:
- the planning repo uses explicit overlay support planes instead of broad root-level artifact directories
- `.goalrail/work/` tracks bounded goals, reports, and Goalrail delivery memory
- `.goalrail/knowledge/` tracks Goalrail advisory research and ideas
- `.punk/publishing.toml` is the committed repo-local publishing binding manifest; it is the desired source of truth for publishing context
- the runtime publishing workspace (drafts, posts, receipts, manual metrics, generated host wrappers, sessions, credentials, platform cache, and secrets) is external to the project repo and lives in user/platform-local storage
- agents and tools must discover the publishing workspace through: `punk publishing locate --project-root . --json`
- physical paths are platform-native and resolver-owned; they must not be committed into repo docs or manifests
- `.punk/publishing/` legacy directory has been removed from repo after external copy/verify; it is ignored to prevent accidental reintroduction
- repo cleanup is complete; resolver implementation remains pending and semantic cleanup of the external workspace remains future work
- secrets, credentials, browser sessions, and platform cache must not be stored in the repo
- symlinks are intentionally not used as part of the architecture
- `.goalrail/flows/` and `.goalrail/evals/` are reserved as planned spec boundaries for future runtime and verification work
- `apps/`, `scripts/`, and `.github/` remain parked until a bounded implementation slice activates them

## D-0018 — Initial web tooling baseline uses React, Vite, and Mantine packages from npm
Date: 2026-04-23
Status: accepted

Decision:
- runnable frontend resources live in `apps/web/<resource>`
- `apps/web/` is the shared namespace and rules boundary for frontend resources, not a single runnable app
- the original local web demo prototype lives in `apps/web/demo-change-packet`; localized copies can be added as separate web resources when explicitly needed
- the baseline uses React + Vite + Mantine with PostCSS and Vitest wired from the start
- Mantine package versions are aligned to the local source checkout in `~/contrib/mantine`
- direct file-linking to `~/contrib/mantine` is not the default because that checkout does not contain built package artifacts required by consumers
- official Mantine MCP and Mantine skills are the preferred AI assistance layer for this stack
- the current demo remains a prototype and must not be treated as proof of a finished Goalrail web product surface

## D-0019 — Goalrail open-source baseline uses Apache-2.0, DCO, and trademark separation
Date: 2026-04-23
Status: accepted

Decision:
- the repository baseline is Apache License 2.0
- inbound contributions use DCO signoff rather than CLA
- trademark and brand rights stay separate from the code/documentation license
- repository community health files live in root and `.github/` as real governance surfaces
- public OSS posture must not imply that reference screenshots or third-party assets are relicensed automatically

## D-0020 — RU demo is a separate web resource
Date: 2026-04-24
Status: accepted

Decision:
- the RU change-packet demo lives in `apps/web/demo-change-packet-ru`
- it is a copied and localized workspace, not in-app i18n inside `apps/web/demo-change-packet`
- EN and RU demos are intended to be deployed as independent demo surfaces on separate domains

## D-0021 — Real console shell is separate from demo surfaces
Date: 2026-04-25
Status: accepted; target-domain direction updated by D-0074

Decision:
- the real console shell lives in `apps/web/console`
- the target subdomain is `console.goalrail.dev`
- the first console version is intentionally empty except for three left-nav surfaces: Contracts, Delivery Readiness, and Proof
- console visualization follows real CLI/server functionality; UI cards must not imply backend, CLI, server, auth, data, or product-loop implementation before those layers exist

## D-0022 — RU console is a separate web resource
Date: 2026-04-25
Status: superseded by D-0074 for source layout only; live `console.goalrail.ru`
static deployment history remains valid until a separate deployment migration

Decision:
- the RU console shell lives in `apps/web/console-ru`
- the target subdomain is `console.goalrail.ru`
- it is a copied and localized workspace, not in-app i18n inside `apps/web/console`
- the first RU console version mirrors the same empty-surface boundary with Russian labels: Контракты, Оценка готовности, Проверка результата

## D-0023 — RU demo deploys on a separate domain
Date: 2026-04-25
Status: accepted

Decision:
- the EN change-packet demo remains deployed from `apps/web/demo-change-packet` at `demo.goalrail.dev`
- the RU change-packet demo deploys from `apps/web/demo-change-packet-ru` at `demo.goalrail.ru`
- the RU demo is a separate deployment and domain, not a locale switch inside the EN demo
- the `goalrail.ru` DNS record is manually managed outside the infra repo; Kubernetes uses HTTP-01 TLS for this domain

## D-0024 — Go CLI canonical binary and layout
Date: 2026-04-25
Status: accepted

Decision:
- the Go CLI implementation lives under `apps/cli` as a separate module
- the canonical binary name is `goalrail` via `apps/cli/cmd/goalrail`
- `gr` may be introduced later as an optional alias
- `gls`, `glr`, and `gor` are not canonical CLI names
- the first CLI slice is a local/demo bootstrap only and does not implement server integration, production repo auth, hosted execution, gate logic, or proof generation

## D-0025 — Go server canonical boundary and stack
Date: 2026-04-25
Status: accepted

Decision:
- the Go server implementation lives under `apps/server` as a separate module
- the canonical server binary name is `goalrail-server` via `apps/server/cmd/goalrail-server`
- the server is the future owner of canonical Goalrail state, while CLI, skills, web resources, and integrations remain adapters/helpers
- the first server stack is stdlib-first: `net/http`, `encoding/json`, `log/slog`, manual wiring, stdlib tests, plus `github.com/caarlos0/env/v11` for environment config
- the first server slice exposes only `/livez`, `/readyz`, and `/version`
- source-neutral intake is the next meaningful server domain, but this slice has no intake endpoint, database, event log persistence, contract composer, gate, or proof implementation

## D-0026 — Goalrail Go apps use the latest stable Go line
Date: 2026-04-25
Status: accepted

Decision:
- new Goalrail Go modules use the latest stable Go major/minor line by default
- current Goalrail Go apps should stay aligned unless compatibility constraints require otherwise
- patch-level toolchain pinning is not required in `go.mod` by default

Rationale:
- keeps CLI and server Go policy aligned
- avoids minimum-version drift between Goalrail Go apps
- matches the project preference for modern Go idioms and current standard-library capabilities

## D-0027 — Intake promotes to Goal before contract or work
Date: 2026-04-25
Status: accepted

Decision:
- a received `IntakeRecord` may be promoted into a server-owned `Goal`
- `Goal` is normalized intent, not an approved contract and not executable work
- Goal promotion must not create `ContractDraft`, `ApprovedContract`, `WorkItem`, `Task`, `GateDecision`, or `Proof`
- Goal promotion writes explicit events such as `goal.created` and `intake.promoted_to_goal`
- CLI, skills, web resources, and integrations remain adapters; they do not own Goal truth

Rationale:
- preserves the product chain from raw intake to normalized intent before clarification and contract composition
- prevents raw intake from collapsing directly into contract or execution scope
- gives the next server implementation slice a bounded target without expanding into contract/gate/proof work

## D-0028 — Goal readiness precedes clarification and contract
Date: 2026-04-25
Status: accepted

Decision:
- a created `Goal` may be evaluated into `needs_clarification`, `ready_for_contract_seed`, or `rejected`
- readiness is a server-owned Goal state transition in the intent plane
- readiness does not create `ClarificationRequest`, `ContractDraft`, `WorkItem`, `GateDecision`, or `Proof`
- the first readiness behavior should be deterministic and inspectable, not LLM-driven
- readiness events should record the Goal state transition and reason codes

Rationale:
- keeps Goal as normalized intent while defining the next bounded server step
- prevents contract generation from being triggered before missing information is assessed
- separates readiness decisions from a later clarification question/answer lifecycle

## D-0029 — Clarification requests preserve server-owned answer truth
Date: 2026-04-25
Status: accepted

Decision:
- a Goal in `needs_clarification` may create a server-owned `ClarificationRequest`
- `ClarificationRequest` groups missing-information questions for a target actor or role
- `ClarificationAnswer` is canonical evidence of submitted answers and is not approval
- answers may update Goal intent-plane hints through a server-owned transition, but they must not make Goal the only place answer content lives
- clarification does not create contract seed, `ContractDraft`, `WorkItem`, `GateDecision`, or `Proof`
- CLI, skills, web resources, and integrations may transport clarification questions and answers, but they do not own canonical clarification truth

Rationale:
- preserves an audit trail for missing information before contract generation
- keeps clarification separate from approval and executable work
- gives the next server implementation slice a bounded target without expanding into contract/gate/proof work

## D-0030 — Repository checkout and checks run behind runner boundary
Date: 2026-04-26
Status: accepted

Decision:
- repository checkout, workspace preparation, code inspection, and check execution belong behind a dedicated runner boundary
- the Goalrail API server owns canonical state, scheduling decisions, task packets, run records, event append, and proof input references, but must not clone repositories or run checks in-process
- Goalrail supports both `goalrail_hosted_runner` and
  `customer_hosted_runner` runner modes
- customer-hosted runners are first-class for security-sensitive teams, self-managed VCS, private networks, and customers that do not allow repository contents to leave their environment
- customer-hosted runners may operate without Goalrail cloud-side VCS connection or clone credential
- VCS provider discovery, repository records, repo bindings, and checkout permission are separate concerns
- `RepoBinding` identifies which repository a Project works with, but does not itself authorize checkout
- checkout authority is determined by runner mode, policy, and checkout access mode
- runners produce receipts and artifacts; final decision remains gate-owned
- persistent full-repository mirrors and repository write access are out of scope for the MVP runner boundary

Rationale:
- keeps the server as source-of-truth owner without turning it into a hidden CI or DevOps platform
- lets early managed pilots use hosted runners where allowed while preserving a path for customer-owned repository access
- supports stronger security posture before GitHub, GitLab, Bitbucket, or custom Git connectors are implemented

## D-0031 — First runner prototype is hosted read-only checkout
Date: 2026-04-26
Status: accepted

Decision:
- first runnable runner prototype starts with `goalrail_hosted_runner`
- first prototype uses a Goalrail-operated hosted runner pool
- hosted runner workers use pull-based / poll-based job leasing from the API server
- first prototype performs read-only ephemeral checkout and returns a checkout receipt
- customer-hosted runner remains first-class in the architecture model, but is deferred from the first implementation slice
- first prototype does not implement repository writes, persistent mirrors, arbitrary command execution, customer-hosted runner installer/registration/auth, gate, or proof

Rationale:
- keeps the first implementation small and testable
- proves the runner boundary without building a full hosted execution platform
- preserves future customer-hosted runner path without blocking MVP progress


## D-0032 — Clarification answers record evidence before application
Date: 2026-04-26
Status: accepted

Decision:
- an open `ClarificationRequest` may record a server-owned `ClarificationAnswer`
- `ClarificationAnswer` is canonical evidence, not approval and not executable work
- answer recording does not update Goal hints, trigger readiness re-check, or create contract seed, `ContractDraft`, `WorkItem`, `GateDecision`, or `Proof`
- a request may transition from `open` to `answered` after successful answer recording
- answer application to Goal hints and Goal readiness re-check are later explicit server-owned transitions

Rationale:
- preserves an auditable answer record before any derived Goal update
- keeps clarification answer storage separate from contract and work creation
- gives the next implementation slice a bounded target without hidden state transitions

## D-0033 — MVP uses direct RepoBinding before RepositoryRecord
Date: 2026-04-26
Status: accepted

Decision:
- MVP uses `User`, `Organization`, `OrganizationMembership`, `Project`, and
  `RepoBinding`
- `RepoBinding` stores repository reference directly
- `RepositoryRecord` and `RepositoryEnrollment` are deferred
- manual or CLI-initialized `RepoBinding` is enough for the repository-context
  foundation
- provider UI integrations are not active MVP scope

Rationale:
- reduces entity count for the first server MVP
- avoids building a repository catalog before the product contour needs it
- keeps repository context independent from provider integration

## D-0034 — Server persistence uses pgx, Squirrel, and goose
Date: 2026-04-26
Status: accepted

Decision:
- `pgx/v5` is used for PostgreSQL execution, pool, and transactions
- Squirrel is used for runtime SQL statement construction in Go code
- Squirrel is not used as executor
- `goose` is used for migrations
- `sqlc` is not used
- ORM is not used
- before production, one editable init migration is allowed
- dev seed is separate from migrations and writes to Postgres

Rationale:
- keeps persistence native and explicit
- avoids ORM and generated-code overhead
- keeps migration history clean before production

## D-0035 — Answer application updates Goal hints only
Date: 2026-04-26
Status: accepted

Decision:
- a recorded `ClarificationAnswer` may be applied to Goal intent-plane hints through a server-owned transition
- answer application updates only allowed Goal mappings: `goal.summary`, `goal.intent_owner`, `goal.scope_hint`, and `goal.acceptance_hint`
- answer application preserves `ClarificationAnswer` as canonical evidence and must not make Goal hints the only answer record
- answer application does not trigger readiness re-check, create contract seed, `ContractDraft`, `WorkItem`, `GateDecision`, or `Proof`, approve anything, or make work executable
- v0 application should be deterministic, require `applied_by`, reject unsupported mappings, and return `409 already_applied` on repeated application
- v0 must not map arbitrary raw text into `goal.intent_owner`; it should require an explicit actor-shaped value or defer that mapping

Rationale:
- keeps answer evidence, hint mutation, readiness, and contract generation as separate inspectable transitions
- prevents hidden transition chains from turning clarification into contract or executable work
- gives the next implementation slice a bounded target without introducing readiness or contract semantics

## D-0036 — Readiness re-check after applied answers stays explicit
Date: 2026-04-26
Status: accepted

Decision:
- after answer application updates Goal intent-plane hints, readiness re-check remains an explicit server-owned transition
- the recommended prototype direction is to reuse `POST /v1/goals/{id}/readiness` for the explicit re-check
- readiness re-check may move Goal to `needs_clarification`, `ready_for_contract_seed`, or `rejected`
- `ready_for_contract_seed` is Goal state only and does not create contract seed, `ContractDraft`, `WorkItem`, `GateDecision`, or `Proof`
- answer application must not automatically call readiness re-check
- this boundary does not modify ADR-0010 persistence or introduce new durable storage requirements

Rationale:
- keeps answer application, readiness, and contract seed as separate auditable transitions
- prevents hidden transition chains from turning clarified intent into contract or executable work
- gives the next implementation slice a bounded target using the existing readiness-check endpoint

## D-0037 — ContractSeed is explicit canonical bridge before drafting
Date: 2026-04-26
Status: accepted

Decision:
- `ContractSeed` may be created only from a Goal whose state is `ready_for_contract_seed`
- seed creation is an explicit server-owned transition, not an automatic side effect of readiness re-check
- `ContractSeed` is canonical state and a snapshot of readiness-checked Goal intent for future contract drafting
- `ContractSeed` is not `ContractDraft`, not an approved contract, not executable work, and not approval
- `ContractSeed` must not create `WorkItem`, `GateDecision`, or `Proof`
- repeated seed creation should return `409 already_seeded` in v0
- this boundary does not modify ADR-0010 persistence or introduce new durable storage requirements

Rationale:
- preserves a clear bridge between intent-plane readiness and contract drafting
- avoids hidden transition chains from readiness to contract artifacts or executable work
- gives the next implementation slice a bounded target before `ContractDraft` generation

## D-0038 — ContractDraft is draft state before approval
Date: 2026-04-26
Status: accepted

Decision:
- `ContractDraft` may be created explicitly from `ContractSeed(created)`
- `ContractDraft` is canonical server-owned draft state containing proposed contract terms
- `ContractDraft` is not an approved Contract, not executable work, and not approval
- `ContractDraft` creation must not create `WorkItem`, start execution, write `GateDecision`, or create `Proof`
- `ContractDraft` creation must not mutate `ContractSeed`; the seed remains `created` unless a later boundary defines a transition
- repeated draft creation for the same `ContractSeed` should return `409 already_drafted` in v0
- this boundary does not modify ADR-0010 persistence or introduce new durable storage requirements

Rationale:
- preserves a bounded drafting stage between ContractSeed and approval
- prevents proposed terms from being treated as approved scope or runnable work
- gives the next implementation slice a focused target before approval, task shaping, gate, and proof boundaries

## D-0039 — Intake, Goal, and EventLog persist in Postgres
Date: 2026-04-26
Status: accepted

Decision:
- IntakeRecord, Goal, and EventLog move from in-memory stores to Postgres-backed stores
- events table is durable audit trail v0, not queue/event bus/outbox
- event IDs use UUIDv7
- events include internal `event_sequence` for DB-local ordering
- payload and artifact refs use jsonb
- clarification persistence is deferred
- ContractSeed persistence is deferred
- v0 event append remains synchronous; shared transaction wrappers are addressed by D-0041
- current HTTP behavior is preserved without adding new list/search endpoints

Rationale:
- makes current core flow survive server restarts
- keeps persistence layer bounded before contract/gate/proof
- avoids introducing async infrastructure too early

## D-0040 — ContractDraft review/update stays draft-only
Date: 2026-04-26
Status: accepted

Decision:
- `ContractDraft` may be reviewed and updated only through explicit server-owned transitions
- updates affect proposed draft fields only and keep `ContractDraft.state = draft`
- editable fields are `title`, `intent_summary`, `proposed_scope`, `proposed_non_goals`, `proposed_constraints`, `proposed_acceptance_criteria`, `proposed_expected_checks`, `proposed_proof_expectations`, and `risk_hints`
- identity/source fields, `source_refs`, `created_at`, and `state` are not editable in this boundary
- updates must write `contract_draft.updated` events with changed fields, `updated_by`, old/new values where safe, and timestamp
- updates do not approve `ContractDraft`, create approved Contract, create `WorkItem`, start execution, write `GateDecision`, or create `Proof`
- `ready_for_approval` remains a later boundary
- this boundary does not modify ADR-0010 persistence or introduce new durable storage requirements

Rationale:
- allows human review/editing of proposed draft terms before approval
- preserves auditability of draft changes without collapsing update into approval
- keeps work item, execution, gate, and proof boundaries downstream of approved Contract

## D-0041 — Canonical writes and event appends share Postgres transactions
Date: 2026-04-26
Status: accepted

Decision:
- Postgres-backed intake create and `intake.received` event append run in one transaction
- Goal promotion and its `goal.created` / `intake.promoted_to_goal` events run in one transaction
- Goal readiness update and readiness events run in one transaction
- events remain synchronous durable audit trail v0, not queue/event bus/outbox
- Postgres store execution uses a private transaction context convention so ordinary store methods use the active transaction when present
- no generic Unit of Work framework is introduced

Rationale:
- prevents canonical records without corresponding audit events
- hardens durable core flow before broader contract/gate/proof work
- keeps persistence simple and synchronous

## D-0042 — ContractSeed and ContractDraft persist in Postgres
Date: 2026-04-26
Status: accepted; runtime config/fallback details superseded by D-0071

Decision:
- ContractSeed and ContractDraft creation move from in-memory-only stores to
  Postgres-backed stores when database config is present
- the original no-DB memory path is no longer production route behavior after
  D-0071; memory stores remain test/dev helpers only
- `contract_seed.created` and `contract_draft.created` append to the durable
  EventLog
- ContractSeed create plus event append and ContractDraft create plus event
  append run in one Postgres transaction
- approval, work items, runner, gate, and proof remain later boundaries

Rationale:
- makes the contract bridge and draft creation state survive server restarts
- keeps seed and draft creation auditable without introducing queue, outbox,
  event bus, sqlc, or ORM
- preserves the boundary that draft state is not approved or executable work

## D-0043 — ContractDraft ready_for_approval is a pre-approval state
Date: 2026-04-26
Status: accepted

Decision:
- `ContractDraft` may transition explicitly from `draft` to `ready_for_approval`
- `ready_for_approval` is a `ContractDraft` state, not approved Contract
- the transition requires minimum completeness checks for title, intent summary, proposed scope, proposed acceptance criteria, proposed proof expectations, repo binding, contract seed, and Goal linkage
- the transition records `marked_by` as audit identity only, not approval authority
- the transition writes `contract_draft.marked_ready_for_approval`
- the transition must not mutate proposed fields; proposed-field edits stay in the ContractDraft update boundary
- the transition does not approve a Contract, create approved Contract, create `WorkItem`, start execution, write `GateDecision`, or create `Proof`
- no new table is introduced; the pre-production init migration allows `draft` and `ready_for_approval` in the existing `contract_drafts.state` check

Rationale:
- creates an auditable handoff point between draft review/update and later approval
- keeps completeness checks separate from approval authority
- prevents draft readiness from becoming executable work or gate/proof semantics

## D-0044 — ApprovedContract is a separate approval snapshot
Date: 2026-04-26
Status: accepted

Decision:
- approval is an explicit server-owned boundary from `ContractDraft(ready_for_approval)` to `ApprovedContract`
- `ApprovedContract` is a canonical approved snapshot, not just a draft state change
- `approved_by` is the approval actor and must be recorded; it is not inferred from `marked_by`
- recommended v0 behavior is to not mutate `ContractDraft` during approval
- repeated approval should return `409 already_approved`
- approval writes `contract.approved`
- approval does not create `WorkItem`, plan tasks, start execution, write `GateDecision`, or create `Proof`
- WorkItem planning remains a later explicit boundary after approved Contract
- this boundary does not introduce storage or migration requirements by itself

Rationale:
- separates draft history from approved contract truth
- keeps approval distinct from execution planning and delivery verification
- prevents approval from becoming gate/proof or task creation semantics

## D-0045 — ApprovedContract persists in Postgres with evented approval
Date: 2026-04-26
Status: accepted

Decision:
- approving `ContractDraft(ready_for_approval)` creates a separate `ApprovedContract(approved)` snapshot
- approval requires `approved_by` and records it on the snapshot and `contract.approved` event
- the snapshot copies current draft terms and source refs without mutating `ContractDraft`
- Postgres-backed approval inserts `approved_contracts` and appends `contract.approved` in one transaction
- repeated approval for the same `contract_draft_id` returns `409 already_approved`
- approval does not create `WorkItem`, plan tasks, start execution, write `GateDecision`, or create `Proof`

Rationale:
- makes approved contract truth durable before work planning exists
- preserves the separation between approval and execution
- keeps audit events synchronous without queue, outbox, event bus, sqlc, or ORM

## D-0046 — WorkItem planning is a non-executable boundary
Date: 2026-04-27
Status: accepted

Decision:
- WorkItem planning is an explicit server-owned boundary from
  `ApprovedContract(approved)` to `WorkItem(planned)`
- WorkItems are canonical planning units derived from approved scope,
  acceptance criteria, and proof expectations
- recommended v0 planning creates one planned WorkItem per ApprovedContract
- repeated planning should return `409 already_planned`
- `owner_hint` is advisory only and does not assign or claim work
- WorkItem planning writes `work_item.created`
- WorkItem planning does not start execution, create `Run`, checkout a repo,
  submit a receipt, write `GateDecision`, or create `Proof`
- assignment, claiming, runtime task packets, runner checkout, execution,
  receipt submission, gate, and proof remain later explicit boundaries
- this boundary does not introduce storage or migration requirements by itself

Rationale:
- gives approved contracts a bounded non-executable planning handoff
- keeps approval separate from work planning and work planning separate from
  execution
- prevents WorkItem creation from becoming hidden runner, receipt, gate, or
  proof semantics

## D-0047 — Public landing demo remains local-only and deterministic
Date: 2026-04-28
Status: accepted

Decision:
- `apps/web/pilot-intake-ru` demonstrates GoalRail's contract-first
  workflow through local deterministic functions (`detectScenario`,
  `buildContractDraft`, `buildReviewReport`, `deriveOutcomeTone`,
  `buildOutcomeReport`)
- the pilot-first interactive landing demo must not call LLMs, AI APIs,
  backend services, repo providers, analytics endpoints, or execution
  runtimes
- the pilot-first interactive landing demo must not persist user input
  in any form (memory beyond the React tree, network, storage)
- the final outcome CTA only focuses the existing email input and may
  apply a temporary local highlight class; it does not POST anything
- the current email lead remains a `mailto:` form; backend handoff for
  email capture requires a separate explicit decision
- new scenarios beyond `manual_review_gate` and `bounded_task` require
  an explicit decision before implementation
- chat-style UI (history, user/assistant turns, avatars, model selector)
  is not allowed
- file upload is not allowed
- a fake numeric readiness score must not be presented as if it were
  real measurement
- canonical copy, scenario rules, and accessibility hardening are
  recorded in `docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md`

Rationale:
- GoalRail's value is the method: intake, clarification, contract,
  review, honest outcome
- a local deterministic demo is safer, cheaper, and more honest for
  first public/pilot positioning than a live AI/agent demo
- it avoids turning the public landing into a generic AI/chat demo
- it preserves trust by not implying that user tasks are executed,
  that real repos are assessed, or that GoalRail has delivered work
- it keeps the public surface aligned with `GOALRAIL_PRODUCT_CONCEPT.md`
  and `GOALRAIL_OPERATING_MODEL.md` instead of drifting into autonomy
  claims

## D-0048 — Public RU pilot-first landing demo candidate approved
Date: 2026-04-28
Status: accepted

Decision:
- `apps/web/pilot-intake-ru` is approved as the candidate public RU
  pilot-first interactive landing demo surface.
- The basis for this approval is the completed internal review captured in
  `docs/ops/PILOT_INTAKE_RU_INTERNAL_REVIEW_NOTES.md`, whose
  recommendation was `READY WITH WARNINGS — READY FOR PUBLIC-DOMAIN
  DECISION`.
- The only review warning is narrow hero-title tightness at very-narrow
  widths around 380px; it is non-blocking and does not gate this approval.
- This decision is approval of the demo surface as the public candidate.
  It is not deployment wiring.
- Publication / deployment requires a separate deployment-prep slice
  recorded in `docs/ops/NEXT.md`.
- D-0047 continues to govern the surface in full: no backend, no
  LLM/API, no repo provider integration, no code execution, no
  persistence, no analytics or session tracking, no chat UI, no file
  upload, no model selector.
- Email lead capture remains `mailto:` / focus-only / manual handoff.
  Any move to backend submission requires its own separate explicit
  decision.
- Analytics remain disallowed. Enabling analytics requires its own
  separate explicit decision.
- New scenarios beyond `manual_review_gate` and `bounded_task`, new
  outcome tones, repo-provider integration, runtime execution, and
  persistence remain disallowed under D-0047 and are not unlocked by
  this decision.

Rationale:
- The 5-step interactive walkthrough demonstrates the GoalRail method
  (intake → clarification → contract → review → honest outcome) more
  faithfully than a static landing page would, while staying local and
  deterministic.
- The surface is explicit about its boundaries: code is not executed,
  repos are not connected, no result is delivered, no fake numeric
  readiness score is shown. This is a stronger trust posture than a
  generic AI/agent demo.
- Canonical copy and governance are recorded in
  `docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md`, so future
  contributors and reviewers have a single source of truth.
- The internal review evidence base (matrix across visual / responsive /
  keyboard / a11y / boundary / outcome flows) is captured in
  `docs/ops/PILOT_INTAKE_RU_INTERNAL_REVIEW_NOTES.md` and references
  this decision.

Consequences:
- `docs/ops/NEXT.md` is updated to point to a `Pilot intake RU
  deployment prep` slice covering domain/surface confirmation, build
  and hosting path without backend behavior, D-0047 boundary
  re-confirmation in the deployment context, optional CSS-only
  very-narrow hero polish, production build/smoke check, secrets/env
  audit, public-copy parity check against
  `docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md`, and verification
  that email capture remains `mailto:` / focus-only / manual handoff.
- `docs/ops/STATUS.md` reflects this candidate-approved state without
  implying deployment, live status, backend, or analytics.
- `docs/ops/COMPONENTS.yaml` `web_surface.notes` is updated with a
  short candidate-approved marker pointing at this decision.
- Any future public-domain/hosting work must preserve D-0047 in full
  and must not introduce backend, analytics, email submission, repo
  integration, or runtime execution without a separate explicit
  decision.

## D-0049 — Pilot intake RU target domain and hosting surface selected
Date: 2026-04-28
Status: superseded by D-0053 for target domain and canonical public URL (active target is now `pilot.goalrail.ru`; the hosting-surface portion was already addressed by D-0050 → D-0051 supersession; D-0049 body is preserved as historical record)

Decision:
- `apps/web/pilot-intake-ru` will be prepared for publication at the
  target domain `pilot.goalrail.dev` with public path `/`.
- Hosting target: `static CDN target TBD` — the concrete CDN /
  static-bucket / DNS surface is not picked in this decision and
  remains a deployment-wiring detail. The chosen surface, when picked,
  must be static-only.
- Public status: `candidate-public` — the surface is the chosen
  candidate for public publication; it is not yet deployed and is not
  yet live.
- This decision unlocks a future `Pilot intake RU deployment wiring`
  slice (recorded in `docs/ops/NEXT.md`) by satisfying the
  domain-decision gating prerequisite identified in
  `docs/ops/PILOT_INTAKE_RU_DEPLOYMENT_PREP.md` §5 / §10.
- This decision does not deploy the surface.
- This decision does not create DNS, hosting, CDN, or build wiring.
- This decision does not approve backend email capture.
- This decision does not approve analytics.
- This decision does not change D-0047 or D-0048; both continue to
  govern the surface in full.
- Because `PUBLIC_PATH` is `/`, deployment wiring must verify root-path
  behavior and static asset paths. No `vite.config.ts` `base` adjustment
  is required at this time.
- Email lead capture remains `mailto:` / focus-only / manual handoff.
  Any move to backend submission requires its own separate explicit
  decision.
- Analytics remain disallowed. Enabling analytics requires its own
  separate explicit decision.
- Deployment wiring must remain static-only. No backend, no serverless
  functions, no server-rendered routes are introduced by this decision
  or by any wiring slice that follows from it.
- If the chosen target domain is later changed (different host, different
  public path, or both), that must be recorded as a separate explicit
  decision in `docs/ops/DECISIONS.md`. This decision pins
  `pilot.goalrail.dev` + `/` until such a future decision supersedes it.

Rationale:
- Phase 8B deployment-prep
  (`docs/ops/PILOT_INTAKE_RU_DEPLOYMENT_PREP.md`) found the candidate
  surface static-hostable with no env vars, secrets, or backend
  dependencies, and explicitly recorded target-domain selection as a
  blocking pre-requisite for the deployment wiring slice.
- A target domain / public path is needed to define the asset path
  layout, the smoke-check URL, the canonical link target in landing
  copy, and the publishing instructions a wiring slice will follow.
- Recording the decision now prevents accidental publication to the
  wrong surface and prevents drift between docs (canonical landing copy,
  status, deployment prep) and the surface that will eventually be
  served.
- Keeping this docs-only preserves the local-only deterministic
  boundary recorded in D-0047 and the candidate-approval recorded in
  D-0048 — no code, hosting config, DNS, or runtime artifacts are
  introduced by this decision.

Consequences:
- `docs/ops/NEXT.md` is updated: the
  `Pilot intake RU deployment wiring` slice no longer carries a
  domain-decision gating prerequisite; its done-means now references
  the values pinned here.
- `docs/ops/STATUS.md` is updated with a concise marker that the
  target-domain decision is recorded and that deployment wiring is
  still pending.
- `docs/ops/PILOT_INTAKE_RU_DEPLOYMENT_PREP.md` §5 (`Target domain`)
  and §8 (`Known pre-publish risks`) are updated to point at this
  decision; the recommendation remains `READY WITH WARNINGS`.
- `docs/ops/COMPONENTS.yaml` `web_surface.notes` is updated with a
  short marker that target domain / hosting is recorded and deployment
  wiring is the next slice.
- The deployment wiring slice that follows must:
  - remain static-only;
  - run a production build;
  - run `vite preview` (or equivalent) and a smoke check across all 5
    walkthrough steps;
  - verify no env or secrets assumptions;
  - verify canonical copy parity against
    `docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md`;
  - verify the `mailto:` / focus / manual email handoff still holds;
  - re-confirm D-0047 boundaries in the deployed/preview context.
- The surface remains not deployed and not live until a future
  deployment-wiring patch completes. This decision does not by itself
  authorise publication.

## D-0050 — Pilot intake RU static hosting provider selected
Date: 2026-04-28
Status: superseded by D-0051 for hosting provider and deployment mode (Cloudflare Pages Direct Upload is no longer the selected RU launch path; Cloudflare Pages, Workers, Functions, KV/R2/D1/Durable Objects/Queues, proxy/CDN, and Web Analytics remain disallowed for this surface per D-0051)

Decision:
- `apps/web/pilot-intake-ru` will use **Cloudflare Pages** for the
  candidate public static surface at `https://pilot.goalrail.dev/`.
- **Hosting target detail:** Cloudflare Pages Direct Upload project for
  pilot-intake-ru; static assets from `apps/web/pilot-intake-ru/dist`;
  the concrete project name on Cloudflare Pages is to be confirmed
  during the deployment wiring slice (see `Pre-conditions` below).
- **DNS strategy:** add `pilot.goalrail.dev` as a Cloudflare Pages
  custom domain, then configure/confirm DNS for `pilot.goalrail.dev`
  to point to the Cloudflare Pages target. If `goalrail.dev` DNS is
  already managed in Cloudflare, allow Cloudflare to create or manage
  the required CNAME during the custom-domain setup; otherwise DNS is
  handled externally by the operator after Pages provides the required
  target.
- **TLS strategy:** Cloudflare-managed TLS. Deployment wiring must
  verify HTTPS is active for `https://pilot.goalrail.dev/` before any
  public use.
- **Deployment mode:** static-only manual Direct Upload after a local
  production build. Use Wrangler or the Cloudflare Pages dashboard to
  upload prebuilt assets from `apps/web/pilot-intake-ru/dist`. **No Git
  integration** and **no automatic redeploys** unless a future explicit
  decision changes the deployment model.
- **Preview mode:** Cloudflare Pages preview deployment /
  `*.pages.dev` preview URL before DNS cutover, plus the local
  `vite preview` smoke check. If provider preview is not yet available
  before the first upload (i.e. before the Cloudflare Pages project
  exists), the wiring slice must record that and rely on local preview
  until the Pages project is created.

Pre-conditions for the deployment-wiring slice authorised by this
decision:
- the deployment-wiring slice must verify whether a suitable
  Cloudflare Pages project name is available **before** recording any
  final provider config in the repo. If the desired name is taken or
  reserved, the wiring slice records the actual project name in
  `docs/ops/PILOT_INTAKE_RU_DEPLOYMENT_WIRING.md` (and, only if a
  different surface or naming pattern is implied, raises the question
  via a separate explicit decision rather than silently picking).

Constraints (the decision itself does not change runtime; these guard
the next slice):
- This decision **does not deploy** the surface.
- This decision **does not create DNS records**.
- This decision **does not add provider config** (e.g. `wrangler.toml`,
  `_redirects`, `_headers`, GitHub Actions workflow). Any minimal
  provider-side config the wiring slice adds must be static-only and
  must not introduce Cloudflare Workers, Functions, KV, R2, D1,
  Durable Objects, Queues, or any other dynamic / stateful Cloudflare
  surface beyond static asset delivery and TLS termination.
- This decision **does not add a CI deploy workflow**. The Direct
  Upload model is explicitly chosen so that production deploys remain
  operator-gated and not triggered by Git events.
- This decision **does not approve backend email capture**.
- This decision **does not approve analytics or session tracking** —
  Cloudflare Pages Web Analytics (or any equivalent) must remain
  disabled by default and must not be enabled without a separate
  explicit decision.
- This decision **does not change D-0047, D-0048, or D-0049**; all
  three continue to govern the surface in full.
- Email lead capture remains `mailto:` / focus-only / manual handoff.
- Any backend / email / analytics / dynamic-edge change requires a
  separate explicit decision.
- Direct Upload means **no automatic Git-based production deploys** in
  this project unless a future explicit decision creates a different
  deployment model.

Rationale:
- Phase 8D found no repo-wide static-hosting convention to inherit,
  and explicitly recorded the deployment-wiring slice as
  `BLOCKED ON HOSTING PROVIDER SELECTION` until a concrete provider
  was chosen.
- A concrete provider decision is required before any provider-specific
  static-hosting config can be added to the repo.
- Cloudflare Pages provides static-asset hosting with managed TLS, a
  free tier appropriate for a pilot landing surface, native support
  for custom domains under DNS that Cloudflare can already manage, and
  Direct Upload mode for operator-controlled deployments.
- Direct Upload (rather than the Git-integration mode) preserves
  D-0047's local-only deterministic boundary by keeping production
  deploys an explicit operator action rather than an automatic
  side-effect of pushing to a branch.
- Static-only usage (no Workers, Functions, KV, R2, D1, Durable
  Objects, Queues) keeps the surface aligned with D-0047 — no backend,
  no execution, no persistence, no analytics.
- Recording the provider as a decision (rather than picking it inside
  the wiring slice) prevents accidental deployment to an unintended
  surface and gives the wiring slice an unambiguous target.

Consequences:
- `docs/ops/NEXT.md` is updated: the
  `Pilot intake RU hosting provider selection (blocker)` slice is
  marked DONE by this decision; the
  `Pilot intake RU provider-specific deployment wiring` slice (formerly
  the `(post-blocker)` deployment-wiring slice) becomes the active next
  slice with concrete Cloudflare Pages values folded in.
- `docs/ops/STATUS.md` is updated with a concise marker that the
  hosting provider decision is recorded, that provider-specific
  deployment wiring is still pending, and that the surface is not
  deployed.
- `docs/ops/PILOT_INTAKE_RU_DEPLOYMENT_WIRING.md` §5 (`Hosting wiring
  status`) is updated from `BLOCKED ON HOSTING PROVIDER SELECTION` to
  `PROVIDER SELECTED — WIRING PENDING`, with provider values and a
  pointer to this decision; §9 (`Recommendation`) is updated
  accordingly.
- `docs/ops/COMPONENTS.yaml` `web_surface.notes` is updated with a
  short marker that the provider decision is recorded and that
  provider-specific wiring is the next slice.
- The provider-specific deployment-wiring slice that follows must:
  - remain static-only (no Workers / Functions / KV / R2 / D1 /
    Durable Objects / Queues / Cloudflare Pages Web Analytics);
  - use the D-0049 values: domain `pilot.goalrail.dev`, public path
    `/`;
  - verify the canonical URL in built `dist/index.html` is
    `https://pilot.goalrail.dev/` (already aligned in Phase 8E);
  - run the production build locally and a `vite preview` smoke check
    plus, when available, a Cloudflare Pages preview / `*.pages.dev`
    smoke check before DNS cutover;
  - verify no env/secrets/runtime configuration is required;
  - verify the `mailto:` / focus / manual email handoff still holds;
  - verify HTTPS is active on the chosen Cloudflare Pages target and
    on `https://pilot.goalrail.dev/` once the custom domain is added;
  - re-confirm D-0047 boundaries in the deployed/preview context.
- The surface remains not deployed and not live until a future
  deployment-wiring patch completes. This decision does not by itself
  authorise publication.

## D-0051 — Pilot intake RU hosting path changed to operator-managed SSH static server
Date: 2026-04-28
Status: accepted
Supersedes: D-0050 (hosting provider and deployment mode only)

Decision:
- `apps/web/pilot-intake-ru` will be prepared for manual static
  deployment to an **operator-managed SSH static server**.
- This decision **supersedes D-0050** for hosting provider and
  deployment mode. Cloudflare Pages Direct Upload is no longer the
  selected RU launch path.
- D-0049 is **preserved**: target domain remains `pilot.goalrail.dev`
  and public path remains `/` with public status `candidate-public`.
- D-0047 and D-0048 are **preserved** and continue to govern the
  surface in full.
- **Hosting provider:** operator-managed SSH static server.
- **Hosting target detail:** operator-managed Linux server reachable
  over SSH; exact host, IP address, SSH port, SSH user, and
  credentials are kept out of repo; static web root and release
  directory will be confirmed during deployment wiring.
- **DNS strategy:** DNS handled externally by the operator;
  `pilot.goalrail.dev` will point to the SSH server or upstream
  reverse proxy using A / AAAA / CNAME as appropriate. If the DNS
  zone is currently managed through Cloudflare, the record must be
  DNS-only / non-proxied or otherwise configured so public traffic
  does **not** depend on Cloudflare Pages, Cloudflare proxy,
  Cloudflare Workers, or Cloudflare CDN services.
- **TLS strategy:** server-managed HTTPS via existing reverse proxy
  or Let's Encrypt. HTTPS for `https://pilot.goalrail.dev/` must be
  verified before any public use.
- **Deployment mode:** manual static upload over SSH after a local
  production build. Preferred mechanism is `rsync` / `scp` to a
  timestamped release directory with an atomic `current` symlink
  switch. **No automatic redeploys.** No CI deploy workflow.
- **Preview mode:** local `vite preview` smoke check plus a server
  smoke check after manual upload. An optional staging vhost / path
  is allowed only if the operator explicitly provides one; this
  decision does not require staging infrastructure and does not
  authorise its creation.
- **Public status:** `candidate-public`. Surface is not deployed and
  not live until the future deployment-wiring patch completes with
  HTTPS verified.

Constraints (the decision itself does not change runtime; these guard
the next slice):
- This decision **does not deploy** the surface.
- This decision **does not create DNS records**.
- This decision **does not add SSH scripts**.
- This decision **does not add Nginx, Caddy, Apache, or other
  reverse-proxy config** to the repo. Any minimal server-side config
  the wiring slice creates lives on the operator-managed server, not
  in this repository, unless a separate explicit decision authorises
  committing repo-side server config.
- This decision **does not approve backend email capture**.
- This decision **does not approve analytics or session tracking**.
- This decision **does not approve server-side forms**.
- This decision **does not approve persistence**.
- This decision **does not approve runtime execution**.
- This decision **does not approve Cloudflare Pages, Cloudflare
  Workers, Cloudflare Functions, Cloudflare KV / R2 / D1 / Durable
  Objects / Queues, Cloudflare proxy/CDN, or Cloudflare Web
  Analytics** for this surface. If the DNS zone happens to live in
  Cloudflare, the record must be DNS-only / non-proxied; Cloudflare
  surfaces beyond plain DNS hosting are not part of the launch path.
- Server hostnames, IP addresses, usernames, SSH keys, tokens, and
  credentials **must not** be committed to the repository.
- Email lead capture remains `mailto:` / focus-only / manual handoff.
- Any backend / email / analytics / dynamic-edge change requires a
  separate explicit decision.
- Deployment wiring must remain static-only and operator-gated.
- Deployment wiring should prefer an atomic release strategy: upload
  to a timestamped release directory; switch the `current` symlink;
  keep at least one previous release for rollback.

Rationale:
- The RU-segment launch path should not depend on Cloudflare Pages
  availability or any Cloudflare-managed surface beyond plain DNS
  hosting.
- An operator-managed SSH static server preserves manual control over
  publication, rollback, and operational state.
- Static upload over SSH fits the existing local-only deterministic
  demo (D-0047) without introducing provider features such as
  analytics, server functions, edge workers, or automatic Git deploys.
- Atomic release strategy with timestamped directories and a
  `current` symlink gives a fast, low-risk rollback path that the
  operator can execute without touching the repo.
- Keeping production deployments operator-gated (no Git integration,
  no CI deploy, no provider-driven auto-build) preserves D-0047's
  trust posture and prevents accidental scope drift into automatic
  pipelines.
- Recording the change as an explicit supersession (rather than
  silently substituting the provider in D-0050) keeps the public
  audit trail honest and prevents readers from acting on the
  Cloudflare Pages instructions in D-0050.

Consequences:
- D-0050 status is updated from `accepted` to
  `superseded by D-0051 for hosting provider and deployment mode`.
  D-0050's body remains in the file as historical record but is no
  longer the active hosting path.
- D-0049 remains in force unchanged: `pilot.goalrail.dev` / `/` /
  `candidate-public`. The Phase 8E canonical-link metadata fix in
  `apps/web/pilot-intake-ru/index.html` (and built `dist/index.html`)
  is still correct.
- `docs/ops/NEXT.md` is updated: the `Slice — Pilot intake RU
  provider-specific deployment wiring` (Cloudflare-Pages-shaped) is
  replaced by `Slice — Pilot intake RU SSH static deployment wiring`
  with concrete SSH-shaped done-means.
- `docs/ops/STATUS.md` is updated with a concise marker that the
  Cloudflare Pages launch path is superseded for the RU segment, that
  the new SSH static path is selected per this decision, and that the
  surface is not deployed.
- `docs/ops/PILOT_INTAKE_RU_DEPLOYMENT_WIRING.md` §5 (`Hosting wiring
  status`) and §9 (`Recommendation`) are updated to point at this
  decision rather than D-0050; provider values, DNS, TLS, deployment,
  and preview modes are restated for the SSH path.
- `docs/ops/COMPONENTS.yaml` `web_surface.notes` is updated to
  describe the SSH static path and to record that the Cloudflare
  Pages path was superseded for this surface.
- The deployment-wiring slice that follows must:
  - remain static-only and operator-gated;
  - use the D-0049 values: domain `pilot.goalrail.dev`, public path
    `/`;
  - confirm web server type / reverse proxy and the deploy root /
    release directory on the chosen SSH server;
  - define a manual upload method (rsync / scp), explicitly avoiding
    any credential, key, token, hostname, IP address, or SSH config
    in the repository;
  - define a rollback method (previous release directory or symlink
    rollback);
  - run the production build locally, run a `vite preview` smoke
    check, and (only after manual upload) a server smoke check;
  - verify the canonical URL is `https://pilot.goalrail.dev/`;
  - verify HTTPS is active on `https://pilot.goalrail.dev/` before
    public use;
  - re-confirm D-0047 boundaries in the deployed/preview context;
  - record the result in
    `docs/ops/PILOT_INTAKE_RU_DEPLOYMENT_WIRING.md`.
- The surface remains **not deployed** and **not live** until the
  future deployment-wiring patch completes. This decision does not by
  itself authorise publication.

## D-0052 — ChatUI / universal natural-language input is deferred as a primary near-term product surface
Date: 2026-04-28
Status: accepted

Decision:
- a primary universal ChatUI / free-form natural-language input
  surface is **deferred** as a near-term GoalRail product surface
- GoalRail is intended to supplement existing developer and business
  tools, not to replace them; users continue working in the tools
  they already use, while GoalRail acts as a bounded control plane
  that normalizes work into inspectable contracts, scopes, jobs,
  artifacts, and later proof
- a primary universal ChatUI would pull the product toward a generic
  AI workspace and increase intent-to-scope risk before trusted
  actor identity, durable scope envelopes, job boundaries, scoped
  workers, and verification / proof boundaries exist
- this decision is **deferral**, not a permanent ban; ChatUI may
  later exist as an operator / debug / review console or as an eval
  playground if and when its boundaries are explicit and supporting
  primitives exist
- near-term allowed product surfaces are:
  - CLI
  - API / control plane
  - MCP / tools for coding agents, once boundaries are stable
  - external adapters such as Jira / Linear later
  - optional web console for status, approvals, review, traces, and
    debugging
- not allowed as an MVP primary surface:
  - universal ChatUI
  - general-purpose free-form agent workspace
  - broad "ask anything and let the agent decide" interface
- this decision does **not** describe any unimplemented runtime,
  worker, gate, proof, or ChatUI as existing
- this decision does **not** introduce new roadmap items beyond the
  deferral itself
- review date: 2026-06-15, after `ActorContext`, durable
  clarifications / `WorkItem`s, `ScopeEnvelope`, and the first
  scoped worker boundary exist

Rationale:
- prevents scope creep into an AI IDE or generic agent platform
- prevents user-workflow displacement by keeping GoalRail an
  intake / contract / verification layer over the user's existing
  tools rather than a replacement workspace
- prevents premature free-form intent routing before trusted actor
  identity, durable scope envelopes, and job boundaries exist
- prevents unsafe broad worker authority arising from a chat surface
  that lets users "ask anything" before scoped workers and gate /
  proof boundaries are defined
- preserves the current direction recorded in D-0001, D-0002,
  D-0012, D-0013, D-0021, D-0024, D-0025, and the intake → goal →
  clarification → contract → approval → work-item chain (D-0027,
  D-0028, D-0029, D-0035, D-0036, D-0037, D-0038, D-0040, D-0043,
  D-0044, D-0045, D-0046)
- keeps the boundary between contract shaping, planning, execution,
  gate, and proof inspectable rather than collapsing them behind a
  single conversational input

## D-0053 — Pilot intake RU target domain changed to pilot.goalrail.ru
Date: 2026-04-28
Status: accepted
Supersedes: D-0049 (target domain and canonical public URL only)

Decision:
- `apps/web/pilot-intake-ru` active target domain changes from
  `pilot.goalrail.dev` to `pilot.goalrail.ru`.
- Public path remains `/`.
- Public status remains `candidate-public`.
- D-0049 is **superseded** for target domain and canonical public
  URL. D-0049's body is preserved as historical record.
- D-0047 remains fully in force.
- D-0048 remains fully in force.
- D-0051 remains the active hosting / deployment path
  (operator-managed SSH static server, manual `rsync` / `scp` upload
  to a timestamped release directory with atomic `current` symlink
  switch, server-managed HTTPS via existing reverse proxy or Let's
  Encrypt, externally-managed DNS, no automatic redeploys, no CI
  deploy workflow); D-0051's target-domain references inside its
  body should be read with D-0053 in mind — the active target is
  now `pilot.goalrail.ru`, while D-0051's hosting / deployment mode
  remains in force unchanged.
- D-0050 remains superseded by D-0051 for hosting provider and
  deployment mode.
- The `.dev` domain (`pilot.goalrail.dev`) is reserved for a later
  global-market rollout and is **not** the current active target.
- This decision **does not deploy** the surface.
- This decision **does not create DNS records**.
- This decision **does not provision TLS**.
- This decision **does not add Nginx, Caddy, Apache, or other
  reverse-proxy config** to the repo.
- This decision **does not approve backend email capture**.
- This decision **does not approve analytics or session tracking**.
- This decision **does not approve server-side forms**.
- This decision **does not approve persistence**.
- This decision **does not approve runtime execution**.
- This decision **does not approve repo-provider or LLM
  integration**.
- This decision **does not introduce new scenarios** beyond
  `manual_review_gate` and `bounded_task`.
- This decision **does not introduce new outcome tones** beyond
  `ready` / `readyWithCaveats` / `blocked`.
- This decision **does not change product behavior**.
- Server hostnames, IP addresses, usernames, SSH keys, tokens, and
  credentials **must not** be committed to the repository.

Rationale:
- The current public/RU launch path is for the Russian-speaking
  segment.
- The RU segment should publish under the `.ru` domain rather than
  the `.dev` domain.
- The `.dev` domain is reserved for a later global-market rollout.
- Updating the target-domain decision before the SSH deployment
  remote half runs avoids publishing under the wrong public
  surface.
- The change is compatible with D-0051's SSH static-hosting path:
  domain choice is orthogonal to hosting provider, so D-0051
  remains in force without modification.
- Recording the change as an explicit supersession (rather than
  silently substituting the domain in D-0049) keeps the public
  audit trail honest and prevents readers from acting on the
  `pilot.goalrail.dev` instructions in D-0049.

Consequences:
- `apps/web/pilot-intake-ru/index.html`
  `<link rel="canonical" href>` is updated from
  `https://pilot.goalrail.dev/` to `https://pilot.goalrail.ru/`.
- The built `dist/index.html` must contain
  `https://pilot.goalrail.ru/` and must not contain
  `https://pilot.goalrail.dev/` outside historical decision bodies.
- Future SSH deployment runs use the env value
  `GR_PILOT_DOMAIN=pilot.goalrail.ru`.
- DNS / TLS verification targets `https://pilot.goalrail.ru/`.
- Server-side and public browser smoke checks target
  `https://pilot.goalrail.ru/`.
- `docs/ops/STATUS.md`, `docs/ops/NEXT.md`,
  `docs/ops/PILOT_INTAKE_RU_DEPLOYMENT_PREP.md`,
  `docs/ops/PILOT_INTAKE_RU_DEPLOYMENT_WIRING.md`, and
  `docs/ops/COMPONENTS.yaml` are updated to refer to
  `pilot.goalrail.ru` as the active target. The
  `pilot.goalrail.dev` references inside D-0049 / D-0050 / D-0051
  bodies are preserved as historical record.
- Any future switch back to `pilot.goalrail.dev`, or to any other
  domain, requires its own separate explicit decision in
  `docs/ops/DECISIONS.md` before it is implemented.
- The surface remains **not deployed** and **not live** until the
  future deployment-wiring patch completes with HTTPS verified
  active on `https://pilot.goalrail.ru/`. This decision does not
  by itself authorise publication.

## D-0054 — Actor identity is server-resolved; payload actor fields are prototype compatibility only
Date: 2026-04-28
Status: accepted

Decision:
- GoalRail must treat actor identity for canonical state transitions
  as **server-resolved**, not as arbitrary client-supplied truth
- the current server prototype still accepts actor-like fields from
  request payloads, including `request_author`, `intent_owner`,
  `submitted_by`, `applied_by`, `updated_by`, `marked_by`, and
  `approved_by`
- in the current prototype, those payload-supplied fields are
  **compatibility / audit labels only**
- they are **not** production authorization and must not be treated
  as proof that a human, worker, or service actor was trusted by the
  server
- this is acceptable only as dev / prototype behavior while GoalRail
  has no production authn / authz layer, no workers, no runner, and
  no public agent runtime
- this behavior **must not** survive into agent / worker runtime as
  the authority model
- future workers must act as **server-resolved service actors**, not
  by placing arbitrary human actor JSON into payload fields
- future human approvals must be resolved from a **trusted request
  context** or an equivalent trusted control-plane identity
  mechanism
- `ActorContext` is the intended bounded primitive for this
  direction, but this decision **does not implement** `ActorContext`
- `ActorContext` introduction is a future bounded slice and must
  not add broad production auth infrastructure without a scoped
  plan
- highest-risk transitions to migrate first, in order:
  1. approve contract draft / create `ApprovedContract`
  2. mark `ContractDraft` `ready_for_approval`
  3. apply clarification answer
  4. update `ContractDraft` proposed fields
  5. intake `request_author` / `intent_owner` handling
- this decision does **not** change current API behavior
- this decision does **not** add middleware, headers, auth provider,
  authz policy, migrations, agents, workers, runner, gate, proof,
  eval harness, or ChatUI
- this decision does **not** expand MVP scope

Rationale:
- prevents forged approval identity from becoming a normalized
  product behavior, which would silently undermine the audit value
  of `contract.approved`, `contract_draft.marked_ready_for_approval`,
  `contract_draft.updated`, and `clarification.answer_applied`
- keeps approval, update, clarification, and future worker
  identities inspectable rather than collapsing them into trusted
  payload strings
- prevents future agents / workers from claiming human authority
  through payload fields, which would make a coding agent capable
  of self-approving its own work
- aligns with GoalRail's server-owned canonical state transitions
  (D-0027, D-0028, D-0029, D-0035, D-0036, D-0037, D-0038, D-0040,
  D-0043, D-0044, D-0045, D-0046)
- aligns with D-0052, which names `ActorContext` as a missing
  primitive that must exist before revisiting ChatUI / universal
  input

## D-0055 — Business-first Founding Pilot landing supersedes technical interactive walkthrough as primary public RU landing
Date: 2026-04-29
Status: accepted

Decision:
- The public RU landing for `apps/web/pilot-intake-ru` should sell the
  safe пилот ИИ-разработки, not GoalRail internals.
- The primary public message is `ИИ-кодинг без хаоса`.
- The primary offer is a safe 2-week пилот ИИ-разработки on one bounded
  product area.
- The landing should explain the business control layer: repository
  readiness, project context, controlled tasks, and verified result.
- The previous 5-step technical interactive walkthrough is demoted to
  internal / technical demo or checkpoint status. It remains available in
  git history and should not be copied into a duplicate app folder unless
  a future explicit bounded slice requests that.
- D-0047 boundaries remain in force: no backend, no LLM/API, no repo
  provider integration, no code execution, no persistence, no analytics or
  session tracking, no chat UI, no file upload, no model selector, and no
  real repository scan claim.
- Illustrative business demo cards may show example readiness values only
  when clearly marked as examples and not as real scan results.
- D-0053 remains in force: active domain and canonical public URL remain
  `https://pilot.goalrail.ru/` with public path `/`.
- D-0051 remains in force: SSH static deployment remains the hosting path.
- This decision does not deploy the surface.
- This decision does not add deployment wiring, SSH scripts, DNS, TLS,
  hosting config, backend, analytics, email backend, persistence, LLM/API,
  repo-provider integration, runtime execution, or autonomous development.

Rationale:
- The current business question is: `Как мне использовать AI в разработке и
  не получить хаос в продукте?`
- Buyers need a calm pilot offer and risk/control framing before they need
  internal GoalRail workflow terminology.
- The previous technical walkthrough demonstrated method, but it made the
  public first screen too technical for the new pilot-first business motion.
- A business-first landing better matches the Founding Pilot hypothesis: teams
  already use AI tools; the open question is how to avoid losing control over
  code quality, architecture, and releases.

Consequences:
- `docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md` becomes the canonical
  business-first pilot landing copy/governance document.
- `apps/web/pilot-intake-ru` is rewritten as a mostly static RU landing rather
  than a 5-step interactive walkthrough.
- `docs/ops/STATUS.md`, `docs/ops/NEXT.md`, and
  `docs/ops/COMPONENTS.yaml` should describe the public RU target as the
  business-first pilot landing and keep deployment status as not live until
  SSH deployment completes.
- Future deployment validation should validate the business-first landing
  before any SSH deployment attempt.

## D-0056 — Minimal RU pilot lead-capture endpoint allowed
Date: 2026-04-29
Status: accepted

Decision:
- `apps/web/pilot-intake-ru` may use a minimal server-side lead-capture
  endpoint for email submissions.
- Scope is limited to `POST /api/pilot-lead` on the operator-managed RU
  server.
- The endpoint may validate an email address, send a notification email to
  `hello@goalrail.dev`, and append a local JSONL lead record.
- Notification subject must start with `Пилот`.
- Sending uses local Postfix configured to relay through
  `skvmrelay.netangels.ru:25`.
- This decision does not approve analytics, tracking, Google Sheets, CRM
  integrations, cookies, sessions, user accounts, LLM/API calls, repo
  integrations, runtime execution, or a broad backend platform.
- Existing D-0047 boundaries remain in force except for this narrow
  lead-capture exception.
- Email remains `hello@goalrail.dev`.
- Active target remains `pilot.goalrail.ru`.
- SSH static hosting path remains active.

Rationale:
- `mailto:` is unreliable for business users and can trigger browser warnings.
- Visitors expect inline confirmation after submitting an email.
- A minimal endpoint improves UX while preserving manual handoff.
- Netangels SMTP relay solves blocked outbound SMTP on VDS START.
- Local JSONL provides a simple backup lead log without a CRM.

Consequences:
- Nginx must route `/api/pilot-lead` to a minimal server endpoint.
- Server must have Postfix configured with relayhost
  `[skvmrelay.netangels.ru]:25`.
- Endpoint must reject invalid email and spam-like submissions.
- No broad backend expansion is allowed.
- The direct `mailto:hello@goalrail.dev` fallback remains available.

## D-0057 — Server-local direct lead recipient override allowed
Date: 2026-04-29
Status: accepted

Decision:
- The RU pilot lead endpoint may use a server-local notification recipient
  override for form submissions.
- The override path is `/srv/goalrail/pilot/backend/lead-recipient.local`.
- The override file is operator-managed server state and must not be committed
  to the repository.
- If the override file exists, the endpoint validates the contained email
  address and sends lead notifications directly to it.
- If the override file is absent, the endpoint falls back to
  `hello@goalrail.dev`.
- Public/manual contact email remains `hello@goalrail.dev`.
- Cloudflare Email Routing remains the manual-email path for direct messages
  sent by visitors to `hello@goalrail.dev`.
- This decision does not approve storing personal recipient addresses in repo
  docs/code/tests.
- This decision does not approve analytics, tracking, CRM, Google Sheets,
  cookies, sessions, user accounts, LLM/API calls, repo integrations, runtime
  execution, or a broad backend platform.
- D-0056 remains the only approved lead-capture endpoint scope.

Rationale:
- Cloudflare Email Routing forwards normal authenticated manual mail to
  `hello@goalrail.dev`, but form-generated notifications from
  `noreply@pilot.goalrail.ru` were classified as `unauthenticatedForward`.
- A direct server-local recipient override makes form notifications reliable
  without running a separate mail server and without exposing a personal
  destination address in the repository.
- Keeping `hello@goalrail.dev` public preserves the manual fallback path while
  separating machine-generated notifications from Cloudflare forwarding.

Consequences:
- Server bootstrap / operations must provision the override file only on the
  operator-managed server when direct form notification is desired.
- The endpoint must validate the override value before using it.
- Docs may mention the override path and redacted status, but must not include
  the actual destination email address.

## D-0058 — Daily RU pilot lead digest allowed
Date: 2026-04-29
Status: accepted

Decision:
- The RU pilot lead-capture path may send a daily server-side digest of leads.
- The digest scope is limited to records in `/srv/goalrail/pilot/leads/leads.jsonl`.
- The digest runs once per day at 07:00 GMT+3, implemented as `04:00 UTC` server cron.
- The digest covers the previous GMT+3 calendar day.
- If the previous day has zero leads, no email is sent.
- If the previous day has one or more leads, one digest email is sent to the
  same server-local recipient selection used by the lead endpoint: direct
  override from `/srv/goalrail/pilot/backend/lead-recipient.local` if present,
  otherwise fallback to `hello@goalrail.dev`.
- New lead records should include both UTC submission time and GMT+3 local
  submission fields so the digest can be audited without inferring local dates.
- The actual direct recipient address remains operator-managed server state and
  must not be committed to the repository.
- This decision does not approve analytics, tracking, CRM, Google Sheets,
  cookies, sessions, user accounts, LLM/API calls, repo integrations, runtime
  execution, a broad backend platform, or a separate mail server.
- D-0056 and D-0057 remain the only approved lead-capture / recipient override
  boundaries.

Rationale:
- Immediate per-lead notification is useful, but the operator also needs a
  clear daily reminder when there were leads that require action.
- Sending nothing on empty days avoids notification noise.
- Reusing the JSONL log and the direct recipient override keeps the mechanism
  simple and avoids adding a database, CRM, or external automation service.

Consequences:
- Server operations must install a PHP CLI digest script and a cron entry.
- The cron entry is server-local operational state and is documented, not a
  repo-side deploy automation system.
- The digest must not mutate the lead log or mark rows as processed.
- Operators should still inspect the JSONL log directly when needed.

## D-0059 — Resend HTTPS transport allowed for RU pilot lead mail
Date: 2026-04-29
Status: accepted

Decision:
- The RU pilot lead-capture path may use Resend as a narrow transactional
  email transport for lead notifications and daily lead digests.
- Scope is limited to the existing `apps/web/pilot-intake-ru` server-side
  email sends from `POST /api/pilot-lead` and
  `/srv/goalrail/pilot/backend/pilot-leads-digest.php`.
- Resend must be called only from the operator-managed server over HTTPS
  (`https://api.resend.com/emails`).
- The sending domain for this path is `skill7.dev`, because the Resend free
  tier already has that single domain configured.
- The sender is `GoalRail Pilot <noreply@skill7.dev>`.
- The API key must live only in server-local state at
  `/srv/goalrail/pilot/backend/resend-api-key.local` and must not be committed
  to the repository, docs, tests, logs, or command transcripts.
- The recipient remains selected by D-0057: server-local direct recipient
  override from `/srv/goalrail/pilot/backend/lead-recipient.local` when present,
  otherwise fallback to `hello@goalrail.dev`.
- The local JSONL lead log remains the only approved lead persistence.
- Postfix / Netangels relay may remain only as a temporary fallback while the
  Resend API key is absent; once the key is installed, Resend is the intended
  primary transport.
- This decision does not approve analytics, tracking, contact lists, marketing
  campaigns, CRM integrations, Google Sheets, cookies, sessions, user accounts,
  LLM/API calls, repo integrations, runtime execution, or a broad backend
  platform.
- This decision does not change the public/manual contact email
  `hello@goalrail.dev` or the active target `pilot.goalrail.ru`.

Rationale:
- Netangels SMTP / port-25 relay accepts messages but direct Gmail delivery is
  unreliable and lacks useful end-to-end diagnostics.
- The operator reports SMTP ports are blocked/limited, making authenticated
  SMTP a poor next step.
- Resend provides a port-443 HTTPS API, domain authentication, and delivery
  diagnostics without running a separate mail server.
- Reusing `skill7.dev` avoids adding another domain to the Resend free tier.

Consequences:
- Server operations must provision the Resend API key only on the
  operator-managed server.
- PHP mail helpers must avoid logging or echoing the API key.
- Lead endpoint and digest behavior must remain otherwise unchanged: validate
  email, suppress duplicates, append local JSONL for first submissions, send
  notification/digest, and keep direct mailto fallback.
- If Resend delivery fails, the app should surface the same generic
  `mail_unavailable` error and the local JSONL log remains the backup source.

## D-0060 — Stabilization tranche precedes new feature slices
Date: 2026-04-30
Status: accepted

Decision:
- Before new feature expansion, Goalrail will stabilize source-of-truth docs,
  public lead-capture reliability, repo checks, PR/mainline governance, and
  pilot lead PII / abuse / retention guardrails.
- This does not expand MVP scope.
- This does not approve analytics, CRM, Google Sheets, LLM/API calls, repo
  integration, runtime execution, broad backend platform, or new public product
  claims.

Rationale:
- Real public/web/governance implementation now exists.
- Repo protection must catch up with implementation reality.
- Public lead capture must not lose leads or create unmanaged PII risk.
- Source-of-truth docs must be aligned before further implementation.

## D-0061 — Pilot lead capture records notification status before duplicate suppression
Date: 2026-04-30
Status: accepted

Decision:
- RU pilot lead capture stores notification status in the local JSONL log.
- Duplicate suppression applies to successfully notified or legacy processed
  rows, not to failed notification attempts.
- Retry after `notification_failed` is allowed.
- In-flight `received` / `pending` attempts are not treated as fresh retries,
  to avoid concurrent duplicate notifications for the same email.
- Browser-facing errors remain generic.
- This does not approve analytics, CRM, Google Sheets, cookies, sessions,
  queue, database, LLM/API, repo integration, runtime execution, or broad
  backend platform.

Rationale:
- Prevent lost leads when mail transport fails.
- Keep JSONL as the narrow backup source.
- Preserve D-0056/D-0058/D-0059 boundaries.

## D-0062 — Pilot lead capture runtime migrates from PHP to Go sidecar
Date: 2026-04-30
Status: accepted

Decision:
- The RU pilot lead-capture runtime will use a narrow Go sidecar under
  `apps/web/pilot-intake-ru/server`.
- PHP-FPM is treated as an accidental transitional implementation detail, not a
  product runtime direction.
- D-0061 reliability semantics are preserved.
- The sidecar remains landing-owned and must not be folded into the core
  `apps/server` product API.
- This does not approve analytics, CRM, Google Sheets, cookies, sessions,
  queue, database, LLM/API, repo integration, runtime execution, gate, proof,
  or broad backend platform.
- CI should later cover Go + web only, not PHP.

Rationale:
- Avoid institutionalizing PHP in repo CI/runtime.
- Keep runtime language aligned with the existing Go direction.
- Preserve the narrow public landing backend exception without expanding
  product scope.

## D-0063 — Repo checks CI covers Go and web surfaces only
Date: 2026-04-30
Status: accepted

Decision:
- Add a repository checks workflow for pull requests that covers current Go and
  web implementation surfaces.
- Go checks cover `apps/server`, `apps/cli`, and
  `apps/web/pilot-intake-ru/server` with `go test ./...`, `go vet ./...`, and
  `go build` for each canonical command.
- Web checks cover `apps/web` with `npm ci`, then workspace `typecheck`,
  `test`, and `build` scripts.
- PHP is not part of CI because D-0062 removed PHP as active repo runtime for
  the RU pilot lead surface.
- This is a repository quality gate only; it does not add deploy automation,
  branch-protection changes, server configuration, secrets, analytics, CRM,
  Google Sheets, cookies, sessions, database, queue, LLM/API, repo integration,
  runtime execution, gate, proof, or broad backend platform.

Rationale:
- Real Go and web implementation surfaces now exist and need repeatable PR
  checks before further stabilization work.
- The previous PHP runtime was removed, so PHP checks would institutionalize a
  no-longer-active runtime.
- CI should catch build/test regressions without changing deployment mode or
  expanding product scope.

## D-0064 — Main branch protection requires current repo checks
Date: 2026-04-30
Status: accepted

Decision:
- `main` is protected through GitHub branch protection.
- Required checks:
  - `docs-check`
  - `pr-intake-gate`
  - `go (apps/cli)`
  - `go (apps/server)`
  - `go (apps/web/pilot-intake-ru/server)`
  - `web workspaces`
- Branch must be up to date before merge.
- Admin enforcement is enabled.
- Force pushes and branch deletion are disabled.
- Required approving reviews are not enabled in this slice to avoid
  solo-maintainer deadlock.
- Signed commits / DCO automation, merge queue, dependency automation, secret
  scanning, and broader security automation remain separate possible slices.
- This does not add product behavior, deployment automation, repo integration,
  runtime execution, gate, proof, analytics, CRM, or broad backend platform.

Rationale:
- Existing runnable surfaces now have deterministic checks.
- Direct mainline changes should not bypass PR checks.
- Branch protection must catch up with implementation reality without blocking
  the repo on unavailable reviewers.

## D-0065 — Pilot lead capture minimizes PII and adds local retention guardrails
Date: 2026-04-30
Status: accepted

Decision:
- New RU pilot lead records no longer store user-agent.
- The Go sidecar supports a local JSONL purge command with safe dry-run
  default.
- Default retention is 90 days with a bounded override from 7 to 365 days.
- Reverse-proxy rate limiting is an operator-managed deployment guardrail, not
  repo-side server config.
- This does not approve analytics, CRM, Google Sheets, cookies, sessions, IP
  logging, fingerprinting, database, queue, LLM/API, repo integration, runtime
  execution, gate, proof, CAPTCHA-by-default, or broad backend platform.

Rationale:
- Reduce unnecessary PII.
- Give the operator a local lifecycle for JSONL lead data.
- Avoid turning the landing endpoint into a tracking or lead-management
  platform.

## D-0066 — Repo-aware WorkItem planning belongs behind controller / runner boundary
Date: 2026-05-02
Status: accepted

Decision:
- The API server owns canonical state, validation, persistence, events, and
  accepted `WorkItem(planned)` records.
- The current direct one-WorkItem planning endpoint remains prototype/simple v0
  behavior only.
- Rich repo-aware WorkItem planning must use a planning request / proposal /
  acceptance boundary coordinated by worker/controller/runner components.
- Workers and runners must not write canonical WorkItems directly to the
  database or own canonical truth.
- This does not add runner code, checkout, execution, `Run`, receipt,
  `GateDecision`, `Proof`, queue, outbox, runtime registry, or endpoint
  implementation.

Rationale:
- Keeps the API server as a Kubernetes-style control-plane state machine.
- Preserves ADR-0008's rule that repo checkout and code inspection stay behind
  runner boundaries.
- Prevents simple v0 WorkItem planning from growing into hidden repo-aware
  computation inside the API server.

## D-0067 — Public Contract identity is stable across lifecycle
Date: 2026-05-02
Status: accepted

Decision:
- Public/control API should expose one stable `Contract` resource and public
  `contract_id` across seed, draft review, ready-for-approval, approval, and
  later planning.
- `ContractSeed`, `ContractDraft`, and `ApprovedContract` remain internal
  lifecycle records; `ApprovedContract` remains the immutable approved
  snapshot.
- Future public routes should use `contracts/{id}` and subresources such as
  `tasks`, `plans`, and `proposals`, not `contract-seeds`,
  `contract-drafts`, `approved-contracts`, `work-items`, or long internal
  planning type names.
- This does not implement a public Contract aggregate, `contracts` table,
  stores, endpoints, migrations, runner/worker/controller behavior, execution,
  `Run`, receipt, `GateDecision`, or `Proof`.

Rationale:
- Keeps public API product-facing rather than mirroring internal Go / DB types.
- Gives callers one stable identity across the contract lifecycle.
- Preserves strict internal lifecycle boundaries while allowing a cleaner
  control-plane API shape.

## D-0068 — Installation is the running control-plane boundary
Date: 2026-05-03
Status: accepted

Decision:
- Goalrail accepts `Installation` as the running control-plane / instance
  boundary above Organization.
- Deployment modes are `self_hosted` and `saas` only.
- Self-hosted starts with one bootstrapped primary Organization and disabled
  organization creation.
- Organization remains the tenant/workspace boundary inside an Installation.
- The existing Organization -> Project -> RepoBinding model remains valid.
- This is documentation only and does not implement installation schema, auth,
  JWT, CLI login, SaaS onboarding, organization creation API, or web UI.

Rationale:
- Keeps self-hosted MVP bootstrap aligned with future SaaS tenancy.
- Prevents self-hosted shortcuts from bypassing `organization_id`.
- Clarifies deployment-mode language before auth and onboarding work begins.

## D-0069 — Self-hosted auth bootstrap and CLI login direction
Date: 2026-05-03
Status: accepted

Decision:
- Goalrail self-hosted MVP uses a bootstrapped first super admin, mapped in the
  data model to `OrganizationMembership(owner)`.
- The MVP has no public registration; super admin / admin users create users
  inside the Organization.
- Created users receive backend-generated temporary passwords and must change
  password on first login.
- Email invite delivery and password reset delivery are deferred.
- Password credentials should live outside `users`.
- Access tokens should be simple short-lived JWTs.
- Refresh tokens should be opaque DB-backed tokens.
- JWTs should not carry broad or stale permission state; role checks happen
  server-side through `OrganizationMembership`.
- The CLI binary remains `goalrail`.
- `goalrail login` uses explicit `server_url` for self-hosted deployments,
  browser-based localhost loopback callback, and stores `server_url` plus the
  selected Organization / Project / RepoBinding profile.
- SaaS onboarding and organization creation are deferred.

Rationale:
- Keeps self-hosted bootstrap narrow while preserving organization-aware auth.
- Separates product/admin language from data roles.
- Gives later auth schema and CLI work a bounded direction without implementing
  endpoints, tokens, password hashing, CLI changes, web UI, SaaS onboarding,
  organization creation, billing, SSO/OIDC, runner, gate, proof, or queue work.

## D-0070 — Console themes are token-only frontend state
Date: 2026-05-03
Status: accepted

Decision:
- Implement console theme switching as named, terminal-inspired visual presets.
- Apply themes through CSS variables only.
- Keep theme selection in frontend component state, with D-0072 governing the
  later narrow local visual preference exception.
- Do not add backend preference persistence, cookies, sessions, user settings
  API, analytics, auth integration, theme-specific layouts, or theme-specific
  components.

Rationale:
- The console shell already uses CSS variables.
- Named visual presets improve console feel without changing product scope.
- Token-only themes prevent layout drift, component duplication, and backend
  preference scope creep.

Review:
- Review after the first implemented theme picker pass.

## D-0071 — Product API runtime requires structured Postgres config
Date: 2026-05-03
Status: accepted

Decision:
- Production server wiring requires Postgres for product and auth API state.
- `GET /livez`, `GET /readyz`, and `GET /version` remain available without DB
  config.
- Product and auth API routes return HTTP 503 with stable error code
  `database_not_configured` when database config is absent.
- The server uses structured database env vars:
  `GOALRAIL_DATABASE_HOST`, `GOALRAIL_DATABASE_PORT`,
  `GOALRAIL_DATABASE_NAME`, `GOALRAIL_DATABASE_USER`,
  `GOALRAIL_DATABASE_PASSWORD`, and `GOALRAIL_DATABASE_SSLMODE`.
- `GOALRAIL_DATABASE_PORT` defaults to `5432`; `GOALRAIL_DATABASE_SSLMODE`
  defaults to `disable` for local/self-hosted development.
- Host, name, user, and password must all be provided for DB-enabled runtime.
- The raw database DSN is not the primary runtime config path.
- DB password stays suitable for secret-only injection and is not printed in
  config errors.

Rationale:
- Product API state should not appear canonical without durable Postgres state.
- Structured env vars let operators manage host/name/user separately from the
  password secret.
- Health/version can remain lightweight process probes while product APIs
  clearly report missing persistence.

## D-0072 — Console theme preference is local-only
Date: 2026-05-03
Status: accepted

Decision:
- Console theme selection may persist in browser localStorage under
  `goalrail.console.theme`.
- Only valid `ThemeId` values may be stored.
- Invalid or missing values fall back to `goalrail-default`.
- This is a local visual preference only.
- It is not account settings, server state, session state, analytics, or auth
  state.
- Login state, users, product surfaces, and product data must not be persisted
  in this slice.
- Structured empty states are allowed for the three console product surfaces,
  but they must not show fake scores, scans, proof, backend data, execution
  results, live statuses, run IDs, or gate decisions.

Rationale:
- Remembering the chosen theme improves console UX.
- Keeping it local avoids introducing user settings API, auth coupling,
  backend persistence, cookies, or sessions.
- Structured empty states make the console less blank without claiming backend
  or runtime behavior that does not exist.

## D-0073 — RU pilot contact uses pilot mailbox and Telegram channel
Date: 2026-05-04
Status: accepted

Decision:
- The RU pilot landing public/manual contact email is `pilot@goalrail.dev`.
- The direct `mailto:` fallback uses `mailto:pilot@goalrail.dev`.
- The landing exposes Telegram channel `@goalrail` as `https://t.me/goalrail`.
- The landing-owned Go sidecar default notification recipient falls back to
  `pilot@goalrail.dev` when the server-local D-0057 direct recipient override
  is absent.
- This decision supersedes the prior pilot-surface `hello@goalrail.dev`
  fallback/contact claims in D-0056, D-0057, D-0058, and D-0059.
- This decision does not change the server-local override path, Resend sender,
  lead-capture endpoint, JSONL lead log, digest scope, retention posture,
  analytics boundary, CRM boundary, LLM/API boundary, repo-integration
  boundary, or runtime-execution boundary.

Rationale:
- The pilot surface now has a dedicated mailbox.
- Telegram is an explicit public contact/publishing point for Goalrail.
- Keeping both contact channels visible gives visitors a direct fallback
  without adding analytics, CRM, chat, accounts, or broader backend scope.

## D-0074 — Console source is one multilingual app
Date: 2026-05-04
Status: accepted

Decision:
- The canonical product console source is now one multilingual React / Vite app
  under `apps/web/console`.
- `apps/web/console-ru` is removed as a repo workspace source after its useful
  bounded auth flow and Russian copy are migrated into `apps/web/console`.
- The canonical package name remains `@goalrail/console-web`.
- The console uses `react-i18next` + `i18next` with statically bundled `en` and
  `ru` resources, `fallbackLng: en`, and no browser language-detector package.
- Locale selection resolves in this order: `?lng=en|ru`,
  `VITE_GOALRAIL_DEFAULT_LOCALE`, `navigator.language`, then `en`.
- Runtime language switching updates i18next state, `document.documentElement.lang`,
  and the URL `lng` query param without storing locale in browser storage.
- The console auth flow calls the existing server auth API:
  `POST /v1/auth/login`, optional `POST /v1/auth/change-password`, `GET /v1/me`,
  and `POST /v1/auth/logout`.
- Console access tokens, refresh tokens, profile data, and auth state remain in
  React memory only; no cookies, `localStorage`, or `sessionStorage` auth
  persistence is introduced.
- `goalrail.console.theme` remains the only accepted browser-storage key in
  this source slice.
- `VITE_GOALRAIL_API_BASE_URL` is the build-time API base URL knob. Empty means
  same-origin `/v1/...`; non-empty values are trimmed and stripped of trailing
  slashes, so `https://api.goalrail.dev/` plus `/v1/me` becomes
  `https://api.goalrail.dev/v1/me`.
- Target domain direction for later deployment slices is:
  product frontend `goalrail.dev`, API `api.goalrail.dev`, and demo sandbox
  `demo.goalrail.dev`.
- `demo.goalrail.dev` remains separate because it is a sandbox / demo surface,
  not the canonical product console source.
- `console.goalrail.ru` may remain a legacy/live static deployment until a
  separate migration slice updates deployment routing and smoke evidence.
- This decision supersedes D-0022 for source layout only. It does not claim
  deployment, backend CORS, live API routing, DNS, proxy, or server-host changes.
- This does not add public registration, signup, SSO, invite/reset email,
  password reset, SaaS onboarding, organization creation API, analytics, repo
  integration, runner, gate, proof, LLM/API calls, chat UI, file upload, model
  selector, admin user backend calls, or product data loops.

Rationale:
- One console source prevents EN/RU shell drift while keeping demo surfaces
  independent.
- The existing RU auth flow already matched ADR-0023 guardrails and is safer
  than the fake EN local login.
- Keeping locale and auth state out of browser storage preserves the current
  trust boundary while allowing a real multilingual console shell.

## D-0075 — Core server CORS uses an exact frontend origin allowlist
Date: 2026-05-05
Status: accepted

Decision:
- `apps/server` may enable source-level browser CORS for the future product
  frontend/API split where the product frontend target is
  `https://goalrail.dev` and the API target is `https://api.goalrail.dev`.
- CORS is configured only through
  `GOALRAIL_HTTP_CORS_ALLOWED_ORIGINS`, a comma-separated allowlist of exact
  `http` / `https` origins such as `https://goalrail.dev`,
  `http://localhost:5173`, or both for local testing.
- Empty or unset `GOALRAIL_HTTP_CORS_ALLOWED_ORIGINS` disables CORS.
- Allowed browser requests echo the exact request `Origin` as
  `Access-Control-Allow-Origin` and set `Vary: Origin`.
- Preflight `OPTIONS` requests from allowed origins return no-content success
  with allowed methods `GET`, `POST`, `PATCH`, and `OPTIONS`, and allowed
  headers `Authorization` and `Content-Type`.
- Wildcard origins are intentionally unsupported and rejected because the MVP
  boundary needs explicit browser API origins, not broad cross-origin access.
- This slice does not set `Access-Control-Allow-Credentials`, add cookies,
  change frontend code, add public registration, add admin/user APIs, or change
  token persistence.
- Deployment, DNS, TLS, reverse proxy routing, hostnames, IPs, ports, SSH
  paths, credentials, secrets, live smoke, and `goalrail.dev` /
  `api.goalrail.dev` provisioning are not part of this decision.
- The pilot lead sidecar under `apps/web/pilot-intake-ru/server` keeps its
  existing same-origin landing boundary and is not covered by this CORS
  allowlist.
- This does not add signup, SSO, invite/reset email, password reset, SaaS
  onboarding, organization creation API, analytics, repo integration, runner,
  gate, proof, LLM/API calls, chat UI, file upload, model selector, admin user
  backend calls, or product data loops.

Rationale:
- The canonical console source already supports `VITE_GOALRAIL_API_BASE_URL`;
  the core API server needs the matching source-level browser access boundary
  before a later deployment/routing slice.
- Exact origins keep the trust surface reviewable and avoid turning CORS into a
  broad platform policy.
- Keeping credentials/cookies out of this slice preserves the current
  bearer-token, in-memory frontend auth posture.

## D-0076 — RepoBinding init starts from server-owned metadata state
Date: 2026-05-05
Status: accepted

Decision:
- Add the first authenticated server-owned RepoBinding init endpoint:
  `POST /v1/projects/{project_id}/repo-bindings/init`.
- `project_id` is authoritative from the path; `organization_id` is derived
  server-side from the Project and is not accepted from the request body.
- The endpoint creates a primary active RepoBinding for the Project when none
  exists, returns the same binding idempotently for the same normalized
  repository identity, and rejects a different active repository with conflict.
- `workflow_base_branch` is required as request semantics. If it is empty, the
  server may use `provider_default_branch`; if both are empty, validation fails.
- The MVP access mode for this slice is `metadata_only`.
- On create, `repo_binding.initialized` is appended transactionally with the
  RepoBinding write where the current Postgres store supports it.
- The endpoint does not add GitHub App, provider API, deploy key provisioning,
  repository clone, audit queue, runner checkout, gate, proof, full CRUD, VCS
  connection, or repository catalog behavior.

Rationale:
- Server state is the canonical project/repository truth, so CLI-local init
  must not become the source of truth before the server owns RepoBinding state.
- A metadata-only RepoBinding boundary lets later CLI and repository marker slices
  bind to a real server object without introducing checkout or provider access.
- Keeping identity comparison on provider plus repository full name makes init
  idempotent across common remote URL forms.

## D-0077 — CLI init server mode is explicit
Date: 2026-05-05
Status: accepted / superseded for normal plain init by D-0080

Decision:
- This decision captured the first explicit low-level server-backed init mode.
  D-0080 later makes plain `goalrail init` the normal server-backed
  repository-context path while preserving `--project <project_id>` as the
  low-level Project-scoped path.
- Server-backed init uses local Git metadata and the stored
  `goalrail login <server_url>` profile to call the D-0076 RepoBinding init
  endpoint.
- Local/demo init remains auth-free and file-free through explicit
  `--local-demo`.
- Server-backed init requires a detected `workflow_base_branch`; it does not
  use the current local branch as a fallback and does not mutate Git state.
- Expired access tokens fail locally before the HTTP request with a login
  renewal message. Token refresh remains deferred.
- This mode does not add `.goalrail/project.yml` writing by itself, keychain
  integration, Organization / Project / RepoBinding profile selection, audit,
  hooks, branch creation, provider API, deploy keys, clone, runner, gate, or
  proof.

Rationale:
- Explicit `--project` prevents local/demo init from silently becoming a server
  registration flow.
- Reusing the existing login profile keeps the slice narrow and avoids adding a
  second credential mechanism.
- Local expiry checks produce deterministic operator feedback without adding a
  refresh framework.

## D-0078 — CLI writes a non-secret project marker after server-backed init
Date: 2026-05-05
Status: accepted

Decision:
- After successful server-backed init, the CLI writes
  `<git_root>/.goalrail/project.yml`.
- Local/demo init never writes the marker.
- The marker is a non-secret committed repository/team marker. It stores:
  `version`, `server_url`, `organization_id`, `project_id`, `repo_binding_id`,
  and repository `provider`, `full_name`, `url`, and `workflow_base_branch`.
- The marker never stores access tokens, refresh tokens, auth/session material,
  contracts, work items, audit status, proof, memory, diffs, or runtime cache.
- Identical existing content reports unchanged. Different existing content is
  not overwritten and fails without adding force or repair behavior.
- The CLI does not modify the root `.gitignore`. D-0088 adds a scoped
  `.goalrail/.gitignore` for Goalrail-owned machine-local state.

Rationale:
- The local checkout needs a durable marker only after the server returns a
  real RepoBinding identity.
- Keeping the file non-secret preserves the server as source of truth and keeps
  repository checkout state reviewable.
- Refusing silent overwrite prevents accidental rebinding of a checkout to a
  different server/project/repository context.

## D-0079 — CLI preflights the local project marker before server-backed init
Date: 2026-05-05
Status: accepted

Decision:
- Before server-backed init performs a D-0076 or D-0080 HTTP request, it reads
  an existing `<git_root>/.goalrail/project.yml` marker if present.
- Obvious local conflicts in `server_url`, repository provider, repository
  full name, repository URL, or `workflow_base_branch` fail locally before the
  server call. The low-level `--project` path also preflights `project_id`
  because it is known before the HTTP request.
- Unparseable marker content fails locally and is not overwritten.
- `organization_id` and `repo_binding_id` are still validated after the server
  response because they come from the server.
- Local/demo init does not treat the marker as canonical behavior and remains
  file-free.
- No `--force`, repair command, audit, provider integration, runner, gate,
  proof, branch, or PR flow is introduced.

Rationale:
- Preflight prevents a server-side init/create/find from succeeding before an
  obvious repository marker conflict is discovered.
- The repository marker remains non-secret, not a competing
  server or inventory source of truth.
- Deferring repair keeps the first marker semantics narrow and reviewable.

## D-0080 — Normal CLI init bootstraps repository context in an existing Organization
Date: 2026-05-05
Status: accepted

Decision:
- Plain `goalrail init` is now the normal server-backed repository-context
  bootstrap path for authenticated users inside a Git worktree.
- The CLI uses the existing `goalrail login <server_url>` profile and local Git
  metadata to call `POST /v1/init/repository-context`.
- The server resolves the authenticated user's active GoalRail Organization
  from server-side membership state. It does not accept `organization_id` from
  the request body, create Organizations, map GitHub/GitLab/Bitbucket owner to
  GoalRail Organization, or add public Organization creation.
- The endpoint creates or reuses one repo-backed active Project per repository
  inside that existing Organization, using deterministic provider plus
  repository-full-name slugs such as `github-acme-frontend`.
- The endpoint first reuses an existing active RepoBinding for the same
  Organization, provider, and repository full name; otherwise it initializes a
  metadata-only RepoBinding for the resolved Project.
- Active RepoBindings are unique per Organization, provider, and repository
  full name. Multiple repositories in the same Organization map to separate
  repo-backed Projects and do not conflict.
- `goalrail init --project <project_id>` remains as the low-level explicit
  Project-scoped path to the D-0076 endpoint.
- `goalrail init --local-demo` is the explicit auth-free local/demo draft mode
  and writes no files.
- Server-backed init still requires a detected `workflow_base_branch`, fails an
  expired access token locally before HTTP, preflights known marker conflicts,
  writes the existing non-secret `.goalrail/project.yml` marker only after
  server success, and does not store auth tokens in the repository.
- This does not add SaaS onboarding, public registration, Organization
  selection UX, Project full CRUD, provider APIs, GitHub App, deploy keys,
  clone, audit queue, runner checkout, work item linking, branch/PR flow,
  verification, gate, proof, token refresh, keychain, `RepositoryRecord`, or
  provider connection object.

Rationale:
- Normal users should not need to paste a raw Project UUID to initialize a
  repository checkout.
- For the self-hosted MVP, the first company/tenant remains an operator action
  through `goalrail-server bootstrap owner`; repository init should only bind
  repository context inside that existing authenticated Organization.
- One repo-backed Project per repository keeps multi-repo companies workable
  without introducing a repository catalog or VCS connection model before the
  MVP needs it.

## D-0081 — CLI can start work from the local repository marker
Date: 2026-05-05
Status: accepted

Decision:
- `goalrail init --base <branch>` is an explicit workflow base branch override.
  It does not create branches, change Git state, call provider APIs, or infer
  the base from the current local branch.
- When local origin default metadata is available, `goalrail init --base`
  keeps it as `provider_default_branch` and sends the explicit base as
  `workflow_base_branch`. If no origin default is available, the server may
  default provider branch metadata from the explicit workflow base.
- The deterministic `.goalrail/project.yml` marker reader is shared by init
  and later CLI commands. The marker remains non-secret and stores only
  server/project/repo binding identity.
- `goalrail work start --title <title> [--body <body>]` is the first
  server-backed command that uses the repository marker as default context. It
  requires the stored login profile, checks expired access tokens locally,
  reads the Git-root marker, calls `/v1/me`, submits `/v1/intakes`, and then
  promotes the intake through `/v1/intakes/{id}/goals`.
- `goalrail work start` creates an IntakeRecord and Goal only. It does not
  create Contracts, WorkItems, audit requests, runs, receipts, gate decisions,
  proof, provider integrations, branches, PRs, hooks, clones, deploy keys, or
  runner jobs.

Rationale:
- Real CLI use needs an escape hatch for repositories without usable
  `origin/HEAD` metadata, but that hatch must not become branch strategy or Git
  mutation.
- Once server-backed init writes a server-owned marker, later commands should
  use it instead of asking users to paste Project and RepoBinding IDs.
- Starting with IntakeRecord plus Goal preserves the existing server lifecycle
  boundary before contract, work-item planning, audit, runner, gate, and proof
  slices.

## D-0082 — Init records a metadata-only repository context snapshot
Date: 2026-05-05
Status: accepted

Decision:
- Normal server-backed `goalrail init` now records a bounded server-side
  repository context snapshot after `POST /v1/init/repository-context`
  succeeds.
- The new snapshot endpoint is
  `POST /v1/repo-bindings/{repo_binding_id}/context-snapshots`.
- The snapshot is linked to the existing Organization, Project, and
  RepoBinding, and is idempotent by RepoBinding plus deterministic fingerprint.
- The CLI snapshot is limited to local metadata inventory: repository identity,
  branch metadata, HEAD SHA, detected marker paths, toolchains, package
  managers, and bounded workspace candidates.
- The local `.goalrail/project.yml` marker remains unchanged and non-secret;
  dynamic project inventory stays server-side.
- The low-level `goalrail init --project <project_id>` path remains the
  metadata-only RepoBinding init path and does not grow into a broader context
  bootstrap flow.
- This does not run tests, evaluate readiness, score the repository, queue
  audit, clone repositories, call provider APIs, connect GitHub App, create
  branches/PRs, assign work, create WorkItems, start runners, write gates, or
  generate proof.

Rationale:
- Repository binding alone tells Goalrail which repository a Project uses, but
  not the initial shape of the local checkout.
- A metadata-only snapshot makes `init` a more complete first step without
  crossing into audit, runner, provider integration, or verification behavior.
- Keeping snapshot history server-owned preserves `.goalrail/project.yml` as a
  committed repository marker rather than a competing inventory store.

## D-0083 — Provider-neutral VCS connection boundary direction superseded
Date: 2026-05-05
Status: superseded by D-0086

Decision:
- Superseded by D-0086.
- The former provider-neutral VCS connection direction is removed from active
  MVP scope.
- RepoBinding remains canonical repository context and not permission to clone.

Rationale:
- Product direction changed toward runner-owned repository credentials and
  API-issued checkout instructions.

## D-0084 — Backend VCS credential-boundary sequencing superseded
Date: 2026-05-05
Status: superseded by D-0086

Decision:
- Superseded by D-0086.
- The backend VCS/provider implementation sequence is no longer active MVP
  guidance.
- Provider UI, OAuth, token storage, provider clients, and live metadata
  listing require fresh research and a new ADR if reconsidered later.

Rationale:
- The active repository-access direction is runner-owned credentials, not
  provider-mediated repository connection.

## D-0085 — Provider credential storage boundary superseded
Date: 2026-05-05
Status: superseded by D-0086

Decision:
- Superseded by D-0086.
- The MVP does not store repository provider credentials in the API server.
- Runner-owned local credentials are the intended MVP checkout access direction.

Rationale:
- The active MVP avoids server-side repository secret storage instead of
  defining provider-token custody for the API server.

## D-0086 — MVP repository access resets to runner-owned credentials
Date: 2026-05-05
Status: accepted

Decision:
- VcsConnection/provider UI integration is removed from active MVP docs.
- MVP repository access uses `RepoBinding` as canonical repo context.
- The API server stores no repository secrets in the MVP.
- Runner-owned local credentials are the intended MVP checkout access
  direction.
- API-issued `CheckoutInstruction` will provide `repo_binding_id`,
  `repository_url`, `ref`, and `path_scope` to the runner.
- Runner startup config provides Goalrail connection and local credential file
  paths only; it must not hard-code git URL, repo binding id, or checkout mode.
- Provider UI integrations require fresh research and a new ADR if
  reconsidered later.

Rationale:
- This keeps repository identity, checkout authority, and credential custody in
  separate boundaries.
- It preserves RepoBinding and repository-context init truth without adding
  provider OAuth, token storage, provider clients, runner checkout, gate, or
  proof behavior.

## D-0087 — Repository baseline is immutable and overlay is separate
Date: 2026-05-06
Status: accepted

Decision:
- Adopt immutable local `RepositoryBaselineProfile` per committed repository
  state.
- Represent dirty, unmerged, partial, ignored/untracked visibility, submodule,
  sparse-checkout, shallow-clone, and worktree-specific state as a separate
  `WorkspaceOverlay`.
- Future `ContractContextPack` records must be task-specific and reference
  exact baseline and overlay versions.
- Do not key the deep baseline directly by dirty state. Dirty state is
  freshness-relevant, but it does not create a new deep baseline version by
  itself.
- Background scans are convenience only and must never bypass synchronous
  freshness checks.
- The server may store summaries and receipts by default, not raw source bodies;
  the API server does not clone repositories or run checks in-process for this
  boundary.
- This decision does not add server-side repository clone, provider OAuth,
  runner checkout, file watcher / daemon, mutable latest baseline, embeddings
  or LLM summaries as truth, raw source upload by default, gate, proof, or broad
  indexing behavior.

Rationale:
- Immutable committed-state baselines keep Project Scan deterministic and
  privacy-preserving.
- A separate overlay avoids stale maps, deep baseline churn on every edit, and
  background-race reliability failures.
- Task-specific context packs keep repository understanding tied to the
  approved contract/task instead of turning it into hidden reusable memory.

## D-0088 — CLI Project Scan v0 starts as local cache and freshness commands
Date: 2026-05-06
Status: accepted

Decision:
- Implement the first local CLI Project Scan v0 foundation under
  `apps/cli/internal/projectscan`.
- Store immutable `RepositoryBaselineProfile` JSON and current
  `WorkspaceOverlay` JSON in the user cache directory keyed by RepoBinding and
  canonical repository root.
- Add `goalrail project scan` to build or refresh the baseline for the current
  committed HEAD and refresh overlay.
- Add `goalrail project status` to refresh overlay and report freshness without
  rebuilding the baseline by default.
- Run a best-effort quick local Project Scan after server-backed `goalrail init`
  writes or verifies `.goalrail/project.yml`.
- Keep `.goalrail/project.yml` as the canonical committed Goalrail project
  marker. It is a team/repository artifact, not machine-local state, and is not
  moved under `.goalrail/local/`.
- Server-backed init also ensures `.goalrail/.gitignore` contains the standard
  Goalrail-owned machine-local ignores: `/local/`, `/cache/`, `/state/`,
  `/tmp/`, `*.local.yml`, `*.local.toml`, and `*.local.json`.
- Keep Project Scan cache artifacts in the OS user cache directory and out of
  tracked repo files. Do not add broad `.goalrail/` rules to the root
  `.gitignore`.
- This does not add server-side baseline persistence, server clone, raw source
  upload, provider OAuth, runner checkout, background daemon, file watcher,
  embeddings, LLM summaries, ContractContextPack generation, gate, proof, or
  broad indexing behavior.

Rationale:
- The CLI already owns local Git context and the non-secret project marker, so
  the first freshness substrate can be deterministic and local without changing
  server boundaries.
- Separating `project scan` from `project status` makes rebuild behavior
  explicit and keeps status checks cheap.
- Running the quick scan after init gives future contract/task commands a local
  baseline substrate while preserving the server no-clone and no-source-upload
  boundary.
