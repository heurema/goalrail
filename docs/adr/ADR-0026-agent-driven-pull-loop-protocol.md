# ADR-0026 - Agent-driven pull-loop protocol through Goalrail CLI

Status: accepted
Date: 2026-05-06

## Context

Goalrail must support users working through local coding agents such as Codex,
Cursor, Claude Code, Gemini CLI, and future local runtimes.

These agents can read repo-local instructions and call local tools, but there
is no provider-neutral server push channel that reliably injects a Goalrail
question back into an arbitrary live agent session.

Goalrail also keeps repository access local by default. The API server remains
the canonical state machine, but it must not clone repositories, store
repository secrets, require raw source uploads, or run repository checks
in-process for the MVP.

Current implemented state:

- `goalrail init` binds local Git metadata to server-side Project and
  RepoBinding context and writes `.goalrail/project.yml`.
- `goalrail agent install` installs provider-neutral Agent Pack guidance.
- `goalrail work start` creates an IntakeRecord and promotes it to a Goal.
- `goalrail work continue` reconciles Goal readiness and returns the next
  agent-facing action.
- `goalrail work answer` submits structured clarification answers and returns
  the next agent-facing action.
- `goalrail contract draft` creates or returns a server Contract draft handle
  for a ready Goal and returns a local repository receipt.
- `goalrail contract update` submits structured proposed ContractDraft fields
  after the local agent reads local code.
- `goalrail contract submit` submits a reviewed draft for explicit approval.
- `goalrail contract approve --confirm-user-approval` approves a submitted
  Contract only after explicit user confirmation.
- `goalrail work plan` creates or returns a server-owned WorkItemPlan for an
  approved Contract.
- `goalrail work plan status` reads authenticated plan/proposal status for
  user review.
- `goalrail work proposal accept --confirm-user-acceptance` accepts a submitted
  WorkItemPlanProposal only after explicit user confirmation.
- Server-side Goal readiness, ClarificationRequest, ClarificationAnswer,
  Contract lifecycle, approval, and WorkItem planning primitives exist.
- Runner, checkout, execution, gate, proof, and provider-specific agent
  adapters are not implemented.

## Decision

Goalrail will use an agent-driven pull-loop protocol:

```text
User -> local agent -> Goalrail CLI -> Goalrail server
                       <- JSON next_action <-
User <- local agent renders result/question
```

Responsibilities:

- Server owns canonical state, state transitions, clarification requests,
  clarification answers, contracts, approvals, planning records, and event log.
- CLI owns local repository detection, local auth/session use, local repository
  receipts, and transport between local agent and server.
- Agent owns conversational UX only: rendering questions, collecting answers,
  reading local files when needed, and calling Goalrail CLI.
- Agent instructions are guidance only, not authority.
- Goalrail server remains the authority for canonical workflow truth.

The server must not try to push questions into arbitrary live Codex, Cursor,
Claude, Gemini, or other agent sessions. Goalrail integrates through local
agent pull: the agent calls CLI commands and renders returned `next_action`
instructions.

## Protocol rules

### Agent Pack bootstrap

The CLI may install a provider-neutral Goalrail Agent Pack:

```text
.goalrail/agent/GOALRAIL.md
.goalrail/agent/commands.json
```

The Agent Pack is bootstrap guidance for local agents. It is not the main
protocol, not a provider adapter, and not an authority over Goalrail state.

Root `AGENTS.md` may be created only as a tiny shim when no root `AGENTS.md`
already exists. Existing provider or agent instruction files must not be
overwritten by default or by `--force`; the CLI should report that a manual
patch is needed instead.

Provider-specific shims for Claude, Cursor, Gemini, Windsurf, Gravity, or other
tools are out of scope for Slice A.

### `next_action`

Agent-facing JSON responses should include a stable protocol envelope:

```json
{
  "schema_version": "goalrail.cli.v1",
  "goal_id": "...",
  "goal_state": "...",
  "display": {
    "summary": "Human-safe summary for the agent to show."
  },
  "next_action": {
    "kind": "...",
    "blocking": true
  }
}
```

Agents must treat `next_action.available=false` as a planned or unavailable
command and must not call `next_action.command` in that case.

### `goalrail work start`

`work start` accepts pasted tracker/plain-text tasks through stdin:

```bash
goalrail work start --title "<title>" --body-file - --format json
```

In Slice A, it creates IntakeRecord and Goal through the existing server
endpoints and returns an agent-facing JSON envelope with a planned Slice B
continuation action. It does not run readiness reconciliation and does not
implement `work continue`.

Slice B updates `work start` so the returned continuation command is available:

- `work start` returns `next_action.kind=continue_goal` with
  `available=true`.
- `work continue` is the universal resume/reconcile command for current Goal
  readiness.
- Slice C adds `work answer` as the clarification answer bridge after
  `next_action.kind=ask_user`.
- Slice D adds `contract draft` so `next_action.kind=draft_contract` is an
  available local pull-loop step.
- Slice E adds `contract update` so `next_action.kind=update_contract` is an
  available local pull-loop step while the Contract remains in draft state.
- Slice F adds `contract submit` and `contract approve` so reviewable drafts can
  move through explicit approval without starting planning.
- Slice G1 adds `work plan` so approved Contracts can create or return a queued
  WorkItemPlan without leases, proposals, or execution.
- Slice G2 adds a minimal API-only planning worker that submits a deterministic
  proposal through the typed lease API.
- Slice G3 adds `work plan status` and `work proposal accept` so users can
  review a submitted proposal and explicitly accept planned WorkItems before any
  execution boundary.

### `goalrail work continue`

`work continue` resumes a Goal through the local pull-loop:

```bash
goalrail work continue --goal-id "<goal_id>" --format json
```

It must load the local `.goalrail/project.yml` marker, validate the stored CLI
login/session, validate the marker Organization against `/v1/me`, then ask the
server to reconcile Goal readiness.

Server reconciliation must validate the bearer token, load the active
OrganizationMembership server-side, and verify that the Goal belongs to that
Organization before mutating readiness or clarification state. Reconciliation
may materialize missing derived state, but it must not create duplicate open
ClarificationRequests. If the Goal needs clarification, the server returns or
creates exactly one open ClarificationRequest.

Continuation `next_action` mapping:

- `ready_for_contract_seed` returns `next_action.kind=draft_contract` with
  `available=true` and the
  `goalrail contract draft --goal-id <goal_id> --format json` command.
- `needs_clarification` returns `next_action.kind=ask_user` with
  `available=true`, `blocking=true`, `request_id`, and questions.
- rejected or blocked states return `next_action.kind=blocked`.

### `goalrail work answer`

`work answer` submits structured answers for one open ClarificationRequest:

```bash
goalrail work answer \
  --clarification-request-id "<clarification_request_id>" \
  --answers-file - \
  --format json
```

The answer file uses question-bound structured answers:

```json
{
  "answers": [
    {
      "question_id": "...",
      "value": "..."
    }
  ]
}
```

The CLI must load the local `.goalrail/project.yml` marker, validate the stored
CLI login/session, validate the marker Organization against `/v1/me`, then send
the answer payload to the server.

The server must validate the bearer token, load the active
OrganizationMembership server-side, resolve
`ClarificationRequest -> Goal -> Organization`, and verify that the Goal
belongs to that Organization before recording an answer. The server records the
canonical ClarificationAnswer, applies allowed answer mappings to Goal hints,
runs explicit readiness reconciliation, and returns the same agent-facing
`next_action` mapping as `work continue`.

Repeated answer submission for an already answered request returns an explicit
conflict instead of creating an ambiguous duplicate canonical answer.

### `goalrail contract draft`

`contract draft` creates or returns a server Contract draft handle for a ready
Goal:

```bash
goalrail contract draft --goal-id "<goal_id>" --format json
```

The CLI must load the local `.goalrail/project.yml` marker, validate the stored
CLI login/session, validate the marker Organization against `/v1/me`, refresh
local Project Scan evidence, and call the public Contract lifecycle API. The
local repository receipt includes repository binding, HEAD SHA, baseline,
overlay, dirty, partial, raw source upload flag, and freshness evidence where
available. It must not upload raw source bodies.

The server must validate the bearer token, load the active
OrganizationMembership server-side, verify that the Goal belongs to that
Organization, and require `Goal(ready_for_contract_seed)` before mutation. The
CLI sends the local marker `project_id` and `repo_binding_id` as expectations;
the server must reject the request before mutation if either expectation does
not match the Goal. This prevents building a local repository receipt from the
wrong local checkout. The server creates or returns the public Contract
aggregate plus internal ContractSeed / ContractDraft records. Repeated calls
for the same Goal return the existing Contract draft handle instead of creating
duplicate draft state.

`contract draft` returns an agent-facing envelope with `schema_version`,
`display.summary`, `goal_id`, `contract_id`, `contract_state`,
`local_repo_receipt`, and `next_action.kind=update_contract`. The update action
is available only when `contract_state=draft`; existing non-draft Contracts do
not advertise a callable update command.

### `goalrail contract update`

`contract update` submits proposed ContractDraft fields after the local agent
has read the relevant local code:

```bash
goalrail contract update \
  --contract-id "<contract_id>" \
  --fields-file - \
  --format json
```

The fields file is structured JSON. Supported editable fields use the current
ContractDraft boundary: `title`, `intent_summary`, `proposed_scope`,
`proposed_non_goals`, `proposed_constraints`,
`proposed_acceptance_criteria`, `proposed_expected_checks`,
`proposed_proof_expectations`, and `risk_hints`. The CLI also accepts
`proposed_verification` as an agent-facing alias for
`proposed_expected_checks`, plus structured `context_refs` and `unknowns` event
metadata. It must reject malformed JSON and updates with no editable fields
before mutation. Missing fields mean no change. Present empty strings, blank
strings, empty arrays, or blank array items are rejected in Slice E rather than
being interpreted as clear/no-op semantics.

The CLI must load the local `.goalrail/project.yml` marker, require
`project_id` and `repo_binding_id`, validate the stored CLI login/session,
validate the marker Organization against `/v1/me`, and send the local marker
`project_id` and `repo_binding_id` as server-side expectations. It must not
upload raw source bodies.

The server must validate the bearer token, load the active
OrganizationMembership server-side, verify the Contract Organization matches
that membership, derive the audit actor from the authenticated user, require
`Contract(draft)`, and reject supplied project/repo expectations that do not
match the Contract before mutation. It updates only the current internal
ContractDraft proposed fields, appends `contract_draft.updated`, preserves
`ContractDraft.state=draft`, and does not submit, approve, plan, run, gate, or
create proof.

`contract update` returns an agent-facing envelope with `schema_version`,
`display.summary`, `contract_id`, `contract_state`, `changed_fields`, and
`next_action.kind=review_contract`.

### `goalrail contract submit`

`contract submit` marks the current draft Contract ready for explicit user
approval:

```bash
goalrail contract submit --contract-id "<contract_id>" --format json
```

The CLI must load the local `.goalrail/project.yml` marker, require
`project_id` and `repo_binding_id`, validate the stored CLI login/session,
validate the marker Organization against `/v1/me`, and send the local marker
`project_id` and `repo_binding_id` as server-side expectations.

The server must validate the bearer token, load the active
OrganizationMembership server-side, verify the Contract Organization matches
that membership, derive `marked_by` from the authenticated user rather than
trusting payload actor fields, reject supplied project/repo expectations that
do not match the Contract before mutation, require `Contract(draft)`, run
existing ContractDraft completeness checks, transition the current draft to
`ready_for_approval`, append `contract_draft.marked_ready_for_approval`, and
not create an ApprovedContract, WorkItem, Run, GateDecision, or Proof.

`contract submit` returns an agent-facing envelope with `schema_version`,
`display.summary`, `contract_id`, `contract_state`, and
`next_action.kind=approve_contract` with `available=true`. The command must
include `--confirm-user-approval` so a later approval requires an explicit
human signal.

### `goalrail contract approve`

`contract approve` approves a submitted Contract only after explicit user
confirmation:

```bash
goalrail contract approve \
  --contract-id "<contract_id>" \
  --confirm-user-approval \
  --format json
```

The CLI must reject the command before HTTP unless `--confirm-user-approval` is
present. With the flag present, the CLI must load the local
`.goalrail/project.yml` marker, require `project_id` and `repo_binding_id`,
validate the stored CLI login/session, validate the marker Organization
against `/v1/me`, and send the local marker `project_id` and `repo_binding_id`
as server-side expectations.

The server must validate the bearer token, load the active
OrganizationMembership server-side, verify the Contract Organization matches
that membership, derive `approved_by` from the authenticated user rather than
trusting payload actor fields, reject supplied project/repo expectations that
do not match the Contract before mutation, require
`Contract(ready_for_approval)`, create an immutable ApprovedContract snapshot,
move the public Contract state to `approved`, append `contract.approved`, guard
repeated approval with an explicit conflict, and not create a WorkItem, Run,
GateDecision, or Proof.

`contract approve` returns an agent-facing envelope with `schema_version`,
`display.summary`, `contract_id`, `contract_state`, and
`next_action.kind=plan_work` with `available=true` and
`goalrail work plan --contract-id <contract_id> --format json`.

### `goalrail work plan`

`work plan` creates or returns the server-owned planning queue record for an
approved Contract:

```bash
goalrail work plan \
  --contract-id "<contract_id>" \
  --format json
```

The CLI must load the local `.goalrail/project.yml` marker, require
`project_id` and `repo_binding_id`, validate the stored CLI login/session,
validate the marker Organization against `/v1/me`, and send the local marker
`project_id` and `repo_binding_id` as server-side expectations.

The server must validate the bearer token, load the active
OrganizationMembership server-side, verify the Contract Organization matches
that membership, derive `requested_by` from the authenticated user rather than
trusting payload actor fields, reject supplied project/repo expectations that
do not match the Contract before mutation, require an approved public
Contract plus approved snapshot, and create or return exactly one
`WorkItemPlan` for the Contract; newly created plans start `queued`.

`work plan` returns an agent-facing envelope with `schema_version`,
`display.summary`, `contract_id`, `plan_id`, `plan_state`, `repo_binding_id`,
and a state-aware `next_action`: `queued` maps to `planning_worker_required`,
`leased` maps to `planning_in_progress`, `proposal_submitted` maps to an
available `review_plan_proposal` command that calls `goalrail work plan status`,
`accepted` maps to unavailable `planned_workitems_ready`, and unknown states map
to `blocked`. It does not acquire a lease, create a proposal, accept a
proposal, create WorkItems, start a worker, run code, make gate decisions, or
create proof.

### `goalrail work plan status`

`work plan status` reads the current plan and submitted proposal, when present:

```bash
goalrail work plan status --plan-id "<plan_id>" --format json
```

The CLI must load the local `.goalrail/project.yml` marker, require
`project_id` and `repo_binding_id`, validate the stored CLI login/session,
validate the marker Organization against `/v1/me`, and send the local marker
`project_id` and `repo_binding_id` as server-side expectations.

The server route is read-only but authenticated. It must validate the bearer
token, load the active OrganizationMembership server-side, verify the plan
Organization matches that membership, reject supplied project/repo expectations
that do not match the plan, and return plan state plus the current proposal if
one exists.

`work plan status` returns `proposal_id`, `proposal_state`, and
`proposed_tasks` when a proposal is available. State mapping is honest:
`queued` remains `planning_worker_required`, `leased` remains
`planning_in_progress`, `proposal_submitted` returns
`next_action.kind=accept_proposal` with
`goalrail work proposal accept --proposal-id <proposal_id>
--confirm-user-acceptance --format json`, `accepted` maps to unavailable
`planned_workitems_ready`, and unknown states map to `blocked`.

### `goalrail work proposal accept`

`work proposal accept` materializes planned WorkItems from a submitted proposal
only after explicit user acceptance:

```bash
goalrail work proposal accept \
  --proposal-id "<proposal_id>" \
  --confirm-user-acceptance \
  --format json
```

The CLI must reject calls missing `--confirm-user-acceptance` before HTTP. With
the flag present, it performs the same local marker, login, and Organization
preflight as other agent-facing commands, then sends local marker `project_id`
and `repo_binding_id` as server-side expectations.

The server must validate the bearer token, derive `accepted_by` from the
authenticated user rather than trusting payload actor fields, load active
OrganizationMembership server-side, verify the proposal/plan Organization
matches that membership, reject supplied project/repo expectations before
mutation, require a submitted proposal and `Plan(proposal_submitted)`, then
materialize one or more durable `WorkItem(planned)` records. Acceptance must
not assign, claim, run, checkout, submit receipts, make gate decisions, or
create proof.

After acceptance the CLI returns `created_task_ids` and
`next_action.kind=planned_workitems_ready` with `available=false` and
`planned_slice=H`.

### Clarification and contracts

Clarification remains server-owned. The server creates ClarificationRequest and
records ClarificationAnswer as canonical state. CLI and agents only transport
questions and answers.

No standalone `work context prepare` command is introduced in the MVP. Local
code context begins with the bounded `contract draft` repository receipt, then
the agent uses local file reads to prepare structured `contract update` fields.

## Non-goals

This ADR does not implement:

- provider-specific Codex, Claude, Gemini, Cursor, Windsurf, Gravity, or other
  adapters
- server push into agent sessions
- Jira or Linear sync
- local LLM ownership of canonical truth
- server-side repository clone
- raw source upload by default
- standalone `work context prepare`
- repo-aware planning worker, runner-backed planner, or execution worker
  implementation
- lease acquisition CLI
- proposal submission CLI
- WorkItem assignment, claiming, or execution from the agent-facing CLI
- runner checkout
- execution
- gate
- proof
- Problem Details migration
- idempotency or optimistic concurrency hardening
- broad queue platform
- generic agent framework

## Consequences

Goalrail can support heterogeneous local agents without owning their runtime or
pretending a universal server push channel exists.

The server remains canonical and auditable. The CLI becomes the local bridge for
repository context, auth, and transport. The agent remains a UX layer.

Slices A-G3 establish the first usable
start -> continue -> answer -> contract draft handle -> contract update ->
submit -> explicit approve -> plan -> worker proposal -> explicit proposal
acceptance pull-loop without implying assignment, runner, gate, or proof are
available.

## Rejected alternatives

### Server push into local agents

Rejected. There is no provider-neutral channel that reliably returns a question
into an arbitrary live agent session.

### Agent instructions as enforcement

Rejected. Repo-local instructions are prompt/context guidance, not a canonical
state or enforcement layer.

### Provider-specific shims by default

Rejected for Slice A. The provider-neutral pack must stay canonical and small;
provider-specific files can only be added later through bounded explicit work.
