# Session handoff — 2026-07-29

Read this first, then `context.md` and `intent.md` in this directory.

## Exactly where we are

OpenSpec change `blocked-terminal-outcome-v0` exists with **2 of 6** artifacts:
`context.md` (Context Pack v1, sufficient, no material unknowns) and
`intent.md` (Intent Snapshot v1, **status: candidate**).

Strict validation reports this change as failing with "no deltas found". That is
**expected and correct** — `specs/` is blocked by `proposal`, and `proposal` may
not be written from a candidate intent. It is not a defect to fix.

Nothing is committed. Five untracked files sit in the worktree.

## The one next action

**Ask the owner to confirm Intent Snapshot `intent-blocked-terminal-outcome-v0`
version 1.** Present a plain-language owner-facing view first (Russian — the
working language of this project); the owner reads Russian and asked repeatedly
for simpler explanations, so avoid schema names and field lists in that view.

Silence, continuation, or absence of objection is not confirmation. Only after
an explicit yes: set status to `confirmed`, fill the Confirmation block, then
proceed `proposal` → `specs` → `design` → `tasks`. Implementation is a separate
gate after that.

## Locations

| what | where |
|---|---|
| this change | worktree `/Users/vi/personal/heurema/goalrail-blocked`, branch `codex/blocked-terminal-outcome-v0`, based on `origin/main` = `71426c0` |
| design brief (v1, superseded) | `evidence/design-brief-v1.md` |
| five-lens review + design v2 | `evidence/design-review-v2.md` — **this is the confirmed design**, not the brief |
| kata, participant repo | `/Users/vi/personal/heurema/katas/account-cleanup`, pushed to `heurema/account-cleanup` (**PRIVATE**), HEAD `455604b` |
| kata, author side | `/Users/vi/personal/heurema/baseline-kata-authoring`, branch `codex/kata-account-cleanup-authoring` @ `e923dff`, not pushed |
| measurement evidence | `katas/account-cleanup-v1/evidence/` in that branch: A/B, retest, private dogfood receipt |

## Design decisions already settled — do not re-litigate

Five reviewers (simplicity, state machine, gaming, contract evolution, product
claim) produced 41 findings. The consensus reshaped the design; the details are
in `evidence/design-review-v2.md`. Settled points:

- **Owner chose variant A**: first-class `blocked` terminal status plus explicit
  receipt versioning `goalrail.terminal-receipt/v1`. Variant B (reuse
  `verification_incomplete` + reason code) was rejected.
- **No WorkSpec change.** A reserved constant escalation path replaces the
  proposed WorkSpec field. `goalrail.work-spec/v0` stays byte-identical; frozen
  digests must be protected by a fixture regression test.
- **The artifact is retained append-only, eagerly**, in `Start` at provider
  observation, mirroring the existing `failureReceipt` path. It is NOT excluded
  from the worktree delta — that idea was the single worst part of the brief
  (`rm blocked.md && gr finish` would have yielded `passed`).
- **Goalrail treats the question as opaque bytes** with hygiene only (regular
  file, size cap, UTF-8, non-empty, secret-shape scan). No semantic validation
  inside Goalrail. The payload schema `goalrail.escalation/v0` is separate and
  validated downstream.
- **State rules**: pre-existing artifact fails `prepare`; artifact plus in-scope
  edits yields `failed`, not `blocked`; `unlinked`/`denied`/`launch_failed` take
  precedence over `blocked`; supplied check results are recorded **verbatim**
  (never forced to unavailable — that would make blocked an evidence-destruction
  channel); `blocked` is never a per-check result state and the `gr finish`
  result grammar is unchanged.
- **The recurrence claim was cut.** "The same conflict never silently recurs" is
  not delivered and must not be claimed. What is delivered: a machine-followable
  chain via a receipt-side intent reference and an intent-side `resolves` +
  `disposition` record.

## Why this change exists (one paragraph)

A 12-run A/B on a conflicting-requirements kata: with a machine-accepted question
channel, 6/6 agents asked; without it, 0/6 asked and 6/6 guessed — including
agents that documented the conflict in prose and shipped anyway. The channel was
built inside the kata, which was the wrong place: it made the kata solve itself
and proved nothing about the harness. This change moves the capability into
Goalrail, where it belongs.

## Constraints that bind the next session

One-shot runs. Fail-closed defaults. No control plane, no mid-run dialogue, no
resume — answering creates a new intent version and a new run. Append-only
evidence. Provider-neutral canonical schemas. Versioned intents; never rewrite a
confirmed version. No background actions. Ask before commit, push, PR, merge, or
any external effect.

Pinned OpenSpec CLI:
`OPENSPEC_TELEMETRY=0 npx --yes @fission-ai/openspec@1.6.0`.

## Environment notes

The canonical `goalrail` checkout is on `codex/dogfood-run-v0` @ `70b31fd` with
18 dirty paths of owner-owned work — **do not touch it**; `.goalrail/` is ignored
via `.git/info/exclude`. Several other worktrees exist from earlier work in this
session. Parallel Codex sessions have been active in `baseline`; branch from
`origin/main`, never from a local `main`, to stay clear of their unpushed work.

`heurema/session-key-rotation` was manually made private by the owner and must
not be touched. K-B and other new katas are not started.
