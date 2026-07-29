# Context Pack

- **Context Pack ID:** context-ambient-connect-v0
- **Version:** 1
- **Previous version:** pending
- **Started at:** 2026-07-29T16:30:00Z
- **Completed at:** 2026-07-29T16:55:03Z
- **Outcome:** sufficient

## Context Items

| ID | Kind | Claim | Source | Observed at | Relevance |
|---|---|---|---|---|---|
| CTX-1 | owner | The product works in the background: the user connects Goalrail once, then works in their own scaffold as usual, giving tasks directly. No `gr` commands per task, no Goalrail-launched sessions. The wrapper lifecycle stays as an internal benchmark tool, not the user experience. | owner:ambient-direction-2026-07-29 | 2026-07-29T16:10:00Z | Reverses the launch-model assumption behind the current adapter work: attachment replaces launching. |
| CTX-2 | repository | Every promoted local-run requirement describes the operator-driven wrapper lifecycle — prepare, one-shot start, finish. No promoted requirement covers attaching to a session Goalrail did not launch. | repo:openspec/specs/local-run/spec.md | 2026-07-29T16:32:00Z | Ambient attachment is a new capability, not a modification of the wrapper lifecycle. |
| CTX-3 | repository | The local Codex configuration already registers a persistent user-level `SessionStart` hook, and the pinned CLI exposes a persistent hook family including `SessionStart` (with an additional-context output) and `Stop`. | local:~/.codex/config.toml | 2026-07-29T16:40:00Z | One-time registration that fires in every session is a working mechanism on this machine, not a design hypothesis. |
| CTX-4 | external | Claude Code registers hooks in the user settings file with the same event family: session start carrying additional context for the agent, and stop. | local:~/.claude/settings.json | 2026-07-29T16:41:00Z | The same attachment shape works for a second scaffold, keeping the mechanism provider-neutral in substance. |
| CTX-5 | repository | The escalation channel is fully built harness-side: the reserved path, hygiene, append-only retention, the fixed announcement text, and a hook-response renderer that emits the announcement only for the session-start source that opens a session. | repo:internal/localrun/announcement.go | 2026-07-29T16:43:00Z | Ambient attachment reuses the built channel and renderer; the missing piece is only who registers the hook and what happens at session stop. |
| CTX-6 | repository | The intent gate already exists and already closes the question loop: flow intent confirmation is blocked while an ambiguity is unresolved, and an answering intent version records which escalation it resolves together with a disposition, parsed by the OpenSpec adapter. | repo:openspec/specs/intent-snapshot/spec.md | 2026-07-29T16:45:00Z | The answer to a run-time question is a new intent version through the existing gate; no second answer mechanism may be built. |
| CTX-7 | repository | The workspace contract forbids monitoring unrelated user sessions and copying unnecessary private content. | repo:openspec/specs/local-run/spec.md | 2026-07-29T16:46:00Z | A persistent hook fires for every session of the user, so acting only inside explicitly connected directories is a hard requirement, not a preference. |
| CTX-8 | repository | The wrapper lifecycle is fail-closed by promoted requirement: a launch that cannot announce the channel is refused. | repo:openspec/specs/local-run/spec.md | 2026-07-29T16:47:00Z | Ambient attachment needs the opposite posture — a Goalrail malfunction must never break the user's ordinary session — and the difference must be stated, not implied. |

## Material Unknowns

None blocking. One bounded limit is recorded rather than resolved: hook
behaviour on both scaffolds is established from configuration evidence and
published schemas, not yet observed end to end in a live session. The same
limit already applies to the reserved path's writability inside the provider
sandbox; a live session closes them together.
