# Goalrail ICP

## 1. Purpose

Этот документ фиксирует, каким командам Goalrail подходит на старте, а каким — нет.

## 2. Primary ICP

### Team shape
- product teams with 5–30 engineers
- есть PM / analyst / tech lead structure
- есть sponsor со стороны CTO / Head of Engineering / VP Engineering

### Workflow shape
- есть регулярный поток задач
- есть 1–2 репо, на которых можно провести pilot
- уже есть engineering delivery loop, но он не полностью управляем при AI usage

### AI readiness
Подходит как для команд, которые:
- уже используют AI хаотично и хотят это упорядочить
так и для команд, которые:
- ещё не используют AI системно, но хотят запустить первый controlled workflow

## 3. Best-fit signals

Лучший fit, если:

- команда уже чувствует ambiguity между PM и dev
- AI usage уже начался или обсуждается всерьёз
- есть pain around handoff, scope and trust
- есть желание не заменить весь стек, а добавить control layer
- leadership готова дать pilot sponsor
- команда готова попробовать новый contract-first workflow

## 4. Acceptable first segments

- B2B SaaS product teams
- internal product/platform teams
- engineering studios
- software teams inside larger businesses

## 5. No-fit signals

Плохой fit на старте:

- нет sponsor
- нет команды, только один индивидуальный пользователь
- нет реального delivery flow
- нет repo / workflow, на котором можно сделать pilot
- ожидание “полного автопилота”
- ожидание полной замены Jira / Linear / IDE
- extreme security posture, который делает pilot невозможным без сложного enterprise layer

## 6. Fit-check questions

1. Есть ли одна реальная команда под pilot?
2. Есть ли один sponsor, который примет решение?
3. Есть ли 1–2 репо или реальные кейсы?
4. Есть ли pressure to adopt AI without losing control?
5. Готова ли команда пройти contract-first workflow?
6. Достаточно ли доступа для bounded deployment?

## 7. Recommended qualification rule

Goalrail должен начинать только с тех клиентов, где:

- можно быстро показать visible value
- можно провести one visible workflow from task to proof
- не требуется full custom consulting before first result

## 8. Recommendation

`ICP` стоит держать отдельным файлом, а не частью GTM model.

Причина:
- ICP меняется медленнее, чем landing или sales copy
- fit logic нужен отдельно для sales и deployment qualification
- этот файл полезен как самостоятельный qualification reference
