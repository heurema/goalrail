---
id: goalrail_repo_structure
title: Goalrail Repository Structure
kind: reference
authority: operational
status: current
owner: docs-governance
truth_surfaces:
  - repository_layout
  - path_placement_rules
lifecycle: active-core
review_after: 2026-07-23
supersedes: []
superseded_by: null
related_docs:
  - docs/INDEX.md
  - docs/ops/COMPONENTS.yaml
  - tools/docs-check/README.md
---
# Goalrail Repository Structure

This is the operational map for where things live in this repository.
It does not override product canon, architecture canon, or the component map.

Use it together with:
- [Docs Index](../INDEX.md)
- [Component map](COMPONENTS.yaml)
- [Docs check](../../tools/docs-check/README.md)

## Top-level map

| Path | Role | Add here when |
| --- | --- | --- |
| `apps/` | Product and demo application code | You add real implementation, prototypes, CLIs, or app workspaces |
| `docs/` | Product canon, ops docs, brand docs, research, and references | You add durable project documentation or promoted reference material |
| `tools/` | Repository tooling, checks, schemas, and fixtures | You add automation that validates or maintains the repo |
| `scripts/` | Thin maintenance commands and wrappers | You add helper commands that do not imply a Goalrail runtime exists |
| `.goalrail/` | Goalrail working overlay | You add working memory, research backlog, flow specs, eval specs, or slice reports |
| `.punk/` | Punk-owned overlay | You add Punk publishing drafts, receipts, or public narrative artifacts |
| `.github/` | GitHub workflows and repository templates | You add CI or GitHub-native repository process files |
| `.codex/` | Agent routing references | You adjust thin local agent routing, not project truth |
| root files | Public/legal/community entry points and minimal repo config | You update existing root policy or entry files only |

## Placement rules

| Need | Put it in |
| --- | --- |
| New app, frontend, demo, CLI, or service code | `apps/<area>/...` |
| Go CLI module | `apps/cli/` |
| Canonical CLI binary entrypoint | `apps/cli/cmd/goalrail/` |
| CLI-only internal packages and DTOs | `apps/cli/internal/` |
| Go server module | `apps/server/` |
| Canonical server binary entrypoint | `apps/server/cmd/goalrail-server/` |
| Server-only internal packages | `apps/server/internal/` |
| Server DTOs and domain value types | `apps/server/internal/spine/` |
| Server intake service logic | `apps/server/internal/intake/` |
| Server Goal promotion and readiness service logic | `apps/server/internal/goal/` |
| Server in-memory stores and event logs | `apps/server/internal/store/` and `apps/server/internal/eventlog/` |
| Web workspace package files | `apps/web/` |
| Real console web shells | `apps/web/console/` and `apps/web/console-ru/` |
| Demo change-packet web apps | `apps/web/demo-change-packet/` and `apps/web/demo-change-packet-ru/` |
| Product canon | `docs/product/` |
| Operational status, next steps, decisions, component map | `docs/ops/` |
| Repository structure guidance | `docs/ops/REPO_STRUCTURE.md` |
| Brand system docs | `docs/brand/` |
| Design screenshots or visual references | `docs/reference/design/` |
| Research that is not canon yet | `docs/research/` or `.goalrail/knowledge/`, depending on durability |
| Docs-check fixtures and schemas | `tools/docs-check/fixtures/` and `tools/docs-check/schemas/` |
| Repository hygiene commands | `scripts/` |
| Goalrail working slice reports | `.goalrail/work/` |
| Future Goalrail flow/spec artifacts | `.goalrail/flows/` |
| Future eval/spec artifacts | `.goalrail/evals/` |
| Punk publishing work | `.punk/publishing/` |

## Root rules

Do not add new top-level paths casually.
If a new top-level path is really needed, update this file and the repo-structure checker allowlist in the same change.

Forbidden root locations by default:
- `package.json`, `package-lock.json`, `pnpm-lock.yaml`, `yarn.lock`, `bun.lockb`
- `node_modules/`, `dist/`, `output/`
- `design/`, `evals/`, `schemas/`
- `knowledge/`, `work/`, `flows/`, `publishing/`

## Local staged-files check

Before committing structure-sensitive changes, run:

```bash
scripts/check-staged.sh
```

This checks staged paths with `tools/docs-check` changed-files mode.
It blocks unregistered top-level paths and still runs the docs ratchet for changed docs.

## Agent rule

Agents should keep `AGENTS.md` and project skills thin.
Use this file as the durable placement reference instead of duplicating layout rules in agent routing files.
