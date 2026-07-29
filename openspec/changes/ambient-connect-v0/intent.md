# Intent Snapshot

- **Intent ID:** intent-ambient-connect-v0
- **Version:** 1
- **Status:** confirmed
- **Owner:** repository owner
- **Context Pack:** context-ambient-connect-v0 version 1
- **Run references:** none; this change authorizes no provider run

## Source Evidence

- **SE-1 (owner):** Goalrail must work in the background: connect once, then the user works in their own scaffold without any per-task Goalrail commands; act only in directories where Goalrail is initialized; close the question loop through the intent gate that already exists in the OpenSpec flow.
- **SE-2 (repository):** Both supported scaffolds register persistent user-level hooks that fire at session start (carrying context for the agent) and at session stop; the local Codex configuration already uses one.
- **SE-3 (repository):** The escalation channel, retention hygiene, announcement discipline, and the hook-response renderer already exist harness-side, and an answering intent version already records which escalation it resolves and with what disposition.
- **SE-4 (repository):** The workspace contract forbids observing unrelated sessions, and the promoted wrapper lifecycle is fail-closed — a posture that would break an ordinary user session if applied to background attachment.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Provide one-time connection: a single explicit command registers Goalrail's persistent session hooks in the scaffold's user configuration, with the user's consent, reversibly; a matching command removes them completely. After connection the user runs no Goalrail command per task. | Connect, observe registration; disconnect, observe complete removal. | SE-1, SE-2, CTX-1, CTX-3, CTX-4 |
| OUT-2 | Act only where Goalrail is initialized: the hook's first act is to check whether the session's directory is a Goalrail-initialized repository, and everywhere else it exits immediately, recording nothing and changing nothing. | Run a session in an unconnected directory and confirm zero Goalrail writes and zero retained observations. | SE-1, SE-4, CTX-7 |
| OUT-3 | At session start, in a connected directory: tell the agent the escalation channel exists, using a fixed ambient announcement under the same discipline as the launch announcement — no task hints, no provider names; and archive a stale question file left by an earlier session into the state root instead of attributing it to the new session. | Inspect the injected context and the archived file; confirm the new session starts with a clean reserved path. | SE-3, CTX-5 |
| OUT-4 | At session stop, in a connected directory: when the reserved path holds a question, retain its exact bytes append-only in the state root outside the repository, and bind the record to the directory's single active confirmed intent — its ID, version, and digest. When no single active intent is identifiable, retain the question unbound with an explicit reason instead of guessing. | Follow a recorded question to the intent it names; produce the unbound case and confirm the explicit reason. | SE-1, SE-3, CTX-6 |
| OUT-5 | Close the loop through the existing intent gate: the answer to a recorded question is a new intent version carrying the existing resolves-and-disposition record, and the next session works from that version. No second answer mechanism, dialogue, or resume exists. | Trace one question from its record to the answering intent version using recorded identifiers only. | SE-3, CTX-6 |
| OUT-6 | Fail quiet: in background mode a Goalrail malfunction never blocks, breaks, or delays the user's session — the session simply proceeds as if Goalrail were absent — while the wrapper lifecycle keeps its promoted fail-closed posture unchanged. | Break the helper deliberately and confirm the session runs normally; confirm wrapper behaviour is untouched. | SE-4, CTX-8 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | No observation, recording, or retention outside connected, Goalrail-initialized directories. | SE-4, CTX-7 |
| NG-2 | Background mode mints no run, no launch claim, no terminal receipt, and no `blocked` status: those belong to the wrapper lifecycle. A background session produces a question record, not a run outcome. | CTX-2, CTX-5 |
| NG-3 | No daemon, no control plane, no mid-run dialogue, no acknowledgement protocol, no background continuation, and no automatic answering. | SE-1, CTX-6 |
| NG-4 | Goalrail launches no provider in background mode and composes no task; the user's scaffold session is entirely the user's. | CTX-1 |
| NG-5 | No scaffold configuration is modified without the user's explicit consent, and disconnection leaves no residue. | SE-2, CTX-3 |
| NG-6 | The wrapper lifecycle, the kata, the benchmark, and the canonical schemas are unchanged. | CTX-2, CTX-8 |
| NG-7 | This snapshot authorizes no implementation, commit, push, pull request, merge, archival, provider run, or external effect. Each remains a separate owner gate. | CTX-1 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | Connection is one act and reversible. | One command registers the hooks with consent; one command removes them; repeating either is safe. | SE-2, CTX-3, CTX-4 |
| SIG-2 | Unconnected directories are invisible to Goalrail. | A session outside a connected directory produces zero Goalrail writes, reads beyond the initialization check, or retained observations. | SE-4, CTX-7 |
| SIG-3 | A question survives and stays attributable. | The retained bytes outlive the worktree file; a stale file is archived, never attributed to a newer session; the record names the intent it belongs to or an explicit unbound reason. | SE-3, CTX-5, CTX-6 |
| SIG-4 | The loop closes through the intent gate. | From a question record, recorded identifiers alone lead to the answering intent version and its disposition. | SE-3, CTX-6 |
| SIG-5 | Background failure is invisible; wrapper failure is loud. | A deliberately broken helper leaves the user's session working; the wrapper's fail-closed launch check still refuses. | SE-4, CTX-8 |

## Ambiguities and Unknowns

None blocking. One recorded limit: hook behaviour on both scaffolds is
established from configuration evidence and published schemas, not observed end
to end; a live session under a separate gate closes it.

## Confirmation

- **Confirmed by:** repository owner
- **Confirmed at:** 2026-07-29T17:00:06Z
- **Verification action:** Owner reviewed a plain-language owner-facing summary of candidate version 1 in the current Goalrail task — one-time consented connection, the initialized-directory gate, the session-start announcement with stale-file archival, session-stop retention bound to the active intent, loop closure through the existing intent gate, and the fail-quiet posture distinct from the wrapper's fail-closed one — asked one clarifying question confirming that ordinary tasks in ordinary projects are the primary scenario while katas remain benchmark material, received the affirmative answer, and explicitly confirmed the snapshot.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
