# Design: Reviewer stall bound and integration isolation

## Context

Two full pre-PR reviews of `codex/admission-hardening-after-pr62` returned
nothing, each consuming its whole deadline (CTX-1). The reviewer had stopped, not
slowed: its session recorded no event for the remaining 23 and then 49 minutes
after one call to a machine-configured MCP server (CTX-2). That call reproduces
in seconds and never returns (CTX-3), while the identical call answers
immediately for another client (CTX-4). Approval policy (CTX-5), the read-only
sandbox (CTX-6), the diff, and the index are each excluded by experiment.

The provider defect is not this repository's to fix (NG-1). What is this
repository's is that a foreign block cost the full budget twice and left the
caller unable to say anything about the branch.

### Decision question

How should a review bound a reviewer that has stopped, and how should a caller
remove an integration the provider offers no switch to remove — without Goalrail
naming provider components, weakening the reviewer's capability boundary, or
growing a supervision layer?

### Constraints

- Goalrail names no provider integration, server, or tool (NG-2, CTX-8, CTX-9).
- The reviewer's read-only boundary is structural and stays that way; it was
  already defeated twice while it lived in an allowance (NG-3).
- The owner's machine configuration and credentials are untouched (NG-4, CTX-10).
- No new dependency, service, or second review implementation (NG-6).
- Liveness is already observable: reviewer output is captured incrementally
  (CTX-13). No new channel is needed.

### Options considered — stall bound

| Option | Description | Trade-off | Decision |
|---|---|---|---|
| A | Bound time since the last captured reviewer output; stop and classify as stalled | Needs a progress clock beside the deadline clock; uses a signal that already exists | **Selected.** It measures the thing that actually failed — silence — and needs no provider cooperation |
| B | Shorten the deadline | Cheap, but trades a stalled reviewer for a truncated healthy one, and diagnoses nothing. The second measured run doubled the deadline and learned nothing new | Rejected |
| C | Bound each provider tool call | Requires per-provider knowledge of tool boundaries Goalrail does not have, and would name provider internals | Rejected (NG-2, NG-6) |

### Options considered — isolation

| Option | Description | Trade-off | Decision |
|---|---|---|---|
| D | Caller names an integration; Goalrail renders the provider's own disable syntax for it | Goalrail knows the *shape* per provider, never a *name*; the argument set stays closed, so no caller input can reach the sandbox, model, or identity settings | **Selected.** It closes the capability question by construction rather than by validation |
| E | Arbitrary provider-argument passthrough | Maximally general, and the reviewer's read-only boundary becomes a string-filtering problem — exactly the allowance shape that was defeated twice before | Rejected (NG-3) |
| F | Goalrail keeps a list of known-hostile integrations | Ages immediately, and asserts about other people's machines | Rejected (NG-2) |
| G | Always isolate | Removes an asset by default and would silently change every existing review | Rejected |

### Independent critique

No external critique was obtained, and none is claimed. Option E would have
crossed the reviewer's capability boundary and required one; it was rejected for
that reason. Option D admits only a name into a fixed rendering, so the closed
boundary stays closed by construction rather than by inspection, and the residual
uncertainty is implementation-level rather than architectural. The reviewer that
would ordinarily provide a second pass is the one this change exists to unblock;
that circularity is recorded rather than worked around, and the pre-PR review of
this change itself remains a required gate before its own PR.

## Goals / Non-Goals

**Goals:**

- Stop a reviewer that has produced nothing for a bounded period, in minutes
  rather than at the deadline (OUT-1).
- Report stalling as its own outcome, carrying the reviewer's last observed
  activity (OUT-1, OUT-2).
- Let a caller remove one named integration from the reviewer's session without
  Goalrail naming any (OUT-3).
- Keep a stalled review out of the receipt record (SIG-5).

**Non-Goals:**

- Diagnosing, reporting on, or working around the provider defect (NG-1).
- Any allowance-shaped argument surface that could reopen the read-only boundary
  (NG-3).
- Detecting integrations, or reasoning about which are safe (NG-2).

## Decisions

### 1. Silence is the measured signal

The review already captures reviewer output incrementally. The progress bound is
time since the last captured byte, not time since start. A reviewer that streams
reasoning, tool lines, or diff content is making progress by definition; one that
emits nothing has stopped from the caller's side, which is the only side that can
observe it. The bound SHALL be shorter than the deadline and nameable by the
caller, and its default SHALL be well under the loop round's deadline so the
common case is caught long before the budget is spent.

### 2. Stalled is an outcome, not a deadline

An exceeded deadline and a stalled reviewer are different facts and SHALL be
reported as different outcomes. Both write no receipt and leave no reviewer
processes behind, reusing the existing teardown rather than adding a second one.
The stall report carries the tail of what was captured, because that tail is what
the caller would otherwise reconstruct from provider session files — the exact
work this investigation had to do by hand.

A stalled review reviewed nothing. It is never counted as completed, clean, or
passing, and the absence of a receipt is what enforces that downstream.

### 3. The caller names the integration; Goalrail renders the shape

The command accepts one repeatable option naming an integration to remove. Per
provider, Goalrail renders that name into the provider's own documented disable
syntax and adds nothing else. Goalrail therefore ships no provider component
name, no allow-list, and no detection. Because the rendering is fixed and the
only caller input is a name in a bounded character set, no caller input can reach
the sandbox mode, the model, the effort, or the reviewer identity: those remain
exactly where they are set today.

A provider with no such syntax refuses the option rather than accepting it and
silently doing nothing, because a silently ignored isolation request reads as
isolation that happened.

### 4. Isolation stays off by default

An integration is ordinarily an asset. Default-on would change every existing
review on every machine to remove capability the caller never asked to lose.

## Consumed Inputs

| Input | Source and trust | Accepted states/variants | Refused states | Mutation/race policy | Verification |
|---|---|---|---|---|---|
| Integration name to remove | The caller's own command line; trusted only as a bounded identifier | One or more names in a bounded identifier character set | Empty, oversized, whitespace-bearing, or containing characters that could alter argument structure; any name where the selected provider has no disable syntax | Read once at invocation; the rendered argument list is fixed before the reviewer starts and never re-read | Unit coverage over the rendered argument list per provider, asserting the sandbox, model, effort, and identity arguments are unchanged and that no caller input reaches them |
| Progress bound | The caller's own command line, or the default | A positive duration shorter than the effective deadline | Zero, negative, non-parsing, or greater than or equal to the deadline | Resolved once with the deadline before the reviewer starts | Unit coverage over resolution against explicit and defaulted deadlines |
| Reviewer output stream | The reviewer process; untrusted as content, trusted only as a liveness signal | Any bytes, on either captured stream | None — content is never interpreted, only its arrival time | Observed continuously while the reviewer runs; the last-activity instant is updated on arrival and never rewound | A test whose fake reviewer emits then falls silent, and one that emits slowly throughout, must be classified differently |

## Correlation and Evidence

- A stalled run reports the outcome, the elapsed time, the bound that was
  exceeded, and the tail of captured reviewer output. It writes no receipt, so
  nothing correlates it to a completed review.
- The reviewer's recorded identity, model, effort, and read-only boundary are
  unchanged by isolation; a receipt from an isolated review is indistinguishable
  from one without isolation except by the isolation record itself.
- The provider defect that motivated this change is retained in `context.md` with
  reproduction recipes, and is not asserted anywhere in shipped code or help.

## Measurement and Stop Conditions

Implementation is complete only when all of the following hold:

1. A reviewer that emits output and then falls silent is stopped within the
   progress bound and reported as stalled, with its last activity named.
2. A reviewer that emits output throughout is never stopped by the progress
   bound, and an overrun is still reported as a deadline overrun.
3. Neither outcome writes a receipt, and neither leaves reviewer processes behind.
4. The rendered reviewer argument list is asserted, per provider, to leave the
   sandbox, model, effort, and identity arguments untouched under every accepted
   and refused isolation input.
5. No provider integration, server, or tool name appears anywhere in the shipped
   repository outside this change's evidence.
6. A full pre-PR review of `codex/admission-hardening-after-pr62`, run with
   isolation, completes and writes a receipt.

Stop and reshape if the progress bound stops a reviewer that was working, if
isolation is found to alter any recorded reviewer property, or if the fixed
rendering cannot express a provider's disable syntax without naming a component.

## Risks / Trade-offs

- **A quiet-but-working reviewer.** A provider that buffers output could be
  stopped while working. The bound is caller-nameable and defaults well above any
  observed quiet period, and the failure is loud and immediate rather than a
  silent wrong verdict.
- **Isolation removes a real asset.** A caller who isolates loses tools the
  reviewer might have used well. It is off by default and named per run.
- **The name is the caller's knowledge.** A mistyped name isolates nothing.
  Refusing a name the provider cannot express is what keeps that visible.
- **Circular review.** This change's own pre-PR review needs the reviewer it
  repairs; the first run of it is therefore also the first test of it.

## Rollback

The stall bound and the isolation option are independent and separately
revertible in code. Reverting either restores exactly today's behavior: a
deadline-only bound, and a reviewer that inherits the machine's integrations. No
stored format, receipt field, or repository artifact changes, so nothing needs
migrating back. A rollback that reopens the stall costs budget again but loses no
recorded evidence.

## Open Questions

None material. The default progress bound's exact value follows from the
measured quiet periods during implementation, and the option's exact spelling
follows existing command conventions; neither changes the boundary, the outcome
set, or the stop conditions above.
