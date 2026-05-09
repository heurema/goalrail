# ADR-0032 - Typed project test command boundary

Status: accepted
Date: 2026-05-09

## Context

ADR-0031 established the H2.5 project command boundary after the fixed
`builtin_diagnostic/workspace_status` action. H2.5.1 then added only the typed
project-aware probe:

```text
Run(started)
  -> ExecutionCommandPlan(project_probe/detect_declared_test_targets)
  -> ExecutionReceipt(project_probe)
```

H2.5.1+ smoke coverage pins that probe as metadata-only. It does not run tests,
spawn a shell, accept command strings, accept user-provided `argv`, capture
stdout/stderr, upload artifacts, upload raw source, upload raw manifest bodies,
write `GateDecision`, or create `Proof`.

The next runtime boundary is different: typed project test execution. Even when
the command is allowlisted, it introduces real process execution, exit codes,
timeouts, output capture, possible workspace writes, possible network access,
secret leakage, flaky tests, and renewed proof-confusion risk.

This ADR is documentation-only. It defines the first safe project test command
boundary before any H2.6 implementation.

## Decision

Goalrail will not add arbitrary project test execution in H2.6.

Shell execution remains forbidden. `bash -lc`, `sh -c`, arbitrary command
strings, user-provided `argv`, LLM-selected commands, and package-manager
script strings remain forbidden.

The first allowed typed test command shape is:

```text
command_kind = "project_test"
action = "run_declared_test_target"
```

This action is not "run all tests". It represents one server-selected test
target that was previously detected by
`project_probe/detect_declared_test_targets`.

The server may create a `project_test/run_declared_test_target` command plan
only when all of these are true:

- the same WorkItem / CheckoutReceipt / Run lineage already has a matching
  `ExecutionReceipt(project_probe)`
- the project-probe metadata contains a supported declared test target
- the target source path is inside `path_scope`
- the server can map that target to one typed allowlisted execution policy
- the runner scope matches organization, project, repo binding, checkout
  receipt, execution job, and Run

The runner may execute an external process only for a server-created typed
test command plan that it recognizes and validates. The runner must reject any
plan whose kind, action, scope, generated argv, timeout, output policy, network
policy, write policy, artifact policy, or raw source policy does not match the
allowlist.

The API server must not execute the command, clone repositories, store
repository credentials, or receive raw source bodies by default.

## Command plan ownership

The server owns command plan creation.

Users and agents may request test intent in a future route, but they must not
submit raw command text, shell snippets, or argv. Any executable argv in a
stored command plan is server-generated from a typed policy and a declared test
target. It is not user-provided input.

The initial H2.6 command plan should carry or derive:

- `command_kind = "project_test"`
- `action = "run_declared_test_target"`
- `source_project_probe_receipt_id`
- `declared_test_target` metadata copied from the project-probe receipt
- `toolchain`
- `target_name`
- `target_source_path`
- server-generated argv or an equivalent typed runner action descriptor
- `shell_allowed = false`
- `working_directory`
- `path_scope`
- `timeout_seconds`
- network policy
- workspace write policy
- stdout/stderr capture policy
- artifact policy
- raw source upload policy

The first implementation must support one narrow target family at a time. It
must not add a broad language matrix, "all tests", package-manager script
execution, or runtime adapter layer in the same slice.

## Recommended first command policy

For the first implementation after this ADR:

```json
{
  "command_kind": "project_test",
  "action": "run_declared_test_target",
  "shell_allowed": false,
  "argv_policy": "server_generated_only",
  "working_directory": ".",
  "path_scope": ["."],
  "timeout_seconds": 120,
  "network_allowed": false,
  "workspace_write_allowed": false,
  "scratch_write_allowed": true,
  "stdout_capture": { "mode": "none" },
  "stderr_capture": { "mode": "none" },
  "artifacts_allowed": false,
  "changed_paths_allowed": false,
  "raw_source_upload_allowed": false
}
```

`scratch_write_allowed=true` means only runner-owned temporary/cache paths
outside the workspace path scope. It does not allow repository writes,
changed-path claims, or artifact upload.

If the runner cannot enforce the network or workspace-write policy for the
selected execution environment, it must fail closed before execution and submit
no test execution receipt.

## Output and secret policy

The first project test command slice disables stdout/stderr capture.

This avoids turning test logs into accidental proof, leaking secrets through
output, or inventing retention and redaction semantics before they are
designed.

A later slice may add capped and redacted stdout/stderr capture, but only after
it defines:

- per-stream byte limits
- truncation markers
- redaction rules
- retention policy
- whether output is inline, referenced, or runner-local only
- whether output can be consumed by GateDecision
- how failed redaction is handled

Until then, the runner must not upload raw stdout/stderr, raw manifest bodies,
raw source snippets, environment dumps, tokens, credentials, or package-manager
debug logs.

## Exit code semantics

A project test receipt may record an exit code only as process evidence.

Exit code `0` means the typed test process exited successfully under the
approved command plan. It does not mean the WorkItem is complete, acceptance
criteria passed, a gate passed, or proof exists.

Non-zero exit codes, timeouts, runner errors, or policy rejections are also
evidence inputs only. They may inform a later `GateDecision`, but they do not
create one.

The first implementation should distinguish at least:

- `process_status = "exited"`
- `process_status = "timed_out"`
- `process_status = "runner_error"`
- `process_status = "policy_rejected"`

The exact enum names may be narrowed during implementation, but they must not
collapse process outcome into Gate / Proof verdicts.

## Receipt and Run model

`ExecutionReceipt` remains evidence input only.

The one-command / one-receipt / one-Run rule remains invariant:

```text
one command plan -> one runner action -> one ExecutionReceipt -> one Run
```

H2.6 must not introduce multiple test commands in one Run. If Goalrail later
needs setup / test / teardown or multiple test targets inside one Run, a later
ADR must introduce `CommandAttempt` or an equivalent model before code.

`WorkItem` remains `planned` unless a later ADR changes WorkItem status
semantics.

`GateDecision` and `Proof` remain deferred.

## Runner policy

For typed project test execution, the runner must reject:

- unknown `command_kind`
- unknown `action`
- shell-enabled plans
- user-provided argv
- argv that does not exactly match server policy
- plans not derived from project-probe metadata
- plans for unsupported target families
- plans outside configured organization / project / repo binding scope
- plans for another checkout receipt, execution job, or Run
- working directories or path scopes outside the workspace
- missing or excessive timeout
- network access when policy says network is disabled
- workspace writes when policy says workspace writes are disabled
- stdout/stderr capture when policy says capture is disabled
- artifact upload when policy says artifacts are disabled
- changed path reporting when policy says changed paths are disabled
- raw source upload

The runner must never log lease tokens, credentials, raw stdout/stderr, raw
source, raw manifests, or secret-bearing environment values.

## Non-goals

This ADR does not implement or authorize:

- code changes
- shell execution
- arbitrary command strings
- user-provided `argv`
- `bash -lc`
- `sh -c`
- package-manager script strings
- LLM-selected commands
- provider runtime adapters
- "run all tests"
- multiple commands per Run
- setup / teardown command attempts
- stdout/stderr capture
- artifact upload
- changed path reporting
- raw source upload
- raw manifest body upload
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

## Recommended first implementation slice

```text
H2.6.1 - typed project_test command plan
```

H2.6.1 should add only one target family and one typed test command path:

- create or return a server-owned
  `ExecutionCommandPlan(project_test/run_declared_test_target)`
- derive the plan only from a matching `ExecutionReceipt(project_probe)`
- generate argv only from server-owned policy
- require `working_directory`, `path_scope`, timeout, network policy, write
  policy, stdout/stderr policy, artifact policy, and raw source policy
- make the runner validate the policy before execution
- execute at most one declared target per Run
- submit one `ExecutionReceipt(project_test)` as evidence input
- keep stdout/stderr capture disabled in the first slice
- keep artifacts, changed paths, raw source upload, GateDecision, Proof, and
  WorkItem status transitions out of scope

If the required runner environment controls for network and workspace writes
are not available, H2.6.1 should stop at server planning and runner
fail-closed validation rather than executing tests.

## Test plan for H2.6.1

Server tests:

- command plan creation requires a prior matching project-probe receipt
- command plan creation rejects unsupported target families
- command plan creation rejects user command strings and user-provided argv
- command plan includes `working_directory`, `path_scope`, timeout, network
  policy, write policy, stdout/stderr policy, artifact policy, and raw source
  policy
- command plan creation rejects non-started Runs
- command plan creation is scoped by organization / project / repo binding
- repeated creation returns the existing plan for the same Run and target
- project-test receipt requires matching command plan, Run, ExecutionJob, and
  CheckoutReceipt
- receipt submission treats exit code as evidence only
- receipt submission does not create GateDecision or Proof
- WorkItem remains `planned`

Runner tests:

- runner rejects unknown command kind or action
- runner rejects shell-enabled plans
- runner rejects user-provided or mismatched argv
- runner rejects plans not derived from project-probe metadata
- runner rejects unsupported target families
- runner rejects path scope outside the workspace
- runner rejects missing or excessive timeout
- runner rejects stdout/stderr capture in the first slice
- runner rejects artifacts, changed paths, and raw source upload
- runner fails closed when network / write policy cannot be enforced
- runner does not log lease tokens, command output, raw manifests, or secrets
- runner submits one bounded project-test receipt

Docs / smoke:

- STATUS and NEXT keep H2.6 as docs-first until code lands
- smoke coverage keeps H2.5.1+ project_probe metadata-only behavior intact
- smoke coverage pins one command receipt per Run
- smoke coverage pins receipt-as-evidence rather than Gate / Proof
