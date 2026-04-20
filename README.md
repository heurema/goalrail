# Goalrail

Goalrail is an intent-to-delivery layer for software teams.

**Tagline:** от бизнес-цели до проверенного изменения в коде.

## What this repository is

This repository currently holds the canonical documentation baseline for Goalrail:

- core product docs in `docs/product/`
- kernel support docs in `docs/PROJECT_SPINE_SCHEMA.md` and `docs/adr/`
- operating docs in `docs/ops/`
- supporting GTM / positioning / design docs in `docs/product/`
- reference screens in `design/reference_screens/`
- repo-local agent guidance in `AGENTS.md`

It does **not** contain a real implementation baseline yet.

## Current status

- the product thesis is fixed
- the MVP architecture is documented
- the build roadmap is documented
- the parallel execution model is documented
- the project spine schema note and kernel ADRs are documented
- the implementation workflow is documented
- supplemental positioning / design docs from the final pack are merged
- implementation will proceed through `punk` in bounded slices

## Read in this order

1. `docs/INDEX.md`
2. `docs/product/GOALRAIL_PRODUCT_BRIEF.md`
3. `docs/product/GOALRAIL_MVP_BLUEPRINT.md`
4. `docs/product/GOALRAIL_BUILD_ROADMAP.md`
5. `docs/PROJECT_SPINE_SCHEMA.md`
6. `docs/product/GOALRAIL_PARALLEL_EXECUTION_MODEL.md`
7. `docs/product/GOALRAIL_IMPLEMENTATION_GUIDE.md`
8. `docs/adr/ADR-0001-runtime-neutral-cli-first.md`
9. `docs/adr/ADR-0002-single-writer-and-advisory-panels.md`
10. `docs/ops/STATUS.md`
11. `docs/ops/NEXT.md`
12. `docs/ops/DECISIONS.md`

## Working rule

- `docs/product/` core canon is the product source of truth
- `docs/PROJECT_SPINE_SCHEMA.md` and `docs/adr/` hold kernel support truth
- supplemental docs in `docs/product/` and `design/reference_screens/` support positioning/design work and do not override the canon
- `docs/ops/` is the current working layer
- implementation in this repo goes through `punk`
- changes should stay small, bounded, and reviewable

## Repo map

```text
.
├─ README.md
├─ AGENTS.md
├─ design/
│  └─ reference_screens/
├─ docs/
│  ├─ INDEX.md
│  ├─ PROJECT_SPINE_SCHEMA.md
│  ├─ adr/
│  ├─ ops/
│  └─ product/
├─ apps/
├─ scripts/
└─ .github/
```
