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

- Python package/import root: `omnigent`
- CLI commands and examples: `omnigent`, `omni`
- environment variables: `OMNIGENT_*`
- local state directories: `.omnigent`, `.omnigents`, `.omniagents`
- GHCR image names: `ghcr.io/omnigent-ai/omnigent-*`
- deploy service/resource names such as `omnigent`, `omnigent-db`,
  `omnigent-server`, `omnigent-creds`
- workflow gates and repository guards using `github.repository`
- old bot identities such as `omnigent-ci[bot]` and
  `omnigent <noreply@omnigent.ai>`
- historical PR/issue links to the old upstream repo

## Notes

- `README.md` now points at `golrail.dev`.
- Render/Railway public repo links now point at `heurema/goalrail`.
- `docs/OMNIGENT_BOT_SETUP.md` was renamed to
  `docs/GOALRAIL_BOT_SETUP.md`, with an explicit transition note explaining
  the still-legacy bot identity values.
- `docs/databricks.md` was updated in prose while preserving historical PR
  links to `omnigent-ai/omnigent`.

## Checks

Run before commit:

- `codebase-memory-mcp cli index_status '{"project":"Users-vi-personal-heurema-goalrail"}'`
- scoped searches for remaining `Omnigent`, `omnigent.ai`,
  `omnigent-ai/omnigent`, and accidental `Goalrail-ai/Goalrail`
- prose-only lowercase `omnigent` scan for target Markdown docs
- `git diff --check`

