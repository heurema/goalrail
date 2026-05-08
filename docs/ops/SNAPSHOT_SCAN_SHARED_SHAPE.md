---
id: goalrail_snapshot_scan_shared_shape
title: Snapshot and Project Scan Shared Shape Direction
kind: ops_status
authority: operational
status: current
owner: ops
truth_surfaces:
  - repository_context_snapshot
  - project_scan_shared_shape
  - init_repository_shape_direction
lifecycle: active-core
review_after: 2026-08-08
supersedes: []
superseded_by: null
related_docs:
  - docs/INDEX.md
  - docs/product/GOALRAIL_MVP_BLUEPRINT.md
  - docs/product/GOALRAIL_PROJECT_SCAN_AND_CONTEXT_PACK_V0.md
  - docs/ops/INIT_LIFECYCLE.md
  - docs/ops/COMPONENTS.yaml
---
# Snapshot and Project Scan Shared Shape Direction

## Purpose

This note defines the future shared repository-shape direction between the
server-advisory repository context snapshot and the local Project Scan baseline.

It is an operational design note only. It does not add implementation, change
schemas, change server APIs, persist Project Scan artifacts on the server, or
expand Goalrail into a broad code intelligence platform.

## Current State

`goalrail init` currently has two repository-shape paths:

- `buildRepositoryContextSnapshot` / `collectRepositoryInventory` builds a
  metadata-only repository context snapshot request for the server.
- `projectscan.BuildBaseline` builds a local `RepositoryBaselineProfile` for
  committed repository state.

The snapshot is a server-friendly advisory summary. It carries bounded metadata
such as repository identity, HEAD SHA, detected paths, detected toolchains,
detected package managers, and workspace candidates. It is not a source index,
readiness verdict, audit score, or server-side baseline profile.

Project Scan is local-first evidence. It builds immutable committed-state
baseline data, writes local cache artifacts, refreshes workspace overlay state,
and records receipts, hashes, skipped paths, partiality, and readiness signal
details. It does not call the server from `goalrail project scan/status`.

The two paths still use separate heuristics. INIT-05 added parity guardrails for
the shared shape signals, but it intentionally did not refactor the collection
model.

## Shared-shape Goal

Future implementation should reduce drift between snapshot inventory and Project
Scan baseline shape without changing their trust boundaries.

The desired direction is:

- preserve Project Scan as local-first repository evidence;
- preserve the snapshot as advisory server metadata;
- avoid raw source upload by default;
- keep schemas stable until there is a concrete schema need;
- make shared repository-shape signal definitions explicit enough that later
  implementation cannot silently widen MVP scope.

## Shared Signals

The following fields and signals should converge on one shared detector table,
manifest list, or small metadata collector:

- HEAD SHA or equivalent provenance reference;
- detected toolchains;
- detected package managers;
- workspace candidates / workspaces;
- CI workflow signals;
- readiness-relevant lightweight metadata-only signals when they are already
  local, deterministic, and safe to summarize.

Shared detection means the signal names and marker-path rules should drift less
over time. It does not mean the snapshot and baseline should expose the same
schema or durability model.

## Snapshot-only Shape

These remain snapshot-only:

- server-friendly summary shape;
- metadata-only snapshot request format;
- snapshot source, schema version, fingerprint, idempotency, and event metadata;
- detected-path summary that is useful for repository-context init.

The snapshot stays bounded init-time metadata. It must not become a source
archive, server scan, server clone, readiness score, Project Scan replacement,
or ContractContextPack.

## Baseline-only Shape

These remain Project Scan baseline-only:

- receipts;
- skipped paths;
- hashes;
- partiality;
- readiness signal details;
- local artifacts;
- workspace overlay and freshness state.

The baseline remains local committed-state evidence. It can carry path-level
receipt and freshness detail that the snapshot should not upload by default.

## Accepted Divergence

`snapshot.DetectedPaths` is not equivalent to
`RepositoryBaselineProfile.Receipts.Scanned`.

Accepted differences:

- snapshot detected paths may include directory markers such as `.github/workflows/`;
- snapshot detected paths may include bounded workspace directory candidates;
- baseline receipts are committed Git file paths scanned from tracked state;
- baseline receipts may exclude skipped or budget-limited paths and record those
  exclusions separately.

Parity tests should guard shared signals such as toolchains, package managers,
workspace/workflow markers, and other explicitly shared metadata signals. They
should not assert that snapshot detected paths and baseline scanned receipts are
the same path model.

## Trust Boundary

The shared-shape direction keeps the current MVP trust boundary:

- snapshot remains client-advisory metadata;
- baseline remains local evidence;
- the API server does not clone repositories;
- raw source bodies are not uploaded by default;
- embeddings and LLM summaries are not sources of truth;
- RepoBinding remains repository context, not checkout permission;
- runner, gate, proof, checkout, provider integration, and runtime execution
  behavior remain outside this note.

## Implementation Direction

The first implementation should be small:

- extract a shared metadata detector table or small shared collector for marker
  paths and signal names;
- start with toolchains, package managers, workspace manifests, and workflow
  paths;
- keep snapshot and Project Scan schemas stable at first;
- keep request/response APIs unchanged;
- keep Project Scan artifact schemas unchanged;
- use the INIT-05 parity tests as guardrails while moving detection rules.

Do not introduce a broad scanner framework, generic code-intelligence platform,
server Project Scan persistence, source indexing service, or provider-specific
execution doctrine.

## Recommended First Coding Slice

The next safe implementation slice is:

> Extract a small shared detector/table for toolchains, package managers,
> workspace manifests, and workflow paths, then route snapshot inventory and
> Project Scan baseline detection through it without changing schemas.

The slice should explicitly exclude:

- server API changes;
- database changes;
- Project Scan artifact schema changes;
- snapshot schema changes;
- server snapshot sync from Project Scan;
- full snapshot/scan unification;
- repair commands;
- runner, gate, proof, checkout, provider, or execution behavior.
