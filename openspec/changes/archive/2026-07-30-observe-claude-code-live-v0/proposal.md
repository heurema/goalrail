## Why

The promoted record says the second supported scaffold has never been exercised
live, and lists three behaviours as unverified. One live run has now settled two
of them. Leaving the record as it stands understates what is known, and a future
reader consulting it would repeat work already done.

The same run leaves the third — whether the scaffold asks the user to approve a
registered hook — exactly as open as before, for reasons worth recording: the run
was non-interactive, where the provider documents that the trust dialog is
skipped, and the hooks were supplied for that run rather than registered. An
absent prompt under those conditions is not evidence.

The reason the earlier attempt was abandoned also turns out to be wrong. It was
recorded that this scaffold's login state is tied to the home directory, leaving
only routes that wrote into the owner's working configuration or copied an
account file. Credentials are in fact held in the operating system keychain, and
the provider's CLI accepts settings for a single run — so the observation needed
neither of the routes that were refused.

## What Changes

- Record announcement delivery and question retention on that scaffold as
  observed, and remove them from the list of what is unverified.
- Keep the approval question open, and state the two conditions that make the
  run inconclusive about it rather than only the conclusion.
- Withdraw the superseded reason for the earlier abandonment, so no requirement
  rests on a blocker that does not exist.
- Keep the evidence in the repository — the invocation, the fixture's shape, the
  agent's reply, the question it wrote, and the retained record — so the
  observation survives the conversation that produced it.
- Add two general scenarios, because this situation will recur: a partly observed
  scaffold must be recorded as partly observed, and an absence produced under
  conditions where the behaviour would not have appeared anyway is not evidence.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Observed behaviours recorded as observed. | OUT-1, SIG-1 | NG-2 |
| The approval question stays open, with its reasons. | OUT-2, SIG-2 | NG-1 |
| Evidence kept in the repository. | OUT-3, SIG-3 | NG-3 |
| The withdrawn blocker no longer supports any claim. | OUT-4 | NG-3 |
| Two general scenarios for partial observation and inconclusive absence. | OUT-1, OUT-2, SIG-1, SIG-2 | NG-1 |

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ambient-connect`: the second scaffold's record distinguishes what a live run
  established from what it did not; the approval question stays open with its
  reasons; the superseded blocker is withdrawn.

## Impact

- `openspec/specs/ambient-connect/spec.md`: one requirement's record of what has
  been observed.
- `openspec/changes/observe-claude-code-live-v0/evidence/`: the run's artifacts.
- No Go change. No behaviour changes: not the registration shape, the
  announcement, retention, intent binding, health reporting, or the wrapper
  lifecycle.
- No new dependency, service, credential, or deployment surface.

## Non-Goals

- Do not claim the approval question is answered, and do not treat the absent
  prompt in a non-interactive run as evidence about an interactive one.
- Do not change any behaviour or any code.
- Do not register hooks in the owner's working configuration, and do not copy
  account files, to obtain further observation.
- Do not adopt project-level registration or reopen any other deferred decision.
- Planning completion authorizes no commit, push, pull request, merge, archival,
  or further provider run. Each remains a separate owner gate.
