## Why

Studying the second scaffold's documentation before attempting a live run found
a defect in shipped code and an overreach in a promoted requirement.

Its session-start matcher takes one exact value per entry — the values that open
a session and the ones that recur inside it are distinguished there, not in the
hook. Omitting the matcher makes the hook fire on every occurrence. The shipped
connection routine writes no matcher at all, so the registration it produces is
exactly that shape: the announcement would accompany resumption, clearing,
compaction, and forking. The promoted contract requires the opposite, and the
first scaffold satisfies it because its transport inspects the hook source; the
second has no equivalent guard.

The same reading corrected a second thing. A promoted requirement records that
registering hooks inside the repository is externally blocked. That evidence came
from one provider, where project-local hooks are silently skipped behind a trust
gate. For this scaffold, project-level hooks are an ordinary settings layer that
merges with user settings. A true observation about one provider was written as a
property of attachment in general.

## What Changes

- Register one session-start entry per matcher value that opens a session, so
  the announcement cannot accompany a recurring session event.
- Keep removal exact under the new shape: disconnection removes every entry the
  connection added across all matcher values and leaves foreign entries alone.
- Correct the recorded limitation to name the scaffold its evidence came from,
  and state that the other scaffold's project layer merges ordinarily rather
  than being a blocked route.
- Record that the second scaffold has never been exercised in a live session,
  and that this fix rests on its published matcher contract together with the
  owner's own working configuration, which agree with each other.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| One registration per opening matcher value. | OUT-1, SIG-1 | NG-3 |
| Exact removal across all matcher values, foreign entries untouched. | OUT-2, SIG-2 | NG-3 |
| Attribute the limitation to the scaffold it came from. | OUT-3, SIG-3 | NG-1 |
| Record what remains unobserved for the second scaffold. | OUT-4, SIG-4 | NG-4 |

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ambient-connect`: session-start registration is per opening matcher value on
  scaffolds that distinguish them; the recorded scope limitation is attributed
  to the scaffold whose evidence produced it; the second scaffold's unobserved
  end-to-end behaviour is stated.

## Impact

- `internal/ambient/connect.go`: the second scaffold's registration and removal
  gain matcher awareness.
- No change to the announcement text, retention, intent binding, fail-quiet
  posture, health reporting, the wrapper lifecycle, or any canonical schema.
- The first scaffold's registration is unchanged: its transport already
  distinguishes the opening event from recurring ones.
- No new dependency, service, credential, or deployment surface.

## Non-Goals

- Do not adopt project-level registration here; correcting the record is not
  the same as changing where hooks are installed, and that move deserves its
  own decision with its own evidence.
- Do not write into the owner's working scaffold configuration, and do not copy
  account files containing unrelated history, to obtain a live run.
- Do not change the announcement, retention, intent binding, fail-quiet
  posture, health reporting, or the wrapper lifecycle.
- Do not claim the second scaffold is verified end to end.
- Planning completion authorizes no implementation, commit, push, pull request,
  merge, archival, provider run, or external effect. Each remains a separate
  owner gate.
