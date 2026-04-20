---
id: goalrail_product_concept
title: Goalrail Product Concept
kind: product_canon
authority: canonical
status: current
owner: product
truth_surfaces:
  - product_concept
  - product_boundaries
  - fixed_core
lifecycle: active-core
review_after: 2026-07-19
supersedes: []
superseded_by: null
related_docs:
  - docs/product/GOALRAIL_OPERATING_MODEL.md
  - docs/product/GOALRAIL_PRODUCT_BRIEF.md
  - docs/product/GOALRAIL_MVP_BLUEPRINT.md
---
# Goalrail Product Concept

## 1. Product thesis

Goalrail — это productized operating layer для AI-assisted delivery в software teams.

Короткая формула:

**от бизнес-цели до проверенного изменения в коде**

Goalrail не пытается заменить IDE, трекеры, code agents или provider-native runtimes.
Он соединяет постановку задачи, управляемое исполнение и проверяемый результат в одном общем контуре.

## 2. Why this product exists

AI ускорил execution, но не решил главную проблему команд:

- задача приходит в виде vague request
- ограничения и non-goals часто не зафиксированы
- инженерный и бизнесовый контекст живут в разных местах
- AI начинает писать код раньше, чем команда согласовала смысл и границы
- на выходе есть patch, diff или PR, но нет единого inspectable объекта, в котором видно:
  - что именно делали
  - в каком scope
  - как это проверили
  - можно ли доверять результату

Goalrail нужен именно для закрытия этого разрыва.

## 3. Product answer

Goalrail даёт команде общий управляемый слой между incoming task и engineering outcome.

Он делает пять вещей:

1. превращает сырой входящий запрос в рабочий контракт
2. делает границы и ограничения явными
3. проводит работу через bounded execution
4. отделяет execution от verification
5. возвращает inspectable proof, а не только статус

## 4. Product category

Goalrail — не “ещё один агент”.
Goalrail — не AI IDE.
Goalrail — не generic workflow engine.

Это:

- shared source of truth for AI-assisted delivery
- contract-first delivery control layer
- verification and proof contour
- bridge between PM / analyst intent and engineering execution

## 5. Core wedge

Устойчивый wedge Goalrail не в том, чтобы быть лучшим агентом или лучшей моделью.

Wedge Goalrail:

- shared working contract
- server-side source of truth
- bounded execution boundary
- verify / proof contour
- business + engineering visibility in one layer

## 6. Product posture

Goalrail должен строиться и продаваться как:

- supplement layer over existing tools
- runtime-neutral by design
- adaptive to provider evolution
- productized operating layer
- pilot-first entry product

Goalrail не должен строиться как:

- fixed monolithic platform truth
- replacement for Jira / Linear
- replacement for provider-native coding agents
- bespoke consulting per customer
- all-in-one DevOps suite

## 7. Two planes

### Plane A — Intent / Planning
Для PM, analyst, product owner, tech lead.

Что происходит:
- goal intake
- clarification
- constraints
- glossary
- acceptance framing
- working contract preparation

### Plane B — Delivery / Execution
Для developer, tech lead, QA, CI / automation.

Что происходит:
- contract review
- task shaping
- bounded execution
- verification
- proof

Обе плоскости соединяются через один Project Spine.

## 8. Canonical flow

`Incoming task -> Clarify -> Working contract -> Tasks -> Run -> Verify -> Proof -> Feedback`

Это главный flow продукта.
Он должен быть понятен бизнесу, инженерии и AI-runtime layer.

## 9. Core objects at business level

- Project — команда и её delivery contour
- Goal — нормализованная бизнес-задача
- Contract — общий рабочий контракт между intent и delivery
- Task — bounded единица исполнения
- Run — один execution attempt
- Decision — итоговая machine-readable оценка результата
- Proof — inspectable acceptance artifact
- Learning — обратная связь в контур

## 10. Fixed core

Следующие вещи считаются фиксированным ядром продукта:

- задача сначала становится working contract
- execution всегда bounded
- final evaluation отделена от execution
- один writable run использует один primary writer runtime
- на выходе есть inspectable proof
- contract / task / run / decision / proof видны как связанный контур

## 11. Configurable knobs

Следующие вещи могут настраиваться под организацию:

- tracker binding
- runtime binding
- review depth
- policy profile
- terminology mapping
- scope templates
- proof strictness
- approval expectations

Настройки не должны ломать fixed core.

## 12. Who this is for

Начальный ICP:

- product teams с 5–30 инженерами
- есть PM / analyst / tech lead структура
- есть давление внедрять AI в delivery без потери контроля
- есть 1–2 репо, на которых можно показать pilot value
- команда готова принять новый operating layer поверх существующего стека

## 13. Who this is not for

Плохой fit на старте:

- команды без реального delivery flow
- команды без sponsor со стороны engineering leadership
- команды, которые хотят “полного автопилота” вместо controlled process
- команды, где любой внешний runtime или managed flow запрещён
- команды, которые ждут полного замещения текущего toolchain

## 14. First honest product promise

Goalrail помогает команде:

- уменьшить ambiguity между PM и dev
- сделать AI-assisted delivery bounded и reviewable
- видеть один и тот же рабочий объект от intent до результата
- получать proof-oriented visibility вместо “кажется, сделали”

## 15. First commercial form

Первый честный коммерческий формат — не broad rollout и не self-serve SaaS.

Правильный вход:

- pilot-first
- one team
- one or two repos
- one visible workflow from incoming task to proof

## 16. Success criteria

Goalrail считается работающим, если команда может:

1. взять реальную задачу
2. собрать по ней working contract
3. провести bounded execution
4. получить verify / proof output
5. принять решение о rollout на основе реального pilot evidence

## 17. Non-goals

На текущем этапе Goalrail не должен:

- строить ещё одну AI IDE
- строить universal agent framework
- заменять Jira / Linear
- становиться giant standalone memory system
- копировать provider-native capabilities, которые уже хорошо закрыты рынком

## 18. One-line summary

**Goalrail — это productized operating layer, который превращает размытый входящий запрос в управляемую и проверяемую инженерную работу.**
