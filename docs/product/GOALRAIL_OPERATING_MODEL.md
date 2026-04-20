---
id: goalrail_operating_model
title: Goalrail Operating Model
kind: product_canon
authority: canonical
status: current
owner: product
truth_surfaces:
  - operating_model
  - contract_first_flow
  - roles
lifecycle: active-core
review_after: 2026-07-19
supersedes: []
superseded_by: null
related_docs:
  - docs/product/GOALRAIL_PRODUCT_CONCEPT.md
  - docs/product/GOALRAIL_MVP_BLUEPRINT.md
  - docs/PROJECT_SPINE_SCHEMA.md
---
# Goalrail Operating Model

## 1. Purpose

Этот документ описывает, как Goalrail работает как operating system для AI-assisted delivery.

Он отвечает не на вопрос “как это реализовано технически”, а на вопрос:

**как команда должна работать через Goalrail.**

## 2. Operating principle

Goalrail вводит единый рабочий контур между business intent и engineering outcome.

Ключевой принцип:

**сначала общий рабочий контракт, потом execution**

Не:
- неясная задача -> prompt -> код -> обсуждение постфактум

А:
- incoming task -> clarification -> contract -> bounded execution -> verify -> proof

## 3. Core flow

### Stage 1 — Incoming task
В систему попадает сырой запрос:
- идея
- задача
- issue
- ticket
- запрос от бизнеса
- engineering request

Вход пока не считается готовым к execution.

### Stage 2 — Clarify
Задача уточняется:
- цель
- контекст
- ограничения
- non-goals
- acceptance intent
- открытые вопросы

Результат: clarified goal.

### Stage 3 — Working contract
Clarified goal превращается в working contract.

Working contract фиксирует:
- что делаем
- зачем делаем
- что входит в scope
- что вне scope
- какие ограничения действуют
- какие проверки ожидаются
- какой результат считается приемлемым

Результат: approved working contract.

### Stage 4 — Task shaping
Contract режется на bounded задачи.

Каждая задача должна иметь:
- чёткий scope
- понятную цель
- expected output
- уровень риска
- execution posture
- verify expectations

Результат: executable bounded tasks.

### Stage 5 — Bounded execution
Одна задача идёт в execution через выбранный runtime.

Правила:
- один writable run = один primary writer runtime
- scope должен быть bounded
- execution packet должен быть понятен и inspectable
- advisory review может существовать отдельно, но не заменяет primary execution lineage

Результат: run artifacts.

### Stage 6 — Verify
Результат execution проходит проверку.

Проверка смотрит на:
- scope
- target
- integrity
- policy
- baseline vs regression
- holdout checks, когда они нужны

Результат: decision.

### Stage 7 — Proof
Decision превращается в proof.

Proof должен отвечать:
- что изменили
- как проверили
- какой verdict
- какие ограничения / риски остаются
- можно ли принимать результат

Результат: inspectable proof.

### Stage 8 — Feedback
Результат возвращается в общий контур.

Feedback может содержать:
- learnings
- changed assumptions
- новые ограничения
- уточнения для следующих задач
- rollout recommendations

## 4. Central object

Главный объект operating model — **working contract**.

Не prompt.
Не ticket.
Не PR.
Не агент.

Именно contract связывает:

- бизнесовую постановку
- инженерные границы
- execution expectations
- verify / proof expectations

## 5. Roles

### Sponsor
Обычно CTO / Head of Engineering / VP Engineering.
Нужен для запуска, policy decisions и rollout approval.

### Intent owner
PM / analyst / product owner / tech lead.
Отвечает за смысл входящей задачи.

### Delivery owner
Tech lead / engineer.
Отвечает за task shaping и execution readiness.

### Runtime operator
Человек или controlled runtime layer, выполняющий bounded work.

### Reviewer / verifier
Проверяет соответствие contract, integrity и policy expectations.

## 6. Operating rules

1. Задача не идёт в execution без минимального contract-level уточнения.
2. Scope должен быть явным.
3. Execution и final verification — разные шаги.
4. Один writable run имеет одного primary writer.
5. Advisory reasoning не является final authority.
6. Proof обязателен как выход системы.
7. Team visibility должна строиться вокруг contract -> task -> run -> decision -> proof chain.

## 7. Fixed surfaces

В любой конфигурации Goalrail должны существовать следующие surfaces:

- task intake surface
- contract surface
- task shaping surface
- execution surface
- verify / proof surface
- team visibility surface

Это логические surfaces; они не обязаны существовать как отдельные страницы в первой версии.

## 8. Configurable parts

Operating model допускает настройки, но только в ограниченном наборе:

- как приходит вход
- где живёт tracker binding
- какой runtime используется
- какой review depth применяется
- какая proof strictness нужна
- как называются сущности в терминологии клиента

## 9. What Goalrail standardizes

Goalrail должен стандартизировать:

- форму handoff между intent и engineering
- структуру рабочего контракта
- минимальный bounded execution packet
- verify / proof output
- decision contour

## 10. What Goalrail does not standardize completely

Goalrail не обязан полностью стандартизировать:

- внутреннюю оргструктуру клиента
- все типы delivery work
- все provider-specific runtime details
- весь CI/CD layer
- все tracker workflows клиента

## 11. Operating output

В нормальной работе команда получает не “кучу AI-activity”, а понятную цепочку:

- incoming task
- clarified goal
- working contract
- bounded task
- run
- decision
- proof

Это и есть основной operational value Goalrail.
