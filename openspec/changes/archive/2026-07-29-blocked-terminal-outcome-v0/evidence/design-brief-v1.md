# Design brief: "blocked" as a terminal outcome in Goalrail

Status: proposal under review. Nothing is implemented yet.

## Context (verified facts, do not re-derive)

Goalrail is a harness around coding agents. Its local-run contract today:

- A **WorkSpec** is frozen before launch: task, exact writable paths, exact
  checks (argv), stop conditions (up to 32), posture. Digest-bound.
  Schema: `goalrail.work-spec/v0`, see
  `/Users/vi/personal/heurema/goalrail/internal/domain/work_spec.go`.
- A run is **one-shot**: one launch claim, one provider session, no retry, no
  fallback. Default `gr start` is fail-closed (`ACTIVATION_REQUIRED`).
- Run states (`/Users/vi/personal/heurema/goalrail/internal/localrun/types.go`):
  `prepared`, `launch_attempted`, `awaiting_verification`, `launch_failed`,
  `unlinked`, `launch_attempted_unknown`, `passed`, `failed`,
  `verification_incomplete`.
- The **TerminalReceipt** (same file) records: workspec identity/digest, run id,
  adapter, root session ref or unlinked reason, base revision, terminal HEAD,
  effective paths, checks + explicit check results (`pass|fail|unavailable`),
  status, reasons, worktree delta (content-addressed, dirty-baseline aware),
  timestamps. Receipts are append-only; `gr finish` records explicit check
  results; Goalrail never runs checks itself and performs no follow-on actions.
- There is **no control plane**: no message bus for live correction, no mid-run
  owner interaction. The workspace contract explicitly defers those. Provider
  sandbox/approvals (e.g. Codex) remain the enforcement boundary.
- Intent lifecycle: confirmed Intent Snapshots are versioned artifacts; a
  material amendment creates a new version (never rewrite). WorkSpec references
  the confirmed intent by id/version/digest.

## Motivating evidence

A/B measurement on a kata whose two normative docs conflict (12 runs, Opus 5 +
Fable 5, plus 3 retest runs and 1 real-clone dogfood run):

- Channel to return a structured question present → 6/6 agents asked, 0 guessed.
- Channel absent → 0/6 asked, 6/6 guessed (all shipped a patch; the strongest
  agents documented the conflict in comments/prose and shipped anyway; one wrote
  the question into a dead file knowing it was unreachable).
- Retest with a much quieter, generalized channel ("blocked return") → 3/3
  still found and used it.

Evidence dir: `/Users/vi/personal/heurema/baseline-kata-authoring/katas/account-cleanup-v1`
(README.md + evidence/).

Conclusion drawn: frontier agents usually KNOW the right behaviour; the
determinant is whether the environment makes the right action expressible and
machine-accepted. Prose caveats do not block anything; a merge lands anyway.

## The proposal under review

Add an escalation outcome to Goalrail's local-run contract:

1. New terminal run state **`blocked`** alongside `passed`/`failed`.
2. WorkSpec declares an **escalation artifact path** (e.g. `blocked.md` or
   similar), OUTSIDE the writable paths, excluded from the worktree delta.
   The artifact carries a structured question: sources, a concrete separating
   example, per-source interpretations, one exact question. (Machine-checkable
   shape; the kata oracle already validates an equivalent schema by executing
   both documented readings against the example.)
3. `gr finish` REFUSES to record `passed` when the escalation artifact exists /
   the run is blocked. Check results stay `unavailable` (never fabricated).
4. The receipt records the blocked status + question reference (hash), bound to
   the frozen WorkSpec digest.
5. The owner's answer does NOT resume the run. It creates a NEW intent version
   (versioned artifact), hence a new WorkSpec + new one-shot run. No mid-run
   dialogue, no session resume, no control plane.

Product claim chosen (deliberately): NOT "agents start asking" (a prompt does
that too) but "the question and the answer become verifiable, versioned
artifacts bound to a frozen WorkSpec; a run cannot be closed as done while the
decision is pending; the decision persists so the same conflict never silently
recurs."

Known cost: this changes the WorkSpec and receipt schemas → a contract change
(needs intent + OpenSpec change per project policy; canonical schemas are
provider-neutral and currently v0).

## Your job

Review this proposal from YOUR assigned lens (given in your prompt). Read the
actual code paths cited above (read-only!) before judging. Be concrete and
adversarial; "looks fine" is a failed review. Judge:

- Is this the SIMPLEST design that delivers the product claim? What can be cut?
- Is it RELIABLE? Where does it break, race, or get gamed?
- What would you change before an intent is written?

Constraints you must respect (rejecting them = rejecting the project, not the
design): one-shot runs, fail-closed defaults, no control plane / mid-run
dialogue, append-only evidence, provider-neutral canonical schemas, versioned
intents, no background actions.
