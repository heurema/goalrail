# The loop reviews its own work before anyone sees it

## Why

Fourteen defects were found across two pull requests in a single day, every one
of them after the pull request was public, and two would have shipped wrong
behaviour. The reviewer was not the problem — it was running last. An author
cannot review their own artifact, so independent review has to be a step of the
loop rather than a stage that begins when the work is already exposed.

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
  An override wins; an undetectable author is recorded as `unknown` and refuses
  nothing. The single refusal is that no reviewer runs at all.
- A **refute** round is available in every mode on the caller's flag, under one
  stated rule: findings exist and the caller is about to act on them. A fresh
  session (the other provider where one exists) tries to refute the findings
  rather than add new ones; both reports are stored verbatim.
- A re-review is **incremental by default**: it covers the range since the
  previous receipt's head, so a round costs what the round is about. The
  receipt records both that range and the full branch digest, and the chain of
  receipts proves cumulative coverage. A full pass stays available by flag.
- Every review runs under a deadline that bounds the whole process tree, with
  the reasoning effort stated rather than inherited from interactive
  configuration.
- The only machine refusal besides that is a **configurable budget gate**: a
  command named in configuration, run first, whose non-zero exit stops the
  review and is reported as the reason. No personal budget path is built into
  the product.
- A **receipt** binds the review to what was reviewed: base and head commits,
  the digest of the reviewed diff, reviewer identity, time, and the reviewer's
  report stored as verbatim bytes with its own digest. Nothing in the receipt is
  derived by reading the report.
- The **diagnosis** gains one line: the current branch's review is current,
  stale, or absent, decided by recomputing the diff digest. New commits make it
  stale by themselves; a fresh review makes it current by itself. A fact, never
  a fault.
- **Reviewer instructions** live in a committed repository file, materialized
  with a default when absent. Promotions from the findings ratchet land there as
  ordinary edits.
- Goalrail supplies the review **step**; the loop that re-runs it after fixes,
  with its declared ceiling, belongs to the calling agent. Nothing here parses,
  scores, or blocks.
- Nothing existing is removed or repurposed, so nothing is breaking.

## Intent Coverage

*(recompiled against intent v5)*

| Change | Intent IDs | How it preserves the boundaries |
|---|---|---|
| One command reviews the branch with no question on the ordinary path | OUT-1, SIG-1 | Runs the vendor CLI as its vendor intends; introduces no prompt (NG-5) |
| Authorship inferred from the environment, refusal only when undecidable | OUT-1, CTX-13 | The refusal is the input-space discipline, not a consent step |
| Budget gate as a configurable command | OUT-2, SIG-2 | The only machine refusal; no personal infrastructure shipped, no human consent inside the loop |
| Findings return to the author in-loop and re-review runs to a declared ceiling | OUT-3 | Goalrail provides the step, the caller owns the loop and its ceiling (NG-6) |
| No question to the user except divergence or a real intent hole | OUT-4 | An improvement is recorded, never asked (NG-7) |
| Receipt bound to base, head and diff digest, report stored verbatim | OUT-5, SIG-3 | Every field is measured; none is read out of the prose (NG-1) |
| Diagnosis line, self-terminating by digest | OUT-6, SIG-4 | Reports a condition, never affects the verdict (NG-2) |
| Committed instructions file with a materialized default | OUT-7 | Ratchet promotions are ordinary repository edits; disposition stays at workspace level (NG-4) |
| Reviewer model stated per provider, never carried across providers | OUT-9 | Defaults only where measured evidence supports one; a provider without evidence keeps its own configuration (NG-3 in spirit: nothing is pinned for its own sake) |
| Vendor CLIs invoked, never wrapped or pinned | OUT-1 | Drift is stated as accepted, not hidden (NG-3) |

## Capabilities

**New Capabilities**

- `independent-review` — running an independent review of the current branch
  inside the author's loop, its refusals, and the receipt it leaves.

**Modified Capabilities**

- `harness-doctor` — a new requirement covers the review-state line and the
  condition under which it changes.

## Impact

- New package for the review step: author detection from the environment,
  reviewer selection, instruction assembly, invocation, receipt writing.
- `internal/ambient` already detects and reports scaffolds; reviewer selection
  reads that registry rather than introducing a second notion of what is
  installed.
- Process spawning already exists for `git` (`internal/ambient/scope.go`,
  `internal/localrun/git_observer.go`). This is the first spawn of a
  non-deterministic, metered process, which is why the budget gate and the
  verbatim receipt are part of the same change rather than later work.
- `internal/harness/doctor.go` — one more reported line, following the
  observability precedent for a fact that is not a fault.
- `internal/localrun` — the receipt belongs with per-clone state; the state root
  resolver already exists.
- Configuration gains the gate command and the instructions path. No new
  dependency: the module has none, and none is needed.

## Non-Goals

- No parsing, scoring, or interpretation of review text.
- No mechanical gate on commit, push, PR, or merge.
- No vendoring, pinning, or wrapping of the vendor CLIs, and no third-party
  review tool.
- No findings disposition or ratchet journal inside Goalrail.
- No interactive prompt in any Goalrail command.
- No orchestration of the re-review loop, and no unbounded loop.
- No scope growth through the question channel.
