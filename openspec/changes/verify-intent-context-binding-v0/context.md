# Context Pack

- **Context Pack ID:** context-verify-intent-context-binding-v0
- **Version:** 1
- **Previous version:** pending
- **Started at:** 2026-07-25T08:10:00Z
- **Completed at:** 2026-07-25T08:30:05Z
- **Outcome:** sufficient

## Context Items

| ID | Kind | Claim | Source | Observed at | Relevance |
|---|---|---|---|---|---|
| CTX-1 | repository | Preparation already resolves the sibling `context.md` declared by a confirmed intent, confines it to the canonical repository root, applies the intent size bound, validates the resulting flow snapshot, and fails closed on missing, unreadable, mismatched, or symlink-escaping context before any state is written. The behaviour is implemented and covered by regression tests. | git:codex/bind-intent-context-v0@b36fabd | 2026-07-25T08:12:00Z | The behaviour exists but no promoted capability states it, so this change adds a spec home rather than new behaviour. |
| CTX-2 | repository | The promoted `local-run` capability on `main` never mentions the Context Pack. Its `Run preparation is inspectable and does not launch` requirement lists the confirmed intent reference only as one invalid-input case in a scenario. | repo:openspec/specs/local-run/spec.md@32913f19 | 2026-07-25T08:14:00Z | Identifies the exact promoted requirement that must be modified and confirms nothing is being duplicated. |
| CTX-3 | repository | The only wording that describes this enforcement lives in the unmerged local change `activate-dogfood-run-v0`, embedded inside a `MODIFIED Requirements` block for `Real activation remains separately gated`. `main` rewrote that requirement in PR #12, so the wording cannot be transplanted as-is. | repo:openspec/changes/activate-dogfood-run-v0/specs/local-run/spec.md | 2026-07-25T08:20:00Z | The enforcement must be extracted into its own standalone requirement instead of reusing an entangled and superseded block. |
| CTX-4 | repository | That same unmerged change declares `intent-activate-dogfood-run-v0` version 1 as confirmed, while the archived change `2026-07-24-activate-dogfood-run-v0` merged into `main` declares the identical intent ID and version with different confirmed wording. | repo:openspec/changes/archive/2026-07-24-activate-dogfood-run-v0/intent.md@32913f19 | 2026-07-25T08:20:00Z | The existing intent ID cannot be reused or re-versioned without rewriting merged evidence, so this change requires a new intent identity. |
| CTX-5 | repository | The promoted `intent-snapshot` and `context-pack` capabilities require a flow Intent Snapshot to reference a valid Context Pack and require every derived item to carry resolvable context IDs, but neither states enforcement at WorkSpec preparation time. | repo:openspec/specs/intent-snapshot/spec.md; repo:openspec/specs/context-pack/spec.md | 2026-07-25T08:16:00Z | Confirms the gap is preparation-time enforcement only; the semantic contract already exists and must not be restated. |
| CTX-6 | repository | Remote `main` is `32913f19e511b63cf0456b0a8110a21e266c506c`. PR #13 is open and modifies the same `local-run` capability file with an unrelated hook-linkage requirement. | git:origin/main@32913f19; github:heurema/goalrail/pull/13 | 2026-07-25T08:28:00Z | Both changes edit `openspec/specs/local-run/spec.md` only at archive time, so they do not conflict while both remain unarchived. |
| CTX-7 | external | The owner directed reconciliation onto merged `main` first, chose to ship the hook-linkage fix alone in PR #13, and deferred this behaviour until its requirement has a valid spec home. | owner:goalrail-reconciliation-decision-2026-07-25 | 2026-07-25T08:05:00Z | Defines the scope of this change as spec-only and keeps the already-written code unchanged. |

## Material Unknowns

None.
