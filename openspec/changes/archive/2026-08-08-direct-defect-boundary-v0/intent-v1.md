# Intent Snapshot

- **Intent ID:** direct-defect-boundary-v0
- **Version:** 1
- **Artifact Contract:** goalrail-context-intent
- **Artifact Contract Version:** 1
- **Status:** candidate
- **Owner:** role:repository-owner
- **Context Pack:** direct-defect-boundary-v0 version 1
- **Run references:** pending

## Source Evidence

- **SE-1 — Owner statement:** The owner filed issue #86, asking that the
  boundary between a direct change and a change requiring the planning cycle
  stop being the author's alone, and proposed wording for it.
- **SE-2 — Owner statement:** The owner stated in the same issue that this
  change needs its own intent cycle by its own terms, and that there is no
  exception for a governing document which constrains future work.
- **SE-3 — Repository fact:** The obligation is triggered by one undefined
  word, "significant", which nothing in the repository defines (CTX-1).
- **SE-4 — Repository fact:** The exemption authors already act on — that a
  defect fix need not open a cycle — is written nowhere, so it cannot be cited
  or disputed (CTX-2).
- **SE-5 — Repository fact:** The specification for Intent Snapshots governs
  every part of one except when it is required (CTX-3).
- **SE-6 — Repository fact:** The absence already cost a P1, and the recovery
  is an archived snapshot that states it was written after the implementation
  and cannot supply the intent it stands in for (CTX-4, CTX-5).
- **SE-7 — Repository fact:** That record names the unanswered question itself:
  the governing contract does not say whether a retrospective confirmation
  settles a procedural departure (CTX-6).
- **SE-8 — Repository fact:** The workspace ratchet records that a promotion
  landing as prose did not stop its class, and that a recurring class must
  escalate toward an enforced check (CTX-7).

## Desired Outcomes

| ID | Outcome | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | A change made directly rather than through the cycle has a stated criterion to meet: it restores an exact requirement already recorded in a confirmed Intent Snapshot or repository contract, without changing that requirement or its semantic boundary. | Read the contract and the specification; find the criterion stated as a condition an author must meet, not as advice. | SE-1, SE-3, SE-4 |
| OUT-2 | The criterion is applied before implementation, not recognized after it. An author names the restored requirement first; where none can be named, the cycle opens first. | Read the requirement; expect the obligation to be ordered before implementation and expect failure to name one to route to the cycle rather than to a later repair. | SE-1, SE-6 |
| OUT-3 | A reviewer has one checkable question in place of a judgement about the author's word: does this diff restore the named requirement without moving its boundary? | Read the requirement's scenarios; expect one that a reviewer can answer from the diff and the named requirement alone. | SE-1, SE-6 |
| OUT-4 | The entry condition lives with the rest of the Intent Snapshot workflow rather than only in contract prose, so it is versioned, traceable and subject to the same validation as the requirements that presume a snapshot exists. | Read `openspec/specs/intent-snapshot/spec.md`; expect the entry condition stated there. | SE-5 |
| OUT-5 | A change that needs a new or amended normative statement to justify itself is named as significant by the criterion itself, so the one word that currently carries the trigger no longer has to be interpreted alone. | Read the criterion; expect the diagnostic that a change requiring a governing contract to move in the same act is not a defect fix. | SE-1, SE-3 |

## Non-Goals

| ID | Boundary | Evidence |
|---|---|---|
| NG-1 | Do not create a retrospective settlement path. The contract already says a candidate needs active confirmation and that history is not rewritten; a general salvage norm would make late intent an ordinary recovery route. | SE-2, SE-7 |
| NG-2 | Do not exempt governing documents. A new norm framed as merely clarifying future behaviour would swallow the rule; only editorial corrections that change no future decision stay outside it. | SE-2 |
| NG-3 | Do not define "significant" exhaustively, and do not turn every correction into a planning cycle. The criterion decides one question — whether a named prior requirement is being restored — and leaves the rest of the judgement where it is. | SE-1, SE-3 |
| NG-4 | Do not add automated enforcement in this change. The criterion is a condition on an author's reasoning before implementation, and the reviewer question is its check; an enforced check is the next rung if the class recurs, not this change's scope. | SE-8 |
| NG-5 | Do not weaken any existing gate. Meeting the criterion authorizes no commit, merge, publication or external effect, and confirmed intent still grants no effect authority. | SE-2 |
| NG-6 | Do not answer whether a retrospective confirmation settles a departure that already happened. That question is recorded as open and stays open. | SE-7 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | The criterion is citable. An author or reviewer can point to one requirement that decides whether a change may proceed directly. | One requirement exists in `openspec/specs/intent-snapshot/spec.md` and one operative line in `AGENTS.md`; today neither exists. | CTX-1, CTX-2, CTX-3 |
| SIG-2 | The obligation is ordered. The requirement states that the prior requirement is named before implementation, and that failure to name one opens the cycle. | Read the requirement; find both the ordering and the routing. | CTX-4, CTX-5 |
| SIG-3 | A reviewer's question is answerable from artifacts rather than from the author's account of their own work. | A scenario states the question in terms of the diff and the named requirement. | CTX-4 |
| SIG-4 | This change is itself carried through the full cycle, which is the first test of its own terms. | Context Pack, confirmed Intent Snapshot, proposal, spec delta, design and tasks all exist before the edit to `AGENTS.md`. | CTX-6, SE-2 |
| SIG-5 | The escalation condition is recorded rather than assumed. If a change is found after the fact to have failed the criterion, the next rung is named. | The design records the recurrence condition and what escalation would mean. | CTX-7 |

## Ambiguities and Unknowns

None material. One question is deliberately left open by NG-6: whether a
retrospective confirmation settles a departure that already occurred. It is
recorded as unanswered in the archive it arose from, and this change does not
answer it.

## Confirmation

- **Status:** candidate
- **Owner confirmation:** pending
- **Active verification action:** pending
- **Confirmed at:** pending
