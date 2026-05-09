# ADR-0033 - Runner sandbox enforcement boundary

Status: accepted
Date: 2026-05-09

## Context

ADR-0032 defined the typed project test command boundary:

```text
ExecutionReceipt(project_probe)
  -> ExecutionCommandPlan(project_test/run_declared_test_target)
  -> runner policy validation
  -> ExecutionReceipt(project_test)
```

H2.6.2 implemented the fail-closed version of that boundary. The runner fetches
and validates the server-owned `project_test/run_declared_test_target` plan, but
does not execute the declared target because network and workspace-write
controls are not enforceable yet. It submits only:

```text
ExecutionReceipt(project_test, process_status=policy_rejected)
```

H2.6.2+ smoke coverage pins that fail-closed behavior. The runner does not run
fake `npm` or declared scripts, and the server rejects `project_test` process
statuses other than `policy_rejected`.

The next risk boundary is not test execution itself. It is whether a runner can
prove that the command environment enforces the policies that make execution
safe enough to attempt:

- no network egress when `network_allowed=false`
- no repository workspace writes when `workspace_write_allowed=false`
- bounded scratch/temp behavior
- no stdout/stderr capture
- no artifact or raw source upload
- no GateDecision, Proof, or WorkItem completion semantics

This ADR is documentation-only. It defines the minimum enforcement boundary
required before any later slice may allow `project_test` receipts with
`process_status=exited`, `timed_out`, or `runner_error`.

## Decision

Goalrail will not allow `project_test` process execution until runner
network/write enforcement semantics are defined and the runner can report that
the required controls are available for the specific command plan.

For the current runtime baseline, the only allowed `project_test` receipt
process status remains:

```text
process_status = "policy_rejected"
```

`process_status = "exited"`, `process_status = "timed_out"`, and
`process_status = "runner_error"` remain deferred for `project_test` until a
later implementation slice can enforce and evidence the required controls.

The runner must fail closed when required controls are unavailable. Fail closed
means no test process is started, no package-manager script is invoked, no shell
is spawned, no stdout/stderr is captured, no artifacts are uploaded, and no raw
source or raw manifest bodies are sent to the server.

The API server must continue to treat `ExecutionReceipt` as evidence input only.
Receipt evidence is not a `GateDecision`, not `Proof`, and not WorkItem
completion.

## Network policy semantics

`network_allowed=false` means the test process and its child processes must not
be able to open network connections.

It is not enough for the runner to promise that it will not intentionally make
network calls. The relevant control is the execution environment around the
test process.

Examples of acceptable future enforcement mechanisms may include:

- a platform sandbox with egress disabled
- a container or namespace with no network interface or blocked egress
- a documented OS-level firewall profile applied to the process tree
- another runner-owned mechanism that can be checked before execution and
  reported as active for the command

If the runner cannot enforce network blocking for the command process tree, it
must report network enforcement as unavailable and reject execution.

## Workspace write policy semantics

`workspace_write_allowed=false` means the test process and its child processes
must not be able to create, modify, rename, or delete files under the repository
workspace or command `path_scope`.

It is not enough to execute and later report an empty `changed_paths_summary`.
The control must prevent writes before they happen.

Examples of acceptable future enforcement mechanisms may include:

- mounting the workspace read-only
- running the command in an overlay where workspace writes are blocked or
  discarded before receipt submission
- filesystem permissions that deny write operations to the process identity
- another runner-owned mechanism that can be checked before execution and
  reported as active for the command

If the runner cannot enforce workspace write blocking for the command process
tree, it must report workspace-write enforcement as unavailable and reject
execution.

## Scratch, temp, and cache policy

`scratch_write_allowed=true` may allow writes only to runner-owned scratch
paths outside the repository workspace and outside command `path_scope`.

Scratch paths must not become artifact upload by implication. A later artifact
policy must explicitly decide whether scratch files can be referenced, retained,
uploaded, or inspected.

For H2.7, scratch/temp/cache behavior is only a policy distinction. It does not
authorize test execution.

## Capability model

Runner enforcement capability is command-relevant evidence, not a permanent
trust claim.

Future implementation should distinguish at least:

```json
{
  "network_enforcement": "unavailable",
  "workspace_write_enforcement": "unavailable",
  "scratch_write_policy": "runner_owned_paths_only",
  "enforcement_decision": "policy_rejected"
}
```

The exact field names may be narrowed during implementation, but the model must
preserve these distinctions:

- startup capability: what the runner believes it can support
- per-command capability: what is active for this command plan
- receipt evidence: what the runner reports for this Run
- trust/attestation: deferred until a later runner trust boundary

Startup capability alone is not enough to allow execution. The runner must
evaluate the selected command plan and record the per-command enforcement
decision.

## Receipt evidence

A future H2.7.1 receipt may include bounded enforcement metadata when it rejects
execution because controls are unavailable. That metadata is evidence input
only.

Recommended initial evidence shape:

```json
{
  "execution_mode": "project_test",
  "process_status": "policy_rejected",
  "enforcement": {
    "network": {
      "required": "blocked",
      "capability": "unavailable"
    },
    "workspace_write": {
      "required": "blocked",
      "capability": "unavailable"
    },
    "scratch_write": {
      "required": "runner_owned_paths_only",
      "capability": "not_evaluated_for_execution"
    }
  }
}
```

This metadata must not include raw process output, raw source, raw manifests,
environment dumps, tokens, credentials, or host-specific secrets.

## Fail-closed behavior

When enforcement is unavailable, the runner should submit a bounded
`ExecutionReceipt(project_test)` with:

- `process_status = "policy_rejected"`
- `exit_code = null`
- `artifact_refs = []`
- `changed_paths_summary = []`
- `raw_source_uploaded = false`
- no stdout/stderr refs or inline output
- optional bounded enforcement metadata after H2.7.1 defines the schema

If a future implementation cannot safely submit a receipt, it may fail without
receipt only if a later ADR explicitly changes the pull-loop behavior. The
current preferred model is a canonical `policy_rejected` receipt because it
keeps the one-command / one-receipt / one-Run contour observable.

## Invariants preserved

H2.7 preserves:

- no shell
- no arbitrary command string
- no user-provided `argv`
- no package-manager script execution unless a later slice enables it under
  enforceable controls
- no stdout/stderr capture
- no artifacts
- no changed paths
- no raw source upload
- no raw manifest body upload
- no GateDecision
- no Proof
- no WorkItem status transition
- one command receipt per Run

## Runner trust remains separate

Enforcement metadata is not runner attestation.

H2.7 does not solve runner registration, runner token hardening, trusted runner
identity, remote attestation, host isolation, or production-grade sandbox
provenance. Those remain a separate future boundary.

A self-reported runner capability can help the server and operator understand
why execution was rejected, but it is not proof that a hostile runner behaved
correctly.

## Non-goals

This ADR does not implement or authorize:

- code changes
- actual project test execution
- `process_status=exited` for `project_test`
- `process_status=timed_out` for `project_test`
- `process_status=runner_error` for a started test process
- `os/exec`
- shell execution
- arbitrary command strings
- user-provided `argv`
- `bash -lc`
- `sh -c`
- package-manager script execution
- stdout/stderr capture
- artifact upload
- changed path reporting
- raw source upload
- raw manifest body upload
- API-server command execution
- server-side repository clone
- repository credential storage
- runner registration or token hardening
- runner attestation
- WorkItem assignment or claiming
- WorkItem completion or delivery status transition
- GateDecision
- Proof

## Recommended first implementation slice

```text
H2.7.1 - runner enforcement capability declaration / fail-closed reporting
```

H2.7.1 should still not execute tests.

It should add only bounded capability reporting for the existing
`project_test/run_declared_test_target` fail-closed path:

- runner validates the server-owned command plan as H2.6.2 does today
- runner evaluates whether network and workspace-write controls are available
- current default result is unavailable / policy rejected
- runner submits or server accepts bounded enforcement metadata for
  `ExecutionReceipt(project_test, process_status=policy_rejected)`
- server continues rejecting `project_test` receipts with `exited`,
  `timed_out`, `runner_error`, artifacts, changed paths, raw source, or output
- WorkItem remains `planned`
- GateDecision and Proof remain deferred

## Test plan for H2.7.1

Server tests:

- accept bounded enforcement metadata only for
  `ExecutionReceipt(project_test, process_status=policy_rejected)`
- reject enforcement metadata that claims execution occurred
- reject `project_test` receipts with `exited`, `timed_out`, or `runner_error`
  until a later ADR enables execution
- reject stdout/stderr, artifacts, changed paths, and raw source upload
- receipt submission does not create GateDecision or Proof
- WorkItem remains `planned`

Runner tests:

- reports network enforcement unavailable in the current environment
- reports workspace-write enforcement unavailable in the current environment
- submits `policy_rejected` without invoking package managers or test scripts
- does not call `os/exec`, `exec.Command`, `bash -lc`, or `sh -c`
- does not capture stdout/stderr
- does not upload artifacts, changed paths, raw source, or raw manifests
- does not log lease tokens, command output, raw manifests, environment values,
  or secrets

Docs / smoke:

- STATUS and NEXT keep H2.7 as enforcement-boundary planning, not execution
- smoke coverage keeps H2.6.2+ fail-closed behavior intact
- smoke coverage preserves one command receipt per Run
- smoke coverage keeps receipts as evidence input, not GateDecision or Proof
