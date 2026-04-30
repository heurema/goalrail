# Goalrail Next

> Only the next bounded slices. Keep this file short.

## Active phase

- **Phase 0 -> Phase 1 transition**
- product and deployment canon is now in place
- repo overlay structure now keeps Goalrail artifacts in `.goalrail/` and Punk publishing artifacts in `.punk/`
- `apps/web/` now exists as the shared namespace for frontend resources
- `apps/web/console` now exists as the empty real console shell for `console.goalrail.dev`, and `apps/web/console-ru` is its separate Russian copy for `console.goalrail.ru`; future cards and detail views should wait until the CLI/server functionality exists
- `apps/web/demo-change-packet` and `apps/web/demo-change-packet-ru` are separate EN/RU demo resources with independent domains; future web work should follow `apps/web/<resource>`
- `apps/web/pilot-intake-ru` now targets a business-first RU pilot landing for `ИИ-кодинг без хаоса`: a mostly static Founding Pilot page for a safe 2-week пилот ИИ-разработки on one bounded product area, with repository readiness, project context, controlled tasks, verified result, a D-0056 minimal `POST /api/pilot-lead` endpoint with duplicate suppression, D-0059 Resend HTTPS notification transport when configured, and direct `mailto:` fallback. D-0055 supersedes the previous technical interactive walkthrough as the primary public RU landing; that walkthrough is demoted to internal / technical demo or checkpoint status in git history. D-0047 boundaries remain in full except for the narrow D-0056 lead-capture endpoint (no analytics, tracking, CRM, Google Sheets, cookies, sessions, LLM/API, repo integration, code execution, broad backend platform, chat UI, file upload, model selector, or real repository scan claim). Active target domain remains `pilot.goalrail.ru` per D-0053; SSH static hosting remains the path per D-0051; server upload, D-0056 lead-capture endpoint wiring, and server-side TLS provisioning are complete, but public DNS still resolves to a different upstream, so the public site is not live until `https://pilot.goalrail.ru/` reaches the operator-managed server and smoke passes.
- `apps/server` now exists as a Go server bootstrap with health/version endpoints plus Postgres-backed source-neutral intake, Project / RepoBinding context validation for intake, Goal promotion, Goal readiness state, ContractSeed creation, ContractDraft creation/update/ready_for_approval, ApprovedContract approval, durable EventLog persistence, transactional canonical write + event append hardening, explicit re-check, and in-memory ClarificationRequest, ClarificationAnswer recording, answer application, and WorkItem planning prototypes; future server work should stay bounded and avoid fake canonical state claims
- ADR-0008 now defines the runner and repository checkout boundary; future repository checkout/check work must happen behind runners, not inside the API server
- ADR-0009 now defines the ClarificationAnswer recording boundary; future answer work must record evidence before Goal hint application or readiness re-check
- ADR-0010 now defines the MVP Organization / Project / RepoBinding and persistence bootstrap boundary; future persistence work should keep direct RepoBinding before RepositoryRecord
- ADR-0011 now defines answer application to Goal hints and the server still keeps clarification request/answer state in-memory; future answer work must keep readiness re-check separate
- ADR-0012 defines explicit readiness re-check after applied answers, and the server verifies that the existing readiness endpoint can move an applied-answer Goal to `ready_for_contract_seed` without creating contract seed
- ADR-0013 now defines the `ContractSeed` boundary, and the server persists `ContractSeed(created)` in Postgres when DB is configured; future contract work must keep approval, work item, gate, and proof as later boundaries
- ADR-0014 now defines the `ContractDraft` boundary, and the server persists `ContractDraft(draft)` creation in Postgres when DB is configured; future contract work must keep approval, work item, gate, and proof as later boundaries
- ADR-0015 now defines the `ContractDraft` review/update boundary, and the server can update proposed draft fields while keeping state `draft`; approval remains a later boundary
- ADR-0016 now defines the `ContractDraft ready_for_approval` boundary, and the server implements it as an explicit `draft -> ready_for_approval` transition with completeness checks and `marked_by` audit identity; approval, approved Contract, work item, gate, and proof remain later boundaries
- ADR-0017 now defines the Contract approval boundary from `ContractDraft(ready_for_approval)` to `ApprovedContract`; the server implements it as explicit ApprovedContract snapshot creation with `approved_by` and `contract.approved`; approval does not start execution, gate, or proof
- ADR-0018 now defines the WorkItem planning boundary from `ApprovedContract(approved)` to `WorkItem(planned)`; the server implements one in-memory planned WorkItem per ApprovedContract in v0, while assignment, claiming, execution, Run, receipt, gate, and proof remain later boundaries
- the next slices should use those overlay boundaries instead of adding ad hoc top-level storage

## Stabilization tranche — source-of-truth and public-surface hardening

This tranche is the immediate priority before new feature slices. It is not a
product expansion, does not change MVP scope, and does not approve new runtime,
analytics, CRM, repo integration, LLM/API, runner, gate, proof, or broad backend
claims.

Ordered follow-up slices:

A. Documentation source-of-truth alignment — docs-only slice; complete when the landing canon, stabilization decision, AGENTS, README, STATUS, and NEXT are aligned.
B. Pilot lead capture reliability patch.
C. Repo checks CI for Go / web / PHP syntax.
D. Branch protection / PR-only governance checklist.
E. PII / abuse / retention guardrails for pilot lead capture.
F. Root onboarding surface map cleanup if not completed in this PR.

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

Status: **SERVER UPLOAD COMPLETE — DNS/TLS PENDING.** The business-first landing rewrite and D-0056 lead-capture patch passed local typecheck / test / build / boundary audit / local preview smoke. The operator-managed SSH server was bootstrapped, Nginx static serving was configured, timestamped releases were uploaded with `rsync`, and `/srv/goalrail/pilot/current` was atomically switched to the latest release. The narrow PHP-FPM `/api/pilot-lead` endpoint was installed, D-0059 Resend HTTPS transport was configured with `skill7.dev` sender and server-local API key, Postfix relay through Netangels remains a temporary fallback, server-local direct recipient override was configured outside the repo, lead JSONL append and duplicate suppression passed, the daily digest cron was installed for 07:00 GMT+3 previous-day summaries, digest dry-run smoke passed, a one-off digest send was accepted/relayed after envelope-sender alignment, a one-off digest send via Resend reported `transport=resend`, and server-local endpoint smoke passed. Server-side TLS provisioning, renew dry-run, and server-local HTTPS smoke succeeded, but public DNS for `pilot.goalrail.ru` still resolves to a different upstream, so public HTTPS smoke does not reach the deployed landing. The public site is not live yet.

Immediate next action:
- correct external DNS for `pilot.goalrail.ru` so the public domain reaches the operator-managed server, then rerun resolver comparison, public HTTPS smoke, `/api/pilot-lead` smoke, and deployed-surface boundary checks. Update ops docs to `LIVE VIA SSH STATIC SERVER — SMOKE PASSED` only after HTTPS and endpoint smoke pass.

Pre-requisites (base gating satisfied; static upload complete, DNS/TLS pending):
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
- a server-local static smoke check passed against the manually-uploaded release; server-side TLS provisioning and server-local HTTPS smoke succeeded, but public `https://pilot.goalrail.ru/` smoke remains pending until DNS reaches the operator-managed server
- the canonical URL in the deployed `index.html` is verified to be `https://pilot.goalrail.ru/`
- canonical copy parity is re-confirmed against `docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md`; any drift is reconciled before going live
- the lead form posts only to same-origin `/api/pilot-lead`, the fallback still resolves to `mailto:hello@goalrail.dev`, and the primary CTA still only focuses the email input with the temporary local highlight
- DNS remains the active blocker: correct external DNS per D-0051 so `pilot.goalrail.ru` (per D-0053) reaches the operator-managed SSH server (or approved upstream reverse proxy) via A / AAAA / CNAME as appropriate; public resolver checks currently show the domain reaching a different upstream. If the DNS zone is in Cloudflare, the record is DNS-only / non-proxied so public traffic does not depend on Cloudflare Pages, Cloudflare proxy, Cloudflare Workers, or Cloudflare CDN
- server-side HTTPS provisioning is installed on the operator-managed server, but public HTTPS at `https://pilot.goalrail.ru/` is not verified live until DNS reaches that server and the public smoke passes
- a real-device pass is performed on iOS Safari and Android Chrome (against the staging path if available, else against the live URL only after HTTPS is confirmed); any blockers are filed as separate small patches before going live
- a native-speaker proofread of canonical Russian copy is performed; any wording changes land in lock-step in `App.tsx` and `docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md`
- D-0047 boundary is re-confirmed in the deployed context with the D-0056 exception: no LLM/API, no repo provider integration, no code execution, no persistence beyond local JSONL lead log, no analytics or session tracking, no cookies, no sessions, no CRM, no Google Sheets, no broad backend platform, no chat UI, no file upload, no model selector; no non-static network requests are observed during page load, and submit only calls same-origin `/api/pilot-lead`
- the result is recorded in `docs/ops/PILOT_INTAKE_RU_DEPLOYMENT_WIRING.md` as `SERVER UPLOAD COMPLETE — DNS/TLS PENDING`; it may move to `LIVE VIA SSH STATIC SERVER — SMOKE PASSED` only once public DNS reaches the operator-managed server and HTTPS smoke passes
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

### Architecture follow-up slices

1. Organization / project / repo binding persistence boundary
   - ADR-0010 documents Goalrail `Organization`, `User`, `OrganizationMembership`, `Project`, `RepoBinding`, future `VcsConnection`, and `RepoBinding.access_mode`
   - direct `RepoBinding` stores repository reference in the MVP
   - `RepositoryRecord` and `RepositoryEnrollment` are deferred
   - manual/dev-seeded RepoBinding comes before GitHub integration
   - support the customer-hosted runner path without requiring GitHub App, GitLab, or Bitbucket cloud connection
2. Runner checkout prototype boundary
   - start with `goalrail_hosted_runner` only as a Goalrail-operated hosted runner pool
   - use pull-based / poll-based job leasing from the API server
   - perform read-only ephemeral checkout and produce a checkout receipt with minimum evidence fields
   - do not implement customer-hosted runner installer/registration/auth, persistent mirrors, repository writes, arbitrary command execution, gate, or proof
3. Customer-hosted runner protocol boundary
   - define later customer-hosted runner protocol, registration/auth, and customer-owned repository credential flow
   - keep clone access inside customer infrastructure and return bounded artifacts only
   - leave optional attestation or receipt signatures for a later trust-hardening slice

### CLI follow-up slices

1. Server-side repo key provisioning API/client
   - define the smallest server-owned provisioning boundary for repo access
   - keep production private-key generation and storage outside the local CLI
2. Real RepoBinding state sync
   - connect `goalrail init` output to server-backed RepoBinding state
   - keep local draft output until server state exists
3. Contract draft/approval flow integration
   - connect `goalrail contract validate` to real contract draft and approval state
   - preserve field-level validation findings
4. Proof retrieval from server
   - add `proof show` support for fetching stored proof by ID when server proof state exists
   - do not generate final gate verdicts in the CLI

### Server follow-up slices

1. WorkItem assignment/claiming boundary design
   - define the smallest explicit transition after `WorkItem(planned)`
   - keep runner, execution, receipt, gate, and proof as later boundaries
   - do not start execution from assignment/claiming
   - preserve `owner_hint` as advisory unless a later boundary upgrades it
2. CLI-to-server intake submit integration
   - submit intake from the CLI to the server once the API boundary exists
   - keep the CLI as an adapter, not a canonical state owner
3. Durable clarification boundary
   - define the smallest durable persistence slice for ClarificationRequest and ClarificationAnswer after intake/Goal persistence
   - preserve current server-owned answer evidence semantics
   - do not create contract seed, work items, gate, proof, runner, or VCS integration

## Deferred until later

- hosted execution implementation beyond bounded runner prototypes
- tracker integrations
- multi-runtime advisory implementation
- external checks implementation
- analytics / console product features
- Goalrail-specific web product features beyond the current change-packet demo prototypes
- persistent repository mirrors
- repository write operations such as branch creation, commits, or pull requests
