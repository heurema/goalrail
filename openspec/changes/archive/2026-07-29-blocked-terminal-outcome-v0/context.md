# Context Pack

- **Context Pack ID:** context-blocked-terminal-outcome-v0
- **Version:** 1
- **Previous version:** pending
- **Started at:** 2026-07-29T11:20:00Z
- **Completed at:** 2026-07-29T12:10:26Z
- **Outcome:** sufficient

## Context Items

| ID | Kind | Claim | Source | Observed at | Relevance |
|---|---|---|---|---|---|
| CTX-1 | repository | A twelve-run A/B measurement on a conflicting-requirements kata showed 6/6 agents returning a structured question when a machine-accepted question channel existed and 0/6 when it did not; without the channel every agent guessed, the strongest documenting the conflict in prose and shipping anyway. A 3-run retest with a quiet generalized channel and a real-clone dogfood run confirmed the channel is found without being advertised. | repo:baseline-kata-authoring/katas/account-cleanup-v1/evidence/ab-measurement-2026-07-29.md | 2026-07-29T09:00:00Z | Establishes the behaviour the harness must make expressible: an escalation outcome, not a smarter agent. |
| CTX-2 | repository | The terminal receipt today carries no schema identifier, and receipt validation whitelists exactly five terminal statuses, so any new status breaks every existing reader regardless of how it is added. | repo:internal/localrun/service.go | 2026-07-29T11:30:00Z | A first-class `blocked` status requires an explicit receipt schema version rather than a silent v0 edit. |
| CTX-3 | repository | `DecodeWorkSpec` uses `DisallowUnknownFields` and the canonical WorkSpec is digest-bound, so any new WorkSpec field forks `goalrail.work-spec/v0` and threatens frozen digests. A reserved constant escalation path requires no WorkSpec change at all. | repo:internal/domain/work_spec.go | 2026-07-29T11:30:00Z | Confines the contract change to the receipt; the WorkSpec schema stays untouched. |
| CTX-4 | repository | `Start` already writes eager terminal receipts for denied, failed and unlinked launches through the `failureReceipt` path, and preparation freezes a content-addressed dirty-baseline observation of the whole worktree. | repo:internal/localrun/service.go | 2026-07-29T11:35:00Z | The eager escalation-artifact capture and the prepare-time artifact gate reuse existing mechanisms instead of adding new ones. |
| CTX-5 | repository | The workspace contract keeps runs one-shot and fail-closed, defers any control plane, and forbids mid-run owner dialogue and background continuation. | repo:AGENTS.md | 2026-07-29T11:20:00Z | The escalation must be a terminal outcome; answering it creates a new intent version and a new run, never a resume. |
| CTX-6 | external | A five-lens design review (simplicity, state machine, gaming, contract evolution, product claim) produced 41 findings; the consensus cuts and state rules are recorded in the retained review document, including: reserved path instead of a WorkSpec field, artifact retained append-only instead of delta-excluded, opaque bytes with hygiene checks instead of semantic validation, prepare-time gate, blocked-plus-edits rule, status precedence, and verbatim check results. | repo:openspec/changes/blocked-terminal-outcome-v0/evidence/design-review-v2.md | 2026-07-29T12:00:00Z | The confirmed design is v2 after review, not the original brief; the brief is retained alongside for provenance. |
| CTX-7 | external | The recurrence claim ("the same conflict never silently recurs") was found undeliverable without a question registry, which the workspace contract forbids as new scope; the claim is cut, and the machine-followable chain is delivered instead through a receipt-side intent reference and an intent-side resolves field. | repo:openspec/changes/blocked-terminal-outcome-v0/evidence/design-review-v2.md | 2026-07-29T12:00:00Z | Bounds the product claim to what the design actually delivers. |
| CTX-8 | external | The owner reviewed the two status options and selected variant A: a first-class `blocked` terminal status together with explicit receipt schema versioning, over reusing `verification_incomplete` with a reason code. | owner:blocked-status-variant-a-2026-07-29 | 2026-07-29T12:05:00Z | Fixes the central representation decision before the intent is confirmed. |

## Material Unknowns

None.
