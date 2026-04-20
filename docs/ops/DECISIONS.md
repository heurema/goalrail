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

## D-0017 — Goalrail adopts punk-style workspace boundaries
Date: 2026-04-20
Status: accepted

Decision:
- the planning repo uses explicit top-level support planes inspired by `punk`
- `work/` tracks bounded goals and reports
- `knowledge/` tracks advisory research and ideas
- `public/` tracks public narrative drafts, receipts, and manual metrics
- `flows/` and `evals/` are reserved as planned spec boundaries for future runtime and verification work
- `apps/`, `scripts/`, and `.github/` remain parked until a bounded implementation slice activates them
