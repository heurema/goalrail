---
id: goalrail_proof_gap_report
title: Goalrail Proof Gap Report
kind: product_canon
authority: canonical
status: current
owner: product
truth_surfaces:
  - proof_gap_report_shape
  - diagnostic_artifact
  - artifact_led_entry_output
lifecycle: incubating
review_after: 2026-06-15
supersedes: []
superseded_by: null
related_docs:
  - docs/product/GOALRAIL_PROOF_GAP_ENTRY.md
  - docs/product/GOALRAIL_PRODUCT_CONCEPT.md
  - docs/product/GOALRAIL_OPERATING_MODEL.md
  - docs/product/GOALRAIL_PILOT_MODEL.md
  - docs/ops/PUBLIC_CLAIMS.md
  - docs/ops/DECISIONS.md
---
# Goalrail Proof Gap Report

## 1. Purpose

`ProofGapReport` is the first artifact-led diagnostic output for Goalrail.

It is the report returned when a user brings a task, a change, and available
evidence into the Proof Gap Entry flow.

It answers:
- What was the intended task?
- What working contract can be reconstructed?
- What changed?
- Did the change stay inside the intended contract?
- What evidence exists?
- What proof is missing?
- What should happen next before trust or acceptance?

`ProofGapReport` is diagnostic.

It is not:
- server-owned `Proof`
- `GateDecision`
- merge approval
- security audit
- generic code review
- generic AI code review

The report helps a user see the gap between intent, implementation, and
evidence. It does not replace Goalrail's long-term contract-to-proof canon.

## 2. Inputs v0

Required:
- task text
- diff text

Optional:
- PR description
- acceptance criteria
- CI / test logs
- reviewer notes
- linked issue text
- repo metadata summary

Explicitly out of v0:
- live repo clone
- GitHub App
- OAuth
- PR comment bot
- tracker sync
- execution
- test running
- sandboxing
- provider runtime calls
- authoritative merge decision

V0 should work from user-supplied artifacts. It should not imply that Goalrail
has connected to the repository, executed code, run tests, or produced
merge-grade proof.

## 3. Report sections

Canonical report sections:

1. Executive summary
2. Source inputs
3. Reconstructed working contract
4. Scope boundary
5. Scope delta
6. Evidence map
7. Proof gaps
8. Risk notes
9. Soft verdict
10. Next required proofs
11. Residual risks
12. Optional baseline-vs-Goalrail delta

The sections are ordered for human reading first. A later renderer or parser may
use the same sections, but this document does not approve implementation.

## 4. Data shape

Human-readable YAML-like shape:

```yaml
proof_gap_report:
  report_id:
  source:
    task_text_ref:
    diff_ref:
    optional_ci_refs:
    optional_pr_ref:
  reconstructed_contract:
    goal:
    scope_in:
    scope_out:
    non_goals:
    acceptance_criteria:
    expected_proofs:
  scope_delta:
    aligned_changes:
    unexplained_changes:
    possible_scope_drift:
    missing_expected_changes:
  evidence_map:
    present_evidence:
    missing_evidence:
    weak_evidence:
    manual_review_evidence:
  proof_gaps:
    - gap:
      severity:
      evidence_basis:
      recommended_next_proof:
  risk_notes:
    - risk:
      reason:
      mitigation:
  verdict:
    status:
    rationale:
    next_required_proofs:
  residual_risks:
    - risk:
      owner_hint:
  optional_delta:
    baseline_summary:
    goalrail_summary:
    delta_axes:
```

This is an artifact shape, not a committed Go struct, JSON schema, database
schema, API contract, or wire format.

## 5. Verdict vocabulary

V0 verdict terms:

- `aligned_but_proof_incomplete` - the change appears aligned with the
  reconstructed contract, but the supplied evidence is not enough for trust or
  acceptance.
- `scope_drift_detected` - the change includes behavior, files, surfaces, or
  effects that are not explained by the reconstructed contract.
- `evidence_too_weak` - the task and change may be understandable, but the
  available checks, logs, reviewer notes, or artifacts do not support the
  acceptance claim.
- `high_risk_needs_review` - the report sees enough risk that trust or
  acceptance should not proceed without more evidence or human review.
- `insufficient_input` - the supplied task, diff, or evidence is too incomplete
  to reconstruct a useful contract or proof gap.

V0 avoids:
- `merge_ready`
- `verified`
- `accepted`
- `safe_to_deploy`

Those terms imply gate authority or acceptance proof that the diagnostic report
does not have.

## 6. Proof gap severity

Simple v0 severity:

- `low`
- `medium`
- `high`

Meaning:

- `high` means trust or acceptance should not proceed without more evidence or
  human review.
- `medium` means the artifact is incomplete but not necessarily blocking by
  itself.
- `low` means a useful follow-up or clarity improvement.

Severity should stay practical and lightweight. It is not a compliance rating
system.

## 7. Baseline-vs-Goalrail delta section

The baseline-vs-Goalrail delta section is optional.

Use it when the same scenario has:
- baseline path
- Goalrail-guided path
- common rubric

Comparison model:

```text
Baseline:
raw task -> direct/raw execution -> result -> evaluator

Goalrail:
raw task -> working contract -> bounded task/proof expectations -> result -> evaluator
```

Delta axes:
- acceptance
- scope adherence
- proof coverage
- regression safety
- change minimality
- review burden
- evidence quality
- out-of-scope changes
- time-to-confidence

The sibling repo `heurema/goalrail-demo` may hold deterministic demo/eval
scenarios for this class of comparison. Those scenarios are reference evidence,
not product implementation and not production proof.

This section is not a statistical benchmark unless a separate benchmark
protocol exists.

## 8. Relationship to future Gate / Proof

Future `Proof` belongs to the verify / gate contour.

`ProofGapReport` may become an input to future proof or gate flows, but it must
not be used as final gate authority.

V0 may recommend next proofs, but it does not decide acceptance.

Keep the boundary explicit:
- `ProofGapReport` diagnoses gaps.
- `GateDecision` decides against frozen verification inputs.
- `Proof` preserves an inspectable acceptance artifact after verification.

## 9. Relationship to pilot

The report is the first artifact-led entry output.

After a report, the honest next step can be:
- repeat run with better inputs
- private repo evaluation
- one-repo Proof-of-Value pilot
- later contract-to-proof rollout

The report should help the user decide whether Goalrail is useful enough for a
pilot. It should not change pricing or commercial canon.

Current commercial posture remains compatible with:
- free qualification / fit check
- paid managed pilot
- one team
- one repo to start
- one visible task-to-proof workflow

## 10. Non-goals

`ProofGapReport` is not:
- security audit
- generic AI code review
- PR merge approval
- test execution
- runtime execution
- sandbox claim
- server-owned proof
- self-serve SaaS
- GitHub-native integration yet

This specification does not approve Go structs, JSON schema implementation,
renderer work, parser work, web pages, HTTP endpoints, CLI commands, GitHub
Actions, PR bots, provider integrations, runtime changes, or gate/proof
implementation.

## 11. Example outline

Short abstract examples:

### Workflow change

- Task: add manual review before approval.
- Baseline gap: direct approval is still possible, so review-gated acceptance
  is not proven.
- Goalrail contract boundary: approval requires `manual_review`, reviewer
  actor, owner, reason, and audit evidence.
- Next proof required: show blocked direct approval and captured review audit
  evidence.
- Soft verdict: `aligned_but_proof_incomplete` or `high_risk_needs_review`
  depending on supplied evidence.

### Pricing copy

- Task: change CTA copy from `Start trial` to `Request access`.
- Baseline gap: copy changed, but billing or provisioning behavior may have
  drifted.
- Goalrail contract boundary: UI copy only; billing, pricing rules, trial
  provisioning, API behavior, and schema are non-goals.
- Next proof required: show copy change and no behavior / API / schema drift.
- Soft verdict: `scope_drift_detected` when behavior changes appear.

### CSV export

- Task: add CSV export for trial requests.
- Baseline gap: CSV can be produced, but permission, field minimization, and
  filter preservation are not proven.
- Goalrail contract boundary: only admin / reviewer export, allowed fields only,
  existing filters preserved.
- Next proof required: show authorized export, blocked unauthorized export,
  allowed fields, and preserved filters.
- Soft verdict: `evidence_too_weak` if the visible export exists without those
  proofs.
