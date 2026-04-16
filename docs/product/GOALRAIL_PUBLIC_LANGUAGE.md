# Goalrail — Public Language

> Translation layer between internal runtime language and external business-facing wording.
> Нужен, чтобы build in public, landing, CTA, sales conversations и short-form content
> говорили на одном понятном языке.

## 1. Purpose

Этот документ фиксирует:
- default external vocabulary
- naming rules
- phrase selection rules
- business-safe wording
- short-form content grammar

Он нужен, чтобы не смешивать:
- internal runtime language
- product architecture language
- sales-safe public language

## 2. Boundary

Этот документ **не** определяет product truth.

Product truth определяется через:
- `GOALRAIL_PRODUCT_CONCEPT.md`
- `GOALRAIL_OPERATING_MODEL.md`
- `GOALRAIL_MVP_BLUEPRINT.md`

`GOALRAIL_MESSAGE_HIERARCHY.md` отвечает за:
- what we say first / second / third
- message stack
- sales-safe positioning

А этот документ отвечает за:
- **какими словами говорить вовне по умолчанию**
- **какие internal terms прятать или переводить**

## 3. Audience default

Внешний язык по умолчанию оптимизируется под:
- PM
- product lead
- tech lead
- CTO / Head of Engineering
- business sponsor

В mixed rooms developer language допустим,
но по умолчанию public-facing materials остаются business-first.

## 4. Default external vocabulary

| Internal notion | Public default | Notes |
|---|---|---|
| Goal | business goal / operating goal | Лучше начинать с goal, а не с task или ticket |
| Contract | shared working contract / delivery contract | Один из ключевых внешних терминов |
| Scope | boundaries / execution boundaries | Слово boundaries обычно понятнее широкой аудитории |
| Task shaping | work breakdown / execution plan | Не обязательно раскрывать как отдельный subsystem |
| Run | execution run / bounded run / attempt | Использовать только когда реально нужен process step |
| Verification | verify / review / checks | `verify` и `checks` обычно звучат проще, чем `validation lane` |
| Decision | verdict / go-no-go decision | Использовать там, где нужен управленческий смысл |
| Proof | proof / evidence / verified result | Один из ключевых внешних терминов |
| Plot | frame / clarify / shape | Internal term по умолчанию не выносить в first-layer public copy |
| Cut | run / execute | Internal term по умолчанию не выносить в first-layer public copy |
| Gate | verify / decide | Internal term по умолчанию не выносить в first-layer public copy |
| Advisory panel | advisory review / second-opinion review | Понятнее для mixed business / tech audience |

## 5. What we say first

По умолчанию public material должен начинаться с одного из этих словарей:
- goal
- rails
- control
- boundaries
- verify
- proof

Не с:
- agents
- orchestration
- model routing
- memory
- protocol families
- runtime matrix

## 6. What we usually avoid in first-layer public copy

По умолчанию не ведём с:
- `plot / cut / gate`
- `MCP`
- `agent swarm`
- `autonomous engineering`
- `orchestration fabric`
- `multi-model panel logic`
- `benchmarking substrate`

Эти слова можно использовать:
- в deeper technical material
- в engineering conversations
- в repo / architecture docs

Но не в first-screen / first-post / first-pitch language.

## 7. Naming rules

### 7.1 Product name

По умолчанию product name:

**Goalrail**

Используется в:
- product docs
- pilot docs
- offer docs
- product surfaces
- system explanation

### 7.2 Public series / campaign line

По умолчанию public series line:

**Goal on Rails**

Используется в:
- build in public framing
- campaign phrasing
- narrative umbrella
- series naming

### 7.3 Phrase line

Допустимая phrase line:

**Put goals on rails.**

### 7.4 Rule

Нельзя без отдельного решения:
- переименовывать продукт в `Goal on Rails`
- переносить series line в legal / formal product identity
- делать campaign phrase основной заменой product name в canonical docs

## 8. Business-safe phrases

### Good default phrases
- pilot
- pilot review
- put goals on rails
- controlled AI delivery
- shared working contract
- verified result
- proof-oriented delivery
- governed execution
- reviewable work

### Use carefully
- autonomy
- self-driving delivery
- multi-agent system
- autonomous engineering

### Usually avoid
- replace your developers
- one-click AI engineering
- deterministic delivery guarantee
- agent army
- full autopilot for software teams

## 9. Public explanation template

Когда нужно быстро объяснить Goalrail вовне, default pattern такой:

1. **Problem**
   AI ускоряет execution, но delivery легко уходит в drift.

2. **Rail**
   Goalrail ставит goal в общий working contract и execution boundaries.

3. **Result**
   Команда получает не vague task flow, а reviewable delivery path.

4. **Proof**
   На выходе важен не просто patch, а verified result.

5. **CTA**
   Предложить pilot review / task review / walkthrough.

## 10. Short-form content grammar

Default grammar for videos / carousels / posts:

**Pain -> Rail -> Result -> Proof -> CTA**

Examples of leading lines:
- AI пишет код быстрее, чем команды успевают договориться о смысле.
- Goal without rails becomes delivery drift.
- Patch is not the same thing as proof.
- If the goal is vague, the AI path will drift too.

## 11. CTA defaults

Preferred CTA set:
- **Получить пилотный разбор**
- **Прислать задачу на разбор**
- **Посмотреть, как goal ставится на rails**
- **Открыть walkthrough**

Avoid default CTA set:
- купить подписку
- запустить полного агента
- заменить процесс целиком
- автоматизировать всё сразу

## 12. Practical rule for internal vocabulary

Internal runtime terms допустимы публично только если выполняются оба условия:

1. Они реально помогают объяснению, а не делают его сложнее.
2. Рядом есть business-readable translation.

Пример:
- допустимо: `gate = verify + final decision boundary`
- нежелательно: просто `gate`, если аудитория не понимает, о чём речь

## 13. Out of scope for this doc

Этот документ пока не фиксирует:
- exact landing copy
- long-form sales deck wording
- mascot dialogue style
- social calendar
- visual slogan system

Для этого позже могут появиться отдельные docs.
