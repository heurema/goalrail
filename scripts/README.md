# Scripts

Utility scripts for repository hygiene and future bounded delivery work.

Current rule:
- scripts may support docs, checks, or repo maintenance
- scripts must not imply a real Goalrail runtime exists before Phase 1 implementation begins

## Current scripts

- `check-staged.sh` — runs the docs-check changed-files ratchet against staged files, including repo-structure placement rules.
- `local-refresh-plan.sh` — prints non-destructive component refresh guidance
  for local dogfood changes based on changed paths or explicit path arguments.
