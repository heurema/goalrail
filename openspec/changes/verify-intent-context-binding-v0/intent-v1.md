# Intent Snapshot

- **Intent ID:** intent-verify-intent-context-binding-v0
- **Version:** 1
- **Status:** confirmed
- **Owner:** repository owner
- **Context Pack:** context-verify-intent-context-binding-v0 version 1
- **Run references:** none; this change authorizes no provider run

## Source Evidence

- **SE-1 (owner):** Reconcile onto merged `main` first, ship the hook-linkage fix alone in PR #13, and hold the Context Pack verification behaviour until its requirement has a valid spec home.
- **SE-2 (repository):** The behaviour is already implemented and covered by regression tests in commit `b36fabd`; only its promoted requirement is missing.
- **SE-3 (repository):** The only existing wording sits inside a superseded activation block in an unmerged change, and its intent ID is already claimed by an archived change on `main`.
- **SE-4 (repository):** The promoted `intent-snapshot` and `context-pack` capabilities already define Context Pack semantics at the intent level.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | State in the promoted `local-run` capability that WorkSpec preparation SHALL validate the confirmed intent together with the sibling Context Pack the intent declares, with both artifacts resolving inside the canonical repository root. | Read the modified requirement and confirm it describes the behaviour already shipped in `b36fabd`. | SE-2, CTX-1, CTX-2 |
| OUT-2 | State that a missing, unreadable, mismatched, malformed, oversized, or root-escaping Context Pack SHALL fail preparation before any prepared state, run ID, launch claim, or provider invocation exists. | Read the failure scenarios and compare them with the existing regression tests. | SE-2, CTX-1, CTX-2 |
| OUT-3 | Keep the requirement provider-neutral: it SHALL NOT add Context Pack fields to the canonical WorkSpec and SHALL NOT depend on proposal, specification, design, task, provider, or activation artifacts. | Confirm the canonical WorkSpec and receipt schemas are unchanged and the requirement names no provider. | SE-2, SE-4, CTX-5 |
| OUT-4 | Give this behaviour a spec home under a new intent identity, leaving the archived `intent-activate-dogfood-run-v0` on `main` untouched as the historical record. | Confirm the new change declares a new intent ID and that no merged archive file is edited. | SE-3, CTX-3, CTX-4 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | Do not change the implementation in `b36fabd`, the canonical WorkSpec or receipt schemas, or any command behaviour. This change is specification-only. | SE-2, CTX-1 |
| NG-2 | Do not reuse, re-version, amend, or edit `intent-activate-dogfood-run-v0`, the archived `2026-07-24-activate-dogfood-run-v0` change, or any other merged evidence. | SE-3, CTX-4 |
| NG-3 | Do not restate the intent-level Context Pack semantics already promoted in `intent-snapshot` and `context-pack`; state only preparation-time enforcement. | SE-4, CTX-5 |
| NG-4 | Do not touch the hook-linkage requirement, PR #13, or the `dogfood-activation` and `operator-cli-help` capabilities. | CTX-6 |
| NG-5 | This snapshot authorizes no proposal, specification, design, tasks, implementation, commit, push, PR, merge, provider run, or external effect. Each remains a separate later owner gate. | SE-1, CTX-7 |
| NG-6 | Do not add a workflow engine, authorization system, effect gateway, credential broker, sandbox platform, database, dashboard, or general activation surface. | CTX-7 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | The promoted `local-run` capability describes preparation-time Context Pack enforcement. | One modified requirement covers the accepted path and every rejection class; strict OpenSpec validation passes. | SE-2, CTX-1, CTX-2 |
| SIG-2 | Specification and implementation agree without new code. | Every scenario in the modified requirement maps to an existing test in `b36fabd`; `go test ./...` is unchanged and passing. | SE-2, CTX-1 |
| SIG-3 | Merged evidence is untouched. | No file under `openspec/changes/archive/` is modified and no existing intent ID or version is reused. | SE-3, CTX-4 |
| SIG-4 | The change stays provider-neutral and narrow. | Canonical WorkSpec and receipt schemas are byte-unchanged; the diff touches only this change's own artifacts. | SE-2, SE-4, CTX-5 |

## Ambiguities and Unknowns

None.

## Confirmation

- **Confirmed by:** repository owner
- **Confirmed at:** 2026-07-25T08:39:57Z
- **Verification action:** Owner reviewed the owner-facing summary of candidate version 1 and explicitly confirmed the snapshot by name and version in the current Goalrail task.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
