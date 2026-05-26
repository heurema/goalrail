---
id: goalrail_agent_skill
title: Goalrail Agent Skill
kind: reference
authority: operational
status: current
owner: ops
truth_surfaces:
  - agent_guidance
  - team_pilot_readiness
  - command_packet_handoff
lifecycle: active-core
review_after: 2026-08-25
supersedes: []
superseded_by: null
related_docs:
  - docs/INDEX.md
  - docs/product/GOALRAIL_MVP_BLUEPRINT.md
  - docs/ops/TEAM_PILOT_RUNBOOK.md
  - docs/ops/LOCAL_DOGFOOD_RUNBOOK.md
  - docs/ops/templates/PR_HANDOFF_TEMPLATE.md
---
# Goalrail Agent Skill

## Purpose

This guide defines how an agent should operate Goalrail during the Team Pilot
stage.

For this stage, Goalrail is a contract-first PR control plane. It helps a small
team move from Goal to Contract to Plan to Proposal to WorkItem to PR handoff.
It is not an autonomous executor, not a gate/proof engine, and not a replacement
for issue trackers, GitHub PR review, or human approval.

## Agent Responsibilities

An agent working in Goalrail should:
- use current repo source and current ops docs before acting;
- read CLI JSON before deciding the next command;
- follow `next_action.command_packet` when it is present instead of inventing
  a command;
- report whether a command mutates state before running it;
- stop at human approval gates;
- keep local marker files and secrets out of commits and reports;
- produce PR handoffs that include Goalrail IDs, scope, non-goals, checks,
  deferred work, and boundary confirmations.

## Reading CLI JSON

Prefer `--format json` when inspecting Goalrail state. Treat these fields as
operator-facing control signals:

- `schema_version`: confirms the CLI output shape.
- `mode`: confirms whether the command is using server mode or local mode.
- `server_url`: identifies the server context without exposing credentials.
- `auth_session`: safe auth observability metadata. It may show whether a stored
  access token was used, whether refresh was attempted, whether refresh
  succeeded, and the reason. It must not include bearer values, refresh tokens,
  auth file contents, JWT contents, or provider credentials.
- `organization_id`, `project_id`, and `repo_binding_id`: confirm the active
  Goalrail context.
- `local_config_path`: confirms marker discovery, usually
  `.goalrail/project.yml`.
- `next_action`: the authoritative handoff for the next supported operation.

## Using Next Actions

Read `next_action.kind` as the workflow phase. Do not assume every kind is safe
to run. The agent must inspect the rest of the next action first.

Use `next_action.command_packet` as the preferred command source when present.
It should provide:
- `cwd`: where to run the command relative to the repository root;
- `argv`: the exact argument vector;
- `description`: what the command does;
- `safety_note`: what the command does and does not do;
- `stop_condition`: where the agent must stop after running it.

If a `command_packet` is absent, do not fabricate one. Report the supported
next action, explain the missing packet, and ask for explicit human direction
when the command would mutate state.

## State Mutation and Human Gates

If `next_action.mutates_state` is true, report that before running the command.

If `next_action.requires_human_approval` is true, stop and wait for explicit
approval. Human approval gates include:
- Contract approval;
- Proposal acceptance;
- any mutating command marked as requiring human approval;
- PR review and merge;
- runner checkout, checkout prepare, execution, gate, proof, verification, or
  completion if a later approved Contract ever brings one of those stages back
  into scope.

Team Pilot work stops before runner and execution boundaries unless a future
Contract explicitly approves them.

## Related IDs

Use `next_action.related_ids` to preserve traceability. PR handoffs should carry
the available Goalrail IDs:
- `goal_id`;
- `contract_id`;
- `plan_id`;
- `proposal_id`;
- `work_item_id` or `task_id`;
- `repo_binding_id` when relevant.

Do not replace Goalrail IDs with local notes or inferred identifiers.

## Local Marker and Auth Handling

`.goalrail/project.yml` is a local marker needed by current CLI flows. It must
remain local, untracked, and uncommitted unless a later approved slice changes
marker policy.

When `HOME` is overridden for CLI auth context, Go may attempt to use a module
or build cache under that temporary home. Use explicit temporary cache
environment variables for Go commands when needed:

```bash
GOMODCACHE=/tmp/goalrail-gomodcache
GOCACHE=/tmp/goalrail-gocache
```

These are examples, not durable repo configuration.

## Secret Redaction

Never print or commit:
- runner bearer tokens;
- access tokens;
- refresh tokens;
- auth JSON;
- JWT contents;
- local DB passwords;
- provider credentials;
- private host details;
- private local paths;
- temporary passwords;
- auth files.

Use placeholders such as `<runner-bearer-token>`, `<local-db-password>`, or
`<private-path>` when a value needs to be described.

## Local Refresh Caution

After server code changes, local launchd or wrapper-based dogfood setups may
continue running a stale temporary server binary. Rebuild or restart from
current source before validating server behavior. Health checks alone can show
that a server is alive; they do not prove it is running the expected source.

Useful read-only checks include:
- `/livez`;
- `/readyz`;
- `/version`;
- source grep for route or command availability;
- current-source CLI help through `go run`.

Do not wipe local Postgres, rotate secrets, or clean up server state as part of
a normal Team Pilot PR unless a future Contract explicitly authorizes it.

## Forbidden Actions During Team Pilot

Do not run or implement as part of Team Pilot unless a later approved Contract
explicitly says otherwise:
- runner checkout;
- checkout prepare;
- execution prepare;
- execution;
- gate, proof, or verification behavior;
- WorkItem completion;
- provider OAuth, provider clients, or stored credentials;
- `.goalrail/templates`;
- `.goalrail` tracking or `.gitignore` marker policy changes;
- broad autonomous agent behavior;
- runtime adapters or a generic execution platform.

## PR Handoff

Use `docs/ops/templates/PR_HANDOFF_TEMPLATE.md` for PR bodies and review
handoffs. At minimum, include:
- Goalrail IDs;
- summary;
- scope delivered;
- non-goals respected;
- checks run;
- artifacts changed;
- deferred work;
- human gates respected;
- secret and local-state confirmation;
- runner/execution boundary confirmation.

## Missing GitHub Checks

When preparing or merging a Team Pilot PR, distinguish missing checks from
failing checks. Missing checks mean the PR has no created status checks for the
current head, for example an empty `statusCheckRollup`, `gh pr checks`
reporting no checks, or no workflow runs/check-runs for the branch or head
commit.

In that case, stop before merge and collect diagnostics:
- PR Checks tab state;
- Actions tab filtered by branch;
- workflow runs for branch and head commit;
- branch protection or ruleset required check names;
- workflow approval, suppression, or incident state;
- required check name or source mismatches.

Do not merge without green required checks. Do not repeatedly push empty commits
to force checks. If no run exists to rerun, use at most one deliberate
empty-commit retrigger only after a human or repo admin confirms the setting or
GitHub incident has been resolved. If checks remain missing, ask for repo-admin
UI inspection instead of bypassing the gate.

## Partial Progress and Blockers

When blocked, report:
- the exact command that failed;
- the exact failure text, with secrets redacted;
- whether the command was read-only or mutating;
- what state was confirmed before the failure;
- the next safe read-only command, if one exists.

Avoid hidden raw API fallbacks. Use a raw API request only when the supported CLI
surface is insufficient, the request is read-only, and the fallback is clearly
reported.
