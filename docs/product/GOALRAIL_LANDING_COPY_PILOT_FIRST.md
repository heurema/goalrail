# Goalrail — Landing Copy (Pilot-First Rewrite)

> Replacement-direction draft for the public landing flow.
>
> This version aligns landing copy with:
> - `GOALRAIL_GTM_MODEL.md`
> - `GOALRAIL_PILOT_MODEL.md`
> - `GOALRAIL_OFFER.md`
> - `GOALRAIL_DESIGN_DECISIONS.md`
>
> Goal:
> move public entry away from prompt-export framing and toward contract-centered, pilot-first lead capture.

---

## 1. Page intent

Страница должна за 5–10 секунд объяснить:

- Goalrail не “ещё один агент”
- Goalrail не замена Jira / Linear
- Goalrail помогает превратить входящую задачу в **рабочий контракт** до начала execution
- Goalrail заходит в команду через **pilot**, а не через большой rollout

Первый честный сценарий страницы:

`incoming task -> draft working contract -> pilot request`

Не:

`incoming task -> prompt export`

---

## 2. Public flow recommendation

### Scene 1
**Incoming task -> draft working contract**

Пользователь видит:
- сырой входящий запрос
- как Goalrail выделяет цель
- как появляются ограничения
- как появляется scope / non-goals
- как задача становится рабочим объектом для команды

Primary CTA:
**Открыть разбор**

### Scene 2
**Contract breakdown -> pilot request**

Пользователь видит:
- что входит в contract
- как это пойдёт дальше в bounded execution
- как будет выглядеть proof
- что Goalrail запускается как pilot для одной команды

Primary CTA:
**Получить пилотный разбор**

---

## 3. Headline options

### Recommended primary
**Из входящей задачи — в рабочий контракт для команды**

### Strong alternatives
1. **От бизнес-задачи — к управляемой инженерной работе**
2. **Сначала задача становится контрактом. Потом начинается execution**
3. **Goalrail превращает сырую постановку в рабочий delivery contour**
4. **Неясная задача — плохой вход для AI. Goalrail делает её рабочей**
5. **От входящего запроса — к bounded execution и proof**

---

## 4. Supporting line options

### Recommended primary
**Goalrail выделяет цель, ограничения и границы, собирает рабочий контракт и запускает pilot-first путь к bounded execution и proof.**

### Alternatives
1. **Goalrail помогает команде превратить vague request в общий рабочий объект до того, как AI начнёт писать код.**
2. **Из входящей задачи появляется contract, по которому и бизнес, и инженерия видят один и тот же scope.**
3. **Goalrail не заменяет текущий стек — он добавляет общий contract, bounded execution и proof-oriented visibility.**
4. **Не просто handoff для агента, а управляемый путь от задачи до проверяемого результата.**

---

## 5. Scene 1 copy

### Input label
**Опишите задачу как она приходит в команду сейчас**

### Input example
**Нужно ускорить загрузку главной страницы на мобильных. Кажется, проблема в тяжёлых изображениях и лишних клиентских скриптах.**

### Example chips
- Ускорить отчёт, который долго собирается
- Поиск иногда не показывает результаты
- Улучшить активацию пользователя в онбординге

### Primary CTA
**Открыть разбор**

### Supporting microcopy
**Сначала Goalrail собирает рабочий контракт. Только потом execution.**

---

## 6. Scene 1 output preview

### Panel title
**Черновик рабочего контракта**

### Block 1 — Goal
**Цель**

Example:
**Ускорить загрузку главной страницы на мобильных устройствах без ухудшения пользовательского опыта.**

### Block 2 — Constraints
**Ограничения**

Example:
- Без визуальной деградации above-the-fold контента
- Не ломать основной JS flow
- Не ухудшить существующую аналитику и трекинг

### Block 3 — Scope
**Что в scope**

Example:
- главная страница
- изображения above-the-fold
- сторонние скрипты
- клиентские JS-бандлы

### Block 4 — Out of scope
**Что вне scope**

Example:
- redesign страницы
- смена analytics provider
- полная переработка frontend architecture

### Block 5 — Verify expectation
**Как будет проверяться**

Example:
- performance checks
- regression review
- acceptance against contract scope
- proof summary at the end

### Microcopy under preview
**Goalrail делает задачу общим рабочим объектом для PM, разработчика и AI-assisted delivery.**

---

## 7. Scene 2 copy

### Section title
**Как это работает дальше**

### Flow explainer
После contract задача идёт в контролируемый delivery flow:

`contract -> bounded execution -> verify -> proof`

### Three explanation blocks

#### Block 1
**Contract first**
Сначала команда видит цель, ограничения, scope и expected checks в одном рабочем объекте.

#### Block 2
**Bounded execution**
Работа идёт в ограниченном контуре, а не через хаотичные prompt loops.

#### Block 3
**Proof at the end**
На выходе видно не только что изменили, но и как проверили и можно ли доверять результату.

---

## 8. Pilot request section

### Section title
**Первый честный вход — pilot на одном реальном кейсе**

### Recommended body copy
Goalrail не продаётся как большая трансформация с первого дня.

Мы начинаем с pilot:
- 1 команда
- 1 repo на старте
- 1 реальный кейс
- bounded deployment
- proof from day one

### Primary CTA
**Получить пилотный разбор**

### Secondary CTA
**Прислать задачу на разбор**

### Small trust line
**Сначала qualification. Потом pilot. Потом решение о rollout.**

---

## 9. Objection-handling strip

### Suggested microcopy
**Не заменяет ваш текущий стек. Не требует большого rollout с первого дня. Не продаёт “автопилот”.**

Alternative version:

**Goalrail дополняет текущие инструменты там, где между задачей и результатом сегодня теряется управляемость.**

---

## 10. Quiet closing line options

### Recommended primary
**Сначала контракт. Потом execution. Потом proof.**

### Alternatives
1. **Неясная задача — плохой вход. Goalrail делает её рабочей.**
2. **Один и тот же рабочий объект для PM, команды и AI-assisted delivery.**
3. **Меньше шума между задачей и результатом. Больше управляемости в delivery.**
4. **Не просто activity. А bounded execution и inspectable proof.**

---

## 11. Suggested exact landing copy (tight version)

### Wordmark
Goalrail

### Headline
**Из входящей задачи — в рабочий контракт для команды**

### Supporting line
**Goalrail выделяет цель, ограничения и границы, собирает рабочий контракт и запускает pilot-first путь к bounded execution и proof.**

### Input label
**Опишите задачу как она приходит в команду сейчас**

### Input example
**Нужно ускорить загрузку главной страницы на мобильных. Кажется, проблема в тяжёлых изображениях и лишних клиентских скриптах.**

### Button
**Открыть разбор**

### Output title
**Черновик рабочего контракта**

### Output blocks
**Цель**  
Ускорить загрузку главной страницы на мобильных устройствах без ухудшения пользовательского опыта.

**Ограничения**  
— Без визуальной деградации above-the-fold контента  
— Не ломать основной JS flow  
— Не ухудшить существующую аналитику

**Что в scope**  
Главная страница, изображения above-the-fold, сторонние скрипты, клиентские JS-бандлы.

**Что вне scope**  
Редизайн страницы, смена analytics provider, полная переработка frontend architecture.

**Как будет проверяться**  
Performance checks, regression review, acceptance against contract scope, proof summary.

### Transition section
**Как это работает дальше**

Contract -> bounded execution -> verify -> proof

### Pilot section
**Первый честный вход — pilot на одном реальном кейсе**

1 команда. 1 repo на старте. 1 реальный кейс. Proof from day one.

### Primary CTA
**Получить пилотный разбор**

### Secondary CTA
**Прислать задачу на разбор**

### Closing line
**Сначала контракт. Потом execution. Потом proof.**

---

## 12. Replacement recommendation

This rewrite should replace the older landing logic that led with:
- agent handoff
- prompt export
- “copy prompt” micro-actions

The public landing should now lead with:
- working contract
- bounded execution path
- pilot request
- proof-oriented delivery value
