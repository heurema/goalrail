# ADR-0030 - Bounded command execution boundary

Status: accepted
Date: 2026-05-08

## Context

ADR-0029 established the H2 runtime boundary after checkout receipt:

```text
WorkItem(planned)
  -> CheckoutReceipt
  -> ExecutionJob
  -> ExecutionLease
  -> Run(started)
  -> ExecutionReceipt(no_command)
```

H2.3 intentionally added only metadata receipt submission. A
`ExecutionReceipt(no_command)` is evidence input from a runner, not command
execution, task completion, `GateDecision`, or `Proof`.

The next risk boundary is real command execution. This is materially different
from no-command receipts because it introduces process behavior, workspace path
scope, output capture, artifact references, exit status, timeout behavior,
secret redaction, and future gate/proof interpretation risk.

Goalrail must not grow from a typed runtime boundary into a generic arbitrary
shell platform by accident.

This ADR is documentation-only. It defines the smallest safe command execution
boundary before any code implements command execution.

## Decision

Goalrail command execution will be introduced as a bounded runner action, not
as arbitrary shell execution.

The API server must not execute commands, clone repositories, or store
repository credentials. The runner remains the only component allowed to touch a
workspace and execute bounded runtime actions.

Shell execution is not allowed in the first command execution implementation
slice. The first implementation must not accept a user-provided command string,
`bash -lc`, arbitrary `argv`, project test command, package-manager command, or
provider-specific runtime adapter.

The first allowed execution mode is:

```text
execution_mode = "builtin_diagnostic"
action = "workspace_status"
```

This is a fixed runner built-in diagnostic action. It is not a project command.
It is not an LLM coding-agent invocation. It is not a provider adapter. It is
used to prove the command-plan / runner-action / receipt protocol before
opening project command execution.

## Command ownership

The server owns command authorization.

The runner owns execution.

Agents and users do not submit arbitrary commands in the first implementation
slice.

For H2.4.1, the server may create or return one fixed command plan for a
started Run:

```text
Run(started)
  -> ExecutionCommandPlan(builtin_diagnostic/workspace_status)
```

The runner may execute only a server-issued command plan whose kind/action it
recognizes and whose scope matches the runner's configured organization,
project, repo binding, checkout receipt, and workspace.

## Object model

### `ExecutionCommandPlan`

Purpose:

- immutable server-owned authorization for one bounded runner action
- binds command execution to one `Run(started)`
- prevents runner-side invention of arbitrary commands
- records the exact action policy that the runner is allowed to execute

Minimum fields:

- `id`
- `organization_id`
- `project_id`
- `repo_binding_id`
- `task_id`
- `checkout_receipt_id`
- `execution_job_id`
- `run_id`
- `command_kind`
- `action`
- `shell_allowed`
- `argv`
- `working_directory`
- `path_scope`
- `timeout_seconds`
- `max_stdout_bytes`
- `max_stderr_bytes`
- `allowed_artifact_kinds`
- `raw_source_upload_allowed`
- `state`
- `created_at`
- `updated_at`

H2.4.1 values:

```json
{
  "command_kind": "builtin_diagnostic",
  "action": "workspace_status",
  "shell_allowed": false,
  "argv": [],
  "working_directory": ".",
  "path_scope": ["."],
  "timeout_seconds": 30,
  "max_stdout_bytes": 0,
  "max_stderr_bytes": 0,
  "allowed_artifact_kinds": [],
  "raw_source_upload_allowed": false,
  "state": "planned"
}
```

### `ExecutionReceipt(command_metadata)`

Purpose:

- runner-submitted evidence input describing the bounded action result
- records what action was attempted and what metadata was produced
- remains separate from Gate / Proof verdicts

H2.4.1 receipt fields should extend the receipt model with bounded command
metadata:

- `execution_mode = "builtin_diagnostic"`
- `command_plan_id`
- `command_kind = "builtin_diagnostic"`
- `action = "workspace_status"`
- `runner_started_at`
- `runner_finished_at`
- `process_status`
- `exit_code = null`
- `stdout_ref = null`
- `stderr_ref = null`
- `artifact_refs = []`
- `changed_paths_summary = []`
- `raw_source_uploaded = false`

The first built-in diagnostic action does not produce process stdout/stderr
streams and does not create artifact refs. It may report bounded metadata such
as whether the configured workspace reference was present and whether the
checkout receipt context matched the runner scope. It must not upload raw
source bodies.

## State transitions

H2.4.1 should use this contour:

```text
Run(started)
  -> ExecutionCommandPlan(planned)
  -> runner executes builtin diagnostic action
  -> ExecutionReceipt(command_metadata)
  -> Run(receipt_submitted)
  -> ExecutionJob(receipt_submitted)
```

`WorkItem` remains `planned`. Assignment, claiming, completion, delivery, and
acceptance status transitions remain deferred.

## Runner command policy

The runner must enforce all of these rules:

- reject unknown `command_kind`
- reject unknown `action`
- reject `shell_allowed=true`
- reject non-empty `argv` for `builtin_diagnostic`
- reject path scope outside the configured workspace
- reject command plans for another `project_id` or `repo_binding_id`
- reject command plans for another `checkout_receipt_id`
- reject raw source upload
- never log lease tokens or secrets
- never persist raw lease tokens to disk
- never treat action completion as Gate / Proof

For H2.4.1, `workspace_status` must not call `os/exec`, spawn a shell, run
project test commands, invoke a package manager, create branches, write commits,
or call LLM/runtime-provider APIs.

## Output capture and artifacts

H2.4.1 must not store inline stdout/stderr.

Future command execution slices may add bounded output capture, but they must
decide all of these explicitly before implementation:

- inline truncation limits
- artifact reference storage
- redaction rules
- max artifact size
- allowed artifact kinds
- retention policy
- whether artifacts are runner-local, server-stored, or externally addressed

Until then, `stdout_ref`, `stderr_ref`, and `artifact_refs` stay empty for the
first built-in diagnostic action.

## Secret and source boundaries

The runner may need workspace credentials in later slices. That does not make
the API server a credential store.

The API server must not receive repository credentials or raw source bodies by
default. Receipts may contain bounded metadata and artifact references only.

Any future raw source upload or source snippet policy requires a separate ADR.

## Difference from GateDecision and Proof

Execution receipts are evidence inputs.

They do not decide whether acceptance criteria passed. They do not certify that
the task is done. They do not create `GateDecision`. They do not create `Proof`.

Gate / Proof remain later boundaries that may consume execution receipts as
inputs.

## Non-goals

This ADR does not implement or authorize:

- code changes
- arbitrary shell execution
- user-provided command strings
- arbitrary `argv`
- `bash -lc`
- project commands such as `npm test`, `go test`, or package-manager scripts
- command execution in the API server
- provider-specific runtime adapter
- LLM coding-agent integration
- branch creation
- commit creation
- pull request or merge request creation
- WorkItem assignment or claiming
- WorkItem completion or delivery status transition
- GateDecision
- Proof
- raw source upload
- server-side repository clone
- repository credential storage in the API server
- broad queue/outbox/runtime registry
- runner registration or dedicated runner-token hardening
- Problem Details migration

## First implementation slice

Recommended first code slice after this ADR:

```text
H2.4.1 - builtin diagnostic command plan and receipt
```

H2.4.1 should add:

- server-owned command plan creation or return for `Run(started)`
- one command kind: `builtin_diagnostic`
- one action: `workspace_status`
- runner support for executing that built-in action without shell/process
  execution
- `ExecutionReceipt(command_metadata)` submission for the built-in action
- tests proving no arbitrary shell, no project command execution, no
  GateDecision, no Proof, and no WorkItem status transition

H2.4.1 must not add real project command execution.

## Test plan for H2.4.1

Server tests:

- command plan creation requires authenticated user or runner route according to
  the chosen route boundary
- command plan creation rejects non-started Run
- command plan creation is scoped by organization / project / repo binding
- command plan creation returns existing plan for the same Run and action
- command plan never contains shell strings or user-provided `argv`
- command metadata receipt requires matching Run / ExecutionJob / CheckoutReceipt
- command metadata receipt rejects unknown command plan
- command metadata receipt rejects raw source upload
- command metadata receipt does not create GateDecision or Proof
- WorkItem remains `planned`

Runner tests:

- runner rejects unknown command kind
- runner rejects unknown action
- runner rejects `shell_allowed=true`
- runner rejects non-empty `argv` for `builtin_diagnostic`
- runner rejects path scope outside workspace
- runner does not import or call `os/exec`
- runner does not log raw lease tokens
- runner does not call project package-manager or test commands
- runner submits bounded command metadata receipt

Docs / smoke:

- docs keep command execution status honest
- STATUS/NEXT do not claim arbitrary command execution, GateDecision, or Proof
- smoke coverage should pin the first built-in diagnostic path before any
  project command execution slice
