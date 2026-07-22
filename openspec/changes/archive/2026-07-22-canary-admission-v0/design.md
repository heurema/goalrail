## Context

The v0 harness currently treats `start` as immediate synthetic assignment and lets `deliver` introduce the check references at the same moment as their aggregate result. That leaves no durable proof that eligibility preceded variant disclosure or that checks were selected before terminal verification. The frozen manifest already defines both ordering rules, the evidence store already owns serialized append validation, and real assignment is still blocked.

This change crosses canonical evidence and durable-state boundaries, so the decisions below are explicit. The owner confirmed the two material semantic choices: candidates retain the existing `change_id`, and “pre-run” means before the first terminal verification rather than before implementation.

## Goals / Non-Goals

**Goals:**

- Make eligible and excluded decisions durable before variant reveal.
- Assign only eligible candidates without introducing a second identity.
- Freeze checks before terminal verification and validate later evidence against the effective set.
- Preserve append-only correction, process-shared serialization, inspectability, and report denominators.
- Exercise the complete behavior synthetically while rejecting real evidence at both service and store boundaries.

**Non-Goals:**

- Activate the real canary or change its manifest, rotation, eligibility, metrics, thresholds, assessment, or abandonment rules.
- Add infrastructure, automatic eligibility judgment, raw-content capture, or provider-specific canonical types.
- Select Temporal, authorization, or sandbox implementations.

## Decisions

### 1. Reuse `change_id` for every candidate

An excluded candidate is still one intended repository modification, so its admission event uses the existing immutable `change_id`. It has no `run_id`, ordinal, or variant. An eligible decision adds assignment and run identity in the same evidence event.

Alternatives considered:

- Add `candidate_id` and promote it to `change_id` — rejected because it creates a mapping lifecycle and second source of identity without a current use case.
- Give exclusions no stable identity — rejected because replay and conflict detection would be unreliable.

Revisit only if admission must happen before a change can be named, or one candidate must deliberately fan out into multiple changes.

### 2. Admission and eligible assignment share one event transaction

Add `EventAdmissionDecided` with an `Admission` payload. An eligible event also carries the existing `Assignment` payload; an excluded event must not. The store validates both payloads together while holding the existing process-shared append lock. This makes the decision and ordinal allocation indivisible and keeps exclusions variant-free.

The existing `EventChangeStarted` reader remains valid for already-recorded schema-v1 evidence, but new operator writes use admission events. The current `start` action becomes the eligible path and gains a required categorical reason; a narrow `exclude` action records the excluded path. Both call one admission primitive rather than introducing a second workflow system.

Alternatives considered:

- Append admission and assignment as two events — rejected because a crash could leave an eligible candidate without its assignment or reveal a variant outside the admission transaction.
- Nest assignment inside admission and remove the established assignment field — rejected because it would force a needless evidence-schema migration and duplicate projection logic.
- Replace all operator commands with a new generic workflow API — rejected as premature abstraction.

### 3. Freeze checks as a first-class append-only event

Add `EventCheckSetFrozen` with a non-empty, duplicate-free `CheckSet` payload. `freeze-checks` is valid only for an assigned, non-terminal change. Delivery retains its bounded check references and succeeds only when their set equals the effective frozen set.

`correct-checks` uses the existing correction event kind with a `CheckSet` payload, a reason, and a link to the latest check-set event. It is valid only before terminal evidence and only when the new set retains every previously selected reference. The original bytes remain in the chain; projection selects the newest valid correction.

Alternatives considered:

- Store checks only at delivery — rejected because it cannot prove pre-verification selection.
- Let delivery silently consume the frozen set without naming checks — rejected because an explicit equality check catches stale or wrong operator inputs at the evidence boundary.
- Allow arbitrary pre-result replacement — rejected because the confirmed intent forbids removing an already selected check.

### 4. Enforce lifecycle invariants in the evidence store

The operator service performs friendly preflight checks, but the append validator is authoritative. It rejects duplicate/conflicting admission, real admission or legacy assignment while inactive, assignment without eligible admission for new writes, check freeze before assignment, repeated freeze, invalid correction, and delivered evidence without an exactly matching effective check set. Rejected operations do not append bytes.

Legacy `change_started` evidence remains readable, and its synthetic flag is validated. No automatic retry loop is added: a concurrent stale-ordinal attempt fails cleanly and may be retried explicitly by the operator with a new event ID after inspection.

### 5. Extend existing projections, not infrastructure

`Inspect(change_id)` exposes the admission decision, its event ID, the effective check set and event ID, plus the existing assignment and lifecycle data. `Report()` adds excluded count and categorical reason counts while continuing to build measurement observations only from assignments. Exclusions therefore cannot affect ordinal, delivery, or outcome-rate denominators.

No database, queue, service, dashboard, plugin, or provider adapter is introduced. Temporal, authorization, effect enforcement, sandboxing, evidence, repair, and team collaboration remain retained target-system layers, not requirements of this local slice.

## Correlation and Evidence

- Excluded: `change_id → admission(excluded)`; no run or provider correlation exists.
- Eligible: one event records `change_id → admission(eligible) + ordinal/variant/run_id`; the existing run-context and provider-authoritative session correlation continue unchanged.
- Check selection: `change_id → check_set_event_id → check_refs`; a correction links to and supersedes only the latest effective check-set event.
- Terminal delivery: its check-reference set must equal the effective set and remains independent of later owner intent assessment.
- Actor, UTC timestamp, bounded source reference, categorical reason, event ID, sequence, previous digest, and digest remain in the existing append-only envelope/record boundary.

No prompt, source body, credential, transcript, or unbounded tool output is accepted as admission or check evidence.

## Measurement and Stop Conditions

- `assigned` remains the count of events with assignments; only eligible decisions have them.
- `excluded` and exclusion-reason counts are reported separately and never enter assigned or delivered denominators.
- Existing pass, hard-stop, overhead, lineage, abandonment, and owner-assessment formulas are unchanged.
- Any non-synthetic admission/assignment attempt, duplicate decision, stale ordinal, missing freeze, check-set mismatch, invalid correction, or integrity failure stops that operation before append.
- The manifest stays `frozen_not_activated`; implementation and tests must not create the repository default evidence file.

## Risks / Trade-offs

- [Older binaries do not understand new event kinds] → keep legacy reads in the new binary, use only synthetic evidence before activation, and do not claim backward write compatibility.
- [Two cooperating eligible starts can compute the same next ordinal] → the serialized validator accepts at most one; the loser fails without mutation and explicit retry happens only after inspection.
- [Aggregate `green` does not preserve per-check output] → retain stable check references and the existing aggregate result for v0; richer per-check result payloads require separate confirmed intent.
- [A correction can grow the check set but cannot fix a bad reference by deleting it] → preserve the confirmed no-removal boundary; stop and create a new canary version if the frozen rule itself proves unusable.
- [Admission reason categories can drift] → keep them bounded tokens and report exact counts; a future controlled vocabulary requires measured evidence and a separate change.

## Rollback

Before real activation, revert the admission/check-write surfaces and leave the manifest frozen. Repository-default evidence must still be absent, so no production migration is required. If a separately located synthetic evidence file must be retained, inspect/export it with the new reader before reverting; never rewrite or truncate the chain. The existing canary stop marker remains the operational kill switch for assignments already recorded in any explicit store.

## Open Questions

None.
