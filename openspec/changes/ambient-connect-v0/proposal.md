## Why

The product direction reverses the launch model. Goalrail today is a wrapper:
the operator prepares a WorkSpec and Goalrail launches the provider. The
product is the opposite — the user connects Goalrail once, then works in their
own scaffold, giving ordinary tasks; Goalrail attaches in the background. The
wrapper lifecycle remains an internal benchmark tool, not the user experience.

The escalation channel this makes reachable is already built: the reserved
path, hygiene, append-only retention, the announcement text and its renderer,
and the intent-side answering record all exist. What does not exist is the
attachment itself: nothing registers Goalrail in the user's scaffold, nothing
scopes it to consented directories, and nothing carries a background session's
question into the intent gate.

Both supported scaffolds already provide the needed mechanism — persistent
user-level hooks firing at session start (with context for the agent) and at
session stop. The local Codex configuration already runs one such hook for an
unrelated purpose, so one-time registration is a working mechanism on this
machine, not a hypothesis.

## What Changes

- Add one-time connection: an explicit, consented, reversible command registers
  Goalrail's persistent session hooks in the scaffold's user configuration; a
  matching command removes them without residue.
- Gate every hook invocation on directory initialization: outside a
  Goalrail-initialized repository the hook exits immediately, recording and
  changing nothing.
- At session start in a connected directory: inject the fixed ambient
  announcement under the launch announcement's discipline, and archive a stale
  question file into the state root instead of attributing it to the new
  session.
- At session stop in a connected directory: retain a written question
  append-only outside the repository and bind the record to the directory's
  single active confirmed intent, or retain it unbound with an explicit reason.
- Close the loop through the existing intent gate: the answer is a new intent
  version carrying the existing resolves-and-disposition record.
- State the fail-quiet posture: background malfunction never breaks the user's
  session, while the wrapper lifecycle keeps its promoted fail-closed posture.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| One-time consented, reversible connection with no per-task commands. | OUT-1, SIG-1 | NG-5 |
| Initialized-directory gate with instant, recordless exit elsewhere. | OUT-2, SIG-2 | NG-1 |
| Session-start announcement plus stale-file archival. | OUT-3, SIG-3 | NG-4 |
| Session-stop retention bound to the active intent or explicitly unbound. | OUT-4, SIG-3 | NG-2 |
| Loop closure through the existing intent gate. | OUT-5, SIG-4 | NG-3 |
| Fail-quiet background posture beside the unchanged fail-closed wrapper. | OUT-6, SIG-5 | NG-6 |

## Capabilities

### New Capabilities

- `ambient-connect`: background attachment to scaffold sessions — consented
  registration, the initialized-directory gate, session-start announcement,
  session-stop question retention and intent binding, and the fail-quiet
  posture.

### Modified Capabilities

None. The wrapper lifecycle in `local-run` is untouched.

## Impact

- `cmd/gr`: the connection, disconnection, initialization, and hook-entry
  surfaces.
- A new internal package for the ambient helper logic: the directory gate,
  announcement injection, stale-file archival, question retention, and intent
  binding, reusing the existing escalation hygiene, retention primitives, and
  the hook-response renderer.
- Scaffold user configuration is touched only by explicit consented
  connection and restored by disconnection.
- `goalrail.work-spec/v0`, `goalrail.terminal-receipt/v1`,
  `goalrail.escalation/v0`, the wrapper lifecycle, the kata, and the benchmark
  are unchanged.
- No new dependency, daemon, service, credential, or deployment surface.

## Non-Goals

- No observation, recording, or retention outside connected initialized
  directories.
- Background mode mints no run, launch claim, terminal receipt, or `blocked`
  status; a background session produces a question record, not a run outcome.
- No daemon, control plane, mid-run dialogue, acknowledgement protocol,
  background continuation, or automatic answering.
- Goalrail launches no provider in background mode and composes no task.
- No scaffold configuration change without explicit consent; disconnection
  leaves no residue.
- Planning completion authorizes no implementation, commit, push, pull
  request, merge, archival, provider run, or external effect. Each remains a
  separate owner gate.
