---
id: goalrail_pilot_candidate_checklist
title: Pilot Candidate Checklist
kind: reference
authority: operational
status: current
owner: ops
truth_surfaces:
  - team_pilot_readiness
  - candidate_selection
lifecycle: active-core
review_after: 2026-08-25
supersedes: []
superseded_by: null
related_docs:
  - docs/ops/TEAM_PILOT_RUNBOOK.md
  - docs/ops/GOALRAIL_AGENT_SKILL.md
  - docs/ops/templates/PR_HANDOFF_TEMPLATE.md
---
# Pilot Candidate Checklist

Use this checklist to choose low-risk PR candidates for the 3-5 PR Team Pilot.
The pilot tests Goalrail as a contract-first PR control plane, not as an
autonomous executor.

## Binary Checklist

Select a candidate only when every required answer is yes.

- [ ] The task can fit in one small, reviewable PR.
- [ ] The task has a clear human owner.
- [ ] The desired outcome can be written as a bounded Contract.
- [ ] The expected files or components are easy to name before implementation.
- [ ] The task can produce clear checks for the PR handoff.
- [ ] The task does not require secrets, credentials, or private local state.
- [ ] The task does not require provider OAuth or stored credential work.
- [ ] The task does not require checkout prepare or runner checkout.
- [ ] The task does not require execution prepare or execution.
- [ ] The task does not require gate, proof, verification, or completion.
- [ ] Human gates are clear for Contract approval, Proposal acceptance, PR
      review, and merge.
- [ ] `.goalrail/project.yml` can remain local, untracked, and uncommitted.

## Risk Flags

Treat any checked risk flag as a reason to defer the task from the Team Pilot or
split it into a smaller candidate.

- [ ] Urgent production incident or hotfix pressure.
- [ ] Broad architecture or product semantics change.
- [ ] Runtime behavior change with unclear blast radius.
- [ ] Secret-heavy or private-local-state work.
- [ ] Provider OAuth, stored credential, or repository credential handling.
- [ ] Runner checkout, checkout prepare, execution, gate, proof, verification,
      or completion work.
- [ ] Unclear ownership or missing reviewer.
- [ ] Acceptance criteria cannot be checked in a normal PR review.

## Suitable Examples

Good Team Pilot candidates are low-risk and bounded:

- docs or runbook updates;
- small CLI output polish;
- small read-only visibility improvements;
- test coverage improvements;
- small internal tooling cleanup;
- bounded developer-experience improvements.

## Unsuitable Examples

Keep these out of the Team Pilot:

- production incidents;
- urgent hotfixes;
- broad architecture changes;
- provider credential or OAuth work;
- autonomous execution work;
- runner checkout or checkout prepare work;
- execution, gate, proof, verification, or completion work;
- secret-heavy tasks;
- private-local-state tasks;
- tasks with unclear ownership.

## Expected Checks

Each accepted candidate should name checks before implementation. Use the
smallest relevant set, such as:

- docs-check fixture self-test, when available;
- docs-check changed-files mode, when docs change;
- `git diff --check`;
- `git diff --cached --check`;
- `scripts/check-staged.sh`;
- relevant Go tests only when code changes.

## Boundary Confirmation

For this stage, candidate selection must preserve these boundaries:

- Goalrail remains a contract-first PR control plane.
- Humans approve Contract, Proposal acceptance, PR review, and merge.
- Checkout prepare and runner checkout remain deferred.
- Execution, gate, proof, verification, and WorkItem completion remain
  deferred.
- `.goalrail/templates` remains deferred.
- `.goalrail/project.yml` remains local, untracked, and uncommitted.
