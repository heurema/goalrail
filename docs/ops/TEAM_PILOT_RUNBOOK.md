---
id: goalrail_team_pilot_runbook
title: Goalrail Team Pilot Runbook
kind: reference
authority: operational
status: current
owner: ops
truth_surfaces:
  - team_pilot_readiness
  - pilot_operating_model
  - pr_handoff_flow
lifecycle: active-core
review_after: 2026-08-25
supersedes: []
superseded_by: null
related_docs:
  - docs/INDEX.md
  - docs/product/GOALRAIL_PILOT_MODEL.md
  - docs/ops/GOALRAIL_AGENT_SKILL.md
  - docs/ops/templates/CONTRACT_TEMPLATE.md
  - docs/ops/templates/PILOT_CANDIDATE_CHECKLIST.md
  - docs/ops/templates/PR_HANDOFF_TEMPLATE.md
---
# Goalrail Team Pilot Runbook

## Purpose

Use this runbook to pilot Goalrail with a small engineering team on 3-5
low-risk PRs.

The Team Pilot stage tests Goalrail as a contract-first PR control plane. It
does not test autonomous execution, runner checkout, gate, proof, verification,
or WorkItem completion.

## Positioning

Goalrail helps the team turn intent into a reviewed delivery contract, then
into planned WorkItems and PR handoffs.

Goalrail is not:
- an autonomous executor;
- an issue-tracker replacement;
- a GitHub PR review replacement;
- a production incident system;
- a credential or provider OAuth system;
- a proof/gate system for this stage.

## Pilot Flow

The Team Pilot flow is:

```text
Goal -> Clarification -> Contract -> Plan -> Proposal -> WorkItem -> PR handoff
```

Humans approve gates. Agents follow surfaced next actions and stop when the
workflow asks for human approval.

## Roles

Human owner:
- chooses pilot candidates;
- approves Contracts and Proposal acceptance;
- decides whether a PR is ready for review;
- owns final merge decisions.

Agent/operator:
- runs current-source Goalrail CLI commands when needed;
- follows `next_action.command_packet`;
- stops at `requires_human_approval`;
- keeps secrets and local marker files out of outputs and commits;
- prepares the PR handoff.

Developer:
- reviews the accepted WorkItem and implementation scope;
- keeps the change bounded;
- runs appropriate checks;
- records deferred work.

Reviewer:
- reviews the PR against the Goalrail Contract and WorkItem;
- verifies non-goals and boundary confirmations;
- approves or requests changes through normal GitHub review.

## Suitable Pilot Tasks

Good Team Pilot candidates are low-risk and easy to review:
- docs or runbook updates;
- small CLI output polish;
- small read-only endpoint visibility;
- test coverage improvement;
- small internal tooling cleanup;
- bounded developer-experience improvement.

Use `docs/ops/templates/PILOT_CANDIDATE_CHECKLIST.md` before starting a Goal
when the team is choosing the next 3-5 pilot PR candidates.

## Unsuitable Pilot Tasks

Do not use Team Pilot for:
- production incidents;
- urgent hotfixes;
- broad architecture changes;
- provider credential or OAuth work;
- autonomous execution work;
- runner checkout, checkout prepare, execution, gate, proof, verification, or
  completion work;
- tasks requiring secret handling or private local state;
- tasks with unclear ownership.

## Human Gates

Human approval is required for:
- Contract approval;
- Proposal acceptance;
- any mutating command with `requires_human_approval`;
- PR review and merge;
- any runner checkout, checkout prepare, execution, gate, proof, verification,
  or completion step if a later approved Contract explicitly includes it.

For this stage, runner and execution gates remain deferred.

## Expected Artifacts

Each pilot PR should include:
- Goalrail IDs;
- a Contract summary;
- WorkItem detail or task identity;
- scope delivered;
- non-goals respected;
- checks run;
- deferred work;
- runner/execution boundary confirmation.

Use `docs/ops/templates/PR_HANDOFF_TEMPLATE.md` for the handoff shape.

## Running a 3-5 PR Pilot

1. Pick 3-5 low-risk candidate tasks with clear ownership.
2. Start each task as a Goal.
3. Answer clarifications with enough detail to produce a bounded Contract.
4. Review and approve the Contract only when scope, non-goals, constraints, and
   acceptance criteria are clear.
5. Create a WorkItemPlan and review the proposal before acceptance.
6. Accept only contract-aware proposals.
7. Inspect the resulting WorkItem.
8. Implement the PR manually from the WorkItem scope.
9. Use the PR handoff template.
10. Merge only through normal human review.

Developers should not manually construct long CLI sequences. Agents should use
the surfaced `next_action.command_packet` where available and report when the
surface is missing or unclear.

## Quickstart Example: Accepted WorkItem to PR Handoff

Use this shape for a low-risk docs-only pilot task such as
`PILOT-EXAMPLE: docs-only runbook wording refinement`.

Example placeholders:
- goal_id: `goal_01EXAMPLE`
- contract_id: `contract_01EXAMPLE`
- plan_id: `plan_01EXAMPLE`
- proposal_id: `proposal_01EXAMPLE`
- work_item_id: `work_item_01EXAMPLE`

After Proposal acceptance:
1. Inspect the WorkItem with the surfaced read-only command packet, or with
   `goalrail work item show --task-id work_item_01EXAMPLE --format json`.
2. Confirm that the title, summary, scope, acceptance refs, proof/check refs,
   and non-goals match the approved Contract.
3. Prepare the PR handoff from
   `docs/ops/templates/PR_HANDOFF_TEMPLATE.md`; replace placeholders with the
   Goalrail IDs and summarize the docs-only change.
4. Run checks appropriate for the files changed, such as docs-check
   changed-files mode, `git diff --check`, `git diff --cached --check`, and
   `scripts/check-staged.sh`.
5. Open the PR with the handoff body and leave normal GitHub review/merge to
   the human reviewer.

Compact handoff body:

```markdown
## Goalrail IDs

- goal_id: goal_01EXAMPLE
- contract_id: contract_01EXAMPLE
- plan_id: plan_01EXAMPLE
- proposal_id: proposal_01EXAMPLE
- work_item_id: work_item_01EXAMPLE

## Summary

- Refined docs-only runbook wording for a Team Pilot task.

## Scope Delivered

- Updated the runbook wording in the approved docs path.
- Preserved the accepted WorkItem scope and non-goals.

## Non-Goals Respected

- No runner checkout.
- No checkout prepare.
- No execution prepare.
- No execution.
- No gate/proof/verification/completion.
- No WorkItem completion.

## Checks Run

- docs-check changed-files mode: pass
- git diff --check: pass
- scripts/check-staged.sh: pass

## Artifacts Changed

- docs/ops/TEAM_PILOT_RUNBOOK.md

## Deferred Work

- Next pilot PR candidates remain separate.
- Runner/execution track remains deferred.

## Human Gates Respected

- Contract approval: yes.
- Proposal acceptance: yes.
- PR review/merge: pending.

## Secret and Local-State Confirmation

- .goalrail/project.yml was not committed.
- Auth files, tokens, local DB passwords, provider credentials, private paths,
  and temporary passwords were not committed or printed.

## Runner and Execution Boundary Confirmation

- Runner checkout was not run.
- Checkout prepare was not run.
- Execution prepare was not run.
- Execution was not run.
- Gate/proof/verification/completion were not run.
```

Keep `.goalrail/project.yml` local, untracked, and uncommitted. Redact tokens,
auth file contents, local DB passwords, provider credentials, private host
details, and private paths. WorkItem completion is also deferred for this
stage.

## Missing GitHub Checks

The normal Team Pilot PR merge path is:

```text
PR ready for review -> GitHub checks appear -> required checks pass -> human merge
```

Missing checks are different from failing checks. A missing-checks case means
GitHub has not created the expected status checks for the PR head. Common signs
are an empty `statusCheckRollup`, `gh pr checks` reporting no checks, and no
workflow runs or check-runs for the branch or head commit.

When checks are missing:
1. Inspect the PR Checks tab for approval prompts, disabled workflow messages,
   or missing required checks.
2. Inspect the Actions tab filtered by the PR branch.
3. Compare workflow runs for the branch and the head commit.
4. Inspect branch protection or rulesets for required check names and source
   apps.
5. Check whether workflow approval, organization policy, or incident recovery
   is suppressing runs.
6. Check for required check name or source mismatches, especially after workflow
   or job renames.

Do not treat missing checks as green checks. Do not merge while required checks
are missing. Do not repeatedly push empty commits. If there is no workflow run
to rerun, use at most one deliberate empty-commit retrigger after a repo admin
or human confirms that settings or a GitHub incident have been resolved. If
checks still do not appear, stop and ask a repo admin to inspect GitHub UI
settings before trying again.

## Success Criteria

The pilot is useful when:
- the team can run 3-5 low-risk PRs through Goalrail without long hand-built
  CLI sequences;
- Contracts make scope, non-goals, checks, and deferred work clearer;
- WorkItems are specific enough to guide implementation;
- PR handoffs are easier to review;
- humans retain approval over gates and merges;
- no secrets, local marker files, or private paths are committed.

## Stop Conditions

Stop and report when:
- the next action requires human approval;
- a command would mutate state but the human has not approved it;
- the CLI command packet is missing for a mutating operation;
- the local server appears stale after server code changes;
- auth expires and the authenticated CLI command fails;
- a task requires runner checkout, execution, gate, proof, verification, or
  completion;
- a task requires provider OAuth, stored credentials, or secret handling.

## Local Operational Notes

`.goalrail/project.yml` is required by the current CLI marker path and must
remain local, untracked, and uncommitted.

Authenticated CLI JSON may include `auth_session` metadata. Treat it as safe
observability only. It should not include token values or auth file contents.

When `HOME` is overridden for CLI auth context, set explicit Go caches for
current-source `go run` commands if needed:

```bash
GOMODCACHE=/tmp/goalrail-gomodcache
GOCACHE=/tmp/goalrail-gocache
```

After server code changes, rebuild and restart local server wrappers from
current source. A stale temporary server binary can pass health checks while
missing newly merged routes.

## Deferred Work

The following are deferred for Team Pilot Readiness:
- runner checkout;
- checkout prepare;
- execution prepare;
- execution;
- gate, proof, and verification;
- WorkItem completion;
- provider OAuth and stored repository credentials;
- `.goalrail/templates`;
- `.goalrail` tracking policy changes;
- broad autonomous delivery claims.
