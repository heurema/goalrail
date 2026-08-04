# The loop reviews its own work before anyone sees it

## Why

Fourteen defects were found across two pull requests in a single day, every one
of them after the pull request was public, and two would have shipped wrong
behaviour. The reviewer was not the problem — it was running last. An author
cannot review their own artifact, so independent review has to be a step of the
loop rather than a stage that begins when the work is already exposed.

The command now exists, but the open change cannot be closed by checking old
boxes. Its Baseline proof failed the success signal it was meant to establish,
the live Codex desktop marker and default-base behavior have drifted, and the
public no-terminal contract was never proved end to end. This reconciliation
keeps the failed run visible, repairs those seams, and proves the same primitive
on a replacement real branch before its pull request opens.

## What Changes

- A new command runs an independent review of the current branch **inside the
  author's loop**, without asking anyone. The mechanism is a fresh session that
  never sees the author's context; a different provider strengthens it but is
  not its precondition. One runnable provider → **fresh** (the same provider in
  a clean session — the ordinary case, because most users have one tool); two →
  **cross** (the non-author). A cross that falls back to fresh because the other
  CLI will not run is recorded with its reason, never silent.
- Authorship is detected, not demanded, matching primary session markers only —
  a companion variable of one tool inside another's session must not count.
  The supported current Codex desktop identity is part of that contract, not an
  implementation accident. An override wins; a genuinely absent or ambiguous
  author is recorded as `unknown` and refuses nothing. The only
  reviewer-selection refusal is that no reviewer runs at all.
- An explicit base always wins. With no `--base`, the command resolves an
  unambiguous repository default from local Git metadata, preferring the
  remote-tracking default over a same-named local branch. Ambiguous metadata
  fails before any reviewer runs and names `--base`; the command performs no
  fetch or hosting-service query.
- A **refute** round is available in every mode on the caller's flag, under one
  stated rule: findings exist and the caller is about to act on them. A fresh
  session (the other provider where one exists) receives the first report and
  the exact reviewed diff, then tries to refute the findings rather than add new
  ones; both reports are stored verbatim.
- A re-review is **incremental by default**: it covers the range since the
  previous receipt's head, so a round costs what the round is about. The
  receipt records both that range and the full branch digest, and the chain of
  receipts proves cumulative coverage. A full pass stays available by flag.
- Every review runs under a deadline that bounds the whole process tree, with
  the reasoning effort stated rather than inherited from interactive
  configuration.
- After valid repository inputs resolve, the only policy refusal besides
  reviewer availability is a **configurable budget gate**: a command named in
  configuration, run first, whose non-zero exit stops the review and is reported
  as the reason. No personal budget path is built into the product.
- A **receipt** binds the review to what was reviewed: base and head commits,
  the selected base ref, the digest of the reviewed range and full branch,
  reviewer identity, mode and reason, duration, execution settings, time, and
  every report stored as verbatim bytes with its own digest. Nothing in the
  receipt is derived by reading a report.
- The **diagnosis** gains one line: the current branch's review is current,
  stale, absent, or unreadable, decided by recomputing the diff digest. New
  commits make it stale by themselves; a fresh review makes it current by
  itself. A fact, never a fault. Initialization, review, and diagnosis retain
  parseable machine output and their unrelated exit semantics without a TTY.
- **Reviewer instructions** live in a committed repository file, materialized
  with a default when absent. Promotions from the findings ratchet land there as
  ordinary edits.
- Goalrail supplies the review **step**; the loop that re-runs it after fixes,
  with its declared ceiling, belongs to the calling agent. Nothing here parses,
  scores, or blocks, and no coordinator, finding normalizer, disposition system,
  or automated fix loop is added.
- The historical Baseline run remains failed evidence. The repair branch is the
  replacement real canary: before its pull request opens it declares the ceiling
  and stop condition and reaches a current receipt on its final head with zero
  clean-path interruptions.
- Nothing existing is removed or repurposed, so nothing is breaking.

## Intent Coverage

*(recompiled against confirmed intent v6)*

| Change | Intent IDs | How it preserves the boundaries |
|---|---|---|
| Fresh/cross selection recognizes the actual supported session identity | OUT-1, SIG-1 | Vendor CLIs remain unwrapped and no prompt is introduced (NG-3, NG-5) |
| Omitted-base resolution cannot silently select a stale local shadow | OUT-10, SIG-6 | Uses local Git metadata only; no fetch or hosting API (NG-9) |
| Budget gate as a configurable command after valid input resolution | OUT-2, SIG-2 | No personal infrastructure ships and no human consent step enters the loop |
| Incremental re-review and the replacement pre-PR canary | OUT-3, OUT-4, SIG-5 | Goalrail provides one step; the caller owns findings, fixes, ceiling, and stop (NG-4, NG-6, NG-8) |
| Refuter receives the first report and exact reviewed diff | OUT-8 | Both reports remain verbatim; Goalrail interprets neither and decides no escalation (NG-1, NG-7) |
| Receipt binds range, full branch, selection, settings, and reports | OUT-5, SIG-3 | Every field is measured; none is derived from review prose (NG-1) |
| Diagnosis and public commands preserve JSON/no-terminal semantics | OUT-6, SIG-4 | Review state is evidence, never a verdict or mechanical gate (NG-2, NG-5) |
| Committed instructions file with a materialized default | OUT-7 | Ratchet promotions are ordinary repository edits; disposition stays workspace-owned (NG-4) |
| Reviewer model is resolved per provider and invocation | OUT-9 | Defaults exist only where measured evidence supports one; vendor CLIs remain unpinned (NG-3) |

## Capabilities

**New Capabilities**

- `independent-review` — running an independent review of the current branch
  inside the author's loop, its refusals, and the receipt it leaves.

**Modified Capabilities**

- `harness-doctor` — a new requirement covers the review-state line and the
  condition under which it changes.

## Impact

- Existing review package: current session-marker detection, runnable-provider
  selection, instruction assembly, bounded invocation, refute transport, and
  receipt writing.
- Provider candidates remain Goalrail's supported scaffolds; executable
  resolution decides what can actually review. No configuration directory is
  mistaken for a runnable command.
- `cmd/gr/review.go` and local Git observation gain safe omitted-base resolution
  without network access; an explicit base remains authoritative.
- Process spawning already exists for `git` (`internal/ambient/scope.go`,
  `internal/localrun/git_observer.go`). This is the first spawn of a
  non-deterministic, metered process, which is why the budget gate and the
  verbatim receipt are part of the same change rather than later work.
- `internal/harness/doctor.go` and command tests — review state remains a fact,
  and initialization/review/diagnosis are proved with machine-readable output
  and no terminal.
- `internal/localrun` — the receipt belongs with per-clone state; the state root
  resolver already exists.
- The gate remains configuration; reviewer instructions remain committed
  repository content. No new dependency, service, or network call is needed.

## Non-Goals

- No parsing, scoring, or interpretation of review text.
- No mechanical gate on commit, push, PR, or merge.
- No vendoring, pinning, or wrapping of the vendor CLIs, and no third-party
  review tool.
- No findings disposition or ratchet journal inside Goalrail.
- No interactive prompt in any Goalrail command.
- No autonomous coordinator, finding normalization, dispositions, or automated
  fix loop; no orchestration of the re-review loop, and no unbounded loop.
- No scope growth through the question channel.
- No fetch or hosting API call to infer the default base.
