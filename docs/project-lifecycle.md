# Goalrail Project Lifecycle

Goalrail does not have one universal source of truth. Each artifact owns a
specific decision, and transitions keep explicit owner and effect gates.

## Artifact order and ownership

| Stage | Artifact | Owns | Does not own |
|---|---|---|---|
| Request | owner statement and source references | the initial evidence and question | Goalrail's interpretation |
| Context | OpenSpec change `context.md`; canonical `ContextPack` validation | bounded local/external claims, provenance, relevance, material unknowns, stop outcome | desired result or effect permission |
| Intent | versioned `intent.md` | desired outcomes, non-goals, observable success signals, active owner confirmation | implementation design or tool authority |
| Compilation | `proposal.md`, delta specs, `design.md`, `tasks.md` | traceable implementation plan for one change | runtime state |
| Implementation | Go code, tests, docs, and adapters | executable behavior and local verification | merge, deployment, or activation approval |
| Durable decisions | `docs/decisions/*.md` | expensive architectural and ownership choices | task progress |
| Runtime evidence | version-specific Goalrail JSONL | assignment, bounded lineage, checks, telemetry references, judgments, terminal state, corrections, verdict inputs | detailed provider observations |
| Provider telemetry | Langfuse | detailed model/tool observations, token/cost/timing data | Goalrail assignment or owner judgment |
| Canonical capability | `openspec/specs/**` after sync/archive | current accepted requirements | historical change rationale |
| History | `openspec/changes/archive/**` and Git/PR metadata | why a completed change existed and its stable change ID | live runtime lookup paths |

## Transition gates

1. Explore only to gather evidence and reduce ambiguity. Create a bounded
   Context Pack; external research is justified only by a material unknown that
   can change intent or the current decision.
2. Compile the durable candidate Intent Snapshot in concise, model-legible
   English. Before confirmation, render a separate plain-language view in the
   resolved owner language. Use an explicit owner preference first, then an
   explicit project or workflow preference, then the current conversation
   language, and finally English as fallback; no language is universally
   hard-coded. The view explains the owner's experience, visible behavior,
   material boundaries, and recognizable success without unnecessary schema or
   implementation detail. It must faithfully cover the exact version's three
   semantic groups. The owner actively confirms that version; silence and
   continued conversation do not confirm it.
3. Generate proposal, specs, design, and tasks only from confirmed intent. Use
   the project-local `goalrail-intent` schema and the dependency order returned
   by OpenSpec.
4. Apply one reversible vertical slice. Local writes are bounded by the change;
   credentials, external calls, deploys, publications, and other effects keep
   their separate authority gates.
5. Validate the implementation with focused tests, repository-wide checks, and
   strict OpenSpec validation. Evidence of a check is not owner acceptance.
6. Commit, push, open a PR, merge, sync canonical specs, and archive only when
   each action is explicitly authorized. Significant PRs carry the stable
   OpenSpec change ID in their body or repository metadata; individual small
   commits do not need it.
7. A runtime manifest is frozen before its first real assignment. Activation is
   a separate owner decision. Already-started runs remain governed by their
   frozen WorkSpec/manifest snapshot.

## Context stopping rule

Context collection ends as `sufficient`, `material_unknown`, or
`budget_exhausted`. `sufficient` means further evidence cannot materially
change the candidate intent or the decision currently being made. An unresolved
material unknown blocks confirmation; it is never filled by model inference.
Goalrail stores concise claims and references, not raw pages, prompts,
transcripts, credentials, or private source bodies.

## Change and amendment rules

- A material change to desired outcomes, non-goals, or success signals creates
  a new Intent Snapshot version and recompiles dependent artifacts.
- A wording-only amendment preserves semantics and provenance and is validated
  as such.
- A material measurement change creates a new manifest version. Evidence
  streams are never relabeled or rewritten.
- OpenSpec is a replaceable development-time compiler/provider. Its types do
  not enter Goalrail's canonical runtime domain.
- Langfuse is a replaceable telemetry provider. Goalrail decisions can be
  rebuilt without Langfuse score writes.

## Current local commands

The managed-project operator sequence is:

```text
init or migrate -> doctor -> exact setup consent -> confirmed intent/change
-> lineage begin -> bounded work -> lineage attach/inspect -> verify-lineage
```

The complete v0.2 machine contract is in [contracts-v0.2.md](contracts-v0.2.md).
Clean-machine setup and exact authorization are in [install.md](install.md), and
the one-project legacy transition is in
[migration-v0.2.md](migration-v0.2.md). OpenSpec development commands for this
repository continue to use the pinned, telemetry-disabled prefix in
`AGENTS.md`; that development pin is not a substitute for a managed setup plan.

Canary v1 and v2 are selected explicitly by manifest version; v2 instructions
live in `canary/intent-canary-v0/operator-flow-v2.md`. Both real canaries remain
unactivated.
