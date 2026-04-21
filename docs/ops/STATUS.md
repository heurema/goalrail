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

Last updated: 2026-04-21
Status: planning / product canon and pilot frame active
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
- two kernel ADRs
- ops rails
- repo-tracked work / knowledge / public surfaces
- planned flow / eval structure
- reference screens
- no implementation baseline yet

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
- kernel schema note and two boundary ADRs exist

### Repo structure
- the repo now mirrors `punk`-style planning boundaries
- `work/` is reserved for goals, reports, and bounded planning artifacts such as demo-planning packs
- `knowledge/` is reserved for advisory research and idea backlog
- `public/` is reserved for public narrative drafts, receipts, and manual metrics
- `flows/` and `evals/` exist as planned future structure, not executable product surfaces
- `apps/`, `scripts/`, and `.github/` remain parked until bounded implementation slices activate them

## What is not real yet

- no schema package
- no runtime registry implementation
- no runtime CLI
- no executable flow specs yet
- no runnable eval harness yet
- no gate/proof implementation
- no advisory panel implementation
- no web UI
- no tracker sync
- no proof-producing demo
- no CTO / Head of Engineering deck yet
- `GOALRAIL_LANDING_COPY.md` still reflects older prompt / handoff framing and needs rewrite under the new pilot-first motion

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
- repo workspace boundaries mirror `punk`-style planning structure
- `GOALRAIL_OFFER.md` exists as the current sellable package source
- next sales-pack slices are explicit and bounded

## Main current risks

1. ops, offer, deck, and landing assets could drift away from the new concept canon
2. schema work could overgrow before the first compiling package exists
3. runtime adapter model could drift into vendor-specific code
4. execution parallelism and advisory parallelism could still leak into one implementation surface
5. MVP scope could widen into a generic agent or tooling platform too early
