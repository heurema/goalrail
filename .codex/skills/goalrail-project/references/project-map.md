# Goalrail Project Map

## Purpose

This file is a routing map, not a cached source of project facts.
Use it to decide where to read next.
For live truth, prefer `docs/INDEX.md`, canonical docs, and `docs/ops/*`.

## Identity

Goalrail is a productized operating layer for AI-assisted delivery.
Working thesis:

**от бизнес-цели до проверенного изменения в коде**

The product connects intent, contract shaping, bounded execution, verification, and proof.
It does not replace trackers, IDEs, or developer runtimes.

## Source of truth

Read in this order:
1. `docs/INDEX.md`
2. concept canon in `docs/product/`
3. product summary and public-language layers as needed
4. architecture canon
5. governance docs
6. `docs/ops/*`
7. advisory research and working surfaces

Priority rule:
1. `docs/product/*`
2. `docs/ops/*`
3. chat context

If `docs/INDEX.md` adds a new document family, follow the index.
Do not wait for this reference file to be updated first.

## Live state sources

Use these files for current repo truth:
- `docs/ops/STATUS.md` — current state and active checkpoint target
- `docs/ops/NEXT.md` — immediate bounded slices
- `docs/ops/COMPONENTS.yaml` — implementation-status anchor

Do not treat this reference file as the authoritative place for current status.

## Topic routing

### Product meaning, offer, pricing, payment, GTM
1. Start from `docs/INDEX.md`
2. Read the relevant `docs/product/*` files named there
3. If the exact topic is new, search directly:

```bash
rg -n "pricing|payment|billing|offer|pilot|gtm" docs/product docs/ops
```

### Implementation status or “is this real?” questions
1. `docs/ops/STATUS.md`
2. `docs/ops/COMPONENTS.yaml`
3. check whether real implementation paths exist

### Architecture and execution model
1. `docs/product/GOALRAIL_MVP_BLUEPRINT.md`
2. `docs/PROJECT_SPINE_SCHEMA.md`
3. `docs/product/GOALRAIL_PARALLEL_EXECUTION_MODEL.md`
4. relevant ADRs

### Governance and doc discipline
1. `docs/product/GOALRAIL_RESEARCH_GATE.md`
2. `docs/product/GOALRAIL_RESEARCH_INTAKE.md`
3. `docs/product/GOALRAIL_DOC_GOVERNANCE.md`
4. `tools/docs-check/README.md`

## Current implementation direction

Before starting implementation, read:
- `docs/ops/STATUS.md`
- `docs/ops/NEXT.md`
- `docs/product/GOALRAIL_BUILD_ROADMAP.md`
- `docs/product/GOALRAIL_IMPLEMENTATION_GUIDE.md`

The active checkpoint target may change.
Do not cache it in the skill as a durable fact.

## Product boundaries that must not drift

Goalrail is not:
- a generic agent framework
- a replacement for trackers in v1
- an IDE
- a chat-over-code product
- a fully autonomous engineering platform

Goalrail is:
- a structured project memory and delivery control layer
- a contract-first execution system
- a verification and proof contour
- a bridge between planning intent and engineering delivery
