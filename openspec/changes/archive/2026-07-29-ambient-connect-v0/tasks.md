## 1. Establish confirmed intent

- [x] 1.1 Record the Context Pack: the owner's background direction, the wrapper-only promoted lifecycle, the verified persistent hook mechanisms on both scaffolds, the already-built channel pieces, the existing intent gate, the privacy prohibition, and the posture difference.
- [x] 1.2 Write candidate Intent Snapshot version 1 covering connection, the directory gate, session-start and session-stop behaviour, loop closure through the intent gate, and the fail-quiet posture.
- [x] 1.3 Present a plain-language owner-facing summary, answer the owner's clarifying question — ordinary tasks in ordinary projects are the primary scenario; katas remain benchmark material — and obtain explicit owner confirmation of version 1.
- [x] 1.4 Record the confirmation receipt and set the snapshot to `confirmed`.

## 2. Compile the proposal from confirmed intent

- [x] 2.1 Trace every proposed change to confirmed intent IDs and to the non-goal it preserves.
- [x] 2.2 State that the wrapper lifecycle, canonical schemas, kata, and benchmark are unchanged, and that background mode produces question records rather than run outcomes.

## 3. State the contract as spec deltas

- [x] 3.1 Add the `ambient-connect` capability: consented reversible connection, the initialized-directory gate, session-start announcement with stale-file archival, session-stop retention with exact-or-explicitly-unbound intent binding, and the fail-quiet posture beside the unchanged fail-closed wrapper.
- [x] 3.2 Modify `intent-snapshot` so the answering reference names a blocked run or a background question record.
- [x] 3.3 Modify `operator-cli-help` so help presents the background surface, states that ordinary work needs no per-task command, and never advertises the scaffold-invoked hook entry point.
- [x] 3.4 Run pinned telemetry-disabled strict OpenSpec validation on the change.

## 4. Record the design

- [x] 4.1 Record that the helper is the `gr` binary invoked by the hook — no daemon, no second executable.
- [x] 4.2 Record that initialization is an explicit marker file, not the `.goalrail/` directory, and why.
- [x] 4.3 Record fail-quiet as exit-zero-always toward the scaffold with errors kept in the state root.
- [x] 4.4 Record that the ambient announcement is its own constant under the launch announcement's pinned discipline, reusing the source-aware renderer.
- [x] 4.5 Record start-time archival, exact-or-explicitly-unbound binding, and why clean-scope rules stay with the wrapper.
- [x] 4.6 Record the open hook-behaviour limit, the stop conditions, and the rollback path.

## 5. Implement — blocked by a separate owner gate

- [x] 5.1 Add directory initialization: the explicit command, the marker file, and the hook-side gate that exits instantly and recordlessly everywhere else.
- [x] 5.2 Add connection and disconnection: consented, reversible, idempotent registration of the persistent session hooks in each supported scaffold's user configuration.
- [x] 5.3 Add the hook entry in `gr`: session-start behaviour — ambient announcement only for the opening event, stale-question archival into the state root — and session-stop behaviour — hygiene, append-only retention, session reference, digest.
- [x] 5.4 Add intent binding: exactly one active confirmed intent binds by ID, version, and digest; zero, several, or unconfirmed yield an unbound record with the reason named.
- [x] 5.5 Enforce fail-quiet on every ambient path: internal errors end as clean exits toward the scaffold and bounded records in the state root.
- [x] 5.6 Confirm the wrapper lifecycle, canonical schemas, and target worktrees gain nothing.

## 6. Verify the implementation

- [x] 6.1 Map every scenario in the delta to a named deterministic test, driving the hook subcommand directly without a live scaffold.
- [x] 6.2 Pin the ambient announcement in both directions with the shared prohibited-content discipline.
- [x] 6.3 Prove zero writes and zero retained observations for sessions in unconnected directories.
- [x] 6.4 Prove stale-file archival, bound and unbound records, and retained bytes surviving worktree deletion.
- [x] 6.5 Prove a deliberately broken helper leaves a simulated session flow untouched, and that connection is idempotent and disconnection residue-free.
- [x] 6.6 Run `gofmt -l`, `go build ./...`, `go vet ./...`, `go test ./...`, and `go test -race ./...`.
- [x] 6.7 Run pinned telemetry-disabled strict OpenSpec validation across the change and every promoted spec.

## 7. Address review findings on the implementation

- [x] 7.1 Check initialization before reading the scaffold payload, so an unconnected session is never observed.
- [x] 7.2 Give question records a canonical identifier of their own and widen the answering reference so a background question can be answered without pretending a session is a run.
- [x] 7.3 Key records by occurrence so two identical questions keep separate, individually answerable records.
- [x] 7.4 Persist the record for an unreadable question: the escalation attempt is itself evidence.
- [x] 7.5 Apply bounded regular-file hygiene to the stale-question archival read, and clear what cannot be retained.
- [x] 7.6 Mark the installed handler and remove exactly it, so a mixed group or a lookalike command is never removed.
- [x] 7.7 Remove configuration files and directories that connection itself created.
- [x] 7.8 Require every event before reporting a connection present, and reconcile per event on reconnection.
- [x] 7.9 Treat an empty scaffold settings file as an empty configuration in planning, as the write path already does.
- [x] 7.10 Add a regression test per finding, state each new rule in the spec deltas, and re-run the full verification set.

## 8. Stop at the owner gates

- [x] 8.1 Obtain an explicit owner instruction before beginning implementation.
- [x] 8.2 Obtain a separate explicit owner instruction before committing.
- [x] 8.3 Obtain a separate explicit owner instruction before pushing and opening a pull request; merging remains ungranted.
- [x] 8.4 Obtain a separate explicit owner instruction before archiving this change and promoting the delta; re-read the promoted text immediately before promotion in case another change archived first. The promoted specs were confirmed unchanged since the deltas were authored, so both modified requirements applied to the text they were written against.
- [ ] 8.5 Obtain a separate explicit owner authorization before any live session or provider run is treated as evidence. One live session closes the recorded limits together: hook behaviour end to end, announcement delivery, and the reserved path's writability.
