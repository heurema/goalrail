## Context

The pinned closure was computed on every release path and compared against
nothing (CTX-1). The pins themselves were not chosen but left: a year behind the
runtime's own LTS line, two minors behind the compiler (CTX-3). A compromise
arrives through the maintainer's own pipeline with valid provenance (CTX-9), so
attestations do not distinguish one; what a reviewer can act on is the set
changing. Install-time execution is structurally absent from this path already
(CTX-10), which bounds what a record can protect against.

## Goals / Non-Goals

**Goals:**

- Make a change to the pinned set a stated difference rather than an unstated one (OUT-1).
- Put the refusal where every release path passes (OUT-2).
- Disclose the age of each pin at adoption (OUT-3).
- Version the expanded document and register it (OUT-4).

**Non-Goals:**

- Deciding freshness or a quarantine window (NG-1).
- Presenting the dates as a gate (NG-2).
- Claiming a human read the change (NG-3).
- Promising external compatibility for an unpublished build input (NG-4).
- Touching the supported-platform contract (NG-5).
- Moving a pinned version (NG-6).
- Claiming protection against install-time execution (NG-7).

## Decisions

**D1 — Counts are recorded beside the digest.** A digest alone answers "did
anything change" and nothing else. The counts answer "what", and the
install-script count answers the question a reviewer most needs after CTX-9: a
package that acquired one is a different kind of event from a version bump.

**D2 — The comparison lives in the shared input loader.** `Build`, `Verify` and
`CheckSourceLock` all call `loadReleaseInputs`. Placing the check in the
inspecting command alone would have left the two commands that actually produce
a release able to proceed against an unreviewed set — which is how it was first
written, and what an independent review found (CTX-8).

**D3 — The dates are disclosure, and the code says so.** Both are supplied by
whoever moves the pin (CTX-11), so a waiting period cannot be enforced on
itself. Validation therefore establishes only what it can: that both are present
and that adoption does not precede publication. Naming the limit in the contract
matters more than the check, because a reader who believes a window is enforced
would stop applying one.

**D4 — A new schema identifier rather than a widened one.** Decoding is strict
(CTX-7), so a v1 reader rejects a v2 document and the v2 validator rejects a v1
one. One identifier describing both would be a claim neither side could rely on.
The registry says identifiers and their semantic boundaries are the automation
contract (CTX-6), so the registry moves with it.

**D5 — No pin moves here.** Applying a rule to its own first use, in the act
that introduces it, would make the rule unfalsifiable at the moment it most
needs testing.

## Consumed Inputs

| Input | Source and trust | Accepted states/variants | Refused states | Mutation/race policy | Verification |
|---|---|---|---|---|---|
| `release/setup/source-lock.json` | Repository content, reviewed as a commit; trusted only after the contract decoder accepts it | The v2 shape: pins, closure record, adoption dates, platform list | Unknown schema, absent or malformed closure record, absent date, adoption preceding publication, unknown fields | Read once per invocation from the checkout; no decision spans two reads | Table-driven cases per refused state, against a copy of this repository's real lock rather than an invented fixture |
| `release/setup/compiler/package-lock.json` | Repository content; the set whose closure is computed | The pinned lockfile version and package shape the existing loader already accepts | Anything that loader already refuses; unchanged by this slice | Read in the same invocation as the record it is compared against | The positive case is this repository's own pinned set: 80 packages, 1 install script |

## Correlation and Evidence

No new evidence artefact. The refusal is an error on the release path, and the
record itself is the durable evidence — it is the thing a reviewer reads when a
pin moves.

## Measurement and Stop Conditions

- The current pinned set is accepted (SIG-1).
- Each of the three closure facts, altered alone, stops the shared path and names
  itself (SIG-2).
- A missing date, and an adoption preceding publication, are refused (SIG-3).
- The expanded shape under the old identifier is refused, and the registry names
  the new one (SIG-4).
- No pinned version differs before and after (SIG-5).

Stop and reshape if the record cannot be kept accurate without a tool to
regenerate it, since a record that is tedious to update becomes a record people
route around.

## Risks / Trade-offs

- **The record can be updated without being read.** Nothing here establishes that
  anyone looked at what changed; it establishes that they had to touch the
  record. That is the honest limit and NG-3 states it.
- **A new identifier for an internal file.** The cost is a coordinated move of
  every reader, which is cheap here because they all live in this repository and
  move in the same commit (CTX-4, CTX-5). It would not be cheap for a published
  artefact, and the registry now records that this one is not published.
- **The freshness rule is absent.** The mechanism records the evidence for a
  decision nobody has made yet. That gap is stated in the contract rather than
  left for a reader to discover.

## Rollback

Reverting the branch restores the previous identifier and removes the
comparison. No published artefact carries the v2 shape, so nothing outside this
repository has to be migrated back.

## Open Questions

None.
