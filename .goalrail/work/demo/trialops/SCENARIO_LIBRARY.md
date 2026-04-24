# TrialOps Demo Sandbox — Scenario Library

## Purpose

Define the demo scenario set before any sandbox implementation starts.

Rules:
- only `workflow-change` is a future implementation target for the first live demo
- all other scenarios remain future scenario cards until explicitly approved
- every scenario must stay business-readable and proof-friendly

## Scenario summary

| ID | Status | Purpose |
|---|---|---|
| `workflow-change` | primary / implement later | show a cross-surface workflow change from request to proof |
| `field-change` | future card only | show a bounded schema/UI/data evolution |
| `bugfix` | future card only | show issue discovery, correction, and verification |
| `policy-review` | future card only | show policy/profile discussion and proof posture without overselling automation |

## `workflow-change`

### Business request

Before a trial request can be approved, we need a manual review step. The reviewer must assign an owner and provide a decision reason. The dashboard should reflect the new status, and the audit log should show who made the decision.

### Why it matters

- easy for business stakeholders to understand
- demonstrates a real workflow change instead of a superficial field edit
- touches intent, contract, implementation, verification, and visible proof
- supports the Goalrail message: one case, one flow, one proof contour

### Expected touched areas

- request status model
- backend validation
- request detail workflow
- dashboard counters
- audit event recording
- seed data
- demo proof pack

### Proof expectations

- business request artifact exists
- clarification and contract artifacts exist
- bounded task plan exists
- smoke path covers the happy flow
- UI shows `manual_review`
- audit log shows actor, transition, and decision reason

### Demo risk

Medium.
This is the right first case, but it spans multiple surfaces and can fail if status flow, validation, and audit behavior drift apart.

## `field-change`

### Business request

Add `customer segment` to trial request creation, detail, and list views so the team can quickly separate SMB, mid-market, and enterprise requests.

### Why it matters

- simple business-facing schema change
- shows backend + frontend + seed + validation without workflow complexity
- useful as a lower-risk backup scenario

### Expected touched areas

- request shape
- list and detail UI
- validation rules
- seed data
- docs / proof artifact updates

### Proof expectations

- field appears in list and detail
- creation/update validation is explicit
- seed data includes at least one example per segment
- smoke or manual check confirms the field round-trip

### Demo risk

Low.
Easy to explain and demo, but less powerful than `workflow-change` because it can look like “just another field”.

## `bugfix`

### Business request

Dashboard currently counts rejected requests as active. Fix the counting logic so active totals only reflect open, actionable requests.

### Why it matters

- shows that Goalrail can handle discovery and correction, not just feature work
- creates a clean proof story: reproduce, fix, verify
- useful for a short technical audience demo

### Expected touched areas

- dashboard aggregation logic
- backend query or projection layer
- smoke or regression checks
- proof readout explaining before/after

### Proof expectations

- bug is reproducible from seeded state
- corrected count is visible after the fix
- verification explicitly distinguishes previous wrong behavior from expected behavior
- proof readout explains the regression check

### Demo risk

Medium-low.
Good for technical audiences, but weaker for business buyers than the main workflow-change scenario.

## `policy-review`

### Business request

High-value enterprise trial requests require stricter review before approval, including deeper review depth and a stronger proof expectation.

### Why it matters

- demonstrates that Goalrail can frame policy/profile discussions, not only direct UI or API edits
- connects well to managed pilot posture and configurable knobs
- good setup for future guided replay steps

### Expected touched areas

- policy profile notes
- review-depth rules
- proof expectations
- scenario artifacts and presenter narration

### Proof expectations

- policy discussion is explicit
- contract captures stricter review expectations
- proof artifacts show why the stronger review path exists
- initial version can remain docs/proof only before runtime implementation

### Demo risk

Medium-high.
Easy to over-explain or oversell if implemented too early. Best kept as a future card until the primary scenario is stable.
