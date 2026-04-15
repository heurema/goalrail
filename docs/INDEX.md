# Goalrail Docs Index

> Единая точка входа в проект.
> Сначала читаем бизнесовый канон, затем deployment и pilot-модель, затем product shape, затем архитектуру и только потом build / ops.

## Read in this order

### 1. Concept canon
1. `docs/product/GOALRAIL_PRODUCT_CONCEPT.md`
2. `docs/product/GOALRAIL_OPERATING_MODEL.md`
3. `docs/product/GOALRAIL_DEPLOYMENT_MODEL.md`
4. `docs/product/GOALRAIL_PILOT_MODEL.md`
5. `docs/product/GOALRAIL_GTM_MODEL.md`
6. `docs/product/GOALRAIL_ICP.md`
7. `docs/product/GOALRAIL_OFFER.md`

### 2. Product summary and market framing
8. `docs/product/GOALRAIL_PRODUCT_BRIEF.md`
9. `docs/product/GOALRAIL_ONE_PAGER.md`
10. `docs/product/GOALRAIL_PAIN_STATEMENT.md`
11. `docs/product/GOALRAIL_MESSAGE_HIERARCHY.md`

### 3. Product shape and external posture
12. `docs/product/GOALRAIL_DESIGN_DECISIONS.md`
13. `docs/product/GOALRAIL_LANDING_COPY.md`
14. `docs/product/GOALRAIL_PROVIDER_BOUNDARIES.md`
15. `docs/product/GOALRAIL_COMPETITOR_MAP.md`
16. `docs/product/GOALRAIL_REFERENCE_DECISION.md`

### 4. Architecture canon
17. `docs/product/GOALRAIL_MVP_BLUEPRINT.md`
18. `docs/PROJECT_SPINE_SCHEMA.md`
19. `docs/product/GOALRAIL_PARALLEL_EXECUTION_MODEL.md`
20. `docs/adr/ADR-0001-runtime-neutral-cli-first.md`
21. `docs/adr/ADR-0002-single-writer-and-advisory-panels.md`

### 5. Delivery and build
22. `docs/product/GOALRAIL_BUILD_ROADMAP.md`
23. `docs/product/GOALRAIL_IMPLEMENTATION_GUIDE.md`
24. `docs/ops/STATUS.md`
25. `docs/ops/NEXT.md`
26. `docs/ops/DECISIONS.md`
27. `docs/ops/COMPONENTS.yaml`

### 6. Research and source material
28. `docs/research/AI_SDLC_RUST_PRODUCT_SUMMARY_SOURCE.md`
29. `design/reference_screens/`

## Roles of the main docs

### Concept canon
- `GOALRAIL_PRODUCT_CONCEPT.md` — канонический ответ на вопрос, что такое Goalrail как продукт и какую проблему он решает
- `GOALRAIL_OPERATING_MODEL.md` — core operating flow: как входящая задача становится contract -> execution -> verify -> proof
- `GOALRAIL_DEPLOYMENT_MODEL.md` — как Goalrail подключается к команде и раскатывается как productized operating layer
- `GOALRAIL_PILOT_MODEL.md` — как выглядит первый pilot engagement, что входит, что не входит и какой результат считается успехом
- `GOALRAIL_GTM_MODEL.md` — как Goalrail продаётся, через какой motion заходит в компанию и какой CTA используется
- `GOALRAIL_ICP.md` — целевые команды, qualification logic и no-fit cases
- `GOALRAIL_OFFER.md` — текущее sellable package: free qualification, paid pilot, outputs, non-goals и expansion path

### Product summary
- `GOALRAIL_PRODUCT_BRIEF.md` — короткая executive-версия продукта
- `GOALRAIL_ONE_PAGER.md` — краткая summary-версия для внутреннего и внешнего использования
- `GOALRAIL_PAIN_STATEMENT.md` — pain framing и market case
- `GOALRAIL_MESSAGE_HIERARCHY.md` — message stack и public wording

### Product shape
- `GOALRAIL_DESIGN_DECISIONS.md` — public entry flow и главные UX-решения
- `GOALRAIL_LANDING_COPY.md` — first landing / entry scene copy
- `GOALRAIL_PROVIDER_BOUNDARIES.md` — что строим, что оборачиваем, где не конкурируем
- `GOALRAIL_COMPETITOR_MAP.md` — reference market map
- `GOALRAIL_REFERENCE_DECISION.md` — внешний reference posture

### Architecture canon
- `GOALRAIL_MVP_BLUEPRINT.md` — перевод концепта в продуктовые слои и архитектурные границы
- `PROJECT_SPINE_SCHEMA.md` — канонические объекты и границы истины
- `GOALRAIL_PARALLEL_EXECUTION_MODEL.md` — writable execution vs advisory parallelism
- `ADR-0001` — runtime-neutral CLI-first boundary
- `ADR-0002` — single-writer and advisory-panels boundary

### Delivery and build
- `GOALRAIL_BUILD_ROADMAP.md` — очередность фаз и checkpoints
- `GOALRAIL_IMPLEMENTATION_GUIDE.md` — правила bounded implementation
- `STATUS.md` — текущее состояние
- `NEXT.md` — ближайшие bounded slices
- `DECISIONS.md` — компактный decision log
- `COMPONENTS.yaml` — component map

## Source-of-truth priority

1. Concept canon
2. Product summary and market framing
3. Product shape
4. Architecture canon
5. Delivery and build
6. Research / reference
7. Chat context

## Working rules

1. Сначала обновляется concept canon.
2. Затем при необходимости обновляются summary / GTM / landing docs.
3. Затем обновляется architecture canon.
4. Только после этого меняются roadmap, implementation и ops docs.
5. Нельзя менять implementation scope без сверки с concept canon.
6. Goalrail продаётся и проектируется как productized operating layer, а не как bespoke consulting per company.
7. Company-specific differences должны покрываться profile / policy / adapter / template настройками, а не пересборкой ядра процесса.

## Current top-level thesis

Goalrail is:

**от бизнес-цели до проверенного изменения в коде**
