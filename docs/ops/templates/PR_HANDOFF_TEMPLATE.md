---
id: goalrail_pr_handoff_template
title: Goalrail PR Handoff Template
kind: reference
authority: operational
status: current
owner: ops
truth_surfaces:
  - pr_handoff_template
  - team_pilot_readiness
lifecycle: active-core
review_after: 2026-08-25
supersedes: []
superseded_by: null
related_docs:
  - docs/ops/TEAM_PILOT_RUNBOOK.md
  - docs/ops/GOALRAIL_AGENT_SKILL.md
---
# Goalrail PR Handoff Template

Use this template for Team Pilot PRs. Replace placeholders and remove sections
that are genuinely not applicable.

## Goalrail IDs

- goal_id:
- contract_id:
- plan_id:
- proposal_id:
- work_item_id:

## Summary

-

## Scope Delivered

-

## Non-Goals Respected

- No runner checkout.
- No checkout prepare.
- No execution prepare.
- No execution.
- No gate, proof, or verification.
- No WorkItem completion.
- No provider OAuth or stored credentials.
- No `.goalrail/templates`.
- No `.goalrail` tracking or `.gitignore` marker-policy change.
- No autonomous executor claims.

## Checks Run

-

## Artifacts Changed

-

## Deferred Work

-

## Human Gates Respected

- Contract approval:
- Proposal acceptance:
- Mutating commands reviewed:
- PR review/merge:

## Secret and Local-State Confirmation

- `.goalrail/project.yml` was not committed.
- Auth files were not committed.
- Tokens, runner bearer tokens, JWT contents, local DB passwords, provider
  credentials, private host details, private paths, and temporary passwords were
  not committed or printed.

## Runner and Execution Boundary Confirmation

- Runner checkout was not run.
- Checkout prepare was not run.
- Execution prepare was not run.
- Execution was not run.
- Gate/proof/verification/completion were not run.

## Review Notes

-
