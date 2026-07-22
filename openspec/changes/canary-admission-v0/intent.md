# Intent Snapshot

- **Intent ID:** CANARY-ADMISSION-V0
- **Version:** 1
- **Status:** confirmed
- **Owner:** t3chn
- **Run references:** Codex thread `019f836a-81ce-7bf0-a3ae-94835a80b10e`; prior boundary PR `heurema/goalrail#5`

## Source Evidence

- **SE-1 — Owner statement:** On 2026-07-22 the owner actively confirmed the bounded vertical slice: append-only `eligible` and `excluded` admission evidence, exclusions consuming no ordinal, the existing `change_id` identifying excluded candidates, check-set freezing before the first terminal verification, and real canary activation remaining blocked.
- **SE-2 — Owner statement:** Keep the project moving through small useful steps and avoid over-engineering.
- **SE-3 — Repository fact:** `canary/intent-canary-v0/manifest-v1.md` requires eligibility to be decided before variant reveal, rejected candidates to retain an exclusion reason without consuming an ordinal, and only accepted changes to receive the next ordinal.
- **SE-4 — Repository fact:** The frozen manifest requires a non-empty check set after scope is understood but before the first terminal verification run and owner assessment; selected checks cannot be removed after results are known, and corrections append rather than rewrite.
- **SE-5 — Repository fact:** The current operator service starts synthetic assignments directly and records check references only with the terminal delivery event; it has no separate admission decision or frozen check-set event.
- **SE-6 — Repository fact:** The evidence store already provides append-only hash-chain validation and process-shared serialization, while the operator service rejects real starts until a separate activation decision.
- **SE-7 — Repository fact:** OpenSpec is a replaceable development-time compiler/provider; canonical canary behavior belongs in provider-neutral Goalrail domain and evidence contracts.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Every considered canary candidate receives an append-only `eligible` or `excluded` admission decision before variant reveal, with a stable candidate reference, actor, timestamp, source reference, and categorical reason. | Admit eligible and excluded synthetic candidates, inspect their evidence, and verify that an excluded receipt does not disclose the next variant. | SE-1, SE-3, SE-6 |
| OUT-2 | Only an eligible admission receives the next immutable ordinal and assigned variant; excluded candidates consume no ordinal and cannot later acquire an assignment through conflicting replay. | Exclude one synthetic candidate, admit two eligible candidates, and verify that their ordinals remain consecutive from 1. | SE-1, SE-3 |
| OUT-3 | Each accepted assignment freezes a non-empty set of stable check references at the agreed pre-run boundary; terminal check results may be recorded only against that set, selected checks cannot later be removed, and corrections append rather than rewrite. | Exercise empty, duplicate, missing-freeze, changed-set, valid-result, and append-only correction cases. | SE-1, SE-4, SE-5 |
| OUT-4 | Admission decisions, assignments, frozen checks, and terminal results remain inspectable and reportable through the existing verifiable evidence stream without weakening cross-process serialization. | Verify the full chain and compare inspect/report output after concurrent synthetic admission attempts. | SE-1, SE-5, SE-6 |
| OUT-5 | The complete slice can be rehearsed locally with synthetic data while every real admission or assignment remains rejected pending a separate owner activation instruction. | Run the synthetic path, attempt a real admission, and verify that no repository evidence file is created by planning or tests. | SE-1, SE-6 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | Do not activate the real 15-change canary, allocate ordinal 1 to real work, or treat this change as `GO_LIVE`. | SE-1, SE-6 |
| NG-2 | Do not change the frozen eligibility rules, 15-position assignment rotation, measurement formulas, pass/stop thresholds, or owner-assessment semantics. | SE-1, SE-3, SE-4 |
| NG-3 | Do not automate the eligibility judgment or infer owner/team facts; an operator supplies the decision and bounded evidence references. | SE-2, SE-3 |
| NG-4 | Do not capture prompts, source code, credentials, raw transcripts, or unbounded tool output in admission or check evidence. | SE-2, SE-7 |
| NG-5 | Do not add a database, service, dashboard, workflow engine, authorization product, sandbox product, or provider-specific runtime abstraction. | SE-2, SE-7 |
| NG-6 | Do not move runtime artifacts back under an active or archived OpenSpec change. | SE-7 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | An exclusion is durable but ordinal-neutral. | One excluded synthetic candidate produces one valid admission event with its reason; assigned count and next ordinal remain unchanged, and no variant appears in the exclusion receipt or event. | SE-1, SE-3 |
| SIG-2 | Eligible admissions assign exactly once in sequence. | After an exclusion, two eligible synthetic candidates receive ordinals 1 and 2 with manifest-defined variants; duplicate or conflicting decisions fail without mutation. | SE-1, SE-3, SE-6 |
| SIG-3 | Check selection precedes terminal verification evidence. | Empty or duplicate selections fail; terminal delivery before a freeze fails; a result referencing a missing or changed check fails; a valid frozen set and matching results succeed. | SE-1, SE-4, SE-5 |
| SIG-4 | Historical check evidence is not rewritten. | Removing or replacing a frozen check after result recording fails; a legitimate correction adds a superseding event and the complete chain remains verifiable. | SE-4, SE-6 |
| SIG-5 | Existing integrity properties survive the slice. | Concurrent cooperating processes retain every accepted event exactly once with canonical sequence and digest links; inspect and report agree with the verified chain. | SE-6 |
| SIG-6 | Real execution remains owner-gated. | Every non-synthetic admission or assignment attempt returns the explicit not-activated error; the manifest remains `frozen_not_activated` and repository `canary/intent-canary-v0/events.jsonl` remains absent. | SE-1, SE-6 |
| SIG-7 | The bounded change is regression-safe. | Strict OpenSpec validation, `go vet ./...`, `go test -race ./...`, and `git diff --check` pass. | SE-2, SE-7 |

## Ambiguities and Unknowns

None.

## Confirmation

- **Confirmed by:** t3chn
- **Confirmed at:** 2026-07-22T09:08:59Z
- **Verification action:** The owner reviewed the explicit version 1 outcomes, non-goals, success signals, and the two open boundary choices in Codex thread `019f836a-81ce-7bf0-a3ae-94835a80b10e`, then replied “да” to confirm use of the existing `change_id` and freezing checks before terminal verification.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
