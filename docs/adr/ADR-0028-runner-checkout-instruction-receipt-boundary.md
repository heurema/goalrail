# ADR-0028 — Runner checkout instruction and workspace receipt boundary

Status: accepted
Date: 2026-05-07

## Context

The agent pull loop now reaches canonical planned work:

```text
Goal
  -> Contract
  -> ApprovedContract
  -> WorkItemPlan
  -> WorkItemPlanProposal
  -> WorkItem(planned)
```

The compact smoke baseline covers the path through `WorkItem(planned)`.
`WorkItem(planned)` is still non-executable. It is not assignment, claiming,
repository checkout, `Run`, receipt, `GateDecision`, or `Proof`.

ADR-0008 already sets the repository checkout boundary: the API server must not
clone repositories, run commands, or store repository secrets. Repository access
belongs behind a runner boundary, and runner-owned local credentials are the MVP
direction.

ADR-0025 sets the local repository evidence boundary:
`RepositoryBaselineProfile` and `WorkspaceOverlay` are local CLI / runner-owned
receipts. Server-side repository clone, raw source upload by default, and
server-maintained repository indexes remain out of scope.

The next delivery-runtime boundary needs one concrete protocol slice before any
execution work:

```text
WorkItem(planned)
  -> CheckoutInstruction
  -> runner-owned workspace preparation
  -> CheckoutReceipt
```

This ADR was accepted as a boundary decision before H1 implementation. H1 now
implements the checkout job / instruction and workspace receipt protocol slice,
but still does not implement actual repository clone/fetch, arbitrary command
execution, `Run`, gate, or proof.

## Decision

Goalrail will introduce a narrow runner checkout preparation boundary before any
execution boundary.

The first code slice after this ADR should create a bounded checkout instruction
from an existing `WorkItem(planned)` and `RepoBinding` context, then accept a
runner-submitted workspace receipt. The runner owns repository credentials and
workspace preparation. The API server owns canonical state and may store bounded
metadata receipts, but it must not receive repository secrets or raw source
bodies by default.

`WorkItem` remains `planned` during this boundary. H1 must not introduce a
`running`, `executing`, or `completed` WorkItem state. If execution later needs a
`Run` record, that belongs to a later explicit runtime boundary.

`CheckoutInstruction` and `CheckoutReceipt` are sidecar delivery-runtime
preparation records. They are not `Run`, not proof, and not gate verdicts.

## H1 implementation target

Implemented first slice:

```text
H1 — runner checkout instruction + workspace receipt
```

H1 should add only:

- server-owned checkout instruction / checkout job creation for a planned task
- a separate runner-side binary/process boundary if checkout is performed
- runner-owned local credential configuration
- read-only checkout or mounted-workspace verification
- checkout receipt submission with bounded metadata
- status / next-action output that says execution remains unavailable

H1 should not add:

- assignment
- claiming
- arbitrary command execution
- `Run`
- execution receipt
- `Decision` / `GateDecision`
- `Proof`
- provider OAuth or provider clients
- server-side clone
- API-server repository credential storage

## Proposed public and runner-facing surfaces

Product-facing task preparation route:

```text
POST /v1/tasks/{id}/checkout-jobs
```

Purpose:

- authenticated user / agent asks Goalrail to prepare checkout for a planned
  WorkItem
- server validates Organization membership and project / repo expectations
- server derives checkout metadata from `WorkItem` and `RepoBinding`
- server creates or returns one checkout job / instruction for that task

Runner-facing routes:

```text
POST /v1/checkout-jobs/leases
POST /v1/checkout-jobs/{id}/receipts
```

Purpose:

- runner pulls one bounded checkout job
- server returns a `CheckoutInstruction`
- runner prepares a workspace using local credentials or mounted workspace
- runner submits a `CheckoutReceipt`

The exact URL vocabulary may be adjusted during implementation, but the product
shape must preserve these boundaries: task-level preparation, runner-side
checkout instruction, runner-submitted receipt.

If H1 does not yet implement runner leasing, it may use an explicit
`goalrail-runner checkout --task-id <task_id> --once` development path. That
path must still be authenticated, runner-owned for credentials, and API-only for
canonical state. It must not add an unauthenticated or manual database bypass.

## Proposed runner binary

The runner should be separate from `apps/server` and separate from the planning
worker.

Recommended placement:

```text
apps/runner/cmd/goalrail-runner
apps/runner/internal/...
```

The runner startup config should include Goalrail API connection and local
credential file paths only, for example:

- `--server-url` / `GOALRAIL_RUNNER_SERVER_URL`
- `--runner-id` / `GOALRAIL_RUNNER_ID`
- runner API token file or equivalent narrow runner auth input
- Git HTTPS token file
- SSH key file
- known_hosts file
- mounted workspace root, when using mounted workspace mode

Runner startup config must not hard-code a repository URL, RepoBinding ID,
checkout mode, task ID, branch, or path scope as canonical truth. Those arrive
from API-issued checkout instructions.

## CheckoutInstruction v0 fields

Minimum instruction fields:

- `checkout_job_id`
- `task_id`
- `contract_id`
- `approved_contract_id`
- `plan_id`
- `proposal_id`
- `organization_id`
- `project_id`
- `repo_binding_id`
- `repository_url`
- `repository_full_name`
- `provider`
- `ref`
- `workflow_base_branch`
- `path_scope`
- `checkout_mode`
- `auth_hint`
- `created_at`
- `expires_at`

Rules:

- `repository_url` is metadata, not a secret.
- `auth_hint` must not contain tokens, private keys, passwords, or credential
  material.
- `ref` should be explicit. If the first slice uses the workflow base branch,
  that must be stated honestly; later slices may tighten this to a commit SHA or
  approved baseline ref.
- `path_scope` may be empty only when the RepoBinding has no narrower path
  scope.

## CheckoutReceipt v0 fields

Minimum receipt fields:

- `checkout_receipt_id`
- `checkout_job_id`
- `task_id`
- `runner_id`
- `runner_mode`
- `checkout_mode`
- `repo_binding_id`
- `repository_url`
- `ref`
- `commit_sha`
- `path_scope`
- `repository_baseline_profile_id`
- `workspace_overlay_id`
- `dirty`
- `partial`
- `partiality_reasons`
- `raw_source_uploaded`
- `workspace_ref`
- `artifact_refs`
- `checkout_started_at`
- `checkout_finished_at`
- `workspace_cleaned`
- `receipt_created_at`

Rules:

- `raw_source_uploaded` must be `false` in H1.
- `workspace_ref` must be a bounded runner-local or opaque reference, not a
  portable secret and not an unrestricted source dump.
- `artifact_refs` may be empty in H1.
- local filesystem paths should not become product truth; if included for
  debugging, they must be runner-local and not used as canonical identity.
- dirty, partial, sparse checkout, shallow clone, submodule, and unmerged states
  must be explicit rather than hidden behind a ready verdict.

## State transitions

H1 may introduce checkout preparation states such as:

```text
CheckoutJob(queued)
CheckoutJob(leased)
CheckoutJob(receipt_submitted)
CheckoutJob(expired)
CheckoutJob(cancelled)
```

`WorkItem(planned)` remains unchanged by checkout preparation.

Submitting a `CheckoutReceipt` may mark only the checkout job as
`receipt_submitted`. It must not mark a WorkItem as assigned, claimed, running,
verified, accepted, or complete.

`Run` remains deferred. A later runtime ADR or slice must define when `Run` is
created and how it relates to a checkout receipt.

## Server responsibilities

The API server must:

- authenticate the caller before checkout job creation
- authorize against current server-side `OrganizationMembership`
- verify `WorkItem.organization_id` matches the caller's Organization
- verify supplied `project_id` / `repo_binding_id` expectations before mutation
- require `WorkItem.status = planned`
- resolve the active `RepoBinding`
- create or return a bounded checkout job for the task
- issue checkout instructions without secrets
- accept bounded checkout receipts
- persist metadata / events transactionally when DB is configured
- keep repository clone, workspace prep, command execution, and credential
  custody outside the API server

The server must not:

- clone repositories
- inspect source trees
- run Git commands, tests, builds, linters, or runtime commands
- store repository tokens, SSH keys, passwords, known_hosts contents, or
  provider credentials
- accept raw source bodies by default
- create `Run`, `Decision`, `GateDecision`, or `Proof` in H1

The existing unauthenticated task read surface must not become the runner
handoff boundary. H1 must either harden task read/status with authentication and
Organization / project / repo checks or use a new authenticated preparation
route that performs those checks.

## Runner responsibilities

The runner must:

- run as a separate process from the API server
- talk to Goalrail only through API routes
- own local repository credentials
- use API-issued checkout metadata for the bounded job
- prepare a read-only ephemeral checkout or verify a mounted workspace
- resolve the actual `commit_sha`
- refresh or produce local baseline / overlay evidence where needed
- submit a bounded checkout receipt
- avoid logging or persisting credential material
- avoid raw source upload by default

The runner must not:

- import `apps/server/internal/*`
- write Postgres directly
- create or update canonical WorkItems directly
- assign or claim tasks
- execute arbitrary customer commands in H1
- create `Run`, receipt-for-execution, `Decision`, `GateDecision`, or `Proof`
- create branches, commits, pull requests, or merge requests

If H1 uses local Git commands for checkout, they must be routed through a narrow
allowlisted checkout helper. That helper may run Git checkout/fetch/status
commands needed for workspace preparation, but it must not expose arbitrary
command execution.

## Agent-facing next actions

After proposal acceptance currently returns:

```json
{
  "kind": "planned_workitems_ready",
  "available": false,
  "planned_slice": "H"
}
```

After H1 implementation, this can become a real preparation action, for example:

```json
{
  "kind": "prepare_checkout",
  "available": true,
  "blocking": true,
  "command": "goalrail work checkout prepare --task-id <task_id> --format json"
}
```

The exact CLI command name may be finalized in H1, but it must not imply code
execution. Text output must say checkout/workspace preparation only.

## Non-goals

This ADR does not define or implement:

- runner code
- server routes
- database migrations
- runner registration
- runner capability registry
- assignment
- claiming
- runtime registry
- primary runtime adapter
- arbitrary command execution
- `Run`
- execution receipt
- `Decision`
- `GateDecision`
- `Proof`
- provider OAuth
- provider clients
- VCS provider metadata listing
- server-side repository clone
- repository write access
- branch / commit / pull request creation
- raw source upload by default
- generic queue / outbox / worker registry
- persistent repository mirrors

## Consequences

### Positive

- Gives the post-planning loop a safe next implementation target.
- Keeps runner checkout separate from execution.
- Keeps API server canonical-state-only for repository access.
- Preserves runner-owned credential custody.
- Avoids silently upgrading `WorkItem(planned)` into execution.
- Gives future gate/proof work bounded checkout metadata to reference.

### Tradeoffs

- H1 still will not execute tasks.
- H1 still will not produce proof.
- A runner auth handshake must be explicit before public runner-facing routes
  become usable.
- Checkout receipt evidence is not enough for final acceptance; it is only
  workspace preparation evidence.

## Review triggers

Review this ADR before:

- introducing `Run`
- adding assignment or claiming
- executing commands through a runner
- uploading source bodies or artifacts beyond bounded metadata refs
- storing repository credentials server-side
- adding provider OAuth or provider clients
- using checkout receipts as proof inputs in gate decisions
