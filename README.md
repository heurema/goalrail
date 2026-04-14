# Goalrail

Goalrail is an intent-to-delivery layer for software teams.

**Tagline:** от бизнес-цели до проверенного изменения в коде.

## What this repository is

This repository currently holds the canonical documentation baseline for Goalrail:

- product docs in `docs/product/`
- operating docs in `docs/ops/`
- repo-local agent guidance in `AGENTS.md`

It does **not** contain a real implementation baseline yet.

## Current status

- the product thesis is fixed
- the MVP architecture is documented
- the build roadmap is documented
- the parallel execution model is documented
- the project spine schema note and kernel ADRs are documented
- the implementation workflow is documented
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

- `docs/product/` is the canonical product source
- `docs/ops/` is the current working layer
- implementation in this repo goes through `punk`
- changes should stay small, bounded, and reviewable

## Repo map

```text
.
├─ README.md
├─ AGENTS.md
├─ docs/
│  ├─ INDEX.md
│  ├─ product/
│  ├─ ops/
│  └─ adr/
├─ apps/
├─ crates/
├─ scripts/
└─ .github/
```
