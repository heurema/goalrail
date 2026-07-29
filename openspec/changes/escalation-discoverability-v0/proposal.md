## Why

The escalation channel is admitted, retained, and scored, but nothing tells a
run it exists. The reserved path is a constant inside Goalrail; the adapter
hands the provider only a working directory, the operator's verbatim arguments,
and the run-context environment value; and the frozen task statement never
reaches the provider at all. No promoted requirement states that the channel is
ever announced.

A channel nobody can find is not a delivered capability. It is also the exact
failure the channel exists to prevent: the measured behaviour was that agents
return a structured question when the environment accepts one and guess when it
does not, and an unannounced channel is indistinguishable from an absent one.

The measurement that motivates this work also recorded that its treatment branch
was loud — the channel occupied about a third of the task statement, and a hint
leaked through a version name — so its authors flagged 6/6 as possibly inflated.
How the channel is announced is therefore a variable, not a formatting decision.

## What Changes

- Require that a launched run is told the escalation channel exists: the
  reserved path, the condition that it applies when the work item cannot be
  completed as specified, the requirement that the declared scope stays
  untouched, and where the payload shape is published.
- Deliver the announcement through the adapter boundary at launch, reusing the
  provider hook Goalrail already renders for lineage rather than adding a second
  provider surface.
- Keep the announced text provider-neutral and fixed; make transport the
  adapter's responsibility. An adapter with no delivery path fails the launch
  explicitly instead of starting a run whose escalation channel is unreachable.
- Announce once, at launch, as a statement: no acknowledgement, no retry, no
  handshake, no mid-run dialogue.
- Keep the announcement minimal: it names the channel and its conditions and
  says nothing about the work item, offers no hint that a conflict exists, and
  never instructs the agent to look for one.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| State the announcement as a promoted requirement rather than leaving it implicit. | OUT-1, SIG-1 | NG-1 |
| Deliver through the existing hook at the adapter boundary. | OUT-2, SIG-1, SIG-2 | NG-2 |
| Keep the text provider-neutral and make an undeliverable announcement fail the launch. | OUT-3, SIG-3 | NG-2 |
| Announce exactly once with no reply path. | OUT-4, SIG-5 | NG-3 |
| Keep the announcement free of task-specific hints, pinned by a test. | OUT-5, SIG-4 | NG-5 |

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `local-run`: a launched run is told the escalation channel exists, through the
  adapter boundary, once, in fixed provider-neutral wording; an adapter that
  cannot deliver the announcement fails the launch rather than starting a run
  that cannot escalate.

## Impact

- `internal/localrun`: the announcement text and the adapter contract that
  carries it; a launch that cannot deliver it fails before the provider runs.
- `internal/adapters/codex`: the rendered hook gains the announcement as its
  session-start context, alongside the lineage correlation it already performs.
- `goalrail.work-spec/v0` and `goalrail.terminal-receipt/v1` are unchanged, and
  the frozen WorkSpec fixture digest is unchanged.
- The operator's command surface, JSON result grammar, and `--result` grammar
  are unchanged. Composing the work prompt remains the operator's act.
- No new dependency, service, credential, deployment surface, or daemon.

## Non-Goals

- Do not hand the frozen WorkSpec task statement to the provider. This change
  announces one capability, not the task.
- No new WorkSpec or receipt field, and no change to either canonical schema.
- No mid-run dialogue, control plane, resume, acknowledgement protocol, or
  background continuation.
- Do not change `goalrail.escalation/v0`, and do not resolve its known
  incompatibility with the kata oracle's mechanical example check. That is a
  separate decision with its own evidence.
- Do not modify the kata, remove its in-task channel, or run any measurement.
- Planning completion authorizes no implementation, commit, push, pull request,
  merge, archival, provider run, or external effect. Each remains a separate
  owner gate.
