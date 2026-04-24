# TrialOps Demo Sandbox — Self-Serve Demo Notes

## Purpose

Capture the future self-serve direction now without building it yet.

Current decision:
- do **not** build a public live AI demo first
- do **not** expose arbitrary user-entered prompts
- do build the asset model so a future guided replay can reuse the same scenario and proof artifacts

## Future product shape

The first self-serve demo should be a **guided interactive replay**.

Target user journey:

1. choose a scenario
2. read the business request
3. step through `clarify -> contract -> task plan -> proof`
4. optionally adjust one or two bounded parameters
5. understand the process without depending on a live code-writing runtime

This is intentionally different from:
- public autonomous execution
- arbitrary prompt-to-code generation
- an expectation that the website is the full Goalrail product

## What to capture now

Phase 0 and later phases should preserve:
- scenario metadata in files
- deterministic scenario IDs
- deterministic proof-pack paths
- deterministic replay event paths
- fake seed data only
- stable step names for presenter narration and future UI replay
- proof artifacts that can be read by humans without a running system

## What not to implement yet

Do not build now:
- public website for the demo
- live AI execution behind a public form
- auth or account system for demo visitors
- multi-tenant storage
- arbitrary scenario authoring UI
- real customer data import
- security-sensitive execution model for anonymous users

## Future repo contract

Future executable sandbox repo:
- `heurema/goalrail-demo`

Planned asset layout:

```text
demo/
  scenarios/
    workflow-change.yaml
    field-change.yaml
    bugfix.yaml
    policy-review.yaml
  proof-packs/
    workflow-change/
      business-request.md
      contract-draft.md
      task-plan.md
      proof.md
      readout.md
  replay/
    workflow-change.events.jsonl
```

Rule:
- the current Goalrail repo defines these contracts only
- the separate demo repo implements them later

## Scenario manifest contract

Every future `demo/scenarios/*.yaml` file should include:

- `id`
- `title`
- `business_request`
- `why`
- `primary_flow`
- `touched_areas`
- `proof_expectations`
- `demo_risk`
- `phase_status`
- `future_replay_assets`

Recommended meaning:
- `phase_status` = `planned`, `ready_for_implementation`, `implemented`, or `future_card`
- `future_replay_assets` = references to proof-pack files, replay events, and optional presenter assets

## Proof-pack contract

Every future implemented scenario should have:

- `business-request.md`
- `contract-draft.md`
- `task-plan.md`
- `proof.md` and/or `readout.md`

Rules:
- paths must be predictable
- artifacts must be readable without the running app
- proof must match the demo behavior actually shown

## Replay event contract

If replay events are added later, each `*.events.jsonl` line should be machine-readable and stable.

Minimum event fields:
- `scenario_id`
- `step_id`
- `event_type`
- `timestamp`
- `artifact_ref`
- `note`

Suggested event types:
- `scenario_selected`
- `business_request_viewed`
- `clarification_presented`
- `contract_viewed`
- `task_plan_viewed`
- `proof_viewed`
- `demo_completed`

## Live mode vs replay mode later

### Live mode

Uses:
- running backend/frontend sandbox
- seeded local state
- prepared proof pack as presenter backup

Best for:
- founder-led live demos
- qualification or pilot conversations

### Replay mode

Uses:
- scenario manifest
- proof pack
- replay event stream
- bounded UI controls

Best for:
- website visitors
- asynchronous sales education
- low-risk process walkthroughs

Rule:
- replay mode should reuse the same scenario IDs and proof-pack structure as live mode
- replay mode should not require live code execution to deliver value
