---
id: goalrail_landing_copy_pilot_first
title: GoalRail Landing Copy — Pilot-First Interactive Demo
kind: product_canon
authority: canonical
status: current
owner: product
truth_surfaces:
  - landing_copy_canon
  - public_demo_governance
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
# GoalRail Landing Copy — Pilot-First Interactive Demo

> Status: canonical for the RU public pilot-first landing demo.
> Implementation: `apps/web/pilot-intake-ru`.
> Current behavior: local deterministic 5-step walkthrough.
>
> Source-of-truth ordering for this surface:
> - this doc (copy + governance)
> - `apps/web/pilot-intake-ru/src/App.tsx` (authoritative implementation)
> - `apps/web/pilot-intake-ru/src/App.css` (visual system)
> - `apps/web/pilot-intake-ru/src/App.test.tsx` (regression net)
>
> If this doc and the code disagree, the code is the implementation truth, but
> the boundaries below must hold.

---

## A. Product intent

The landing lets a visitor describe a task, PR, change, or case and then walks
them through a local simulation of GoalRail's contract-first workflow. The goal
is to show the **method**, not to execute work.

What it is:
- a public product-led landing/demo surface;
- a guided contract walkthrough: raw task → clarification → contract draft →
  review/gaps → honest outcome;
- a calm, technical, bounded demonstration of how GoalRail thinks about
  AI-assisted delivery.

What it is **not**:
- a generic AI assistant;
- a chat demo;
- a live execution environment;
- a place where a real repo is assessed;
- a place where a result is "delivered".

---

## B. Core promise

Public promise:

> **Покажите задачу — мы проведём вас по контракту**

Meaning:
- user brings a real-ish task;
- GoalRail shows how it becomes a bounded contract;
- clarification comes before execution;
- risk and ambiguity are visible;
- the outcome is honest — including the case where a pilot is not yet ready.

Hero subtitle (canonical):

> Опишите реальную задачу, PR, изменение или кейс. Демо покажет, как GoalRail
> превращает её в контракт, уточняющие вопросы, критерии и честный следующий
> шаг — без выполнения кода и подключения репозитория.

---

## C. Firm boundaries

The landing demo must not, without an explicit superseding decision:

- call any backend;
- persist user input anywhere (memory, storage, network);
- call any LLM or AI API;
- connect to any repo provider (GitHub/GitLab/Bitbucket);
- execute code or run any runner;
- accept file uploads;
- send analytics/session/tracking events;
- render chat history;
- show a model selector;
- show user/assistant turns;
- show avatars;
- claim that the user's task was completed;
- claim that a real repo was assessed;
- show a numeric readiness "score" as if it were real measurement.

Final CTA contract:
- the primary outcome CTA only **focuses** the existing email input
  (`#pilot-email`) inside the lead block and applies a temporary local
  highlight class (`ctaCard--highlight`).
- it does not POST anything anywhere.
- the email lead remains a `mailto:hello@goalrail.dev` form;
  any future backend handoff requires its own decision.

These boundaries are recorded in `docs/ops/DECISIONS.md` (D-0047).

---

## D. 5-step walkthrough

The local state machine has exactly five `demoStep` values:
`intake → clarification → contract → review → outcome`.

### Step 1 — Goal Intake / Запрос

- Purpose: capture the user's raw task description as plain text.
- User-visible: hero, eyebrow `ПИЛОТНЫЙ ФОРМАТ`, composer textarea, safety
  microcopy, three example chips, primary CTA `Запустить демо`.
- CTA enabled when trimmed length is 20–1500 characters.
- Local/deterministic: textarea is a controlled input; `detectScenario` runs
  on submit.
- Must not imply: that pasting a real PR sends it anywhere.

### Step 2 — Clarification / Уточнения

- Purpose: surface 3 deterministic questions to scope the contract.
- User-visible: panel `CLARIFICATION · Шаг 2 из 5`, intake summary `ЗАПРОС
  ПРИНЯТ` with the original text rendered as text only, three radiogroups,
  progress `Ответы: N / 3`, primary CTA `Подготовить контракт`.
- CTA enabled only when all three questions are answered.
- Local/deterministic: questions and option labels are constants per scenario.
- Must not imply: that GoalRail is asking the user using an LLM.

### Step 3 — Contract Draft / Контракт

- Purpose: render a bounded structured contract preview.
- User-visible: panel `CONTRACT DRAFT · Шаг 3 из 5`, structured `<dl>` with
  rows (Название / Цель / Scope / Правило review / Failure mode / Вне scope /
  Критерии приёмки / Open ambiguity / Следующий шаг). Acceptance criteria are
  numbered AC-NN. Open ambiguities are amber-tinted cards with ID, text and
  `требует решения` badge.
- Primary CTA: `Перейти к проверке`. Secondary: `Вернуться к уточнениям`.
  Tertiary: `Изменить запрос`.
- Local/deterministic: `buildContractDraft({scenario, answers, intakeText})`.
- Must not imply: that this is an AI-generated proposal or that the contract
  is approved/binding.

### Step 4 — Review / Проверка

- Purpose: structured local check of the contract draft.
- User-visible: panel `REVIEW · Шаг 4 из 5`. Three sections — `Готовность к
  следующему шагу` (readiness items with status badges `ГОТОВО`/`ОГОВОРКА`/
  `БЛОКЕР`), `Риски и ambiguity` (R-NN risk cards with severity badges
  `СПРАВКА`/`ОГОВОРКА`/`БЛОКЕР` plus the inherited ambiguity list),
  `Проверочный вывод` with deterministic conclusion text.
- Primary CTA: `Подготовить итог`. Secondary: `Вернуться к черновику`.
- Local/deterministic: `buildReviewReport({scenario, answers, draft})`.
- Must not imply: that this is a real readiness scan or runtime verification.
  The `Demo execution is simulated` advisory risk is always present.

### Step 5 — Honest Outcome / Итог

- Purpose: a final structured verdict about whether this case looks like a
  good first pilot candidate.
- User-visible: panel `OUTCOME · Шаг 5 из 5`. Three mini-digests (intake,
  contract, review) with counter pills (Блокеры / Оговорки / Открытых
  ambiguity). One dominant `verdictCard` keyed by tone with label, title,
  body, and recommended next step. Four sections — `Что стало ясно`, `Что
  осталось открытым`, `Что демо не делало`, `Следующий шаг`.
- Primary CTA: `outcomeReport.ctaLabel` (varies by tone — see section H).
  Clicking it focuses the email input and adds the `ctaCard--highlight`
  state for ~2.4s. No submission.
- Secondary: `Вернуться к проверке`. Tertiary: `Начать заново`.
- Local/deterministic: `buildOutcomeReport({scenario, draft, review})` plus
  `deriveOutcomeTone(review)`.
- Must not imply: that GoalRail has actually committed to anything, that the
  task was completed, or that a real assessment was performed.

---

## E. Canonical copy

Hero (intake):

- Eyebrow: `ПИЛОТНЫЙ ФОРМАТ`
- Title: `Покажите задачу — мы проведём вас по контракту`
- Subtitle: `Опишите реальную задачу, PR, изменение или кейс. Демо покажет,
  как GoalRail превращает её в контракт, уточняющие вопросы, критерии и
  честный следующий шаг — без выполнения кода и подключения репозитория.`

Composer (intake):

- Header label: `GOAL INTAKE · Шаг 1 из 5`
- Stage marker: `→ Уточнения`
- Placeholder: `Опишите задачу, PR, изменение или кейс…`
- Safety microcopy (referenced by `aria-describedby` on the textarea):
  `Не вставляйте секреты, токены и приватные данные.`
- Counter format: `<N> / 1500`
- Primary CTA: `Запустить демо`

Example chips (canonical 3):

- `Добавить manual review перед proof approval`
- `Разобрать PR с неясными критериями приёмки`
- `Оценить готовность repo к AI-assisted delivery`

Top status chips (3 fixed):

- `ПИЛОТ ОТКРЫТ`
- `РУЧНОЙ ФОРМАТ`
- `РЕАЛЬНЫЙ КЕЙС`

5-step rail labels (in order):

1. `Запрос`
2. `Уточнения`
3. `Контракт`
4. `Проверка`
5. `Итог`

Context strip rows (per step):

- intake idle: `демо-кейс / без подключения repo / ручной пилот / оценим
  после описания`
- intake valid: `demo-contract / без подключения repo / ручной пилот /
  демо-оценка появится дальше`
- clarification: `demo-contract / без подключения repo / clarification /
  появится после контракта`
- contract: `demo-contract / без подключения repo / contract-draft /
  появится после проверки`
- review: `demo-contract / без подключения repo / review / локальная
  демо-проверка`
- outcome: `demo-contract / без подключения repo / outcome / метод
  показан`

Right-column cards (per step):

- always visible: `Что вы увидите`, `Безопасно для демо`, `Итог пилота`.
- step 2 prepends: `Зачем уточнения`.
- step 3 prepends: `Что в черновике`.
- step 4 prepends: `Что проверяется`.
- step 5 prepends: `Что в итоге`.

Safety card (`Безопасно для демо`) items — must remain visible everywhere:

- `код не выполняется`
- `repo не подключается`
- `данные не сохраняются как production-сущности`
- `изменений в вашей среде не происходит`

Outcome `notDone` list (canonical 4 items, must remain explicit):

1. `код не выполнялся`
2. `repo не подключался`
3. `production-сущности не создавались`
4. `результат не является выполненной задачей`

---

## F. Scenario model

Two local scenarios, picked deterministically by `detectScenario(text)`:

### 1. `manual_review_gate`

- Intent: tasks about review/approval/proof gates.
- Detection: lowercase-substring match on any of:
  `review`, `reviewer`, `approval`, `approve`, `proof`, `gate`, `manual`,
  `провер`, `ревью`, `аппрув`, `соглас`, `утвержд`.
- Clarification questions (3, fixed):
  - `Где должен применяться review gate?` —
    `только новые контракты` / `новые и активные контракты` /
    `только repo-scoped контракты` / `пока не уверен`
  - `Кто должен принимать review decision?` —
    `один назначенный reviewer` / `любой operator` /
    `quorum / два человека` / `пока не определено`
  - `Что должно произойти, если review не пройден?` —
    `proof блокируется` / `контракт возвращается в clarification` /
    `execution останавливается` / `нужно решить вручную`
- Contract draft: title `Ручной review gate перед proof approval`, fixed
  outOfScope and acceptanceCriteria, scope text and reviewer/failure-mode
  rows derived from answers.
- Review: 4 readiness items (Scope зафиксирован / Review decision rule
  выбран / Failure mode определён / Out of scope отделён). Risks include
  `Нужна политика для активных контрактов`, `Владелец решения по review не
  определён`, `Путь ручного решения не описан` based on answers, plus the
  always-on advisory `Выполнение в демо симулируется`.
- Outcome: tone derived from review (see H).

### 2. `bounded_task`

- Intent: any task that is not picked up as `manual_review_gate`.
- Detection: fallback when no review-gate signal matches.
- Clarification questions (3, fixed):
  - `Какая граница задачи?` —
    `один repo / один кейс` / `несколько частей продукта` /
    `процесс всей команды` / `пока не уверен`
  - `Что должно быть видно в контракте?` —
    `критерии приёмки` / `риски и ambiguity` /
    `proof / проверка результата` / `всё перечисленное`
  - `Какой честный итог нужен?` —
    `можно запускать пилот` / `нужно уточнить scope` /
    `кейс пока не подходит` / `хочу увидеть риски`
- Contract draft: title `Черновик рабочего контракта`, fixed outOfScope and
  acceptanceCriteria, scope text from selected boundary, no reviewer/failure
  rows.
- Review: 4 readiness items including `Что должно быть видно в контракте`
  flagged warning if `всё перечисленное`. Risks include `Scope может быть
  слишком широким`, `Scope процесса команды слишком широкий`, `Scope не
  определён`, `Контракт может быть перегружен` based on answers, plus the
  always-on advisory `Выполнение в демо симулируется`.
- Outcome: tone derived from review (see H).

Governance:
- new scenarios are not added without an explicit decision.
- existing scenarios can have their copy refined; they cannot be expanded
  into network-driven behavior.

---

## G. Pure local logic

The demo is built from five pure functions plus React state. No effects
except the local cleanup timeout for the email-CTA highlight.

| Helper | Input | Output | Behavior |
|--------|-------|--------|----------|
| `detectScenario(text)` | trimmed user text | `Scenario` | lowercase substring match |
| `buildContractDraft({scenario, answers, intakeText})` | local state | `ContractDraft` | builds title/scope/criteria/ambiguities deterministically |
| `buildReviewReport({scenario, answers, draft})` | scenario + answers + draft | `ReviewReport` | builds readiness items + risks + readiness label/summary/nextStep |
| `deriveOutcomeTone(review)` | review report | `OutcomeTone` | structural derivation (see H) |
| `buildOutcomeReport({scenario, draft, review})` | scenario + draft + review | `OutcomeReport` | builds verdict, learned items, remains-open list, next step, ctaLabel |

Boundary:
- these helpers do not read any global state outside their arguments;
- they do not call `fetch`, `XMLHttpRequest`, or any persistence API;
- they do not mutate input;
- they are intended to demonstrate the GoalRail method and remain
  inspectable line by line.

---

## H. Outcome tones

Three tones, derived from structured review data only — **never** from a
substring match on the Russian readiness label.

`deriveOutcomeTone(review)`:

1. If any `readinessItem.status === "blocking"` or any
   `riskItem.severity === "blocking"` → `blocked`.
2. Else if any `readinessItem.status === "warning"` or any
   `riskItem.severity === "warning"` or
   `review.ambiguityItems.length > 0` → `readyWithCaveats`.
3. Else → `ready`.

| Tone | Label | Title | CTA label |
|------|-------|-------|-----------|
| `ready` | `ГОТОВ К ПИЛОТУ` | Кейс подходит для короткого пилота | `Обсудить пилот` |
| `readyWithCaveats` | `ПОДХОДИТ С ОГОВОРКАМИ` | Кейс можно брать в пилот, но с явными условиями | `Обсудить оговорки` |
| `blocked` | `СНАЧАЛА НУЖНЫ РЕШЕНИЯ` | Кейс пока не готов к пилоту | `Разобрать кейс` |

Tone meaning:
- `ready` — good candidate for short manual pilot.
- `readyWithCaveats` — useful case, but risks/ambiguity must be fixed or
  acknowledged before starting.
- `blocked` — case is not ready; this is a useful honest output, not a demo
  failure. The verdict copy is explicit:
  *«Это не провал демо: GoalRail должен показать, где рано обещать
  результат.»*

Color rules:
- olive — only for ready/clear states (status `ok`, ready verdict tone).
- amber — for warnings, caveats, open ambiguity.
- rust (`--blocking: #c47156`) — for blocking states. Strong but not
  alarming; it is not "production failure" red.
- there is no fake numeric readiness score anywhere on the page.

---

## I. Accessibility and UX hardening

The implementation through Phase 6C ships:

- skip link `<a href="#main-content">К основному содержимому</a>` revealed
  on focus, with explicit `onClick` that calls `target.focus()` and
  `scrollIntoView`.
- main landmark with `id="main-content"`, `tabIndex={-1}`, and
  `aria-label="Демо GoalRail"` so the skip link can land focus and screen
  readers can name the page region.
- sidebar `<aside aria-label="Разделы лендинга">` and right-column
  `<aside aria-label="О демо">` as labelled complementary landmarks.
- composer textarea `aria-describedby="demo-intake-safety"` referencing the
  safety microcopy span.
- 5-step rail with `aria-current="step"` on the active step and
  `data-step-state` attribute (`done` / `active` / `muted`) for non-color
  state cues.
- structured radiogroups for clarification options with `aria-labelledby`
  pointing at each question prompt.
- structured `aria-label`'d counter spans for the review digest in step 5.
- horizontal-scroll demo rail at narrow widths with CSS `scroll-snap-type`
  and a right-edge `mask-image` scroll-shadow, plus a compact <420px
  breakpoint.
- subtle 140ms CSS-only fade-in on step blocks, gated by
  `prefers-reduced-motion: reduce` and by `data-skip-animation="true"` for
  back-navigation.
- outcome primary CTA focuses the email input and applies a temporary
  `ctaCard--highlight` state, cleaned up with a guarded `setTimeout` and
  unmount cleanup.

---

## J. Future extension rules

May be considered later, only after explicit decision:

- additional scenarios (beyond `manual_review_gate` and `bounded_task`);
- hash-based shareable local demo state (e.g. `#/clarification?case=…`);
- a docs-to-code copy extraction layer that pulls canonical strings from
  this doc instead of from `App.tsx` constants;
- visual regression baseline (e.g. Playwright snapshots);
- localization mirror (EN side-by-side with RU);
- analytics — only after a separate decision that explicitly defines what
  is collected, why, and how the user is informed.

Must not happen without a new explicit decision (see D-0047):

- backend submission of intake or email;
- LLM or AI API call;
- repo provider integration;
- real execution / runner / sandbox;
- persistence of user input;
- analytics or session tracking of any kind;
- chat-style UI (history, turns, avatars, model selector).

---

## K. Implementation pointer

- App: `apps/web/pilot-intake-ru`
- Main code: `apps/web/pilot-intake-ru/src/App.tsx`
- Styles: `apps/web/pilot-intake-ru/src/App.css`
- Tests: `apps/web/pilot-intake-ru/src/App.test.tsx`

Quality gates as of this docs update:
- typecheck: `npm run pilot-intake-ru:typecheck` (passes)
- tests: `npm run pilot-intake-ru:test` (full walkthrough plus
  Phase 6A/6B/6C polish coverage; previously reported as 67 tests)
- build: `npm run pilot-intake-ru:build` (passes)

This doc is the canonical product/copy/governance reference for the RU
pilot-first interactive landing demo. Any change that contradicts the
boundaries in section C requires a new entry in `docs/ops/DECISIONS.md`.
