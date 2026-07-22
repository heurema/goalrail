# Intent Canary v0 Manifest

- **Manifest ID:** `intent-canary-v0`
- **Version:** 2
- **Frozen at:** `2026-07-22T15:10:00Z`
- **State:** `frozen_not_activated`
- **Assignment count:** 15
- **Evidence stream:** `canary/intent-canary-v0/events-v2.jsonl`
- **Activation:** requires a separate explicit owner instruction; this manifest admits synthetic rehearsals only

Version 2 freezes the Context-Pack and structured-evaluation flow. Version 1,
its manifest, and its evidence remain immutable historical records. This file
does not activate either version and does not authorize external effects.

## Assignment

| Ordinal | Variant |
|---:|---|
| 1 | `flow` |
| 2 | `flow` |
| 3 | `baseline` |
| 4 | `flow` |
| 5 | `flow` |
| 6 | `baseline` |
| 7 | `flow` |
| 8 | `flow` |
| 9 | `baseline` |
| 10 | `flow` |
| 11 | `flow` |
| 12 | `baseline` |
| 13 | `flow` |
| 14 | `flow` |
| 15 | `baseline` |

Eligibility is decided before variant disclosure. Exclusions consume no
ordinal. Assigned ordinals are never reused or changed.

## Frozen rule references

| Concern | Rule reference | Version 2 meaning |
|---|---|---|
| Eligibility | `canary-rule:eligibility-v1` | Preserve the existing bounded candidate rule. |
| Check recording | `canary-rule:check-recording-v1` | Preserve checks as evidence independent from owner intent assessment. |
| Context | `canary-rule:context-v2` | Flow requires a sufficient Context Pack before confirmed intent; baseline has no pre-execution pack. |
| Assessment basis | `canary-rule:assessment-basis-v2` | Flow basis is pre-execution; baseline basis is explicitly `post_delivery`. |
| Assessment timing | `canary-rule:assessment-timing-v2` | Every basis item is judged once before an aggregate outcome is accepted. |
| Telemetry | `canary-rule:telemetry-v2` | Read-only session reconciliation retains bounded trace intervals and explicit unavailable/conflict outcomes. |
| Missing data | `canary-rule:missing-data-v2` | Missing telemetry is unavailable, never zero, and blocks rehearsal completion readiness. |
| Flow overhead | `canary-rule:overhead-v2` | Machine and owner-review intervals remain separate; the total is derived. |
| Rates | `canary-rule:rates-v1` | Preserve the directional non-match formula and existing pass/stop thresholds. |

## Context and phase boundary

For `flow`, the append-only chain records the Context Pack identity, confirmed
assessment basis, and a completed intent-flow phase before ordinary
implementation. Trace selection uses the explicit phase interval. A trace that
mixes the intent phase with implementation is unavailable evidence rather than
an estimated split. `baseline` does not receive a Context Pack or intent-flow
phase.

## Telemetry and overhead

Only verified root session identity may query telemetry. Reconciliation is
read-only and bounded. Goalrail retains session and trace references plus exact
UTC envelopes; Langfuse retains detailed observations. Duplicate traces are
deduplicated by identity. A session conflict, malformed interval, empty bounded
result, or cross-session child trace produces an explicit non-available result.

Flow overhead is:

```text
agent_seconds = sum(distinct flow-window trace envelopes)
owner_review_seconds = owner_review_end - owner_review_start
flow_overhead_minutes = (agent_seconds + owner_review_seconds) / 60
```

No component is imputed. Baseline may retain trace references but never receives
flow overhead or flow-only owner-review cost.

## Structured owner assessment

The assessment basis lists stable desired-outcome, non-goal, and success-signal
IDs. Flow records it before implementation from confirmed intent. Baseline
records an assessment-only basis after delivery from the original request and
marks that hindsight asymmetry explicitly.

The owner judges every item exactly once. All achieved/preserved/observed means
`match`; a partial outcome or missing signal without a miss or violation means
`partial`; any missed outcome or violated non-goal means `miss`. Checks remain
separate, so green checks with `partial` or `miss` remain wrong-but-green.

## Synthetic proof and activation gate

Before an activation decision, one completed synthetic flow case and one
completed synthetic baseline case must traverse the version 2 chain and
deterministic report. The immutable rotation may require an explicitly
abandoned synthetic flow spacer before ordinal 3; that spacer is not evaluated
as the completed proof case. Any mixed-version evidence, missing required
metric, raw or sensitive payload, invalid assessment, or reconciliation
conflict fails the rehearsal.

Real admission remains rejected. Activation, changing thresholds, or changing
any frozen rule requires another explicit owner decision and a new change or
manifest version as applicable.
