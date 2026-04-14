# Goalrail Status

Last updated: 2026-04-14
Status: planning / phase-1 design baseline active
Owner: Vitaly

## Current state

The project currently has:
- product brief
- MVP blueprint
- build roadmap
- parallel execution model
- implementation guide
- project spine schema note
- two kernel ADRs
- ops rails
- no implementation baseline yet

## What is real now

### Product
- thesis fixed: **от бизнес-цели до проверенного изменения в коде**
- two-plane model fixed:
  - Intent / Planning
  - Delivery / Execution
- Project Spine fixed as the canonical center
- runtime-neutral, CLI-first posture is explicit

### Architecture
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

## What is not real yet

- no schema package
- no runtime registry implementation
- no runtime CLI
- no gate/proof implementation
- no advisory panel implementation
- no web UI
- no tracker sync
- no proof-producing demo

## Active checkpoint target

Current target:
- **C1 — Core objects compile and persist**

Current exit condition:
- core domain types compile
- event envelope exists in code
- serialization and validation tests exist
- canonical vs derived state remains explicit in code

## Main current risks

1. schema work could overgrow before the first compiling package exists
2. runtime adapter model could drift into vendor-specific code
3. execution parallelism and advisory parallelism could still leak into one implementation surface
4. MVP scope could widen into a generic agent or tooling platform too early
5. docs could drift unless ops files are updated per completed slice
