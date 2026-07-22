## Context

The foundation merged in `e1c9684` keeps real assignments blocked, but PR #3 review found three pre-activation defects. `Store.Append` protects a read/validate/append transaction only with a process-local mutex; `lineageOutcome` does not distinguish a terminal `SESSION_CONFLICT` from an ordinary unresolved link; and wording-only amendment validation preserves stable item IDs without preserving their source provenance. PR #4 review then found two incomplete edge cases in the initial fixes: a reader can check for the first evidence file before acquiring the shared lock, and a later resolution-exhaustion event can hide an earlier session conflict in the latest-event projection.

The slice affects one local storage adapter, one operator projection, and canonical validation. It must add no durable service or third-party dependency and must leave the real canary at `NO-GO_NOW`.

## Goals / Non-Goals

**Goals:**

- Make cooperating local Goalrail processes serialize evidence transactions for one file.
- Make one terminal root-session conflict produce the already-defined wrong-join hard stop.
- Preserve source evidence and per-item evidence-reference sets during wording-only edits.
- Reproduce each prior defect in a focused regression test and keep the full suite green.

**Non-Goals:**

- Activating or changing the 15-change canary, its metrics, or its frozen assignment rules.
- Replacing JSONL with SQLite, a workflow runtime, a service, or a remote coordination layer.
- Repairing an already-corrupt evidence file; verification continues to fail closed.
- Expanding platform support, archiving OpenSpec changes, or changing credentials and infrastructure.

## Decisions

### 1. Hold an OS advisory lock on a stable evidence sidecar for the complete transaction

**Decision question:** How should separate CLI processes serialize append-only evidence without adding a database or dependency?

**Verified facts:** The current `sync.Map` shares a mutex only inside one process. `O_APPEND` protects write placement but does not make the preceding chain read, validation, sequence selection, and digest calculation atomic. The current local development and intended rootless execution surfaces are macOS and Linux, where Go exposes `syscall.Flock`.

**Material unknown:** Remote filesystems can implement advisory locking differently. Remote or multi-host evidence storage is not part of this slice.

The evidence adapter will open a stable empty `<evidence>.lock` sidecar, validate that it is regular, and acquire an exclusive OS lock for `Append` or a shared lock for `ReadAll` and `Verify`. Readers acquire the shared sidecar lock before checking whether the evidence file exists, so a first append cannot be observed as an empty store while its exclusive transaction is in progress. `Append` holds the lock from the chain read through the evidence-file `Sync`. The data file is created only after event validation succeeds, preserving the existing rejection contract. The existing in-process mutex remains as a cheap first layer. Process exit releases lock ownership, so the ignored empty sidecar needs no stale-owner or cleanup protocol.

The OS primitive lives behind two small build-tagged files. macOS and Linux use blocking `flock`; other platforms return an explicit unsupported-lock error and never fall back to process-local safety. A later platform adapter can replace this boundary without changing canonical events or file format.

Alternatives:

- A cross-platform locking dependency would reduce platform-specific code, but adds a new dependency and contract for a two-function adapter.
- An exclusive sidecar lock file using create/delete would avoid OS-specific calls, but a crashed process can leave a stale lock and requires unsafe recovery heuristics.
- SQLite or another transactional store would provide compare-and-append semantics, but introduces a new durable subsystem before the local slice needs one.

This decision is reversible by replacing only the lock adapter and locked-file helper. Revisit it when Goalrail supports Windows, remote filesystems, or multi-host writers.

### 2. Classify only terminal `SESSION_CONFLICT` as a wrong join

`ExecutionLineage.UnlinkedReasonCode` already preserves `SESSION_CONFLICT`. The change projection will retain an internal sticky signal when any bounded lineage event for the change records that conflict. `lineageOutcome` will return `CanaryLineageWrong` from that signal before interpreting the latest lineage event, so a later retry or `RESOLUTION_ATTEMPT_EXHAUSTED` event cannot erase the wrong join. Missing identity, unsupported input, context conflicts, and resolution exhaustion without an earlier session conflict retain their current unresolved or pending behavior.

This keeps the change tied to the reviewed defect and does not invent new semantics for other reason codes.

### 3. Compare wording-only provenance by stable identity and value

Wording-only validation will compare source evidence as a map keyed by `SourceEvidenceID`, requiring the same IDs and identical kind, statement, and reference values. It will also compare each stable semantic item's `EvidenceRefs` as a set, so harmless ordering changes remain valid while adding, removing, or replacing a reference fails.

Semantic statement text and item order remain editable. Material changes continue to use a new candidate version and confirmation cycle. No new canonical fields are introduced.

## Correlation and Evidence

- The lineage event already contains change ID, run ID, root session, identity source, status, reason code, and bounded-attempt count; the fix changes only its report projection.
- Evidence locking changes transaction coordination, not the JSONL schema, digests, or accepted payload.
- Regression evidence stays in tests and OpenSpec task completion. No real canary event file is created.
- Wording-only validation preserves the original evidence objects and per-item references; it does not append runtime canary events.

## Measurement and Stop Conditions

- A synchronized subprocess test is the canary for the storage decision: both valid events must be retained once and `Verify` must pass.
- A reader started while the first writer owns the sidecar lock must wait and then observe the committed event.
- One `SESSION_CONFLICT` must yield `WrongJoins == 1`, set `HardStopSignals.WrongJoin`, and return `STOP`, including after a later resolution-exhaustion event.
- One ordinary unresolved lineage remains below the existing two-link hard-stop threshold.
- Source-evidence or item-reference mutation must fail; statement-only edits and reordering must pass.
- Stop implementation if process locking cannot be verified on both macOS compilation/runtime and Linux cross-compilation without adding an unapproved subsystem.

## Risks / Trade-offs

- **Advisory locks protect only cooperating Goalrail writers** → Keep complete-chain verification fail-closed for external writes or tampering.
- **Remote filesystem lock semantics may differ** → Restrict this slice to local files and revisit before remote or multi-host storage.
- **Unsupported platforms lose the previous permissive behavior** → Return an explicit error rather than silently accept corruptible concurrency.
- **A stricter wording-only validator can reject previously accepted edits** → Reject only provenance value/set changes; preserve text and order flexibility with regression tests.
- **Reason-code mapping could over-classify unrelated failures** → Match only the existing `SESSION_CONFLICT` constant.

## Rollback

Revert this change before real canary activation. The evidence file format and canonical domain remain unchanged, so rollback requires no migration. Existing valid evidence continues to verify. If a pre-fix file is already corrupt, rollback and this fix both continue to reject it rather than rewriting history.

## Open Questions

None for this local slice. Windows, remote filesystem, and multi-host locking are explicit revisit conditions rather than current requirements.
