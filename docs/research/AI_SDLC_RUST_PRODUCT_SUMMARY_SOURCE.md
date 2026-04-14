# AI-SDLC Rust Rewrite — Product Concept Summary

> Сводка самого важного из обсуждения: от исходной постановки до продуктовой концепции, архитектурных решений и next steps.

## 1. Исходная постановка

Стартовая задача была такой:

- подробно изучить текущий проект `ai-sdlc`
- понять, стоит ли его переписывать на Rust
- усилить новое решение идеями из `specpunk` и `signum`
- не делать тупой port 1:1, а собрать более сильный и более продуктовый вариант

Дальше фокус сместился с чисто технического rewrite на продуктовую плоскость:

- зачем мы это делаем
- кому продаем
- как это позиционировать
- как соединить постановку задач и разработку
- как назвать проект
- как стартовать с первым лендингом и MVP

---

## 2. Что показал разбор текущего проекта

### Что в текущем `ai-sdlc` сильное

Текущий проект уже содержит много правильных идей:

- Knowledge Vault / project context
- task board и decomposition
- MCP / context delivery
- chronicle / memory of decisions
- phased workflow: analyze -> refine -> plan -> execute -> verify
- server-side queue, auth, budgets, key pool, admin UI
- analyst / PM / DevOps role thinking
- structured delivery instead of raw prompting

### Что в текущем состоянии слабое

Текущее состояние не выглядит хорошей базой для прямого port на Rust:

- продукт слишком широкий
- в одном слое смешаны core, workflow, integrations и product surfaces
- часть состояния хранится file-first и размазана по разным механизмам
- test baseline drifted
- repo богат идеями, но уже не sharp как trusted kernel

Подтвержденные practical issues при последней проверке:

- `ai_kit` test suite: 4 failing tests
- `ai_kit_server` test suite: 15 failing tests
- `STATUS.md` и package versions drifted
- ai-kit MCP tools существуют в коде, но в текущей сессии не были доступны как реальные инструменты
- repo лучше воспринимать как **source material for extraction**, а не как основу для line-by-line rewrite

### Главный технический вывод

Правильный путь:

**не переносить `ai-sdlc` в Rust как есть, а строить новый продукт с узким kernel и совместимыми edges.**

---

## 3. Синтез из ai-sdlc, specpunk и signum

### Что берем из `ai-sdlc`

Берем:

- project knowledge / shared context
- task shaping
- analyst / PM surfaces
- queue / budgets / admin visibility как будущий product layer
- идею структурированной AI-delivery среды

Не берем в core:

- feature sprawl
- слишком широкий MCP surface в ядре
- file-first mutable state как основную truth model
- попытку тащить все product features в v1 kernel

### Что берем из `specpunk`

Берем:

- small stable kernel
- `plot / cut / gate` как permission boundaries
- append-only ledger / one state truth
- kernel vs shell split
- provider-aligned architecture
- VCS-aware isolation
- idea of one decision writer

Не берем:

- rebuild-the-whole-world mindset
- shell complexity ради shell complexity
- все, что не усиливает boundedness, inspectability и rollback

### Что берем из `signum`

Берем:

- contract-first execution
- contract quality gate
- redacted engineer contract
- execution policy from contract
- holdout verification
- deterministic audit first
- proofpack / explicit verdict semantics

Не берем:

- shell-heavy product form
- слишком тяжелый ritual для tiny tasks
- multi-model review как обязательный trust anchor

### Сводный архитектурный вывод

Новый продукт должен быть:

- **product surfaces** от `ai-sdlc`
- **kernel and runtime truth** от `specpunk`
- **correctness layer** от `signum`

---

## 4. Обновленная продуктовая цель

Изначально акцент ушел в сторону brownfield, но после уточнения была зафиксирована более правильная рамка:

### Важно поддерживать оба сценария

- **existing repos**
- **new projects**

Они равноправны.

Разница только во входе:

- для нового проекта bootstrap начинается с intent / constraints / conventions
- для существующего проекта bootstrap начинается со scan / extraction / reconciliation текущей реальности

После bootstrap оба сценария должны входить в **один и тот же delivery loop**.

### Зачем делаем продукт

Цель — дать командам инструмент, который помогает внедрить AI в разработку **без потери управляемости проекта**.

Это одновременно:

- продукт для разработчиков
- инструмент для PM / analysts / leads
- возможный low-friction entry point в компании
- основа для onboarding, consulting и дальнейших услуг

### GTM-гипотеза

- стартовый рынок: **Россия**
- сначала откатываем positioning и workflow локально
- потом переносим на international market

Модель:

- **tool-first**
- при необходимости — setup / onboarding / consulting / сопровождение

---

## 5. Две ключевые плоскости продукта

После уточнения стало понятно, что продукт нельзя мыслить как dev-only tool.

Нужны **две связанные плоскости**.

## Plane A — Intent / Planning

Для:

- PM
- analyst
- product owner
- tech lead

Что происходит:

- формулируются бизнес-цели
- описываются инициативы и задачи
- фиксируются ограничения
- вытягиваются assumptions
- обнаруживаются противоречия и missing inputs
- AI помогает превратить vague request в structured delivery input

Выход:

- clarified goal
- scope / non-goals
- glossary
- constraints
- acceptance criteria
- open questions
- priority / risk
- draft delivery contract

## Plane B — Delivery / Execution

Для:

- developers
- tech leads
- QA

Что происходит:

- engineering берет approved goal / contract
- связывает это с конкретным repo
- режет на tasks
- выполняет bounded changes
- проверяет результат
- возвращает proof / verdict / feedback

Выход:

- task decomposition
- code changes
- verification results
- decision
- proofpack
- discovered constraints / blockers / learnings

---

## 6. Центральная идея: Project Spine

Между intent plane и delivery plane нужен не просто набор markdown-файлов и не просто task tracker.

Нужен **единый project spine**.

В нем живут ключевые сущности:

- Project
- Initiative
- Goal
- Decision
- Constraint
- Glossary
- Feature
- Contract
- Task
- Run
- Proof
- Learnings

Канонический поток:

`Goal -> Clarify -> Contract -> Tasks -> Change -> Verify -> Proof -> Feedback`

Это и есть центральная модель будущего продукта.

---

## 7. Роль Jira / Linear / Notion

Ключевой вывод:

**не надо строить новый Linear / Jira в v1.**

Правильная роль продукта:

- Jira / Linear / Notion = external systems of record
- новый продукт = **intent-to-delivery layer**

То есть продукт должен:

- импортировать / принимать goals and task inputs
- нормализовать их
- связывать с контекстом проекта
- помогать провести delivery
- возвращать proof / status / artifacts обратно

### Три режима на старте

1. **Manual mode**  
   Пользователь вставляет описания руками.

2. **Connected mode**  
   Есть import/export в Linear / Jira / Notion.

3. **Native lightweight mode**  
   Для маленьких команд без зрелого tracker.

---

## 8. Предлагаемая форма продукта

### Product thesis

**Инструмент, который переводит бизнес-цели и проектные требования в проверяемую инженерную работу.**

Короткая версия:

**от цели до проверенного изменения в коде**

### Что продукт НЕ должен быть

- не AI IDE
- не чат над кодом
- не “еще один multi-agent framework”
- не замена Jira / Linear
- не “autonomous engineering magic”

### Что продукт должен быть

- shared project memory
- goal clarification layer
- delivery contract layer
- bounded execution system
- verification and proof system
- bridge between PM/analyst intent and engineering delivery

---

## 9. Целевые user flows

## Flow 1 — PM / Analyst intake

1. Создается initiative / goal
2. AI задает уточняющие вопросы
3. Система выделяет:
   - assumptions
   - constraints
   - risks
   - acceptance criteria
   - missing context
4. PM / analyst approve
5. Артефакт попадает в Project Spine

## Flow 2 — Engineering shaping

1. Tech lead / dev берет approved goal
2. Привязывает его к repo
3. Система строит draft contract
4. Делается task decomposition
5. Tasks уходят в execution loop

## Flow 3 — Developer execution

1. Агент или инженер берет task
2. Работает в bounded scope
3. Запускает checks / verify / gate
4. Формирует result + proof

## Flow 4 — Feedback back to planning

1. Вверх возвращаются:
   - blockers
   - changed assumptions
   - discovered constraints
   - estimate shifts
   - implementation notes
2. PM / analyst обновляют intent / scope
3. Цикл повторяется

---

## 10. Recommended runtime model

Техническая модель, которая пока выглядит strongest:

### Core modes

- **plot** — understand, clarify, contract, plan
- **cut** — execute bounded change
- **gate** — verify, decide, pack evidence

### Core truth model

Append-only event ledger + materialized views.

### Core kernel objects

- Project
- Goal
- Contract
- Scope
- Workspace
- Run
- DecisionObject
- Proofpack
- Ledger

### Core correctness rule

Сначала correctness definition, потом code.

---

## 11. MVP recommendation

### Что должно войти в MVP

- goal intake
- clarification with AI
- shared project spine
- bootstrap for both new and existing projects
- contract generation
- task shaping
- bounded execution
- verification + proof
- lightweight sync with existing trackers

### Что НЕ должно войти в MVP

- full PM suite
- full replacement for Linear / Jira
- heavyweight enterprise governance
- giant admin product
- all current ai-sdlc features
- advanced debates / councils / figma / stabilize as core

---

## 12. Naming discussion

### Candidate 1 — Devrail

Плюсы:

- developer-friendly
- понятно звучит
- хорошо про safe rails

Минус:

- слишком узко звучит после появления PM / analyst plane

### Candidate 2 — Goalrail

Плюсы:

- лучше отражает обе плоскости
- хорошо маппится на формулу “от цели до реализации”
- сильнее в positioning

### Candidate 3 — Flowrail

Плюсы:

- мягче и шире
- хорошо про end-to-end workflow

На текущий момент **наиболее сильный рабочий кандидат — `Goalrail`**.

### Working tagline

**Goalrail — от бизнес-цели до проверенного изменения в коде.**

---

## 13. Предлагаемое позиционирование

### One-line positioning

**Платформа, которая соединяет intent-management и AI-assisted delivery в одном проектном контуре.**

### Simpler market version

**Goalrail помогает аналитикам, PM и разработчикам переводить цели проекта в проверяемую инженерную работу.**

### Engineering-first variant

**Внедряйте AI в разработку без хаоса в проекте — от постановки задачи до проверенного результата.**

---

## 14. Идея первого лендинга

Первый лендинг должен продавать не архитектуру и не внутренние implementation terms.

Он должен продавать outcomes.

### Hero

**От бизнес-цели до проверенного изменения в коде**

### Subheadline

**Goalrail помогает аналитикам, PM и разработчикам работать в одном AI-контуре: уточнять требования, связывать их с контекстом проекта, нарезать инженерную работу, выполнять изменения в bounded scope и проверять результат до merge.**

### 3 value props

1. **Собирает intent**  
   Превращает vague task description в структурированный delivery input.

2. **Ведет разработку по границам**  
   AI и инженеры работают в scope проекта, а не наугад.

3. **Возвращает proof**  
   Видно, что изменили, как проверили и можно ли этому доверять.

---

## 15. Практический старт

Логика старта сейчас выглядит такой:

1. Зафиксировать working name
2. Зафиксировать product thesis
3. Сделать первый one-pager / landing
4. Подготовить короткий demo flow
5. Прогнать интервью на рынке
6. Только после этого пилить MVP kernel

### Suggested immediate order

1. **Name**  
   Goalrail / Flowrail / final pick

2. **Product one-pager**  
   ICP, pains, promise, flows, differentiation

3. **Landing skeleton**  
   hero, value props, CTA

4. **MVP definition**  
   first thin slice

5. **Rust kernel design**  
   crates, schemas, event model, proof model

---

## 16. Главные финальные выводы

1. **Не делать прямой TypeScript -> Rust port.**  
   Текущий `ai-sdlc` слишком широк и drifted для этого.

2. **Новый продукт должен быть dual-plane.**  
   Нужны и intent/planning, и delivery/execution.

3. **Новые и существующие проекты равноправны.**  
   Различается bootstrap, а не core workflow.

4. **Нужен единый Project Spine.**  
   Он связывает goals, decisions, tasks, runs и proof.

5. **В v1 нельзя пытаться заменить Jira / Linear.**  
   Ценность — в связке intent, project context, bounded execution и proof.

6. **Самая сильная архитектурная формула сейчас:**  
   - product surfaces от `ai-sdlc`
   - kernel / ledger / mode boundaries от `specpunk`
   - contract / audit / proof semantics от `signum`

7. **Самая сильная продуктовая формула сейчас:**  
   **от цели до проверенного изменения в коде**

---

## 17. Recommended next artifacts

Следом стоит подготовить:

- `ONE-PAGER.md` — product thesis and positioning
- `LANDING-COPY.md` — hero, sections, CTA
- `MVP-SCOPE.md` — first release scope
- `RUST-KERNEL-ADR.md` — architecture baseline
