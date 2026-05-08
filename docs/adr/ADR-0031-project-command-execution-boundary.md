# ADR-0031 - Project command execution boundary

Status: accepted
Date: 2026-05-08

## Context

ADR-0029 established that `ExecutionReceipt` is runner evidence input, not a
`GateDecision` or `Proof`.

ADR-0030 then introduced bounded command execution through the fixed
`builtin_diagnostic/workspace_status` action:

```text
WorkItem(planned)
  -> CheckoutJob
  -> CheckoutReceipt
  -> ExecutionJob(queued)
  -> ExecutionLease
  -> Run(started)
  -> ExecutionCommandPlan(builtin_diagnostic/workspace_status)
  -> ExecutionReceipt(builtin_diagnostic)
```

H2.4.1 proves the command-plan / runner-action / receipt protocol without
project command execution. It intentionally keeps `artifact_refs`,
`changed_paths_summary`, stdout/stderr refs, raw source upload, Gate, Proof, and
WorkItem status transitions out of the runtime baseline.

The next risk boundary is project-aware command execution. That boundary can
easily drift into arbitrary shell execution, accidental stdout/stderr or
artifact proof claims, unstable multi-command receipt semantics, or production
runner trust assumptions. Goalrail needs a command-policy decision before any
H2.5 code slice.

This ADR is documentation-only. It defines the safe project command execution
boundary before implementation.

## Decision

Goalrail will not allow arbitrary shell execution in the MVP.

Goalrail will not accept arbitrary command strings, `bash -lc`, `sh -c`, or
user-provided `argv` for project command execution in the MVP.

The first project command step must be typed, allowlisted, and policy-bound.
It must be represented as a server-owned `ExecutionCommandPlan`, not as a raw
command line.

The server owns command plan creation and authorization.

The runner executes only server-approved typed command plans whose
organization, project, repo binding, checkout receipt, Run, command kind,
action, working directory, and path scope match the runner's configured scope.

Every project command plan must include:

- `command_kind`
- `action`
- `working_directory`
- `path_scope`
- shell policy
- argv policy
- network policy
- write policy
- artifact policy
- stdout/stderr capture policy
- raw source upload policy

`working_directory` and `path_scope` are part of the authorization boundary,
not display metadata. A runner must reject plans whose working directory or path
scope escapes the configured workspace or exceeds the checkout / WorkItem
scope.

Stdout/stderr capture must be explicitly bounded or disabled per command kind.
It must not become implicit inline evidence. The first project command step
disables stdout/stderr capture.

Artifact refs are evidence references only. They are not proof, not final
verification, and not acceptance criteria satisfaction.

`ExecutionReceipt` remains evidence input only. It records what the runner
attempted under an approved plan and what bounded metadata or evidence refs were
produced. It does not decide success, create a `GateDecision`, create `Proof`,
or complete a `WorkItem`.

`GateDecision` and `Proof` remain deferred.

`WorkItem` remains `planned` unless a later ADR changes WorkItem status
semantics.

Runner trust, runner registration, runner token hardening, runner attestation,
and production runner identity remain a separate future boundary. H2.5 must not
smuggle production trust claims into command execution.

## One receipt per Run

The current H2 model keeps one command receipt per Run.

H2.5 will preserve this rule:

```text
one command plan -> one runner action -> one ExecutionReceipt -> one Run
```

If Goalrail later needs multiple commands inside one Run, it must introduce a
separate `CommandAttempt` or equivalent model through a later ADR before code.
Until then, future command execution should create separate Runs rather than
packing multiple command attempts into one receipt.

This keeps the receipt model stable and prevents stdout/stderr, artifacts,
changed paths, or process status from becoming ambiguous across several
commands.

## First project command shape

The first safe project command after the built-in diagnostic is a project probe,
not a test run:

```json
{
  "command_kind": "project_probe",
  "action": "detect_declared_test_targets",
  "working_directory": ".",
  "path_scope": ".",
  "shell": false,
  "argv": null,
  "network_allowed": false,
  "write_allowed": false,
  "artifacts_allowed": false,
  "changed_paths_allowed": false,
  "stdout_capture": { "mode": "none" },
  "stderr_capture": { "mode": "none" }
}
```

`project_probe/detect_declared_test_targets` may inspect only allowlisted
project metadata inside `path_scope` to detect declared test targets for later
planning. It must not execute those targets.

It is not `npm test`, `go test ./...`, `pytest`, a package-manager script, a
shell command, an LLM-selected command, or a runtime adapter invocation.

## Runner policy

For H2.5.1, the runner must reject:

- unknown `command_kind`
- unknown `action`
- shell-enabled plans
- any non-null or non-empty `argv`
- plans with network access
- plans with write access
- plans allowing artifacts
- plans allowing changed paths
- stdout/stderr capture modes other than `none`
- raw source upload
- plans outside the configured organization / project / repo binding scope
- plans for another checkout receipt or Run
- plans whose working directory or path scope escapes the workspace

The runner must not call `os/exec`, spawn a shell, invoke a package manager,
run project tests, create branches, write commits, call provider APIs, or call
LLM / coding-agent APIs for the first project probe.

## Receipt policy

The first project-probe receipt may report bounded metadata such as:

- command plan id
- command kind and action
- runner start / finish timestamps
- `process_status`
- detected target metadata refs or inline bounded structured metadata if later
  narrowed by implementation

It must keep:

- `exit_code = null`
- `stdout_ref = null`
- `stderr_ref = null`
- `artifact_refs = []`
- `changed_paths_summary = []`
- `raw_source_uploaded = false`

Any future stdout/stderr capture, artifact upload, changed-path reporting, raw
source upload, or process exit-code semantics require explicit narrowing before
implementation.

## Recommended first implementation slice

```text
H2.5.1 - typed project_probe command plan
```

H2.5.1 should add only:

- server-owned plan creation or return for
  `project_probe/detect_declared_test_targets`
- runner validation for the typed allowlisted project-probe policy
- runner-owned metadata detection without shell/process execution
- one `ExecutionReceipt(project_probe)` as evidence input
- smoke coverage proving no shell, no arbitrary command string, no user argv,
  no stdout/stderr capture, no artifacts, no changed paths, no GateDecision, no
  Proof, and no WorkItem status transition

H2.5.1 must preserve the one-command / one-receipt / one-Run rule.

## Non-goals

This ADR does not implement or authorize:

- code changes
- shell execution
- arbitrary command strings
- user-provided `argv`
- `bash -lc`
- `sh -c`
- project test execution
- `npm test`
- `go test ./...`
- `pytest`
- package-manager script execution
- LLM-selected commands
- provider runtime adapter
- LLM coding-agent integration
- artifact upload
- stdout/stderr capture
- changed path reporting
- raw source upload
- API-server command execution
- server-side repository clone
- repository credential storage
- runner registration or runner token hardening
- runner attestation
- branch creation
- commit creation
- pull request or merge request creation
- WorkItem assignment or claiming
- WorkItem completion or delivery status transition
- GateDecision
- Proof

## Test plan for H2.5.1

Server tests:

- command plan creation returns only the typed
  `project_probe/detect_declared_test_targets` action
- command plan creation rejects non-started Runs
- command plan creation is scoped by organization / project / repo binding
- command plan contains `working_directory` and `path_scope`
- command plan forbids shell, arbitrary command strings, and user-provided
  `argv`
- command plan disables network, writes, artifacts, changed paths,
  stdout/stderr capture, and raw source upload
- repeated creation returns the existing plan for the same Run and action
- project-probe receipt requires matching command plan, Run, ExecutionJob, and
  CheckoutReceipt
- receipt submission does not create GateDecision or Proof
- WorkItem remains `planned`

Runner tests:

- runner rejects unknown command kind or action
- runner rejects shell-enabled plans
- runner rejects any `argv`
- runner rejects network/write/artifact/changed-path/stdout/stderr permissions
- runner rejects path scope outside the workspace
- runner does not import or call `os/exec`
- runner does not call package managers or project test commands
- runner does not log lease tokens or secrets
- runner submits one bounded project-probe receipt

Docs / smoke:

- STATUS and NEXT keep project command execution status honest
- README/root overview points readers to STATUS / COMPONENTS for implementation
  truth without rewriting the public narrative
- smoke coverage keeps H2.4.1 builtin diagnostic behavior intact
- smoke coverage pins H2.5.1 as typed project-probe metadata only
