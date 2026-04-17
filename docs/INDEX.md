# Goalrail Docs Index

> Единая точка входа в проект.
> Сначала читаем бизнесовый канон, затем deployment и pilot-модель, затем market framing,
> затем public narrative и внешний язык, затем brand/mascot layer, затем product shape,
> архитектуру и только потом build / ops.

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

### 4. Brand and mascot layer
14. `docs/brand/INDEX.md`
15. `docs/brand/PUNK_CHARACTER_SYSTEM.md`
16. `docs/brand/SHORT_FORM_CONTENT_SYSTEM.md`
17. `docs/brand/VISUAL_IDENTITY_V0.md`
18. `docs/brand/TITLE_CARD_AND_THUMBNAIL_TEMPLATES.md`
19. `docs/brand/MASCOT_ASSET_RULES.md`
20. `docs/brand/MOTION_RULES_V0.md`

### 5. Product shape and external posture
21. `docs/product/GOALRAIL_DESIGN_DECISIONS.md`
22. `docs/product/GOALRAIL_LANDING_COPY.md`
23. `docs/product/GOALRAIL_PROVIDER_BOUNDARIES.md`
24. `docs/product/GOALRAIL_COMPETITOR_MAP.md`
25. `docs/product/GOALRAIL_REFERENCE_DECISION.md`

### 6. Architecture canon
26. `docs/product/GOALRAIL_MVP_BLUEPRINT.md`
27. `docs/PROJECT_SPINE_SCHEMA.md`
28. `docs/product/GOALRAIL_PARALLEL_EXECUTION_MODEL.md`
29. `docs/adr/ADR-0001-runtime-neutral-cli-first.md`
30. `docs/adr/ADR-0002-single-writer-and-advisory-panels.md`

### 7. Delivery and build
31. `docs/product/GOALRAIL_BUILD_ROADMAP.md`
32. `docs/product/GOALRAIL_IMPLEMENTATION_GUIDE.md`
33. `docs/ops/STATUS.md`
34. `docs/ops/NEXT.md`
35. `docs/ops/DECISIONS.md`
36. `docs/ops/COMPONENTS.yaml`

### 8. Research and source material
37. `docs/research/AI_SDLC_RUST_PRODUCT_SUMMARY_SOURCE.md`
38. `design/reference_screens/`

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
- `GOALRAIL_PUBLIC_LANGUAGE.md` — translation layer между internal runtime terms и внешним business-facing wording; naming, phrase and disclaimer rules

### Brand and mascot layer
- `docs/brand/INDEX.md` — ownership boundary между brand layer и product docs
- `docs/brand/PUNK_CHARACTER_SYSTEM.md` — working character canon for Punk as mascot / on-screen brand carrier
- `docs/brand/SHORT_FORM_CONTENT_SYSTEM.md` — working grammar for short videos, carousels, captions, subtitles and CTA structure
- `docs/brand/VISUAL_IDENTITY_V0.md` — visual system draft with tonal modes, starter motifs, title-card and thumbnail rules
- `docs/brand/TITLE_CARD_AND_THUMBNAIL_TEMPLATES.md` — repeatable production templates for title cards and thumbnails across two tonal modes
- `docs/brand/MASCOT_ASSET_RULES.md` — production-safe mascot asset rules for framing, intensity, pose families, and product-first coexistence
- `docs/brand/MOTION_RULES_V0.md` — motion behavior rules for Punk, rails, signals, text, subtitles, transitions and timing across two tonal modes

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
4. Brand and mascot layer
5. Product shape and external posture
6. Architecture canon
7. Delivery and build
8. Research / reference
9. Chat context

## Working rules

1. Сначала обновляется concept canon.
2. Затем при необходимости обновляются summary / GTM docs.
3. Если меняется public story, campaign framing или внешний vocabulary, обновляются `GOALRAIL_PUBLIC_NARRATIVE.md` и `GOALRAIL_PUBLIC_LANGUAGE.md`.
4. Если меняется mascot role, character tone, short-form grammar, visual draft, template system, mascot asset rules, motion rules или brand-carrier rules, обновляется brand layer.
5. Затем обновляются landing / design / product-shape docs.
6. Затем обновляется architecture canon.
7. Только после этого меняются roadmap, implementation и ops docs.
8. Нельзя менять implementation scope без сверки с concept canon.
9. Goalrail продаётся и проектируется как productized operating layer, а не как bespoke consulting per company.
10. Company-specific differences должны покрываться profile / policy / adapter / template настройками, а не пересборкой ядра процесса.

## Current top-level thesis

Goalrail is:

**от бизнес-цели до проверенного изменения в коде**
