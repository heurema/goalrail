# Goalrail Repository Agent Guide

## Read order

1. Read `docs/INDEX.md` first.
2. Then read the core docs in this order:
   - `GOALRAIL_PRODUCT_BRIEF.md`
   - `GOALRAIL_MVP_BLUEPRINT.md`
   - `GOALRAIL_BUILD_ROADMAP.md`
   - `GOALRAIL_PARALLEL_EXECUTION_MODEL.md`
   - `GOALRAIL_IMPLEMENTATION_GUIDE.md`
3. Then read `docs/ops/*`.

## Source-of-truth priority

1. `docs/product/*`
2. `docs/ops/*`
3. chat context

## Core rules

- Do not invent implemented runtime, services, crates, apps, or integrations that do not exist yet.
- Treat `docs/product/` as canonical product truth.
- Treat `docs/ops/` as the working operating layer.
- When real implementation starts, do it with `punk` as the repo delivery discipline.
- Keep docs synchronized when architecture, MVP boundaries, trust model, or repo shape changes.
- Follow the current doc law: brief first, blueprint second, then roadmap/implementation rules.
- Prefer small, reviewable changes.
- Do not add framework sprawl or fake scaffolding.
- When creating implementation later, keep scope explicit and bounded.

## Current repository state

This repository is currently a documentation-first planning repo.
Empty directories in `apps/`, `crates/`, `scripts/`, and `.github/` are placeholders, not evidence of implementation.
