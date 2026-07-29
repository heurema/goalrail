## 1. Establish confirmed intent

- [x] 1.1 Record the Context Pack from the live verification and the external evidence: what the session proved, the silent-first-session defect, the scaffold's trust rule, the blocked project-local alternative, and the absence of any installer trust API.
- [x] 1.2 Write candidate Intent Snapshot version 1 covering disclosure, health reporting, the refusal to forge trust, the recorded limitation, and leaving the verified chain untouched.
- [x] 1.3 Present a plain-language owner-facing summary and obtain explicit owner confirmation of version 1.
- [x] 1.4 Record the confirmation receipt and set the snapshot to `confirmed`.

## 2. Compile the proposal from confirmed intent

- [x] 2.1 Trace every proposed change to confirmed intent IDs and to the non-goal it preserves.
- [x] 2.2 State that the live-verified chain and every canonical schema are unchanged.

## 3. State the contract as spec deltas

- [x] 3.1 Add the `ambient-connect` requirement for health reporting and the prohibition on forging trust, with the read-not-write line stated.
- [x] 3.2 Modify the `ambient-connect` connection requirement to disclose the trust step and to record the user-scope limitation with its cause and revisit condition.
- [x] 3.3 Modify `operator-cli-help` so the health command is discoverable, since untrusted is otherwise indistinguishable from broken.
- [x] 3.4 Run pinned telemetry-disabled strict OpenSpec validation on the change.

## 4. Record the design

- [x] 4.1 Record what the live session established and what it failed to establish, with the sequence that revealed the trust gate.
- [x] 4.2 Record the attempted project-local arrangement, the evidence that it is blocked, and why that makes user scope the only working route.
- [x] 4.3 Record why forging trust is prohibited rather than merely unimplemented, and where the read/write line falls.
- [x] 4.4 Record why health reporting distinguishes states instead of returning a verdict.
- [x] 4.5 Record the stop conditions and the rollback path.

## 5. Implement — blocked by a separate owner gate

- [x] 5.1 Add the disclosure to connection output: the trust requirement, the surface, and that the attachment does not act until then.
- [x] 5.2 Add the health command: scaffold connected, repository initialized, hooks trusted, with a next action for each state that is not working.
- [x] 5.3 Implement trust inspection as read-only, reporting an unknown state rather than guessing when it cannot be determined.
- [x] 5.4 Present the health command in help alongside the rest of the background surface.
- [x] 5.5 Confirm no code path writes, computes, or reproduces scaffold trust records.

## 6. Verify the implementation

- [x] 6.1 Map every scenario in both deltas to a named deterministic test.
- [x] 6.2 Assert the connection output names the trust requirement and the surface.
- [x] 6.3 Drive the health command through all four states and assert the answer and next action for each.
- [x] 6.4 Pin the prohibition: assert no write path to scaffold trust state exists.
- [x] 6.5 Confirm the live-verified behaviour keeps its existing tests unmodified.
- [x] 6.6 Run `gofmt -l`, `go build ./...`, `go vet ./...`, `go test ./...`, and `go test -race ./...`.
- [x] 6.7 Run pinned telemetry-disabled strict OpenSpec validation across the change and every promoted spec.
- [x] 6.8 Re-check on the live stand where the defect was first observed: health reports `pending` before trust and names `/hooks`, and connection output states that nothing runs until then.

## 7. Stop at the owner gates

- [x] 7.1 Obtain an explicit owner instruction before beginning implementation.
- [ ] 7.2 Obtain a separate explicit owner instruction before committing.
- [ ] 7.3 Obtain a separate explicit owner instruction before pushing, opening a pull request, or merging.
- [ ] 7.4 Obtain a separate explicit owner instruction before archiving this change and promoting the deltas; re-read the promoted text immediately before promotion in case another change archived first.
- [ ] 7.5 Revisit the recorded limitation when the external project-local hook defect is fixed; the structural boundary is the first thing to reconsider.
