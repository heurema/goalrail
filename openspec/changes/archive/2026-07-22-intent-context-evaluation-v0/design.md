## Context

Goalrail already has provider-neutral intent, lineage, canary assignment, append-only evidence, assessment, reporting, and an adapter that can turn verified Langfuse identities into bounded references. The current OpenSpec schema begins at `intent.md`; the current Langfuse adapter never performs a network read; the operator accepts an externally calculated overhead total; and assessment stores only the aggregate owner label.

Manifest v1 is frozen, not activated, and has no real assignments. Adding context collection and a more structured evaluation changes the measured flow, so mutating v1 would invalidate its own change-control rule. The owner selected the complete context/evaluation flow after an independent Claude Fable critique. The critique's material objections are accepted as follows: version the manifest instead of mutating it, retain human and machine cost separately, make Langfuse a derived read model rather than the source of truth, and structure owner judgment without pretending it is objective. Existing v1 rules already cover baseline lineage and missing traces; those rules are preserved rather than reimplemented differently.

The current Langfuse API surface, checked again during implementation on 2026-07-22, still exposes `GET /api/public/traces`, but its primary documentation recommends `GET /api/public/v2/observations` for new row-level reads. The v2 endpoint supports a structured `sessionId` filter, `fields=core`, bounded page size, cursor pagination, and returns observation `traceId`, `startTime`, and `endTime` directly. No live project query or credential use is required to implement or test the adapter.

## Goals / Non-Goals

**Goals:**

- Make bounded context evidence an explicit OpenSpec predecessor of intent for future Goalrail changes.
- Keep canonical context and assessment semantics independent of OpenSpec and Langfuse.
- Freeze v2 without rewriting v1 and isolate their evidence streams.
- Reconcile trace intervals automatically through verified session identity and retain human review as a separate component.
- Produce auditable item-level owner judgments while keeping the owner as the outcome authority.
- Explain the project lifecycle and sources of truth in one durable project document.
- Prove the flow and baseline measurement paths with deterministic synthetic evidence.

**Non-Goals:**

- Real canary activation, external Langfuse calls during this change, score writes, datasets, dashboards, LLM judging, crawling, or raw-source retention.
- Selection or implementation of durable execution, authorization, effect enforcement, credential-broker, or sandbox providers.
- Making baseline and flow causally equivalent; v2 remains a directional canary with its existing unequal allocation.

## Decisions

### 1. Add `context.md` to schema version 2

The `goalrail-intent` schema will add a `context` artifact before `intent`. `context.md` is produced from a new template, and `intent` will require it. The artifact records the Context Pack lifecycle and concise evidence; it is not another semantic intent group.

The active migration change was created under schema version 1, so implementation will first add and validate a backfilled `context.md` for this change, then update the schema and templates. Status must remain coherent after the schema begins enforcing the new dependency.

Alternative rejected: keep context inside exploration chat. It is not durable or reviewable. Adding context as a fourth Intent Snapshot group is also rejected because it conflates facts with requested outcomes.

### 2. Keep the canonical Context Pack small

The domain model will contain:

- stable pack ID and version;
- collection start/completion timestamps and outcome;
- context items with stable ID, `repository` or `external` kind, bounded claim, source reference, observation/access timestamp, and bounded relevance;
- explicit material unknowns.

`sufficient` forbids unresolved material unknowns. `material_unknown` and `budget_exhausted` require at least one explicit unknown and cannot confirm flow intent. Bounded text accepts a concise sentence but rejects newlines, control characters, secret-shaped content, and over-limit values. Raw documents remain at their provider; Goalrail retains only claims and references.

Alternative rejected: a trust-level taxonomy. A single `untrusted` value adds no information; source kind, provenance, applicability through relevance, and explicit unknowns are sufficient for the canary.

### 3. Link context into intent without changing its semantics

`IntentSnapshot` will reference a completed Context Pack and each `IntentItem` may reference stable context-item IDs. Validation checks pack timing, completeness, reference existence, and wording-only provenance stability. OpenSpec parsing will split `SE-*` source-evidence references from `CTX-*` context references.

OpenSpec remains an adapter: canonical validation consumes domain values and has no OpenSpec types.

### 4. Preserve v1 and run v2 in a separate evidence stream

The same canary ID retains manifest versions 1 and 2. `CanaryAssignment.ManifestVersion` already exists. Evidence events will gain an optional top-level manifest version so exclusions, stop events, context/basis events, and telemetry events can be validated before an assignment exists. Historical v1 records without that field remain valid as effective version 1 and are never rewritten.

`evidence.Store` and `operator.Service` will receive a manifest version. Existing constructors retain v1 compatibility; version-aware constructors support v2. CLI selection is explicit with `--manifest-version 1|2`; v1 remains the compatibility default during this local change, while v2 guidance always supplies `--manifest-version 2`. The default v2 evidence path is `canary/intent-canary-v0/events-v2.jsonl`, preventing accidental chain mixing.

Alternative rejected: silently switch the old path/default to v2. Explicit selection is safer before activation and preserves old tests and operator behavior.

### 5. Store an assessment basis before item judgments

An append-only assessment-basis event stores only the confirmed snapshot reference, version, and stable item IDs grouped as outcomes, non-goals, and success signals.

- Flow records the basis from its confirmed Intent Snapshot before implementation and links the required Context Pack.
- Baseline does not expose an Intent Snapshot to the delivery agent. After delivery and before assessment, the owner confirms an assessment-only basis compiled from the original request; its `post_delivery` timing remains visible so the canary does not claim symmetric pre-execution intent handling.

Assessment then records exactly one judgment for every basis item. Aggregate rules are deterministic: all satisfied is `match`; partial outcome or missing signal without a miss/violation is `partial`; any missed outcome or violated non-goal is `miss`. Corrections append and retain the earlier judgments.

Alternative rejected: let an LLM judge infer missing items or the aggregate. That would replace the owner-relative construct with a different, uncalibrated construct.

### 6. Reconcile Langfuse through a narrow read-only port

The provider-neutral operator boundary accepts a `TraceSource` that lists trace observations for one session. The Langfuse adapter implements it with the Go standard library against the current observations v2 contract:

- Basic authentication supplied by an explicit caller, never stored in domain values or repository files;
- `GET /api/public/v2/observations` with a structured exact `sessionId` filter,
  `fields=core,basic`, and explicit `fromStartTime`/`toStartTime` bounds;
- page size at most 100 and a hard ceiling of 10 cursor pages;
- caller-controlled HTTP timeout and no hidden retry;
- response body limit and strict JSON decoding;
- validation of session, trace ID, UTC timestamps, and non-reversed intervals.

The adapter groups observations by trace and emits one envelope per trace. A missing page or empty result is `unavailable`, not zero. A second explicit reconciliation may append a correction that supersedes the first; the system performs no unbounded eventual-consistency polling.

Alternative rejected: retain the legacy trace-list endpoint for a new reader. It remains available, but the current primary documentation recommends the v2 observation rows needed here and avoids per-trace reads. Dynamic tags and transcript parsing are also rejected because verified provider session ID is already the stronger key. A Langfuse SDK dependency is unnecessary for this narrow API.

### 7. Select flow-only turns by an explicit phase window

V2 flow evidence records a flow phase start and completion before ordinary implementation. Reconciliation selects traces whose trace start lies inside that window and retains their full root envelope. If implementation begins in the same trace as the flow completion action, the measurement is unavailable; eligible v2 flow work therefore needs a separable planning/apply boundary. Nested observations are not summed.

The existing overhead calculator remains canonical. Reconciled trace envelopes fill `AgentTurns`; the owner-review start/end actions fill `OwnerReview`. Delivery accepts the derived measurement, not a caller-authored total. Baseline traces may be referenced but never produce flow overhead.

Alternative rejected: sum all traces in the session. That measures implementation rather than intent-flow overhead.

### 8. Goalrail JSONL owns decisions; Langfuse owns detailed telemetry

The append-only Goalrail stream owns manifest version, assignment, basis, judgments, checks, terminal state, trace references, component measurements, and verdict. Langfuse owns detailed provider observations, tokens, cost, and timing. A Goalrail event references exact traces but does not copy prompt/output bodies. Langfuse scores are deferred; if added later they must be rebuildable projections from Goalrail evidence.

### 9. Record the decision and lifecycle in durable docs

An ADR will record why v2 supersedes the unactivated v1 and why JSONL plus referenced Langfuse telemetry is the selected source-of-truth split. A project lifecycle document will describe artifact order, source ownership, validation, sync/archive behavior, and separate owner gates. These documents, OpenSpec artifacts, canonical specs after archive, code, and evidence have distinct roles; none is described as a universal source of truth.

## Correlation and Evidence

The stable join remains:

```text
change_id -> run_id -> root session_id -> Langfuse trace_id -> observation envelope
```

Only lifecycle-hook or launch-receipt identity may establish the root session. V2 events state their manifest version and use a version-specific evidence stream. Context Pack and assessment-basis events retain bounded artifact/source references; they do not copy raw context or intent text into the evidence chain.

Resume traces sharing the root session are included when they fall inside the phase window. Child work nested in those trace observations is covered by the root envelope. Child work emitted under another session remains unavailable unless a future provider-authoritative lineage contract links it; v2 will not guess.

## Measurement and Stop Conditions

- Assignment remains 10 flow and 5 baseline in the frozen rotation.
- Primary comparison remains the aggregate owner non-match rate per variant.
- Item judgments diagnose which intent group failed; they do not change the frozen rate formula.
- `wrong-but-green`, prevented misunderstandings, repeat opt-in, abandonments, lineage integrity, and existing pass/stop thresholds remain unchanged.
- Machine overhead comes only from reconciled flow-window trace envelopes.
- Human overhead comes only from the explicit owner-review interval.
- Any missing required component makes flow overhead unavailable and blocks completion readiness.
- A wrong session join, malformed telemetry, mixed manifest evidence, raw/sensitive context payload, inconsistent assessment, or synthetic-pair failure stops this implementation from requesting activation.

No metric or threshold changes after a real v2 assignment. A material change creates another manifest version.

## Risks / Trade-offs

- [Context collection becomes unbounded] -> require a material unknown, terminal outcome, bounded claims, and stop when evidence no longer changes intent or the current decision.
- [Structured assessment increases owner effort] -> reuse stable intent IDs, keep three small judgment vocabularies, measure repeat opt-in, and do not add a second annotator now.
- [Baseline assessment basis has hindsight bias] -> compile it only from the original request, mark `post_delivery` timing explicitly, keep the aggregate owner label as the directional measure, and do not claim causal equivalence.
- [Langfuse API changes] -> isolate schema parsing in one adapter, pin tests to captured minimal fixtures, and fail unavailable rather than reinterpret fields.
- [Trace window contains implementation work] -> require a separate apply boundary and mark an overlapping/mixed trace unavailable.
- [V1/v2 evidence is mixed] -> explicit manifest selection, event version validation, separate paths, and regression tests proving both chains remain readable.
- [Secrets enter context or HTTP logs] -> inject credentials only into the HTTP request, never serialize them, redact errors, bound response bodies, and reject secret-shaped context values.
- [This slice becomes a general workflow platform] -> keep one command, one JSONL stream per manifest version, stdlib HTTP, and no daemon, database, dashboard, scores, or scheduler.

## Rollback

Rollback removes schema-v2 enforcement, v2-only commands/types, manifest-v2 files, and v2 documentation before activation. Manifest v1, its compatibility constructors, default evidence path, canonical specs, and any existing v1 evidence remain usable. If v2 synthetic evidence already exists, rollback preserves its separate JSONL file as inert evidence rather than deleting or rewriting it. No Langfuse state needs rollback because this change performs reads only.

## Open Questions

- Whether resumed and child-agent observations are always emitted under the root session is provider-dependent. The implementation treats cross-session child work as unavailable and the synthetic rehearsal must document the observed supported shape; no other design decision depends on assuming support.
