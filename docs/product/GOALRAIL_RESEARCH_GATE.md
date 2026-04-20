# Goalrail Research Gate

## Purpose

This document defines when Goalrail needs research before product, architecture, governance, or implementation decisions move forward.

The goal is simple: important changes must be informed before they become canon, ADRs, ops direction, or implementation slices.

## Principle

Research is required for important decisions, not for every small edit.

Goalrail is currently docs-first and production-facing in posture, but it does not have an implementation baseline yet. Because of that, the cost of incorrect architecture or false readiness claims is high. Research is used to reduce that risk, not to slow down every minor change.

Research output is advisory until it is promoted into Goalrail-owned truth.

## When research is required

Research is required before changing any of the following:

- product core laws
- Project Spine objects or the Project Spine object chain
- contract lifecycle
- verification / gate / proof semantics
- event ledger / telemetry / provenance model
- privacy / redaction / export policy
- runtime adapter boundaries
- project memory model
- MVP scope
- pilot model or deployment model
- public claims about trust, proof, automation, or production readiness

Research is also required for governance changes that would alter source-of-truth priority, document authority, or the enforcement posture around false implementation claims.

## When research is not required

Research is not required for:

- typo fixes or docs cleanup
- implementation of an already approved doc, ADR, or bounded slice
- small bounded edits with no architecture or product effect
- ops status updates that do not change scope

R0 work is valid when the change is clearly mechanical, already decided, or too small to affect Goalrail meaningfully.

## Research depth levels

### R0: no research

Use when the change is mechanical, already approved elsewhere, or too small to change product, architecture, governance, or public-claim meaning.

### R1: quick scan

Use for bounded questions with limited blast radius.

Expected output:

- a short question
- a small curated source set
- a concise recommendation

### R2: design research

Use for design and architecture questions that may affect canon, MVP boundaries, trust semantics, or deployment shape.

Expected output:

- curated source review
- explicit pattern extraction
- explicit failure-mode extraction
- Goalrail mapping
- ADR-ready recommendation

### R3: deep research

Use for major product direction, governance, privacy/export, memory, or trust-boundary decisions that would be expensive to reverse.

Expected output:

- broad source review
- trade-off comparison
- explicit risks
- roadmap / policy implications
- required eval / proof implications

## Research lifecycle

```text
Question
  -> Source collection
  -> Source quality rating
  -> Pattern extraction
  -> Failure-mode extraction
  -> Goalrail mapping
  -> Recommendation
  -> ADR / product doc / ops patch
  -> implementation slice
  -> eval / proof
```

The important rule is that research does not end at source review. It must be translated into Goalrail terms, then into bounded decisions, then into evidence-bearing work.

## Source quality tiers

### Tier A — primary and authoritative

Examples:

- official specifications
- official product or repository docs
- primary technical papers with clear methodology
- vendor docs about the vendor's own system
- direct implementation evidence from the actual system being studied

### Tier B — credible field evidence

Examples:

- mature engineering writeups
- postmortems
- issue threads or PR discussions with concrete technical evidence
- comparative analyses grounded in real implementation details

### Tier C — weak or exploratory

Examples:

- unsourced opinions
- hype posts
- benchmark summaries without methodology
- anonymous commentary
- inspiration without reproducible evidence

Tier C may inspire questions. It must not justify a core Goalrail decision by itself.

## Adjacent-project rule

- Punk and similar projects are adjacent sources, not Goalrail truth.
- Direct copy is not allowed unless the mechanism is process-only and does not conflict with Goalrail product canon.
- Any borrowed mechanism must be mapped to Goalrail's product, deployment, MVP, and trust boundaries.

Adjacency is useful because it can reveal mechanisms, anti-patterns, and failure modes. It does not create roadmap inheritance, truth inheritance, or default implementation inheritance.

## Promotion rule

Research does not become Goalrail truth just because it exists.

Promotion path:

```text
research note
  -> Goalrail intake classification
  -> ADR / product doc / ops decision
  -> bounded slice
  -> eval / proof
  -> status update
```

If the promotion path is not clear, the idea stays advisory.

## Contract rule

If a change crosses the Research Gate, the bounded Goalrail work item should carry:

- the exact research question
- the selected depth level
- the source set or synthesis reference
- the Goalrail mapping
- the target ADR / product doc / ops patch
- the eval / proof implication

If a change is R0, the work item should say why no research was required.

## Anti-patterns

Do not:

- keep research only in chat
- copy an adjacent project without Goalrail mapping
- use weak sources for core decisions
- expand MVP from inspiration
- treat benchmark or leaderboard claims as product truth
- skip eval implications
- skip failure modes
