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
