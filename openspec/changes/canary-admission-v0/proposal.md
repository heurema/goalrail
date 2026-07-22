## Why

The frozen canary rules already require eligibility to be decided before variant reveal and checks to be selected before terminal verification, but the current harness jumps directly to assignment and records checks only at delivery. This slice closes those evidence gaps before any real canary activation.

## What Changes

- Add an explicit append-only admission decision for every considered candidate, using the existing immutable `change_id` and recording `eligible` or `excluded`, actor, time, source, and categorical reason.
- Make an eligible admission and its ordinal/variant assignment one serialized evidence transaction; excluded admissions contain no assignment or variant and do not advance the ordinal.
- Add an append-only check-set freeze before terminal verification, with bounded corrections that preserve earlier events and cannot remove a selected check.
- Require delivered check evidence to reference exactly the effective frozen set, while leaving abandonment and owner-assessment semantics unchanged.
- Project admission, exclusion reasons, effective frozen checks, assignment, and terminal evidence through operator inspect/report surfaces.
- Keep all new behavior synthetic-only; direct and operator-level real admission or assignment remain rejected until a separate owner activation change.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Record eligible/excluded admission before variant reveal, keyed by `change_id`. | OUT-1, SIG-1, SIG-2 | NG-1, NG-3, NG-4 |
| Atomically assign only eligible candidates and fail closed on duplicate or conflicting decisions. | OUT-2, SIG-1, SIG-2, SIG-5 | NG-1, NG-2, NG-5 |
| Freeze a non-empty check set before terminal verification and validate delivery against it. | OUT-3, SIG-3, SIG-4 | NG-2, NG-4 |
| Expose the new evidence through existing inspect/report paths while retaining the serialized hash chain. | OUT-4, SIG-4, SIG-5 | NG-5, NG-6 |
| Exercise the complete path synthetically and reject every real attempt. | OUT-5, SIG-6, SIG-7 | NG-1, NG-2, NG-6 |

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `intent-canary`: Add candidate admission and pre-verification check-set freezing to the existing canary lifecycle and evidence contract.

## Impact

- Canonical Go domain types gain provider-neutral admission and frozen-check payloads and projections.
- The append-only evidence validator gains admission, check-freeze, and bounded check-correction transitions while preserving the existing process-shared transaction boundary.
- The operator service and `goalrail-canary` CLI gain excluded-candidate and check-freeze actions; an eligible start records admission and assignment atomically.
- Current operator guidance and deterministic synthetic fixtures change to exercise the full admission-to-assessment path.
- No new dependency, service, database, workflow engine, provider contract, credential path, or deployment surface is introduced.

## Non-Goals

- Do not activate the real canary or allocate a real ordinal.
- Do not change eligibility rules, rotation, metrics, thresholds, owner assessment, or abandonment semantics.
- Do not automate eligibility judgment or ingest unbounded content.
- Do not introduce new infrastructure or leak OpenSpec/provider types into the canonical domain.
- Do not relocate runtime artifacts into OpenSpec lifecycle directories.
