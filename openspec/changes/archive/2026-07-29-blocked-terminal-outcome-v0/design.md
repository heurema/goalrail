## Context

Intent Snapshot `intent-blocked-terminal-outcome-v0` version 1 is confirmed. It
moves an escalation channel that was measured inside a kata into the harness,
where the outcome belongs: a run can end because the work item cannot be done as
specified, and that ending must be a recorded terminal state rather than prose.

The design in this document is v2, produced after a five-lens review (simplicity,
state machine, gaming, contract evolution, product claim) reshaped the original
brief. The review is retained as `evidence/design-review-v2.md`; the superseded
brief is retained as `evidence/design-brief-v1.md` for provenance only.

Four repository facts constrain the shape:

- `validateTerminalReceipt` whitelists exactly five terminal statuses
  (`internal/localrun/service.go:660`) and `TerminalReceipt`
  (`internal/localrun/types.go:98`) carries neither a schema identifier nor an
  intent reference. Any new status breaks existing readers regardless of how it
  is introduced.
- `DecodeWorkSpec` rejects unknown fields and the canonical WorkSpec is
  digest-bound (`internal/domain/work_spec.go`), so a new WorkSpec field forks
  `goalrail.work-spec/v0` and threatens frozen digests.
- Worktree observation already includes ignored files
  (`git ls-files --others --ignored --exclude-standard`,
  `internal/localrun/git_observer.go:93`), so a reserved path under an ignored
  directory is still observed. The channel cannot go dark because of a
  `.gitignore` entry.
- `Start` already writes eager terminal receipts through `failureReceipt`
  (`internal/localrun/service.go:452`), and preparation freezes a
  content-addressed baseline observation of the whole worktree
  (`internal/localrun/service.go:161`). Both mechanisms this change needs already
  exist.

## Goals / Non-Goals

**Goals:**

- Make "cannot be done as specified" an expressible terminal outcome.
- Retain the question as append-only evidence that outlives the worktree file.
- Close the gaming paths by construction rather than by convention.
- Break receipt readers once, explicitly, with a version, instead of silently.
- Make the question-to-decision chain followable from recorded identifiers.

**Non-Goals:**

- No mid-run dialogue, control plane, resume, or background continuation.
- No WorkSpec schema change and no digest impact on frozen WorkSpecs.
- No semantic validation of the payload inside Goalrail.
- No recurrence registry and no prevention claim.
- No implementation in this change; planning stops at the owner gate.

## Decisions

### 1. A first-class `blocked` status with an explicitly versioned receipt

The owner selected variant A over reusing `verification_incomplete` with a
reason code. The contract lens showed the "v0-additive" framing was fiction: the
status whitelist at `service.go:660` breaks every existing reader on any new
status, and the receipt has no schema field to negotiate with. Since the break
is unavoidable, it happens once, deliberately, together with the version that
makes it legible — `goalrail.terminal-receipt/v1`.

Variant B avoided today's break but made owner triage of blocked runs
second-class and deferred a first-class status into a second migration. The
rejection is recorded, not re-litigated.

Receipts stored without a schema identifier remain readable as the unversioned
predecessor. They are not rewritten: rewriting merged evidence to fit a newer
schema is precisely what append-only retention forbids.

### 2. A reserved constant path, not a WorkSpec field

The original brief proposed an escalation-path field on the WorkSpec. Four lenses
rejected it. A path "outside the writable paths" would be unwritable inside the
provider sandbox, so the channel would be dead on arrival and agents would return
to guessing; and any new field forks the canonical schema.

A constant path fixed by Goalrail needs no WorkSpec change at all, so
`goalrail.work-spec/v0` stays byte-identical. The premise of the brief — that
this is a WorkSpec contract change — was false; the contract change is confined
to the receipt.

Two facts verified for the concrete path: observation includes ignored files, so
an ignored directory does not hide the artifact; and the `.goalrail/` namespace
in a repository already holds unrelated local content (dogfood capsules appear
there in archived evidence). The preparation gate therefore applies to the exact
reserved file, never to its directory.

### 3. Eager retention at provider observation, artifact inside the delta

Excluding the artifact from the worktree delta was the worst part of the brief and
was rejected unanimously. An excluded path is the only unauditable write channel
in the whole contract: substitution is invisible, and `rm <artifact> && gr finish`
would have yielded `passed` — the receipt would outlive the evidence it certifies.

Instead the artifact stays in the observed changed paths and its exact bytes are
copied into the append-only run store at provider observation, mirroring the
existing `failureReceipt` path. Retained bytes are authoritative for the run;
deleting the worktree file afterwards changes nothing.

The artifact is excluded from one place only: scope violations. `CompareWorktrees`
marks every changed path outside the declared scope as a violation
(`git_observer.go:145`), which would turn every escalation into `failed`. The
reserved path is therefore never a scope violation and never counts as an in-scope
edit — including when the declared scope is broad enough to contain it, which
`pathInScope` would otherwise satisfy for a scope of `.`.

### 4. One observation decides both the artifact and the delta

The state-machine lens found a TOCTOU gap between reading the artifact and writing
the receipt. Today the happy path observes the worktree only in `Finish`
(`service.go:373`), while `failureReceipt` observes in `Start`.

`Start` will therefore perform one worktree observation after provider
observation, retain the artifact from that observation, and persist the
observation itself. `Finish` uses the persisted observation to compute the delta
rather than observing again, so a single snapshot decides both facts. A run whose
persisted observation is absent — only possible for state predating this change —
falls back to observing at finish.

This moves the moment the delta is fixed from finish-time to
provider-observation-time. That is a behaviour change and is accepted
deliberately: edits made between the provider returning and the operator running
`gr finish` are not part of what the run did, and counting them was never correct.

### 5. Hygiene only, on the existing retained-text rules

Goalrail never runs checks, so it does not grade the payload. It validates that
the bytes are safe to retain, using the rules already applied to retained
WorkSpec text (`validateRetainedText`, `internal/domain/work_spec.go:302`):
non-empty, valid UTF-8, no unsupported control characters, no secret-shaped
content, within a bound. The escalation bound is its own constant, sized for a
question rather than reusing `MaxTextBytes` (1 KiB), which is too small.

Reading is subject to the same descriptor-level guarantees preparation already
uses for bounded artifacts: reject a non-regular file, do not follow a
substituted symbolic link, do not block on a substituted pipe.

The payload's meaning belongs to `goalrail.escalation/v0`, validated downstream.
One schema family, two responsibilities, no semantic parser inside Goalrail.

### 6. State rules that close the gaming paths by construction

- **Pre-existing artifact fails preparation.** Otherwise a stale or foreign
  question silently attaches itself to a new run and is certified by it.
- **Artifact plus in-scope edits yields `failed`, not `blocked`.** Otherwise an
  agent hedges — ships a patch and a question, and takes whichever sticks.
- **`unlinked`, `denied`, and `launch_failed` win.** An unattributable session
  must not mint an authoritative question; the digest is still recorded as
  evidence.
- **A failing check keeps `failed`.** This is the resolution of the panel's only
  disagreement. The gaming lens showed that forcing checks to unavailable would
  make `blocked` an evidence-destruction channel: an agent with red checks
  converts `failed` into `blocked` and the failures evaporate. Refusing `finish`
  outright, as the state-machine lens proposed, creates a dead end. Recording
  verbatim preserves evidence in both directions and fabricates nothing, and
  keeping an explicit failure at `failed` removes the incentive entirely.
- **Never `passed` while a retained artifact exists**, and `blocked` is never a
  per-check result state. The `--result` grammar is unchanged.

### 7. A followable chain, not a prevention claim

"The same conflict never silently recurs" is not delivered by this design: nothing
stops a new WorkSpec from referencing a pre-answer intent version, and a registry
of answered questions is new scope the workspace contract forbids. The claim is
cut.

What is delivered is a chain that is machine-followable, which it is not today:
the receipt gains the frozen intent reference (the receipt carries no intent at
all right now), and an answering intent version may record the run identifier,
the escalation digest, and a disposition. `spurious` and `withdrawn` exist so a
low-value escalation is visible and countable — answering both the DoS finding
(questions cost the owner attention) and its inverse (a culture that punishes
asking).

### 8. Corrections from automated review of the implementation

An automated review of the first implementation found six defects, all
confirmed against the code and all fixed. Two of them broke properties this
design had claimed as closed by construction, which is worth recording plainly:

- **The single-observation guarantee was not enforced.** The observation hashes
  the artifact and retention read it a second time; nothing compared the two, so
  a file changed between them would bind a receipt to bytes the delta never saw.
  Retention now compares its read against the observed digest and records a
  mismatch as invalid rather than binding it.
- **`passed` was still reachable.** The observation was persisted before
  retention, so a retention failure left an observation naming the reserved path
  with no record beside it, and Finish read that as "no question". Retention now
  precedes the observation write, and Finish fails closed when an observed
  question has no retained record.
- **The preparation gate had a window.** A prepared run is reused rather than
  re-observed and can be started much later, so an artifact created between
  preparation and launch would be attributed to a provider that had not run yet.
  Start now gates the reserved path before launching.
- **The reserved path had no boundary check.** A `.goalrail` symlink pointing
  outside the repository would let a provider write the question where
  observation never sees it — the silent-dead-channel failure this design exists
  to avoid. The reserved path now gets the same ancestor-boundary treatment as a
  declared scoped path.
- **The promised chain could vanish.** A v1 receipt with an absent or malformed
  intent reference still validated. Validation now requires a usable reference
  from v1 onwards, keeping the relaxed path only for receipts that predate the
  identifier.
- **An answering record could be backdated.** Nothing stopped a wording-only
  amendment from adding or changing `ResolvedEscalation` on an already confirmed
  version. Wording-only amendments must now preserve it exactly.

## Correlation and Evidence

- **Work identity:** confirmed intent `intent-blocked-terminal-outcome-v0`
  version 1 and Context Pack `context-blocked-terminal-outcome-v0` version 1.
- **Specification evidence:** one `ADDED` and two `MODIFIED` requirements for
  `local-run`, one `ADDED` requirement for `intent-snapshot`, and one new
  `escalation-question` capability with two requirements.
- **Measurement evidence:** the 12-run A/B, the 3-run quiet-channel retest, and
  the real-clone dogfood receipt, retained in
  `katas/account-cleanup-v1/evidence/` on the authoring branch.
- **Ownership:** `local-run` owns the terminal outcome, retention, and receipt.
  `intent-snapshot` owns the answering record. `escalation-question` owns the
  payload shape. `work-spec` is untouched and is protected by a frozen fixture
  digest regression test rather than by a new requirement.
- **Failure handling:** every rejection class is terminal for its stage.
  Preparation rejections leave no prepared state, run ID, launch claim, or
  provider invocation. A retention failure at provider observation must fail the
  run rather than emit a receipt referencing bytes that were never retained.

## Measurement and Stop Conditions

- Strict pinned OpenSpec validation reports this change and every promoted spec
  as valid.
- Every scenario in every delta maps to a named deterministic test.
- A checked-in frozen `goalrail.work-spec/v0` fixture keeps its digest
  byte-for-byte, and an existing schema-less receipt still validates.
- No input combination yields `passed` while a retained artifact exists, proven
  by test rather than asserted.
- The gaming tests fail against an implementation without the corresponding rule:
  pre-existing artifact, artifact plus edits, precedence, verbatim results.
- Stop and reshape if the reserved path cannot be written inside the provider
  sandbox for the supported adapter, if a scenario cannot be mapped to a test, or
  if any rule would require a WorkSpec field after all.

## Risks / Trade-offs

- **[The reserved path is unwritable in the provider sandbox]** → This killed the
  brief's original design and is the one risk that can still kill this one. It
  must be verified against the supported adapter's writable-path behaviour before
  implementation begins, and it is a stop condition above, not a late discovery.
- **[Introducing a new status breaks existing receipt readers]** → Accepted and
  deliberate; it is why the schema version is introduced in the same change.
  Schema-less receipts stay readable, so only forward readers change.
- **[Moving the delta observation to provider-observation time changes existing
  behaviour]** → Accepted; the previous timing counted post-run operator edits as
  part of the run. Existing tests that observe at finish will need updating, and
  that update is itself the evidence that the timing moved.
- **[Retention at provider observation adds a failure mode to `Start`]** → A run
  that cannot retain its artifact must fail rather than proceed, which converts a
  disk error into a terminal run. Preferred over a receipt that references bytes
  that do not exist.
- **[Excluding the reserved path from scope violations creates one privileged
  path]** → Bounded by construction: the path is constant, is still observed and
  reported as changed, is hygiene-checked, is retained append-only, and blocks
  preparation when pre-populated. It is privileged in exactly one dimension.
- **[Hygiene rejects a legitimate question]** → A rejected artifact yields
  `failed` with an explicit reason and is still retained, so the question is not
  lost and the reason is inspectable.
- **[`spurious` becomes a punishment channel]** → The disposition is recorded on
  the owner's side and is countable in both directions; the design deliberately
  records it rather than leaving low-value escalation invisible.
- **[Modifying two promoted requirements rewrites their intent annotations]** →
  Follows the existing convention: the `Intent IDs` line moves to this change's
  intent, and the prior annotation stays readable in the archived change that
  introduced it.
- **[Another change archives against `local-run` first]** → Deltas apply at
  archive time; whichever change archives second must re-read the promoted text
  before promotion.

## Retained Target Architecture Hypotheses

Not part of this slice, and deliberately not designed for here: deterministic
citation grounding, where every source citation in a question must byte-match the
file at the frozen `BaseRevision` through a read-only lookup. It would kill
fabricated questions without violating "Goalrail runs no checks", and it is a
second iteration. Likewise a control plane, mid-run correction, and any registry
of answered questions remain out of scope.

## Rollback

Delete the change directory. Nothing else is affected until a separate owner gate
authorizes implementation, and nothing is promoted until a separate archival step.
Once implemented, rollback is reverting the implementation commits: the receipt
version is additive to readers that ignore unknown fields, no stored evidence is
rewritten, and no WorkSpec digest moves. Receipts already written with the new
status remain valid evidence of what happened and are not rewritten by a
rollback.

## Open Questions

None.
