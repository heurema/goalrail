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
- policy may narrow runtime exposure beyond what risk alone suggests
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
