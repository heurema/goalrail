# Version the Context and Evaluation Flow

- **Status:** accepted by owner; real canary activation remains separately gated
- **Date:** 2026-07-22
- **Decision owner:** t3chn
- **Confirmed intent:** `intent-context-evaluation-v0` version 1
- **OpenSpec change:** `intent-context-evaluation-v0`

## Context

Intent is Goalrail's primary upstream artifact, but the existing development
schema began at `intent.md`, the canary stored only an aggregate owner label,
and Langfuse telemetry was referenced without a bounded read path. Manifest v1
was already frozen and cannot truthfully measure a materially richer flow.

The decision must preserve four boundaries: evidence is not owner intent;
confirmed intent is not effect authority; detailed provider telemetry is not
canonical Goalrail state; and manifest history is never rewritten.

## Options

1. Add fields to manifest v1 and keep context in exploration chat. This is
   smaller, but destroys measurement comparability and leaves the most
   important upstream evidence informal.
2. Freeze manifest v2 with a Context Pack predecessor, item-level owner
   judgments, and a narrow read-only Langfuse reconciliation adapter. Keep
   Goalrail JSONL authoritative for decisions and retain only provider
   references plus bounded timing envelopes.
3. Build a hosted evaluation platform with datasets, scores, a workflow engine,
   and dashboards. This could support later operations, but adds several systems
   before one local canary demonstrates a need.

## Evidence

- The owner explicitly identified misunderstood intent as the failure that
  invalidates all downstream execution and required visible context gathering.
- Manifest v1 is `frozen_not_activated`, so versioning has no migration cost and
  avoids rewriting real evidence.
- Current Langfuse observations v2 supports an exact structured `sessionId`
  filter, bounded cursor pagination, and the trace and timing fields needed for
  reconciliation.
- The existing domain calculator already separates agent-turn and owner-review
  time, but v1 allowed the terminal caller to supply the total.
- One independent Claude Fable critique accepted the direction while requiring
  manifest versioning, separate human cost, structured owner judgment, and a
  clear Goalrail/Langfuse ownership split.

## Decision

Select option 2.

- OpenSpec schema version 2 compiles `context -> intent -> proposal -> specs ->
  design -> tasks`.
- Manifest v2 uses a separate `events-v2.jsonl` chain and remains
  `frozen_not_activated`.
- Flow records Context Pack identity, a pre-execution assessment basis, and an
  explicit intent-flow phase. Baseline records its assessment basis visibly
  post-delivery.
- A stdlib HTTP adapter performs one bounded read of Langfuse observations v2
  using verified root session identity. It does not write scores or copy raw
  prompts and outputs.
- Goalrail records session and trace references, trace envelopes, owner-review
  interval, derived overhead components, judgments, and aggregate outcome.
- Every basis item is judged exactly once; the aggregate is validated
  deterministically but remains the owner's judgment.

The owner confirmation authorizes this local implementation only. It does not
authorize a real assignment, credential use against Langfuse, commit, push,
merge, deploy, or publication.

## Rejected Objections

- Langfuse should not become the durable source of canary decisions: provider
  telemetry can be re-read, while assignment, owner judgment, and verdict are
  Goalrail semantics.
- A missing trace cannot be treated as zero overhead. It remains explicit
  unavailable evidence and prevents completion readiness without erasing the
  delivery or owner assessment.
- Baseline and flow are not claimed to be causally symmetric. The baseline
  assessment basis is marked `post_delivery`, and the canary remains
  directional.

## Rollback

Before activation, revert schema-v2 enforcement and v2-only code while keeping
manifest v1 and its evidence path untouched. Preserve any synthetic v2 JSONL as
inert history; no Langfuse rollback is required because reconciliation is
read-only.

## Revisit Condition

Revisit after a completed 15-change v2 canary, a provider contract change that
cannot be isolated in the adapter, or evidence that repository-local JSONL no
longer meets concurrency, retention, or team-access needs.
