# Goalrail Documentation Governance

## Purpose

This document defines how Goalrail documentation is governed: who owns truth, how documents change, how metadata is intended to work, how false implementation claims are contained, and how enforcement should ratchet in later PRs.

PR1 defines the framework only. It does not implement schemas, fixtures, checker tooling, CI, or metadata migration.

## Truth model

```text
canonical doc content = source of truth
frontmatter = machine-readable metadata
docs/INDEX.md = human read-order / priority view
checker = consistency guard, not semantic authority
```

Generated documentation maps may appear later, but only as derived artifacts. They do not replace canonical content.

## Governance layers

1. **Research Governance**
   Decides when new evidence is required before Goalrail changes meaningfully.

2. **Doc Governance**
   Decides how truth is labeled, linked, superseded, reviewed, and bounded across docs.

3. **Deterministic Enforcement**
   Later checker and report logic that detects explicit inconsistencies without becoming the semantic owner of Goalrail meaning.

These layers are ordered. Research informs docs; docs define truth; enforcement checks explicit consistency against that truth.

## Authority model

Authority labels classify document role. They do not replace the source-of-truth order in `docs/INDEX.md`.

| Authority | Meaning | Typical use |
|---|---|---|
| `canonical` | Owns product or architecture truth | product canon, architecture canon |
| `operational` | Tracks current working state and near-term execution | ops docs |
| `advisory` | Useful analysis or synthesis, but not truth by itself | research notes, adjacent synthesis |
| `derived` | Generated or computed view over canonical or operational truth | future maps, reports, generated indexes |
| `public_entry` | Public-facing summary or entry point | `README.md`, future public entry docs |
| `reference` | Supporting material with no truth ownership | screenshots, references, examples |

When documents conflict:

- canonical content wins over every other authority
- operational docs must align with canonical docs
- public entry docs may summarize but may not redefine canon
- advisory and reference material may inform changes but may not overrule them

## Lifecycle model

Goalrail needs two separate concepts:

- **status** = document artifact state
- **lifecycle** = maturity of the surface, capability, or governance area described

Typical document movement:

`draft -> current -> superseded or retired`

Typical surface movement:

`incubating -> active-core -> parked or retired`

These are intentionally separate so a current document can describe a parked area, and a draft document can propose changes to an active-core area.

## Frontmatter v0

Frontmatter v0 is the planned metadata vocabulary for later migration. PR1 defines fields and semantics only.

No normative frontmatter template is introduced in this PR.

### Fields

| Field | Meaning |
|---|---|
| `id` | Stable document identifier |
| `title` | Human-readable document title |
| `kind` | Document class |
| `authority` | Truth role for the document |
| `status` | Artifact state of the document |
| `owner` | Responsible maintainer or owning role |
| `truth_surfaces` | Specific surfaces the document speaks for |
| `lifecycle` | Maturity of the area described |
| `review_after` | Planned review date for drift checks |
| `supersedes` | Older docs or surfaces replaced by this doc |
| `superseded_by` | Newer doc that replaced this one |
| `related_docs` | Repo-relative links to directly related docs |

### Recommended enums

`kind`

- `product_canon`
- `architecture_canon`
- `ops_status`
- `ops_plan`
- `adr`
- `brand_canon`
- `public_entry`
- `research_note`
- `derived_view`
- `reference`

`authority`

- `canonical`
- `operational`
- `advisory`
- `derived`
- `public_entry`
- `reference`

`status`

- `current`
- `draft`
- `superseded`
- `retired`
- `reference`

`lifecycle`

- `active-core`
- `incubating`
- `parked`
- `retired`

### Field semantics

- `status` describes the state of the document artifact itself.
- `lifecycle` describes the maturity of the capability, surface, or governance area described.
- `truth_surfaces` should be explicit and bounded, not narrative.
- `supersedes` and `superseded_by` are for lineage, not for adding new meaning silently.
- `related_docs` should stay repo-relative and mechanical.

## DocImpact v0

DocImpact v0 is a policy concept, not a schema implementation.

- schema and examples may appear in a later PR
- no separate DocImpact records folder exists in v0
- real records can live inside ops reports when needed
- DocImpact becomes required later only for changes to governance, source-of-truth priority, implementation status, MVP scope, or public claims

The intent is to preserve why a meaningful documentation change happened without forcing a heavy artifact before the rest of the system exists.

## README boundary

`README.md` is `public_entry`.

Rules:

- README may summarize and link
- README must not introduce canonical product truth
- if README conflicts with canonical docs, canonical docs win

README is allowed to explain repository status, current boundaries, and safe entry paths. It is not allowed to silently redefine Goalrail product meaning, MVP scope, trust model, or implementation status.

## Supersede / archive / retire policy

Goalrail should preserve truth lineage instead of silently deleting meaning.

- **supersede** when one document replaces another as the active owner of the same truth surface
- **archive** when a document is kept for historical reference but is no longer part of the active read path
- **retire** when a surface or doc is intentionally no longer active and should not be treated as current guidance

Rules:

- prefer explicit supersession over silent replacement
- preserve repo-relative links when moving a document out of the active path
- do not rewrite history to hide earlier assumptions; mark them as superseded or retired
- do not use metadata-only patches to smuggle semantic product changes

## False implementation claims

False implementation claims are the main Goalrail v0 governance risk.

This repo is docs-first. It does not yet have a Goalrail implementation baseline. Documentation must not imply otherwise.

Explicitly avoid claims that suggest:

- a runtime exists when it does not
- a gate / proof engine exists when it does not
- integrations are implemented when they are only planned
- production readiness exists when the status docs say planning
- automation is real when only policy or concept docs exist

Ambiguous language should be tightened early. Explicitly false readiness language should be treated as a violation later.

## Metadata migration rule

Metadata migration patches may add or adjust frontmatter, related_docs, lifecycle/status metadata, and mechanical links.
They must not rewrite product thesis, MVP scope, pilot model, public narrative, architecture meaning, or implementation status.
If semantic changes are needed, create a separate patch.

This separation matters because metadata work is mechanical, while canon changes are semantic.

## Deterministic enforcement model

Future enforcement must stay deterministic.

Rules:

- checker stays a consistency guard, not a semantic authority
- no semantic or LLM judge
- explicit keyword, status, path, and metadata checks are preferred
- broken repo-relative links, invalid enums, missing lineage, and explicit status conflicts are valid targets

`docs/ops/COMPONENTS.yaml` will become the implementation-status anchor in a later PR. Not in PR1.

Brand and public narrative docs should start warning-first for ambiguous language, but explicit production-readiness claims that conflict with status should become hard fail later.

## Ratchet rollout

- **Phase 0:** report-only
- **Phase 1:** hard fail on touched canonical docs only
- **Phase 2:** hard fail on touched docs under `docs/product`, `docs/ops`, `docs/adr`
- **Phase 3:** scheduled repo-wide report only
- **Phase 4:** optional repo-wide hard gate after migration

Rule:

- no new violations
- legacy violations are tracked until migrated

The ratchet is intentional. Goalrail should improve reliability without pretending the repo is already fully migrated.

## Non-goals

- no schema implementation in PR1
- no checker in PR1
- no CI in PR1
- no frontmatter migration in PR1
- no runtime or product implementation
- no public CLI
- no repo-wide hard gate
