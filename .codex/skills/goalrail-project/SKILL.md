---
name: goalrail-project
description: "Use when working inside the Goalrail repository and you need project-specific guidance on canon, live docs navigation, docs governance, or bounded implementation slices. The skill is a router, not a knowledge cache: start from docs/INDEX.md and current ops docs, then use references only for workflow guardrails."
---

# Goalrail Project

## Overview

Use this skill for work inside Goalrail.
It keeps the agent aligned with canonical docs, the current status anchor, docs-governance v0, and the bounded-slice implementation discipline.

This skill is intentionally thin.
It should route the agent to live project truth, not duplicate fast-changing project facts.

Goalrail is still a docs-first planning repo.
Do not invent runtime, CLI, gate/proof, web, tracker, or adapter implementation that the repo does not actually contain.

## When to use

Use this skill when you need to:
- orient a fresh session inside Goalrail
- edit product, ops, or governance docs
- check whether a claim matches current repo truth
- work with `tools/docs-check/`
- interpret `docs/ops/COMPONENTS.yaml`
- prepare or implement one bounded Goalrail slice

## Workflow

1. Start with `docs/INDEX.md`.
2. Use live docs as truth:
   - `docs/product/*` for canon
   - `docs/ops/STATUS.md` for current state
   - `docs/ops/COMPONENTS.yaml` for implementation status
3. Identify the task lane:
   - canon/product change
   - docs/ops synchronization
   - bounded implementation slice
4. Read only the needed reference file:
   - `references/project-map.md`
   - `references/docs-workflow.md`
   - `references/implementation-slice.md`
5. If the topic is new or not obvious in the skill references, trust `docs/INDEX.md` and search the repo docs directly.
6. Keep scope bounded.
7. If docs metadata, links, claims, or component status are touched, run the docs checker.
8. Before claiming anything is implemented, verify both code reality and `docs/ops/COMPONENTS.yaml`.

## Freshness model

- This skill is a router and workflow guard, not a project knowledge cache.
- `docs/INDEX.md` is the live navigation map.
- Canonical and ops docs are fresher than this skill.
- If a new doc appears in `docs/INDEX.md`, follow the index even if the skill does not name that doc yet.
- If the skill and live docs disagree, live docs win.

## Discovery under change

When a topic changes quickly, use this order:
1. `docs/INDEX.md`
2. the relevant canonical or ops doc family
3. targeted repo search when the topic is new

Useful local searches:

```bash
rg -n "pricing|payment|billing|offer" docs
```

```bash
rg -n "status|planned|implemented|docs_only|prototype" docs/ops
```

```bash
rg -n "<topic-keyword>" docs knowledge work publishing
```

## Hard guards

- `docs/product/*` is canonical product truth.
- `docs/ops/*` is the operating/status layer.
- `README.md` and public-entry docs never override canon.
- Adjacent projects such as Punk contribute discipline, not truth, roadmap, or dependency.
- Governance docs control process and metadata posture; they do not override product canon.
- Most Goalrail components are still `docs_only` or `planned`.
- Governance rollout v0 is complete; do not expand it by default.
- Prefer one bounded slice at a time.
- The active implementation checkpoint target comes from `docs/ops/STATUS.md` and `docs/ops/NEXT.md`; verify it before starting work.

## Task lanes

### Canon or product work

Read `references/project-map.md` first.
If product meaning changes, update canon first and only then dependent ops or public-facing surfaces.

### Docs and governance work

Read `references/docs-workflow.md`.
Use deterministic checks only.
Do not build extra governance platform surface unless the user asks for a concrete pain-driven change.

### Implementation work

Read `references/implementation-slice.md`.
Start from one roadmap checkpoint and one proof target.
Do not jump directly from roadmap text to broad implementation.

## Validation

When docs or metadata are touched, use these commands as needed:

```bash
python3 tools/docs-check/docs_check.py \
  --fixtures evals/cases/docs \
  --self-test \
  --report-json /tmp/goalrail-docs-fixtures-report.json \
  --report-md /tmp/goalrail-docs-fixtures-report.md
```

```bash
python3 tools/docs-check/docs_check.py \
  --root . \
  --mode report-only \
  --report-json /tmp/goalrail-docs-check-report.json \
  --report-md /tmp/goalrail-docs-check-report.md
```

```bash
python3 tools/docs-check/docs_check.py \
  --root . \
  --mode changed-files \
  --changed-files-file /tmp/changed-doc-files.txt \
  --report-json /tmp/goalrail-docs-changed-report.json \
  --report-md /tmp/goalrail-docs-changed-report.md
```

Rules:
- live `report-only` scans must not be treated as repo-wide hard gates
- changed-files mode is the active ratchet for new deterministic hard violations
- generated reports should not be committed

## When to update this skill

Update the skill or its references only when one of these changes:
- source-of-truth order
- read order in `docs/INDEX.md`
- stable doc families or routing lanes
- docs-check workflow or commands
- implementation discipline

Do not update the skill for ordinary doc-content changes such as pricing, offer details, wording, or status edits inside the docs themselves.
