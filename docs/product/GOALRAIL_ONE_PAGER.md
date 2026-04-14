# Goalrail — One Pager

## 1. Что это

**Goalrail** — intent-to-delivery layer для software teams.

Он помогает перевести размытый бизнес-запрос в проверяемую инженерную работу:

`vague request -> clarified goal -> contract -> task -> bounded run -> verify -> proof`

Короткая формула:

**От бизнес-цели до проверенного изменения в коде.**

---

## 2. Для кого

### Primary ICP

- product teams с 5–30 инженерами
- команды, где уже есть PM / analyst / tech lead
- команды, которые уже пробуют AI в delivery, но не хотят терять управляемость
- B2B SaaS / internal product teams / engineering studios

### Пользователи

#### Plane A — Intent / Planning
- PM
- analyst
- product owner
- tech lead

#### Plane B — Delivery / Execution
- developer
- tech lead
- QA

---

## 3. Какую проблему решает

Сегодня команды сталкиваются не только с проблемой «какой агент лучше», а с проблемой **разрыва между постановкой задачи и инженерным результатом**:

- задачи формулируются расплывчато
- ограничения и non-goals не фиксируются явно
- AI начинает писать код слишком рано
- неясно, что было в scope
- в конце есть merge / patch / diff, но нет proof, которому можно доверять

Goalrail решает именно этот разрыв.

---

## 4. Что делает продукт

### На входе
- сырой запрос / initiative / task
- проектный контекст
- ограничения / glossary / acceptance criteria

### Внутри
- уточняет intent
- собирает structured goal packet
- строит delivery contract
- режет на bounded tasks
- запускает bounded execution
- прогоняет verify / gate
- возвращает proof и feedback

### На выходе
- не просто code change
- а **проверяемый инженерный результат**

---

## 5. Ключевой механизм

Центр продукта — **Project Spine**.

Это единый проектный контур, где связаны:

- Goal
- Constraint
- Glossary
- Contract
- Task
- Run
- Decision
- Proof
- Learnings

Канонический поток:

`Goal -> Clarify -> Contract -> Tasks -> Change -> Verify -> Proof -> Feedback`

---

## 6. Что Goalrail не пытается быть

- не AI IDE
- не generic coding agent
- не замена Jira / Linear
- не чат над кодом
- не “autonomous engineering magic”
- не heavyweight enterprise governance suite в первой версии

---

## 7. Чем отличается

### Вместо “ещё один AI tool”
Goalrail — это **слой управления AI-assisted delivery**.

### Вместо “просто prompt”
Goalrail даёт:
- clarified intent
- bounded contract
- execution boundary
- verification
- proof

### Вместо “сделали вроде бы”
Goalrail возвращает:
- что изменили
- как проверили
- можно ли этому доверять

---

## 8. MVP

### Входит в MVP
- goal intake
- clarification with AI
- project spine
- bootstrap для new и existing repos
- contract generation
- task shaping
- bounded execution
- verification + proof
- lightweight tracker sync later

### Не входит в MVP
- замена Linear / Jira
- full PM suite
- giant admin layer
- broad platform sprawl

---

## 9. Первый market entry

### Первый рынок
**Россия**

### Первая коммерческая форма
**Pilot / design partner engagement**

Формат:
- 1 команда
- 1–2 repo
- 2–4 недели
- intent -> contract -> task -> run -> verify -> proof

### Первый CTA
Не “купить подписку”, а:
- открыть контрольный / smoke case
- попробовать быстрый прогон
- забрать handoff для своего агента

---

## 10. Самая короткая версия

**Goalrail помогает команде переводить размытые задачи в проверяемую инженерную работу — от бизнес-цели до proof.**
