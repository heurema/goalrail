---
id: goalrail_landing_copy
title: Goalrail Landing Copy — Historical Technical Draft
kind: public_entry
authority: reference
status: superseded
owner: product
truth_surfaces:
  - historical_technical_landing_draft
lifecycle: retired
review_after: 2026-07-19
supersedes: []
superseded_by: docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md
related_docs:
  - docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md
  - docs/ops/DECISIONS.md
---
# Goalrail — Landing Copy

> **Historical / superseded.**
> This is a historical technical landing draft for prompt / agent handoff framing.
> It is **not** the current public RU landing canon.
> Current canon: `docs/product/GOALRAIL_LANDING_COPY_PILOT_FIRST.md`.
> Implementation surface: `apps/web/pilot-intake-ru`.

> Версия для первого публичного экрана.
> Не классический лендинг. Не SaaS-homepage. Не dashboard.
> Это single-screen entry scene с одним полезным действием.

---

## 1. Page intent

Страница должна за 5–10 секунд объяснить:

- Goalrail не “ещё один агент”
- Goalrail не замена Jira / Linear
- Goalrail помогает перевести сырую постановку в рабочий handoff для агента и инженерии

Первый честный сценарий страницы:

`сырой запрос -> clarified goal -> constraints -> scope -> agent handoff prompt`

---

## 2. Headline options

### Recommended primary
**Из сырой постановки — в рабочий handoff для агента**

### Alternatives
1. **Размытая задача — плохой вход для AI. Goalrail исправляет именно это**
2. **Goalrail превращает постановку задачи в инженерно пригодный handoff**
3. **От постановки задачи — к работе, которую можно проверить**
4. **Сначала смысл. Потом границы. Потом handoff**
5. **Goalrail переводит intent в bounded delivery**

---

## 3. Supporting line options

### Recommended primary
**Goalrail выделяет цель, ограничения и границы, а затем собирает один сильный prompt для агента.**

### Alternatives
1. **Описали задачу как есть — получили структурированный handoff для инженерии и AI.**
2. **Goalrail помогает команде убрать шум из постановки до того, как AI начнёт писать код.**
3. **Из vague request получается goal, scope, constraints и рабочий prompt.**
4. **Неясная задача становится bounded артефактом, который можно передать дальше.**
5. **Goalrail собирает первую проверяемую структуру вокруг задачи, а не просто красивый prompt.**

---

## 4. Main input area

### Label
**Опишите задачу как её ставят сейчас**

### Prefilled example
**Нужно как-то ускорить загрузку главной страницы, особенно на мобильных. Кажется, картинки тяжелые, и скриптов много лишних.**

### Example chips
- Улучшить активацию пользователя
- Поиск иногда не показывает результаты
- Сократить время генерации отчёта

---

## 5. Primary CTA

### Recommended primary
**Разобрать задачу**

### Alternatives
1. **Собрать handoff**
2. **Запустить быстрый прогон**
3. **Преобразовать в handoff**
4. **Разложить задачу**
5. **Подготовить handoff**

---

## 6. Output preview copy

### Panel title
**Предпросмотр результата**

### Block 1 — Goal
**Цель**

Example:
**Ускорить загрузку главной страницы на мобильных устройствах.**

### Block 2 — Constraints
**Ограничения**

Example:
- Без потери визуального качества
- Не ломать основной JavaScript-поток

### Block 3 — Scope
**Границы**

Example:
**Главная страница, изображения above-the-fold, сторонние скрипты, клиентские JS-бандлы.**

### Block 4 — Agent prompt
**Prompt для агента**

Example preview:

> ROLE: Frontend Performance Engineer  
> TASK: Analyze homepage assets with focus on mobile TTI and LCP.  
> OBJECTIVE: Reduce payload and render delay without visual degradation.  
> STEPS: Audit image pipeline, isolate non-critical scripts, propose scoped implementation plan.

### Tiny action
**Скопировать prompt**

### Compatibility note
**Работает с Codex, Claude Code и Gemini**

---

## 7. Alternative microcopy for agent handoff

1. **Открыть в Codex**
2. **Открыть в Claude Code**
3. **Открыть в Gemini**
4. **Скопировать prompt**
5. **Скачать стартовый prompt**

If there is only one small compatibility line, use:

**Работает с Codex, Claude Code и Gemini**

---

## 8. Quiet closing line options

### Recommended primary
**Сначала смысл. Потом границы. Потом proof.**

### Alternatives
1. **Неясная задача — плохой вход. Goalrail делает его рабочим.**
2. **Сначала intent. Потом handoff. Потом delivery.**
3. **Goalrail начинает там, где задача ещё не готова к инженерии.**
4. **Сырой запрос — это ещё не работа. Goalrail делает следующий шаг.**
5. **Меньше шума в постановке. Больше управляемости в delivery.**

---

## 9. Suggested exact page copy (tight version)

### Wordmark
Goalrail

### Headline
**Из сырой постановки — в рабочий handoff для агента**

### Supporting line
**Goalrail выделяет цель, ограничения и границы, а затем собирает один сильный prompt для агента.**

### Input label
**Опишите задачу как её ставят сейчас**

### Input example
**Нужно как-то ускорить загрузку главной страницы, особенно на мобильных. Кажется, картинки тяжелые, и скриптов много лишних.**

### Button
**Разобрать задачу**

### Output title
**Предпросмотр результата**

### Output blocks
**Цель**  
Ускорить загрузку главной страницы на мобильных устройствах.

**Ограничения**  
— Без потери визуального качества  
— Не ломать основной JavaScript-поток

**Границы**  
Главная страница, изображения above-the-fold, сторонние скрипты, клиентские JS-бандлы.

**Prompt для агента**  
ROLE: Frontend Performance Engineer  
TASK: Analyze homepage assets with focus on mobile TTI and LCP.  
OBJECTIVE: Reduce payload and render delay without visual degradation.

### Tiny action
**Скопировать prompt**

### Compatibility
**Работает с Codex, Claude Code и Gemini**

### Closing line
**Сначала смысл. Потом границы. Потом proof.**
