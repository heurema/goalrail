# Goalrail Adjacent Experiments Synthesis

## Purpose

This document explains how Goalrail can learn from independent adjacent experiments such as Punk without creating product dependency, roadmap coupling, or source-of-truth confusion.

It is advisory source material only. It does not define Goalrail canon.

## Source set reviewed

Goalrail baseline docs reviewed for comparison:

- `docs/INDEX.md`
- `docs/product/GOALRAIL_PRODUCT_CONCEPT.md`
- `docs/product/GOALRAIL_OPERATING_MODEL.md`
- `docs/product/GOALRAIL_DEPLOYMENT_MODEL.md`
- `docs/product/GOALRAIL_PILOT_MODEL.md`
- `docs/product/GOALRAIL_PRODUCT_BRIEF.md`
- `docs/product/GOALRAIL_MVP_BLUEPRINT.md`
- `docs/product/GOALRAIL_PUBLIC_NARRATIVE.md`
- `docs/product/GOALRAIL_PUBLIC_LANGUAGE.md`
- `docs/product/GOALRAIL_IMPLEMENTATION_GUIDE.md`
- `docs/PROJECT_SPINE_SCHEMA.md`
- `docs/ops/STATUS.md`
- `docs/ops/NEXT.md`
- `docs/ops/DECISIONS.md`
- `docs/ops/COMPONENTS.yaml`
- `README.md`

Adjacent Punk material reviewed as advisory only:

- Punk README
- Punk START-HERE
- Punk RESEARCH-GATE
- Punk RESEARCH-INTAKE
- Punk TELEMETRY
- Punk 2026-04-19 project ideas intake
- Punk 2026-04-19 research idea backlog

## Goalrail baseline

Goalrail baseline:

- productized operating layer
- contract-first flow
- Project Spine
- bounded execution
- verify / proof contour
- pilot-first deployment
- production-facing product for software teams

## Punk baseline

Punk baseline:

- independent experimental R&D sandbox
- local-first bounded work kernel / sandbox
- experiments around plot/cut/gate, event logs, eval harnesses, contract lifecycle, gate decisions, proofpack, inspectable state
- not Goalrail dependency
- not Goalrail truth

## Relationship model

Punk and Goalrail are separate projects.

Punk may generate ideas, mechanisms, experiments, and anti-patterns.
Goalrail may optionally inspect those ideas.
Goalrail adopts only what fits its own product canon, MVP boundaries, deployment model, and trust model.

Required intake flow:

```text
Adjacent experiment or external idea
  -> optional Goalrail relevance check
  -> Goalrail Research Intake
  -> adopt / adapt / defer / park / avoid
  -> Goalrail ADR / product doc / ops patch
  -> bounded implementation slice
  -> eval / proof
```

Adjacency is useful. Dependency is not implied.

## Overlap

There is meaningful overlap in discipline, not in product identity.

Shared themes:

- explicit research before high-impact changes
- bounded work instead of vague execution
- visible truth ownership
- evidence, verify, and proof orientation
- concern about hidden telemetry and hidden truth surfaces
- interest in project memory that stays inspectable

These overlaps are useful because they reveal mechanisms and anti-patterns Goalrail can evaluate.

## Complement

Punk is useful as a mechanism laboratory.

Goalrail is useful as the productized operating-layer frame that decides which mechanisms are worth turning into product truth, deployment rules, or pilot-safe behavior.

In practice:

- Punk can explore candidate trust mechanisms earlier
- Goalrail can decide which of those mechanisms survive product, deployment, MVP, and public-claim mapping
- Goalrail can also reject mechanisms that are valid for Punk but wrong for a production-facing operating layer

## Tensions and non-transferable parts

Not everything transfers.

Key tensions:

- Punk is local-first by default; Goalrail is productized and deployment-oriented
- Punk lifecycle language is `plot -> cut -> gate`; Goalrail canon is contract -> task -> run -> decision -> proof
- Punk may explore kernel-first operator mechanics; Goalrail must preserve business-facing product clarity
- Punk can park many ideas as experiments; Goalrail must avoid accidental public promises

Non-transferable or high-friction areas:

- autonomous agent execution as a default operator path
- marketplace or plugin-ecosystem expansion
- hidden analytics or hidden network export
- memory as a giant prompt
- adapters or runtimes owning truth
- direct copy of Punk product form, terminology, or roadmap

## Adoption map

| Source mechanism | Goalrail mapping | Recommendation | Reason | Required Goalrail doc / ADR / eval |
|---|---|---|---|---|
| Research Gate | Research-before-change policy for core decisions | `adopt` | Goalrail needs explicit evidence discipline before major canon or trust changes | `docs/product/GOALRAIL_RESEARCH_GATE.md` |
| Research Intake | Intake classification before roadmap spread | `adopt` | Goalrail needs a bounded intake path for adjacent ideas | `docs/product/GOALRAIL_RESEARCH_INTAKE.md` |
| adopt / adapt / defer / park / avoid classification | Standard adjacent-idea decision vocabulary | `adopt` | Goalrail needs more than accept/reject to avoid roadmap sprawl | `docs/product/GOALRAIL_RESEARCH_INTAKE.md` |
| active-core / incubating / parked vocabulary | Lifecycle vocabulary for docs, surfaces, and governance areas | `adapt` | Useful maturity language, but Goalrail must not inherit Punk operator status semantics directly | `docs/product/GOALRAIL_DOC_GOVERNANCE.md` |
| append-only event ledger | Future event/provenance backbone inside Goalrail truth model | `adapt` | Useful trust mechanism, but Goalrail must fit it to Project Spine and deployment boundaries | later Goalrail ledger / provenance ADR and evals |
| replay-derived inspect views | Future derived views rebuilt from canonical evidence | `adapt` | Valuable inspectability pattern, but must stay derived and not become truth | later Goalrail inspect / derived-view ADR and evals |
| proofpack manifest with artifact hashes | Future proof provenance structure | `adapt` | Useful for verify/proof trust, but Goalrail must map it to decision/proof semantics | later Goalrail proof / provenance ADR and evals |
| guard denial receipts | Explainable denial or escalation evidence | `adapt` | Strong anti-ambiguity pattern for verify/gate behavior | later Goalrail gate / policy / receipt ADR and evals |
| machine JSON + human Markdown eval reports | Dual-surface evidence reporting | `adapt` | Good reporting pattern, but Goalrail should shape outputs around product proof needs | later Goalrail eval/report format docs and evals |
| no-network default | Privacy/export discipline for sensitive paths | `adapt`, not direct `adopt` | Valuable guardrail, but Goalrail deployment modes may need a different default contour than Punk | later Goalrail privacy/export research and ADR |
| redaction fixture suite | Deterministic privacy/redaction evidence | `adapt` | Good trust mechanism, but fit must follow Goalrail data/export policy | later Goalrail redaction ADR and evals |
| project-memory link graph | Linked project memory across artifacts | `adapt` | Valuable for inspectability, but Goalrail must preserve canonical ownership and avoid giant-prompt memory | later Goalrail project-memory doc / ADR |
| autonomous agent execution as default operator path | Not a Goalrail MVP default | `avoid` for MVP | Conflicts with Goalrail contract-first, bounded, pilot-first posture | keep out of MVP and public promise |
| marketplace / plugin ecosystem | Possible later ecosystem topic | `park` | Premature and scope-widening for Goalrail now | keep out of canon and roadmap until explicitly promoted |
| hidden analytics / hidden network export | Anti-pattern | `avoid` | Conflicts with trust posture and honest public claims | preserve explicit export policy only |
| memory as giant prompt | Anti-pattern | `avoid` | Conflicts with inspectable, linked, authoritative project memory | preserve structured memory model only |
| adapters owning truth | Anti-pattern | `avoid` | Conflicts with canonical-doc and Project Spine truth ownership | preserve adapter-as-boundary only |
| AI writing final decisions | Anti-pattern | `avoid` | Conflicts with Goalrail gate / decision boundary | preserve gate-owned verdict semantics |

## Goalrail-specific anti-patterns

- treating Punk as Goalrail upstream
- copying Punk product form directly
- making Goalrail local-only by accident
- making Goalrail a generic agent platform
- adding hidden telemetry
- proof without artifact hashes
- memory as a giant prompt
- adapters or runtimes owning truth
- skipping gate / proof because runtime succeeded
- promoting external experiments without evidence
- expanding MVP from adjacent-project inspiration

## Recommended next docs

This synthesis supports the following immediate docs:

- `docs/product/GOALRAIL_RESEARCH_GATE.md`
- `docs/product/GOALRAIL_RESEARCH_INTAKE.md`
- `docs/product/GOALRAIL_DOC_GOVERNANCE.md`

Likely later follow-ups, in separate PRs:

- provenance / proof ADRs
- privacy / redaction / export ADRs
- project-memory governance doc
- deterministic checker spec and migration plan
