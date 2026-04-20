# Goalrail Rule Stack

## Purpose

Goalrail is a productized operating layer for AI-assisted delivery. The repository that defines it should be governed through the same disciplined shape that Goalrail asks teams to use.

Goalrail should not present a controlled delivery process while being developed through an uncontrolled one.

This rule stack exists to keep repository work aligned with product canon and to prevent:

- undocumented implementation
- false implementation claims
- component/status drift
- product scope drift
- lower-level rules overriding higher-level truth

The discipline may borrow from adjacent Punk-like experiments, but Punk is not upstream of Goalrail, not a dependency, and not a source of truth for this repository.

## Dogfooding law

Goalrail must be developed through a Goalrail-shaped delivery loop:

`goal -> contract -> bounded execution -> verification -> proof`

This is currently manual/process dogfooding.

- the repository follows the operating discipline before the product automates it
- PRs and implementation slices should make goal, scope, component impact, validation, and proof explicit
- this document does not claim that Goalrail already automates this process

**GR-L0-000 — Dogfood the operating model.**

## Manual now, automated later

Goalrail dogfooding has maturity levels.

### Level 0 — Manual dogfooding

Current state:

- docs
- PR descriptions
- component mapping
- docs-check
- human review
- explicit validation

### Level 1 — Tool-assisted dogfooding

Near-term:

- checker-assisted validation
- changed-files ratchet
- component/status checks
- PR templates
- CODEOWNERS
- slice contracts

### Level 2 — Product-assisted dogfooding

Future:

- Goalrail product manages its own goals, contracts, runs, decisions, and proofs

Level 2 is future state only. It is not current repository reality.

## Rule hierarchy

```text
Root laws
  -> Product canon
  -> Governance rules
  -> Component rules
  -> Scoped module rules
  -> Slice contracts
  -> PR / code / tests
```

Each layer has a different role:

- **Root laws** — repository-wide invariants
- **Product canon** — product truth, MVP boundaries, Project Spine, and operating model
- **Governance rules** — research gate, research intake, doc governance, and adjacent-experiments policy
- **Component rules** — `docs/ops/COMPONENTS.yaml`
- **Scoped module rules** — future subtree-specific rules such as subtree `AGENTS.md` files or component notes
- **Slice contracts** — bounded implementation contracts for one work slice
- **PR / code / tests** — execution artifacts, not product truth by themselves

## Non-override law

Lower-level rules may narrow higher-level rules, but they may not override them.

Examples:

- a subtree or package-level rule may ban network access even if root rules do not mention it
- a subtree or component-local rule may not claim ownership of Project Spine truth
- a slice contract may narrow scope, but it may not expand MVP scope
- code may implement a documented component, but code alone may not redefine product truth

**GR-L0-001 — Lower rules may narrow, never override.**

## Root laws

| ID | Law |
|---|---|
| `GR-L0-000` | **Dogfood the operating model.** Goalrail development follows a Goalrail-shaped loop: `goal -> contract -> bounded execution -> verification -> proof`. |
| `GR-L0-001` | **Lower rules may narrow, never override.** Lower-level rules may only make higher-level rules stricter or more specific. |
| `GR-L0-002` | **Canon wins.** If implementation, README, advisory research, or local notes conflict with canonical product docs, canonical product docs win. |
| `GR-L0-003` | **Contract-first flow remains fixed.** Work should move from intent to contract before bounded execution. |
| `GR-L0-004` | **Execution is bounded.** Implementation slices must have explicit in-scope and out-of-scope boundaries. |
| `GR-L0-005` | **Verification and proof are separate from execution.** A runtime, developer, or implementation step does not become final authority just because it produced code. |
| `GR-L0-006` | **No component, no code.** Production/runtime code must map to a component in `docs/ops/COMPONENTS.yaml`. |
| `GR-L0-007` | **No public surface without documentation.** CLI commands, APIs, schemas, event formats, proof formats, adapters, config surfaces, and user-facing behavior need an owning document. |
| `GR-L0-008` | **Status follows reality.** Implementation status must not be inflated beyond what exists in the repository. |
| `GR-L0-009` | **No false implementation claims.** Docs, README, PRs, and public surfaces must not claim unimplemented runtime or product capabilities exist. |
| `GR-L0-010` | **Research before core changes.** Changes to core laws, Project Spine, verification/proof semantics, runtime boundaries, or MVP scope require the Research Gate. |
| `GR-L0-011` | **Adjacent experiments are advisory only.** Punk and other adjacent projects can inspire mechanisms, but they do not define Goalrail truth. |
| `GR-L0-012` | **One source of truth per surface.** A surface must have a declared truth owner; derived views and summaries must not become second sources of truth. |

## Product canon layer

Key canonical docs:

- `docs/product/GOALRAIL_PRODUCT_CONCEPT.md`
- `docs/product/GOALRAIL_OPERATING_MODEL.md`
- `docs/product/GOALRAIL_MVP_BLUEPRINT.md`
- `docs/PROJECT_SPINE_SCHEMA.md`
- `docs/product/GOALRAIL_IMPLEMENTATION_GUIDE.md`

These documents anchor product and architecture truth for the repository. The implementation guide keeps execution aligned with that truth. Changing these surfaces is a product or architecture change, not an incidental code edit.

## Governance layer

Governance docs:

- `docs/product/GOALRAIL_RESEARCH_GATE.md`
- `docs/product/GOALRAIL_RESEARCH_INTAKE.md`
- `docs/product/GOALRAIL_DOC_GOVERNANCE.md`
- `docs/research/GOALRAIL_ADJACENT_EXPERIMENTS_SYNTHESIS.md`

These documents define how Goalrail truth changes and how adjacent ideas are evaluated. They do not override product canon. Adjacent experiments remain advisory.

## Component layer

`docs/ops/COMPONENTS.yaml` is the component and status anchor.

Rules:

- every implementation path must map to a component
- every component must have a `truth_owner`
- component status controls allowed implementation and public claims
- `public_claim_allowed: false` means no public maturity claim for that component
- implementation paths should not appear before they are intentionally assigned

This PR documents the rule only. It does not require new checker behavior.

## Scoped module rules

Future scoped rules may exist closer to specific subtrees or components.

Examples:

- subtree `AGENTS.md`
- future component-level rules
- module-local notes

Rules:

- scoped rules may only narrow higher rules
- scoped rules cannot change product canon
- scoped rules cannot invent public claims
- scoped rules cannot move a parked component into active implementation

This PR does not create scoped module files.

## Slice contracts

Implementation slices should be expressed as bounded contracts.

A slice contract should include:

- goal
- canonical refs
- affected components
- implementation paths
- in scope
- out of scope
- validation commands
- proof expectations
- docs/status impact

Non-normative sketch for later PRs:

```text
Goal: Add X without changing Y
Canonical refs: ...
Affected components: ...
Implementation paths: ...
In scope: ...
Out of scope: ...
Validation: ...
Proof: ...
Docs/status impact: ...
```

This PR does not create `docs/ops/contracts/`.

## PR impact declarations

Every implementation PR should declare:

- affected components
- docs updated
- docs not updated reason, if applicable
- status changes
- public surfaces added or changed
- validation commands
- proof / evidence

This will be operationalized later through a PR template and CODEOWNERS. This PR does not add either one.

## Enforcement model

Rule Stack v0 separates deterministic enforcement from human review.

Deterministic checks can cover:

- path/component mapping
- frontmatter validity
- broken repo-relative links
- false implementation claim patterns
- component status shape
- missing truth owner
- changed-files hard violations

Human review covers:

- product meaning
- MVP scope
- semantic architecture drift
- whether implementation still matches Goalrail positioning
- whether Goalrail is drifting into a generic agent platform

No semantic or LLM judge is a hard gate in v0. LLM review may become advisory later, but it is not blocking in Rule Stack v0.

## Conflict resolution

When rules conflict:

- the higher-level rule wins
- if a lower-level change needs to violate a higher-level rule, the higher-level rule must change first through the appropriate governance path
- code cannot silently update product truth
- implementation cannot override product canon by existing

Examples:

- if code wants a new Project Spine object, update the Project Spine schema or related canon first, or in the same approved architecture PR
- if a runtime adapter appears, it must map to a component and a `truth_owner`
- if public copy says something exists, `docs/ops/COMPONENTS.yaml` status must allow the claim

## Non-goals

This document is not:

- a new implementation framework
- a replacement for product canon
- a semantic judge
- a repo-wide hard gate
- a new source of truth
- a claim that Goalrail already automates its own development process
- a copy of Punk structure
