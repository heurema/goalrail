# Milestone 1 Report

## Scope

Public documentation, deployment guides, and GitHub metadata were updated for
the first visible Goalrail rebrand pass.

Changed:

- root public docs: `README.md`, `SECURITY.md`, `RELEASING.md`
- deploy Markdown guides under `deploy/`
- public docs under `docs/`
- GitHub issue/discussion metadata: `.github/ISSUE_TEMPLATE/config.yml`
- maintainer/reviewer metadata comments: `.github/MAINTAINER`, `.github/reviewers`

Not changed in this milestone:

- Python package/import root: `goalrail`
- CLI commands and examples: `goalrail`, `goalrail`
- environment variables: `GOALRAIL_*`
- local state directories: `.goalrail`, `.goalrail`, `.goalrail`
- GHCR image names: `ghcr.io/heurema/goalrail-*`
- deploy service/resource names such as `goalrail`, `goalrail-db`,
  `goalrail-server`, `goalrail-creds`
- workflow gates and repository guards using `github.repository`
- old bot identities such as `goalrail-ci[bot]` and
  `goalrail <noreply@goalrail.dev>`
- historical PR/issue links to the old upstream repo

## Notes

- `README.md` now points at `goalrail.dev`.
- Render/Railway public repo links now point at `heurema/goalrail`.
- `docs/GOALRAIL_BOT_SETUP.md` was renamed to
  `docs/GOALRAIL_BOT_SETUP.md`, with an explicit transition note explaining
  the still-legacy bot identity values.
- `docs/databricks.md` was updated in prose while preserving historical PR
  links to `heurema/goalrail`.

## Checks

Run before commit:

- `codebase-memory-mcp cli index_status '{"project":"Users-vi-personal-heurema-goalrail"}'`
- scoped searches for remaining `Goalrail`, `goalrail.dev`,
  `heurema/goalrail`, and accidental `Goalrail-ai/Goalrail`
- prose-only lowercase `goalrail` scan for target Markdown docs
- `git diff --check`
