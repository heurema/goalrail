# Live Codex correlation probe

Date: 2026-07-21

Scope: one disposable Codex CLI session and one disposable Codex App task.
The temporary repository `SessionStart` hook was removed immediately after the
probe. Neither session could edit the repository, and neither session called
tools or read transcripts.

## CLI result: VERIFIED

- Launch context was process-scoped through `GOALRAIL_RUN_CONTEXT`.
- `change_id`: `intent-canary-v0`
- `run_id`: `run-live-cli-001`
- Hook-observed root `session_id`: `019f8454-5fa3-7cf1-b96b-bb869a23cfbb`
- CLI event-stream thread ID: `019f8454-5fa3-7cf1-b96b-bb869a23cfbb`
- Hook event: `SessionStart`, matcher `startup`
- Context SHA-256: `b10561007322a24662705195df37de102cea34aedb321ecf0e86a93d78b43e77`
- Result: the hook receipt and CLI event stream independently exposed the same
  session identity. No owner copied an identifier and no shared mutable change
  pointer was used.

## Codex App result: UNSUPPORTED on the tested path

- App task reference: `019f8454-d971-7570-8290-d9685eafdf4d`
- The disposable App task completed with its exact acknowledgement and no tool
  calls.
- No `SessionStart` receipt was produced, and the task event log contained no
  hook event or hook diagnostic.
- The tested App background-task path therefore does not currently provide a
  verified project-hook binding. This is recorded as `UNSUPPORTED`, not
  `UNLINKED`, because the hook did not run and its session identity was not
  observed through the required hook contract.
- Whether the missing hook is caused by project-hook trust handling or by the
  App background-task startup path is not distinguishable from the exposed
  evidence. No second attempt or heuristic fallback was used.

## Gate consequence

CLI binding is feasible with immutable per-process context. Deterministic App
binding is not established. Task 1.4 must stop at an owner gate rather than
introducing manual ID copying, transcript parsing, repository-wide mutable
state, or a Langfuse plugin fork.

## Owner gate resolution

On 2026-07-21 the owner explicitly approved the provider-authoritative App
launch receipt: Goalrail-launched App tasks use the immutable `threadId`
returned by `create_thread`; CLI runs use lifecycle-hook identity. App tasks
created outside Goalrail remain `UNLINKED` and ineligible for the v0 canary.
The decision adds no manual identifier copying, transcript parsing, shared
mutable current-change state, or Langfuse plugin fork.
