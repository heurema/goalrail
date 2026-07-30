## 1. Establish confirmed intent

- [x] 1.1 Record the Context Pack: health's correct detection and its prescribed remedy, connection's marker-only presence test, each scaffold's write path, the committed test comment that scoped the defect out, issue #25's constraints, the promoted requirement's binding rules, and the trust consequence of replacing a command.
- [x] 1.2 Write candidate Intent Snapshot version 1 covering repair on reconnect, in-place replacement, byte-identical repeat, foreign handlers, the scoped session-start entry, and the trust disclosure.
- [x] 1.3 Present a plain-language owner-facing summary and obtain explicit owner confirmation of version 1.
- [x] 1.4 Record the confirmation receipt and set the snapshot to `confirmed`.

## 2. Compile the proposal from confirmed intent

- [x] 2.1 Trace every proposed change to confirmed intent IDs and to the non-goal it preserves.
- [x] 2.2 State that health's detection, trust reporting, the announcement, retention, intent binding, the fail-quiet posture, and the wrapper lifecycle are unchanged.

## 3. State the contract as spec deltas

- [x] 3.1 Modify the `ambient-connect` connection requirement: presence judged by what the registration names, repair replacing rather than accompanying, a current registration left byte-identical, foreign handlers and already-current events untouched, the repaired session-start entry still scoped, the repair disclosure under the existing per-scaffold discipline, and the rule that a remedy a report prescribes must be one the named command can perform.
- [x] 3.2 Restate every promoted scenario in the delta, verified by set difference rather than by reading, because a `MODIFIED` requirement replaces the promoted one wholesale at archival and a dropped scenario is a silent deletion.
- [x] 3.3 Carry no normative rule that traces to no confirmed outcome; a draft rule requiring the plan to name the stale executable was removed on that ground.
- [x] 3.4 Run pinned telemetry-disabled strict OpenSpec validation on the change.

## 4. Record the design

- [x] 4.1 Record why presence and currency are separate questions and why folding them together would break health.
- [x] 4.2 Record why the executable path is compared rather than the whole command, what that deliberately does not catch, and that path inequality is broader than the observed defect — a still-runnable different binary is repaired too, at the cost of a review step — together with the unmitigated hazard of connecting from a short-lived binary.
- [x] 4.3 Record why repair must replace, how each scaffold fails if it merely appends, that the first scaffold's write removes first unconditionally, and the two visible consequences of reusing disconnection's removal — the block relocates, and an unterminated block fails the repair loudly.
- [x] 4.4 Record why reconciliation stays per event.
- [x] 4.5 Record why the repair is the only moment the stale trust record can be known, and why omitting the disclosure would trade one silent failure for another.
- [x] 4.6 Record why the disclosure stays per-scaffold.
- [x] 4.7 Record that making the test helper's executable explicit exposes a previously unpinned guarantee.
- [x] 4.8 Record that the repair disclosure does not survive into a later health check, that closing it contradicts confirmed non-goal NG-1 and confirmed signal SIG-5, and the observation-with-memory route that would close it under a new intent version.
- [x] 4.9 Record the adjacent gap in health that is deliberately left alone.
- [x] 4.10 Record the stop conditions and the rollback path.

## 5. Implement — blocked by a separate owner gate

- [x] 5.1 Add executable extraction for every managed handler, per scaffold, reusing one shared parser for the command form. `executableFromCommand` reverses the shell quoting; `registeredExecutables` reaches the commands through each scaffold's own encoding.
- [x] 5.2 Keep health's existing check equivalent across that refactor, changing neither which handler it inspects nor what it reports. It still inspects the first handler only. Its verdict is unchanged except where the old text-scanning parser was wrong: a path containing an apostrophe was reported missing when present, and review made that a correction worth taking rather than a boundary worth keeping, since the same input would otherwise get two answers in one package.
- [x] 5.3 Judge connection presence by both the marker test and the executable comparison, leaving the marker test's own meaning and its health caller unchanged. `isConnected` untouched; `staleExecutable` is a second, separate question only connection asks.
- [x] 5.4 Carry the stale executable path in the plan as reported detail, not as a promoted contract. `ConnectionPlan.Repair` and `RegisteredExecutable`.
- [x] 5.5 Replace in place on the first scaffold by removing any existing managed block before writing, unconditionally rather than only on the repair path, reusing disconnection's removal including its refusal on an unterminated block. Unconditional in `connectCodex`; a test pins that it also closes the partial-block duplication.
- [x] 5.6 Replace in place on the second scaffold per event, reusing the strip helper the existing session-start repair uses, so foreign handlers and their occurrences survive and an event that is already current is not rewritten. Both events now go through `stripManagedHandlers`; `claudeCodeEventIsCurrent` skips a current event.
- [x] 5.7 Keep the repaired session-start entry scoped to the occurrence that opens a session. The scoped branch also fires on staleness, so a stale-and-unscoped registration repairs to the scoped form.
- [x] 5.8 Add the repair disclosure under the existing per-scaffold discipline: asserted where the trust gate was observed, marked unverified where it was not. `RepairNotice`, asserted for the observed scaffold and marked unverified for the other.
- [x] 5.9 Never report a repaired attachment as active on the strength of the existing trust record, and write no trust record. `runConnect` returns early on repair with `active_now: false`; a test pins that no trust record is written.

## 6. Verify the implementation

- [x] 6.1 Map every scenario in the delta to a named deterministic test.
- [x] 6.2 Assert a stale registration is repaired on each scaffold, leaving exactly one registration per event naming the current executable. `TestConnectRepairsAStaleExecutable` over both scaffolds.
- [x] 6.3 Assert the plan reports the attachment as not already present and carries the stale executable. Same test asserts `AlreadyPresent` false, `Repair` true, and the stale path.
- [x] 6.4 Assert a registration already naming the current executable leaves the configuration byte-identical, on each scaffold. `TestRepairIsNotTriggeredForTheCurrentExecutable` over both scaffolds.
- [x] 6.5 Assert a foreign handler for a repaired event survives with its command and occurrence unchanged, and that an event whose registration is already current is not rewritten while another event is repaired. `TestRepairPreservesAForeignHandlerForTheSameEvent` and `TestRepairLeavesAnEventThatIsAlreadyCurrent`, the latter using a sentinel key our writer never produces.
- [x] 6.6 Assert a stale and unscoped session-start registration is repaired to the scoped form. `TestRepairKeepsSessionStartScoped`.
- [x] 6.7 Assert health names the missing binary before the repair and reports working after it. `TestHealthReportsWorkingAfterTheRepair` follows the advice health gives and checks it worked.
- [x] 6.8 Assert the repair discloses that review applies again, that the disclosure stays unverified for the scaffold whose gate was not observed, and that a repaired attachment is not reported as active. `TestRepairNoticeMatchesWhatWasActuallyObserved` and `TestConnectDoesNotCallARepairedAttachmentActive`, the latter seeding a trust record so that reporting active could only come from the mistake.
- [x] 6.9 Make the shared test helper's executable explicit and confirm the two idempotency tests now pin a stable path. Confirmed by experiment: restoring the old helper shape makes the idempotency test fail, so it never pinned same-binary idempotency.
- [x] 6.10 Confirm the tests pinning that no trust record is ever written are unmodified and still pass. `TestNoTrustRecordIsEverWritten` unmodified; `TestRepairWritesNoTrustRecord` added for the repair path.
- [x] 6.11 Run `gofmt -l`, `go build ./...`, `go vet ./...`, `go test ./...`, and `go test -race ./...`. All clean, including `-race`.
- [x] 6.12 Run pinned telemetry-disabled strict OpenSpec validation across the change and every promoted spec. 11 passed, 0 failed.

## 7. Answer the review

- [x] 7a.1 Decode the shell-quoted executable rather than reading the text between the last two apostrophes; a path containing an apostrophe otherwise reports every current registration as stale and rewrites it. Confirmed by probe before fixing, pinned by `TestRegistrationSurvivesAnApostropheInThePath`.
- [x] 7a.2 Read the TOML scaffold's `command` assignments inside the managed block rather than scanning the file, so a copy of an old stanza outside the markers is not counted. Pinned by `TestStalenessIgnoresACommandOutsideTheManagedBlock`, whose first version did not isolate the fix and was rewritten after it passed with the defect restored.
- [x] 7a.3 Remove every managed block before writing, not one; a configuration carrying two was reachable from the write path this change replaces. Confirmed by probe, pinned by `TestRepairRemovesEveryManagedBlock`.
- [x] 7a.4 Point health's executable check at the same decoder so one input does not get two answers in one package, keeping its first-handler-only scope intact.
- [x] 7a.5 Stop the repair notice from pointing at `gr health`, which reports the repaired attachment as working and so contradicts the notice. It now states that limit and why; the other scaffold keeps the pointer, where health reports the trust state as undetermined rather than working. Pinned by `TestRepairNoticeDoesNotSendTheUserToASurfaceThatContradictsIt`.
- [ ] 7a.6 Decide whether the repaired-but-unreviewed state must also be observable to a later health check. Confirmed signal SIG-5 says health reports working after a repair and confirmed non-goal NG-1 excludes health's trust reporting, so closing it reverses a confirmed signal and needs its own intent version. Recorded in design decision 8 with the route.

## 8. Stop at the owner gates

- [x] 8.1 Obtain an explicit owner instruction before beginning implementation. Recorded before the first edit.
- [x] 8.2 Obtain a separate explicit owner instruction before committing; record it before the commit, not after. Recorded before the commit.
- [x] 8.3 Obtain a separate explicit owner instruction before pushing and opening a pull request; merging remains ungranted. Recorded before the push.
- [x] 8.4 Obtain a separate explicit owner instruction before committing and pushing the review round. Recorded before the action. Pushed as its own commit rather than an amended one, so the pull request shows what changed in answer to the review.
- [x] 8.5 Obtain a separate explicit owner instruction before archiving this change and promoting the delta; re-read the promoted text immediately before promotion in case another change archived first. Recorded before the action. The promoted requirement was re-read and still carried its 12 scenarios, unchanged since the delta was authored.
- [x] 8.6 Issue #25 closes on merge through the trailer in the commit message. The owner reviewed that and chose to keep it rather than gate the closure separately. Recorded because the ordering is visible: the issue closes at merge, while the delta is promoted in the step above, so there is a short window in which the fix is merged and its contract is not yet promoted.
