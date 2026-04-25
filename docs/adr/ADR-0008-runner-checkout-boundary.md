# ADR-0008 — Runner and repository checkout boundary

Status: accepted
Date: 2026-04-26

## Context

Goalrail needs repository access for real delivery work, but repository access
must not collapse the product into a hosted CI system, an AI IDE, or a hidden
DevOps platform.

The current server boundary makes the Go server the future owner of canonical
Goalrail state. CLI, web resources, skills, and integrations remain adapters or
helper surfaces. The server already has in-memory source-neutral intake, Goal
promotion, and Goal readiness prototypes, but no durable storage, real
RepoBinding sync, repository clone/readiness, contract composition, gate, proof,
or workers.

Repository access will eventually be needed for:

- repository catalog and binding checks
- baseline snapshots
- scoped code inspection
- test / lint / build checks
- frozen verification bundles
- execution receipts
- proof evidence

That does not mean the primary API server should clone repositories or execute
checks directly.

Goalrail must support teams that cannot or should not send repository contents to
Goalrail-hosted infrastructure. Some customers will need all code access and
execution to happen inside their own infrastructure while Goalrail receives only
bounded receipts, artifacts, and proof inputs.

## Decision

Repository checkout, workspace preparation, code inspection, and check execution
belong behind a dedicated runner boundary.

The Goalrail API server owns canonical state, scheduling decisions, task packets,
run records, event append, and proof input references. It must not directly clone
repositories, run tests, mutate workspaces, or execute runtime commands inside
the main API process.

A `Runner` is an execution-side component that can prepare an isolated workspace,
obtain or receive repository access, run bounded checks or runtime commands, and
return receipts plus artifacts to the server.

Goalrail supports two runner deployment modes:

1. `goalrail_hosted_runner`
   - runner infrastructure operated by Goalrail
   - suitable for early managed pilots and low/safe repository policies
   - may use provider-issued short-lived clone credentials

2. `customer_hosted_runner`
   - runner deployed in the customer's infrastructure
   - suitable for security-sensitive teams, private networks, self-managed VCS,
     local-only policies, or customers that do not allow repository contents to
     leave their environment
   - repository credentials and clone access may remain entirely customer-owned

The runner boundary is part of the Delivery Runtime layer, not the Intent Plane,
not the Gate, and not the API server.

Provider-side repository discovery is not required for `customer_hosted_runner`
mode. Goalrail may operate on a repository through a customer-hosted runner even
when Goalrail cloud has no VCS provider connection, clone credential, or ability
to see repository contents.

## Boundary rules

1. The API server does not clone repositories in-process.
2. The API server does not run tests, builds, linters, or runtime commands in-process.
3. The API server may issue or broker short-lived checkout instructions for a runner.
4. A runner may clone, inspect, and check a repository only for a bounded task or
   verification bundle.
5. A runner must return a receipt and artifact references; it must not become the
   canonical source of truth.
6. Gate reads frozen receipts, baseline snapshots, and artifacts; gate must not
   trust a mutable live workspace as final proof.
7. Customer-hosted runners are first-class, not an enterprise-only afterthought.
8. Persistent full-repository mirrors are out of scope for the MVP unless a later
   ADR explicitly authorizes them.
9. Repository write access is out of scope for the first runner boundary.
10. Provider-specific credential handling must stay behind provider and runner
    adapters, not inside the kernel.
11. VCS provider discovery, repository binding, and checkout permission are
    separate concerns.
12. `VcsConnection` is not a checkout credential.
13. `RepositoryRecord` is not repository access.
14. `RepoBinding` identifies which repository a Project works with; it does not
    itself authorize checkout.

Shorthand: `VcsConnection != CheckoutCredential`, `RepositoryRecord !=
RepoAccess`, and `RepoBinding != permission to clone`.

## Repository source modes

Future implementation may model repository source using conceptual source modes.
These names are conceptual for this ADR and are not required for the next code
slice.

### `provider_discovered`

The repository was found through a VCS adapter such as GitHub, GitLab, Bitbucket,
or a custom Git provider. For example, a GitHub App installation may sync
selected repository metadata into Goalrail.

### `manual_declared`

An owner or admin manually declares a repository record for a Goalrail Project.
Goalrail cloud may have no provider-side access. This path is useful for
self-managed Git, private networks, early pilots, and security-sensitive setups.

### `runner_reported`

A customer-hosted runner reports bounded metadata about a local workspace or
repository. Goalrail receives minimal metadata, receipt references, and artifact
references, not unrestricted repository contents.

## Checkout access modes

Checkout authority is determined by runner mode, policy, access mode, and the
checkout instruction for a bounded run. `VcsConnection`, `RepositoryRecord`, and
`RepoBinding` may support discovery, catalog, and project mapping, but they do
not by themselves grant clone permission.

Future implementation may model checkout access using conceptual modes:

### `provider_token_checkout`

A runner receives a short-lived provider credential. For example, a
Goalrail-hosted GitHub flow may use a GitHub App installation token scoped to the
selected repository and permissions. This is valid only when organization policy
allows Goalrail-hosted checkout.

### `customer_runner_checkout`

A customer-hosted runner has repository access inside customer infrastructure.
Goalrail cloud is not required to have a VCS provider connection or clone
credential.

### `customer_mounted_workspace`

A customer-hosted runner receives an already prepared workspace. For example, CI
may perform checkout before the runner collects receipt and check results.

### `metadata_only`

The repository is used only as catalog or binding metadata. No checkout happens.

## Runner responsibilities

A runner may perform these bounded responsibilities:

- receive a task packet or verification bundle reference
- resolve the selected `RepoBinding` through server-issued instructions
- prepare an isolated workspace
- clone or mount the repository according to policy
- apply path scope or sparse checkout where supported
- capture baseline metadata
- run allowed read/check commands
- capture command output and structured check results
- produce a receipt
- upload or return artifact references
- clean up workspace and temporary credentials

A runner must not decide final acceptance. Final verdict remains gate-owned.

## Checkout modes

### Mode A — No checkout

Used for repository catalog, provider metadata, access validation, branch listing,
and other integration-level operations that do not need a working tree.

### Mode B — Ephemeral checkout

Used when Goalrail-hosted or customer-hosted runners need a temporary working
copy for checks, baseline capture, or verification bundle preparation.

The workspace should be isolated and deleted after the run unless a later policy
explicitly keeps selected artifacts.

### Mode C — Customer-mounted workspace

Used when the runner runs inside customer infrastructure and the customer chooses
to mount, clone, or otherwise provide the repository locally.

Goalrail receives receipts and artifacts, not unrestricted repository contents.

### Mode D — Persistent mirror

Deferred. A persistent mirror may improve performance later, but it introduces
multi-tenant storage, revocation, deletion, and stale-state risks. It is not part
of the MVP runner boundary.

## Runner deployment records

The eventual canonical model may include:

| Object | Purpose | Authoritative writer |
| --- | --- | --- |
| `RunnerRegistration` | registered runner identity and mode | server / setup flow |
| `RunnerCapability` | supported checkout, runtime, isolation, and check capabilities | runner / server |
| `RunnerAssignment` | selected runner for a bounded run or verification task | scheduler / policy engine |
| `WorkspaceSnapshot` | immutable reference to checkout metadata and baseline inputs | runner |
| `RunReceipt` | execution/check evidence returned by the runner | runner |

These names are conceptual for this ADR. They are not a required implementation
schema for the next code slice.

## Policy posture

Runner selection is policy-controlled:

- low-risk work may use a Goalrail-hosted runner when the organization allows it
- medium/high-risk work may require stronger isolation or customer-hosted runners
- security-sensitive work may require `customer_hosted_only`, `local_only`, or
  `single_vendor_only`
- policy may forbid repository contents from leaving customer infrastructure
- policy may require human signoff before any checkout or command execution

Risk controls review and execution posture. It does not authorize hidden
unbounded repository access.

## Credential posture

For Goalrail-hosted GitHub access, the preferred future path is GitHub App
installation credentials with short-lived installation tokens scoped to the
selected repository and permissions.

For customer-hosted runners, repository access may be entirely customer-owned:

- local Git credentials
- CI-provided credentials
- mounted checkout
- self-managed Git server credentials
- provider tokens stored in customer infrastructure

Goalrail should avoid storing long-lived repository credentials whenever a
short-lived token, customer-hosted credential, or delegated checkout can satisfy
the job.

## Artifact minimization

A customer-hosted runner should not automatically upload unrestricted repository
contents to Goalrail. Future policy may limit returned artifacts to metadata
only, redacted logs, selected check outputs, selected files, patch or diff
summaries, or a full verification bundle only when explicitly allowed.

## Receipt trust and minimum evidence

Runner receipts and artifacts are evidence inputs for Gate. Gate must understand
the source and trust posture of that evidence before writing a final decision.

A minimum conceptual checkout or check receipt may include:

- `runner_id`
- `runner_mode`
- `job_id`
- `commit_sha`
- `workspace_ref`
- `artifact_hashes`
- `receipt_created_at`
- optional later: `receipt_signature`

Gate should be able to see who performed the check, where it ran, which commit it
covered, and which artifact hashes are attached. A runner still does not write
the final verdict. Signed receipts and stronger attestation can come later and
are not required for the next code slice.

## Revocation posture

If repository access is revoked, new checkout or run assignment should be blocked
or require reconnect / reapproval. Existing immutable proof artifacts remain
historical evidence. Future repository and runner state should be able to reflect
`revoked` or `needs_reconnect` status without mutating past proof.

## Events

Possible future events:

- `runner.registered`
- `runner.capabilities_reported`
- `runner.assigned`
- `workspace.prepared`
- `workspace.snapshot_recorded`
- `run.receipt_submitted`
- `workspace.cleaned_up`
- `runner.failed`

Events must reference canonical server objects and artifacts. Runner events do
not replace `Run`, `Decision`, or `Proof`.

## Rejected alternatives

### API server clones repositories directly

Rejected. It mixes canonical state ownership with execution-side workspace
management, increases blast radius, and makes customer-hosted execution harder.

### Goalrail requires all repositories to be cloned into Goalrail cloud

Rejected. This blocks security-sensitive customers and conflicts with the idea
that execution setup can remain user-owned or runtime-owned.

### Customer-hosted runners are deferred until enterprise later

Rejected. The model must support this from the beginning, even if the first code
slice is only a skeleton. Otherwise early abstractions will bias toward hosted
repository access.

### Persistent repository mirrors are the default

Rejected for MVP. Mirrors create storage, revocation, deletion, and stale-state
obligations before Goalrail has durable state and policy enforcement.

### Runner writes final verdict

Rejected. Runners produce receipts and artifacts. Gate writes the final decision.

## Non-goals

This ADR does not implement or define final details for:

- GitHub, GitLab, Bitbucket, or custom Git connector implementation
- organization/user/VCS connection schema
- `RepositoryRecord.source_kind`
- `RepoBinding.access_mode`
- durable storage
- job queue
- runner authentication protocol
- runner binary packaging
- hosted runner infrastructure
- customer-hosted runner installer
- repository write access
- branch creation
- commit creation
- pull request creation
- persistent mirrors
- gate or proof implementation

## Implementation implications

The next bounded work should document Organization, User, Membership,
`VcsConnection`, `RepositoryRecord`, `RepositoryRecord.source_kind`,
`RepoBinding`, and `RepoBinding.access_mode` boundaries before building real
GitHub integration. GitHub can be the first implementation target without making
GitHub App concepts part of the core domain model. GitLab, Bitbucket,
self-managed Git, and custom Git should remain representable later.

A later runner prototype should be its own slice and should start with the
smallest safe behavior:

- server records a conceptual runner mode
- server creates a bounded checkout request
- runner performs an ephemeral read-only checkout or accepts a customer-mounted
  workspace
- runner returns a deterministic checkout receipt with minimum evidence fields
- no repository writes
- no gate decision
- no proof generation
- no persistent mirror

## Open questions

1. Should the first runner prototype be Goalrail-hosted, customer-hosted, or both
   with a mock transport?
2. What is the minimum runner authentication handshake?
3. Should runner assignment be stored as a canonical object before durable
   storage exists?
4. Which artifact shape should represent a checkout receipt?
5. How should customer-hosted runners upload artifacts without exposing full
   repository contents?
6. Should runner policy live with organization policy, project policy, or
   RepoBinding policy first?
7. Should manually declared repositories require a separate approval object later,
   or can `RepoBinding.access_mode` and policy cover v0?
