## 1. Establish confirmed intent

- [x] 1.1 Record the Context Pack: the missing announcement, what the adapter actually passes the provider, the hook already rendered for lineage, the provider's published hook output contract, the provider-neutrality constraint, and the measurement's loud-treatment caveat.
- [x] 1.2 Write candidate Intent Snapshot version 1 covering the announcement, its transport, its single delivery, and its deliberate minimality.
- [x] 1.3 Present a plain-language owner-facing summary and obtain explicit owner confirmation of version 1; continuation, silence, and absence of objection are not confirmation.
- [x] 1.4 Record the confirmation receipt and set the snapshot to `confirmed`.

## 2. Compile the proposal from confirmed intent

- [x] 2.1 Trace every proposed change to confirmed intent IDs and to the non-goal it preserves.
- [x] 2.2 State that no canonical schema changes and that composing the work prompt stays with the operator.

## 3. State the contract as spec deltas

- [x] 3.1 Add the `local-run` requirement for the launch-time announcement: content, neutrality, single delivery, and the prohibition on task hints.
- [x] 3.2 Modify the `local-run` adapter requirement so carrying the announcement is an adapter obligation and an adapter that cannot deliver fails the launch.
- [x] 3.3 Run pinned telemetry-disabled strict OpenSpec validation on the change.

## 4. Record the design

- [x] 4.1 Record why the existing hook is reused and why the environment variable, adapter arguments, and worktree-file alternatives were rejected.
- [x] 4.2 Record the neutral-content/specific-transport split and why the text is fixed rather than composed per run.
- [x] 4.3 Record why an undeliverable announcement fails the launch instead of degrading silently.
- [x] 4.4 Record why the announcement is one statement and not a protocol.
- [x] 4.5 Record the minimality decision against the measurement's loud-treatment caveat, and what the announcement is not allowed to become.
- [x] 4.6 Record the open provider-behaviour question, the stop conditions, and the rollback path.

## 5. Implement — blocked by a separate owner gate

- [x] 5.1 Add the fixed announcement text as provider-neutral content owned by the local-run boundary, not by the Codex adapter.
- [x] 5.2 Extend the adapter contract so a launch carries the announcement, and make an adapter without a delivery path fail the launch before any run ID or launch claim exists.
- [x] 5.3 Deliver the announcement through the rendered `SessionStart` hook as its session context, alongside the lineage correlation the hook already performs.
- [x] 5.4 Confirm the canonical WorkSpec, the terminal receipt, and the target worktree gain nothing: no field, no file, no path.

## 6. Verify the implementation

- [x] 6.1 Map every scenario in both deltas to a named deterministic test.
- [x] 6.2 Pin the announcement text with a test asserting it names no provider and contains nothing about conflicts, documents, or the work item.
- [x] 6.3 Prove with a fixture adapter that has no delivery path that the launch fails explicitly and starts no run.
- [x] 6.4 Assert the announcement is delivered exactly once, with no acknowledgement, retry, or second delivery.
- [x] 6.5 Confirm the frozen `goalrail.work-spec/v0` fixture digest and the terminal receipt schema are unchanged.
- [x] 6.6 Run `gofmt -l`, `go build ./...`, `go vet ./...`, `go test ./...`, and `go test -race ./...`.
- [x] 6.7 Run pinned telemetry-disabled strict OpenSpec validation across the change and every promoted spec.

## 7. Address review findings on the implementation

- [x] 7.1 Delegate the delivery report to the launcher, so the adapter cannot claim an announcement that only a capsule it does not install would deliver.
- [x] 7.2 Emit the announcement only for the session-start source that opens a run, so resumption, clearing, and compaction do not repeat it.
- [x] 7.3 Verify delivery before consuming the one-time admission, so an undeliverable announcement does not discard the owner's authorization.
- [x] 7.4 Retain the measurement claims beside the Context Pack and repoint the misattributed constraint at the promoted capability that actually states it.
- [x] 7.5 Add a regression test per finding, state each new rule in the spec delta, and re-run the full verification set.
- [x] 7.6 Leave the confirmed intent unmodified: the corrections make the shipped behaviour honest against outcomes version 1 already states.

## 8. Stop at the owner gates

- [x] 8.1 Obtain an explicit owner instruction before beginning implementation.
- [x] 8.2 Obtain a separate explicit owner instruction before committing.
- [x] 8.3 Obtain a separate explicit owner instruction before pushing and opening a pull request; merging remains ungranted.
- [x] 8.4 Obtain a separate explicit owner instruction before archiving this change and promoting the deltas; re-read the promoted text immediately before promotion in case another change archived first. The promoted `local-run` text was confirmed unchanged since the delta was authored, so the modified adapter requirement applied to the text it was written against.
- [ ] 8.5 Obtain a separate explicit owner authorization before any real provider run. That run closes two open questions at once: whether the reserved path is writable inside the provider sandbox, and whether the announced context reaches the agent. Neither is evidence until it happens.
