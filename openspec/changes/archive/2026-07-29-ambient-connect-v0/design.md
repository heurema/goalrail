## Context

Intent Snapshot `intent-ambient-connect-v0` version 1 is confirmed.

The owner reversed the launch model: Goalrail attaches to the user's own
scaffold sessions in the background instead of launching providers itself. The
wrapper lifecycle stays as an internal benchmark tool. The escalation channel,
its hygiene and retention, the announcement discipline, the hook-response
renderer, and the intent-side answering record all exist; this change adds the
attachment.

Verified mechanism facts:

- Codex supports persistent user-level hooks in `config.toml`; the owner's own
  configuration already registers a `SessionStart` hook for an unrelated
  purpose, and the pinned CLI exposes the full event family including `Stop`
  and a session-start output that carries additional context for the agent.
- Claude Code registers hooks in the user settings file with the same event
  family.

## Goals / Non-Goals

**Goals:**

- Connect once, with consent, reversibly; no per-task Goalrail commands.
- Act only inside Goalrail-initialized directories; be invisible elsewhere.
- Announce the channel at session start; archive stale questions.
- Retain a session's question durably and bind it to the active intent.
- Close the answer loop through the existing intent gate.
- Never harm an ordinary session: background failure is a silent no-op.

**Non-Goals:**

- No run, receipt, or `blocked` status from background sessions.
- No daemon, control plane, dialogue, or automatic answering.
- No provider launch and no task composition by Goalrail.
- No change to the wrapper lifecycle, canonical schemas, kata, or benchmark.

## Decisions

### 1. The helper is the `gr` binary itself, invoked by the hook

No daemon and no separate binary. The hook command invokes `gr` with a
dedicated hook subcommand; the process does its bounded work and exits. This
keeps the installed surface to one executable, makes disconnection complete by
construction, and leaves nothing resident.

### 2. Initialization is an explicit marker file, not the directory

`.goalrail/` cannot itself mean opt-in: it already holds unrelated local
content (activation capsules), and the reserved question path lives inside it.
Initialization is therefore one explicit marker file created by an explicit
initialization command, and the hook's gate checks exactly that file. Creating
the marker is a deliberate act in the repository, mirroring how connection is
a deliberate act in the scaffold: two consents, two scopes — the scaffold and
the directory.

### 3. Fail-quiet is implemented as exit-zero-always in ambient paths

The scaffold treats a failing hook as something to surface or act on. The
ambient helper therefore never signals failure to the scaffold: every internal
error ends as a clean exit with no output beyond what the contract expects.
Errors worth keeping are recorded in the state root, where the owner can read
them later. This is the opposite of the wrapper's promoted fail-closed
posture, and both are stated side by side in the spec so neither erodes the
other.

### 4. Announcement reuses the discipline, not the string

The launch announcement's wording says a run ends as blocked — background
sessions have no runs. The ambient announcement is its own fixed constant with
the same pinned discipline: names the reserved path and conditions, no
provider names, no task hints, delivered only for the session-opening event.
The existing hook-response renderer already distinguishes opening from
resumption, clearing, and compaction, and is reused as-is.

### 5. Stale questions are archived at start, not judged

A question left by an earlier session must not be attributed to a new one —
this is the ambient analogue of the wrapper's prepare and pre-launch gates.
The start hook moves the file into the state root archive with its digest and
timestamp. It does not judge, bind, or discard: archival preserves evidence
and clears the path, and any binding decision stays with the stop-time record
that originally captured it, or with the owner.

### 6. Intent binding is exact-or-explicitly-unbound

The stop hook binds a question to the directory's active confirmed intent only
when exactly one is identifiable: one current change whose intent is
confirmed. Zero, several, or unconfirmed candidates yield an unbound record
with the reason named. Guessed attribution would poison the chain the record
exists to serve; an unbound question is still caught, retained, and answerable.

### 7. Clean-scope rules do not apply to background sessions

The wrapper's hedge rule — a question plus in-scope edits is a failure — exists
to stop a one-shot run from claiming both outcomes. An interactive session
edits legitimately throughout its life, and there is no frozen scope to
violate. Background mode therefore retains the question without judging the
worktree, and mints no status at all. The hedge rule continues to guard the
wrapper, where it belongs.

### 8. Corrections from automated review of the implementation

Nine findings, all confirmed against the code and all fixed. Two struck at
promises this change makes rather than at details.

- **The privacy boundary was ordered wrong.** The hook read and parsed the
  scaffold payload before checking the marker. A payload carries prompts,
  transcript paths, and authorization fields, so unrelated sessions were
  observed no matter what happened next. The check now precedes the read, and a
  test supplies a reader that fails if anything touches it.
- **The loop could not actually close.** A background session has no run ID by
  design, but the only answering parser required `<run-id> escalation <digest>`,
  so no recorded ambient identifier could produce a valid answering version
  without pretending a session was a run. Question records now carry a canonical
  identifier of their own, and the answering field is a resolved reference —
  a blocked run or a question record — rather than a run ID. This widens the
  promoted `intent-snapshot` requirement, so it travels as a delta.
- **Identical questions collided.** Two sessions writing byte-identical
  questions produced the same record path; the second write was rejected and the
  fail-quiet hook swallowed the loss. Records are now keyed by occurrence while
  the payload still deduplicates.
- **Unreadable questions vanished.** The invalid record was constructed and
  never persisted, so an escalation attempt against an unreadable artifact left
  no evidence at all. It is persisted now.
- **Archival bypassed its own hygiene.** The stale-question read was unbounded
  and followed links, so it could block startup on a FIFO or copy bytes from
  outside the repository into Goalrail state — precisely when a stale artifact
  exists. It uses the bounded regular-file reader, and clears what it cannot
  retain.
- **Disconnection could remove foreign configuration.** A group mixing our
  handler with someone else's was dropped whole, and any command merely ending
  in `gr hook` matched. The installed handler now carries a managed marker, and
  removal filters that exact handler.
- **Disconnection left files connection created.** A configuration file and
  directory created solely for the registration survived removal, contradicting
  the residue-free contract. Both are removed when nothing else remains.
- **A partial registration read as complete.** With one event hand-removed, the
  connection reported itself present and the consented command could never
  restore the missing half. Presence now requires every event, and connection
  reconciles per event.
- **An empty settings file blocked connection.** Planning applied a stricter
  parse than the write path, which already treats empty content as an empty
  configuration.

The confirmed intent is unmodified: every correction makes the shipped
behaviour honest against outcomes version 1 already states.

## Correlation and Evidence

- **Work identity:** confirmed intent `intent-ambient-connect-v0` version 1
  and Context Pack `context-ambient-connect-v0` version 1.
- **Specification evidence:** one new `ambient-connect` capability with five
  `ADDED` requirements; `local-run` untouched.
- **Implementation evidence:** each scenario maps to a deterministic test; the
  gate, archival, retention, binding, and fail-quiet paths are all testable
  without a live scaffold by invoking the hook subcommand directly.
- **Ownership:** `ambient-connect` owns attachment, the gate, ambient
  announcement, question records, and the fail-quiet posture. `local-run`
  keeps the wrapper lifecycle. `intent-snapshot` keeps the answering record.
- **Failure handling:** ambient-path errors are silent no-ops toward the
  scaffold and bounded records in the state root; connection and
  initialization commands, being explicit user acts, fail loudly as ordinary
  commands do.

## Measurement and Stop Conditions

- Strict pinned OpenSpec validation passes for the change and every promoted
  spec.
- Every scenario maps to a named deterministic test, including: zero writes in
  unconnected directories; announcement only on the opening event; stale-file
  archival; bound and unbound records; retained bytes surviving worktree
  deletion; a deliberately broken helper leaving a simulated session flow
  untouched; connection idempotence and residue-free disconnection.
- The ambient announcement constant passes the same two-direction pinning
  tests as the launch announcement.
- Stop and reshape if attachment would require a daemon, a scaffold fork, a
  canonical schema change, or any observation outside initialized directories.

## Risks / Trade-offs

- **[Hook behaviour unobserved end to end]** → Established from configuration
  evidence and published schemas; a live session under a separate gate closes
  it, together with the sandbox-writability question. Recorded, not hidden.
- **[Fail-quiet hides real breakage]** → Deliberate: the alternative breaks
  user sessions. Balanced by recording internal errors in the state root where
  the owner can read them; silence toward the scaffold is not silence toward
  the owner.
- **[Editing user scaffold configuration is sensitive]** → Only inside the
  explicit consented connection command, reversible, idempotent, and covered
  by residue tests. Codex additionally tracks hook trust; connection surfaces
  whatever consent flow the scaffold itself imposes rather than bypassing it.
- **[Two consents may confuse]** → Connection (scaffold-level) and
  initialization (directory-level) are distinct deliberate acts. Documented
  plainly; the alternative — one act implying the other — would either observe
  unconnected directories or initialize repositories implicitly.
- **[Unbound questions may accumulate]** → An unbound record names why it is
  unbound; accumulation is visible evidence of missing intent hygiene in the
  directory, not silent loss.
- **[Ambient and wrapper announcements may drift]** → Both are fixed constants
  under identical pinned test discipline; a shared test helper keeps the
  prohibited-content list common to both.

## Rollback

Delete the change directory before implementation; revert the implementation
commits after. Disconnection removes scaffold registration; removing the
marker file de-initializes a directory. No canonical schema, stored wrapper
evidence, or promoted spec changes until a separate archival step promotes the
delta. Question records already retained remain valid evidence and are not
rewritten by rollback.

## Open Questions

None.
