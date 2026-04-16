# Goalrail Qualification Checklist

> Canonical checklist for early Goalrail qualification calls.
>
> Purpose:
> quickly determine whether a prospect is a fit for the standard Goalrail pilot model.
>
> Read together with:
> - `docs/product/GOALRAIL_ICP.md`
> - `docs/product/GOALRAIL_OFFER.md`
> - `docs/product/GOALRAIL_PILOT_MODEL.md`
> - `docs/product/GOALRAIL_DEPLOYMENT_MODEL.md`
> - `docs/product/GOALRAIL_EMAIL_SEQUENCE.md`

## 1. Purpose

Этот документ нужен для короткого qualification / fit-check разговора.

Его задача:
- не уходить в длинный аудит
- быстро отсеивать no-fit cases
- подтверждать, что у клиента есть нормальный pilot candidate
- готовить переход к paid pilot

Qualification call не должен превращаться в:
- большой discovery project
- process consulting session
- architecture deep dive
- product demo без решения о следующем шаге

## 2. Qualification principle

Главный вопрос qualification:

**Есть ли у этой команды нормальный pilot-fit для стандартного Goalrail deployment?**

Не:
- “как идеально устроить весь их delivery process”

А:
- можно ли провести один реальный кейс через:
  `incoming task -> working contract -> bounded execution -> verify -> proof`

## 3. Recommended call shape

### Duration
- 20–25 minutes recommended
- 30 minutes max by default

### Participants
Best case:
- CTO / Head of Engineering / VP Engineering sponsor
- tech lead or engineering owner

Optional:
- PM / product owner if the task flow discussion requires it

## 4. Output of a good qualification call

В конце разговора должен быть один из трёх исходов:

1. **Go to pilot framing**
2. **Need one missing condition before pilot**
3. **No-fit for now**

Нельзя заканчивать call состоянием:
- “интересно, когда-нибудь вернёмся”
- “надо ещё поговорить без следующего шага”

## 5. Checklist sections

### Section A — Team fit

#### A1. Is there one real pilot team?
- Да / Нет
- Название или тип команды:
- Размер команды:

#### A2. Is the team in the best-fit range?
Recommended:
- 5–30 engineers

Mark:
- Best fit
- Acceptable fit
- Weak fit

#### A3. Is there a real engineering owner for the pilot?
- Да / Нет
- Кто именно:

### Section B — Sponsor fit

#### B1. Is there a sponsor with decision authority?
Recommended:
- CTO
- Head of Engineering
- VP Engineering

- Да / Нет
- Кто именно:

#### B2. Can this sponsor approve a bounded pilot?
- Да / Нет
- Неясно

### Section C — Workflow fit

#### C1. Does the company have a real delivery flow already?
Check:
- задачи реально приходят в команду
- есть engineering execution loop
- есть результат, который можно проверить

Mark:
- Yes, clear delivery flow
- Partial / weak flow
- No real flow

#### C2. Is there visible ambiguity between task and result?
Signals:
- vague requests
- weak scope definition
- weak handoff between PM and dev
- unclear review expectations
- AI activity without strong proof

Mark:
- Strong pain
- Moderate pain
- Weak pain

### Section D — Pilot case fit

#### D1. Is there one real case for the pilot?
- Да / Нет
- Кейс:

#### D2. Is the case bounded enough?
Good pilot case should be:
- real
- valuable
- not too broad
- not too politically risky
- possible to evaluate after 1 run

Mark:
- Good pilot case
- Possible but needs narrowing
- Bad pilot case

#### D3. Is there one repo or one clear workflow path?
- Да / Нет
- Какой repo / path:

### Section E — AI readiness fit

#### E1. Does the team already use AI in development?
Mark:
- Yes, already using it
- Not systematically, but interested
- No, and no real urgency

#### E2. Is there pressure to adopt AI without losing control?
- Strong
- Medium
- Weak

### Section F — Runtime and deployment fit

#### F1. Can they support a bounded pilot deployment?
Need at minimum:
- one team
- one repo or workflow path
- one real case
- readiness to try contract-first flow

Mark:
- Yes
- With one missing condition
- No

#### F2. Is remote-first acceptable?
- Yes
- Maybe
- No

#### F3. Are there obvious blockers for Codex / Claude Code style runtime usage?
- No blocker
- Restricted but workable
- Major blocker

### Section G — Security / policy fit

#### G1. Are there security restrictions that block even a bounded pilot?
- No
- Manageable
- Severe blocker

#### G2. If restricted, can a narrower profile still work?
Examples:
- single-vendor only
- local-only later
- limited scope exposure

- Yes
- Maybe
- No

## 6. Qualification scoring recommendation

Use a simple three-level decision:

### Green — Good fit
Usually means:
- sponsor exists
- team exists
- real case exists
- repo/workflow path exists
- pain is visible
- bounded pilot is feasible

### Yellow — Possible fit, but one missing condition
Typical reasons:
- no clear pilot case yet
- sponsor not confirmed
- scope too broad
- security concerns need clarification

### Red — No-fit for now
Typical reasons:
- no real team
- no sponsor
- no delivery flow
- wants full autopilot, not bounded pilot
- pilot impossible under current constraints

## 7. Recommended closing decision

### If Green
Say:
- there is a clear pilot fit
- next step is pilot framing / paid pilot proposal

### If Yellow
Say:
- fit is possible
- one missing condition must be resolved first
- define exactly what is missing

### If Red
Say:
- Goalrail is probably not the right fit for a pilot right now
- do not force the sale

## 8. Recommended call script

### Opening
“Цель разговора — не сделать длинный аудит, а быстро понять, есть ли у вас нормальный pilot-fit для Goalrail.”

### Middle
“Нам важно понять четыре вещи: есть ли команда, sponsor, реальный кейс и возможность провести bounded workflow до proof.”

### Closing
“По итогам у нас должен быть один честный вывод: go to pilot, one missing condition, or no-fit for now.”

## 9. What to capture in notes

At minimum record:
- company
- participant names and roles
- sponsor name
- team name / size
- candidate pilot case
- repo / workflow path
- main pain signal
- blockers
- final qualification verdict
- next step

## 10. Not a demo rule

Qualification call should not become:
- a broad product tour
- a technical deep dive
- a custom consulting session

Only show as much product framing as needed to help the client understand:
- what Goalrail is
- how the pilot works
- what would happen next

## 11. Recommended next-step matrix

### Outcome: Green
Next step:
- send pilot framing
- send offer / deck if needed
- propose paid pilot

### Outcome: Yellow
Next step:
- request one missing input
- clarify scope / sponsor / repo / security
- do not jump into pilot yet

### Outcome: Red
Next step:
- stop politely
- optionally leave door open for later

## 12. One-line summary

**A qualification call is successful when it produces a clear pilot decision, not when it produces a long discussion.**
