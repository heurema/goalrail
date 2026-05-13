---
id: goalrail_proof_gap_entry
title: Goalrail Proof Gap Entry
kind: product_canon
authority: canonical
status: current
owner: product
truth_surfaces:
  - artifact_led_entry
  - proof_gap_entry
  - baseline_vs_goalrail_evidence_principle
lifecycle: incubating
review_after: 2026-06-15
supersedes: []
superseded_by: null
related_docs:
  - docs/product/GOALRAIL_PRODUCT_CONCEPT.md
  - docs/product/GOALRAIL_OPERATING_MODEL.md
  - docs/product/GOALRAIL_PILOT_MODEL.md
  - docs/product/GOALRAIL_OFFER.md
  - docs/research/GOALRAIL_DEPLOYCO_DEPLOYMENT_ENGINE_RESEARCH.md
  - docs/ops/DECISIONS.md
---
# Goalrail Proof Gap Entry

## Purpose

Goalrail should first demonstrate value through a user-visible artifact, not
through a pitch, demo call, broad onboarding, or platform claim.

The user brings a task, change, and available evidence. Goalrail returns a
`Proof Gap Report` that shows the gap between intent, change, and evidence. The
user then decides whether Goalrail is useful enough to continue into an optional
one-repo pilot.

This is an artifact-led entry mechanism. It is not the whole product and does
not replace Goalrail's contract-to-proof canon.

## Entry flow

Canonical entry flow:

```text
user task/change/evidence
  -> reconstructed working contract
  -> scope boundary
  -> scope delta
  -> evidence map
  -> proof gaps
  -> soft verdict
  -> optional one-repo pilot
```

## V0 inputs

V0 stays deliberately simple.

Accepted inputs:
- task text
- diff text
- optional CI/test logs
- optional PR description
- optional acceptance criteria

Explicitly out of V0:
- GitHub App
- OAuth
- provider integration
- repo clone
- tracker sync
- PR comment bot
- server-owned `GateDecision`
- server-owned `Proof`
- self-serve SaaS
- authoritative merge verdict

## V0 output

`Proof Gap Report` is the entry artifact.

It contains:
- reconstructed working contract
- scope boundary
- scope delta
- evidence map
- proof gaps
- risk notes
- next required proofs
- soft verdict

The report should be inspectable by the user and grounded in the supplied task,
change, and evidence. It must not claim that Goalrail has verified the change
for merge.

## Verdict vocabulary

V0 must avoid `merge-ready`, `safe to merge`, `approved`, and similar
merge-grade language.

Allowed soft verdict terms:
- `aligned_but_proof_incomplete`
- `scope_drift_detected`
- `evidence_too_weak`
- `high_risk_needs_review`
- `insufficient_input`

These verdicts describe diagnostic confidence only. They do not create a
server-owned gate decision or acceptance proof.

## Baseline-vs-Goalrail evidence principle

A contract alone does not prove product value.

Goalrail should eventually demonstrate the delta between raw AI-assisted
execution and Goalrail-guided execution on the same task. When possible,
evaluation should compare a baseline path against a Goalrail path.

The sibling repository `heurema/goalrail-demo` is the deterministic demo/eval
sandbox for this class of paired scenarios. It can carry reference scenarios
and fake-data evidence for baseline-vs-Goalrail comparisons, but it is not the
Goalrail product implementation and must not be treated as production proof.

Required comparison model:

```text
Baseline:
raw task -> AI/runtime/human execution -> patch/result -> evaluator

Goalrail:
raw task -> working contract -> bounded packet -> AI/runtime/human execution -> patch/result -> evaluator
```

The comparison should show whether Goalrail improves the quality of scope,
evidence, reviewability, and confidence, not whether it can produce more
activity.

## Measurement axes

Evaluate the entry artifact and later pilot work across these axes:
- acceptance pass/fail
- scope adherence
- proof coverage
- regression safety
- change minimality
- review burden
- evidence quality
- out-of-scope changes
- time-to-confidence

## Relationship to future Proof

`ProofGapReport` is a diagnostic artifact.

Future `Proof` is a gate / verification artifact.

Do not conflate them. A Proof Gap Report may point toward the evidence needed
for future `Proof`, but it is not proof of acceptance by itself and must not be
presented as a merge-grade artifact.

## Relationship to Goalrail canon

This document does not replace the long-term product canon.

Goalrail remains a contract-to-proof operating layer for AI-assisted software
delivery:

`incoming task -> clarify -> working contract -> tasks/run -> verify -> proof -> feedback`

The fixed core remains:
- contract-first flow
- bounded execution
- separated verification
- inspectable proof

Proof Gap Entry is the first artifact-led market entry / wedge. It helps a user
see value on their own work before committing to broader onboarding or a managed
pilot.

## CTA language

First CTA:
- `Run a Proof Gap check`

Post-report CTA:
- `Turn this into a 2-week Proof-of-Value pilot`
- `Install on one repo for 14 days`

The exact public CTA can be tested later after the report shape, claims, and
examples are stable.

## Non-goals

Proof Gap Entry is not:
- generic AI code review
- security audit
- PR verification before merge
- GitHub-native integration yet
- live coding agent
- autonomous merge/deploy
- self-serve SaaS
- server-owned Gate / Proof

This document does not approve source-code implementation, runtime work,
renderer work, parser work, web-page work, GitHub Action work, PR bot work,
provider integration, runner changes, or gate/proof implementation.
