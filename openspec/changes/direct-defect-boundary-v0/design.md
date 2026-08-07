## Context

A change made outside the planning cycle rests on the author asserting one word
about their own work. The contract triggers the cycle on "significant", defines
it nowhere, and records no defect-class exception, so the exemption authors
already act on cannot be cited or disputed (CTX-1, CTX-2).

Version 1 of this change proposed to fix that with contract prose. Two
independent critiques refused it for the same reason from different directions:
an instruction surface may not define a second materiality policy (CTX-9), and
the mechanical part of the question is already decided by the admission verifier
(CTX-10). What survives is narrower and sharper — the verifier decides paths,
not order, and the one recorded instance of the failure is an ordering failure
that a fully active verifier would pass (CTX-11, CTX-12).

## Goals / Non-Goals

- **Goal.** Make the claim "this only restores what was already required" a
  bound, ordered record rather than an assertion.
- **Goal.** Let a machine decide the part that is decidable — ancestry and
  binding — and leave the reviewer the part that is not.
- **Non-goal.** A second materiality policy. What counts as material stays with
  the committed policy.
- **Non-goal.** A machine judgement of semantic equivalence.
- **Non-goal.** Adoption of this repository or activation of shared admission.

## Decisions

**The exception model, not a new capability.** `work-unit-lineage` already makes
every exemption, break-glass and bootstrap path an immutable, digest-identified
event carrying a categorical class, actor, reason, exact scope and owner
decision, visible after closure and never normalized into ordinary validity. A
restoration claim is the same shape. Adding a class costs one requirement;
adding a capability would create the parallel authority the governance contract
forbids.

**Ancestry, not a timestamp.** The events already carry an observation time, and
that is exactly what does not help: a timestamp is written by the actor making
the claim, and the `#85` sequence is a well-ordered set of honest timestamps
that still records the code first. Commit ancestry is a property of the history
being admitted rather than of the record describing it, and it is what the
verifier already has in a frozen range.

**Ancestry needs parents, which the frozen range does not carry.** Recorded
during implementation, because the first version of this design asserted the
question was decidable from what the verifier already receives and that was not
checked. The range carries an ordered commit list and, per changed path, the
commits that touched it — enough to know which commit came first, not enough to
know what is an ancestor of what. Reverse topological order means an ancestor
always appears earlier, but the converse fails: two commits on parallel branches
have a list order and no ancestry relation, so a claim anchored on a side branch
would pass a position comparison. The collector therefore emits each commit's
parents, and the verifier decides reachability over that bounded graph. The
collector gains a `--parents` read; it decides nothing.

**An anchor outside the frozen range is not accepted.** The verifier cannot see
past `base`, so it cannot establish ancestry for a commit it was never given. A
claim must be anchored inside the range it excuses. This is stricter than the
prose alone implies and it is the honest limit of a verifier that reads only its
input: a claim recorded in an earlier merged branch is real, but unprovable
here, and an unprovable claim must not be accepted on trust.

**Ordering binds the restoration class only.** Break-glass is invoked during an
emergency; a bootstrap path exists precisely where prior lineage does not.
Requiring either to precede its own work would describe something that cannot
happen. Restoration is different in kind: its whole content is a claim about
what the author knew beforehand.

**The claim is falsifiable but not self-proving.** A valid event establishes
that the claim was made in the required form, bound to a digest, before the
work. It does not establish that the claim is true — that is the boundary
question, and the specification says plainly that an affirmative verifier result
must not be reported as having settled it. Making that explicit is deliberate:
the failure mode of a check like this is that its green result gets read as
approval of the reasoning it never examined.

**A restoration cannot amend.** If the same work unit both claims restoration
and adds or amends a requirement in its scope, there is no unchanged prior
requirement to bind. This is the issue's own diagnostic — a change that needs a
governing contract to move in the same act is not a defect fix — expressed as a
binding failure rather than as advice.

## Consumed Inputs

- Confirmed Intent Snapshot `direct-defect-boundary-v0` version 2.
- Context Pack `direct-defect-boundary-v0` version 2, items CTX-1 through
  CTX-14.
- Two independent critiques, recorded in the confirmation's verification action.

## Correlation and Evidence

The falsifiable case is `#85`: implementation at 14:25, snapshot at 15:51,
confirmation at 16:10, with no prior restoration claim. Reproduced as a commit
fixture, it must be denied with the ordering reason. If it is not, this change
did not do what it exists for.

## Measurement and Stop Conditions

- A restoration claim recorded after the first material commit in its scope is
  denied; moving the anchor to an ancestor makes the same fixture pass.
- A claim bound to a stale or absent digest is denied with a different stable
  reason than the ordering failure.
- No new materiality classification appears in the delta.
- `AGENTS.md` gains a reference and restates no policy.

Stop when those hold. Do not extend this change into adoption or activation.

## Risks / Trade-offs

- **The check cannot run on this repository yet.** `.goalrail/` is ignored
  entirely and holds no tracked file, so the committed declaration the
  governance contract requires cannot be added without an owner decision
  (CTX-13, issue #89). Shared admission is separately owner-gated (CTX-14). The
  relation, the verifier support and the fixtures are deliverable now; the live
  check follows adoption. This is stated rather than hidden, and it is the
  honest limit of what this change achieves.
- **A determined author can still bind a loosely related requirement.** The
  ordering and the digest are decided; the fit is not. What changes is that the
  claim is now recorded, scoped and reviewable before the work rather than
  reconstructed after it, and the reviewer's question is bounded to one thing.
- **A green result may be misread as approval.** Mitigated by stating in the
  specification that it must not be reported as settling the boundary question,
  and by keeping the classification visibly non-`VALID`.

## Rollback

The delta adds one exception class and two verifier checks. Reverting the commit
removes both; no existing exception class, materiality rule or command exit
status is touched, so nothing else depends on it.

## Open Questions

- Whether a retrospective confirmation settles a departure that already occurred
  stays open, as recorded in the archive it arose from. This change makes the
  departure detectable going forward; it does not settle the ones behind it.
- **Revisit condition.** If a change is found after the fact to have carried a
  valid restoration claim that a reviewer would have rejected on the boundary
  question, the fit is escaping the ordering check and the next rung is a
  stricter binding — for example requiring the claim to name the exact
  requirement heading rather than the artifact containing it.
