---
id: goalrail_contract_template
title: Goalrail Contract Template
kind: reference
authority: operational
status: current
owner: ops
truth_surfaces:
  - contract_template
  - team_pilot_readiness
lifecycle: active-core
review_after: 2026-08-25
supersedes: []
superseded_by: null
related_docs:
  - docs/ops/TEAM_PILOT_RUNBOOK.md
  - docs/ops/GOALRAIL_AGENT_SKILL.md
---
# Goalrail Contract Template

Use this template for Team Pilot work. Keep it concrete, bounded, and honest
about deferred runtime behavior.

## Title

`<PROJECT-or-SLICE-ID>: <short delivery outcome>`

## Intent Summary

Explain the desired outcome in one or two paragraphs. State why this change is
useful now and what decision or user workflow it supports.

## Scope

In scope:
- `<specific deliverable>`
- `<specific file/path or product surface>`
- `<specific visibility, documentation, or behavior change>`

Out of scope:
- `<explicit non-goal>`
- `<deferred runtime or platform behavior>`

## Non-Goals

- No runner checkout unless separately approved.
- No checkout prepare unless separately approved.
- No execution prepare or execution unless separately approved.
- No gate, proof, verification, or WorkItem completion unless separately
  approved.
- No provider OAuth, provider clients, stored credentials, or secret handling
  unless explicitly scoped.
- No claims that Goalrail replaces issue trackers, GitHub PR review, or human
  approval.

## Constraints

- Follow product canon and current ops docs.
- Keep changes small and reviewable.
- Follow `next_action.command_packet` where available.
- Stop at `requires_human_approval`.
- Report `mutates_state` before running mutating commands.
- Keep `.goalrail/project.yml` local, untracked, and uncommitted.
- Keep secrets and private paths redacted.

## Acceptance Criteria

- `<observable criterion>`
- `<reviewable artifact>`
- `<boundary or non-goal confirmation>`
- `<check or manual review expectation>`

## Expected Checks

- `git diff --check`
- `git diff --cached --check`
- `scripts/check-staged.sh`
- `<docs or code checks relevant to the slice>`

## Proof Expectations

- PR includes Goalrail IDs when available.
- PR summarizes scope delivered and non-goals respected.
- PR lists changed files or artifacts.
- PR lists checks run.
- PR confirms deferred runner/execution/gate/proof/completion boundaries.
- PR confirms `.goalrail/project.yml`, auth files, tokens, local DB passwords,
  JWT secrets, provider credentials, private host details, and private paths
  were not committed.

## Risks

- Scope creep:
- False implementation claims:
- Secret/local-state leakage:
- Human-gate bypass:

## Deferred Work

- `<known follow-up>`
- `<future approval boundary>`

## Human Gates

- Contract approval:
- Proposal acceptance:
- Mutating command approvals:
- PR review and merge:

## Secret and Local-State Rules

- Use placeholders for secrets.
- Do not paste bearer values, refresh tokens, auth JSON, JWT contents, provider
  credentials, local DB passwords, private host details, or private paths.
- Do not commit `.goalrail/project.yml`.
