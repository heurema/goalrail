---
id: goalrail_console_readonly_goal_contract_checkpoint
title: Goalrail Console Read-only Goal / Contract Checkpoint
kind: ops_status
authority: operational
status: current
owner: ops
truth_surfaces:
  - console_goal_contract_readonly_boundary
  - console_goal_contract_flow
  - console_deferred_non_goals
lifecycle: active-core
review_after: 2026-08-09
supersedes: []
superseded_by: null
related_docs:
  - docs/INDEX.md
  - docs/ops/DECISIONS.md
  - docs/ops/STATUS.md
  - docs/ops/NEXT.md
  - docs/ops/COMPONENTS.yaml
  - apps/web/console/README.md
---
# Goalrail Console Read-only Goal / Contract Checkpoint

## Scope

This checkpoint records the completed read-only Console Goal / Contract
implementation tranche.

It is limited to the Console's current Intent & Oversight view of Goal
qualification, Contract discovery, selected Contract detail, current draft
rendering, and metadata-only repository context.

Current product flow remains:

```text
Agent -> Goalrail CLI -> Goalrail Server canonical state -> Console read-only dashboard
```

The checkpoint does not change product canon, create a new architecture
decision, add source behavior, or approve workflow mutation from the browser.
D-0091 remains the decision record for the read-only Console direction.

## What is implemented

- `apps/web/console` is the canonical multilingual EN/RU Console source.
- Authenticated Console entry uses the existing server auth profile flow and
  reads `/v1/me` for current profile and active Organization membership.
- Delivery Readiness consumes the read-only qualification feed and renders
  Qualification / Clarification / Contract / Blocked lanes.
- Delivery Readiness is read-only: it shows stored backend state, open
  clarification question text/context, readiness state, blocked state, linked
  Contract state, and calm local timestamps.
- Delivery Readiness cards use one primary status per card with the D-0091
  display priority.
- Linked Contract cards expose `Open contract` navigation only.
- Contracts consumes authenticated, organization-scoped Contract discovery and
  renders a compact rail/list.
- Contracts supports state filtering and repo-binding filtering.
- Contracts list refresh, manual refresh, and simple frontend polling are
  read-only.
- Selected Contract detail consumes the authenticated public Contract aggregate
  endpoint.
- Selected Contract current draft consumes the read-only current draft endpoint
  only when `current_draft_id` is linked.
- The current draft body renders read-only in selected Contract detail.
- The Contracts repository context panel consumes Organization repository
  context metadata and prefers the selected Contract's `repo_binding_id` match.
- Repository context is metadata-only Project / RepoBinding visibility.
- Downstream task, execution, gate, runner, and proof data remains unavailable
  in this view.
- No UI lifecycle mutations are present in the completed Goal / Contract
  Console flow.

## Read-only boundary

The Console is an Intent & Oversight / visualization surface, not workflow
control.

- CLI / agent owns workflow actions.
- Server owns canonical Goal, clarification, Contract, planning, event, and
  later runtime state.
- Console reads canonical state and navigates between surfaces.
- Console may use simple frontend polling of read-only endpoints.
- Manual Refresh / Retry / Select / Filter / Search / Open contract actions are
  allowed because they do not mutate workflow state.
- True long polling, SSE, WebSocket, daemon / heartbeat infrastructure, and
  `Agent working` remain deferred until there is a real source of truth.

## Current user-visible flow

1. User authenticates into the Console.
2. Console reads `/v1/me` for profile and active Organization context.
3. Delivery Readiness calls `GET /v1/qualification-feed?limit=50`.
4. Delivery Readiness renders read-only qualification cards across the current
   lanes with one primary status and calm timestamps.
5. Open clarification questions are displayed as read-only backend state.
6. If a feed item has linked Contract state, the card shows the linked Contract
   summary and an `Open contract` button.
7. `Open contract` navigates to Contracts and loads the selected Contract
   through `GET /v1/contracts/{id}`.
8. Contracts calls `GET /v1/contracts?limit=50` by default.
9. Contracts can refine discovery with state and repo-binding filters through
   `GET /v1/contracts?repo_binding_id=&state=&limit=`.
10. Contracts keeps manual refresh and scheduled simple polling read-only.
11. Selected Contract detail renders the public aggregate from
    `GET /v1/contracts/{id}`.
12. If the aggregate has `current_draft_id`, selected detail loads
    `GET /v1/contracts/{id}/current-draft` and renders the current draft body
    read-only.
13. Contracts reads repository context through
    `GET /v1/organizations/{organization_id}/repository-context` and shows
    metadata for the selected Contract repo binding when available.
14. The view honestly states that task, execution, gate, runner, and proof data
    are unavailable.

## Backend endpoints consumed

| Endpoint | Console use | Boundary |
| --- | --- | --- |
| `GET /v1/me` | Current user, auth/profile, active Organization source. | Read-only profile source for Console context. |
| `GET /v1/qualification-feed?limit=50` | Delivery Readiness feed. | Read-only Goal / clarification / linked Contract state. |
| `GET /v1/contracts?repo_binding_id=&state=&limit=` | Contracts rail/list discovery, including default `GET /v1/contracts?limit=50`. | Read-only, authenticated, organization-scoped Contract discovery. |
| `GET /v1/contracts/{id}` | Selected Contract aggregate detail and linked Contract navigation. | Read-only, authenticated, organization-scoped Contract aggregate. |
| `GET /v1/contracts/{id}/current-draft` | Selected Contract current draft body. | Read-only current draft detail behind stable public Contract id. |
| `GET /v1/organizations/{organization_id}/repository-context` | Contracts repository context panel and Settings / Repository metadata. | Metadata-only Organization / Project / RepoBinding visibility. |

## Frontend surfaces involved

- Delivery Readiness surface: reads qualification feed, renders lanes, read-only
  questions, status, timestamps, and linked Contract handoff.
- Contracts rail/list: reads Contract discovery, supports state and
  repo-binding filters, manual refresh, and simple polling.
- Selected Contract aggregate detail: reads `GET /v1/contracts/{id}` and shows
  stable public Contract identity, lifecycle state, linked ids, and timestamps.
- Selected Contract current draft detail: reads
  `GET /v1/contracts/{id}/current-draft` when linked and renders the draft body
  read-only.
- Contracts repository context panel: reads Organization repository-context
  metadata and shows only metadata-only Project / RepoBinding context.
- Proof surface: remains a structured empty surface; no fake proof data is
  introduced by this tranche.

## Completed slices

| Task | PR | Result |
| --- | --- | --- |
| `TASK-CONSOLE-READONLY-DASHBOARD-DOCS-001` | PR #188 | Documented D-0091 read-only Console direction. |
| `TASK-CONSOLE-READONLY-DASHBOARD-IMPL-002` | PR #191 | Made Delivery Readiness read-only. |
| `TASK-CONSOLE-READINESS-STATUS-TIME-003` | PR #193 | Polished Delivery Readiness status and time display. |
| `TASK-CONTRACT-DISCOVERY-API-004` | PR #195 | Added authenticated read-only Contract discovery, `GET /v1/contracts`. |
| `TASK-CONSOLE-CONTRACT-LIST-005` | PR #197 | Added the frontend Contracts rail/list. |
| `TASK-CONSOLE-CONTRACT-DETAIL-POLISH-006` | PR #198 | Polished selected Contract aggregate detail. |
| `TASK-CONSOLE-CONTRACT-REFRESH-007` | PR #200 | Added simple frontend refresh for Contracts list/detail. |
| `TASK-CONTRACT-DRAFT-DETAIL-API-008` | PR #201 | Added authenticated read-only current draft detail, `GET /v1/contracts/{id}/current-draft`. |
| `TASK-CONSOLE-CONTRACT-DRAFT-RENDER-009` | PR #202 | Rendered current draft body in selected Contract detail. |
| `TASK-CONTRACT-GET-AUTH-SCOPE-010` | PR #203 | Auth-scoped `GET /v1/contracts/{id}`. |
| `TASK-CONSOLE-CONTRACT-DETAIL-ERRORS-011` | PR #204 | Polished Contract detail access errors. |
| `TASK-CONSOLE-GOAL-CONTRACT-FLOW-SMOKE-012` | PR #205 | Added regression coverage for Delivery Readiness -> Contract -> current draft handoff. |
| `TASK-CONSOLE-CONTRACT-REPO-CONTEXT-013` | PR #206 | Added Contracts repository/project context panel. |
| `TASK-CONSOLE-CONTRACT-REPO-FILTER-014` | PR #207 | Added Contracts repo-binding filter. |

## Intentionally deferred

- Goal continuation from the Console.
- Clarification answer submission from the Console.
- Contract creation, update, submission, approval, or plan creation from the
  Console.
- Plan work controls.
- Activity timeline / agent-run history.
- True long polling, server wait/cursor semantics, SSE, WebSocket, daemon, or
  heartbeat status infrastructure.
- `Agent working` until a real daemon / heartbeat source of truth exists.
- Task, execution, gate, runner, and proof data in the Goal / Contract view.
- Provider authorization, checkout, runner, execution, readiness, or proof
  claims from repository context metadata.

## Do not reintroduce

Future agents must not reintroduce these into the completed read-only Console
Goal / Contract flow:

- `continue_goal` UI control.
- Clarification answer form.
- `draft_contract` / create-contract UI control.
- Contract create / update / submit / approve buttons.
- Plan work controls.
- Copy CLI command button.
- `Managed via CLI` labels.
- `Agent working` without real daemon / heartbeat source of truth.
- Fake Proof, readiness, task, execution, gate, runner, or proof data.
- Provider auth / checkout / runner claims from repository context metadata.

## Follow-up posture

Future Console work should start from this boundary and either:

1. stay read-only and consume existing read endpoints, or
2. first document and implement the real canonical source of truth for any new
   status or workflow state before exposing it in the browser.

Any change that turns the Console into workflow control needs a fresh bounded
slice and the appropriate product / architecture update before implementation.
