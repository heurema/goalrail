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
4. Before adding or moving files, read `docs/ops/REPO_STRUCTURE.md`.

## Source-of-truth priority

1. `docs/product/*`
2. `docs/ops/*`
3. chat context

## Core rules

- Do not invent implemented runtime, services, packages, apps, or integrations that do not exist yet.
- Treat `docs/product/` as canonical product truth.
- Treat `docs/ops/` as the working operating layer.
- When real implementation starts, do it with `punk` as the repo delivery discipline.
- Keep docs synchronized when architecture, MVP boundaries, trust model, or repo shape changes.
- Follow the current doc law: brief first, blueprint second, then roadmap/implementation rules.
- Prefer small, reviewable changes.
- Do not add framework sprawl or fake scaffolding.
- Use `docs/ops/REPO_STRUCTURE.md` for where to add code, docs, tools, overlays, and new top-level paths.
- When adding real implementation, create new code under `apps/` unless canon or ops docs explicitly assign a different bounded path.
- When creating implementation later, keep scope explicit and bounded.

## Goalrail implementation guardrails

- Follow `docs/product/GOALRAIL_RULE_STACK.md`.
- Product canon beats local notes and implementation assumptions.
- No implementation without a component mapping in `docs/ops/COMPONENTS.yaml`.
- No public surface without documentation.
- Status must match reality.
- Lower-level rules may narrow, never override.
- Implementation PRs must fill `ComponentImpact` and `DocImpact` in the PR template.
- If a change affects product concept, operating model, Project Spine, MVP scope, verification/proof semantics, or runtime boundaries, use the Research Gate / architecture docs before or with implementation.
- Do not add runtime or product implementation outside the current bounded slice.
- Do not make false implementation claims.

## Current repository state

This repository is currently documentation-first, with a local mocked demo under `apps/web/demo-change-packet/`.
Placeholders in `apps/`, `scripts/`, and `.github/` are not evidence of implementation unless `docs/ops/COMPONENTS.yaml` and code reality agree.
