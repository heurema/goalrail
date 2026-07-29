## 1. Establish confirmed intent

- [x] 1.1 Record the Context Pack with the measurement, the receipt and WorkSpec constraints, the review consensus, and the owner's variant selection; complete it as `sufficient` with no material unknown.
- [x] 1.2 Write candidate Intent Snapshot version 1 covering the terminal outcome, the reserved path, the state rules, receipt versioning, the chain, and the payload shape.
- [x] 1.3 Present a plain-language owner-facing summary and obtain explicit owner confirmation of version 1; continuation, silence, and absence of objection are not confirmation.
- [x] 1.4 Record the confirmation receipt — owner, timestamp, and the active verification action — and set the snapshot to `confirmed`.

## 2. Compile the proposal from confirmed intent

- [x] 2.1 Trace every proposed change to confirmed intent IDs and to the non-goal it preserves.
- [x] 2.2 State the one intended contract break — the terminal status whitelist — together with the receipt version that makes it legible.
- [x] 2.3 Record that `goalrail.work-spec/v0` is untouched and that the impact is confined to the receipt, the lifecycle, and the intent-side answering record.

## 3. State the contract as spec deltas

- [x] 3.1 Add the `local-run` requirement for the blocked terminal outcome: retention, hygiene, the state rules, and the precedence rules, with a scenario per rule.
- [x] 3.2 Modify the `local-run` preparation requirement to fail closed on a pre-populated reserved path, scoped to the exact file rather than its directory.
- [x] 3.3 Modify the `local-run` terminal receipt requirement for the explicit schema version, the frozen intent reference, and the escalation block, keeping schema-less receipts readable.
- [x] 3.4 Add the `intent-snapshot` requirement for the optional answering record and its dispositions, granting no effect authority.
- [x] 3.5 Add the `escalation-question` capability: the provider-neutral payload shape and downstream-only semantic validation.
- [x] 3.6 Run pinned telemetry-disabled strict OpenSpec validation on the change.

## 4. Record the design

- [x] 4.1 Record the variant A decision, the rejected variant B, and why the break happens once with a version.
- [x] 4.2 Record the reserved-path decision against the verified observation and namespace facts, and the falsified premise it replaces.
- [x] 4.3 Record eager retention inside the delta, and the single-observation decision together with the behaviour change it implies.
- [x] 4.4 Record the resolution of the panel's disagreement over check results, and why a failing check keeps `failed`.
- [x] 4.5 Record the cut recurrence claim and the chain that replaces it.
- [x] 4.6 Record the stop conditions, including sandbox writability of the reserved path, and the rollback path.

## 5. Implement

- [x] 5.1 Verify that the supported adapter can write the reserved path inside the provider sandbox. **Established by analysis, not by experiment:** Goalrail configures no sandbox — it passes the repository root as the working directory and leaves the filesystem policy to the provider, as the declared posture requires; the local provider runs `workspace-write`, which makes the working directory writable, with exclusions parameterized rather than fixed to directory names; and the reserved path lies inside that working directory. `codex sandbox` requires a permission profile whose configuration shape is not documented locally, and `codex exec` would launch a real provider run, which is a separate owner gate. The empirical confirmation therefore lands at 8.5, before any real run is treated as evidence.
- [x] 5.2 Add the reserved-path constant and its bounded hygiene check, reusing the retained-text rules and the descriptor-level regular-file guarantees rather than adding a second family. The descriptor-level read moved into `internal/boundedio` and both callers now share it.
- [x] 5.3 Add the preparation gate on the frozen baseline observation, rejecting the exact reserved file and leaving unrelated content in its directory alone.
- [x] 5.4 Observe the worktree once after provider observation in `Start`, retain the artifact bytes append-only, and persist that observation; fail the run when retention fails. Retention was later reordered ahead of the observation write — see 7.3.
- [x] 5.5 Compute the delta in `Finish` from the persisted observation, falling back to observing at finish only when no persisted observation exists.
- [x] 5.6 Exclude the reserved path from scope violations and from the in-scope edit test while keeping it in the observed changed paths.
- [x] 5.7 Add the `blocked` state, the precedence rules, the hedge rule, the failing-check rule, and the never-`passed` rule.
- [x] 5.8 Add the receipt schema identifier, the frozen intent reference, and the escalation block; keep schema-less receipts readable and never rewrite a stored receipt.
- [x] 5.9 Parse the optional `resolves` and `disposition` records on an intent version in the OpenSpec adapter, rejecting malformed and duplicate records.
- [x] 5.10 Publish `goalrail.escalation/v0` as a spec artifact consumable by the kata oracle, with no Goalrail runtime dependency: `docs/schemas/goalrail-escalation-v0.md`.

## 6. Verify the implementation

- [x] 6.1 Map every scenario in every delta to a named deterministic test. Two `escalation-question` scenarios — a payload that names a provider, and a payload rejected by downstream validation — are owned by the downstream consumer by construction, because Goalrail never parses the payload. They are stated as contract for that consumer and have no Goalrail-side test.
- [x] 6.2 Add the gaming regression tests: pre-existing artifact fails preparation; artifact plus in-scope edits yields `failed`; each precedence case; verbatim results preserved; a failing check stays `failed`; no input combination yields `passed` while an artifact is retained.
- [x] 6.3 Add the retention test: delete the worktree file after provider observation and confirm the recorded outcome and the retained bytes are unchanged.
- [x] 6.4 Add the frozen `goalrail.work-spec/v0` fixture digest regression test, an unknown-field probe proving the schema cannot silently gain an escalation field, and confirm a schema-less receipt still validates.
- [x] 6.5 Cover the moved observation timing. No existing test asserted finish-time observation, so the timing change is proven by two new tests instead: one asserting `Finish` takes no further observation, and one asserting the fallback still observes when no start-time observation exists.
- [x] 6.6 Run `gofmt -l`, `go build ./...`, `go vet ./...`, `go test ./...`, and `go test -race ./...`.
- [x] 6.7 Run pinned telemetry-disabled strict OpenSpec validation across the change and every promoted spec.

## 7. Address review findings on the implementation

- [x] 7.1 Gate the reserved path in `Start` before launching, closing the window between preparation and a later start of a reused prepared run.
- [x] 7.2 Compare the bytes read for retention against the digest the deciding observation recorded, and record a mismatch as invalid instead of binding it to the receipt.
- [x] 7.3 Retain before persisting the observation, and fail `Finish` closed when a persisted observation names the reserved path without a retained record.
- [x] 7.4 Apply the repository-boundary check to the reserved path's ancestors, so a symlinked reserved directory cannot silently move the channel outside the repository.
- [x] 7.5 Require a usable frozen intent reference on v1 receipts, keeping the relaxed path only for receipts that predate the schema identifier.
- [x] 7.6 Require a material amendment for any change to the answering escalation record.
- [x] 7.7 Add a regression test per finding, state each new rule in the spec deltas, and re-run the full verification set including the race detector and strict OpenSpec validation.

## 8. Stop at the owner gates

- [x] 8.1 Obtain an explicit owner instruction before beginning implementation.
- [x] 8.2 Obtain a separate explicit owner instruction before committing.
- [x] 8.3 Obtain a separate explicit owner instruction before pushing and opening a pull request; merging remains ungranted.
- [x] 8.4 Obtain a separate explicit owner instruction before archiving this change and promoting the deltas; re-read the promoted text immediately before promotion in case another change archived first. The promoted specs were confirmed unchanged since the deltas were written, so both modified requirements applied to the text they were authored against.
- [ ] 8.5 Obtain a separate explicit owner authorization before any real provider run; planning and implementation completion authorize none.
