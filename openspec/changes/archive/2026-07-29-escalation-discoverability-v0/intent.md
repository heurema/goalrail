# Intent Snapshot

- **Intent ID:** intent-escalation-discoverability-v0
- **Version:** 1
- **Status:** confirmed
- **Owner:** repository owner
- **Context Pack:** context-escalation-discoverability-v0 version 1
- **Run references:** none; this change authorizes no provider run

## Source Evidence

- **SE-1 (owner):** The escalation channel now lives in the harness, but nothing tells a run it exists; make the channel discoverable so the capability is reachable, before cleaning the channel out of the kata or repeating the measurement.
- **SE-2 (repository):** No promoted requirement states that a run is ever told the reserved path exists, and the adapter hands the provider only a working directory, the operator's arguments, and the run-context environment value.
- **SE-3 (repository):** Goalrail already renders an invocation-local provider hook at session start for lineage, and that hook's published output contract carries an optional context string.
- **SE-4 (repository):** The canonical WorkSpec is provider-neutral by promoted requirement and cannot carry provider launch details.
- **SE-5 (repository):** The measurement that motivates the channel recorded that its treatment branch was loud, so how the channel is announced is a variable in its own right, not a formatting decision.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | State as a promoted requirement that a launched run is told the escalation channel exists: the reserved path, when it applies, that it requires an otherwise untouched scope, and where the payload shape is published. Without an announcement the capability is unreachable, and admitting a channel nobody can find is not a delivered capability. | Read the requirement and confirm the announcement is required rather than optional. | SE-1, SE-2, CTX-1, CTX-2 |
| OUT-2 | Deliver the announcement through the adapter boundary at launch time, reusing the provider hook Goalrail already renders rather than adding a second provider surface. The canonical WorkSpec, receipt, and run store gain no field for it. | Confirm the announcement travels the existing hook path and that `goalrail.work-spec/v0` and the terminal receipt are unchanged. | SE-3, SE-4, CTX-3, CTX-4, CTX-5 |
| OUT-3 | Keep the announcement provider-neutral in substance and provider-specific only in transport: the text Goalrail composes names no provider, and each adapter is responsible for delivering it in its own contract's terms. An adapter that cannot deliver it MUST make that explicit rather than launching a run that cannot escalate. | Confirm the composed text names no provider and that an adapter without a delivery path fails rather than silently degrading. | SE-3, SE-4, CTX-5 |
| OUT-4 | Announce the channel exactly once, at launch, as a statement rather than a negotiation. No mid-run dialogue, no capability handshake, no acknowledgement, and no retry if the announcement is not observed. | Confirm the announcement is one launch-time act with no reply path. | CTX-8 |
| OUT-5 | Keep the announcement minimal and measurable: it states the channel and its conditions without describing the work item, without hinting that a conflict exists, and without instructing the agent to look for one. The measurement's loud-treatment caveat is treated as a design constraint, not a footnote. | Compare the announcement text against the recorded caveat and confirm it introduces no task-specific hint. | SE-5, CTX-6, CTX-7 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | Do not hand the frozen WorkSpec task statement to the provider. Composing the work prompt remains the operator's act; this change announces one capability, not the task. | CTX-2 |
| NG-2 | No new WorkSpec field, no receipt field, and no change to `goalrail.work-spec/v0` or `goalrail.terminal-receipt/v1`. | SE-4, CTX-5 |
| NG-3 | No mid-run dialogue, control plane, resume, acknowledgement protocol, or background continuation. | CTX-8 |
| NG-4 | Do not change the payload shape `goalrail.escalation/v0`, and do not resolve its known incompatibility with the kata oracle's mechanical example check. That is a separate decision with its own evidence. | CTX-7 |
| NG-5 | Do not modify the kata, remove its in-task channel, or run any measurement. Those depend on this change landing first. | SE-1, CTX-7 |
| NG-6 | This snapshot authorizes no implementation, commit, push, pull request, merge, archival, provider run, or external effect. Each remains a separate owner gate. | CTX-8 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | The channel is reachable by a run that was never told about it in its task. | A launched run receives the announcement without the reserved path appearing in the WorkSpec, the task statement, or the target repository. | SE-1, CTX-1, CTX-2 |
| SIG-2 | The canonical contracts are untouched. | The frozen WorkSpec fixture digest is unchanged and the terminal receipt schema is unchanged. | SE-4, CTX-5 |
| SIG-3 | An adapter that cannot announce cannot silently launch. | A deterministic test shows an adapter without a delivery path failing explicitly rather than starting a run whose escalation channel is unreachable. | SE-3, CTX-3 |
| SIG-4 | The announcement carries no task-specific hint. | The composed text is fixed, names no provider, and contains nothing about conflicts, documents, or the work item; a test pins it. | SE-5, CTX-6 |
| SIG-5 | Announcement is one act with no reply path. | No acknowledgement, retry, or second delivery exists in the lifecycle. | CTX-8 |

## Ambiguities and Unknowns

None blocking. One recorded limit: the provider's context-delivery behaviour is
taken from its published hook schema and has not been observed end to end,
because doing so requires a real provider run under a separate gate.

## Confirmation

- **Confirmed by:** repository owner
- **Confirmed at:** 2026-07-29T14:11:44Z
- **Verification action:** Owner reviewed a plain-language owner-facing summary of candidate version 1 in the current Goalrail task — the unreachable capability, delivery through the hook already rendered for lineage, provider-neutral text with provider-specific transport, one launch-time statement with no reply path, and a deliberately minimal announcement derived from the measurement's loud-treatment caveat — together with the recorded limit that context delivery is taken from the provider's published schema rather than observed, and explicitly confirmed the snapshot by name and version.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
