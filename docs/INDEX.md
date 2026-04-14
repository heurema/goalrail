# Goalrail Docs Index

> Стартовая точка для работы над проектом.

## Read in this order

### Core product canon
1. `docs/product/GOALRAIL_PRODUCT_BRIEF.md`
2. `docs/product/GOALRAIL_MVP_BLUEPRINT.md`
3. `docs/product/GOALRAIL_BUILD_ROADMAP.md`
4. `docs/PROJECT_SPINE_SCHEMA.md`
5. `docs/product/GOALRAIL_PARALLEL_EXECUTION_MODEL.md`
6. `docs/product/GOALRAIL_IMPLEMENTATION_GUIDE.md`
7. `docs/adr/ADR-0001-runtime-neutral-cli-first.md`
8. `docs/adr/ADR-0002-single-writer-and-advisory-panels.md`

### Supporting product / GTM / design docs
9. `docs/product/GOALRAIL_ONE_PAGER.md`
10. `docs/product/GOALRAIL_PAIN_STATEMENT.md`
11. `docs/product/GOALRAIL_MESSAGE_HIERARCHY.md`
12. `docs/product/GOALRAIL_PROVIDER_BOUNDARIES.md`
13. `docs/product/GOALRAIL_COMPETITOR_MAP.md`
14. `docs/product/GOALRAIL_FIGMA_WORKFLOW.md`
15. `docs/product/GOALRAIL_DESIGN_DECISIONS.md`
16. `docs/product/GOALRAIL_TOOLING_DECISION.md`
17. `docs/product/GOALRAIL_REFERENCE_DECISION.md`
18. `docs/product/GOALRAIL_LANDING_COPY.md`

### Ops
19. `docs/ops/STATUS.md`
20. `docs/ops/NEXT.md`
21. `docs/ops/DECISIONS.md`
22. `docs/ops/COMPONENTS.yaml`

### Research / sources
23. `docs/research/AI_SDLC_RUST_PRODUCT_SUMMARY_SOURCE.md`
24. `design/reference_screens/`

## Roles of the main docs

### Core product source
- `GOALRAIL_PRODUCT_BRIEF.md` — product thesis, positioning, users, MVP promise
- `GOALRAIL_MVP_BLUEPRINT.md` — canonical MVP architecture and layer map
- `GOALRAIL_BUILD_ROADMAP.md` — phases, checkpoints, exits, demo targets
- `GOALRAIL_PARALLEL_EXECUTION_MODEL.md` — execution groups, advisory panels, isolation, fan-in barrier
- `GOALRAIL_IMPLEMENTATION_GUIDE.md` — operating rules for bounded implementation

### Kernel support notes
- `PROJECT_SPINE_SCHEMA.md` — canonical objects, derived views, event envelope, ownership boundaries
- `docs/adr/ADR-0001-runtime-neutral-cli-first.md` — runtime boundary decision
- `docs/adr/ADR-0002-single-writer-and-advisory-panels.md` — execution vs advisory boundary decision

### Supporting product docs
- `GOALRAIL_ONE_PAGER.md` — concise product summary for internal/external use
- `GOALRAIL_PAIN_STATEMENT.md` — market pain framing and sources
- `GOALRAIL_MESSAGE_HIERARCHY.md` — messaging stack and hero narrative
- `GOALRAIL_PROVIDER_BOUNDARIES.md` — supplement-vs-replace boundary rules
- `GOALRAIL_COMPETITOR_MAP.md` — competitor and wedge mapping
- `GOALRAIL_FIGMA_WORKFLOW.md` — design workflow guidance
- `GOALRAIL_DESIGN_DECISIONS.md` — public entry flow and design constraints
- `GOALRAIL_TOOLING_DECISION.md` — Stitch vs Figma workflow decision
- `GOALRAIL_REFERENCE_DECISION.md` — external reference posture
- `GOALRAIL_LANDING_COPY.md` — first public scene copy draft

### Ops source
- `STATUS.md` — where the project is now
- `NEXT.md` — next bounded slices only
- `DECISIONS.md` — compact decision log
- `COMPONENTS.yaml` — compact component map

### Research / reference assets
- `docs/research/*` — source material and background research
- `design/reference_screens/` — collected screenshots and explorations from the final pack

## Source-of-truth priority

1. Core product canon in `docs/product/` (`GOALRAIL_PRODUCT_BRIEF.md` -> `GOALRAIL_IMPLEMENTATION_GUIDE.md`)
2. Kernel support notes (`docs/PROJECT_SPINE_SCHEMA.md`, `docs/adr/*`)
3. Supporting product docs in `docs/product/`
4. `docs/ops/*`
5. chat context

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
