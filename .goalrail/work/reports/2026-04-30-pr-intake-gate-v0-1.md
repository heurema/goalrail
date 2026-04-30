# PR Intake Gate v0.1 Report

- Date: 2026-04-30
- Owner: Vitaly
- Related goal: `.goalrail/work/goals/goal_pr_intake_gate_v0_1.md`
- Outcome: repository-governance gate added
- Proof refs:
  - `python3 tools/pr-intake-gate/test_pr_intake_gate.py`
  - `python3 -m py_compile scripts/pr_intake_gate.py tools/pr-intake-gate/test_pr_intake_gate.py`
  - `scripts/check-staged.sh`

## Summary

Added Goalrail PR Intake Gate v0.1 as repository governance only. The gate keeps trusted repository authors low-friction and applies stricter intake to external contributors.

## What changed

- Added a safe `pull_request_target` workflow that checks out trusted base code and runs `scripts/pr_intake_gate.py`.
- Added a Goalrail-specific intake policy for trusted authors, high-risk paths, linked intent, external context, and labels.
- Added a stdlib-only deterministic gate implementation that reads PR metadata/files through the GitHub API and never executes PR head code.
- Added fixture coverage for trusted author pass, fallback association trust, external strict context failures, high-risk blocking, maintainer override, accepted-for-pr, first-time contributor labeling, markdown section parsing, and best-effort label/comment side effects.
- Updated PR template, contributor docs, scripts docs, and repository governance component paths.

## Checks run

- `git diff --check` - PASS
- `python3 -m py_compile scripts/pr_intake_gate.py tools/pr-intake-gate/test_pr_intake_gate.py` - PASS
- `python3 tools/pr-intake-gate/test_pr_intake_gate.py` - PASS
- `scripts/check-staged.sh` - PASS, 0 hard findings, 0 warnings for staged docs-check changed-files mode
- `python3 tools/docs-check/docs_check.py --fixtures tools/docs-check/fixtures --self-test --report-json "$tmpdir/fixtures.json" --report-md "$tmpdir/fixtures.md"` - PASS; fixture self-test reported expected hard/warning findings with 0 fixture failures

## Follow-ups

- Configure branch protection after the PR lands so `pr-intake-gate` is required on `main`.
- Create/update GitHub labels after pushing the branch.
- Test the trusted-author live path with a temporary high-risk draft PR after the gate lands on `main`.
- Consider extracting a reusable shared PR intake action only after Goalrail, Punk, and Signum policies have all stabilized.
