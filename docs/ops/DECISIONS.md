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

## D-0014 — Early deployment is managed-first
Date: 2026-04-15
Status: accepted

Decision:
- early Goalrail deployments default to managed deployment
- guided deployment comes later after the playbook stabilizes
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

## D-0017 — Goalrail adopts overlay workspace boundaries
Date: 2026-04-20
Status: accepted

Decision:
- the planning repo uses explicit overlay support planes instead of broad root-level artifact directories
- `.goalrail/work/` tracks bounded goals, reports, and Goalrail delivery memory
- `.goalrail/knowledge/` tracks Goalrail advisory research and ideas
- `.punk/publishing/` tracks public narrative drafts, receipts, and manual metrics owned by the Punk publishing layer
- `.punk/publishing/` uses an explicit publication-plane name to avoid confusion with conventional frontend/static-assets `public/` directories
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
Status: accepted

Decision:
- the real console shell lives in `apps/web/console`
- the target subdomain is `console.goalrail.dev`
- the first console version is intentionally empty except for three left-nav surfaces: Contracts, Delivery Readiness, and Proof
- console visualization follows real CLI/server functionality; UI cards must not imply backend, CLI, server, auth, data, or product-loop implementation before those layers exist

## D-0022 — RU console is a separate web resource
Date: 2026-04-25
Status: accepted

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
- Goalrail supports both `goalrail_hosted_runner` and `customer_hosted_runner` deployment modes
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
- `VcsConnection` remains a future provider layer
- manual `RepoBinding` is enough before GitHub integration

Rationale:
- reduces entity count for the first server MVP
- avoids building a repository catalog before the product contour needs it
- keeps GitHub/GitLab/Bitbucket integration optional later

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
- gives the next implementation slice a bounded target using the existing readiness endpoint

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
- v0 event append remains synchronous, with shared transaction wrappers deferred
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
