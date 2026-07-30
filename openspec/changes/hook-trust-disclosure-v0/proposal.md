## Why

The first live verification of background attachment proved the built chain
works: the announcement reached the agent, which quoted the reserved path
verbatim; given a contradictory task the same model wrote a question instead of
guessing and left the source untouched; the stop hook retained that question
with its own identifier and an explicit unbound reason.

It also exposed a gap. A registered hook does not run until the user reviews and
trusts its exact definition through the scaffold's own surface. Connection says
nothing about this, so the user connects, works, and observes nothing — with no
message, warning, or error. The most likely conclusion is that the product is
broken.

The structurally stronger arrangement was attempted and does not work. Moving
the attachment into the repository would make "we do nothing in unconnected
directories" a fact of layout rather than a promise of our code. Project-local
hooks are silently skipped with no trust prompt even in a trusted project, the
session-start event has passed by the time the user can act, and the defect is
open with no workaround. The user-scope arrangement is therefore the only
working one, not a preference.

## What Changes

- Connection discloses the remaining step: it states that the scaffold requires
  the user to review and trust the registered hooks, names the surface where
  that happens, and says the attachment does not act until then.
- A diagnostic command reports attachment health: scaffold connected,
  repository initialized, hooks trusted — with the next action for whatever is
  missing.
- The contract forbids forging trust: Goalrail must not write, compute, or
  reproduce the scaffold's hook-trust records, and must not automate the user's
  approval.
- The contract records the accepted scope limitation with its cause and
  external evidence, rather than presenting user-scope registration as a design
  preference.
- The live-verified behaviour — announcement, retention, intent binding,
  fail-quiet posture — is untouched.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Connection states the trust requirement and names the surface. | OUT-1, SIG-1 | NG-3 |
| Diagnostic command answers every attachment state with a next action. | OUT-2, SIG-2 | NG-4 |
| Contract forbids forging or automating trust. | OUT-3, SIG-3 | NG-1 |
| Contract records the limitation, its cause, and the blocked alternative. | OUT-4, SIG-4 | NG-2 |
| The verified chain keeps its behaviour and its tests. | OUT-5, SIG-5 | NG-3 |

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ambient-connect`: connection discloses the trust step; attachment health is
  reportable on demand; forging trust is prohibited; the user-scope limitation
  is recorded with its cause.
- `operator-cli-help`: help presents the diagnostic command alongside the
  existing background surface.

## Impact

- `cmd/gr`: connection output gains the disclosure; a diagnostic subcommand is
  added.
- `internal/ambient`: a read-only inspection of connection, initialization, and
  trust state. Reading the scaffold configuration to report trust is
  observation; writing it remains prohibited.
- No change to the announcement text, retention, intent binding, fail-quiet
  behaviour, the wrapper lifecycle, or any canonical schema.
- No new dependency, daemon, service, or credential surface.

## Non-Goals

- Do not write, compute, or reproduce scaffold trust records, and do not
  automate the user's approval in any form.
- Do not adopt project-local hooks while they are silently skipped, and do not
  work around that external defect by other means.
- Do not change the announcement, retention, intent binding, fail-quiet
  posture, or the wrapper lifecycle.
- No daemon, control plane, dialogue, or polling of scaffold state.
- Planning completion authorizes no implementation, commit, push, pull request,
  merge, archival, provider run, or external effect. Each remains a separate
  owner gate.
