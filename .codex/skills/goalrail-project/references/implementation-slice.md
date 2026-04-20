# Goalrail Implementation Slice

## Default implementation posture

If implementation starts, do it as one bounded slice.
Do not treat roadmap phases as executable tasks by themselves.

Goalrail implementation discipline says:
- roadmap item -> bounded research question -> exact handoff artifact -> implementation slice
- real implementation proceeds through `punk` as the repo delivery discipline
- proof target must be explicit before coding starts

## How to pick the live slice

Before assuming the next implementation target, read:
1. `docs/ops/STATUS.md`
2. `docs/ops/NEXT.md`
3. `docs/product/GOALRAIL_BUILD_ROADMAP.md`
4. `docs/product/GOALRAIL_IMPLEMENTATION_GUIDE.md`

The active checkpoint target may change.
Treat this file as a workflow guide, not as the status source of truth.

## If the active target is still C1

Scope should stay inside:
- core domain types
- IDs and enums
- canonical objects
- event envelope
- serialization tests
- validation tests
- explicit canonical vs derived boundaries

## Out of scope for C1

Do not pull in:
- runtime registry implementation
- CLI runtime
- web UI
- tracker sync
- gate/proof implementation
- advisory panel implementation
- broad schema sprawl beyond the bounded core
- integrations or vendor-specific adapters

## Working method

1. Re-read the relevant canon and roadmap checkpoint.
2. Define exact in-scope and out-of-scope items.
3. Define proof of done.
4. Implement the smallest compiling slice.
5. Run tests or assertions.
6. Update `docs/ops/STATUS.md`, `docs/ops/NEXT.md`, `docs/ops/DECISIONS.md`, and `docs/ops/COMPONENTS.yaml` only if the slice changed current repo truth.

## Reminder about parallelism

Goalrail has two distinct parallel models:
- parallel task execution
- advisory parallel review

Do not collapse them into one implementation surface.
One writable run still has one primary writer runtime.
Advisory lanes remain advisory.
