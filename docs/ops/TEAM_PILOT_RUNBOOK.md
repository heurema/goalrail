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
