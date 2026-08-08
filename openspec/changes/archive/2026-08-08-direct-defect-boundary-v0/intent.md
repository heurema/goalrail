# Intent Snapshot

- **Intent ID:** direct-defect-boundary-v0
- **Version:** 2
- **Artifact Contract:** goalrail-context-intent
- **Artifact Contract Version:** 1
- **Status:** confirmed
- **Owner:** role:repository-owner
- **Context Pack:** direct-defect-boundary-v0 version 2
- **Run references:** pending
- **Previous versions:** 1 (`intent-v1.md`), candidate, never confirmed. Amended
  after two independent critiques. Version 1 proposed the rule as contract prose
  with automated enforcement named as a non-goal; both critiques found that this
  would define a second materiality policy the governance contract forbids
  (CTX-9), and would restate mechanically what the admission verifier already
  decides (CTX-10). The answer is inverted here: materiality stays entirely with
  the existing policy, and what this change adds is the one thing that verifier
  does not model — the order in which a direct restoration is claimed (CTX-11).
  The enforcement non-goal is withdrawn, because the ordering question is
  decidable and prose alone was already measured as the rung that does not hold.

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
- **SE-5 — Repository fact:** The specification for Intent Snapshots covers
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
- **SE-9 — Repository fact:** An instruction surface may not define a second
  materiality policy or select a parallel spec authority, and materiality is
  already owned deterministically by the admission policy (CTX-9).
- **SE-10 — Repository fact:** The mechanical part of the problem is already
  solved: candidate intent cannot satisfy a required relation, and material
  paths without lineage are denied admission (CTX-10).
- **SE-11 — Repository fact, verified:** That verifier decides paths, not
  order. Nothing in it models an author naming a requirement before
  implementation (CTX-11).
- **SE-12 — Repository fact, verified:** The one recorded instance of the
  failure survives the existing mechanism. In `#85` the code was committed at
  14:25, the snapshot at 15:51 and its confirmation at 16:10; after that
  backfill a fully active verifier sees a confirmed snapshot bound to the work
  unit and no violation (CTX-12).
- **SE-13 — Repository fact, verified:** This repository cannot presently carry
  the committed declaration its own governance contract requires, because
  `.goalrail/` is ignored entirely and holds no tracked file (CTX-13).
- **SE-14 — Repository fact:** Activation of shared admission is staged behind
  an owner-authorized external action, and the workflow that would carry it has
  no production caller (CTX-14).

## Desired Outcomes

| ID | Outcome | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | A claim that a change merely restores an existing requirement is a recorded relation rather than an author's assertion. It binds the digest of the previously confirmed requirement, the path scope claimed for the restoration, and the work unit. | Read the requirement; expect a relation with those three bindings and expect a free-form justification to be insufficient on its own. | SE-1, SE-4, SE-9 |
| OUT-2 | The claim is ordered against the work it excuses: it must exist in an ancestor of the first commit touching a material path in its scope. A claim recorded afterwards does not make the route available. | Read the requirement; expect the ancestry condition stated over commits rather than over wall-clock time. | SE-6, SE-12 |
| OUT-3 | The order is decided by the verifier rather than by a reviewer's reading. Ancestry and binding are checked mechanically; a reviewer is left with the one question that is genuinely semantic — whether the diff moved the requirement's boundary. | Read the requirement's scenarios; expect ordering and binding failures to be verifier outcomes and expect the boundary question to remain a review question. | SE-8, SE-11 |
| OUT-4 | Materiality stays where it already lives. This change adds a relation to the lineage model; it introduces no second classification of what counts as material, and no parallel authority over it. | Read the delta; expect no new materiality rule and expect the existing policy to remain the sole classifier. | SE-9, SE-10 |
| OUT-5 | The contract carries a pointer to the mechanism rather than a copy of it, so the instruction surface stays a thin adapter. | Read `AGENTS.md`; expect a reference naming the canonical requirement and no restated policy. | SE-9 |
| OUT-6 | The failure recorded in `#85` is denied by the new check. The shape — material commits first, confirmed intent afterwards, no prior restoration claim — is the case the relation exists to catch. | Construct that commit shape in a test fixture and run the verifier; expect denial with a stable reason identifier. | SE-12, SE-6 |

## Non-Goals

| ID | Boundary | Evidence |
|---|---|---|
| NG-1 | Do not define a second materiality policy, and do not restate in the contract what the admission policy already decides. | SE-9, SE-10 |
| NG-2 | Do not create a retrospective settlement path. A claim recorded after the fact fails the ordering condition; it does not become a sanctioned recovery. | SE-2, SE-7 |
| NG-3 | Do not exempt governing documents. A change needing a new or amended normative statement to justify itself cannot be a restoration, because there is no prior requirement to bind. | SE-2 |
| NG-4 | Do not attempt to decide semantic equivalence mechanically. Whether a diff moved a requirement's boundary stays a review question; the verifier decides only ordering and binding. | SE-11 |
| NG-5 | Do not adopt this repository as a managed project, and do not activate shared admission. Both are blocked or owner-gated, and neither is this change's to perform. | SE-13, SE-14 |
| NG-6 | Do not weaken any existing gate. A valid restoration claim authorizes no commit, merge, publication or external effect. | SE-2 |
| NG-7 | Do not answer whether a retrospective confirmation settles a departure that already occurred. That question is recorded as open and stays open. | SE-7 |
| NG-8 | Do not make the direct route the expected path. It is an exception with a recorded claim, visible as such, not a second ordinary lane. | SE-1, SE-9 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | A restoration claim recorded after the first material commit in its scope is denied. | A test fixture with that commit order produces a denial with a stable reason identifier; the same fixture with the claim moved to an ancestor does not. | CTX-12 |
| SIG-2 | A claim whose bound digest does not match the requirement it names is denied. | A fixture binding a stale or absent digest produces a denial distinguishable from the ordering failure. | CTX-10 |
| SIG-3 | The recorded `#85` shape is denied. | The commit order from that pull request, reproduced as a fixture, is refused. | CTX-12 |
| SIG-4 | No new materiality classification appears. | The delta adds no rule classifying paths, and the existing policy remains the only classifier. | CTX-9 |
| SIG-5 | The contract gained a pointer, not a policy. | `AGENTS.md` names the canonical requirement in one place and restates none of it. | CTX-9 |
| SIG-6 | The check exists as code with tests rather than as guidance. | The verifier decides the ordering; no part of this outcome rests on a reviewer remembering it. | CTX-7 |

## Ambiguities and Unknowns

- **AMB-1 — What a restoration claim may bind to.** A confirmed Intent Snapshot
  has a digest; a "repository contract" as the version 1 wording put it does not
  obviously have one. Proposed: the claim binds a digest-addressable artifact
  only — a retained confirmed snapshot or a committed spec file at a recorded
  revision — so that "the requirement I am restoring" is a checkable reference
  rather than a name. **Resolved by owner confirmation of this version.** A
  claim that cannot name a digest-addressable artifact has nothing to bind, and
  the direct route is unavailable to it.
- **AMB-2 — Whether the check can be exercised here.** The relation and its
  verifier can be implemented and tested against fixtures without this
  repository being a managed project, but it cannot run over this repository's
  own history until #89 is settled and admission is activated. Proposed: deliver
  the relation, the verifier support and the fixtures now; the live check
  follows adoption. Recorded rather than resolved, because it bounds what the
  success signals can measure.

## Confirmation

- **Status:** confirmed
- **Owner confirmation:** role:repository-owner
- **Confirmed at:** 2026-08-07T17:59:40Z
- **Active verification action:** The owner read a plain-language view of
  version 1 and declined to confirm it, directing an independent critique
  instead. That critique found the proposal would define a second materiality
  policy; a claim built on it — that the rule already existed and only needed
  enabling — was then put to a second critic in a fresh session, which refuted
  it and established the residual ordering gap (CTX-11, CTX-12) and the
  adoption blocker (CTX-13). The owner then read a plain-language view of
  version 2, including the proposed resolution of AMB-1, and confirmed that
  version by name.
- **Amendment rule:** A material change to outcomes, non-goals, or success
  signals creates a new version; wording-only edits preserve this version.
