# ADR-0025 — Repository baseline profile lifecycle

Status: accepted
Date: 2026-05-06

## Context

Goalrail needs local repository understanding to support contract-first
delivery: project setup, contract drafting, task shaping, bounded execution
packets, verification expectations, and proof-oriented evidence.

The API server remains the canonical state machine, but it must not clone
repositories, store repository secrets, or run repository checks in-process.
Repository access is local CLI / runner owned.

`goalrail init` already binds local Git metadata to server-side Project /
RepoBinding context. The next repository-understanding step must remain
minimal, local, deterministic, and privacy-preserving. It must help future
`ContractContextPack` construction without becoming a hidden memory system or
broad code-indexing platform.

Pressure testing found three reliability risks:

1. A repository map can become stale through HEAD changes, linked worktrees,
   detached HEAD, sparse-checkout visibility, submodule movement, or scan schema
   changes.
2. Dirty working-tree state should not force a deep baseline rebuild on every
   edit.
3. Baseline must remain orientation and freshness evidence; task execution
   context must remain contract-specific.

## Decision

Goalrail will model repository understanding as:

1. `RepositoryBaselineProfile` - immutable local profile of committed repository
   shape.
2. `WorkspaceOverlay` - separate local receipt of working-tree deviation and
   partiality.
3. `ContractContextPack` - future task-specific context cut that references
   exact baseline and overlay versions.

### RepositoryBaselineProfile

A `RepositoryBaselineProfile` is immutable once written. It describes committed
repository shape only.

Its identity is:

```text
repo_binding_id + canonical_repo_root + HEAD_SHA + schema_version
```

It may include summary and derived local artifacts such as:

- detected manifests
- workspace candidates
- toolchains
- package managers
- CI and test signals
- rule / agent instruction files
- CODEOWNERS evidence
- entrypoint candidates
- skipped / partial / truncated scan reasons
- lightweight repo map or signatures when available

It must not be treated as source truth for code behavior.

### WorkspaceOverlay

Dirty, unmerged, partial, ignored/untracked visibility, submodule,
sparse-checkout, shallow clone, and worktree-specific state are represented
separately as a `WorkspaceOverlay` receipt.

Dirty state is freshness-relevant, but it does not by itself mint a new deep
baseline version.

The overlay may be refreshed cheaply when a command needs repository
understanding. It should be based on stable, machine-readable local Git state
where available, plus explicit partiality flags when the state cannot be fully
observed.

### Rebuild and rescan rules

Goalrail rebuilds or refreshes repository understanding according to these
rules:

- Run a quick local Project Scan during `goalrail init`.
- Rebuild the baseline when missing, corrupt, HEAD mismatches, scanner schema
  mismatches, or the user explicitly reruns scan.
- Do not rebuild the full baseline merely because the working tree is dirty.
- Refresh `WorkspaceOverlay` when a command needs current repository context.
- If the overlay touches scan-critical files, perform a quick structural rescan
  and mark the previous baseline context as structurally stale until refreshed.

Scan-critical files include, at minimum:

- root and workspace manifests
- workspace configuration
- lockfiles where they affect workspace/package-manager inference
- CI configuration
- rule / agent instruction files
- CODEOWNERS
- `.gitmodules`
- files that define project scan behavior once such configuration exists

### Freshness gate

Any command that depends on repository understanding must validate baseline
freshness and overlay freshness before use.

This includes future contract drafting, task shaping, execution packet creation,
run submission, verification, and proof production.

Background scans are best-effort convenience only. They must publish results
atomically and never bypass synchronous freshness checks.

### Partiality

Goalrail must record partial states explicitly rather than hiding them behind a
ready verdict.

At v0, these states are partial or out-of-scope unless a later ADR changes the
boundary:

- sparse checkout pretending to represent the full repo
- unmerged/conflicted working tree
- submodule interiors
- shallow clone limitations
- scan budget truncation
- skipped large/binary/generated/vendor paths
- unsupported or unreadable filesystem paths

### Server boundary

The server may store summary and receipts linked to `RepoBinding`.

By default, the server does not receive raw source code bodies, clone
repositories, run checks, or maintain a repository index. Heavy artifacts and
derived indexes remain local CLI / runner artifacts unless a later explicit
artifact/proof boundary permits upload.

### ContractContextPack boundary

A `ContractContextPack` is task-specific. It must reference the exact
`RepositoryBaselineProfile` and `WorkspaceOverlay` versions from which it was
cut.

It must not become reusable hidden project memory. It should include clause
coverage, included/excluded evidence, raw refs, and explicit unknowns for a
specific contract or task.

## Consequences

### Positive

- Avoids stale repository maps becoming hidden truth.
- Avoids deep baseline churn on every dirty edit.
- Keeps `goalrail init` lightweight.
- Keeps server privacy and no-clone boundaries intact.
- Gives future contract/task context a deterministic freshness base.
- Supports monorepos and large repos through explicit partiality rather than
  fake completeness.

### Negative / tradeoffs

- A dirty workspace may require overlay refresh before commands that need
  context.
- Some edge cases are marked partial rather than fully supported in v0.
- Full precise navigation and semantic retrieval remain deferred.
- Users may sometimes see a rescan/freshness message before contract/task work.

## Explicit non-goals

This ADR does not add:

- server-side repository clone
- provider OAuth or provider repository discovery
- runner checkout implementation
- file watcher or always-on daemon
- mutable latest baseline
- recursive submodule scan
- sparse-checkout full-baseline claim
- embeddings or LLM summaries as source of truth
- raw source upload by default
- gate/proof implementation
- broad indexing platform

## Review trigger

Review this ADR after `Project Scan v0` and `ContractContextPack v0` have
produced working fixtures, or earlier if runner checkout, proof artifacts, or
server-side artifact upload boundaries are introduced.
