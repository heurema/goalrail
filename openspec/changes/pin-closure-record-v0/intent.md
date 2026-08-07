# Intent Snapshot

- **Intent ID:** pin-closure-record-v0
- **Version:** 1
- **Artifact Contract:** goalrail-context-intent
- **Artifact Contract Version:** 1
- **Status:** candidate
- **Owner:** role:repository-owner
- **Context Pack:** pin-closure-record-v0 version 1
- **Run references:** pending

## Provenance

This snapshot was composed **after** the implementation, from the observed diff
of branch `pin-moves-are-dated-events` against `c30bd84`. It is not evidence of
owner intent that preceded the work, and it does not claim to be. The work was
carried out as a direct change; an independent review flagged the missing
snapshot twice, and this is the honest form of what can be written now.

What follows is therefore an interpretation of what the implementation does,
offered for owner confirmation as the basis of a merge decision. Confirming it
establishes intent for that decision. It cannot make the implementation have
been intent-first, and whether a retrospective confirmation settles the
procedural departure is a question the governing contract does not answer
(CTX-12) — it is recorded below as an ambiguity rather than assumed away.

## Source Evidence

- **SE-1 — Repository fact:** The pinned closure was computed, reported, and
  compared against nothing, so a pin could move inside eighty packages of
  lock-file churn with no verdict (CTX-1, CTX-2).
- **SE-2 — Repository fact:** The pins were not chosen but left: the runtime is
  a year behind its own LTS line, the compiler two minors behind (CTX-3).
- **SE-3 — Repository fact:** The document is not published and its only readers
  move in the same commit, so an incompatible change is bounded to this
  repository — but the registry declares identifiers to be the automation
  contract, and decoding is strict (CTX-4, CTX-5, CTX-6, CTX-7).
- **SE-4 — Repository fact:** A comparison placed in one command leaves the
  commands that actually produce a release able to bypass it (CTX-8).
- **SE-5 — External fact:** A compromise arrives through the maintainer's own
  workflow with valid provenance, so attestations do not distinguish it; what a
  reviewer can act on is the closure changing (CTX-9).
- **SE-6 — Repository fact:** Install-time execution is structurally absent from
  Goalrail's path, and both adoption dates come from whoever moves the pin
  (CTX-10, CTX-11).

## Desired Outcomes

| ID | Outcome | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | The dependency closure a reviewer last looked at is recorded beside the pins, as package count, install-script count and digest, so a change to the set is a stated difference rather than an unstated one. | Read `release/setup/source-lock.json` and compare its closure record with `lock-check` output. | SE-1, CTX-2 |
| OUT-2 | Building a release, verifying one, and checking the lock all refuse when the computed closure disagrees with the record, and the refusal names which of the three facts disagreed. | Alter each of the three in turn and observe the refusal from the shared input path. | SE-4, CTX-8 |
| OUT-3 | The source lock discloses when each pinned version was published upstream and when this repository adopted it, so the age at adoption is visible to whoever reviews a pin move. | Read both dates for the runtime and the compiler. | SE-2, CTX-3, CTX-11 |
| OUT-4 | The expanded document carries its own schema identifier, and the operator registry describes it. | Read the identifier in the lock, the validator, and `docs/contracts-v0.2.md`. | SE-3, CTX-6, CTX-7 |

## Non-Goals

| ID | Boundary | Evidence |
|---|---|---|
| NG-1 | Do not decide how current a pin should be, or how long a version should have been public before adoption. That rule is deferred, and the contract says so in the same act. | SE-2 |
| NG-2 | Do not present the adoption dates as a gate. Both are supplied by whoever moves the pin, so no waiting period can be enforced on itself; the only inconsistency the record catches alone is a pin claiming to predate its own publication. | SE-6, CTX-11 |
| NG-3 | Do not claim that a human reviewed the closure. The mechanism makes a difference stop the build and require an explicit update; it cannot establish that anyone read what changed. | SE-1 |
| NG-4 | Do not promise external compatibility or a migration path for the source lock. It is a build input, not a published artefact. | SE-3, CTX-4 |
| NG-5 | Do not change the supported-platform contract, which the workflows read from the same file. | CTX-5 |
| NG-6 | Do not move the pinned versions themselves in this change. Moving them under a rule introduced by the same act would apply that rule retroactively to its own first use. | SE-2 |
| NG-7 | Do not claim the record defends against a compromise while a package is being installed. Install-time execution is structurally absent from this path already, and what the record exposes is the set changing. | SE-5, CTX-10 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | The current pinned set is accepted as recorded. | `lock-check` on this repository succeeds. | CTX-2 |
| SIG-2 | Each of the three closure facts, altered alone, stops the shared release-input path. | Three cases, three refusals, each naming what disagreed. | CTX-8 |
| SIG-3 | An adoption record missing either date, or claiming adoption before publication, is refused. | Three cases, three refusals. | CTX-11 |
| SIG-4 | A document carrying the expanded shape under the previous identifier is refused, and the registry names the new one. | One refusal; one registry line. | CTX-6, CTX-7 |
| SIG-5 | The change moves no pinned version. | The runtime and compiler versions are identical before and after. | NG-6 |

## Ambiguities and Unknowns

| ID | Question | Evidence |
|---|---|---|
| AMB-1 | Whether confirming this snapshot after the implementation settles the departure from the intent-first order, or only records it. The governing contract requires the snapshot before the work and forbids rewriting history, and says nothing about the case where the order was not followed. This is the owner's to decide, and the answer determines whether the branch merges as it stands or is reworked. | CTX-12 |

## Confirmation

- **Confirmed by:** pending
- **Confirmed at:** pending
- **Verification action:** pending
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
