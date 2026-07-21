## Why

An agent can execute every downstream step correctly and still produce the wrong result when it has misunderstood the owner's intent. Goalrail needs a first-class, owner-confirmed intent artifact and a small real-work comparison that shows whether the added flow reduces those misses without imposing unacceptable process cost.

## What Changes

- Add a versioned Intent Snapshot lifecycle that keeps source evidence distinct from model interpretation and gates proposal generation on active owner confirmation.
- Define automatic lineage from stable `change_id` through `run_id` to the root Codex `session_id`; expose absent linkage as `UNLINKED` instead of asking people to copy identifiers or guessing a join.
- Define a 15-change directional canary using a fixed `flow → flow → baseline` rotation, contemporaneous evidence, owner outcome assessment, overhead measurement, and explicit pass and stop signals.
- Keep the slice provider-replaceable: OpenSpec expresses the development flow and Langfuse may supply trace evidence, but neither becomes Goalrail's canonical runtime model or sole source of truth.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Intent Snapshot lifecycle and proposal gate | OUT-1, OUT-2 | NG-1, NG-2 |
| Automatic execution lineage and explicit `UNLINKED` state | OUT-3, SIG-1, SIG-6 | NG-3, NG-4 |
| Fixed canary assignment, owner assessment, and append-only evidence | OUT-4, SIG-2, SIG-3, SIG-4, SIG-5, SIG-7, SIG-8 | NG-4, NG-5 |
| Replaceable development-time boundaries | OUT-5 | NG-1, NG-2, NG-3 |

## Capabilities

### New Capabilities

- `intent-snapshot`: Evidence-to-candidate-to-confirmed intent lifecycle, stable IDs, material amendments, and proposal gating.
- `intent-canary`: Automatic change/run/session lineage plus the bounded comparison, assessment, measurement, and stop protocol.

### Modified Capabilities

None. Goalrail has no canonical capability specs yet.

## Impact

- Adds the first Goalrail OpenSpec change and future canonical capability specs after archival.
- Constrains the next implementation slice to a small project-local correlation adapter, evidence record, tests, and canary operating instructions.
- Uses the existing shared Langfuse project and `project:goalrail` tagging as an evidence source where available; no Langfuse fork, dashboard, or credential change is required.
- Introduces no runtime API, deployment, external write, or selected durable-execution, authorization, or sandbox provider in this planning change.

## Non-Goals

- Treating this accepted proposal as approval to implement or to start the 15-change canary; each requires a later explicit owner instruction.
- Selecting Temporal versus Restate, OpenFGA versus SpiceDB, or a final sandbox tier.
- Building a workflow engine, authorization system, sandbox, evidence dashboard, retention automation, or parallel ledger.
- Forking the Langfuse Codex plugin or requiring manual identifier copying.
- Treating a small directional canary as general causal proof.
