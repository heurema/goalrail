# Goalrail Deployment Model

## 1. Purpose

Этот документ описывает, как Goalrail внедряется как productized operating layer.

Цель deployment model:

- не превращать Goalrail во внедрение “под каждого с нуля”
- не скатываться в bespoke consulting
- сохранять продуктовый каркас
- покрывать различия компаний через ограниченный набор настроек

## 2. Deployment principle

Goalrail внедряется не как “мы изучим ваш уникальный процесс и пересоберём решение”.

Goalrail внедряется как:

**готовый operating model + ограниченная конфигурация + pilot-first rollout**

## 3. Fixed core vs configurable deployment

### Fixed core
Во всех внедрениях сохраняется:

- incoming task -> contract -> execution -> verify -> proof flow
- contract-first logic
- bounded execution
- one primary writer per writable run
- inspectable proof
- source-of-truth contour

### Configurable deployment knobs
Под организацию можно настраивать:

- tracker mode
- runtime mode
- policy profile
- review depth
- terminology mapping
- approval profile
- proof strictness
- scope templates

## 4. Supported deployment modes

### Mode A — Managed deployment
Команда Goalrail руками ведёт initial setup, pilot и stabilization.

Рекомендуется для первых design partners.

### Mode B — Guided deployment
Клиент получает toolkit, setup guidance и короткий enablement path.

Рекомендуется позже, когда deployment playbook стабилизируется.

### Recommended default
На первом этапе Goalrail должен запускаться как **managed deployment**.
Guided deployment — позже.

## 5. Deployment phases

### Phase 0 — Qualification / Fit Check
Цель:
- понять, подходит ли клиент под стандартный deployment

Проверяем:
- team size and structure
- sponsor presence
- repo availability
- task flow existence
- AI readiness
- security blockers

Результат:
- go / no-go
- pilot candidate
- initial profile selection

### Phase 1 — Profile Selection
Выбирается базовый deployment profile.

Определяем:
- tracker mode
- runtime mode
- policy profile
- review depth
- terminology profile

Результат:
- chosen deployment profile

### Phase 2 — Bootstrap
Подключаются минимально необходимые элементы.

Подключаем:
- repo binding
- runtime binding
- intake mode
- contract template
- proof template
- policy defaults

Результат:
- working pilot environment

### Phase 3 — Onboarding
Команде объясняется новый operating model.

Показываем:
- как задача входит
- как выглядит contract
- как выглядит execution path
- как выглядит verify / proof output

Результат:
- team readiness for pilot

### Phase 4 — Pilot Run
Один реальный кейс проходит через стандартный Goalrail flow.

Результат:
- completed pilot run
- proof output
- before / after readout

### Phase 5 — Stabilization
Подкручиваются только настройки.

Изменяются:
- policy strictness
- review depth
- templates
- terminology mapping
- runtime preferences

Не изменяется:
- fixed operating core

Результат:
- stabilized deployment profile

### Phase 6 — Expansion
Решается, куда расширяться дальше.

Варианты:
- second repo
- second use case
- second team
- stop / pause if no-fit

## 6. Deployment profiles

### Profile 1 — Standard Product Team
Для обычной продуктовой команды.
- tracker connected or manual
- Codex or Claude Code
- standard review depth
- lightweight proof required

### Profile 2 — Strict Review Team
Для команд с более жёстким review / approval path.
- stricter policy
- deeper verification
- explicit signoff
- stronger proof requirements

### Profile 3 — Security-Sensitive Team
Для команд с ограничениями на data exposure и vendor usage.
- restricted runtime set
- local-only or single-vendor rules where required
- stronger policy profile
- more constrained execution packets

## 7. Runtime recommendation

Рекомендуемый первый runtime set:

- Codex — required support
- Claude Code — required support
- Gemini — optional / experimental support

Причина:
- первые два покрывают самый вероятный initial deployment scope
- третий стоит держать как optional adapter, а не как обязательный anchor

## 8. Proof posture

Рекомендуемое правило:

**proof required from day one**

Но proof может быть lightweight в первой поставке.

Минимальный proof:
- task / contract reference
- what changed
- how checked
- verdict
- open risks / follow-ups

## 9. What deployment is not

Deployment Goalrail не должен означать:

- полную перестройку всей engineering организации
- замену всех текущих процессов клиента
- полный audit factory-style complexity под каждого клиента
- кастомную пересборку core operating logic

## 10. Deployment success criteria

Deployment считается успешным, если:

1. команда реально прошла через Goalrail pilot flow
2. contract стал центральным working object
3. execution был bounded
4. result завершился proof output
5. клиент понимает, имеет ли смысл rollout дальше

## 11. Default recommendation

Рекомендуемый порядок запуска Goalrail:

1. qualification
2. profile selection
3. managed bootstrap
4. team onboarding
5. one pilot run
6. stabilization
7. expansion decision
