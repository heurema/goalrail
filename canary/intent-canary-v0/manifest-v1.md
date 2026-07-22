# Intent Canary v0 Manifest

- **Manifest ID:** `intent-canary-v0`
- **Version:** 1
- **Frozen at:** `2026-07-21T13:17:33Z`
- **State:** `frozen_not_activated`
- **Assignment count:** 15
- **Activation:** requires a separate explicit owner instruction; this manifest does not start the real-change canary

This document is the frozen measurement contract for the first directional
canary. A material rule change creates a new manifest version. It never rewrites
events recorded under this version.

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

Eligibility is decided before the variant is revealed. The next accepted
change receives the next ordinal. Delivery, abandonment, difficulty, and result
never change an assigned ordinal or variant. An abandoned assignment is not
replaced.

## Eligibility

**Rule reference:** `canary-rule:eligibility-v1`

A change is eligible only when all conditions hold before assignment:

1. It is a real owner- or team-requested repository modification, not work
   invented to exercise Goalrail or its instrumentation.
2. Implementation has not started and its outcome is not already known.
3. It is one intended modification in one repository, expected to use one
   immutable `change_id`, one v0 `run_id`, and one root Codex session.
4. Goalrail can obtain the root identity automatically from a lifecycle hook or
   its own App launch receipt. A manually created App task is ineligible.
5. The work can stop at a reviewable handoff. The canary itself requires no
   merge, deploy, publish, send, credential mutation, production change, or
   other external effect.
6. The work is internal and non-sensitive enough for the existing shared
   Langfuse retention boundary.
7. At least one meaningful automated or explicit manual verification check can
   be named and tied to a stable source reference before terminal assessment.
8. The owner agrees to record a terminal outcome; for a delivered `flow`
   change, the owner also records `repeat_optin`.

Rejected candidates record an exclusion reason but consume no ordinal. The
eligibility decision cannot use the next variant, expected difficulty, or an
anticipated result.

## Check recording

**Rule reference:** `canary-rule:check-recording-v1`

- `flow` and `baseline` use the same rule.
- The non-empty check set is recorded after the scope is understood but before
  the first terminal verification run and before owner assessment.
- Every check has a stable check reference and a result tied to the exact source
  reference reviewed or tested.
- `green=true` only when every selected check passed and none is missing.
  Failed, missing, or empty check evidence is not green.
- A selected check is never removed after its result is known. A legitimate
  correction appends a reasoned correction event; it does not rewrite the
  original selection or result.
- Owner intent outcome remains independent. Green checks with `partial` or
  `miss` are reported as `wrong-but-green`.

## Owner assessment timing

**Rule reference:** `canary-rule:assessment-timing-v1`

- `delivered` means the reviewable result and its check/source references were
  handed to the owner or reviewer. Merge and deploy are irrelevant.
- The owner records `match`, `partial`, or `miss` after delivery and before work
  is changed in response to that assessment.
- The owner separately records whether a material misunderstanding was
  corrected before delivery. Wording-only corrections do not count.
- A delivered `flow` change records `repeat_optin=yes|no` in the same assessment
  action. Missing opt-in is not silently converted to yes.
- `abandoned` records a categorical reason and no intent-match assessment. A
  process-caused flow abandonment is marked separately.
- Agents may prepare the assessment surface but never infer or record the
  owner's values on the owner's behalf.

## Missing data

**Rule reference:** `canary-rule:missing-data-v1`

- Nothing is imputed as zero, pass, match, or yes.
- A delivered change without owner assessment is excluded from its variant's
  non-match denominator and blocks completion readiness.
- Abandoned changes are never placed in the non-match denominator; they remain
  in assigned and abandonment counts.
- A missing flow-overhead measurement is excluded from the provisional median
  and blocks completion readiness. Available measurements can still trigger the
  `>30 minutes` hard stop before completion.
- A missing `repeat_optin` counts as not-yes for the two-thirds threshold. It
  remains visible as missing.
- Missing Langfuse observations do not block execution, delivery, lineage
  recording, or owner assessment. The verified session lookup remains, while
  trace-dependent overhead stays unavailable.
- An unresolved lineage after the one bounded resolution attempt is recorded as
  unresolved. Two unresolved links or one wrong join trigger `STOP`.
- A zero rate denominator produces an unavailable rate, never `0%`.

## Flow overhead

**Rule reference:** `canary-rule:overhead-v1`

The metric measures only work added by the intent-first flow. Ordinary
implementation, verification, and delivery time are excluded because both
variants incur them.

### Timestamp sources

1. For agent work, record the distinct Langfuse trace references for Codex
   turns used only to elicit, draft, revise, confirm, compile, or validate the
   Intent Snapshot and proposal before ordinary implementation.
2. For each referenced root `Codex Turn`, use the trace envelope
   `min(startTime)..max(endTime)` from its observations. Nested generations,
   tools, spans, and subagents are not summed separately.
3. Deduplicate by trace ID. A malformed, duplicated, or reversed interval is
   invalid evidence rather than a zero duration.
4. Owner review is one explicit UTC interval from the owner's `review-start`
   action to the confirmation or terminal review action. Its elapsed duration
   is deliberately conservative: pauses inside the interval count.
5. A flow change abandoned before owner review has no owner-review interval and
   uses zero owner-review seconds. Once review starts, a missing end timestamp
   makes the measurement unavailable.

### Formula

```text
flow_agent_seconds = sum(endTime - startTime for each distinct flow-only root turn trace)
owner_review_seconds = owner_review_end - owner_review_start
flow_overhead_minutes = (flow_agent_seconds + owner_review_seconds) / 60
```

Calculations retain timestamp precision and do not round before the report.
Every flow assignment, including an abandoned one, requires a measurement for
completion. Baseline changes never receive `flow_overhead_minutes`.

This formula uses the existing Codex integration: one Langfuse trace per Codex
turn with backdated step timing, grouped by the verified Codex session. It adds
no plugin fork, dynamic credential path, transcript parsing, or manual ID copy.

## Rates and verdicts

**Rule reference:** `canary-rule:rates-v1`

For each variant:

```text
non_match_rate = (partial + miss) / delivered assessed changes
```

- Compare rates, not the unequal raw counts from 10 flow and 5 baseline.
- Median flow overhead is the middle sorted value; for an even count, it is the
  arithmetic mean of the two middle values.
- Repeat opt-in passes when `flow_yes * 3 >= flow_assigned * 2`; abandonment and
  missing opt-in therefore do not become yes.
- `PASS`, early `STOP`, and completion `RESHAPE` use the exact thresholds in the
  confirmed intent and capability spec. Hard-stop evaluation precedes pass
  evaluation.

## Change control

The manifest is frozen but not activated. Any material interpretation change
to eligibility, assignment, checks, assessment timing, missing-data treatment,
overhead, rates, or thresholds creates version 2. Once real assignment starts,
the v1 record is append-only; replacing the rules requires stopping v1 and
starting a separately approved version.
