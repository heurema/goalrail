## Why

`gr health` detects a registration whose executable has moved, refuses to call
the attachment working, and tells the user to re-run the consented connection
command. That command then does nothing: presence is decided by finding the
managed marker inside a handler, never by checking what the handler points at.
The tool diagnoses a real failure and prescribes a remedy it cannot perform,
while the attachment stays silent in every session — background mode is
fail-quiet, so nothing else surfaces the breakage.

Recorded as issue #25 by a test written for the previous change, which seeded a
registration carrying an old executable path and found that nothing repaired it.

## What Changes

- Decide connection presence by what the registration points at, not only by the
  marker being there: a managed handler naming a different executable is treated
  as needing repair.
- Replace rather than accompany: the managed block is removed before it is
  rewritten, and each event's handler is swapped, so a repair cannot leave two
  registrations or a removal that finds only the first.
- Leave a registration that already names the current executable byte-identical,
  keep foreign handlers untouched, and keep the repaired session-start entry
  scoped to the occurrence that opens a session.
- Disclose that a repaired handler needs the scaffold's review step again, under
  the same per-scaffold discipline the existing disclosure follows: asserted
  where the gate was observed, marked unverified where it was not. A repair is
  the only moment Goalrail knows a trust record went stale, because it changed
  the definition itself.
- Make the connection command's own test helper name its executable explicitly.
  The two idempotency tests passed a freshly created path on each call, so they
  never pinned same-binary idempotency at all — a gap this change exposes and
  closes.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Presence is judged by the executable the registration names. | OUT-1, SIG-1 | NG-1 |
| Repair replaces in place for both scaffolds. | OUT-2, SIG-1 | NG-3 |
| A current registration stays byte-identical. | OUT-3, SIG-2 | NG-3 |
| Foreign handlers, and events already current, survive a repair. | OUT-4, SIG-3 | NG-3 |
| The repaired session-start entry stays scoped. | OUT-5, SIG-4 | NG-3 |
| The repair discloses that trust applies again. | OUT-6, SIG-6 | NG-2 |
| Health's existing report becomes actionable rather than changed. | OUT-1, SIG-5 | NG-1 |
| The test helper names its executable, pinning OUT-3 for the first time. | OUT-3, SIG-2 | NG-3 |

No proposed change lies outside this table. A draft rule requiring the plan to
name the stale executable was removed rather than promoted: it traced to no
confirmed outcome, and an untraceable requirement in promoted text is the drift
this table exists to prevent. The plan still carries the path as an
implementation detail, which a test pins without the contract claiming it.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ambient-connect`: connection presence accounts for the executable a
  registration names; a stale registration is repaired in place rather than
  reported as present; a repair discloses that the scaffold's review step
  applies again; a remedy a health report names must be one the named command
  can perform.

## Impact

- `internal/ambient/connect.go`: presence detection gains executable awareness;
  both scaffolds' write paths replace rather than append; the plan carries the
  stale path.
- `internal/ambient/health.go`: gains the repair disclosure text alongside the
  existing connection notice. Its detection of a missing or unrunnable
  executable, its trust reporting, and its state machine are unchanged.
- `cmd/gr/ambient.go`: connection output reports a repair and never calls a
  repaired attachment active, because the trust record it would read was
  recorded against the command that was just replaced.
- `internal/ambient/connect_test.go`: the shared helper takes an explicit
  executable so idempotency is pinned against a stable path.
- No change to the announcement text, retention, intent binding, fail-quiet
  posture, the wrapper lifecycle, or any canonical schema.
- No new dependency, service, credential, or deployment surface.

## Non-Goals

- Do not change how health detects a missing executable or reports trust state;
  this change makes connection able to act on what health already diagnoses.
- Do not write, compute, or reproduce a scaffold trust record. Disclosing that
  review is needed again is not granting or simulating it.
- Do not change the announcement, retention, intent binding, fail-quiet posture,
  or the wrapper lifecycle.
- Do not adopt project-level registration or reopen any other deferred decision.
- Planning completion authorizes no implementation, commit, push, pull request,
  merge, archival, provider run, or external effect. Each remains a separate
  owner gate.
