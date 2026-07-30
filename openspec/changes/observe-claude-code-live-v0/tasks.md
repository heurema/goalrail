## 1. Establish confirmed intent

- [x] 1.1 Record the Context Pack: what the promoted requirement claims is unobserved, the superseded blocker and the two facts that withdraw it, what the live run established, and the two conditions that leave the approval question open.
- [x] 1.2 Write candidate Intent Snapshot version 1 covering the observed behaviours, the preserved gap with its reasons, the retained evidence, and the withdrawn blocker.
- [x] 1.3 Present the run's result and the proposed scope in plain language and obtain explicit owner confirmation.
- [x] 1.4 Record the confirmation receipt and set the snapshot to `confirmed`.

## 2. Compile the proposal from confirmed intent

- [x] 2.1 Trace every proposed change to confirmed intent IDs and to the non-goal it preserves.
- [x] 2.2 State that no behaviour and no code changes.

## 3. State the contract as spec deltas

- [x] 3.1 Modify the `ambient-connect` connection requirement: announcement delivery and question retention recorded as observed, the approval question kept open with both conditions that make the run inconclusive, and the superseded blocker withdrawn.
- [x] 3.2 Add two general scenarios: a partly observed scaffold is recorded as partly observed, and an absence produced under conditions where the behaviour would not have appeared is not evidence.
- [x] 3.3 Generate the delta from the promoted requirement rather than retyping it, and assert by set difference that no promoted scenario is dropped. 20 promoted, 22 in the delta, none dropped.
- [x] 3.4 Run pinned telemetry-disabled strict OpenSpec validation on the change.

## 4. Record the design

- [x] 4.1 Record that the earlier blocker was a property of the two routes considered rather than of the scaffold, and that a blocker deserves the same scepticism as the thing it blocks.
- [x] 4.2 Record how delivery and retention were established rather than assumed.
- [x] 4.3 Record why the absent approval prompt is not evidence, and why the record states the conditions rather than the conclusion.
- [x] 4.4 Record why two general scenarios were added rather than only amending one paragraph.
- [x] 4.5 Record what evidence is kept and why keeping the fixture whole costs no privacy.
- [x] 4.6 Record that the delta was generated from the promoted text with a parity guard.
- [x] 4.7 Record the stop conditions and the rollback path.

## 5. Preserve the evidence

- [x] 5.1 Copy the invocation, the hooks supplied for the run, the fixture's shape, the agent's reply, the question it wrote, and the retained record into the change's evidence directory.
- [x] 5.2 Reduce absolute scratch paths to a placeholder, so the file reads correctly after the directory is gone.
- [x] 5.3 Record what the run did not establish in the evidence file itself, not only in the requirement.

## 6. Verify

- [x] 6.1 Confirm the requirement no longer lists announcement delivery or question retention as unverified.
- [x] 6.2 Confirm the requirement still states that the approval question is open, naming both the non-interactive mode and the per-run supply.
- [x] 6.3 Confirm no promoted scenario was dropped.
- [x] 6.4 Confirm there is no Go diff and the existing suite passes unmodified.
- [x] 6.5 Confirm the owner's scaffold configuration holds no managed handler after the run.
- [x] 6.6 Run pinned telemetry-disabled strict OpenSpec validation across the change and every promoted spec.

## 7. Stop at the owner gates

- [x] 7.1 Obtain a separate explicit owner instruction before committing and pushing; record it before the action. Recorded before the commit.
- [x] 7.2 Obtain a separate explicit owner instruction before opening a pull request and before merging. The owner granted the sequence explicitly in one instruction covering commit, push, pull request, merge, and promotion, for a change that alters no behaviour.
- [x] 7.3 Obtain a separate explicit owner instruction before archiving this change and promoting the delta; re-read the promoted text immediately before promotion in case another change archived first. Covered by the same instruction; the delta was generated from the promoted text minutes earlier and re-checked for parity before promotion.
- [ ] 7.4 Answer the approval question only when an interactive session with a registration in the owner's own configuration is something the owner chooses to run, since that arrangement fires the hook inside the sessions doing the work.
