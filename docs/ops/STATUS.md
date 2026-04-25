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

Last updated: 2026-04-25
Status: planning / product canon and pilot frame active; first local Go CLI bootstrap exists
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
- three kernel/CLI boundary ADRs
- ops rails
- repo-tracked Goalrail and Punk overlay surfaces
- planned flow / eval structure
- reference screens
- shared web stack rules under `apps/web/`
- empty real console shells under `apps/web/console` and `apps/web/console-ru`
- local change-packet demo prototypes under `apps/web/demo-change-packet` and `apps/web/demo-change-packet-ru`
- a local RU pilot-intake landing prototype under `apps/web/pilot-intake-ru`
- an open-source community baseline (`LICENSE`, `NOTICE`, contributor docs, issue forms, `CODEOWNERS`)

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

### Delivery model
- roadmap phases defined
- checkpoint model defined
- bounded slice workflow defined
- implementation discipline fixed: `punk`
- execution parallelism and advisory parallelism are separated conceptually
- kernel schema note and three boundary ADRs exist

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
- `.github/` now contains real contributor/community health surfaces and the docs-check workflow
- `scripts/` remains parked for future bounded implementation slices

## What is not real yet

- no schema package
- no runtime registry implementation
- no production runtime CLI beyond the local/demo `apps/cli` command foundation
- no server integration for the CLI
- no production repo authorization or deploy-key provisioning in the CLI
- no real RepoBinding state sync
- no executable flow specs yet
- no runnable eval harness yet
- no gate/proof implementation; `proof show` only renders provided local JSON
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
- `apps/web/pilot-intake-ru` provides a verified local RU pilot-intake landing prototype for the pilot-first public entry
- `apps/web/` remains a shared multi-resource namespace instead of a single runnable app surface
- repository community health and OSS baseline are explicit and inspectable
- next sales-pack slices are explicit and bounded

## Main current risks

1. ops, offer, deck, and landing assets could drift away from the new concept canon
2. schema work could overgrow before the first compiling package exists
3. runtime adapter model could drift into vendor-specific code
4. execution parallelism and advisory parallelism could still leak into one implementation surface
5. MVP scope could widen into a generic agent or tooling platform too early
6. reference screenshots or brand assets could be relicensed accidentally without a provenance audit
