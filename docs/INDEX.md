# Goalrail Docs Index

> Единая точка входа в проект.
> Сначала читаем бизнесовый канон, затем deployment и pilot-модель, затем market framing,
> затем public narrative и внешний язык, затем product shape, архитектуру и только потом build / ops.

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

### 3. Public narrative and external language
12. `docs/product/GOALRAIL_PUBLIC_NARRATIVE.md`
13. `docs/product/GOALRAIL_PUBLIC_LANGUAGE.md`

### 4. Product shape and external posture
14. `docs/product/GOALRAIL_DESIGN_DECISIONS.md`
15. `docs/product/GOALRAIL_LANDING_COPY.md`
16. `docs/product/GOALRAIL_PROVIDER_BOUNDARIES.md`
17. `docs/product/GOALRAIL_COMPETITOR_MAP.md`
18. `docs/product/GOALRAIL_REFERENCE_DECISION.md`

### 5. Architecture canon
19. `docs/product/GOALRAIL_MVP_BLUEPRINT.md`
20. `docs/PROJECT_SPINE_SCHEMA.md`
21. `docs/product/GOALRAIL_PARALLEL_EXECUTION_MODEL.md`
22. `docs/adr/ADR-0001-runtime-neutral-cli-first.md`
23. `docs/adr/ADR-0002-single-writer-and-advisory-panels.md`

### 6. Delivery and build
24. `docs/product/GOALRAIL_BUILD_ROADMAP.md`
25. `docs/product/GOALRAIL_IMPLEMENTATION_GUIDE.md`
26. `docs/ops/STATUS.md`
27. `docs/ops/NEXT.md`
28. `docs/ops/DECISIONS.md`
29. `docs/ops/COMPONENTS.yaml`

### 7. Research and source material
30. `docs/research/AI_SDLC_RUST_PRODUCT_SUMMARY_SOURCE.md`
31. `design/reference_screens/`

## Roles of the main docs

### Concept canon
- `GOALRAIL_PRODUCT_CONCEPT.md` — канонический ответ на вопрос, что такое Goalrail как продукт и какую проблему он решает
- `GOALRAIL_OPERATING_MODEL.md` — core operating flow: как входящая задача становится contract -> execution -> verify -> proof
- `GOALRAIL_DEPLOYMENT_MODEL.md` — как Goalrail подключается к команде и раскатывается как productized operating layer
- `GOALRAIL_PILOT_MODEL.md` — как выглядит первый pilot engagement, что входит, что не входит и какой результат считается успехом
- `GOALRAIL_GTM_MODEL.md` — как Goalrail продаётся, через какой motion заходит в компанию и какой CTA используется
- `GOALRAIL_ICP.md` — целевые команды, qualification logic и no-fit cases
- `GOALRAIL_OFFER.md` — текущее sellable package: free qualification, paid pilot, outputs, non-goals и expansion path

### Product summary and market framing
- `GOALRAIL_PRODUCT_BRIEF.md` — короткая executive-версия продукта
- `GOALRAIL_ONE_PAGER.md` — краткая summary-версия для внутреннего и внешнего использования
- `GOALRAIL_PAIN_STATEMENT.md` — pain framing и market case
- `GOALRAIL_MESSAGE_HIERARCHY.md` — message stack и public wording

### Public narrative and external language
- `GOALRAIL_PUBLIC_NARRATIVE.md` — relationship между Goalrail, Goal on Rails и Specpunk; business-first build-in-public frame; narrative laws
- `GOALRAIL_PUBLIC_LANGUAGE.md` — translation layer между internal runtime terms и внешним business-facing wording; naming и phrase rules

### Product shape and external posture
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
3. Public narrative and external language
4. Product shape and external posture
5. Architecture canon
6. Delivery and build
7. Research / reference
8. Chat context

## Working rules

1. Сначала обновляется concept canon.
2. Затем при необходимости обновляются summary / GTM docs.
3. Если меняется public story, campaign framing или внешний vocabulary, обновляются `GOALRAIL_PUBLIC_NARRATIVE.md` и `GOALRAIL_PUBLIC_LANGUAGE.md`.
4. Затем обновляются landing / design / product-shape docs.
5. Затем обновляется architecture canon.
6. Только после этого меняются roadmap, implementation и ops docs.
7. Нельзя менять implementation scope без сверки с concept canon.
8. Goalrail продаётся и проектируется как productized operating layer, а не как bespoke consulting per company.
9. Company-specific differences должны покрываться profile / policy / adapter / template настройками, а не пересборкой ядра процесса.

## Current top-level thesis

Goalrail is:

**от бизнес-цели до проверенного изменения в коде**
