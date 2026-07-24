# Context Pack

- **Context Pack ID:** `context-activate-dogfood-run-v0`
- **Version:** 1
- **Previous version:** pending
- **Started at:** 2026-07-24T05:03:18Z
- **Completed at:** 2026-07-24T05:14:15Z
- **Outcome:** sufficient

## Context Items

| ID | Kind | Claim | Source | Observed at | Relevance |
|---|---|---|---|---|---|
| CTX-1 | repository | Goalrail is being rebuilt around controlled coding-agent delivery in real repositories while the product team retains priorities, product decisions, and review. | repo:README.md | 2026-07-24T05:04:00Z | The first activated run must remain owner-assisted and end at a reviewable local result. |
| CTX-2 | repository | Goalrail requires a versioned candidate Intent Snapshot before proposal or implementation, treats confirmed intent as distinct from effect authority, and prefers the smallest reversible slice without removing retained target architecture layers. | repo:AGENTS.md | 2026-07-24T05:04:10Z | This change may define one activation boundary but cannot treat intent confirmation as general provider or external-effect authority. |
| CTX-3 | repository | The confirmed `dogfood-run-v0` intent defines one provider-neutral WorkSpec, one explicit operator start, at most one trusted local Codex run, provider-enforced sandbox and approvals, bounded terminal evidence, and a separate later activation gate. | repo:openspec/changes/archive/2026-07-23-dogfood-run-v0/intent.md | 2026-07-24T05:06:00Z | Activation should exercise the accepted contract rather than broaden or replace it. |
| CTX-4 | repository | The current local-run specification requires production `start` to fail with `ACTIVATION_REQUIRED` before creating a run ID, launch claim, or provider invocation until a separate activation change is accepted. | repo:openspec/specs/local-run/spec.md | 2026-07-24T05:06:20Z | This change must replace the denial only for the exact authorized run and remain fail-closed otherwise. |
| CTX-5 | repository | The canonical WorkSpec is provider-neutral and binds confirmed intent, an exact repository root and base revision, bounded paths, ordered checks, stop conditions, and the `trusted-local-provider-enforced-v0` posture. | repo:openspec/specs/work-spec/spec.md | 2026-07-24T05:06:30Z | The activation boundary can pin the exact dogfood task without adding Codex-specific fields to WorkSpec semantics. |
| CTX-6 | repository | The local-run service denies `start` when no adapter is configured, creates an immutable launch claim before invoking an adapter, and rejects a duplicate launch claim for the same prepared WorkSpec. | repo:internal/localrun/service.go | 2026-07-24T05:11:20Z | Existing state transitions already provide the fail-closed, at-most-one-launch basis needed for a single run. |
| CTX-7 | repository | A Codex local-run adapter already exists; it launches in the pinned repository directory, passes operator-supplied provider arguments through, binds provider-authoritative session identity, filters credential-shaped base environment variables, and returns a bounded provider observation. | repo:internal/adapters/codex/local_run.go | 2026-07-24T05:11:30Z | The slice can use the implemented adapter and preserve Codex sandbox and approval enforcement instead of creating a new execution boundary. |
| CTX-8 | repository | `gr help` currently prints only `usage: gr <prepare\|inspect\|start\|finish> [flags]`; command flag sets expose individual flags, existing command results are JSON, and tests already protect pre-activation denial and keep fixture activation out of help. | repo:cmd/gr/main.go; repo:cmd/gr/main_test.go | 2026-07-24T05:10:00Z | Improving built-in help is a small real task that can preserve JSON outputs and execution semantics while exercising the bounded run. |
| CTX-9 | repository | The local-run guide documents the prepare, inspect, start, and finish lifecycle, keeps state outside the repository, and explicitly excludes automated checks, Git lifecycle actions, credentials, external effects, retries, and background continuation. | repo:docs/local-run-v0.md | 2026-07-24T05:09:00Z | The guide is both part of the bounded help task and a source for the user-visible lifecycle and safety boundary. |
| CTX-10 | repository | Before this change was created, the worktree was clean at detached `HEAD` `a9268126ae09048918d4627d8b730a42640fe61c`, exactly matching the current local `origin/main`. | repo:git@a9268126ae09048918d4627d8b730a42640fe61c | 2026-07-24T05:13:55Z | A later WorkSpec can pin the exact reviewed base revision, and any drift must fail closed before launch. |

## Material Unknowns

None.
