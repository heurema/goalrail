# Intent Snapshot

- **Intent ID:** `intent-context-evaluation-v0`
- **Version:** 1
- **Status:** confirmed
- **Owner:** owner
- **Context Pack:** `context-intent-context-evaluation-v0` version 1
- **Run references:** pending; no implementation run or real canary assignment exists yet

## Source Evidence

- **SE-1 — Owner statement:** Intent is the primary artifact because correct downstream execution is not useful when an agent misunderstood the requested result.
- **SE-2 — Owner statement:** Goalrail must visibly gather relevant external context before intent is confirmed; the context-gathering step must not disappear inside an informal exploration conversation.
- **SE-3 — Owner statement:** Intent-flow experiments must retain their history in Langfuse and Goalrail and be compared with explicit evidence rather than an unstructured impression.
- **SE-4 — Owner verification:** After reviewing the minimal v2 direction and the independent critique, the owner explicitly instructed Goalrail to document the idea in detail so it is not lost and to implement it reliably.
- **SE-5 — Repository fact:** `intent-canary-v0` manifest version 1 is frozen but not activated, has no real assignments, and requires a new manifest version for a material change to the measured flow.
- **SE-6 — Repository fact:** Current execution lineage automatically joins `change_id`, `run_id`, and the root Codex `session_id`, while the Langfuse adapter records only a session lookup and performs no trace retrieval.
- **SE-7 — Repository fact:** The current overhead contract combines distinct flow-only Codex turn envelopes with an explicit owner-review interval, but the operator currently supplies the already-derived total.
- **SE-8 — Project constraint:** Goalrail uses the project-local `goalrail-intent` OpenSpec schema, keeps OpenSpec and Langfuse replaceable, and does not treat confirmed intent as effect authority.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Before a candidate Intent Snapshot can be confirmed, the flow assembles a bounded Context Pack that records the material local and external evidence used to understand the request without turning that evidence into owner intent. | Inspect and validate a flow artifact containing concise context items, explicit relevance, provenance, and a terminal collection reason before confirmation. | SE-1, SE-2, SE-4, CTX-1 |
| OUT-2 | Context collection remains proportional: required repository anchors are read, external research occurs only for a material unknown, and collection stops when further evidence can no longer change intent or the current decision. | Exercise sufficient, unresolved, and bounded-stop cases and verify that no unbounded crawl or raw-source archive is required. | SE-2, SE-4, SE-8, CTX-4 |
| OUT-3 | Owner assessment is auditable against the confirmed Intent Snapshot: every desired outcome, non-goal, and observable success signal receives an explicit judgment before an aggregate `match`, `partial`, or `miss` is accepted. | Attempt incomplete, inconsistent, and complete assessments and verify deterministic aggregate validation without inferring judgments for the owner. | SE-1, SE-3, SE-4, CTX-5 |
| OUT-4 | Goalrail automatically reconciles verified Codex session identity with Langfuse trace envelopes for both flow and baseline, retains exact trace references, and leaves missing or conflicting telemetry explicit rather than guessed. | Reconcile synthetic flow and baseline sessions against deterministic Langfuse responses, including duplicates, delayed/missing traces, session mismatch, and resumed-turn envelopes. | SE-3, SE-4, SE-6, CTX-3, CTX-4 |
| OUT-5 | Flow overhead preserves the frozen semantic distinction between agent time and owner-review time: trace-derived machine intervals and explicit human review intervals remain separate inputs to one deterministic total. | Calculate available and unavailable measurements from exact intervals and verify that neither component is imputed or silently replaced by the other. | SE-3, SE-4, SE-7, CTX-5 |
| OUT-6 | The changed flow is frozen as a new manifest version before any real assignment, while manifest v1 and all prior evidence remain immutable historical records. | Inspect both manifest versions and verify that commands reject real activation and cannot append v2 evidence without a separate owner action. | SE-4, SE-5, SE-8, CTX-2 |
| OUT-7 | A project-level lifecycle document explains where intent, context, change design, implementation tasks, runtime evidence, Langfuse telemetry, canonical specs, decisions, and archived changes live and how the project advances between them. | Follow the document from a fresh request through validated local implementation and identify the source of truth and owner gate at each transition. | SE-2, SE-3, SE-4, SE-8, CTX-1 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | This change does not activate the real canary, create ordinal 1, commit, push, merge, deploy, publish, or use production credentials. | SE-4, SE-5, SE-8, CTX-2 |
| NG-2 | It does not rewrite or relabel manifest v1, archived OpenSpec artifacts, or existing append-only evidence. | SE-5, SE-8, CTX-2 |
| NG-3 | It does not add a dashboard, database, workflow engine, Langfuse plugin fork, dynamic-tag correctness dependency, or raw transcript parser. | SE-4, SE-6, SE-8, CTX-3, CTX-4 |
| NG-4 | It does not build an autonomous web crawler or retain full external pages, prompts, transcripts, credentials, or sensitive source content in canonical Goalrail artifacts. | SE-2, SE-4, SE-8, CTX-4 |
| NG-5 | It does not add Langfuse score writes, a Langfuse Dataset, an LLM judge, or a second-annotator workflow before live canary evidence justifies them. | SE-3, SE-4, SE-8, CTX-3 |
| NG-6 | It does not treat a 15-change directional canary as causal proof or select Temporal, Restate, OpenFGA, SpiceDB, or a sandbox provider. | SE-4, SE-8, CTX-5 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | Every confirmed flow intent references a valid Context Pack completed before confirmation. | Validator rejects absent, post-confirmation, duplicate, malformed, raw-content, and unbounded context evidence. | SE-1, SE-2, CTX-1 |
| SIG-2 | External context remains traceable and small. | Each external item has a stable ID, concise claim, source reference, access time, and reason it can affect this change; no raw source body is stored. | SE-2, SE-8, CTX-4 |
| SIG-3 | Context collection has a deterministic stopping surface. | Every pack records `sufficient`, `material_unknown`, or `budget_exhausted`; unresolved material unknowns block intent confirmation. | SE-2, SE-4, CTX-4 |
| SIG-4 | Owner outcome labels are supported by item-level judgments. | All stable intent item IDs are judged exactly once; inconsistent aggregate labels fail validation; revisions append rather than rewrite. | SE-1, SE-3, CTX-5 |
| SIG-5 | Langfuse reconciliation is automatic and provider-bounded. | Given verified lineage, the adapter queries by session identity, validates and deduplicates trace envelopes, and emits provider-neutral references and intervals for both variants without manual trace-ID copying. | SE-3, SE-6, CTX-3, CTX-4 |
| SIG-6 | Missing telemetry never becomes zero or success. | Missing or delayed traces leave machine overhead unavailable, preserve the session lookup, and block completion readiness under the manifest rule without blocking delivery or owner assessment. | SE-3, SE-7, CTX-3 |
| SIG-7 | Human and machine cost remain independently inspectable. | Synthetic evidence contains distinct agent-turn and owner-review seconds plus the exact total; baseline records machine telemetry but no flow-overhead total. | SE-3, SE-7, CTX-5 |
| SIG-8 | The revised canary cannot start accidentally. | Manifest v2 is `frozen_not_activated`; real admission remains rejected until a separate explicit activation change and owner instruction. | SE-4, SE-5, CTX-2 |
| SIG-9 | One synthetic flow case and one synthetic baseline case prove the measurement path end to end. | Both cases produce verified lineage, trace references, frozen checks, structured owner assessment where applicable, and a report derived from the canonical JSONL chain. | SE-3, SE-4, CTX-2, CTX-3 |
| SIG-10 | Repository checks remain green. | `go vet ./...`, `go test -race ./...`, focused adapter tests, `git diff --check`, and strict OpenSpec validation pass. | SE-4, SE-8, CTX-1 |

## Ambiguities and Unknowns

None.

## Confirmation

- **Confirmed by:** owner
- **Confirmed at:** 2026-07-22T14:55:24Z
- **Verification action:** The owner reviewed the proposed minimal v2 scope and independent Claude Fable critique, then explicitly instructed Goalrail to document that idea in detail and implement it reliably. The unresolved provider-shape fact is isolated as AMB-1 and cannot weaken fail-closed telemetry behavior.
- **Amendment rule:** A material change to outcomes, non-goals, or success signals creates a new version; wording-only edits preserve this version.
