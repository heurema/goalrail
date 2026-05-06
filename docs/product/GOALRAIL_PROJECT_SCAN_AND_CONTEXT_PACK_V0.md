---
id: goalrail_project_scan_and_context_pack_v0
title: Goalrail Project Scan and Context Pack v0
kind: architecture_canon
authority: canonical
status: current
owner: architecture
truth_surfaces:
  - project_scan_v0
  - repository_baseline_profile
  - workspace_overlay
  - contract_context_pack_v0
lifecycle: active-core
review_after: 2026-07-19
supersedes: []
superseded_by: null
related_docs:
  - docs/product/GOALRAIL_PRODUCT_CONCEPT.md
  - docs/product/GOALRAIL_OPERATING_MODEL.md
  - docs/product/GOALRAIL_MVP_BLUEPRINT.md
  - docs/adr/ADR-0025-repository-baseline-profile-lifecycle.md
---
# Goalrail Project Scan and Context Pack v0

## Purpose

Project Scan v0 gives Goalrail a minimal, deterministic, local understanding of
a repository after it has been bound through `goalrail init`.

It exists to support later contract drafting, task shaping, execution packet
creation, verification expectations, and proof-oriented evidence. It is not a
code intelligence platform and not an audit verdict.

## Product framing

User-facing wording should stay simple:

> `goalrail init` connects the repository and builds a lightweight project map.
> Goalrail uses that map later to prepare bounded work and proof expectations.

Avoid calling this a full audit in v0. Use **Project Scan** or **project map**.

## v0 architecture

```text
goalrail init
  -> repo binding
  -> quick local Project Scan
  -> RepositoryBaselineProfile(summary/local receipts)
  -> optional best-effort background scan

future contract/task command
  -> freshness gate
  -> WorkspaceOverlay refresh
  -> ContractContextPack cut for this specific contract/task
```

## RepositoryBaselineProfile

`RepositoryBaselineProfile` is an immutable profile of committed repository
shape.

Identity:

```text
repo_binding_id + canonical_repo_root + HEAD_SHA + schema_version
```

Recommended fields:

```json
{
  "repository_baseline_profile_id": "...",
  "repo_binding_id": "...",
  "canonical_repo_root": "...",
  "head_sha": "...",
  "schema_version": 1,
  "status": "quick|complete|partial|stale|error",
  "scan_budget": {
    "elapsed_ms": 0,
    "file_limit": 0,
    "byte_limit": 0
  },
  "shape": {
    "workspaces": [],
    "toolchains": [],
    "package_managers": [],
    "entrypoint_candidates": []
  },
  "readiness_signals": {
    "tests": [],
    "ci": [],
    "agent_rules": [],
    "codeowners": [],
    "proof_surface": "none|partial|strong"
  },
  "partiality": {
    "sparse_checkout": false,
    "shallow_repository": false,
    "submodules_present": false,
    "truncated": false,
    "reasons": []
  },
  "local_artifacts": {
    "tracked_manifest_ref": "...",
    "repo_map_ref": "...",
    "signature_index_ref": "..."
  },
  "receipts": {
    "scanned": [],
    "skipped": [],
    "hashes": []
  }
}
```

The server may store summary and receipt metadata, but raw source bodies remain
local by default.

## WorkspaceOverlay

`WorkspaceOverlay` describes working-tree deviation from the committed
baseline.

It should be cheap to refresh and should not force a full baseline rebuild on
every edit.

Recommended fields:

```json
{
  "workspace_overlay_id": "...",
  "repository_baseline_profile_id": "...",
  "repo_binding_id": "...",
  "canonical_repo_root": "...",
  "base_head_sha": "...",
  "created_at": "...",
  "state": "clean|dirty|unmerged|partial|unknown",
  "changed_paths": [],
  "scan_critical_changed_paths": [],
  "unmerged_paths": [],
  "untracked_visibility": "not_checked|checked|partial",
  "ignored_visibility": "not_checked|checked|partial",
  "submodule_flags": [],
  "partiality_reasons": [],
  "raw_status_receipt_ref": "..."
}
```

If `scan_critical_changed_paths` is not empty, Goalrail should perform a quick
structural rescan before relying on the baseline for contract/task context.

## ContractContextPack

`ContractContextPack` is not created by Project Scan. It is created later for a
specific contract or task.

It must reference exact baseline and overlay versions.

Recommended fields:

```json
{
  "contract_context_pack_id": "...",
  "contract_id": "...",
  "task_id": "...",
  "repo_binding_id": "...",
  "repository_baseline_profile_id": "...",
  "workspace_overlay_id": "...",
  "head_sha": "...",
  "scope": {
    "paths_allowed": [],
    "paths_denied": []
  },
  "included": [
    {
      "kind": "full_file|snippet|signature_map|diff|test_evidence|rule|manifest",
      "path": "...",
      "lines": [1, 120],
      "hash": "...",
      "supports_clauses": []
    }
  ],
  "excluded": [
    {
      "path": "...",
      "reason": "out_of_scope|duplicate|too_large|generated|low_value|partial"
    }
  ],
  "clause_coverage": [
    {
      "clause_id": "...",
      "status": "covered|partial|unknown",
      "evidence_refs": []
    }
  ],
  "unknowns": [],
  "stale": false
}
```

## Freshness gates

Commands that depend on repository understanding must check:

1. Baseline exists.
2. Baseline schema matches current scanner schema.
3. Baseline HEAD matches current HEAD.
4. Overlay is current enough for the command.
5. No blocking partiality exists for the requested operation.

If a command only needs broad orientation, partial baseline may be acceptable
with warnings.

If a command affects execution, verification, or proof, stale or unmerged
context should block or force bounded refresh.

## v0 scan-critical files

A changed overlay path is scan-critical when it affects repository shape or
Goalrail operating expectations.

Initial list:

- `go.mod`, `go.work`, `package.json`, `Cargo.toml`, `pyproject.toml`,
  `requirements.txt`, `Gemfile`, `composer.json`
- lockfiles used for package-manager inference
- workspace files such as `pnpm-workspace.yaml`, root monorepo config, or future
  known workspace manifests
- `.github/workflows/*`
- `AGENTS.md`, `CLAUDE.md`, `.github/copilot-instructions.md`,
  `.cursor/rules/*`
- `CODEOWNERS`, `.github/CODEOWNERS`
- `.gitmodules`
- future Goalrail scan config files

## Edge-case posture

| Edge case | v0 posture |
| --- | --- |
| HEAD changed | full bounded baseline rebuild |
| dirty worktree | overlay refresh, not full rebuild by default |
| dirty scan-critical files | quick structural rescan |
| unmerged/conflicted paths | partial/blocking for contract execution/proof |
| linked worktree | key by canonical worktree root + HEAD |
| detached HEAD | supported; branch name is not identity |
| sparse checkout | mark partial; do not claim full baseline |
| submodules | record boundary/gitlink state; do not recurse by default |
| shallow repository | record partiality flag |
| large/binary/generated/vendor files | skip or bucket with explicit receipts |
| ignored files | follow standard exclusions for untracked enumeration; do not assume tracked files are ignored |
| background scan race | atomic write; freshness gate still required |

## Local vs server storage

Local:

- heavy tracked-file manifest
- path-level skip reasons
- raw Git status receipt
- repo map / signature artifacts
- parser/lexical indexes when introduced
- raw source refs and hashes

Server:

- baseline summary
- scan status
- detected workspaces/toolchains/package managers
- proof-surface summary
- partiality flags
- artifact hashes/IDs
- timestamps and schema versions

No raw source upload by default.

## Explicit non-goals for v0

- full audit scoring
- server clone
- server-side checks
- provider repository discovery
- always-on daemon
- file watchers
- mutable latest baseline
- recursive submodule indexing
- pretending sparse checkout is complete
- embeddings as source of truth
- LLM summaries as source of truth
- broad code intelligence platform
