## Why

Goalrail currently begins with an Intent Snapshot but does not preserve the bounded local and external context that shaped the candidate interpretation, and its canary cannot yet reconcile Langfuse telemetry into reproducible measurements. Because the existing manifest is frozen but has no real assignments, now is the reversible point to define the complete measured flow before activation.

## What Changes

- Add a provider-neutral Context Pack completed before flow-intent confirmation. It records concise evidence references, relevance, collection outcome, and material unknowns without retaining raw source bodies or turning evidence into owner intent.
- Strengthen owner outcome assessment so every stable desired outcome, non-goal, and success signal is judged explicitly before the aggregate `match`, `partial`, or `miss` result is accepted.
- Add a read-only Langfuse reconciliation adapter that queries exact traces through verified root session identity, validates and deduplicates trace envelopes, and produces provider-neutral observation references and timing intervals for both variants.
- Preserve separate machine and owner-review overhead components and calculate the existing flow-overhead total without manual trace-ID copying or missing-data imputation.
- Freeze a new, non-activated canary manifest version for the changed flow while preserving manifest v1 and all prior evidence unchanged.
- Document the Goalrail development lifecycle and sources of truth from owner evidence through Context Pack, Intent Snapshot, OpenSpec planning, implementation, runtime evidence, telemetry, canonical specs, decisions, and archives.
- Prove the new measurement surface with one synthetic flow case and one synthetic baseline case. Real activation remains a separate owner decision.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Bounded Context Pack and collection stop rule | OUT-1, OUT-2, SIG-1, SIG-2, SIG-3 | NG-4 |
| Item-level owner assessment with aggregate validation | OUT-3, SIG-4 | NG-5 |
| Session-based Langfuse trace reconciliation | OUT-4, SIG-5, SIG-6 | NG-3, NG-5 |
| Separate machine and human overhead calculation | OUT-5, SIG-6, SIG-7 | NG-2 |
| Frozen manifest v2 and synthetic pair | OUT-6, SIG-8, SIG-9 | NG-1, NG-2, NG-6 |
| Project lifecycle documentation | OUT-7, SIG-10 | NG-3 |

## Capabilities

### New Capabilities

- `context-pack`: Bounded context evidence, provenance, material-unknown handling, collection completion, and the gate before intent confirmation.

### Modified Capabilities

- `intent-snapshot`: Require a completed Context Pack for flow confirmation while keeping evidence separate from the three semantic intent groups.
- `intent-canary`: Add item-level owner judgments, automatic Langfuse trace reconciliation, component overhead evidence, manifest version 2, and the synthetic flow/baseline proof.

## Impact

- Adds provider-neutral domain types and validation for Context Packs and item-level intent assessment.
- Extends the OpenSpec adapter to parse and validate the context artifact without leaking OpenSpec types into the canonical domain.
- Extends the Langfuse adapter with a read-only trace source boundary and a stdlib HTTP implementation exercised through deterministic test servers; no credentials are added or used during this change.
- Extends append-only evidence, operator commands, report projection, fixtures, and integration tests while retaining existing v1 compatibility.
- Adds a new runtime canary manifest path/version and a project lifecycle document.
- Adds no database, daemon, dashboard, workflow engine, provider SDK, or production deployment.

## Non-Goals

- Activating the real canary, assigning ordinal 1, or performing commit, push, merge, deploy, publication, or production credential use.
- Rewriting manifest v1, archived changes, canonical history, or existing evidence.
- Adding full-page external source retention, autonomous crawling, transcript parsing, a Langfuse plugin fork, or dynamic-tag-dependent joins.
- Writing Langfuse scores, building a Langfuse Dataset, adding an LLM judge, or introducing a second-annotator workflow in this slice.
- Treating the directional canary as causal proof or selecting retained durable execution, authorization, or sandbox providers.
