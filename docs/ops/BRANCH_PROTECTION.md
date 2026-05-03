---
id: goalrail_branch_protection
title: Goalrail Branch Protection
kind: reference
authority: operational
status: current
owner: repository-governance
truth_surfaces:
  - github_branch_protection
  - required_pr_checks
lifecycle: active-core
review_after: 2026-07-29
supersedes: []
superseded_by: null
related_docs:
  - docs/ops/DECISIONS.md
  - docs/ops/COMPONENTS.yaml
  - .github/workflows/codex-review-gate.yml
  - .github/workflows/docs-check.yml
  - .github/workflows/pr-intake-gate.yml
  - .github/workflows/repo-checks.yml
---
# Goalrail Branch Protection

This is the operational record for the active GitHub branch protection policy.
Branch protection is GitHub external configuration; workflow files alone do not
represent the active repository policy.

## Target branch

- `main`

## Required status check contexts

The protected `main` branch requires these PR checks before merge:

- `docs-check`
- `codex-review-gate`
- `pr-intake-gate`
- `go (apps/cli)`
- `go (apps/server)`
- `go (apps/web/pilot-intake-ru/server)`
- `web workspaces`

## Verified settings

Verified on 2026-05-02:

- required status checks are enabled;
- required checks must be up to date before merge (`strict: true`);
- administrators are included in enforcement;
- force pushes are disabled;
- branch deletion is disabled;
- conversation resolution is required;
- required approving reviews are not enabled in this slice;
- signed commits are not required in this slice;
- merge queue is not enabled in this slice.

## Verification commands

Classic branch protection settings:

```bash
gh api repos/heurema/goalrail/branches/main/protection \
  --jq '{
    contexts: .required_status_checks.contexts,
    strict: .required_status_checks.strict,
    enforce_admins: .enforce_admins.enabled,
    allow_force_pushes: (.allow_force_pushes.enabled // false),
    allow_deletions: (.allow_deletions.enabled // false),
    required_conversation_resolution: (.required_conversation_resolution.enabled // false),
    required_reviews: (.required_pull_request_reviews != null),
    restrictions: (.restrictions != null)
  }'
```

Required signed commits setting:

```bash
gh api repos/heurema/goalrail/branches/main/protection/required_signatures \
  --jq '{required_signatures: (.enabled // false)}'
```

Repository rulesets / merge queue check:

```bash
gh api repos/heurema/goalrail/rulesets \
  --jq '[.[] | {id, name, target, enforcement, rules: [.rules[]?.type]}]'
```

Expected required contexts:

```text
docs-check
codex-review-gate
pr-intake-gate
go (apps/cli)
go (apps/server)
go (apps/web/pilot-intake-ru/server)
web workspaces
```

Expected values for the other recorded settings:

```text
strict: true
enforce_admins: true
allow_force_pushes: false
allow_deletions: false
required_conversation_resolution: true
required_reviews: false
restrictions: false
required_signatures: false
rulesets: []
```

## Emergency path

- No silent direct push to `main`.
- Emergency override requires a follow-up entry in `docs/ops/DECISIONS.md`.
- Any temporary weakening of branch protection must be reverted and documented.
- If GitHub configuration and this file disagree, verify with `gh api` first;
  then update either the external setting or this record in a bounded PR.
