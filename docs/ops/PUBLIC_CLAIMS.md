---
id: goalrail_public_claims
title: Goalrail Public Claims
kind: ops_plan
authority: operational
status: current
owner: docs-governance
truth_surfaces:
  - public_claims
  - implementation_claims
  - proof_gap_entry_claims
lifecycle: incubating
review_after: 2026-06-15
supersedes: []
superseded_by: null
related_docs:
  - docs/INDEX.md
  - docs/product/GOALRAIL_PRODUCT_CONCEPT.md
  - docs/product/GOALRAIL_PROOF_GAP_ENTRY.md
  - docs/ops/STATUS.md
  - docs/ops/COMPONENTS.yaml
  - docs/ops/DECISIONS.md
---
# Goalrail Public Claims

## Purpose

This is a small wording guardrail for public and sales-facing claims while
Proof Gap Entry is being stabilized.

Canonical product truth stays in `docs/product/*`. Implementation truth stays in
`docs/ops/STATUS.md` and `docs/ops/COMPONENTS.yaml`.

## Safe now

- Goalrail is building a contract-to-proof operating layer for AI-assisted
  software delivery.
- Goalrail is testing `Proof Gap Report` as the first artifact-led entry.
- Goalrail is human-in-the-loop by design.

## Safe only with qualification

- `from business goal to verified code change` as thesis / roadmap, not as a
  claim that the full runtime already exists.
- `bounded execution` as scaffolding / typed boundary, not as broad live code
  execution.
- `proof` as a future gate / verification artifact, unless the claim is about
  local, demo, reference, or diagnostic artifacts.

## Unsafe now

- Goalrail verifies PRs before merge.
- Goalrail runs tests in a safe sandbox and returns merge-grade proof.
- Goalrail has GitHub-native issue / PR proof live.
- Goalrail is already self-serve SaaS.
- Goalrail produces server-owned `Proof` artifacts today.

## Review rule

Before publishing a new claim about implemented behavior, check:
- `docs/ops/STATUS.md`
- `docs/ops/COMPONENTS.yaml`
- the relevant code path, if the claim says behavior exists

If those sources do not agree, qualify the claim or do not publish it.
