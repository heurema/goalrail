# Goalrail Next

> Only the next bounded slices. Keep this file short.

## Active phase

- **Phase 0 -> Phase 1 transition**
- product and deployment canon is now in place
- repo overlay structure now keeps Goalrail artifacts in `.goalrail/` and Punk publishing artifacts in `.punk/`
- `apps/web/` now exists as the shared namespace for frontend resources
- `apps/web/console` now exists as the empty real console shell for `console.goalrail.dev`, and `apps/web/console-ru` is its separate Russian copy for `console.goalrail.ru`; future cards and detail views should wait until the CLI/server functionality exists
- `apps/web/demo-change-packet` and `apps/web/demo-change-packet-ru` are separate EN/RU demo resources with independent domains; future web work should follow `apps/web/<resource>`
- `apps/server` now exists as a Go server bootstrap with health/version endpoints plus in-memory source-neutral intake, Goal promotion, Goal readiness, ClarificationRequest, and ClarificationAnswer recording prototypes; future server work should stay bounded and avoid fake canonical state claims
- ADR-0008 now defines the runner and repository checkout boundary; future repository checkout/check work must happen behind runners, not inside the API server
- ADR-0009 now defines the ClarificationAnswer recording boundary; future answer work must record evidence before Goal hint application or readiness re-check
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

### Architecture follow-up slices

1. Organization / user / VCS connection boundary
   - define Goalrail `Organization`, `User`, `Membership`, `VcsConnection`, `RepositoryRecord`, `RepositoryRecord.source_kind`, `RepoBinding`, and `RepoBinding.access_mode`
   - make GitHub the first implementation target without making GitHub App concepts part of the core domain model
   - keep GitLab, Bitbucket, self-managed Git, and custom Git paths representable as later VCS adapters
   - support the customer-hosted runner path without requiring GitHub App, GitLab, or Bitbucket cloud connection
   - avoid introducing `RepositoryEnrollment` as a mandatory v0 object unless policy needs it
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

1. Answer application to Goal hints boundary design
   - define the explicit server-owned transition from recorded ClarificationAnswer evidence to Goal intent-plane hints
   - keep answer application separate from answer recording and from automatic readiness re-check
   - do not create contract seed / contract draft / work items / gate / proof
2. CLI-to-server intake submit integration
   - submit intake from the CLI to the server once the API boundary exists
   - keep the CLI as an adapter, not a canonical state owner
3. Durable state research/decision
   - decide the smallest Postgres event table path before durable persistence
   - do not add Postgres until the decision and slice are explicit

## Deferred until later

- hosted execution implementation beyond bounded runner prototypes
- tracker integrations
- multi-runtime advisory implementation
- external checks implementation
- analytics / console product features
- Goalrail-specific web product features beyond the current change-packet demo prototypes
- persistent repository mirrors
- repository write operations such as branch creation, commits, or pull requests
