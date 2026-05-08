# ADR-0029 - Run and execution receipt boundary

Status: accepted
Date: 2026-05-08

## Context

The Goalrail pull loop now reaches planned work and checkout preparation:

```text
Goal
  -> Contract
  -> ApprovedContract
  -> WorkItemPlan
  -> WorkItemPlanProposal
  -> WorkItem(planned)
  -> CheckoutJob
  -> CheckoutInstruction
  -> runner checkout lease
  -> CheckoutReceipt
```

ADR-0008 defines the broad runner boundary: the API server owns canonical
state, while repository checkout, workspace preparation, command execution,
receipts, and artifacts belong behind a runner boundary.

ADR-0028 then implemented the first concrete checkout preparation slice.
`CheckoutInstruction` and `CheckoutReceipt` are sidecar preparation records.
They are not `Run`, not proof, and not gate verdicts. A `WorkItem` remains
`planned` after checkout receipt submission.

The next boundary is execution. Execution must not silently turn checkout
metadata into proof, must not run arbitrary shell commands by default, and must
not mix runtime execution with Gate / Proof.

This ADR is documentation-only. It defines the next runtime boundary before any
code slice implements execution jobs, runs, command execution, or execution
receipts.

## Decision

Goalrail will introduce execution through a separate server-owned
`ExecutionJob` before creating a `Run`.

`ExecutionJob` is the leaseable unit. It represents an authorized attempt to
execute one planned WorkItem using one accepted checkout receipt and frozen
task context.

`Run` is the canonical execution attempt. It is created only when a runner with
a valid execution lease explicitly starts execution. It is not created by:

- `WorkItem(planned)` creation
- checkout job creation
- checkout receipt submission
- execution job creation
- execution lease acquisition

This avoids false `Run` records when an execution job is queued or leased but no
runner actually starts the bounded runtime action.

The first implementation after this ADR should still avoid arbitrary command
execution. It should establish the execution job / lease / run-start /
execution-receipt boundary first, with command execution kept behind a later
explicit runtime adapter slice.

## H2 implementation target

First implementation slice:

```text
H2.1 - ExecutionJob preparation from CheckoutReceipt
```

H2.1 adds:

- agent-facing preparation command for creating or returning one execution job
  from a planned WorkItem and CheckoutReceipt
- server-owned `ExecutionJob` records

H2.1 must not add:

- runner-facing execution job lease route
- runner-facing run-start route
- `Run`
- execution receipt
- command execution
- arbitrary shell command execution
- generic runtime adapter platform
- LLM coding-agent integration
- assignment or claiming
- branch creation, commit creation, pull request creation, or merge request
  creation
- GateDecision
- Proof
- provider OAuth or provider clients
- server-side repository clone
- server-side repository credentials
- raw source upload by default

Second implementation slice:

```text
H2.2 - execution lease plus explicit Run start
```

H2.2 adds:

- runner-facing execution job lease route
- runner-facing run-start route that creates `Run` only with valid lease proof
- a runner `execution-start` mode that leases an execution job and starts a Run
  without executing commands

H2.2 must not add:

- execution receipt
- command execution
- arbitrary shell command execution
- generic runtime adapter platform
- LLM coding-agent integration
- assignment or claiming
- branch creation, commit creation, pull request creation, or merge request
  creation
- GateDecision
- Proof
- provider OAuth or provider clients
- server-side repository clone
- server-side repository credentials
- raw source upload by default

Later H2 slices may add:

- runner-facing execution receipt submission route for bounded metadata
- state-aware next actions that keep Gate / Proof unavailable

Later H2 slices must still not add:

- arbitrary shell command execution without a separate bounded adapter decision
- generic runtime adapter platform
- LLM coding-agent integration
- assignment or claiming
- branch creation, commit creation, pull request creation, or merge request
  creation
- GateDecision
- Proof
- provider OAuth or provider clients
- server-side repository clone
- server-side repository credentials
- raw source upload by default

If a later H2 slice cannot record a meaningful execution receipt without running commands,
it may stop at execution job leasing and run-start state, but it must not
pretend execution occurred.

## Object model

### `ExecutionJob`

Purpose:

- server-owned leaseable request to execute one planned WorkItem
- binds execution to one `CheckoutReceipt`
- freezes the minimum context needed by the runner
- remains separate from `Run`

Minimum fields:

- `execution_job_id`
- `organization_id`
- `project_id`
- `repo_binding_id`
- `task_id`
- `contract_id`
- `approved_contract_id`
- `plan_id`
- `proposal_id`
- `checkout_job_id`
- `checkout_receipt_id`
- `state`
- `requested_by`
- `execution_mode`
- `created_at`
- `updated_at`

Initial states:

```text
queued
leased
run_started
receipt_submitted
expired
cancelled
```

Rules:

- one active execution job per WorkItem / CheckoutReceipt pair in v0
- create-or-return is required for retries
- job creation requires `WorkItem.status = planned`
- job creation requires a matching `CheckoutReceipt`
- job creation does not create `Run`
- job creation does not mutate `WorkItem.status`

### `ExecutionLease`

Purpose:

- typed runner reservation for one execution job
- carries a one-time raw lease token only in the lease response

Minimum fields:

- `execution_lease_id`
- `execution_job_id`
- `task_id`
- `runner_id`
- `lease_token_hash`
- `state`
- `expires_at`
- `created_at`
- `updated_at`

Rules:

- raw lease token is returned only once on lease creation
- raw lease token must not be logged or persisted by the runner
- server stores only a hash
- lease acquisition is scoped by Organization / Project / RepoBinding
- expired lease cannot start a Run or submit an execution receipt

### `Run`

Purpose:

- canonical execution attempt record
- created only when execution actually starts

Minimum fields:

- `run_id`
- `execution_job_id`
- `execution_lease_id`
- `task_id`
- `checkout_receipt_id`
- `runner_id`
- `state`
- `started_at`
- `finished_at`
- `created_at`
- `updated_at`

Initial states:

```text
started
receipt_submitted
failed
cancelled
expired
```

Rules:

- `Run` creation requires valid execution lease proof
- `Run` creation is idempotent for one active lease start attempt when possible
- `Run` does not imply success
- `Run` does not imply verification
- `Run` does not imply proof

### `ExecutionReceipt`

Purpose:

- runner-submitted metadata for what happened during a Run
- input for later Gate / Proof boundaries
- not a verdict

Minimum fields:

- `execution_receipt_id`
- `run_id`
- `execution_job_id`
- `task_id`
- `checkout_receipt_id`
- `runner_id`
- `workspace_ref`
- `commit_sha`
- `baseline_id`
- `overlay_id`
- `execution_mode`
- `started_at`
- `finished_at`
- `process_status`
- `exit_code`
- `artifact_refs`
- `changed_paths_summary`
- `raw_source_uploaded`
- `created_at`

Rules:

- `raw_source_uploaded` must be `false` until a later artifact boundary
  explicitly allows otherwise
- `artifact_refs` are references, not proof
- `changed_paths_summary` is runner-reported metadata, not source truth
- execution receipt does not write `GateDecision`
- execution receipt does not create `Proof`
- execution receipt does not decide whether acceptance criteria passed

## Frozen execution input

An execution job should freeze references to:

- `WorkItem(planned)`
- public `Contract`
- `ApprovedContract`
- `WorkItemPlan`
- `WorkItemPlanProposal`
- `CheckoutJob`
- `CheckoutReceipt`
- `RepoBinding`
- repository source ref
- baseline / overlay receipt IDs when available
- path scope
- proof expectation refs

The runner may receive enough metadata to execute the bounded task, but the API
server must not send repository secrets or raw source bodies by default.

## Proposed surfaces

Agent-facing preparation route:

```text
POST /v1/tasks/{id}/execution-jobs
```

Purpose:

- authenticated user / agent requests execution preparation for one planned
  WorkItem
- request includes expected `project_id`, `repo_binding_id`, and
  `checkout_receipt_id`
- server creates or returns one execution job
- server does not start a Run

Implemented H2.1 CLI:

```bash
goalrail work execution prepare \
  --task-id <task_id> \
  --checkout-receipt-id <checkout_receipt_id> \
  --format json
```

Runner-facing routes:

```text
POST /v1/execution-jobs/leases
POST /v1/execution-jobs/{id}/runs
POST /v1/runs/{id}/receipts
```

Purpose:

- runner leases one execution job from its scoped Organization / Project /
  RepoBinding
- runner validates the instruction against local scope and workspace context
- runner starts one `Run` with lease proof
- runner submits one execution receipt with lease / run proof

The route names may be adjusted during implementation, but the boundary must
preserve separate job creation, lease acquisition, Run start, and receipt
submission.

## Runner responsibilities

The runner must:

- run as a separate process from the API server
- talk to the API server only through authenticated API routes
- use runner-owned workspace and credentials
- request execution leases only for configured Organization / Project /
  RepoBinding scope
- validate each leased execution instruction against that scope
- require a matching checkout receipt before execution start
- create `Run` only through the server route with lease proof
- submit bounded execution receipt metadata
- avoid raw source upload by default
- avoid logging lease tokens, repository credentials, or secret material

The runner must not:

- import `apps/server/internal/*`
- write Postgres directly
- mutate WorkItems directly
- assign or claim tasks
- run arbitrary shell commands in H2
- create GateDecision
- create Proof
- decide final acceptance
- create branches, commits, pull requests, or merge requests

## Server responsibilities

The API server must:

- authenticate the user before execution job creation
- load current `OrganizationMembership` before authorization
- verify WorkItem Organization ownership
- verify supplied Project / RepoBinding expectations before mutation
- require `WorkItem.status = planned`
- require a matching `CheckoutReceipt`
- create or return one bounded `ExecutionJob`
- authenticate runner-facing lease, run-start, and receipt routes
- scope runner leases by Organization / Project / RepoBinding
- store only hashed lease tokens
- create `Run` only on explicit runner start with valid lease proof
- accept execution receipts without turning them into verdicts
- append durable events transactionally where DB is configured

The API server must not:

- clone repositories
- run Git commands, tests, builds, linters, or runtime commands in-process
- store repository secrets
- accept raw source bodies by default
- create GateDecision in H2
- create Proof in H2
- treat checkout receipt or execution receipt as final verification

## Execution receipt vs GateDecision vs Proof

Execution receipt answers:

```text
What did the runner report happened during this Run?
```

GateDecision answers:

```text
Does the collected evidence satisfy scope, target, integrity, and policy lanes?
```

Proof answers:

```text
What frozen evidence package explains the accepted or blocked outcome?
```

Execution receipt is evidence input only. It may contain process status,
artifacts, changed-path metadata, and runtime logs or refs when allowed. It must
not say the task is accepted, verified, or proven.

## State transitions

H2 should preserve this contour:

```text
WorkItem(planned)
  -> ExecutionJob(queued)
  -> ExecutionJob(leased)
  -> Run(started)
  -> ExecutionReceipt(submitted)
```

`WorkItem` should remain `planned` until a later ADR defines assignment,
claiming, completion, or delivery status transitions.

Gate / Proof remain later:

```text
ExecutionReceipt(submitted)
  -> GateDecision
  -> Proof
```

## Agent-facing next actions

After H1, checkout preparation output can honestly stop at:

```json
{
  "kind": "runner_checkout_required",
  "available": false
}
```

After H2.1 implementation, an agent can prepare execution explicitly:

```json
{
  "kind": "prepare_execution",
  "available": true,
  "blocking": true,
  "command": "goalrail work execution prepare --task-id <task_id> --checkout-receipt-id <checkout_receipt_id> --format json"
}
```

After run preparation:

```json
{
  "kind": "execution_runner_required",
  "available": false,
  "planned_slice": "H2.3"
}
```

H2.2 keeps this agent-facing action unavailable because the next step belongs to
the runner. Runner start creates `Run(started)` but does not execute commands or
submit an execution receipt.

If a later H2 slice includes execution receipt submission, the next action after
execution receipt should be:

```json
{
  "kind": "gate_review",
  "available": false,
  "planned_slice": "I"
}
```

No text output should claim verification, acceptance, proof, or gate decision.

## H2 tests required

H2.1 should test:

- execution job creation fails without auth before mutation
- execution job creation rejects Organization / Project / RepoBinding mismatch
- execution job creation rejects non-planned WorkItem
- execution job creation rejects missing or mismatched CheckoutReceipt
- repeated execution job creation returns the existing job
- execution job creation does not create Run

Later H2 slices should test:

- execution lease acquisition is scoped by Organization / Project / RepoBinding
- raw execution lease token is returned only once and stored only as hash
- Run start requires valid lease proof
- Run is not created by lease acquisition alone
- Run start creates `Run(started)`
- repeated Run start does not create duplicate Runs
- Run start does not execute commands
- WorkItem remains `planned`

Later execution receipt slices should test:

- execution receipt requires valid Run / lease proof
- execution receipt with `raw_source_uploaded=true` is rejected
- execution receipt does not create GateDecision or Proof
- no assignment, claiming, branch, commit, pull request, or merge request is
  created
- runner does not import server internals, stores, Postgres, or arbitrary
  command execution packages

## Non-goals

This ADR does not implement or authorize:

- code changes
- assignment
- claiming
- arbitrary shell command execution
- provider-specific runtime adapter
- LLM coding-agent integration
- branch / commit / pull request / merge request creation
- server-side repository clone
- server-side repository credential storage
- provider OAuth
- provider clients
- raw source upload by default
- GateDecision
- Proof
- broad idempotency or concurrency framework
- generic queue / outbox / worker registry
- runtime registry implementation
- production runner registration / dedicated runner token protocol

## Consequences

### Positive

- Avoids false `Run` records for queued or merely leased work.
- Keeps checkout preparation, execution, gate, and proof as separate trust
  boundaries.
- Gives H2 a narrow implementation target without arbitrary command execution.
- Keeps server canonical state separate from runner workspace and credentials.
- Preserves future Gate / Proof semantics by treating receipts as evidence
  inputs, not verdicts.

### Tradeoffs

- Adds one more object before `Run`.
- Later H2 slices may still not execute real commands.
- A dedicated runner registration / token protocol remains deferred.
- Actual runtime adapter behavior remains deferred to a later ADR or slice.

## Review triggers

Review this ADR before:

- enabling command execution in the runner
- adding provider-specific runtime adapters
- changing WorkItem status beyond `planned`
- adding assignment or claiming
- accepting raw source upload or unrestricted artifacts
- creating GateDecision from execution receipts
- creating Proof
- using receipt metadata as trusted attestation
