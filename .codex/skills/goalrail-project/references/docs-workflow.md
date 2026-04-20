# Goalrail Docs Workflow

## Purpose

Use this workflow when editing docs, metadata, component status, or any claim about what Goalrail is and what is actually implemented.

The main drift risks are:
1. truth drift
2. implementation drift
3. process drift

## Truth model

Current truth model is:
- canonical doc content = source of truth
- frontmatter = machine-readable metadata
- `docs/INDEX.md` = human read-order and source-priority view
- checker = deterministic consistency guard, not semantic authority

## Freshness rule

This skill and its references are workflow guides, not a knowledge cache.
When docs change quickly:
- trust `docs/INDEX.md` as the live map
- trust canonical and ops docs over skill references
- treat new indexed docs as immediately usable even before the skill is updated

## Update order

When meaning changes:
1. update concept/product canon first
2. update architecture canon if boundaries changed
3. update governance/process docs only if process changed
4. update `docs/ops/*` to reflect current operating state
5. keep public-entry or advisory surfaces downstream from canon

Do not let ops or README lead canon.

## Current governance posture

Docs-governance v0 is complete.
That means:
- report-only repo-wide scans exist
- changed-files ratchet exists
- `COMPONENTS.yaml` anchors implementation status
- core docs have initial metadata

Do not start new governance rollout work by default.
Do not start Batch 2 metadata migration unless the user explicitly asks.
Do not add semantic or LLM-based checking.

## Claims discipline

Before writing implementation claims:
1. check the relevant canonical docs
2. check `docs/ops/STATUS.md`
3. check `docs/ops/COMPONENTS.yaml`
4. check whether real implementation paths exist

If any of those disagree, be conservative.
Prefer `planned`, `docs-only`, or explicit non-implementation wording.

## Discovery when new docs appear

If the repo gets a new doc family, do this:
1. find it through `docs/INDEX.md`
2. read the new doc family directly
3. use targeted search if needed
4. update the skill later only if the repo structure or routing rules changed

Example searches:

```bash
rg -n "pricing|payment|billing|offer" docs
```

```bash
rg -n "status|checkpoint|component|implemented|planned" docs/ops docs/product
```

## Docs checker usage

Use fixture self-test when checker logic, schemas, or fixtures change:

```bash
python3 tools/docs-check/docs_check.py \
  --fixtures evals/cases/docs \
  --self-test \
  --report-json /tmp/goalrail-docs-fixtures-report.json \
  --report-md /tmp/goalrail-docs-fixtures-report.md
```

Use live report-only scan for repo visibility:

```bash
python3 tools/docs-check/docs_check.py \
  --root . \
  --mode report-only \
  --report-json /tmp/goalrail-docs-check-report.json \
  --report-md /tmp/goalrail-docs-check-report.md
```

Use changed-files mode for local PR-like validation:

```bash
python3 tools/docs-check/docs_check.py \
  --root . \
  --mode changed-files \
  --changed-files-file /tmp/changed-doc-files.txt \
  --report-json /tmp/goalrail-docs-changed-report.json \
  --report-md /tmp/goalrail-docs-changed-report.md
```

## When to update the skill

Update the skill or references when these change:
- source-of-truth order
- read order in `docs/INDEX.md`
- stable doc families or routing lanes
- docs-check commands or workflow
- implementation discipline

Do not update the skill for ordinary content churn inside docs.
That includes changing prices, offer details, wording, payment terms, or routine status text.

## Scope rules

Allowed docs work usually means:
- bounded updates to canon, ops, or public docs
- metadata-only migration when explicitly requested
- status-anchor updates when implementation state changed

Avoid by default:
- new governance phases
- repo-wide hard gates
- generated report commits
- public CLI for docs tooling
- turning adjacent experiments into Goalrail truth
