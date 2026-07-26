## Context

Intent Snapshot `intent-verify-intent-context-binding-v0` version 2 supersedes version 1. Version 1 declared the change specification-only and listed implementation as a non-goal, while the change ships the preparation-time verification itself; version 1 is retained unmodified as `intent-v1.md`.

The behaviour lives in `internal/adapters/openspec/work_spec_intent.go`. `IntentResolver.Verify` reads the referenced intent artifact, confines it to the canonical repository root, applies a bounded size limit, compares its digest, and then resolves the Context Pack of the change. When the pack resolves, it is read under the same bound, parsed, and the intent is parsed against it before flow validation.

Review of the first attempt confirmed four further defects, each reproduced before being fixed: the missing-context path applied no schema-aware rule; the requirement stated verification before freezing, which `Service.Prepare` does not do; `os.Open` on a FIFO blocks indefinitely; and two Context Pack rows packed two evidence references into one cell.

## Goals / Non-Goals

**Goals:**

- Verify the confirmed intent against the Context Pack of its change during preparation.
- Require that pack exactly when the change loader would, so the no-pack path cannot become a bypass.
- Reject non-regular artifacts before opening them.
- State the enforcement in one promoted requirement at the boundary that holds.
- Keep this change's own Context Pack valid under the verification it introduces.

**Non-Goals:**

- No change to canonical schemas, command behaviour, or error contracts.
- No reordering of preparation to make an aspirational boundary true.
- No restatement of intent-level Context Pack semantics owned by other capabilities.
- No workflow engine, authorization system, effect gateway, credential broker, sandbox platform, database, or dashboard.

## Decisions

### 1. Modify the preparation requirement instead of adding a new one

Preparation-time enforcement is a property of preparation, not a separate capability. Attaching it to `Run preparation is inspectable and does not launch` keeps the accepted path and its failure modes in one place and avoids a second requirement restating the same boundary.

### 2. State the boundary that holds, and do not reorder preparation

`Service.Prepare` freezes the WorkSpec before invoking the verifier, and persists prepared state only after both succeed. Reordering to satisfy the earlier wording would change behaviour for a documentation defect; stating the boundary as "before any prepared state is persisted" is accurate, is the property that matters, and leaves the implementation untouched.

### 3. Reuse the change loader's requirement rule rather than restating it

`changeRequiresContext` already encodes the distinction: a current change under the project intent schema always requires a pack, an archived change requires one only when its intent declares it, and another schema does not. Preparation calls that same function, so the two paths cannot drift. The intent's own declaration remains an additional trigger for changes the loader does not cover.

### 4. Guard both bounded reads, not only the reported one

The FIFO finding named the Context Pack read, but the intent artifact is opened by the same pattern in the same call path. A guard on only one would leave an identical hang reachable through the other, so `requireRegularFile` is applied to both. It uses `Lstat` on an already symlink-resolved path, so a symlink appearing between resolution and open is rejected rather than followed.

### 5. Amend the snapshot rather than remove the implementation

The alternative to version 2 was splitting the implementation out of this change. That would leave the promoted requirement describing behaviour absent from `main`, which is the defect this change exists to close. Amending the snapshot keeps requirement and implementation in one reviewable unit.

## Correlation and Evidence

- **Work identity:** confirmed intent `intent-verify-intent-context-binding-v0` version 2 and Context Pack `context-verify-intent-context-binding-v0` version 2.
- **Specification evidence:** one `MODIFIED Requirements` delta for `local-run` with seven scenarios.
- **Implementation evidence:** each scenario maps onto a named test. The accepted path and identity checks are covered by the confirmed-identity and context-bound resolver tests; the rejection classes by the invalid-binding table; the requirement rule by the current-schema and outside-current-schema tests; the non-regular artifact by the FIFO test, which runs under a deadline because the defect is a hang.
- **Ownership:** the requirement owns the preparation boundary only. Context Pack structure and intent semantics remain owned by `context-pack` and `intent-snapshot`.
- **Failure handling:** every failure class is terminal for preparation. No prepared state, run ID, launch claim, receipt, or provider invocation may exist after a rejection.

## Measurement and Stop Conditions

- Strict pinned OpenSpec validation reports this change and every promoted spec as valid.
- Every scenario in the delta maps to a named test, and the full suite plus the race detector pass.
- Each fixed finding has a test that fails against the previous implementation.
- Every Context Pack source cell matches the accepted evidence-reference pattern.
- The diff touches only this change's artifacts and the intent resolver with its tests.
- Stop and reshape if a scenario cannot be mapped to a test, or if stating the requirement would require editing merged evidence or another capability.

## Risks / Trade-offs

- **[Specification could drift from implementation]** → Each scenario is written against an observed test, and the boundary wording was corrected to the order `Service.Prepare` actually uses.
- **[Reusing the loader's rule couples two paths]** → That coupling is the point: it is what prevents the bypass. If the rule changes, both paths change together.
- **[The regular-file guard could reject a legitimate artifact]** → Only non-regular artifacts are rejected; a regular file reached through symlinks still resolves, because resolution happens before the guard.
- **[Modifying a requirement rewrites its intent annotation]** → The `Intent IDs` line moves to this change's intent, following the convention already used when a promoted requirement is modified. The prior annotation remains readable in the archived change that introduced it.
- **[Two changes touch the same capability file]** → Deltas apply to the promoted spec only at archive time. The merged hook-linkage change modified a different requirement, so the delta still applies; whichever archives second must re-read the promoted text first.

## Rollback

Delete the change directory and revert the resolver commits. Nothing else is affected: no schema, command behaviour, promoted spec, or merged file is modified until a separate archival step promotes the delta. Archival itself is reversible by restoring the previous promoted requirement text, because the delta replaces one requirement block and adds no capability.

## Open Questions

None.
