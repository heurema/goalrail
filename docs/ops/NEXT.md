# Goalrail Next

> Only the next bounded slices. Keep this file short.

## Active phase

- **Phase 0 -> Phase 1 transition**
- product and deployment canon is now in place
- repo overlay structure now keeps Goalrail artifacts in `.goalrail/` and Punk publishing artifacts in `.punk/`
- `apps/web/` now exists as the shared namespace for frontend resources
- `apps/web/console` now exists as the empty real console shell for `console.goalrail.dev`, and `apps/web/console-ru` is its separate Russian copy for `console.goalrail.ru`; future cards and detail views should wait until the CLI/server functionality exists
- `apps/web/demo-change-packet` and `apps/web/demo-change-packet-ru` are separate EN/RU demo resources with independent domains; future web work should follow `apps/web/<resource>`
- `apps/server` now exists as a Go server bootstrap with health/version endpoints plus Postgres-backed source-neutral intake, Project / RepoBinding context validation for intake, Goal promotion, Goal readiness state, intake/goal EventLog persistence, transactional canonical write + event append hardening, explicit re-check, and in-memory ClarificationRequest, ClarificationAnswer recording, answer application, ContractSeed creation, and ContractDraft creation prototypes; future server work should stay bounded and avoid fake canonical state claims
- ADR-0008 now defines the runner and repository checkout boundary; future repository checkout/check work must happen behind runners, not inside the API server
- ADR-0009 now defines the ClarificationAnswer recording boundary; future answer work must record evidence before Goal hint application or readiness re-check
- ADR-0010 now defines the MVP Organization / Project / RepoBinding and persistence bootstrap boundary; future persistence work should keep direct RepoBinding before RepositoryRecord
- ADR-0011 now defines answer application to Goal hints and the server still keeps clarification request/answer state in-memory; future answer work must keep readiness re-check separate
- ADR-0012 defines explicit readiness re-check after applied answers, and the server verifies that the existing readiness endpoint can move an applied-answer Goal to `ready_for_contract_seed` without creating contract seed
- ADR-0013 now defines the `ContractSeed` boundary, and the server has an in-memory `ContractSeed(created)` prototype; future contract work must keep approval, work item, gate, and proof as later boundaries
- ADR-0014 now defines the `ContractDraft` boundary, and the server has an in-memory `ContractDraft(draft)` prototype; future contract work must keep approval, work item, gate, and proof as later boundaries
- ADR-0015 now defines the `ContractDraft` review/update boundary; future update work must keep state `draft` and keep `ready_for_approval` and approval as later boundaries
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

1. ContractDraft review/update prototype
   - update proposed draft fields through explicit server-owned update transition
   - append `contract_draft.updated` events with `updated_by` and changed fields
   - keep `ContractDraft.state = draft` and preserve `ready_for_approval` as a later boundary
   - do not create approved Contract, work items, execution, gate, or proof
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
