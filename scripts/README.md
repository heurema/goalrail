# Scripts

Utility scripts for repository hygiene and future bounded delivery work.

Current rule:
- scripts may support docs, checks, or repo maintenance
- scripts must not imply a real Goalrail runtime exists before Phase 1 implementation begins

## Current scripts

- `check-staged.sh` — runs the docs-check changed-files ratchet against staged files, including repo-structure placement rules.
- `pr_intake_gate.py` — deterministic GitHub PR Intake Gate used by `.github/workflows/pr-intake-gate.yml`; it reads trusted base-branch code and GitHub PR metadata only.
