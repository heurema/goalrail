# ADR-0002 — Single-writer execution and advisory panels

Status: accepted
Date: 2026-04-14

## Context

Goalrail must support two different kinds of parallel work:
1. different tasks executing concurrently
2. the same task being reviewed or explored by multiple runtimes in parallel

If these are modeled as one thing, trust boundaries collapse.
The product also needs risk-based review depth, quorum-style advisory decisions, and comparative exploration without turning execution into uncontrolled multi-writer behavior.

## Decision

Goalrail separates writable execution from advisory multi-runtime reasoning.

### Writable execution
- one `Run` uses one primary writer runtime
- parallel execution across different tasks uses `ExecutionGroup`
- each writable run has isolated workspace lineage

### Advisory reasoning
- one task may attach one or more `AdvisoryPanel` records
- advisory modes may include `panel`, `quorum`, `verify`, and `diverge`
- advisory outputs normalize into `ConsensusRecord`
- advisory panels are inputs to Gate, not replacements for Gate

### Risk and policy
- each task gets an explicit risk level
- risk affects review depth and advisory fan-out
- policy may narrow exposure beyond risk defaults
- sensitive tasks may require `single-vendor-only`, `local-only`, or human signoff

## Consequences

### Positive
- audit trail stays clean because one run has one writer
- multi-runtime strength is preserved for review, verification, and exploration
- scheduler logic and advisory logic stay separate
- high-risk tasks can get deeper review without corrupting execution lineage

### Negative
- users may want more automation than the kernel should allow
- comparative modes like `diverge` require more orchestration and clearer cost controls
- risk and policy defaults must be made inspectable to avoid hidden routing logic

## Not now

This ADR does not imply:
- multiple writers inside one mutable run
- automatic majority merge from advisory panels
- unrestricted multi-vendor exposure for sensitive inputs
- swarm-style autonomy as a default execution model
