## Why

PR #3 exposed three integrity gaps before real canary activation: concurrent processes can corrupt the append-only evidence chain, a terminal identity conflict can avoid the immediate wrong-join stop, and wording-only amendments can rewrite provenance. This slice closes those gaps while the canary remains blocked.

## What Changes

- Serialize each evidence read/validate/append transaction across cooperating Goalrail processes and make concurrent reads observe only complete transactions.
- Project terminal root-session conflicts as wrong joins so the existing one-conflict hard stop is enforced.
- Reject wording-only amendments that replace source evidence or change evidence references while preserving valid statement-only edits and stable-item reordering.
- Add deterministic regression tests for all three findings and retain the existing provider-neutral contracts.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Cross-process evidence transaction serialization | OUT-1, OUT-4, SIG-1, SIG-4 | NG-1, NG-2, NG-3 |
| Conflict-to-wrong-join projection | OUT-2, OUT-4, SIG-2, SIG-4 | NG-1, NG-3 |
| Wording-only provenance preservation | OUT-3, OUT-4, SIG-3, SIG-4 | NG-1, NG-3 |

## Capabilities

### New Capabilities

- `canary-integrity-hardening`: Defines cross-process evidence transaction safety, immediate wrong-join projection, and immutable provenance for wording-only intent amendments.

### Modified Capabilities

None. The repository has no archived main capability specs yet; this bounded hardening contract remains independently traceable until the earlier foundation change is synchronized or archived.

## Impact

- Affects the repository-local Go evidence adapter, operator report projection, canonical intent amendment validation, and their tests.
- Adds a small platform adapter for evidence-file locking on the currently supported local execution platforms.
- Adds no database, workflow engine, service, dashboard, credentials, or third-party dependency.
- Does not change public APIs beyond rejecting previously unsafe amendment inputs and serializing cooperating file-store operations.

## Non-Goals

- No real canary activation, OpenSpec archive, deployment, credential change, or external infrastructure change.
- No replacement of the JSONL evidence store and no unrelated storage or workflow redesign.
- No relaxation of append-only evidence, automatic lineage, bounded resolution, or hard-stop rules.
