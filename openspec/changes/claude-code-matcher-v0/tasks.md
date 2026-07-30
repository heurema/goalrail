## 1. Establish confirmed intent

- [x] 1.1 Record the Context Pack: the provider's matcher contract, the shipped registration verified on a stand, the owner's own four-entry configuration, the promoted single-delivery rule, the project-layer difference, the overreached limitation, and the abandoned live attempt.
- [x] 1.2 Write candidate Intent Snapshot version 1 covering per-occurrence registration, exact removal, the attributed limitation, and the recorded unobserved behaviour.
- [x] 1.3 Present a plain-language owner-facing summary and obtain explicit owner confirmation of version 1.
- [x] 1.4 Record the confirmation receipt and set the snapshot to `confirmed`.

## 2. Compile the proposal from confirmed intent

- [x] 2.1 Trace every proposed change to confirmed intent IDs and to the non-goal it preserves.
- [x] 2.2 State that the first scaffold's registration and every live-verified behaviour are unchanged.

## 3. State the contract as spec deltas

- [x] 3.1 Modify the `ambient-connect` connection requirement: per-occurrence registration where the scaffold distinguishes them, removal spanning every registered occurrence, the limitation attributed to its scaffold, and the second scaffold's unobserved behaviour recorded.
- [x] 3.2 Run pinned telemetry-disabled strict OpenSpec validation on the change.

## 4. Record the design

- [x] 4.1 Record why the constraint belongs in registration rather than in the transport, and that this keeps one rule with one meaning across scaffolds.
- [x] 4.2 Record why removal must widen with registration and why the failure it prevents is silent.
- [x] 4.3 Record why attributing the limitation is substantive rather than cosmetic.
- [x] 4.4 Record that the fix rests on two agreeing sources and is not observation.
- [x] 4.5 Record that project-level registration is a deferred decision, not a forgotten one.
- [x] 4.6 Record the stop conditions and the rollback path.

## 5. Implement — blocked by a separate owner gate

- [x] 5.1 Register one session-start entry per opening occurrence for the scaffold that distinguishes them, and register no form that fires unconditionally.
- [x] 5.2 Keep the reconciliation per event and per occurrence, so a repeated consented connection repairs a partial registration without duplicating a present one.
- [x] 5.3 Widen removal to every registered occurrence while leaving foreign entries for the same event untouched.
- [x] 5.4 Leave the first scaffold's registration unchanged.

## 6. Verify the implementation

- [x] 6.1 Map every scenario in the delta to a named deterministic test.
- [x] 6.2 Assert the produced registration carries one entry per opening occurrence and none that fires on every occurrence.
- [x] 6.3 Assert connect-then-disconnect restores the configuration byte for byte, including with a foreign entry present for the same event.
- [x] 6.4 Assert a partial registration is repaired by a repeated connection without duplicating what is present.
- [x] 6.5 Confirm the first scaffold's tests and every live-verified behaviour are unmodified.
- [x] 6.6 Run `gofmt -l`, `go build ./...`, `go vet ./...`, `go test ./...`, and `go test -race ./...`.
- [x] 6.7 Run pinned telemetry-disabled strict OpenSpec validation across the change and every promoted spec.
- [x] 6.8 A test written for this change surfaced an adjacent defect outside its scope: a registration whose executable path is stale is reported as already present, so health's advice to re-run connection does nothing. Recorded as issue #25 rather than widened into this change.

## 7. Stop at the owner gates

- [x] 7.1 Obtain an explicit owner instruction before beginning implementation.
- [x] 7.2 Obtain a separate explicit owner instruction before committing; record it before the commit, not after. Recorded before the commit this time.
- [x] 7.3 Obtain a separate explicit owner instruction before pushing and opening a pull request; merging remains ungranted. Recorded before the action.
- [ ] 7.4 Obtain a separate explicit owner instruction before archiving this change and promoting the delta; re-read the promoted text immediately before promotion in case another change archived first.
- [ ] 7.5 Exercise the second scaffold in a live session when the owner can run it in their own environment, and record the result; until then its end-to-end behaviour stays unobserved.
- [ ] 7.6 Revisit project-level registration for the second scaffold as its own decision, with its own evidence.
