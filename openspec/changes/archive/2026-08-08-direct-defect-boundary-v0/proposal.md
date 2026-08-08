## Why

A change made outside the planning cycle rests on the author's own word that it
merely restores something already required. The contract triggers the cycle on
one undefined term and records no defect-class exception at all, so the claim
cannot be cited, checked, or disputed (OUT-1).

The mechanism that would otherwise catch this decides paths, not order. Lineage
exception events are immutable, digest-identified and dated, but nothing
requires one to precede the work it excuses. That is exactly the recorded
failure: in `#85` the implementation was committed at 14:25, the snapshot at
15:51 and its confirmation at 16:10, and after that backfill a fully active
verifier sees a confirmed snapshot bound to the work unit and no violation
(OUT-2, OUT-6).

## What Changes

- Give the restoration claim a categorical class in the lineage exception model,
  binding a digest-addressable requirement artifact, the path scope claimed, and
  the work unit. A claim that can name no such artifact has nothing to bind.
- Require a restoration claim to be anchored in an ancestor of the first commit
  touching a material path within its scope. A claim recorded afterwards is
  out-of-scope evidence, not a late repair.
- Have the verifier decide that ancestry and that binding, leaving the reviewer
  the one genuinely semantic question — whether the diff moved the requirement's
  boundary.
- Point the contract at the mechanism instead of restating it.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Restoration claim as a bound exception class | OUT-1, SIG-2 | NG-1, NG-3, NG-8 |
| Ancestry condition against the first material commit | OUT-2, SIG-1, SIG-3 | NG-2, NG-7 |
| Verifier decides ordering and binding | OUT-3, OUT-6, SIG-6 | NG-4 |
| Materiality untouched | OUT-4, SIG-4 | NG-1 |
| Contract carries a pointer | OUT-5, SIG-5 | NG-1, NG-6 |

## Capabilities

### New Capabilities

None. The exception model, the event ordering rules and the verifier all exist;
what is added is one class and the condition that distinguishes it.

### Modified Capabilities

- `work-unit-lineage`: the requirement that already makes exceptions first-class
  evidence gains the restoration class and the ordering condition specific to
  it. Break-glass and bootstrap keep their present timing, which is inherent to
  what they are.
- `lineage-admission`: the verifier's decided set gains the ancestry and binding
  of a restoration claim, with stable reason identifiers distinguishing a claim
  recorded too late from one bound to the wrong artifact.

## Impact

- `internal/lineage`: one exception class and its bindings.
- `internal/admission`: ancestry and binding checks, with fixtures reproducing
  the `#85` commit order.
- `AGENTS.md`: one reference naming the canonical requirement.
- No change to materiality classification, to any existing exception class, or
  to what any command exits with.

## Non-Goals

- Not a second materiality policy. What counts as a material path remains the
  committed policy's decision alone.
- Not a retrospective settlement path. A claim recorded after the first material
  commit fails the ordering condition; it does not become a sanctioned recovery,
  and whether such a departure can be settled at all stays an open question.
- Not a machine judgement of semantic equivalence. Whether a diff moved a
  requirement's boundary remains a review question.
- Not adoption or activation. This repository cannot presently carry a committed
  declaration (#89), and shared admission remains owner-gated; the check ships
  with fixtures and runs live only after those are settled.
