# Context Pack

- **Context Pack ID:** context-escalation-discoverability-v0
- **Version:** 1
- **Previous version:** pending
- **Started at:** 2026-07-29T13:55:00Z
- **Completed at:** 2026-07-29T14:07:48Z
- **Outcome:** sufficient

## Context Items

| ID | Kind | Claim | Source | Observed at | Relevance |
|---|---|---|---|---|---|
| CTX-1 | repository | The promoted blocked-outcome requirement defines how an escalation is admitted, retained, and scored, but no promoted requirement states that a run is ever told the channel exists. The reserved path is a constant inside Goalrail and appears in no artifact the provider receives. | repo:openspec/specs/local-run/spec.md | 2026-07-29T13:56:00Z | The capability is currently unreachable in practice: an agent cannot use a channel whose existence it cannot learn. |
| CTX-2 | repository | The Codex adapter passes the provider only a working directory, the operator's verbatim adapter arguments, and an environment carrying the run-context value. The frozen WorkSpec task statement is never handed to the provider. | repo:internal/adapters/codex/local_run.go | 2026-07-29T13:57:00Z | Nothing Goalrail already sends the provider can carry the announcement, so the mechanism has to be named rather than assumed. |
| CTX-3 | repository | Goalrail already renders an invocation-local Codex `SessionStart` hook that invokes the local capsule with a fixed subcommand, and the hook is already wired for lineage correlation. | repo:internal/adapters/codex/hook_config.go | 2026-07-29T13:58:00Z | A launch-time delivery point already exists and is already trusted for one purpose; a second purpose needs no new provider surface. |
| CTX-4 | external | The Codex hook output contract defines `hookSpecificOutput` for `SessionStart` with an optional `additionalContext` string, alongside `continue`, `stopReason`, `suppressOutput`, and `systemMessage`. | codex-cli:SessionStartHookSpecificOutputWire | 2026-07-29T14:02:00Z | The existing hook can carry text into the session, so the announcement can travel the path Goalrail already establishes. |
| CTX-5 | repository | The canonical WorkSpec is provider-neutral by promoted requirement and must not contain provider names, provider launch arguments, provider session identity, or approval state. | repo:openspec/specs/work-spec/spec.md | 2026-07-29T13:59:00Z | Any announcement mechanism must live in the adapter boundary; it cannot become a WorkSpec field. |
| CTX-6 | external | A twelve-run A/B measured 6/6 agents returning a structured question when the environment accepted one machine-checkably and 0/6 when it did not. The authors record an honest caveat: the treatment branch was loud, with the channel occupying about a third of the task statement plus a leaked version-name hint, so 6/6 may be inflated. The measurement lives in a separate repository, so the claims are retained verbatim next to this pack. | repo:openspec/changes/escalation-discoverability-v0/evidence/measurement-extract.md | 2026-07-29T13:50:00Z | Establishes that expressibility drives the behaviour, and that how loudly the channel is announced is itself a variable rather than a detail. |
| CTX-7 | external | In the measured kata the channel consisted of a documented return format plus an accepting branch in the return script; the control copy differed only by their absence. Both responsibilities are now duplicated by the harness. Retained verbatim next to this pack, because the kata is a separate repository. | repo:openspec/changes/escalation-discoverability-v0/evidence/measurement-extract.md | 2026-07-29T13:52:00Z | Removing them from the kata is only safe once the harness announces the channel itself, or the measurement becomes unrepeatable. |
| CTX-8 | repository | The promoted local-run capability keeps a blocked outcome terminal and forbids Goalrail from creating a dialogue, control plane, or background continuation from it. | repo:openspec/specs/local-run/spec.md | 2026-07-29T13:54:00Z | The announcement must be a launch-time statement, not a conversation or a negotiated capability handshake. |
| CTX-9 | repository | The provider signals session start for four distinct sources — startup, resumption, clearing, and compaction — and the rendered hook matcher does not distinguish them. | repo:internal/adapters/codex/correlation.go | 2026-07-29T14:35:00Z | An exactly-once announcement must accompany only the source that opens a run, or one statement becomes a recurring instruction. |
| CTX-10 | repository | The Codex adapter has no operating-system launcher in this repository; the launcher is supplied by a separate activation. Whether the announcement reaches the session therefore depends on a component the adapter only invokes. | repo:internal/adapters/codex/local_run.go | 2026-07-29T14:36:00Z | The adapter cannot truthfully report delivery on its own; the report must come from the component that completes the path. |

## Material Unknowns

None blocking. One bounded limit is recorded rather than resolved: the
`additionalContext` contract is read from the provider's own published hook
schema, and no run has yet observed the text arriving in an agent's context.
Confirming that requires a real provider run, which is a separate owner gate.
The same limit already applies to the reserved path's writability inside the
provider sandbox.
