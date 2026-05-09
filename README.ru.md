# Goalrail

[English](README.md)

Goalrail — это продуктовый операционный слой для разработки с поддержкой AI.

**Теглайн:** от бизнес-цели до проверенного изменения в коде.

## Что это

Goalrail помогает продуктовым и инженерным командам переходить от намерения к ограниченному и проверяемому изменению в коде.
Он находится между планированием и исполнением:
- проясняет цели и ограничения
- формирует ограниченные контракты на выполнение
- выполняет работу через существующие среды разработки
- возвращает верификацию и доказательства, а не просто статус

## Текущее состояние

Канон продукта, форма MVP и операционная модель задокументированы.
В репозитории уже есть ранние implementation prototypes под `apps/server`,
`apps/cli`, `apps/worker`, `apps/runner` и `apps/web`.

Актуальная implementation truth живет в `docs/ops/STATUS.md` и
`docs/ops/COMPONENTS.yaml`. Этот README — только high-level entry point.

Реализованный server slice покрывает server-owned lifecycle от intake до
work-item planning и первый bounded runner receipt baseline:
IntakeRecord → Goal → ContractSeed → ContractDraft (`draft`) →
ContractDraft (`ready_for_approval`) → ApprovedContract (`approved`) →
WorkItem (`planned`) → CheckoutReceipt → ExecutionJob → Run (`started`) →
ExecutionCommandPlan (`builtin_diagnostic/workspace_status`) →
ExecutionReceipt (`builtin_diagnostic`). HTTP routes, Postgres-backed
persistence для основных canonical objects, migrations, auth и event append для
ключевых transitions уже есть.

Это **не** полный Goalrail runtime и **не** agent platform.
Goalrail остаётся contract-first, bounded control plane, который дополняет
существующие developer и business tools, а не заменяет их.

## Поверхности репозитория

- `apps/server` — ранний Go server prototype и bounded persistence slice; см.
  `apps/server/README.md`.
- `apps/cli` — ранний local/demo Go CLI.
- `apps/web` — shared web workspace с console shells, demo surfaces и RU pilot
  landing; см. `apps/web/README.md`.

## Что не реализовано

- arbitrary shell или project command execution
- project test execution вроде `npm test`, `go test ./...` или `pytest`
- gate, proof generation или runnable eval harness
- WorkItem assignment / claiming или completion semantics
- runner registration / token hardening или runner attestation
- actual repository clone/fetch, repository writes, provider OAuth или stored
  repository credentials
- tracker sync, analytics, CRM или full product web loop
- broad backend platform, LLM coding-agent integration или provider runtime
  adapters

## С чего начать

- `docs/INDEX.md`
- `docs/product/GOALRAIL_PRODUCT_BRIEF.md`
- `docs/product/GOALRAIL_MVP_BLUEPRINT.md`
- `docs/product/GOALRAIL_BUILD_ROADMAP.md`

## Контакты

- [hello@goalrail.dev](mailto:hello@goalrail.dev)

## Открытый исходный код и сообщество

- [LICENSE](LICENSE)
- [CONTRIBUTING](CONTRIBUTING.md)
- [CODE_OF_CONDUCT](CODE_OF_CONDUCT.md)
- [SECURITY](SECURITY.md)
- [SUPPORT](SUPPORT.md)
- [TRADEMARKS](TRADEMARKS.md)
