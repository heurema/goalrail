# Goalrail Next

> Only the next bounded slices. Keep this file short.

## Active phase

- **Phase 0 -> Phase 1 transition**
- product and deployment canon is now in place
- repo overlay structure now keeps Goalrail artifacts in `.goalrail/` and Punk publishing artifacts in `.punk/`
- `apps/web/` now exists as the shared namespace for frontend resources
- `apps/web/console` now exists as the empty real console shell for `console.goalrail.dev`, and `apps/web/console-ru` is its separate Russian copy for `console.goalrail.ru`; future cards and detail views should wait until the CLI/server functionality exists
- `apps/web/demo-change-packet` and `apps/web/demo-change-packet-ru` are separate EN/RU demo resources with independent domains; future web work should follow `apps/web/<resource>`
- `apps/web/pilot-intake-ru` now ships the complete RU pilot-first interactive landing demo as a deterministic local 5-step walkthrough (Goal Intake → Clarification → Contract Draft → Review → Honest Outcome), with no backend, no LLM, no repo connection, no execution, no persistence, and no analytics; canonical copy and firm boundaries are in `docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md` and `docs/ops/DECISIONS.md` D-0047. The surface has been approved as the candidate public RU landing demo per `docs/ops/DECISIONS.md` D-0048 (based on `docs/ops/PILOT_INTAKE_RU_INTERNAL_REVIEW_NOTES.md`), deployment-prep is complete (READY WITH WARNINGS) per `docs/ops/PILOT_INTAKE_RU_DEPLOYMENT_PREP.md`, active target domain is now `pilot.goalrail.ru` per `docs/ops/DECISIONS.md` D-0053 (which supersedes D-0049 for target domain and canonical public URL; public path `/` and public status `candidate-public` are preserved; the `.dev` domain is reserved for a later global-market rollout), the pre-publish canonical metadata in `apps/web/pilot-intake-ru/index.html` matches D-0053 (`https://pilot.goalrail.ru/`, also reflected in built `dist/index.html`) per Phase 8K (Phase 8E originally aligned the canonical to D-0049's `.dev` value; Phase 8K realigned it to D-0053's `.ru` value), the static hosting path is operator-managed SSH static server per D-0051 (D-0050 Cloudflare Pages Direct Upload superseded for hosting provider and deployment mode), and SSH static deployment wiring is captured in `docs/ops/PILOT_INTAKE_RU_DEPLOYMENT_WIRING.md` with status `READY FOR SSH DEPLOY — RUNTIME VALUES REQUIRED` after Phase 8H and a same-day Phase 8I re-attempt (Phase 8H ran local preflight, source boundary scan, and local preview smoke — all PASS; Phase 8I re-ran the same local half end-to-end with identical PASS outcomes — but in both phases no remote SSH connection was opened because the operator did not provide the required runtime env vars; no server identifiers, hostnames, IPs, ports, usernames, keys, tokens, or credentials are committed to the repo). Future landing-demo work must respect D-0047 in full.
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

### Slice — Pilot intake RU SSH static deployment wiring
Goal:
- publish `apps/web/pilot-intake-ru` at `https://pilot.goalrail.ru/` on the operator-managed SSH static server per D-0048 + D-0053 (which supersedes D-0049 for target domain and canonical public URL) + D-0051 (which supersedes D-0050 for hosting provider and deployment mode), without expanding product scope and without weakening D-0047

Status: **READY FOR SSH DEPLOY — RUNTIME VALUES REQUIRED.** Phase 8H executed the local half (typecheck/test/build, source boundary scan, local preview smoke against `npm run pilot-intake-ru:preview`) — all PASS. Phase 8I re-attempted the same slice and re-ran the local half end-to-end with identical PASS outcomes (typecheck 0 errors, tests 67/67 in ~18.91s, build ~198ms, dist canonical line confirmed at the time as `https://pilot.goalrail.dev/` per the then-active D-0049, source boundary scan PASS, local preview smoke at `http://localhost:4173/` PASS — full RG walkthrough → outcome `ready`, primary outcome CTA focused email, 0 console errors, 0 console warnings, 0 non-static network requests). Phase 8J was stopped at the env gate immediately. Phase 8K then realigned target-domain metadata to D-0053 (`pilot.goalrail.ru`) ahead of any remote attempt — the active canonical in `apps/web/pilot-intake-ru/index.html` and built `dist/index.html` is now `https://pilot.goalrail.ru/`, and any future SSH deployment attempt must use `GR_PILOT_DOMAIN=pilot.goalrail.ru`. In Phase 8H, 8I, and 8J the remote half was **not** performed because the operator did not provide the required runtime env vars in the shell (`GR_PILOT_REMOTE_DEPLOY=yes`, `GR_PILOT_SSH_TARGET`, `GR_PILOT_RELEASE_ROOT`, `GR_PILOT_CURRENT_LINK`, `GR_PILOT_DOMAIN`); no SSH connection was opened, no `rsync`/`scp` was run, and no remote state was changed. Repo did not acquire any server identifiers.

Immediate next operator action:
- export the five required env vars in the shell (server hostname/user/path values stay out of repo) and re-run the SSH static deployment wiring slice; that run will execute the remote half (rsync/scp upload to the timestamped release directory, atomic `current` symlink switch, server-side and remote-domain smoke checks). Optional `GR_PILOT_SSH_OPTS`, `GR_PILOT_RSYNC_OPTS`, `GR_PILOT_RELEASE_ID`, `GR_PILOT_KEEP_RELEASES`, `GR_PILOT_PREVIOUS_RELEASE` may also be exported.

Pre-requisites (gating, all satisfied):
- the target-domain decision is recorded (D-0053: `pilot.goalrail.ru` / `/` / `candidate-public`, supersedes D-0049 for target domain and canonical public URL; D-0051 hosting/deployment mode preserved)
- the canonical-link metadata fix has landed (Phase 8E — DONE)
- the hosting-path decision is recorded (D-0051: operator-managed SSH static server; D-0050 is `superseded by D-0051 for hosting provider and deployment mode`)
- the local preflight, source boundary scan, and local preview smoke have been executed and all PASS (Phase 8H)

Done means:
- the operator-managed SSH static server is identified out-of-repo (host, IP, SSH port, SSH user, credentials, and any reverse-proxy details remain out of repo); the static web root and release directory are confirmed and recorded in `docs/ops/PILOT_INTAKE_RU_DEPLOYMENT_WIRING.md` without leaking server identifiers
- the web server / reverse-proxy posture is confirmed (e.g. existing nginx/Caddy/Apache/etc.) without committing reverse-proxy config to this repo unless a separate explicit decision authorises it
- production build is run locally (`npm run pilot-intake-ru:typecheck`, `npm run pilot-intake-ru:test`, `npm run pilot-intake-ru:build`) and `apps/web/pilot-intake-ru/dist/` is the upload payload
- the manual upload method is exercised: rsync / scp from the local machine to a timestamped release directory on the SSH server (e.g. `<release-root>/<YYYYMMDDHHMMSS>/`), followed by an atomic `current` symlink switch; at least one previous release is retained for rollback
- no SSH keys, tokens, hostnames, IP addresses, usernames, ports, paths, deploy scripts, server config, or other server identifiers are committed to the repository
- the rollback method is documented: switching the `current` symlink back to a previously-known-good release directory, with no repo-side state changes
- because `PUBLIC_PATH` is `/`, no `vite.config.ts` `base` adjustment is required; root-path behavior and static asset paths are explicitly verified on the deployed surface
- no env vars, no secrets, and no runtime configuration are introduced anywhere — neither in repo nor in the deployed assets
- a local `vite preview` smoke check (`npm run pilot-intake-ru:preview`) is run before upload; visually verifies all 5 walkthrough steps and the 3 outcome tones (ready / readyWithCaveats / blocked) plus the bounded_task fallback
- a server-side smoke check is performed against the manually-uploaded release after upload — first against any staging vhost / path the operator provides (optional), then against `https://pilot.goalrail.ru/` only after DNS and HTTPS are verified
- the canonical URL in the deployed `index.html` is verified to be `https://pilot.goalrail.ru/`
- canonical copy parity is re-confirmed against `docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md`; any drift is reconciled before going live
- the email handoff still resolves to `mailto:hello@goalrail.dev` and the outcome primary CTA still only focuses the email input with the temporary local highlight
- DNS is configured externally per D-0051: `pilot.goalrail.ru` (per D-0053) → SSH server (or upstream reverse proxy) via A / AAAA / CNAME as appropriate; if the DNS zone is in Cloudflare, the record is DNS-only / non-proxied so public traffic does not depend on Cloudflare Pages, Cloudflare proxy, Cloudflare Workers, or Cloudflare CDN
- HTTPS is verified active on `https://pilot.goalrail.ru/` (server-managed via existing reverse proxy or Let's Encrypt); the wiring slice does not declare go-live until HTTPS is confirmed
- a real-device pass is performed on iOS Safari and Android Chrome (against the staging path if available, else against the live URL only after HTTPS is confirmed); any blockers are filed as separate small patches before going live
- a native-speaker proofread of canonical Russian copy is performed; any wording changes land in lock-step in `App.tsx` and `docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md`
- D-0047 boundary is re-confirmed in the deployed context: no backend, no LLM/API, no repo provider integration, no code execution, no persistence, no analytics or session tracking, no chat UI, no file upload, no model selector; no non-static network requests are observed during a deployed-context walkthrough
- the result is recorded in `docs/ops/PILOT_INTAKE_RU_DEPLOYMENT_WIRING.md` (status moves from `PROVIDER SELECTED — WIRING PENDING` to a deployed-status marker once HTTPS is verified active and a server-side smoke check passes)
- if the chosen target domain or public path ever changes, that change is recorded as a separate explicit decision in `docs/ops/DECISIONS.md` before it is implemented; if the deployment mode is later changed (e.g. CI deploy, automatic deploys, repo-side server config), that also requires its own separate decision per D-0051

Out of immediate scope (do not introduce without a new decision):
- backend or any form submission
- analytics or session tracking
- LLM / AI API calls
- repo provider integration
- real execution / runner / sandbox
- persistence of user input
- Cloudflare Pages, Cloudflare Workers, Cloudflare Functions, Cloudflare KV / R2 / D1 / Durable Objects / Queues, Cloudflare proxy / CDN, Cloudflare Web Analytics
- automatic deploys (CI / Git-triggered / cron / file-watch / etc.)
- storing SSH keys, server credentials, hostnames, IP addresses, ports, usernames, deploy scripts, or reverse-proxy config in the repository
- new demo scenarios beyond `manual_review_gate` and `bounded_task`
- new outcome tones beyond `ready` / `readyWithCaveats` / `blocked`
- canonical copy rewrites beyond findings reconciled with the canonical doc
- email lead capture beyond the current `mailto:` / focus / manual handoff

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
