# Goalrail Next

> Only the next bounded slices. Keep this file short.

## Active phase

- **Phase 0 -> Phase 1 transition**
- product and deployment canon is now in place
- repo workspace now mirrors `punk`-style planning boundaries (`work/`, `knowledge/`, `publishing/`, `flows/`, `evals/`)
- `apps/web/` now exists as the shared namespace for frontend resources
- `apps/web/demo-change-packet` is the current change-packet demo prototype; future web work should follow `apps/web/<resource>`
- the next slices should use those boundaries instead of adding ad hoc top-level storage

## Next bounded slices

### Preflight — Implementation guardrails v0
Goal:
- make future implementation PRs declare component, docs, status, scope, validation, and proof impact

Done means:
- PR template exists
- `AGENTS.md` points agents and developers to the Rule Stack
- existing NEXT slices remain intact and continue below after this preflight

### Slice 1 — CTO deck outline
Goal:
- create a 6–8 slide outline for CTO / Head of Engineering conversations

Done means:
- problem, product, operating model, deployment, pilot, outputs, and next step are sequenced clearly
- the outline is derived from the current canon rather than ad hoc pitch copy

### Slice 2 — Landing copy rewrite
Goal:
- rewrite `docs/product/GOALRAIL_LANDING_COPY.md` for pilot-first, contract-centered motion

Done means:
- prompt-export framing is removed
- CTA is aligned to pilot qualification / task review
- public flow matches `GOALRAIL_DESIGN_DECISIONS.md` and `GOALRAIL_GTM_MODEL.md`

### Slice 3 — Spine package bootstrap
Goal:
- create first implementation package for core domain types and events

Done means:
- IDs, enums, object skeletons, and event envelope compile
- basic serialization / validation tests exist
- implementation starts from the updated canon rather than the older docs baseline

## Deferred until later

- hosted execution
- tracker integrations
- multi-runtime advisory implementation
- external checks implementation
- analytics / admin product
- Goalrail-specific web product features beyond the current change-packet demo prototype
