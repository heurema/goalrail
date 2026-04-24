# TrialOps Demo Sandbox — Demo Build Plan

## Purpose

Break the demo work into stable slices with explicit gates.

Rules:
- do not continue automatically to the next phase
- after each phase: stabilize, verify, summarize, and stop
- if a gate fails, simplify the current phase instead of expanding scope
- Phase 0 stays in the current Goalrail planning repo under `.goalrail/work/demo/trialops`
- Phase 1 and later happen in `heurema/goalrail-demo`

## Global constraints

Always preserve:
- one repo for the executable sandbox
- one backend
- one frontend
- one seed/reset path
- one smoke-test path
- fake data only
- no external services
- no auth / payments / cloud / microservices
- no public self-serve AI execution

Future command contract for `heurema/goalrail-demo`:
- `npm install`
- `npm run dev`
- `npm run reset`
- `npm run smoke`

Rollback / simplification rule:
- if a phase gate is not met, revert to the smallest slice that preserves startup/reset/smoke reliability
- document what was intentionally cut
- stop and wait for approval before opening the next phase

## Phase 0 — Discovery and planning pack

### Goal

Produce the planning pack without adding runtime code to the current Goalrail repo.

### Deliverables

- `DEMO_BLUEPRINT.md`
- `DEMO_BUILD_PLAN.md`
- `SCENARIO_LIBRARY.md`
- `SELF_SERVE_DEMO_NOTES.md`
- minimal docs sync for `.goalrail/work/demo/`

### Checks

- validate wording against current canon:
  - `docs/product/GOALRAIL_PRODUCT_CONCEPT.md`
  - `docs/product/GOALRAIL_PILOT_MODEL.md`
  - `docs/product/GOALRAIL_GTM_MODEL.md`
  - `docs/product/GOALRAIL_OFFER.md`
  - `docs/product/GOALRAIL_DEPLOYMENT_MODEL.md`
- run doc checks if governed docs changed

### Gate

Phase 0 passes when:
- no app/runtime code was added here
- docs are enough for Phase 1 handoff
- the repo split is explicit
- self-serve replay is planned, not implemented

## Phase 1 — Demo repo skeleton

### Goal

Create the minimal monorepo skeleton in `heurema/goalrail-demo`.

### Deliverables

- workspace package setup
- placeholder `README.md`
- `apps/api/`
- `apps/web/demo-change-packet/`
- `packages/shared/`
- `data/seed.json`
- `demo/scenarios/*.yaml`
- minimal `reset` and `smoke` scripts

### Checks

- `npm install`
- `npm run reset`
- `npm run smoke`

### Gate

Phase 1 passes when:
- the repo installs cleanly
- reset and smoke commands exist
- scenario manifests exist
- no unnecessary runtime complexity was introduced

## Phase 2 — Minimal backend

### Goal

Implement a tiny reliable local API with deterministic data reset.

### Deliverables

- health endpoint
- trial request list endpoint
- trial request detail endpoint
- status update endpoint
- audit log endpoint
- reset path from seed data

### Checks

- `npm run dev`
- `npm run reset`
- `npm run smoke`
- manual `/health` verification

### Gate

Phase 2 passes when:
- backend starts locally
- health endpoint is stable
- smoke checks cover the API basics
- reset produces the same baseline each time

## Phase 3 — Minimal frontend

### Goal

Add a believable but simple web UI over the minimal backend.

### Deliverables

- list view
- detail view
- dashboard counters
- status transition control
- audit log visibility

### Checks

- `npm run dev`
- `npm run smoke`
- manual walk-through of list, detail, status change, and dashboard refresh

### Gate

Phase 3 passes when:
- backend and frontend run together
- presenter can show the core baseline flow
- smoke checks still pass

## Phase 4 — Main scenario assets

### Goal

Prepare the main Goalrail proof flow before implementing the workflow change.

### Deliverables

- `demo/proof-packs/workflow-change/business-request.md`
- `demo/proof-packs/workflow-change/contract-draft.md`
- `demo/proof-packs/workflow-change/task-plan.md`
- `demo/proof-packs/workflow-change/proof-template.md`
- `docs/demo/DEMO_SHOW_SCRIPT.md`
- `docs/demo/DEMO_DRY_RUN_CHECKLIST.md`

### Checks

- presenter dry-run of the script
- proof-pack completeness review

### Gate

Phase 4 passes when:
- the presenter can narrate the full flow without unstable live AI dependency
- proof artifacts are stable enough to serve as fallback

## Phase 5 — Implement workflow-change

### Goal

Implement the main business change in the sandbox.

### Deliverables

- `manual_review` status in the model
- owner assignment before approval
- decision reason before approval
- dashboard count update
- audit log captures reviewer and reason
- seeded example for manual review
- smoke path for the happy flow

### Checks

- `npm run reset`
- `npm run smoke`
- manual UI walk-through of the main scenario
- proof pack update to match actual behavior

### Gate

Phase 5 passes when:
- the primary scenario works end to end
- reset remains deterministic
- proof assets match reality

## Phase 6 — Reliability hardening

### Goal

Make the demo safe for repeated live use.

### Deliverables

- startup troubleshooting
- before-demo checklist
- after-demo reset checklist
- faster smoke path
- documented presenter fallback path

### Checks

- clean-from-scratch dry run
- repeated reset + smoke cycle

### Gate

Phase 6 passes when:
- a presenter can start from scratch and recover from common failures
- no known flaky step remains undocumented

## Phase 7 — Future self-serve readiness

### Goal

Finish the replay-ready asset contract without building the public demo.

### Deliverables

- richer scenario manifest structure if needed
- predictable proof-pack layout
- optional replay event file for the primary scenario
- future guided replay architecture note

### Checks

- artifact-path review
- replay-readiness review against scenario and proof contracts

### Gate

Phase 7 passes when:
- future self-serve guided replay is straightforward to build
- no public live AI execution was introduced
- no new security posture was required

## Stop-after-each-phase output contract

At the end of every implementation phase, return:

1. `PHASE COMPLETED`
2. `FILES CHANGED`
3. `WHAT WORKS NOW`
4. `CHECKS RUN`
5. `KNOWN LIMITATIONS`
6. `NEXT PHASE PROPOSAL`
7. `STOP / WAIT FOR APPROVAL`
