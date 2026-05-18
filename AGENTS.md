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
- All Go changes must follow `docs/ops/GO_CODE_GUIDE.md`.
- When working on Go code, use `.codex/skills/go-reference/SKILL.md` as a reference navigator: proceed directly when clear; when making meaningful architecture/API/layout/concurrency/dependency/test decisions or when uncertain, consult the linked Go references before deciding.

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
- Do not introduce provider-specific execution doctrine into Goalrail core.
- Treat user/runtime execution setup as external to the kernel.
- Goalrail standardizes contract, bounded task packet, receipts, verification, gate decision, proof, and feedback — not private prompts, skills, or model settings.
- Prefer outcome verification over executor prescription.

## Current repository state

This repository remains canon-first and docs-governed.
Early implementation prototypes exist under `apps/server`, `apps/cli`, `apps/worker`, `apps/runner`, and `apps/web`.
`apps/web/pilot-intake-ru` is a public RU pilot landing surface with narrow D-0056/D-0058/D-0059 backend exceptions for lead capture, daily digest, and Resend mail transport.
Auth, minimal planning worker, and minimal runner receipt / preparation prototypes exist where `docs/ops/STATUS.md`, `docs/ops/COMPONENTS.yaml`, and code reality agree.
No gate, proof generation, real project test execution, provider OAuth, actual repository clone/fetch/write, WorkItem assignment/claiming/completion, tracker sync, broad backend platform, analytics, CRM, or full product web loop exists yet.
Paths in `apps/`, `scripts/`, and `.github/` are evidence of implementation only when `docs/ops/COMPONENTS.yaml` and code reality agree.

## Punk publishing tasks

- Do not write directly to `.punk/publishing/` as the default workspace.
- Resolve publishing context with: `punk publishing locate --project-root . --json`
- If the resolver is unavailable:
  1. Read `.punk/publishing.toml` as committed binding metadata.
  2. Check for optional ignored local pointer: `.punk/publishing.local.toml`.
  3. Use `.punk/publishing.local.toml` only if:
     - it contains `workspace_root`.
     - its `workspace_ref`, if present, matches `.punk/publishing.toml`.
     - it is treated as local-only and not project truth.
  4. If the local pointer is missing or invalid:
     - do not invent a workspace path or store runtime artifacts in the repo.
     - produce the draft in the response, or ask for an explicit external target path.
     - suggest manual bootstrap by creating an ignored `.punk/publishing.local.toml`.
- `.punk/publishing/` has been removed from repo and must not be used or recreated as a publishing workspace.
- If it appears locally, treat it as ignored legacy residue and do not write to it.
- Use resolver or validated `.punk/publishing.local.toml`.
- Never store credentials, tokens, browser sessions, account secrets, platform secrets or publishing account secrets in repo.
- Physical paths are platform-native and resolver-owned. Do not commit expanded user paths into repo files.
- Symlinks are not part of the publishing architecture and must not be created as a workaround.
- `.punk/local/` must not be created as a workaround.
