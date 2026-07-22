# Intent Snapshot

- **Intent ID:** INT-HARDEN-CANARY-INTEGRITY
- **Version:** 1
- **Status:** confirmed
- **Owner:** Goalrail owner
- **Run references:** request:goalrail-review-fixes-2026-07-21

## Source Evidence

- **SE-1 — Owner instruction:** After reviewing the three validated PR #3 findings, the owner explicitly instructed Goalrail to implement the fixes.
- **SE-2 — Review finding:** `github:heurema/goalrail#3:discussion_r3623609267` shows that the current process-local mutex does not serialize the evidence read/validate/append transaction across CLI processes.
- **SE-3 — Review finding:** `github:heurema/goalrail#3:discussion_r3623609274` shows that a terminal `SESSION_CONFLICT` is projected as unresolved rather than as a wrong join.
- **SE-4 — Review finding:** `github:heurema/goalrail#3:discussion_r3623609279` shows that wording-only amendment validation permits source evidence and evidence references to change in place.
- **SE-5 — Repository fact:** `git:e1c9684` keeps real canary assignments blocked and records `NO-GO_NOW`; no real canary evidence has been created.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Concurrent Goalrail CLI processes targeting one evidence file cannot append competing sequence/digest records or corrupt the chain. | Run a deterministic multi-process append test and verify the complete chain and retained events. | SE-1, SE-2 |
| OUT-2 | A verified root-session conflict is a wrong join and triggers the existing immediate canary `STOP` rule. | Project one `SESSION_CONFLICT` through the operator report and observe `wrong_joins=1` and `STOP`. | SE-1, SE-3 |
| OUT-3 | A wording-only intent amendment may change wording but cannot replace source evidence or repoint semantic items to different evidence. | Attempt provenance mutations under `wording_only` and require validation failures while genuine wording edits continue to pass. | SE-1, SE-4 |
| OUT-4 | Each reviewed defect has a regression test that fails on `e1c9684` behavior and passes after the smallest scoped fix. | Run focused tests, the full race suite, and strict OpenSpec validation. | SE-1, SE-2, SE-3, SE-4 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | Do not activate the real canary, archive either OpenSpec change, deploy, change credentials, or change external infrastructure in this slice. | SE-1, SE-5 |
| NG-2 | Do not replace the repository-local JSONL evidence store with a database, workflow engine, or new service, and do not add a dashboard. | SE-1, SE-2 |
| NG-3 | Do not weaken append-only history, automatic lineage, bounded resolution, or the existing wrong-join hard stop. | SE-1, SE-3, SE-4 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | Cross-process evidence writes remain valid. | Two or more synchronized subprocesses append distinct valid events to one store; `Verify` succeeds and every event is retained exactly once. | SE-2 |
| SIG-2 | Wrong joins stop immediately. | One terminal `SESSION_CONFLICT` yields `WrongJoins == 1`, `HardStopSignals.WrongJoin == true`, and verdict `STOP`; ordinary unresolved lineage keeps its existing threshold. | SE-3 |
| SIG-3 | Wording edits preserve provenance. | Changes to `SourceEvidence` or semantic-item `EvidenceRefs` fail wording-only validation; statement-only wording and stable-item reordering remain valid. | SE-4 |
| SIG-4 | The slice introduces no regression. | Focused tests, `go vet ./...`, `go test -race ./...`, Python correlation tests, and strict OpenSpec validation all pass. | SE-1, SE-5 |

## Ambiguities and Unknowns

None. The owner reviewed the concrete findings and explicitly instructed implementation of all three fixes. Platform support and lock implementation remain replaceable design choices rather than unresolved product meaning.

## Confirmation

- **Confirmed by:** Goalrail owner
- **Confirmed at:** 2026-07-21
- **Verification action:** The owner read the evidence-backed summary of all three findings, including the proposed separate fix slice and regression tests, and explicitly replied `делай`.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
