# ADR-0034 - Runner capability model boundary

Status: accepted
Date: 2026-05-09

## Context

ADR-0033 defined the runner sandbox / network / workspace-write enforcement
boundary required before any `project_test` receipt may record real process
outcomes.

The current baseline remains fail-closed:

```text
ExecutionCommandPlan(project_test/run_declared_test_target)
  -> runner policy validation
  -> ExecutionReceipt(project_test, process_status=policy_rejected)
  -> enforcement_report(network/workspace/process_tree unavailable)
```

H2.7.1 added canonical `enforcement_report` metadata for this fail-closed
receipt path. H2.7.1+ smoke coverage pins the report shape, active-control
claim rejection, no-command process-status guard, no-process runner behavior,
and no token logging.

The next design risk is capability drift. A runner may be able to say it
supports a sandbox, network isolation, workspace write blocking, process-tree
control, output policy, or artifact policy. That statement is useful metadata,
but it is not proof that the controls exist, were configured correctly, were
active for a command, or should unlock test execution.

This ADR is documentation-only. It defines the capability model boundary before
any implementation records runner capabilities.

## Decision

Goalrail will introduce a runner capability model as metadata first.

Self-declared runner capabilities are untrusted metadata. They may explain what
the runner operator or runner process says it can support, but they do not
authorize command execution, do not make controls trusted, and do not allow
`project_test` receipts to record `exited`, `timed_out`, or `runner_error`.

For the current runtime, `project_test` remains:

```text
process_status = "policy_rejected"
```

Trusted capabilities require a later runner registration / trust boundary. That
future boundary must define server-owned runner identity, authentication,
scope, registration lifecycle, operator configuration, possible attestation,
revocation, and per-command enforcement evidence before any capability can be
used for execution policy.

Capability metadata alone can never unlock execution. Even a future trusted
runner capability record is only one input. A later ADR and implementation must
explicitly allow `project_test` process execution, define receipt status
semantics, and preserve the evidence vs GateDecision vs Proof separation.

## RunnerCapabilityReport

A `RunnerCapabilityReport` is a structured declaration about controls a runner
claims to support.

It is not an `ExecutionReceipt`, not a `GateDecision`, not `Proof`, and not an
attestation.

Suggested first shape:

```json
{
  "capability_report_id": "runner-capability-report-1",
  "runner_id": "runner-1",
  "organization_id": "organization-1",
  "project_id": "project-1",
  "repo_binding_id": "repo-binding-1",
  "network_isolation_declared": false,
  "workspace_write_isolation_declared": false,
  "process_tree_control_declared": false,
  "stdout_stderr_policy_declared": "none",
  "artifact_policy_declared": "none",
  "trust_state": "self_declared_untrusted",
  "reported_at": "2026-05-09T00:00:00Z"
}
```

The exact schema may be narrowed during implementation, but the model must keep
these fields conceptually separate:

- runner identity
- organization / project / repo-binding scope
- declared network isolation capability
- declared workspace-write isolation capability
- declared process-tree control capability
- declared stdout/stderr policy capability
- declared artifact policy capability
- trust state
- report timestamp

Capability reports must not include raw source, raw manifests, stdout/stderr,
environment dumps, credentials, tokens, host secrets, or broad machine
inventory.

## Self-declared capabilities

In the first capability slice, a runner may self-declare only metadata such as:

- network isolation declared / not declared
- workspace-write isolation declared / not declared
- process-tree control declared / not declared
- stdout/stderr policy declared
- artifact policy declared
- scratch/temp policy declared

All self-declared reports must use:

```text
trust_state = "self_declared_untrusted"
```

The server must reject reports that claim a trusted state unless a later trust
boundary has been implemented.

Self-declared capability metadata may be displayed to operators or included in
diagnostic evidence. It must not:

- unlock `process_status=exited`
- unlock `process_status=timed_out`
- unlock `process_status=runner_error` for a started project-test process
- allow `os/exec`
- allow shell
- allow stdout/stderr capture
- allow artifacts
- allow workspace writes
- allow network
- create GateDecision
- create Proof
- transition WorkItem out of `planned`

## Trusted capability requirements

No runner capability is trusted in H2.7.2.

A later ADR must define a trusted capability state before any trusted report can
exist. At minimum, trusted capability requires:

- server-owned runner registration
- stable server-side runner identity
- authentication bound to that runner identity
- organization / project / repo-binding scope
- operator-controlled configuration or attestation model
- explicit enforcement mechanism per declared control
- per-command evaluation showing controls were active for the command plan
- revocation or expiry semantics
- audit logs or receipts that distinguish declaration from enforcement

Runner registration alone is not enough. A registered runner can still be
misconfigured, outdated, or unable to enforce a specific command policy.

Operator configuration alone is not enough. It records intent or installation
state, not proof that controls were active for a command process tree.

Attestation alone is not enough unless a later ADR defines what is attested, who
trusts it, how it is verified, how it expires, and how it maps to command
policy.

## Trust scope

Capability trust must not be global by default.

The future model may have multiple scopes:

- per runner: what a registered runner is generally configured to support
- per lease: what the runner claims for a leased work scope
- per job: what the runner can support for an execution job
- per command: what controls were evaluated and active for one command plan

Only the per-command enforcement decision can be considered for project-test
execution. Broader scopes can inform scheduling or operator diagnostics, but
they do not prove a specific command was sandboxed.

## Execution unlock rule

Self-declared capability metadata never unlocks process execution.

Trusted capability metadata also does not unlock execution by itself. A later
execution ADR must decide all of the following before `project_test` can record
process outcomes:

- which trusted capability states are accepted
- which command policy can execute
- how network isolation is enforced and evidenced
- how workspace-write isolation is enforced and evidenced
- how process-tree control is enforced and evidenced
- whether stdout/stderr remain disabled or gain a capped/redacted policy
- whether artifacts remain disabled
- what `exited`, `timed_out`, and `runner_error` mean
- how failures remain evidence inputs, not GateDecision or Proof

Until that later boundary lands, the server must continue to reject
`project_test` receipts that claim real process outcomes.

## Relationship to enforcement_report

`RunnerCapabilityReport` and `enforcement_report` answer different questions.

`RunnerCapabilityReport` answers:

```text
What does this runner say it can support?
```

`enforcement_report` answers:

```text
What enforcement decision did the runner record for this receipt?
```

For the current baseline, `enforcement_report` remains canonical for the
fail-closed `project_test` receipt:

```text
network_enforcement = "unavailable"
workspace_write_enforcement = "unavailable"
process_tree_enforcement = "unavailable"
decision = "policy_rejected"
reason = "enforcement_unavailable"
```

If H2.7.3 records self-declared capability reports, project-test receipt
behavior still remains unchanged.

## Proposed H2.7.3 implementation slice

Recommended next implementation slice:

```text
H2.7.3 - self-declared runner capability report metadata
```

Scope:

- add a server-owned `RunnerCapabilityReport` shape
- allow runner submission or recording of self-declared capability metadata
- require `trust_state = self_declared_untrusted`
- scope reports to runner / organization / project / repo binding
- reject trusted or active enforcement claims
- keep `project_test` receipts `policy_rejected`
- do not change command planning
- do not execute tests

Non-goals for H2.7.3:

- no runner registration
- no trusted runner identity beyond the current authenticated request context
- no attestation
- no sandbox implementation
- no network isolation implementation
- no workspace-write enforcement implementation
- no process execution
- no `process_status=exited`
- no GateDecision
- no Proof

## Test plan for H2.7.3

Future implementation tests should cover:

- accepts a self-declared capability report with `trust_state=self_declared_untrusted`
- rejects any report that claims trusted, active, attested, or enforced state
- rejects reports outside the caller's organization / project / repo-binding
  scope
- rejects raw output, raw source, raw manifest, environment, token, or secret
  fields
- persists report metadata without mutating execution jobs, runs, receipts,
  WorkItems, GateDecision, or Proof
- keeps `project_test` receipt validation limited to `policy_rejected`
- runner logs do not include tokens or secrets while reporting capability
- docs/status do not claim sandboxing or actual test execution

## Non-goals

This ADR does not implement or authorize:

- code changes
- runner capability storage
- runner registration
- trusted runner identity
- runner attestation
- sandbox implementation
- network isolation implementation
- workspace-write enforcement implementation
- process-tree enforcement implementation
- actual project test execution
- `process_status=exited` for `project_test`
- `process_status=timed_out` for `project_test`
- `process_status=runner_error` for a started project-test process
- `os/exec`
- shell execution
- arbitrary command strings
- user-provided `argv`
- `bash -lc`
- `sh -c`
- stdout/stderr capture
- artifact upload
- changed path reporting
- raw source upload
- raw manifest body upload
- API-server command execution
- server-side repository clone
- repository credential storage
- WorkItem assignment or claiming
- WorkItem status transition
- GateDecision
- Proof

## Consequences

This keeps H2.7 conservative:

- runners may start reporting capability metadata later
- operators can inspect why execution remains blocked
- the server does not mistake runner intent for enforcement
- project-test execution remains fail-closed until a separate trust and
  execution boundary exists

The cost is another small slice before real test execution. That is deliberate:
capability declaration, trust, enforcement, and execution are different
boundaries and must not collapse into one implementation step.
