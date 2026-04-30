---
id: goalrail_landing_copy_pilot_first
title: GoalRail Landing Copy — Business-First Пилот ИИ-разработки
kind: product_canon
authority: canonical
status: current
owner: product
truth_surfaces:
  - landing_copy_canon
  - public_demo_governance
  - pilot_positioning
lifecycle: active-core
review_after: 2026-07-19
supersedes: []
superseded_by: null
related_docs:
  - docs/product/GOALRAIL_PRODUCT_CONCEPT.md
  - docs/product/GOALRAIL_OPERATING_MODEL.md
  - docs/ops/STATUS.md
  - docs/ops/NEXT.md
  - docs/ops/DECISIONS.md
  - docs/ops/COMPONENTS.yaml
---
# GoalRail Landing Copy — Business-First Пилот ИИ-разработки

> Status: canonical for the RU public pilot landing at `pilot.goalrail.ru`.
> Implementation: `apps/web/pilot-intake-ru`.
> Current behavior: mostly static business-first landing for a safe 2-week пилот ИИ-разработки on one product area.
>
> Source-of-truth ordering for this surface:
> - this doc (copy + governance)
> - `apps/web/pilot-intake-ru/src/App.tsx` (authoritative implementation)
> - `apps/web/pilot-intake-ru/src/App.css` (visual system)
> - `apps/web/pilot-intake-ru/src/App.test.tsx` (regression net)

---

## A. Product intent

The public RU landing must sell a safe business pilot, not explain GoalRail as
a technical system.

Business question answered by the page:

> Как мне использовать ИИ в разработке и не получить хаос в продукте?

Primary message:

> **ИИ-кодинг без хаоса**

Core promise:

> GoalRail helps product teams safely introduce ИИ-разработку by checking
> repository readiness, collecting project context, creating a controlled task
> process, and running the first supervised pilot on one real product area.

Above-the-fold framing must make clear that this is a safe 2-week pilot for
1 team and 1 product area, not a full AI transformation or a promise to
replace the team's development process.

Business framing:

> Ваши разработчики уже используют ИИ-инструменты. Вопрос уже не в том,
> использовать ли ИИ. Вопрос в том, как не потерять контроль над качеством
> кода, архитектурой и релизами.

Primary offer:

> Запустите безопасный 2-недельный пилот ИИ-разработки на одном участке
> продукта.

## B. Positioning rules

Tone:
- calm;
- adult;
- business-friendly;
- no ИИ hype;
- no `10x speed` promise;
- no fully autonomous ИИ agent promise;
- no deep technical architecture in the hero.

The page should preserve:
- existing dark GoalRail palette;
- premium technical mood;
- muted panels;
- amber primary CTA;
- violet accents;
- IBM Plex fonts;
- inline email submission through the narrow D-0056 lead endpoint, with
  D-0059 Resend HTTPS transport for notification delivery when configured and
  `mailto:hello@goalrail.dev` as a fallback.

The page must remove or demote:
- `Goal Intake` as hero;
- 5-step interactive walkthrough as the main page object;
- Clarification / Contract Draft / Review / Honest Outcome as public
  first-screen flow;
- technical contract-object UI;
- deep internal system terms;
- agent execution language;
- YAML/JSON examples;
- terminal-heavy first screen.

## C. Required landing sections

### 1. Hero

Hero title:

> ИИ-кодинг без хаоса

Hero body:

> Ваши разработчики уже используют Cursor, Claude Code, Codex или Copilot.
> GoalRail помогает внедрить ИИ-разработку как управляемый процесс: с аудитом
> репозитория, проектным контекстом, понятными задачами и проверяемым
> результатом.

Supporting line:

> Запустите безопасный 2-недельный пилот на одном участке продукта — не
> полную ИИ-трансформацию, а управляемую проверку качества, архитектуры и
> релизного контроля.

Hero microcopy near CTA:

> 2 недели · 1 команда · 1 участок продукта · без перестройки процесса

Primary CTA:

> Обсудить пилот

CTA behavior:
- focus the existing email/contact area;
- submit only to the same-origin `POST /api/pilot-lead` endpoint allowed by
  D-0056;
- no analytics;
- no external network call from the browser.

Hero right card:
- visible header label: `Пример пилотного отчёта`
- title: `Пилот ИИ-разработки`
- fields:
  - `Готовность репозитория: 74 / 100`
  - `Участок пилота: Внутренний админ-модуль`
  - `Найдено рисков: 5`
  - `Задачи под контролем: Готово`
- next step: `Запустить первый сценарий под контролем`
- disclaimer: `Иллюстрация результата. Реальный аудит проводится отдельно.`

The readiness value is illustrative only. It must not imply a real repository
scan.

### 2. Problem

Title:

> ИИ ускоряет разработку. Но без процесса он быстро создает скрытый хаос.

Copy:

> На старте все выглядит хорошо: разработчики быстрее пишут код, быстрее
> собирают прототипы и закрывают задачи. Проблема появляется позже: ИИ
> предлагает разные решения для похожих задач, дублирует логику, обходит
> архитектурные ограничения и меняет код без понятного объяснения.

Bullets:
- архитектура начинает разъезжаться;
- одинаковая логика появляется в разных местах;
- Пул-реквесты выглядят рабочими, но их сложнее объяснить;
- тесты не покрывают реальные риски;
- разработчики используют ИИ по-разному;
- бизнес видит скорость, но не видит риски.

### 3. Control layer framing

Title:

> Проблема не в ИИ-инструментах. Проблема в отсутствии слоя контроля.

Copy:

> Cursor, Claude Code, Codex и Copilot уже помогают разработчикам писать код.
> Но они не отвечают на вопросы бизнеса: готов ли репозиторий к
> ИИ-разработке, какие участки продукта безопасно отдавать в пилот, какие
> ограничения должен соблюдать ИИ, как проверить результат и как понять, что
> команда не накапливает скрытый технический долг.

Closing line:

> GoalRail добавляет этот слой контроля поверх ИИ-инструментов.

Simple visual explanation:

1. `AI-инструменты` — `Cursor · Claude Code · Codex · Copilot`
2. `GoalRail control layer` — `аудит · контекст · правила задач · проверка`
3. `Безопасный пилот` — `один участок продукта · контролируемые AI-задачи · проверяемый результат`
4. `Решение для бизнеса` — `что масштабировать · что исправить · где AI пока рискован`

This visual must stay a compact business explanation, not a technical
architecture diagram, execution engine, terminal, code block, YAML, JSON, or
agent log.

### 4. What GoalRail does

Title:

> GoalRail ставит ИИ-разработку на рельсы

Four cards:
- `Аудит готовности` — Проверяем, насколько ваш репозиторий готов к работе с
  ИИ-инструментами: структура, тесты, документация, контекст, риски и
  ограничения.
- `Контекст проекта` — Собираем базу знаний о проекте, чтобы ИИ и
  разработчики работали не вслепую, а с пониманием архитектуры и правил
  команды.
- `Контролируемые задачи` — Каждая ИИ-задача проходит через понятную рамку:
  цель, границы изменений, ограничения, проверки и ожидаемый результат.
- `Проверяемый результат` — Команда получает не просто сгенерированный код,
  а прозрачный процесс: что было сделано, почему так, как проверено и какие
  риски остались.

### 5. Pilot offer

Title:

> Начните с безопасного пилота, а не с большого внедрения

Copy:

> Мы не предлагаем сразу перестраивать всю разработку. GoalRail Founding Pilot
> запускается на одном выбранном участке продукта: модуле, внутреннем сервисе,
> админ-панели, интеграции или другой зоне, где можно проверить подход без
> риска для критичного ядра.

Repeatable-process line:

> Пилот проводится по повторяемому процессу GoalRail, а не как разовый
> консалтинг-проект.

List:
1. выбор подходящего участка продукта;
2. аудит репозитория;
3. оценка готовности к ИИ;
4. список рисков и блокеров;
5. настройка проектного контекста;
6. введение процесса ИИ-задач;
7. запуск первого реального рабочего сценария;
8. финальный отчет: что масштабировать, что исправить, где ИИ пока рискован.

Important phrase:

> Цель пилота — не доказать, что ИИ может писать код. Цель — понять, как
> вашей команде использовать ИИ безопасно, системно и предсказуемо.

### 6. Business demo cards

Title:

> Как выглядит пилотный результат

This section shows three illustrative cards, not an interactive technical demo.
Mark them as examples / illustrative, not real результаты реального сканирования.

Cards:
- `Готовность репозитория` with `Оценка готовности: 74 / 100`, ready items, and
  risks.
- `Контролируемая AI-задача` with task, goal, boundaries, and checks.
- `Результат пилота` with result, evidence, and recommendation.

### 7. For whom / not for whom

Title:

> Для команд, которые уже пробуют ИИ-разработку

Fits:
- у вас продуктовая команда от 3 до 20 разработчиков;
- разработчики уже используют Cursor, Claude Code, Codex, Copilot или похожие
  инструменты;
- вы хотите использовать ИИ активнее, но боитесь пускать его в критичный код;
- у вас есть участок продукта, на котором можно провести безопасный пилот;
- вы хотите не просто скорость, а контроль над качеством и архитектурой.

Not a fit:
- вы ждете полной автономной разработки без людей;
- вы хотите обещание десятикратной скорости;
- у вас огромный старый монолит и вы хотите сразу внедрить ИИ во все;
- команда еще вообще не использует ИИ-инструменты;
- вам нужен кастомный консалтинг без повторяемого процесса.

### 8. До и после

Title:

> Что меняется после пилота

Before:
- ИИ используется каждым разработчиком по-своему;
- неясно, какие части кода безопасны для ИИ;
- контекст проекта живет в головах людей;
- задачи формулируются промптами без общих правил;
- результат проверяется вручную и не всегда системно;
- бизнес видит скорость, но не видит риски.

After:
- понятно, где ИИ можно использовать безопасно;
- есть оценка готовности репозитория;
- есть база знаний проекта;
- ИИ-задачи проходят через общий процесс;
- результаты проверяются по понятным критериям;
- бизнес видит не только скорость, но и управляемость.

### 9. Частые вопросы

Required questions:
- Это заменяет разработчиков?
- Это альтернатива Cursor или Claude Code?
- Вы ускоряете разработку?
- Можно внедрить сразу во весь продукт?
- Что нужно от команды?

Answers must be concise and aligned with the business positioning:
- no developer replacement;
- not an alternative to Cursor / Claude Code / Codex / Copilot;
- speed may improve, but control is the primary promise;
- do not start with whole-product rollout;
- the team must provide an engineering owner, repository or representative
  codebase access, a safe pilot area, and one real workflow to test.

### 10. Final CTA

Title:

> Проверьте, готова ли ваша команда к AI-разработке

Copy:

> Начните с ограниченного пилота. Мы поможем выбрать безопасный участок
> продукта, провести аудит и запустить первый управляемый рабочий сценарий
> ИИ-разработки.

Primary CTA:

> Обсудить пилот

Email/contact area:
- keep `hello@goalrail.dev`;
- primary path: email field submits to same-origin `POST /api/pilot-lead`;
- keep direct `mailto:hello@goalrail.dev` fallback;
- use honest microcopy such as `Без рассылок, трекинга, CRM и автоматической
  воронки. Если форма не сработает, напишите напрямую.`;
- success copy: `Спасибо. Получили почту — вернёмся с коротким следующим
  шагом.`;
- duplicate copy: `Этот адрес уже есть в списке. Повторно заявку не
  отправляем, чтобы не дублировать письма. Мы вернёмся с коротким следующим
  шагом.`;
- error copy: `Не удалось отправить заявку. Напишите напрямую:
  hello@goalrail.dev`;
- no analytics.

## D. Firm boundaries

This landing must not, without an explicit superseding decision:

- call any LLM or ИИ API;
- connect to any repo provider;
- execute code or run any runner;
- accept file uploads;
- send analytics/session/tracking events;
- render chat history;
- show a model selector;
- claim that a real repo was scanned;
- claim that a result was delivered;
- claim fully autonomous development.

Allowed:
- static illustrative cards;
- illustrative readiness score only with explicit disclaimer;
- local button behavior that focuses the email/contact field;
- `POST /api/pilot-lead` as the only browser `fetch`, limited to email lead
  capture under D-0056;
- `mailto:` fallback handoff.

D-0047 remains in force for no-analytics / no-real-scan / no-runtime /
no-repo / no-LLM boundaries except for the narrow D-0056 lead-capture
exception. D-0055 records that the business-first Founding Pilot landing
supersedes the technical interactive walkthrough as the primary public RU
landing. D-0056 allows only `POST /api/pilot-lead` on the operator-managed RU
server: validate email, write local JSONL lead state with notification status,
attempt notification, mark successful notifications as `notified`, and keep
`notification_failed` rows retryable. In-flight `received` / `pending` rows are
not treated as fresh retries, so near-simultaneous submissions do not start
concurrent duplicate notifications. Duplicate suppression applies to
successfully notified addresses and legacy rows without `notification_status`,
which are treated conservatively as already processed. D-0057 allows a
server-local direct recipient override at
`/srv/goalrail/pilot/backend/lead-recipient.local`; if it is absent, the
endpoint falls back to `hello@goalrail.dev`. The public/manual contact email
remains `hello@goalrail.dev`. Browser-facing mail errors stay generic as
`mail_unavailable`. These decisions do not approve analytics, tracking,
cookies, sessions, CRM, Google Sheets, user accounts, LLM/API calls, repo
integration, runtime execution, or a broad backend platform.

## E. Technical walkthrough demotion

The previous 5-step local technical walkthrough is no longer the primary public
RU landing shape.

Status:
- keep it in git history as the checkpointed technical demo;
- do not copy it into a duplicate app folder by default;
- treat it as an internal / technical demo checkpoint unless a future bounded
  slice explicitly restores or moves it.

## F. Metadata

Active public URL:

> `https://pilot.goalrail.ru/`

`apps/web/pilot-intake-ru/index.html` canonical must remain:

> `https://pilot.goalrail.ru/`

## G. Implementation pointer

- App: `apps/web/pilot-intake-ru`
- Main code: `apps/web/pilot-intake-ru/src/App.tsx`
- Styles: `apps/web/pilot-intake-ru/src/App.css`
- Tests: `apps/web/pilot-intake-ru/src/App.test.tsx`
- Lead endpoint/digest sidecar: `apps/web/pilot-intake-ru/server` (Go,
  landing-owned, not the core `apps/server` product API)

Quality gates:
- typecheck: `npm run pilot-intake-ru:typecheck` from `apps/web`
- tests: `npm run pilot-intake-ru:test` from `apps/web`
- build: `npm run pilot-intake-ru:build` from `apps/web`
- preview: app workspace exposes `npm run preview --workspace @goalrail/pilot-intake-ru-web`

This doc is the canonical product/copy/governance reference for the RU
business-first pilot landing. Any server behavior beyond the D-0056
`POST /api/pilot-lead` exception, analytics, deployment mode change,
third-party lead sink, repo integration, runtime execution, or autonomous-agent
claim requires a separate decision. D-0058 allows only a server-local daily
lead digest from the existing JSONL log at 07:00 GMT+3 for the previous local
calendar day; empty days send no digest. D-0059 allows only a narrow Resend
HTTPS transactional email transport for these lead notifications/digests, using
`skill7.dev` as the configured sending domain and a server-local API key. The
lead endpoint may record `notification_status`, `notification_attempted_at`,
`notification_updated_at`, successful `notification_transport`, and generic
`notification_error: "mail_unavailable"` in the local JSONL backup log; it must
keep `notification_failed` retryable while treating `received` / `pending` as
in-flight for new submissions, and must not expose transport exception details
to the browser.
