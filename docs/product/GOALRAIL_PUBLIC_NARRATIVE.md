# Goalrail — Public Narrative

> Канонический документ для внешней narrative-рамки.
> Фиксирует relationship между Goalrail, Goal on Rails и Specpunk,
> а также правила для business-first build in public.

## 1. Purpose

Этот документ нужен, чтобы публичная история не расползалась между:
- product positioning
- build in public
- short-form content
- landing / campaign wording
- mascot / character layer boundaries

Он фиксирует **не продуктовую архитектуру**, а **публичную narrative-рамку**.

## 2. Scope and boundaries

Этот документ отвечает за:
- публичный story frame
- relationship между Goalrail, Goal on Rails и Specpunk
- current public objective
- tone
- narrative laws
- content pillars

Этот документ **не** заменяет:
- `GOALRAIL_PRODUCT_CONCEPT.md` — канон продукта
- `GOALRAIL_OPERATING_MODEL.md` — operating flow
- `GOALRAIL_MESSAGE_HIERARCHY.md` — message stack и public wording
- `GOALRAIL_LANDING_COPY.md` — screen-level copy

Mascot / character ownership lives in:
- `docs/brand/PUNK_CHARACTER_SYSTEM.md`

Этот документ также **не** фиксирует:
- финальную visual identity system
- mascot art bible
- motion/animation production rules
- legal brand policy

## 3. Current public objective

Текущая публичная цель Goalrail:

**собирать business-side audience вокруг проблемы AI delivery drift и вокруг честного pilot-first narrative.**

Primary audience:
- PM
- product lead
- tech lead
- CTO / Head of Engineering
- business sponsor

Secondary audience:
- developers
- AI builders
- operators

Публичная подача по умолчанию остаётся **business-first**, а не developer-first.

## 4. Brand stack

### 4.1 Goalrail

**Goalrail** — это продукт.

Это default product name для:
- product docs
- pilot conversations
- offer / GTM
- product surfaces
- system explanation

### 4.2 Goal on Rails

**Goal on Rails** — это public series / campaign line / narrative umbrella.

Это не default product name.

Его задача:
- собирать публичную историю
- держать метафору rails
- связывать short-form content, build in public и headline-level phrasing

### 4.3 Specpunk

**Specpunk** — sibling builder/runtime project, через который публично строится Goalrail.

В narrative-слое Specpunk — это vehicle of construction:
- builder
- runtime ethos
- anti-drift engineering energy

Specpunk не должен подменять собой Goalrail как продаваемый продукт.

### 4.4 Punk

**Punk** — mascot / character layer.

Punk нужен как brand carrier, но не как source of truth.

Правило:
- персонаж усиливает narrative
- продуктовую истину определяют canonical product docs
- детальный mascot canon хранится в brand layer, а не здесь

## 5. Core story

Current working public story:

1. Бизнес ставит цель.
2. Без общего control layer AI delivery начинает drift.
3. Goalrail ставит goal на rails.
4. Specpunk строит этот слой публично.
5. На выходе нужен не просто patch, а проверяемый результат с proof logic.

Short version:

**Business sets a goal. Goalrail puts it on rails. Specpunk builds the system in public.**

## 6. Public promise

Goalrail публично обещает не “магический автопилот”, а:
- controlled pilot motion
- bounded AI-assisted delivery
- ясные границы работы
- проверяемый результат
- честный build in public

Что это значит practically:
- если продукта ещё нет в finished form, это проговаривается прямо
- если есть prototype / pilot layer, это так и называется
- нельзя создавать ложное впечатление полной зрелости или broad rollout readiness

## 7. Tone

Default tone:
- business-first
- clear
- concrete
- anti-enterprise theater
- serious cyberpunk fun

Это значит:
- не слишком corporate-smooth
- не meme-only
- не “edgy ради edgy”
- не overly technical by default

## 8. Narrative laws

### Law 1 — Lead with pain and control

Сначала объясняется проблема:
- drift
- ambiguity
- weak handoff
- lack of proof

А не:
- agents
- models
- runtime internals

### Law 2 — One visible object in motion

Публичная история должна показывать один понятный объект:

**goal -> rails -> execution -> verification -> result**

Не нужно одновременно продавать десять внутренних сущностей.

### Law 3 — Product first, lore second

Goalrail остаётся real product object, который объясняется прямо.

Нельзя делать public story настолько lore-heavy, что становится непонятно, что именно строится.

### Law 4 — Show build evidence

Когда возможно, public content должен опираться на реальные build artifacts:
- docs
- flows
- wireframes
- pilot shape
- runtime slices
- product decisions

### Law 5 — Canon wins

Если public narrative начинает конфликтовать с product canon,
побеждает:
- concept canon
- operating model
- deployment / pilot model
- MVP blueprint

## 9. Content pillars

Current default pillars:

### 9.1 Why AI delivery drifts
Почему скорость execution не решает проблему управляемости.

### 9.2 Put goals on rails
Что значит поставить goal в управляемый delivery contour.

### 9.3 Building Goalrail in public
Как именно Goalrail собирается как продукт и pilot layer.

### 9.4 Honest pilot motion
Как продавать и запускать не broad platform, а pilot-first engagement.

### 9.5 Proof over vibes
Почему patch / diff / PR недостаточны без verify / proof logic.

## 10. Current working formulas

Primary working formulas:
- **Put goals on rails.**
- **От бизнес-цели до проверенного результата.**
- **Build with proof, not vibes.**

Optional Punk-layer line:
- **Не в дрифт. На рельсы.**

## 11. Practical naming rule

Current default naming rule:
- **Goalrail** — product name
- **Goal on Rails** — public series / campaign line / narrative umbrella
- **Specpunk** — builder/runtime sibling project
- **Punk** — mascot / character layer

Rule:

Нельзя silently переименовывать продукт в `Goal on Rails` across product docs,
product UI, offer docs или pilot docs без отдельного явного решения.

Если analogy to Rails используется публично,
она должна оставаться analogy / campaign phrasing, а не formal product identity.

## 12. Out of scope for this doc

Этот документ пока не фиксирует:
- final logo system
- mascot design canon
- exact palettes / typography
- motion system
- content calendar
- asset production workflow

Для этого позже могут появиться отдельные docs в `docs/brand/`.
