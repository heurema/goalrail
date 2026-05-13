---
id: goalrail_guided_demo_north_star
title: GoalRail Guided Demo North Star
kind: product_canon
authority: canonical
status: current
owner: product
truth_surfaces:
  - guided_demo_north_star
  - contract_preview_entry
  - risk_preview_language
lifecycle: incubating
review_after: 2026-06-15
supersedes: []
superseded_by: null
related_docs:
  - docs/product/GOALRAIL_PRODUCT_CONCEPT.md
  - docs/product/GOALRAIL_OPERATING_MODEL.md
  - docs/product/GOALRAIL_PROOF_GAP_ENTRY.md
  - docs/product/GOALRAIL_PROOF_GAP_REPORT.md
  - docs/ops/PUBLIC_CLAIMS.md
  - docs/ops/NEXT.md
  - docs/ops/DECISIONS.md
---
# GoalRail Guided Demo North Star

## Purpose

This document anchors the intended future user-facing guided demo experience.

It exists so scenario, eval, checker, packet, and renderer work does not become
the product story by itself. The deterministic `heurema/goalrail-demo` sandbox
can help stabilize report language and evidence examples, but the product north
star remains a guided Goalrail experience in a user's own repository.

This document is not an implementation approval.

## User-facing story

User installs the Goalrail CLI.

User runs a future guided demo command inside an existing repository.

Goalrail initializes or scans the repository using real Goalrail primitives.

User enters a task.

Goalrail runs clarification.

Goalrail generates a working contract.

Goalrail produces a Contract Preview / Risk Preview report.

User sees whether Goalrail is useful before committing to a one-repo
Proof-of-Value pilot.

## Future command direction

Possible future command names:

- `goalrail demo contract-preview`
- `goalrail preview`

These names are candidates only.

No command is approved or implemented by this document. Any implementation
requires a separate bounded brief / Signum with scope, non-goals, proof target,
affected components, and validation.

## Expected future flow

```text
repo init / project scan
  -> repo context summary
  -> task input
  -> clarification
  -> working contract
  -> ambiguity/risk preview
  -> report output
  -> optional Proof-of-Value pilot
```

The guided demo should be a wrapper over real Goalrail primitives, not
demo-only fake logic that cannot later become product behavior.

## Contract Preview Report shape

Future Contract Preview / Risk Preview output should include:

- original task
- repo context summary
- clarification questions and answers
- generated working contract
- scope in
- scope out
- non-goals
- constraints
- expected checks
- expected proof
- ambiguity forks without contract
- possible scope drift
- what Goalrail constrained
- residual risks
- next step

The report is pre-change. It should show how a task becomes bounded before
execution, not claim that a change has already been verified.

## Safe language

Use:

- possible ambiguity fork
- potential scope drift
- missing constraint
- unbounded interpretation risk
- Goalrail constrained this by...

Avoid:

- AI would definitely fail
- Goalrail guarantees correctness
- contract prevents all drift
- safe to merge
- verified proof

The point is to show where a raw task permits ambiguity or scope drift, not to
claim model-failure prediction or guaranteed correctness.

## Relationship to Proof Gap Report

Contract Preview is pre-change.

Proof Gap Report is post-change.

Contract Preview predicts ambiguity and risk forks before execution.

Proof Gap Report inspects actual change and evidence after execution or manual
work.

Both are artifact-led entry modes. Neither is final server-owned `Proof`, a
`GateDecision`, PR verification, or merge approval.

## Relationship to goalrail-demo

The sibling repository `heurema/goalrail-demo` may provide deterministic
eval/demo scenarios and generated/manual report examples.

It is not the production Goalrail implementation.

Its scenario work supports report language, delta evidence, and artifact shape
for this future guided demo. It must not become the product runtime or the only
definition of user-facing value.

## Non-goals

This document does not approve:

- code
- CLI command implementation
- fake execution
- live AI runtime
- GitHub Action
- PR bot
- server-owned `Proof`
- authoritative verification
- self-serve SaaS claim
