---
id: goalrail_docs_index
title: Goalrail Docs Index
kind: reference
authority: reference
status: current
owner: docs-governance
truth_surfaces:
  - read_order
  - source_priority_view
lifecycle: active-core
review_after: 2026-07-19
supersedes: []
superseded_by: null
related_docs:
  - docs/product/GOALRAIL_PRODUCT_CONCEPT.md
  - docs/product/GOALRAIL_DOC_GOVERNANCE.md
  - docs/ops/STATUS.md
---
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
8. `docs/product/GOALRAIL_PRICING_MODEL.md`

### 2. Product summary and market framing
9. `docs/product/GOALRAIL_PRODUCT_BRIEF.md`
10. `docs/product/GOALRAIL_ONE_PAGER.md`
11. `docs/product/GOALRAIL_PAIN_STATEMENT.md`
12. `docs/product/GOALRAIL_MESSAGE_HIERARCHY.md`

### 3. Public narrative and external language
13. `docs/product/GOALRAIL_PUBLIC_NARRATIVE.md`
14. `docs/product/GOALRAIL_PUBLIC_LANGUAGE.md`

### 4. Brand and mascot layer
15. `docs/brand/INDEX.md`
16. `docs/brand/PUNK_CHARACTER_SYSTEM.md`
17. `docs/brand/SHORT_FORM_CONTENT_SYSTEM.md`
18. `docs/brand/VISUAL_IDENTITY_V0.md`
19. `docs/brand/TITLE_CARD_AND_THUMBNAIL_TEMPLATES.md`
20. `docs/brand/MASCOT_ASSET_RULES.md`
21. `docs/brand/MOTION_RULES_V0.md`

### 5. Product shape and external posture
22. `docs/product/GOALRAIL_DESIGN_DECISIONS.md`
23. `docs/product/GOALRAIL_LANDING_COPY.md`
24. `docs/product/GOALRAIL_PROVIDER_BOUNDARIES.md`
25. `docs/product/GOALRAIL_COMPETITOR_MAP.md`
26. `docs/product/GOALRAIL_REFERENCE_DECISION.md`

### 6. Architecture canon
27. `docs/product/GOALRAIL_MVP_BLUEPRINT.md`
28. `docs/PROJECT_SPINE_SCHEMA.md`
29. `docs/product/GOALRAIL_PARALLEL_EXECUTION_MODEL.md`
30. `docs/adr/ADR-0001-runtime-neutral-cli-first.md`
31. `docs/adr/ADR-0002-single-writer-and-advisory-panels.md`
32. `docs/adr/ADR-0003-go-cli-layout.md`
33. `docs/adr/ADR-0004-go-server-boundary-and-selected-stack.md`
34. `docs/adr/ADR-0005-intake-to-goal-promotion-boundary.md`
35. `docs/adr/ADR-0006-goal-clarification-readiness-boundary.md`
36. `docs/adr/ADR-0007-clarification-request-boundary.md`
37. `docs/adr/ADR-0008-runner-checkout-boundary.md`
38. `docs/adr/ADR-0009-clarification-answer-boundary.md`
39. `docs/adr/ADR-0010-organization-project-repo-binding-persistence-boundary.md`
40. `docs/adr/ADR-0011-answer-application-to-goal-hints-boundary.md`
41. `docs/adr/ADR-0012-explicit-readiness-recheck-after-applied-answers.md`
42. `docs/adr/ADR-0013-contract-seed-boundary.md`
43. `docs/adr/ADR-0014-contract-draft-boundary.md`
44. `docs/adr/ADR-0015-contract-draft-review-update-boundary.md`
45. `docs/adr/ADR-0016-contract-draft-ready-for-approval-boundary.md`
46. `docs/adr/ADR-0017-contract-approval-boundary.md`
47. `docs/adr/ADR-0018-workitem-planning-boundary.md`

### 7. Governance and change control
48. `docs/product/GOALRAIL_RESEARCH_GATE.md`
49. `docs/product/GOALRAIL_RESEARCH_INTAKE.md`
50. `docs/product/GOALRAIL_DOC_GOVERNANCE.md`
51. `docs/product/GOALRAIL_RULE_STACK.md`

### 8. Delivery, build, and pilot operations
52. `docs/product/GOALRAIL_BUILD_ROADMAP.md`
53. `docs/product/GOALRAIL_IMPLEMENTATION_GUIDE.md`
54. `docs/ops/STATUS.md`
55. `docs/ops/NEXT.md`
56. `docs/ops/DECISIONS.md`
57. `docs/ops/COMPONENTS.yaml`
58. `docs/ops/REPO_STRUCTURE.md`
59. `docs/product/GOALRAIL_PILOT_PROPOSAL_TEMPLATE.md`
60. `docs/product/GOALRAIL_QUALIFICATION_CHECKLIST.md`

### 9. Advisory research, reference material, and overlay working surfaces
61. `docs/research/GOALRAIL_ADJACENT_EXPERIMENTS_SYNTHESIS.md`
62. `docs/research/GOALRAIL_AI_SDLC_DISCOVERY_WORKSHOP.md`
63. `docs/reference/design/reference_screens/`
64. `.goalrail/work/`
65. `.goalrail/knowledge/`
66. `.punk/publishing/`
67. `.goalrail/flows/`
68. `.goalrail/evals/`


## Roles of the main docs

### Concept canon
- `GOALRAIL_PRODUCT_CONCEPT.md` — канонический ответ на вопрос, что такое Goalrail как продукт и какую проблему он решает
- `GOALRAIL_OPERATING_MODEL.md` — core operating flow: как входящая задача становится contract -> execution -> verify -> proof
- `GOALRAIL_DEPLOYMENT_MODEL.md` — как Goalrail подключается к команде и раскатывается как productized operating layer
- `GOALRAIL_PILOT_MODEL.md` — как выглядит первый pilot engagement, что входит, что не входит и какой результат считается успехом
- `GOALRAIL_GTM_MODEL.md` — как Goalrail продаётся, через какой motion заходит в компанию и какой CTA используется
- `GOALRAIL_ICP.md` — целевые команды, qualification logic и no-fit cases
- `GOALRAIL_OFFER.md` — текущее sellable package: free qualification, paid pilot, outputs, non-goals и expansion path
- `GOALRAIL_PRICING_MODEL.md` — текущая pricing and packaging logic: USD pilot anchor (`Managed pilot — from $5,000`), internal pilot ranges, design-partner discount mode, modifiers and post-pilot retainers

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
- `ADR-0003` — Go CLI layout and canonical binary boundary
- `ADR-0004` — Go server boundary and selected stack
- `ADR-0005` — IntakeRecord to Goal promotion boundary
- `ADR-0006` — Goal clarification readiness boundary
- `ADR-0007` — Clarification request boundary
- `ADR-0008` — Runner and repository checkout boundary
- `ADR-0009` — Clarification answer boundary
- `ADR-0010` — Organization, project, repo binding, and persistence bootstrap boundary
- `ADR-0011` — Answer application to Goal hints boundary
- `ADR-0012` — Explicit readiness re-check after applied answers boundary
- `ADR-0013` — ContractSeed boundary
- `ADR-0014` — ContractDraft boundary
- `ADR-0015` — ContractDraft review/update boundary
- `ADR-0016` — ContractDraft ready_for_approval boundary
- `ADR-0017` — Contract approval boundary
- `ADR-0018` — WorkItem planning boundary

### Governance and change control
- `GOALRAIL_RESEARCH_GATE.md` — когда обязателен research перед изменением product / architecture / governance / public-claim boundaries
- `GOALRAIL_RESEARCH_INTAKE.md` — как adjacent/external ideas классифицируются без roadmap sprawl
- `GOALRAIL_DOC_GOVERNANCE.md` — truth model, metadata vocabulary, lifecycle rules и staged deterministic enforcement posture
- `GOALRAIL_RULE_STACK.md` — rule precedence, dogfooding law, component/slice/PR hierarchy, and non-override behavior

### Delivery, build, and pilot operations
- `GOALRAIL_BUILD_ROADMAP.md` — очередность фаз и checkpoints
- `GOALRAIL_IMPLEMENTATION_GUIDE.md` — правила bounded implementation
- `STATUS.md` — текущее состояние
- `NEXT.md` — ближайшие bounded slices
- `DECISIONS.md` — компактный decision log
- `COMPONENTS.yaml` — component map
- `REPO_STRUCTURE.md` — operational map for where code, docs, tools, overlays, and root-level files belong
- `GOALRAIL_PILOT_PROPOSAL_TEMPLATE.md` — draft operational template для post-qualification pilot proposal; client-facing working copy, не product canon
- `GOALRAIL_QUALIFICATION_CHECKLIST.md` — draft founder-facing fit-check checklist для короткого qualification call; operational screen, не stabilised sales process

### Advisory research, reference material, and overlay working surfaces
- `docs/research/GOALRAIL_ADJACENT_EXPERIMENTS_SYNTHESIS.md` — advisory synthesis of adjacent experiments such as Punk; useful for intake and anti-pattern extraction, but not canonical product truth
- `docs/research/GOALRAIL_AI_SDLC_DISCOVERY_WORKSHOP.md` — advisory discovery workshop synthesis on AI-SDLC pain, validation, pilot candidates, and proof-oriented delivery; discussion input, not product canon
- `docs/reference/design/reference_screens/` — visual reference material without product-truth authority
- `.goalrail/work/` — Goalrail-tracked goals, reports, and bounded slice memory
- `.goalrail/knowledge/` — Goalrail advisory research and idea backlog; не источник канона без promotion
- `.punk/publishing/` — Punk-owned public narrative drafts, receipts, and manual metrics
- `.goalrail/flows/` — planned flow/spec boundary for future runtime semantics
- `.goalrail/evals/` — planned eval/spec boundary for future verification semantics

## Source-of-truth priority

1. Concept canon
2. Product summary and market framing
3. Public narrative and external language
4. Brand and mascot layer
5. Product shape and external posture
6. Architecture canon
7. Governance and change control
8. Delivery, build, and pilot operations
9. Advisory research, reference material, and overlay working surfaces
10. Chat context

Notes:
- Governance and change-control docs govern process, metadata, and enforcement posture; they do not override product canon.
- Advisory research, reference material, and overlay working surfaces may inform changes, but canonical product and architecture docs still win.

## Working rules

1. Сначала обновляется concept canon.
2. Затем при необходимости обновляются summary / GTM docs.
3. Если меняется public story, campaign framing или внешний vocabulary, обновляются `GOALRAIL_PUBLIC_NARRATIVE.md` и `GOALRAIL_PUBLIC_LANGUAGE.md`.
4. Если меняется mascot role, character tone, short-form grammar, visual draft, template system, mascot asset rules, motion rules или brand-carrier rules, обновляется brand layer.
5. Затем обновляются landing / design / product-shape docs.
6. Затем обновляется architecture canon.
7. После architecture canon сверяются research / doc governance rules, если меняется truth model, public claims или core boundaries.
8. Только после этого меняются roadmap, implementation и ops docs.
9. Adjacent experiments и research synthesis имеют advisory-статус и не могут переопределять canonical docs без Goalrail intake / promotion path.
10. Нельзя менять implementation scope без сверки с concept canon.
11. Goalrail продаётся и проектируется как productized operating layer, а не как bespoke consulting per company.
12. Company-specific differences должны покрываться profile / policy / adapter / template настройками, а не пересборкой ядра процесса.
13. Overlay planning surfaces (`.goalrail/work/`, `.goalrail/knowledge/`, `.punk/publishing/`, `.goalrail/flows/`, `.goalrail/evals/`) поддерживают работу и исследования, но не переопределяют канонические docs.

## Current top-level thesis

Goalrail is:

**от бизнес-цели до проверенного изменения в коде**
