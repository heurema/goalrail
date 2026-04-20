---
id: goalrail_qualification_checklist
title: Goalrail Qualification Checklist
kind: reference
authority: operational
status: draft
owner: founder-sales
truth_surfaces:
  - qualification_checklist
  - fit_check_template
lifecycle: incubating
review_after: 2026-07-20
supersedes: []
superseded_by: null
related_docs:
  - docs/product/GOALRAIL_ICP.md
  - docs/product/GOALRAIL_OFFER.md
  - docs/product/GOALRAIL_PILOT_MODEL.md
  - docs/product/GOALRAIL_PRICING_MODEL.md
---
# Goalrail Qualification Checklist

> Draft checklist.
> Использовать как рабочую основу для первого qualification / fit-check call.
> Не считать финальным sales process до стабилизации первых pilot loops.

## Purpose

Этот документ нужен, чтобы быстро понять:
- есть ли у клиента реальный fit для Goalrail pilot
- есть ли у нас шанс получить bounded proof-oriented case
- стоит ли двигаться в paid pilot или лучше честно сказать no-go

## Decision rule

Qualification не должен заканчиваться vague “давайте подумаем”.

На выходе должен быть один из трёх результатов:
- **go to pilot proposal**
- **not now / come back later**
- **no-fit**

---

## 1. Account basics

- Company: [ ]
- Team / function: [ ]
- Main contact: [ ]
- Role of main contact: [ ]
- Date: [ ]
- Notes: [ ]

## 2. Sponsor check

### Questions
- Есть ли sponsor со стороны CTO / Head of Engineering / product/tech owner?
- Кто сможет принимать решение по pilot?
- Кто будет защищать pilot внутри команды?

### Good signs
- есть явный owner
- sponsor понимает, зачем нужен pilot
- sponsor готов дойти до decision after readout

### Red flags
- нет owner
- “пусть команда сама посмотрит”
- никто не может принять решение о запуске

### Notes
- Sponsor present: [yes / no / partial]
- Sponsor role: [ ]
- Confidence: [high / medium / low]

## 3. Team fit

### Questions
- Есть ли реальная команда, а не один пользователь?
- Какая структура: PM / analyst / tech lead / developers?
- Размер команды?

### Good signs
- product team 5–30 engineers or small startup team with a real delivery loop
- есть как минимум product/tech counterpart
- команда реально делает delivery, а не только исследует идеи

### Red flags
- один человек без команды
- нет delivery loop
- неясно, кто будет работать с результатом pilot

### Notes
- Team size: [ ]
- Team shape: [ ]
- Fit: [high / medium / low]

## 4. Problem / urgency check

### Questions
- В чём сейчас pain?
- Почему это нужно решать сейчас?
- Что не устраивает в текущем AI-assisted flow?

### Good signs
- ambiguity between task and implementation
- weak visibility into result quality
- AI is already being used, but without enough control
- reviewability / proof gap is visible

### Red flags
- клиент просто “хочет попробовать AI” без реального процесса
- pain слишком общий и не привязан к delivery
- нет urgency

### Notes
- Main pain: [ ]
- Urgency: [high / medium / low]
- Is the pain pilot-testable: [yes / no]

## 5. Pilot case quality

### Questions
- Есть ли **один реальный кейс**, а не абстрактный wish list?
- Кейc достаточно важный для бизнеса?
- Кейc bounded enough for 2-week pilot?

### Good signs
- один видимый case
- понятный scope
- кейс можно провести через task -> contract -> execution -> proof
- результат можно проверить и показать

### Red flags
- хотят “внедрить AI в разработку вообще”
- кейс слишком большой или слишком расплывчатый
- нет понятного результата, который можно проверить

### Notes
- Candidate case: [ ]
- Bounded enough: [yes / no / maybe]
- Business visibility: [high / medium / low]

## 6. Repo / workflow readiness

### Questions
- Есть ли один repo, на котором можно показать bounded result?
- Есть ли доступ к этому repo?
- Есть ли текущий task flow, который можно взять как baseline?

### Good signs
- один repo already identified
- понятный task input path
- есть baseline workflow and review path

### Red flags
- нет repo
- нет доступа
- нет реального workflow, только презентационная идея

### Notes
- Repo available: [yes / no]
- Access complexity: [low / medium / high]
- Workflow readiness: [high / medium / low]

## 7. Security / environment constraints

### Questions
- Есть ли ограничения на vendor/runtime usage?
- Есть ли ограничения на external AI tools?
- Есть ли env/setup blockers для bounded pilot?

### Good signs
- ограничения известны заранее
- можно выбрать допустимый runtime/profile
- нет тяжёлого enterprise blocker before pilot

### Red flags
- security posture unclear
- client expects pilot without giving constraints
- restrictions make even one bounded case impossible

### Notes
- Constraints known: [yes / no]
- Constraint level: [light / medium / heavy]
- Pilot still feasible: [yes / no / maybe]

## 8. Expectation check

### Questions
- Чего клиент ждёт от pilot?
- Понимает ли он, что это не broad rollout?
- Понимает ли он, что мы продаём pilot, а не full platform subscription?

### Good signs
- клиент понимает bounded scope
- клиент готов к result = expand / retry / stop
- клиент не ждёт полной замены стека

### Red flags
- “сделайте нам полную AI-трансформацию”
- “замените Jira / IDE / команду”
- ожидание бесконечного consulting engagement

### Notes
- Expectation fit: [high / medium / low]
- Any dangerous expectations: [ ]

## 9. Commercial fit

### Questions
- Готов ли клиент к paid pilot?
- Понимает ли он формат `from $5,000` as the public anchor?
- Есть ли основания для design-partner discount?

### Good signs
- клиент воспринимает pilot как платный bounded engagement
- цена не вызывает концептуального сопротивления
- design-partner mode нужен по стратегии, а не чтобы спасать weak-fit account

### Red flags
- готов только на free work
- хочет неопределённый diagnostic before any payment
- просит discount because there is no sponsor / no readiness / no scope

### Notes
- Commercial fit: [high / medium / low]
- Pricing reaction: [ ]
- Design partner candidate: [yes / no / maybe]

## 10. Final qualification summary

### Fit summary
- Sponsor: [high / medium / low]
- Team fit: [high / medium / low]
- Pilot case quality: [high / medium / low]
- Repo readiness: [high / medium / low]
- Security feasibility: [high / medium / low]
- Commercial fit: [high / medium / low]

### Recommended verdict
- [ ] go to pilot proposal
- [ ] not now / come back later
- [ ] no-fit

### Why
[short rationale]

### If go
- proposed pilot case: [ ]
- proposed repo: [ ]
- proposed commercial mode: [standard pilot / design partner pilot]
- next action: [send proposal / book follow-up / gather access]

### If not now
- what must become true first: [ ]

### If no-fit
- main blocker: [ ]

---

## Internal founder note

Say **no-go** when:
- there is no sponsor
- there is no bounded case
- there is no repo or task flow to anchor the pilot
- expectations are broad-rollout-first
- discount request is trying to compensate for weak fit
