# Goalrail Docs Index

> Стартовая точка для работы над проектом.

## Read in this order

1. `docs/product/GOALRAIL_PRODUCT_BRIEF.md`
2. `docs/product/GOALRAIL_MVP_BLUEPRINT.md`
3. `docs/product/GOALRAIL_BUILD_ROADMAP.md`
4. `docs/PROJECT_SPINE_SCHEMA.md`
5. `docs/product/GOALRAIL_PARALLEL_EXECUTION_MODEL.md`
6. `docs/product/GOALRAIL_IMPLEMENTATION_GUIDE.md`
7. `docs/adr/ADR-0001-runtime-neutral-cli-first.md`
8. `docs/adr/ADR-0002-single-writer-and-advisory-panels.md`
9. `docs/ops/STATUS.md`
10. `docs/ops/NEXT.md`
11. `docs/ops/DECISIONS.md`
12. `docs/ops/COMPONENTS.yaml`

## Roles of the main docs

### Product source
- `GOALRAIL_PRODUCT_BRIEF.md` — product thesis, positioning, users, MVP promise
- `GOALRAIL_MVP_BLUEPRINT.md` — canonical MVP architecture and layer map
- `GOALRAIL_BUILD_ROADMAP.md` — phases, checkpoints, exits, demo targets
- `GOALRAIL_PARALLEL_EXECUTION_MODEL.md` — execution groups, advisory panels, isolation, fan-in barrier

### Kernel support notes
- `PROJECT_SPINE_SCHEMA.md` — canonical objects, derived views, event envelope, ownership boundaries
- `docs/adr/ADR-0001-runtime-neutral-cli-first.md` — runtime boundary decision
- `docs/adr/ADR-0002-single-writer-and-advisory-panels.md` — execution vs advisory boundary decision

### Build rules
- `GOALRAIL_IMPLEMENTATION_GUIDE.md` — operating rules for bounded implementation

### Ops source
- `STATUS.md` — where the project is now
- `NEXT.md` — next bounded slices only
- `DECISIONS.md` — compact decision log
- `COMPONENTS.yaml` — compact component map

## Source-of-truth priority

1. `docs/product/*`
2. `docs/ops/*`
3. chat context

## Working rule

Advance one bounded slice at a time.
Implementation work proceeds through `punk`.
Every completed slice should leave behind:
- updated `STATUS.md`
- updated `NEXT.md`
- any changed decision in `DECISIONS.md`
- proof or demo note referenced from the completed slice

## Current top-level build thesis

Goalrail is:

**от бизнес-цели до проверенного изменения в коде**
