## Context

Fourteen defects in one day, all found after their pull requests were public
(CTX-1). Grouped by cause they collapse to six classes, two of which crossed a
three-occurrence threshold on the first day of counting (CTX-2). The reviewer
was never the weak part — it ran last.

The workspace contract already requires a bounded pre-pull-request adversarial
review in a fresh context, cross-provider, with every finding dispositioned
(CTX-9). That policy is prose today, and prose disciplines are precisely what
the ratchet says lag. This change is what makes it executable.

Two measurements shape the design. `codex review --base <branch>` and
`claude -p` are installed, non-interactive, and subscription-legal, but
`codex review` emits no machine-readable output and neither CLI can be pinned
(CTX-3, CTX-4, CTX-9-evidence). And a session's own scaffold is visible in its
environment (CTX-13) — measured live, including the awkward case where a Claude
Code session also carries a Codex companion variable.

## Goals / Non-Goals

Goals, from the confirmed intent (v5; OUT-9 adds the stated reviewer model — the list below predates v4 and the v4 decisions further down supersede where they differ): run the review inside the author's loop
with no question on the ordinary path (OUT-1); make the budget gate the only
other refusal (OUT-2); close findings before the pull request opens under a
declared ceiling (OUT-3); leave the user alone except for a divergence or a real
intent hole (OUT-4); bind the receipt to what was reviewed (OUT-5); report
staleness that ends by itself (OUT-6); keep reviewer instructions as repository
content (OUT-7).

Non-goals that constrain the implementation rather than merely bounding scope:
no parsing of review text (NG-1), no mechanical gate (NG-2), no vendoring or
pinning of vendor CLIs (NG-3), no disposition or journal inside Goalrail (NG-4),
no prompt anywhere (NG-5), no orchestration of the loop (NG-6), no scope growth
through the question channel (NG-7).

## Decisions

**Goalrail is the review step. The loop belongs to the caller.**

The calling agent runs review → read → fix → re-review, declares its ceiling
before the first round, and stops on the ceiling or on nothing-material. Goalrail
performs one review and writes one receipt. This is what keeps NG-1 and NG-6
true at once: the thing that decides "nothing material remains" is the agent
reading the report, never Goalrail reading it.

The default instructions ask the reviewer to end its report with a fixed marker
line stating whether anything material remains. That marker is the reviewer's
own statement, stored verbatim like the rest of the report; Goalrail neither
requires it, validates it, nor acts on it. It exists so the calling agent has a
cheap thing to look for, and a reviewer that omits it produces a valid receipt.

**Authorship is detected. (v4 supersedes the refusal below: an undecidable author is recorded as `unknown` and refuses nothing.)**

`CLAUDECODE` identifies a Claude Code session; a Codex session carries its own
`CODEX_*` session marker. The rule is deliberately narrow: only a provider's
*primary session marker* counts, because the environment measured on this
machine carried `CODEX_COMPANION_SESSION_ID` inside a Claude Code session
(CTX-13). A companion variable means another tool is reachable, not that it
authored anything. Treating any `CODEX_*` variable as evidence would have made
the very first real invocation pick the author as its own reviewer — failing
silently, which is the worst available failure for this feature.

Both primary markers present, or neither, refuses and names the override. The
override always wins. This is the promoted `input-space` discipline applied to
the first thing the command does: enumerate the legal environments — one
provider, two providers, none, override — and decide each.

**The reviewer runs through the vendor's own interface, unpinned and unwrapped.**

`codex review --base <ref>` with instructions on stdin; `claude -p` with the
instructions and a bounded tool surface. Minimum flags on purpose: every flag is
a surface that can drift, and the drift risk is accepted openly (NG-3) rather
than hidden behind a wrapper that would silently paper over a changed interface.
A non-zero exit or an unusable invocation surfaces to the caller as the
reviewer's own failure, and no receipt is written — a receipt for a review that
did not happen is worse than none.

**The budget gate is a command, not a path.**

Configuration names a command; non-zero exit refuses and is reported as the
reason. The personal usage guard this workspace uses becomes one line of local
configuration rather than a path compiled into a released binary. No gate
configured means no gate — turning absence into a refusal would reintroduce the
consent step OUT-2 removes.

**The receipt measures; it never reads.**

Base commit, head commit, canonical diff digest, reviewer identity, time, report
bytes, report digest. The diff is rendered canonically — no external diff
drivers, no colour, no textconv, full index — so the same range digests
identically anywhere, which is what makes the staleness comparison in OUT-6 mean
something. Nothing is extracted from the prose.

The receipt lives in the per-clone state root, not the repository. It is
evidence about one clone's branch at one moment; a receipt someone else receives
by pulling would assert a review of a diff they may not have.

**Staleness is a digest comparison, like the adoption advisory.**

The diagnosis recomputes the branch's present diff digest and compares. New
commits make it stale by themselves; a fresh review makes it current by itself.
No flag, no dismissal — the same self-terminating shape settled for the rules
advisory in `openspec-adoption-v0`, so the product speaks one language about
"this evidence no longer describes the thing".

**Instructions are committed, and therefore not under `.goalrail/`.**

Goalrail's own ignore rule excludes `.goalrail/` from version control, so
instructions cannot live there and remain shareable. They go to a dot-prefixed
file at the repository root, outside that excluded directory, materialized with
a default only when absent and never overwritten. Ratchet promotions are then
ordinary edits, reviewable in the same pull requests as the code they protect.

**v4: fresh mode is the default shape, and modes are recorded, not judged.**

The owner's reframing (SE-10) inverted a premise: most users have one provider,
so a design that refuses without a second one fails the ordinary case. The
mechanism was never the second vendor — it is the fresh session. Selection is
now: candidates are the supported providers, runnability (the executable
resolving) makes one eligible, the non-author is preferred, and the receipt
records mode and reason. The refusal survives only where nothing runs at all.

**v4: the refute trigger is behavioural, not evaluative.**

"Risky" required someone to judge riskiness — the author (biased) or the user
(whom the loop must not disturb). The trigger is now a fact the calling agent
already holds: findings exist and it is about to act on them. Goalrail neither
triggers nor reads; the refuter gets the report verbatim, which is transport,
not parsing.

**v4: incremental re-review, with its limit stated.**

Round cost was O(branch) per round, measured twice into owner stops. The next
round's range starts at the previous receipt's head; the receipt carries both
the round's range and the full branch digest, so staleness stays one comparison
and the chain proves cumulative coverage. Accepted limit, stated: a later fix
can break what an earlier round approved and the incremental diff will not show
it — the limit human incremental review carries. The full pass remains a flag.

**v4: the deadline bounds the tree.**

`exec.CommandContext` kills the direct child; the reviewers are wrappers whose
grandchildren hold the pipes, and a twenty-minute deadline was measured letting
a run reach fifty-three. The child gets its own process group, cancellation
signals the group, and a short wait-delay stops waiting on inherited pipes.

## Correlation and Evidence

The receipt is the correlation record: it ties a reviewer's report to an exact
base, head, and diff digest, so a later reader can prove which bytes were
reviewed. The diagnosis line derives from that receipt and from a recomputation,
never from memory.

Findings disposition stays at the workspace level (NG-4). Goalrail records that
a review of a specific diff happened and what it said; which findings were
accepted, rejected, or promoted is the workspace's journal, and a repository
tool asserting that would be claiming a decision it never saw.

## Measurement and Stop Conditions

Done when the five confirmed signals hold, measured on a real branch:

- one invocation with no flags from a Claude Code session selects Codex, runs,
  and receipts, with zero prompts and unchanged exit-status semantics (SIG-1);
- a failing gate refuses and names itself; no gate configured reviews (SIG-2);
- the diff digest recomputes from the receipt's own base and head, and the
  stored report is byte-identical (SIG-3);
- the diagnosis moves current → stale on a commit and stale → current on a fresh
  review, with no flag anywhere (SIG-4);
- the Baseline API extraction branch goes through review, in-loop fixes, and a
  current re-review receipt before its pull request opens, with zero user
  interruptions on the clean path (SIG-5).

Stop condition for the detection logic specifically: it is finished when every
enumerated environment — one primary marker, two, none, override — has a decided
outcome. It is not finished by growing a heuristic that guesses better; pressure
in that direction is the signal to stop and refuse instead.

## Risks / Trade-offs

- **Vendor CLI drift.** Neither reviewer is pinnable, so a changed interface
  breaks invocation. Accepted and stated (NG-3). Mitigated by using the minimum
  documented flags and by surfacing the vendor's own failure unchanged, so the
  cause is visible to whoever is standing there. Not mitigated by wrapping —
  a wrapper would convert a loud break into a quiet one.
- **Spending inside a loop.** The first Goalrail action that costs money without
  a separate human act. The gate is the control, and it is deliberately the
  user's command rather than our judgement of their budget.
- **A review nobody reads.** Goalrail cannot force the loop to act on findings,
  and NG-2 forbids making it a gate. The staleness line is the only pressure,
  and it is advisory by design. If the loop ignores reviews, this change buys
  nothing — that is a property of the loop's discipline, not something to fix
  here by adding a refusal.
- **Detection is environment-shaped and will meet environments we have not
  seen.** The refusal path is the mitigation, and the override is the escape.
  The failure mode we care about — silently choosing the author as reviewer — is
  the one the narrow primary-marker rule exists to prevent.

## Rollback

Additive: a new command, a new receipt in per-clone state, one diagnosis line,
one materialized repository file. Reverting the code restores prior behaviour;
existing receipts become inert files and a reverted binary ignores them. The
only repository artifact is the instructions file, which a user may delete or
keep as ordinary content.

## Open Questions

None blocking. One deliberately deferred and recorded so it is not mistaken for
an oversight: whether a review's findings should ever reach the workspace
ratchet journal automatically. Doing so would require a repository-scoped tool
to write workspace-scoped state and to decide what counts as a finding, which
crosses both NG-1 and NG-4. Today the loop's author carries findings across that
boundary by hand.
