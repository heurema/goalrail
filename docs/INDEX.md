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
  - docs/product/GOALRAIL_PROJECT_SCAN_AND_CONTEXT_PACK_V0.md
  - docs/product/GOALRAIL_DOC_GOVERNANCE.md
  - docs/adr/ADR-0025-repository-baseline-profile-lifecycle.md
  - docs/ops/STATUS.md
  - docs/ops/INIT_LIFECYCLE.md
  - docs/ops/SNAPSHOT_SCAN_SHARED_SHAPE.md
  - docs/ops/INIT_STABILIZATION_CHECKPOINT.md
  - docs/ops/CONSOLE_MAIN_DEPLOYMENT_WIRING.md
  - docs/ops/CONSOLE_RU_DEPLOYMENT_WIRING.md
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
23. `docs/product/GOALRAIL_GLOBAL_START_ASSISTANT.md`
24. `docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md`
25. `docs/product/GOALRAIL_LANDING_COPY.md` — historical / superseded technical draft; not current public landing canon
26. `docs/product/GOALRAIL_PROVIDER_BOUNDARIES.md`
27. `docs/product/GOALRAIL_COMPETITOR_MAP.md`
28. `docs/product/GOALRAIL_REFERENCE_DECISION.md`

### 6. Architecture canon
29. `docs/product/GOALRAIL_MVP_BLUEPRINT.md`
30. `docs/product/GOALRAIL_PROJECT_SCAN_AND_CONTEXT_PACK_V0.md`
31. `docs/PROJECT_SPINE_SCHEMA.md`
32. `docs/product/GOALRAIL_PARALLEL_EXECUTION_MODEL.md`
33. `docs/adr/ADR-0001-runtime-neutral-cli-first.md`
34. `docs/adr/ADR-0002-single-writer-and-advisory-panels.md`
35. `docs/adr/ADR-0003-go-cli-layout.md`
36. `docs/adr/ADR-0004-go-server-boundary-and-selected-stack.md`
37. `docs/adr/ADR-0005-intake-to-goal-promotion-boundary.md`
38. `docs/adr/ADR-0006-goal-clarification-readiness-boundary.md`
39. `docs/adr/ADR-0007-clarification-request-boundary.md`
40. `docs/adr/ADR-0008-runner-checkout-boundary.md`
41. `docs/adr/ADR-0009-clarification-answer-boundary.md`
42. `docs/adr/ADR-0010-organization-project-repo-binding-persistence-boundary.md`
43. `docs/adr/ADR-0011-answer-application-to-goal-hints-boundary.md`
44. `docs/adr/ADR-0012-explicit-readiness-recheck-after-applied-answers.md`
45. `docs/adr/ADR-0013-contract-seed-boundary.md`
46. `docs/adr/ADR-0014-contract-draft-boundary.md`
47. `docs/adr/ADR-0015-contract-draft-review-update-boundary.md`
48. `docs/adr/ADR-0016-contract-draft-ready-for-approval-boundary.md`
49. `docs/adr/ADR-0017-contract-approval-boundary.md`
50. `docs/adr/ADR-0018-workitem-planning-boundary.md`
51. `docs/adr/ADR-0019-workitem-planning-controller-runner-boundary.md`
52. `docs/adr/ADR-0020-public-contract-identity-boundary.md`
53. `docs/adr/ADR-0021-workitem-plan-pull-lease-boundary.md`
54. `docs/adr/ADR-0024-minimal-planning-worker-loop-boundary.md`
55. `docs/adr/ADR-0022-installation-boundary.md`
56. `docs/adr/ADR-0023-user-bootstrap-auth-and-cli-login-boundary.md`
57. `docs/adr/ADR-0025-repository-baseline-profile-lifecycle.md`
58. `docs/adr/ADR-0026-agent-driven-pull-loop-protocol.md`
59. `docs/adr/ADR-0027-organization-user-management-boundary.md`
60. `docs/adr/ADR-0028-runner-checkout-instruction-receipt-boundary.md`
61. `docs/adr/ADR-0029-run-execution-receipt-boundary.md`
62. `docs/adr/ADR-0030-bounded-command-execution-boundary.md`
63. `docs/adr/ADR-0031-project-command-execution-boundary.md`

### 7. Governance and change control
64. `docs/product/GOALRAIL_RESEARCH_GATE.md`
65. `docs/product/GOALRAIL_RESEARCH_INTAKE.md`
66. `docs/product/GOALRAIL_DOC_GOVERNANCE.md`
67. `docs/product/GOALRAIL_RULE_STACK.md`

### 8. Delivery, build, and pilot operations
68. `docs/product/GOALRAIL_BUILD_ROADMAP.md`
69. `docs/product/GOALRAIL_IMPLEMENTATION_GUIDE.md`
70. `docs/ops/STATUS.md`
71. `docs/ops/NEXT.md`
72. `docs/ops/DECISIONS.md`
73. `docs/ops/COMPONENTS.yaml`
74. `docs/ops/INIT_LIFECYCLE.md`
75. `docs/ops/SNAPSHOT_SCAN_SHARED_SHAPE.md`
76. `docs/ops/INIT_STABILIZATION_CHECKPOINT.md`
77. `docs/ops/BRANCH_PROTECTION.md`
78. `docs/ops/REPO_STRUCTURE.md`
79. `docs/ops/GO_CODE_GUIDE.md`
80. `docs/ops/CONSOLE_MAIN_DEPLOYMENT_WIRING.md`
81. `docs/ops/CONSOLE_RU_DEPLOYMENT_WIRING.md`
82. `docs/ops/START_ASSISTANT_IMPLEMENTATION_PLAN.md`
83. `docs/ops/START_ASSISTANT_WORKER_ARCHITECTURE.md`
84. `docs/ops/START_ASSISTANT_PUBLIC_KB_PIPELINE.md`
85. `docs/ops/START_ASSISTANT_STAGE_3B_PLAN.md`
86. `docs/ops/START_ASSISTANT_LIVE_RUNBOOK.md`
87. `docs/ops/START_ASSISTANT_KNOWLEDGE_SYNC.md`
88. `docs/ops/START_ASSISTANT_SECURITY_AND_PRIVACY.md`
89. `docs/ops/START_ASSISTANT_API_CONTRACT.md`
90. `docs/ops/DECISION_LOG_START_ASSISTANT_WORKER_SNIPPET.md`
91. `docs/ops/DECISION_LOG_START_ASSISTANT_SNIPPET.md`
92. `docs/product/GOALRAIL_PILOT_PROPOSAL_TEMPLATE.md`
93. `docs/product/GOALRAIL_QUALIFICATION_CHECKLIST.md`

### 9. Advisory research, reference material, and overlay working surfaces
94. `docs/research/GOALRAIL_ADJACENT_EXPERIMENTS_SYNTHESIS.md`
95. `docs/research/GOALRAIL_AI_SDLC_DISCOVERY_WORKSHOP.md`
96. `docs/reference/design/reference_screens/`
97. `docs/reference/start-assistant/`
98. `.goalrail/work/`
99. `.goalrail/knowledge/`
100. `.goalrail/public-kb/manifest.yaml`
101. `.punk/publishing.toml`
102. `.goalrail/flows/`
103. `.goalrail/evals/`
104. `docs/ops/PUBLISHING_MIGRATION.md`
105. `docs/ops/PUBLISHING_RESOLVER_CONTRACT.md`


## Roles of the main docs

### Concept canon
- `GOALRAIL_PRODUCT_CONCEPT.md` — канонический ответ на вопрос, что такое Goalrail как продукт и какую проблему он решает
- `GOALRAIL_OPERATING_MODEL.md` — core operating flow: как входящая задача становится contract -> execution -> verify -> proof
- `GOALRAIL_DEPLOYMENT_MODEL.md` — как Goalrail подключается к команде и раскатывается как productized operating layer
- `GOALRAIL_PILOT_MODEL.md` — как выглядит первый pilot engagement, что входит, что не входит и какой результат считается успехом
- `GOALRAIL_GTM_MODEL.md` — как Goalrail продается, через какой motion заходит в компанию и какой CTA используется
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
- `GOALRAIL_GLOBAL_START_ASSISTANT.md` — draft global `/start` assistant surface: English-first guided entry page, public knowledge boundary, static-first then source-grounded assistant sequence, and no repo scan / code execution posture
- `GOALRAIL_LANDING_COPY_PILOT_FIRST.md` — current public RU pilot landing canon and copy/governance reference for the business-first landing (`ИИ-кодинг без хаоса`); implementation lives at `apps/web/pilot-intake-ru`; D-0055 makes this the primary public RU landing and demotes the previous technical interactive walkthrough to internal/checkpoint status; D-0047 boundaries remain except for the narrow D-0056 lead-capture endpoint, D-0058 server-local daily digest, and D-0059 Resend HTTPS mail transport (no analytics, no LLM, no repo connection, no execution, no persistence beyond local JSONL lead log)
- `GOALRAIL_LANDING_COPY.md` — historical / superseded technical prompt-handoff landing draft; not current public landing canon and not the source of truth for `apps/web/pilot-intake-ru`
- `GOALRAIL_PROVIDER_BOUNDARIES.md` — что строим, что оборачиваем, где не конкурируем
- `GOALRAIL_COMPETITOR_MAP.md` — reference market map
- `GOALRAIL_REFERENCE_DECISION.md` — внешний reference posture

### Architecture canon
- `GOALRAIL_MVP_BLUEPRINT.md` — перевод концепта в продуктовые слои и архитектурные границы
- `GOALRAIL_PROJECT_SCAN_AND_CONTEXT_PACK_V0.md` — Project Scan v0, immutable `RepositoryBaselineProfile`, `WorkspaceOverlay`, and task-specific `ContractContextPack` freshness model
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
- `ADR-0019` — WorkItem planning controller / runner boundary
- `ADR-0020` — Public Contract identity boundary
- `ADR-0021` — WorkItemPlan pull lease boundary; typed planning queue and future lease protocol direction
- `ADR-0024` — Minimal planning worker loop boundary; first thin
  `goalrail-worker` prototype polls typed plan leases, without runner,
  checkout, execution, direct DB writes, or WorkItem creation by the worker
- `ADR-0022` — Installation boundary; running control-plane instance above
  Organization, with `self_hosted` and `saas` as the only deployment modes
- `ADR-0023` — user bootstrap, auth, and CLI login boundary; self-hosted
  bootstrapped owner, admin-created users, token direction, and browser
  loopback `goalrail login`
- `ADR-0025` — repository baseline profile lifecycle; local Project Scan,
  immutable committed-state baseline, separate workspace overlay, and
  task-specific context packs without server-side clone or checks
- `ADR-0026` — agent-driven pull-loop protocol through Goalrail CLI;
  server-owned canonical state, CLI as local repo/transport bridge, local agent
  as UX layer, and Agent Pack as bootstrap guidance
- `ADR-0027` — Organization user management boundary; future
  Console-backed server API, owner-only v0 authorization, canonical `User` plus
  `OrganizationMembership`, one-time temporary password handling, and no CLI
  user creation
- `ADR-0028` — runner checkout instruction and workspace receipt boundary;
  concrete H1 target for `WorkItem(planned)` to checkout preparation without
  assignment, execution, `Run`, gate, proof, server-side clone, or server-side
  repository secrets
- `ADR-0029` — Run and execution receipt boundary; H2 direction for
  `ExecutionJob` as the leaseable unit, `Run` creation only on runner start
  with lease proof, and execution receipts as evidence inputs rather than gate
  verdicts or proof
- `ADR-0030` — bounded command execution boundary; H2.4 direction for
  server-issued command plans, runner-only built-in diagnostic execution first,
  no arbitrary shell, and command metadata receipts as evidence inputs rather
  than gate verdicts or proof
- `ADR-0031` — project command execution boundary; H2.5 direction for typed,
  allowlisted project probes only, no shell, no arbitrary command strings, no
  user-provided argv, one command receipt per Run, and project command
  receipts as evidence inputs rather than gate verdicts or proof

### Governance and change control
- `GOALRAIL_RESEARCH_GATE.md` — когда обязателен research перед изменением product / architecture / governance / public-claim boundaries
- `GOALRAIL_RESEARCH_INTAKE.md` — как adjacent/external ideas классифицируются без roadmap sprawl
- `GOALRAIL_DOC_GOVERNANCE.md` — truth model, metadata vocabulary, lifecycle rules и staged deterministic enforcement posture
- `GOALRAIL_RULE_STACK.md` — rule precedence, dogfooding law, component/slice/пул-реквест hierarchy, and non-override behavior

### Delivery, build, and pilot operations
- `GOALRAIL_BUILD_ROADMAP.md` — очередность фаз и checkpoints
- `GOALRAIL_IMPLEMENTATION_GUIDE.md` — правила bounded implementation
- `STATUS.md` — текущее состояние
- `NEXT.md` — ближайшие bounded slices
- `DECISIONS.md` — компактный decision log
- `COMPONENTS.yaml` — component map
- `INIT_LIFECYCLE.md` — operational design note for current `goalrail init`
  modes, local marker / snapshot / Project Scan distinctions, trust boundary,
  and MVP partial-failure recovery direction
- `SNAPSHOT_SCAN_SHARED_SHAPE.md` — operational direction for reducing drift
  between repository context snapshot inventory and local Project Scan baseline
- `INIT_STABILIZATION_CHECKPOINT.md` — operational checkpoint for completed
  `goalrail init` stabilization slices, remaining risks, non-goals, and the
  next safe init follow-up options
- `BRANCH_PROTECTION.md` — operational record for verified GitHub `main` branch protection and required PR check contexts
- `REPO_STRUCTURE.md` — operational map for where code, docs, tools, overlays, and root-level files belong
- `GO_CODE_GUIDE.md` — repo-wide Go coding rules for keeping future Go work consistent with the current architecture and style
- `CONSOLE_MAIN_DEPLOYMENT_WIRING.md` — operational record for the main `goalrail.dev` console and `api.goalrail.dev` API Flux GitOps deployment and smoke status
- `CONSOLE_RU_DEPLOYMENT_WIRING.md` — operational record for the static `console.goalrail.ru` deployment wiring and smoke status
- `START_ASSISTANT_IMPLEMENTATION_PLAN.md` — staged implementation plan for `/start`, from static page to source-grounded assistant
- `START_ASSISTANT_WORKER_ARCHITECTURE.md` — Stage 3A architecture for the separate public-edge assistant Worker and `/api/start-chat` ownership boundary
- `START_ASSISTANT_PUBLIC_KB_PIPELINE.md` — public KB source whitelist, build process, vector store lifecycle, manifest storage, freshness, and rollback rules
- `START_ASSISTANT_STAGE_3B_PLAN.md` — smallest live Worker implementation plan, tests, smoke checks, security validation, and non-goals
- `START_ASSISTANT_LIVE_RUNBOOK.md` — live `/start` route ownership, Worker deploy/smoke commands, rollback path, and remaining limits
- `START_ASSISTANT_KNOWLEDGE_SYNC.md` — initial public knowledge sync policy for compiling whitelisted docs into retrieval artifacts
- `START_ASSISTANT_SECURITY_AND_PRIVACY.md` — public assistant safety boundary, input policy, logging posture, abuse controls, and safe refusals
- `START_ASSISTANT_API_CONTRACT.md` — `POST /api/start-chat` request/response contract and system-instruction draft
- `DECISION_LOG_START_ASSISTANT_WORKER_SNIPPET.md` — proposed decision snippet for the separate public-edge Worker boundary
- `DECISION_LOG_START_ASSISTANT_SNIPPET.md` — proposed decision snippet for the global `/start` assistant surface
- `GOALRAIL_PILOT_PROPOSAL_TEMPLATE.md` — draft operational template для post-qualification pilot proposal; client-facing working copy, не product canon
- `GOALRAIL_QUALIFICATION_CHECKLIST.md` — draft founder-facing fit-check checklist для короткого qualification call; operational screen, не stabilised sales process

### Advisory research, reference material, and overlay working surfaces
- `docs/research/GOALRAIL_ADJACENT_EXPERIMENTS_SYNTHESIS.md` — advisory synthesis of adjacent experiments such as Punk; useful for intake and anti-pattern extraction, but not canonical product truth
- `docs/research/GOALRAIL_AI_SDLC_DISCOVERY_WORKSHOP.md` — advisory discovery workshop synthesis on AI-SDLC pain, validation, pilot candidates, and proof-oriented delivery; discussion input, not product canon
- `docs/reference/design/reference_screens/` — visual reference material without product-truth authority
- `docs/reference/start-assistant/` — static quick questions and answer source material for the `/start` guided assistant surface
- `.goalrail/work/` — Goalrail-tracked goals, reports, and bounded slice memory
- `.goalrail/knowledge/` — Goalrail advisory research and idea backlog; не источник канона без promotion
- `.goalrail/public-kb/manifest.yaml` — explicit public KB whitelist input for the future source-grounded `/start` assistant
- `.punk/publishing.toml` — committed publishing binding manifest; runtime workspace is external and resolved via CLI
- legacy `.punk/publishing/` was removed; see `docs/ops/PUBLISHING_MIGRATION.md`
- `.goalrail/flows/` — planned flow/spec boundary for future runtime semantics
- `.goalrail/evals/` — planned eval/spec boundary for future verification semantics
- `docs/ops/PUBLISHING_MIGRATION.md` — planning for legacy Punk workspace migration
- `docs/ops/PUBLISHING_RESOLVER_CONTRACT.md` — machine contract for external publishing workspace resolution

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
11. Goalrail продается и проектируется как productized operating layer, а не как bespoke consulting per company.
12. Company-specific differences должны покрываться profile / policy / adapter / template настройками, а не пересборкой ядра процесса.
13. Overlay planning surfaces (`.goalrail/work/`, `.goalrail/knowledge/`, `.punk/publishing.toml`, `.goalrail/flows/`, `.goalrail/evals/`) поддерживают работу и исследования, но не переопределяют канонические docs.

## Current top-level thesis

Goalrail is:

**от бизнес-цели до проверенного изменения в коде**
