# Goalrail

[Русский](README.ru.md)

Goalrail is a productized operating layer for AI-assisted delivery.

**Tagline:** from business goal to verified code change.

![Goalrail pipeline overview](goalrail-pipeline.png)

## What it is

Goalrail helps software teams move from intent to a bounded, verified delivery change.
It sits between planning and execution:
- clarifies goals and constraints
- shapes bounded delivery contracts
- executes through existing developer runtimes
- returns verification and proof, not just status

## Current state

Goalrail's product canon, MVP shape, and operating model are documented,
and early implementation prototypes exist under `apps/server`, `apps/cli`,
`apps/worker`, `apps/runner`, and `apps/web`.

For current implementation truth, use `docs/ops/STATUS.md` and
`docs/ops/COMPONENTS.yaml`. This README is only a high-level entry point.

The implemented server slice covers a server-owned lifecycle from intake
through work-item planning and the first bounded runner receipt baseline:
IntakeRecord → Goal → ContractSeed → ContractDraft (`draft`) →
ContractDraft (`ready_for_approval`) → ApprovedContract (`approved`) →
WorkItem (`planned`) → CheckoutReceipt → ExecutionJob → Run (`started`) →
ExecutionCommandPlan (`builtin_diagnostic/workspace_status`) →
ExecutionReceipt (`builtin_diagnostic`). HTTP routes, Postgres-backed
persistence for the core canonical objects, migrations, auth, and event append
for the key transitions are in place.

This is **not** a full Goalrail runtime and **not** an agent platform.
Goalrail remains a contract-first, bounded control plane that supplements
existing developer and business tools rather than replacing them.

## Repo surfaces

- `apps/server` — early Go server prototype and bounded persistence slice; see
  `apps/server/README.md`.
- `apps/cli` — early local/demo Go CLI.
- `apps/web` — shared web workspace with console shells, demo surfaces, and the
  RU pilot landing; see `apps/web/README.md`.

## What is not implemented

- arbitrary shell or project command execution
- project test execution such as `npm test`, `go test ./...`, or `pytest`
- gate, proof generation, or runnable eval harness
- WorkItem assignment / claiming or completion semantics
- runner registration / token hardening or runner attestation
- actual repository clone/fetch, repository writes, provider OAuth, or stored
  repository credentials
- tracker sync, analytics, CRM, or full product web loop
- broad backend platform, LLM coding-agent integration, or provider runtime
  adapters

## Read first

- `docs/INDEX.md`
- `docs/product/GOALRAIL_PRODUCT_BRIEF.md`
- `docs/product/GOALRAIL_MVP_BLUEPRINT.md`
- `docs/product/GOALRAIL_BUILD_ROADMAP.md`

## Contact

- [hello@goalrail.dev](mailto:hello@goalrail.dev)

## Open source and community

- [LICENSE](LICENSE)
- [CONTRIBUTING](CONTRIBUTING.md)
- [CODE_OF_CONDUCT](CODE_OF_CONDUCT.md)
- [SECURITY](SECURITY.md)
- [SUPPORT](SUPPORT.md)
- [TRADEMARKS](TRADEMARKS.md)
