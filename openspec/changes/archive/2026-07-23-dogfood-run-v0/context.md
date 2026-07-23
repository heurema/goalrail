# Context Pack

- **Context Pack ID:** `context-dogfood-run-v0`
- **Version:** 1
- **Previous version:** pending
- **Started at:** 2026-07-23T05:59:26Z
- **Completed at:** 2026-07-23T06:02:07Z
- **Outcome:** sufficient

## Context Items

| ID | Kind | Claim | Source | Observed at | Relevance |
|---|---|---|---|---|---|
| CTX-1 | repository | Goalrail is being rebuilt around controlled coding-agent delivery in a real repository while the product team retains priorities, product decisions, and review. | repo:README.md | 2026-07-23T05:59:30Z | A first dogfood run should exercise one real local repository task and keep the operator as decision owner. |
| CTX-2 | repository | Goalrail requires context before candidate intent, keeps OpenSpec replaceable, and prefers the smallest reversible slice without removing retained target architecture layers. | repo:AGENTS.md | 2026-07-23T05:59:35Z | The WorkSpec and flow must remain provider-neutral and must not pull future infrastructure into v0. |
| CTX-3 | repository | The accepted lifecycle states that already-started runs remain governed by a frozen WorkSpec or manifest snapshot and that intent, implementation, effects, Git actions, and activation retain separate gates. | repo:docs/project-lifecycle.md | 2026-07-23T06:01:40Z | The v0 flow needs one immutable run input and explicit operator gates without treating intent as execution authority. |
| CTX-4 | repository | The only current executable entry point is the canary-specific `goalrail-canary` command with admission, lineage, checks, terminal evidence, assessment, and reporting actions; there is no general `gr` run surface. | repo:cmd/goalrail-canary/main.go | 2026-07-23T06:00:45Z | A minimal `gr` flow is a new general dogfood surface and should reuse proven boundaries without inheriting the full canary protocol. |
| CTX-5 | repository | The canonical domain already separates Intent Snapshot data from execution lineage, and verified lineage records change, run, root provider session, identity source, and context digest without granting effect authority. | repo:internal/domain/types.go | 2026-07-23T06:00:55Z | The WorkSpec can reference existing intent and lineage semantics instead of defining provider-specific identity as canonical state. |
| CTX-6 | repository | The current operator flow launches a local Codex child with immutable per-run context, derives the run ID, binds provider-authoritative session identity, freezes checks before terminal verification, and keeps append-only evidence inspectable. | repo:canary/intent-canary-v0/operator-flow.md | 2026-07-23T06:01:50Z | These are tested local patterns for an operator-assisted run, but the dogfood flow can select only the subset needed for one trusted run. |
| CTX-7 | repository | Accepted decisions keep runtime-owned artifacts outside OpenSpec archives and make OpenSpec a development-time compiler rather than a runtime source of truth. | repo:docs/decisions/0001-decouple-canary-runtime-artifacts-from-openspec.md | 2026-07-23T06:01:45Z | A run-pinned WorkSpec must be canonical Goalrail data even if its inputs originate in an OpenSpec change. |
| CTX-8 | repository | Current real canaries remain unactivated, and local planning or implementation does not authorize credentials, external calls, deployment, publication, or other effects. | repo:docs/project-lifecycle.md | 2026-07-23T06:01:40Z | The dogfood milestone must not be interpreted as canary activation or expanded effect authority. |

## Material Unknowns

None.
