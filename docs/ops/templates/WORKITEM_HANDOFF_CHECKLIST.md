---
id: goalrail_workitem_handoff_checklist
title: WorkItem Handoff Checklist
kind: reference
authority: operational
status: current
owner: ops
truth_surfaces:
  - workitem_handoff
  - team_pilot_readiness
lifecycle: active-core
review_after: 2026-08-25
supersedes: []
superseded_by: null
related_docs:
  - docs/ops/TEAM_PILOT_RUNBOOK.md
  - docs/ops/templates/PR_HANDOFF_TEMPLATE.md
---
# WorkItem Handoff Checklist

Use this checklist after Proposal acceptance and before implementation. It is a
compact bridge between `goalrail work item show` and the PR handoff template.

## Identity

- [ ] WorkItem ID captured.
- [ ] Goal ID captured.
- [ ] Contract ID captured.
- [ ] Plan ID captured.
- [ ] Proposal ID captured.
- [ ] Repo binding ID checked when relevant.

## Scope Readiness

- [ ] Title reflects the approved Contract.
- [ ] Summary is specific enough for implementation.
- [ ] Scope is bounded to the accepted WorkItem.
- [ ] Acceptance refs are visible or summarized.
- [ ] Proof/check expectations are visible or summarized.
- [ ] Non-goals are explicit.

## Agent Controls

- [ ] `next_action` was inspected.
- [ ] `command_packet` was used when present.
- [ ] `mutates_state` was reported before mutating commands.
- [ ] `requires_human_approval` gates were honored.
- [ ] `safety_note` and `stop_condition` were read.
- [ ] `related_ids` were preserved in notes and PR handoff.

## Local and Secret Safety

- [ ] `.goalrail/project.yml` remains local and untracked.
- [ ] Auth files were not read into durable outputs.
- [ ] Tokens, runner bearer tokens, JWT contents, local DB passwords, provider
      credentials, private host details, private paths, and temporary passwords
      remain redacted.

## Deferred Boundaries

- [ ] Runner checkout was not run.
- [ ] Checkout prepare was not run unless separately approved.
- [ ] Execution prepare was not run.
- [ ] Execution was not run.
- [ ] Gate/proof/verification/completion were not run.

## PR Handoff

- [ ] `docs/ops/templates/PR_HANDOFF_TEMPLATE.md` is filled in.
- [ ] Checks run are listed exactly.
- [ ] Deferred work is recorded.
- [ ] Human gates are documented.
