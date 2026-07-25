## Context

Intent Snapshot `intent-verify-intent-context-binding-v0` version 1 is confirmed. This change is specification-only: it gives an already-shipped preparation behaviour a promoted contract in the `local-run` capability.

The behaviour lives in `internal/adapters/openspec/work_spec_intent.go`. `IntentResolver.Verify` reads the referenced intent artifact, confines it to the canonical repository root, applies a bounded size limit, compares its digest, and then hands the raw bytes to a resolver that looks for a sibling `context.md`. When that artifact resolves, it is read under the same bound, parsed as a Context Pack, and the intent is parsed against it before flow validation. When it does not resolve, the intent is inspected for a declared Context Pack: a declaration without an artifact fails; no declaration succeeds on the intent alone.

The promoted `local-run` capability on `main` does not describe any of this. Its `Run preparation is inspectable and does not launch` requirement mentions the confirmed intent reference only as one invalid-input case in a scenario.

The only existing wording is inside the unmerged `activate-dogfood-run-v0` change, embedded in a `MODIFIED Requirements` block for `Real activation remains separately gated`. That requirement was rewritten on `main` when the exact admission boundary landed, and that change declares an intent ID an archived change already claims. The wording is therefore extracted rather than transplanted.

## Goals / Non-Goals

**Goals:**

- State preparation-time Context Pack enforcement in exactly one promoted requirement.
- Describe the accepted path, every rejection class, and the no-pack path that the implementation already supports.
- Keep the requirement provider-neutral and free of canonical WorkSpec schema effects.
- Leave merged evidence and every other capability untouched.

**Non-Goals:**

- No implementation, refactor, or test change.
- No restatement of the intent-level Context Pack semantics already promoted in `intent-snapshot` and `context-pack`.
- No change to the hook-linkage requirement or to any activation, help, or WorkSpec capability.
- No workflow engine, authorization system, effect gateway, credential broker, sandbox platform, database, or dashboard.

## Decisions

### 1. Modify the preparation requirement instead of adding a new one

Preparation-time enforcement is a property of preparation, not a separate capability. Attaching it to `Run preparation is inspectable and does not launch` keeps the accepted path and its failure modes in one place, and avoids a second requirement that would have to restate the same freeze boundary.

### 2. Extract the wording rather than transplant the unmerged block

The unmerged wording is entangled with the superseded activation requirement. Extracting it produces a requirement that stands on its own, does not reference activation, and does not depend on the unmerged change surviving.

### 3. State the no-pack path explicitly

The implementation accepts an intent that declares no Context Pack. Leaving that implicit would let a future reader treat the pack as unconditionally mandatory and tighten the check into a regression against existing callers.

### 4. Use a new intent identity

`intent-activate-dogfood-run-v0` is claimed by an archived change on `main` with different confirmed wording. Reusing or re-versioning it would rewrite merged evidence, which the workspace contract forbids. A new intent ID leaves the archive as the historical record.

## Correlation and Evidence

- **Work identity:** confirmed intent `intent-verify-intent-context-binding-v0` version 1 and Context Pack `context-verify-intent-context-binding-v0` version 1.
- **Specification evidence:** one `MODIFIED Requirements` delta for `local-run` with six scenarios.
- **Implementation evidence:** each scenario maps onto an existing test. The accepted path and the no-pack path are covered by the confirmed-identity and context-bound resolver tests; the rejection classes are covered by the invalid-binding table, which includes missing context, mismatched context identity, unknown context reference, and a context symlink escaping the repository.
- **Ownership:** the requirement owns the preparation boundary only. Context Pack structure and intent semantics remain owned by `context-pack` and `intent-snapshot`.
- **Failure handling:** every failure class is terminal for preparation. No prepared state, run ID, launch claim, receipt, or provider invocation may exist after a rejection.

## Measurement and Stop Conditions

- Strict pinned OpenSpec validation reports this change and every promoted spec as valid.
- Every scenario in the delta maps to a named existing test; no scenario requires new code.
- `go test ./...` and `go vet ./...` are unchanged and passing, confirming the change adds no behaviour.
- The diff touches only this change's own artifacts under `openspec/changes/verify-intent-context-binding-v0/`.
- Stop and reshape if a scenario cannot be mapped to an existing test, or if stating the requirement would require editing merged evidence or another capability.

## Risks / Trade-offs

- **[Specification could drift from implementation]** → Each scenario is written against an observed test rather than intended behaviour, and the mapping is recorded in this design.
- **[Modifying a requirement rewrites its intent annotation]** → The `Intent IDs` line moves to this change's intent, following the convention already used when a promoted requirement is modified. The prior annotation remains readable in the archived change that introduced it.
- **[Two open changes touch the same capability file]** → Deltas apply to the promoted spec only at archive time, so concurrent unarchived changes do not conflict. Whichever archives second must re-read the promoted text first.
- **[Stating the no-pack path could look permissive]** → The path is bounded: it applies only when the intent declares no pack, and a declared pack without an artifact still fails closed.

## Rollback

Delete the change directory. Nothing else is affected: no code, schema, command behaviour, promoted spec, or merged file is modified until a separate archival step promotes the delta. Archival itself is reversible by restoring the previous promoted requirement text, because the delta replaces one requirement block and adds no capability.

## Open Questions

None.
