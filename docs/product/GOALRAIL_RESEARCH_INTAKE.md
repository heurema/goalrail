# Goalrail Research Intake

## Purpose

This document defines how ideas from Punk and other adjacent or external projects enter Goalrail without turning the roadmap into an unbounded collection of interesting mechanisms.

Goalrail uses research to extract mechanisms and anti-patterns. It does not import another project's truth, roadmap, or product form.

## Intake rule

Every non-trivial adjacent or external idea must be classified before it can affect Goalrail canon, ADRs, ops direction, implementation slices, or public claims.

Each idea must land in exactly one outcome:

- adopt
- adapt
- defer
- park
- avoid

## Classification meanings

| Classification | Meaning |
|---|---|
| `adopt` | Belongs in Goalrail now with minimal adaptation |
| `adapt` | Valuable, but must be reshaped for Goalrail's production and product context |
| `defer` | Valuable, but not needed in the current phase |
| `park` | Out of scope until the roadmap explicitly promotes it |
| `avoid` | Conflicts with Goalrail laws, MVP, trust model, or product posture |

## Required fields

Future intake records should include:

- source project
- source document
- extracted mechanism
- Goalrail mapping
- recommendation
- reason
- risk
- required docs
- required ADRs
- required evals
- trigger condition if deferred
- explicit out-of-scope notes

These fields exist to force translation into Goalrail terms before anything is promoted.

## Adopt criteria

An idea can be adopted when all of the following are true:

- it fits current Goalrail canon with minimal reshaping
- it strengthens a current product, governance, or trust surface
- it does not widen MVP scope by itself
- it does not create a false implementation claim
- it does not conflict with pilot-first deployment or contract-first delivery

Adopt is for mechanisms that already match Goalrail's direction closely enough to become Goalrail-owned process or doctrine now.

## Adapt criteria

An idea should be adapted when it is valuable but the original form does not fit Goalrail directly.

Typical reasons to adapt:

- the source project is experimental while Goalrail is production-facing
- the source mechanism assumes a different truth model
- the source mechanism assumes local-first or operator-local defaults that are not Goalrail defaults
- the source language would distort Goalrail's product form if copied directly
- the mechanism is useful, but only after mapping to Project Spine, verify/proof, pilot, deployment, and public-claim boundaries

Adapt requires explicit Goalrail mapping before promotion.

## Defer criteria

An idea should be deferred when it is valuable but not needed in the current phase.

Deferred items need:

- a trigger condition
- a clear reason they are not needed now
- a statement that they are not active Goalrail scope yet

## Park criteria

An idea should be parked when it is outside current Goalrail scope until the roadmap explicitly promotes it.

Park is stronger than defer. A parked item should not leak into current product explanation, MVP expectation, or implementation planning.

## Avoid criteria

An idea should be avoided when it conflicts with Goalrail's core laws, MVP boundary, trust posture, or honest public posture.

Avoid items are useful as anti-patterns and review checks, not as backlog candidates.

## Artifact flow

```text
source idea
  -> research note / synthesis entry
  -> intake classification
  -> ADR or product doc if accepted/adapted
  -> ops roadmap/status patch
  -> bounded implementation slice
  -> eval / proof
```

The key rule is sequencing: classification comes before roadmap spread, implementation, or public claims.

## Current recommended intake from adjacent Punk research

The table below captures the current recommendation for adjacent Punk mechanisms reviewed during this governance pass. Punk is treated here as advisory source material only.

| Adjacent mechanism | Recommendation | Reason | Required Goalrail follow-up |
|---|---|---|---|
| Research Gate | `adopt` | Goalrail needs an explicit rule for when research is mandatory before core changes | `docs/product/GOALRAIL_RESEARCH_GATE.md` |
| Research Intake | `adopt` | Goalrail needs a bounded intake process for adjacent ideas | `docs/product/GOALRAIL_RESEARCH_INTAKE.md` |
| active-core / incubating / parked vocabulary | `adapt` | Useful lifecycle language, but it must describe Goalrail doc or surface maturity, not Punk operator status | `docs/product/GOALRAIL_DOC_GOVERNANCE.md` |
| event ledger / replay inspect | `adapt` | Useful evidence pattern, but Goalrail must map it to its own spine, provenance, and deployment model | later Goalrail ADRs for ledger / provenance / inspect views |
| proofpack / provenance | `adapt` | Useful trust mechanism, but Goalrail proof must fit verify / decision / pilot evidence contours | later Goalrail provenance / proof ADRs and evals |
| redaction / no-hidden-telemetry | `adapt` | Strong anti-leak and trust discipline, but Goalrail must map it to productized deployment realities | later Goalrail privacy / telemetry / redaction ADRs and evals |
| machine + human eval reports | `adapt` | Useful reporting pattern, but Goalrail needs product-shaped eval outputs rather than Punk-shaped receipts | later Goalrail eval/report format docs |
| project-memory link graph | `adapt` | Valuable for inspectable continuity, but Goalrail must avoid giant-prompt memory and preserve canonical truth ownership | later Goalrail project-memory doc / ADR |
| marketplace / plugin ecosystem | `park` | Not needed for current Goalrail phase and would widen scope prematurely | keep out of current roadmap and canon |
| AI final decision | `avoid` | Conflicts with Goalrail gate / decision boundary | preserve gate-owned decision semantics |
| adapters owning truth | `avoid` | Conflicts with Goalrail source-of-truth model | preserve canonical-doc and spine-owned truth |
| hidden telemetry | `avoid` | Conflicts with Goalrail trust posture and honest public claims | preserve explicit telemetry/export policy only |
| memory as giant prompt | `avoid` | Conflicts with inspectable, linked, bounded project memory | preserve structured memory and explicit authority |
