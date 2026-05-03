---
id: goalrail_go_code_guide
title: Goalrail Go Code Guide
kind: reference
authority: operational
status: current
owner: engineering
truth_surfaces:
  - go_style
  - go_architecture_discipline
  - go_testing_expectations
lifecycle: active-core
review_after: 2026-08-03
supersedes: []
superseded_by: null
related_docs:
  - docs/INDEX.md
  - docs/adr/ADR-0004-go-server-boundary-and-selected-stack.md
  - docs/ops/REPO_STRUCTURE.md
  - apps/server/README.md
  - .codex/skills/go-reference/SKILL.md
---
# Goalrail Go Code Guide

This guide applies to all Go code in the repository.

`apps/server` is currently the main Go implementation area. Examples in this
guide may use `apps/server` because that is where most current server-side Go
architecture exists.

`apps/web/pilot-intake-ru/server` is not part of the main Go server
implementation unless a slice explicitly scopes it that way. It is a narrow
landing-owned sidecar for the public RU pilot lead endpoint, digest, and purge
commands. These Go rules still apply where relevant, but that sidecar must not
be used to infer behavior, dependencies, routes, or platform architecture for
`apps/server`.

## Baseline

Follow official Go style principles and the existing Goalrail architecture:
- use `gofmt` / `go fmt` for formatting;
- prefer clarity before cleverness;
- prefer simplicity before abstractions;
- prefer maintainability and consistency over local cleverness;
- do not refactor code only to satisfy a personal style preference.

Keep current Go code simple:
- use `net/http` where applicable;
- use `encoding/json` where applicable;
- use `slog` where structured logs are needed;
- use `pgx` plus Squirrel for current persistence code;
- use manual wiring through constructors and explicit composition roots.

Do not add by default:
- web frameworks unless explicitly approved;
- dependency-injection containers;
- ORMs;
- generic repository frameworks;
- broad platform abstractions;
- premature generics;
- store-specific transaction wrappers;
- business-specific transaction wrapper methods such as `CreateWithEvent`,
  `CreateWithEvents`, or `AcceptProposalWithWorkItemsAndEvents`.

Use `.codex/skills/go-reference/SKILL.md` as the decision navigator for
meaningful Go architecture, API, package layout, concurrency, dependency, and
test strategy decisions.

## Required dependencies and nil checks

Required dependencies are passed as required constructor arguments.

Required dependencies are used directly. Do not add repeated defensive
`if dep != nil` or `if dep == nil` branches in service or business methods.
Do not make required dependencies optional through `With...` options.

Tests should provide fake dependencies instead of relying on nil fallback
behavior.

Nil checks are allowed only for:
- genuinely optional dependencies;
- external input or configuration boundaries;
- low-level helpers that intentionally preserve stable error behavior.

For mutation flows that require atomic writes/events, `TransactionRunner` is a
required dependency. Do not add non-transactional fallback branches for
production mutation flows. Use a fake `TransactionRunner` in tests.

Bad:

```go
if s.TxRunner != nil {
	return s.TxRunner.RunReadCommitted(ctx, func(txCtx context.Context) error {
		return s.Store.Create(txCtx, value)
	})
}

return s.Store.Create(ctx, value)
```

Good:

```go
func NewService(store Store, txRunner TransactionRunner, clock Clock) *Service {
	return &Service{
		Store:    store,
		TxRunner: txRunner,
		Clock:    clock,
	}
}

func (s *Service) Do(ctx context.Context) error {
	return s.TxRunner.RunReadCommitted(ctx, func(txCtx context.Context) error {
		return s.Store.Create(txCtx, value)
	})
}
```

## Store/persistence rules

Concrete implementations live in implementation packages. Service-facing
interfaces live in consumer packages.

Typed scanners stay local to the persistence code that needs them. Preserve
column and scan order when changing queries.

Use low-level helpers such as `execSQL`, `execUpdate`, or `queryRow` only for
exact duplicate SQL setup.

Do not unify behavior-sensitive paths when nil checks, error messages,
`CommandTag`, `RowsAffected`, scan order, or domain errors differ.

Do not use store-specific transaction wrappers. Stores should expose concrete
persistence operations; service/use-case code owns transaction boundaries.

## Transaction rules

The service/use-case layer owns transaction boundaries.

Use `TransactionRunner.RunReadCommitted` for atomic mutation flows that must
write canonical state and events together.

Use `txCtx` inside transaction callbacks. Do not accidentally write with the
outer `ctx` inside the callback.

Do not reintroduce store-specific transaction wrappers or business-specific
combined store methods.

## App wiring rules

Manual composition roots are preferred.

Grouped private helpers are okay when they make wiring easier to read without
introducing hidden behavior.

Do not add a DI framework.

Do not add dynamic route maps for public routes unless explicitly approved.
Route wiring remains explicit so public API behavior is easy to inspect.

## HTTP handler rules

Preserve stable status codes and JSON response shapes.

Use private response helpers only for exact duplicate response behavior.

Do not change public API behavior in cleanup PRs.

Keep request parsing, validation, service calls, and response writing direct and
auditable. Do not hide public behavior behind broad handler frameworks.

## Testing rules

Before finalizing Go changes, run:

```bash
git diff --check
```

Run Go tests from the relevant Go module root:

```bash
go test ./...
```

For `apps/server`, run:

```bash
cd apps/server && go test ./...
```

Add focused tests for behavior-affecting changes.

Prefer fake dependencies over nil dependencies in tests. Required service
dependencies should be present in tests even when the fake is minimal.

For docs-only Go guidance changes, Go tests are optional. Running
`cd apps/server && go test ./...` is acceptable when the change touches
server-facing guidance or when extra confidence is useful.

## When not to refactor

Do not refactor for aesthetics alone.

Do not unify code if behavior differs.

Do not remove nil checks at external input, configuration, or helper
boundaries where the nil check is part of stable behavior.

Do not introduce a broad abstraction just to remove small local duplication.

Do not change endpoint behavior, schema behavior, persistence semantics, public
JSON shape, or migration/dependency posture as part of cleanup unless the slice
explicitly scopes that behavior change.
