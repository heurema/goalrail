---
id: goalrail_init_lifecycle
title: Goalrail Init Lifecycle
kind: ops_status
authority: operational
status: current
owner: ops
truth_surfaces:
  - init_lifecycle
  - init_recovery_semantics
  - repository_context_bootstrap
lifecycle: active-core
review_after: 2026-08-07
supersedes: []
superseded_by: null
related_docs:
  - docs/INDEX.md
  - docs/product/GOALRAIL_MVP_BLUEPRINT.md
  - docs/product/GOALRAIL_PROJECT_SCAN_AND_CONTEXT_PACK_V0.md
  - docs/adr/ADR-0003-go-cli-layout.md
  - docs/adr/ADR-0025-repository-baseline-profile-lifecycle.md
  - docs/ops/STATUS.md
  - docs/ops/COMPONENTS.yaml
---
# Goalrail Init Lifecycle

## Purpose

This note records the current `goalrail init` bootstrap lifecycle and the MVP
recovery direction for partial failures.

It is operational documentation only. It does not broaden the Goalrail MVP or
claim runner, gate, proof, checkout, provider app integration, server clone,
source upload, runtime execution, analytics, CRM, or a broad backend platform.

## Current command modes

### `goalrail init --local-demo`

`--local-demo` is the old auth-free demo path.

It:
- reads local Git metadata when available
- emits a `RepoBindingDraft`
- writes no files
- performs no auth
- makes no server call
- records no repository context snapshot
- runs no Project Scan
- installs no Agent Pack

Its status is a local/demo draft only, not a server `RepoBinding`.

### `goalrail init`

Plain `goalrail init` is the default server-backed repository-context
bootstrap.

Current implementation sequence:

1. Discover local Git root, origin URL, provider/repository identity, HEAD, and
   workflow base branch.
2. Load the stored `goalrail login` profile and fail locally when the access
   token is expired.
3. Preflight an existing Git-root `.goalrail/project.yml` marker for known
   server/repository/base conflicts.
4. Call authenticated `POST /v1/init/repository-context`.
5. The server creates or reuses one repo-backed Project and creates or reuses
   one metadata-only RepoBinding inside the authenticated user's existing
   Organization.
6. Write the non-secret Git-root `.goalrail/project.yml` marker.
7. Ensure `.goalrail/.gitignore` protects Goalrail-owned machine-local state.
8. Build a bounded repository inventory locally and post it to
   `POST /v1/repo-bindings/{repo_binding_id}/context-snapshots`.
9. Run a best-effort local Project Scan cache write for the committed HEAD plus
   current workspace overlay.

Snapshot recording is advisory post-marker work. If snapshot recording fails
after binding and marker write, the command reports `success_with_warnings`
rather than leaving the repository without a local marker.

### `goalrail init --project <project_id>`

`--project` keeps the lower-level Project-scoped RepoBinding init path.

Current implementation sequence:

1. Discover local Git metadata.
2. Load the stored `goalrail login` profile.
3. Preflight an existing Git-root `.goalrail/project.yml` marker for the
   requested Project and repository identity.
4. Call authenticated
   `POST /v1/projects/{project_id}/repo-bindings/init`.
5. The server validates Project access and creates or reuses one metadata-only
   RepoBinding for that Project.
6. Write `.goalrail/project.yml`.
7. Ensure `.goalrail/.gitignore`.
8. Run a best-effort local Project Scan cache write.

This mode does not currently record a repository context snapshot.

## Init result output

Server-backed init JSON includes a stable top-level `status` and compact
`steps` array in addition to the existing top-level fields.

Current status values are:
- `success`
- `success_with_warnings`
- `partial_failed`
- `failed` when a successful output object can safely represent a fatal state;
  normal pre-binding command failures may still exit with no output object

Current step names are:
- `repository_context` for default `goalrail init`
- `repo_binding` for `goalrail init --project <project_id>`
- `local_marker`
- `local_gitignore`
- `context_snapshot` for default `goalrail init`
- `project_scan`

Step status values are `ok`, `skipped`, `warning`, and `error`. Recoverable
warning/error steps may include a `retry_command`.

## Local files

### `.goalrail/project.yml`

The marker is committed repository identity, not credentials.

It stores:
- `server_url`
- `organization_id`
- `project_id`
- `repo_binding_id`
- repository provider, full name, URL, and workflow base branch

It does not store access tokens, refresh tokens, provider tokens, deploy keys,
sessions, source content, scan artifacts, runner state, gate decisions, or
proof.

Existing marker handling is conservative:
- matching content is verified and reused
- known preflight conflicts stop before the server call
- different post-response content is not overwritten
- unparseable content fails with a repair-oriented message

### `.goalrail/.gitignore`

Init writes or updates `.goalrail/.gitignore`, not the root `.gitignore`.

It ignores Goalrail-owned local machine state:
- `/local/`
- `/cache/`
- `/state/`
- `/tmp/`
- `*.local.yml`
- `*.local.toml`
- `*.local.json`

This keeps committed marker/agent files separate from local cache, state, and
machine-specific overrides.

## Binding, marker, snapshot, and scan

| Surface | Owner | Durability | Role |
| --- | --- | --- | --- |
| RepoBinding | Server | Canonical metadata | Binds Project to repository identity and workflow base branch. |
| `.goalrail/project.yml` | Repo local | Committed marker | Lets CLI commands prove they are operating in the expected Project/RepoBinding context. |
| Repository context snapshot | Server | Advisory metadata | Stores a bounded init-time inventory fingerprint and summary fields. |
| Project Scan | Local CLI cache | Local artifact | Builds `RepositoryBaselineProfile` and `WorkspaceOverlay` for committed HEAD and workspace freshness. |

These are related but not interchangeable. A server RepoBinding is not checkout
permission. A marker is not server truth. A snapshot is not a source index. A
Project Scan is local repository evidence, not an audit verdict.

## Snapshot semantics

The repository context snapshot is metadata-only.

Current CLI inventory includes bounded signals such as:
- detected known paths and manifests
- immediate workspace candidates under conventional roots such as `apps/`,
  `packages/`, and `services/`
- detected toolchains
- detected package managers
- Git remote name and local HEAD SHA

The server validates that the snapshot matches the active RepoBinding provider,
repository full name, URL, and workflow base branch. It stores an idempotent
fingerprint and appends a snapshot event when a new snapshot is recorded.

The snapshot is advisory metadata. It is not a server-side baseline profile and
does not upload raw source bodies by default.

## Project Scan semantics

Project Scan remains local-first.

Real current local surfaces:
- `goalrail project scan`
- `goalrail project status`

`goalrail project scan` builds or refreshes a local immutable
`RepositoryBaselineProfile` for the current committed HEAD and refreshes a
`WorkspaceOverlay`. It requires `.goalrail/project.yml`, writes local cache
artifacts, and does not call the server.

`goalrail project status` refreshes the cheap overlay and reports freshness. It
does not rebuild the baseline by default and does not call the server.

There is no MVP `--require-scan` flag for `goalrail init`. Strict/CI behavior
should not be added unless a later strict use case is explicitly accepted.

## Trust boundary

MVP repository init keeps this boundary:

- snapshot is advisory metadata
- Project Scan remains local-first
- no server clone
- no raw source upload by default
- no server-side repository checks
- no checkout permission from RepoBinding
- no provider app integration
- no runner
- no gate
- no proof generation
- no runtime execution

RepoBinding stores repository context. Checkout authority remains outside
RepoBinding and belongs to future runner-owned local credential boundaries.

## Partial failure direction

Current CLI behavior is mostly fail-fast around server and marker work, with
Project Scan already represented as a warning-style result in init output.
The MVP recovery direction should make bootstrap state explicit without
pretending advisory work is canonical.

| Failure point | Current behavior | MVP direction |
| --- | --- | --- |
| Server binding / repository-context init fails | Command fails before marker, snapshot, or scan. | `failed`; no local marker should be written. |
| Marker write fails after server success | Command emits `partial_failed` output when possible, then exits with the local write error. | `partial_failed`; recoverable local bootstrap failure. A repair/status command should guide marker rewrite or reconciliation. |
| `.goalrail/.gitignore` write fails after marker | Command emits `partial_failed` output when possible, then exits with the local write error. | Treat with the marker local-bootstrap family; prefer recoverable local repair without changing server binding. |
| Snapshot build/post fails after binding and marker | Command succeeds with `success_with_warnings`. | `success_with_warnings`; binding and marker are sufficient, snapshot is advisory and can be retried. |
| Project Scan cache write fails after binding and marker | Command succeeds with `project_scan_status=error`, `success_with_warnings`, and a recoverable `project_scan` step. | `success_with_warnings`; local scan can be retried with `goalrail project scan`. |

The desired user-visible distinction is:
- `failed`: canonical binding did not happen or cannot be trusted
- `partial_failed`: server binding succeeded, but local bootstrap is incomplete
- `success_with_warnings`: binding and marker succeeded, but advisory snapshot
  or local scan evidence is missing or stale

The current CLI emits this stable status enum for server-backed init JSON
outputs. Binding and preflight failures can still exit before an output object
exists.

## Non-goals

Init does not:

- create Organizations
- infer Goalrail Organizations from Git providers
- create provider connections
- provision deploy keys
- create branches
- install hooks
- run readiness/audit checks
- create a `ContractContextPack`
- start workers
- start runner checkout
- start execution
- write gate decisions
- generate or retrieve proof
