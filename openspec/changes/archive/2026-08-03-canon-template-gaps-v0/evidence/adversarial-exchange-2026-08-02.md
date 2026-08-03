# Adversarial exchange — 2026-08-02

The owner directed the method: the proposal goes to the other provider as an
opponent instructed to refute, before any canon byte changes. Proposer: this
Claude session. Opponent: codex (gpt-5.6-sol, effort high, read-only sandbox,
full repository access). The proposal text included the proposer's own known
weaknesses as explicit attack targets.

Outcome: all three edits AMENDED with exact replacement text, the
schema-version bump REFUTED, both stated weaknesses STANDS, and one blocking
discovery — previousCanons is empty, so the first canon change would classify
every adopter's overlay as edited and break clean `gr update`.

## Opponent's verdict, verbatim

codex
Все три правки в исходном виде не проходят. Главная проблема даже не в формулировках: выпуск изменённого канона сейчас сломает безопасный `gr update` для существующих overlay.

1. `design.md` / Input space — AMEND

Четыре предложенные категории не образуют пространство входов:

- `absent` и `empty` — состояния;
- `user-edited` — происхождение;
- `other-legal-syntax` — слишком широкая корзина;
- не покрыты malformed/unsupported, unreadable, oversized, stale, concurrent mutation, unknown version, conflicting inputs и symlinks.

Кроме того, «anything the change reads» захватит слишком много нерелевантных чтений. Частота 22/78 обосновывает приоритет, но не доказывает, что именно этот набор категорий предотвратит класс ошибок. Утверждение «продолжил повторяться после 2026-08-01» в [context.md](/Users/vi/personal/heurema/goalrail/openspec/changes/canon-template-gaps-v0/context.md:17) не содержит числа последующих случаев.

Точная поправка:

```md
## Consumed Inputs

For every behavior-affecting external input introduced or changed by this
slice, record:

| Input | Source and trust | Accepted states/variants | Refused states | Mutation/race policy | Verification |
|---|---|---|---|---|---|

Consider where applicable: absent, empty, unreadable, malformed or unsupported,
alternate legal syntax/version, oversized, user-edited, and changed after
validation.

If the slice introduces or changes no external input handling, write:
"None — verified within <scope> by <bounded search or trace>."
```

Это должно относиться только к входам, поведение которых меняется данным slice.

2. `intent.md` / `Confirmed wording` — AMEND

Пункт сформулирован некорректно: колонка уже называется `Confirmed wording` в [intent.md](/Users/vi/personal/heurema/goalrail/internal/harness/canon/templates/intent.md:16), но новое имя не указано.

И одной колонки недостаточно:

- `Confirmed boundary` имеет ту же ошибку для candidate-снимка ([intent.md](/Users/vi/personal/heurema/goalrail/internal/harness/canon/templates/intent.md:22));
- schema description уже называет артефакт owner-confirmed, хотя инструкция допускает `candidate` ([schema.yaml](/Users/vi/personal/heurema/goalrail/internal/harness/canon/schema.yaml:27)).

Точная поправка:

```diff
-| ID | Confirmed wording | Verification action | Evidence |
+| ID | Outcome | Verification action | Evidence |

-| ID | Confirmed boundary | Evidence |
+| ID | Boundary | Evidence |
```

И:

```diff
-description: Versioned owner-confirmed statement ...
+description: Versioned candidate-or-confirmed statement ...
```

Подтверждение должно определяться только `Status` и блоком `Confirmation`, а не заголовками смысловых полей.

3. `context.md` / Verification command — AMEND

Команда не равна воспроизводимости:

- историческое состояние команда позднее не воспроизводит, а измеряет заново;
- owner statement или issue не является исполняемой проверкой;
- external/UI/session-scoped факт может не иметь безопасной команды;
- команда без cwd, prerequisites и ожидаемого результата двусмысленна;
- обязательность команды поощряет сетевые, credential-dependent или изменяющие состояние «проверки».

Сам приведённый прецедент это опровергает: CTX-8 содержит инструкцию прочитать URL, а CTX-13 требует специальной Claude Code session environment ([pre-pr context](/Users/vi/personal/heurema/goalrail/openspec/changes/pre-pr-review-v0/context.md:29)).

Точная поправка: сохранить существующий `Source` и добавить не команду, а recipe:

```md
| ID | Kind | Claim | Source | Verification recipe | Observed at | Relevance |
```

Правило:

```md
Verification recipe MUST be bounded and read-only and state prerequisites and
the expected observation. For historical or non-repeatable evidence, write:
"Not independently reproducible — <reason>; retained evidence: <stable ref>."
A refresh command must not be described as reproducing the earlier observation.
```

A. OpenSpec не обеспечивает секции — STANDS

Это полностью убивает слово «mandatory». Секция будет prompt-required, не schema-enforced.

Более того, текущий test вводит в заблуждение: комментарий обещает strict validation, но код только валидирует схему, создаёт change и проверяет доступность шаблонов; `openspec validate` для созданного change вообще не запускается ([canon_pinned_cli_test.go](/Users/vi/personal/heurema/goalrail/internal/harness/canon_pinned_cli_test.go:51)).

Позиционный эффект возможен, но пока не измерен. Он облегчает противоречие автору; наличие или честность claim не обеспечивает.

B. Empty-section rot — STANDS

Фраза `reads nothing` сама по себе не является достаточной защитой. Она становится проверяемой только после независимого определения scope и поиска чтений.

Разрешённая пустая форма должна быть ровно такой:

```md
None — scope checked: <changed components/interfaces>;
evidence: <bounded search, graph trace, or test>.
```

Голое `N/A`, `None` или «change reads nothing» должно считаться незаполненной секцией.

C. Одних templates недостаточно — AMEND

Недостаточно даже добавить те же слова в `schema.yaml`.

Нужны одновременно:

1. `internal/harness/canon/templates/{design,intent,context}.md`;
2. соответствующие instruction/description в `internal/harness/canon/schema.yaml`;
3. зеркальные файлы в `openspec/schemas/goalrail-intent/`;
4. тесты обязательного текста/заголовков;
5. история предыдущего канона для безопасного обновления существующих overlay.

Первые две копии обязаны быть byte-identical: это закреплено [canon_test.go](/Users/vi/personal/heurema/goalrail/internal/harness/canon_test.go:10). Поэтому «three edits» — неверный scope.

D. Бамп schema `2 → 3` — REFUTED

Бамп не решает заявленную проблему. Все `.openspec.yaml` фиксируют только:

```yaml
schema: goalrail-intent
```

Версия там не закреплена. Следовательно, открытые changes всё равно увидят текущую семантику того же имени.

Кроме того, version 2 в принятом ADR обозначает структурный переход `context → intent → proposal → specs → design → tasks`, а не редакцию prose-шаблонов ([ADR 0002](/Users/vi/personal/heurema/goalrail/docs/decisions/0002-version-context-and-evaluation-flow.md:48)).

Оставить `version: 2`. Если нужна воспроизводимость конкретной семантики, сначала ввести явный `canon_digest` binding; один числовой bump создаст только видимость pinning.

E. Deferred per-decision Context Pack binding — AMEND

Отложить новый тяжёлый per-decision artifact разумно; отложить любое связывание решений с контекстом — нет. Иначе проверенный до intent Context Pack может устареть к моменту design.

Точная минимальная поправка в design instruction:

```md
Every material decision MUST cite the Context Pack item IDs it relies on.
If a decision depends on a material fact absent from the current Context Pack,
or on a fact that must be refreshed after its observation time, create a new
Context Pack version before settling that decision.
```

Это использует уже существующее версионирование Context Pack и не вводит отдельный артефакт на каждое решение.

Сильнейшее возражение

Предложение небезопасно выпускать как «три правки». `previousCanons` сейчас пуст ([overlay.go](/Users/vi/personal/heurema/goalrail/internal/harness/overlay.go:14)). После изменения байтов канона overlay существующего пользователя не совпадёт ни с новым, ни с известным предыдущим каноном и будет классифицирован как `edited`, а не `behind`. Затем обычный `gr update` остановится с `ErrLocalEdits`, если пользователь не разрешит `--discard-local-edits` ([update.go](/Users/vi/personal/heurema/goalrail/internal/harness/update.go:115)).

Следовательно, утверждение Context Pack, что «ничего не ломается и одна команда чинит», ложно. Пока старый канон не добавлен в upgrade history и это не закреплено migration test, предложение создаёт ложный конфликт с пользовательскими правками именно на первом реальном обновлении канона.

Репозиторий не изменялся; существующий untracked `openspec/changes/canon-template-gaps-v0/` оставлен нетронутым.
