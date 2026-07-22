## Context

PR #5 archives the two completed OpenSpec changes. The canary command previously defaulted its append-only evidence store to a directory owned by the active `intent-canary-v0` change, so the archive operation required moving that default to a stable runtime path. The frozen manifest was moved, but the archived operator flow still describes itself as the current surface and names the removed path. The initial migration also lacked its own confirmed Intent Snapshot.

The owner has now confirmed this bounded repair. OpenSpec remains a replaceable development-time provider; the canary manifest, current operator guidance, and future evidence are runtime-owned artifacts. The real canary is still frozen and has no repository evidence chain.

## Goals / Non-Goals

**Goals:**

- Keep the frozen manifest, current operator guidance, and default evidence path at `canary/intent-canary-v0`.
- Make documentation drift from the executable default fail deterministic validation.
- Preserve the complete dated archive while clearly distinguishing historical guidance from the current operator surface.
- Record owner-confirmed intent and PR traceability before merge.

**Non-Goals:**

- Activate the real canary or allocate an ordinal.
- Change any frozen measurement, assignment, evidence, or verdict rule.
- Add admission or check-set-freezing behavior.
- Add a registry, database, service, or generic artifact resolver.

## Decisions

### 1. Keep one explicit runtime root

`canary/intent-canary-v0` is the stable runtime root. It contains `manifest-v1.md`, current `operator-flow.md`, and the default `events.jsonl` location. The command uses this path directly relative to the repository root.

Alternatives rejected:

- Continue using the active change directory: archiving would break the harness or require keeping a completed change active forever.
- Resolve the dated archive dynamically: runtime behavior would depend on an OpenSpec-specific lifecycle and ambiguous archive discovery.
- Add a generic artifact registry: one canary does not justify a new abstraction.

### 2. Preserve history but label it as history

The dated archive retains the original operator-flow artifact for change completeness. A short notice at its top marks it historical and links to the stable current file. The current file uses the provider-neutral default path. A deterministic test reads the current guidance and rejects both a missing current path and the retired active-change path.

Removing the historical file was rejected because it would make the archive incomplete. Treating the archived file as current was rejected because it can direct an operator to a nonexistent or different evidence chain.

### 3. Treat owner confirmation and PR review as separate gates

The confirmed Intent Snapshot authorizes proposal compilation for this requested result but grants no effect authority. PR #5 still requires review and merge; real canary activation remains a separate owner instruction after all development changes.

## Correlation and Evidence

- Intent identity: `STABILIZE-CANARY-RUNTIME-BOUNDARY`, version 1.
- Development traceability: OpenSpec change `stabilize-canary-runtime-boundary` and PR #5.
- Runtime identity remains the frozen manifest ID `intent-canary-v0`, version 1.
- This repair writes no `events.jsonl`; tests use temporary repositories and stores.
- The dated OpenSpec archive remains historical evidence, not a runtime lookup target.

## Measurement and Stop Conditions

The repair is acceptable only when all of the following hold:

- the command and current operator flow name exactly `canary/intent-canary-v0/events.jsonl`;
- the default-store regression passes without an `openspec/` directory;
- the stable manifest is byte-identical to the archived frozen manifest;
- strict OpenSpec validation, `go vet ./...`, `go test -race ./...`, and `git diff --check` pass;
- the manifest remains `frozen_not_activated`, no repository evidence file exists, and real assignment remains rejected.

Stop and return to intent review if satisfying the feedback would require changing frozen canary rules, implementing admission, or activating real work.

## Risks / Trade-offs

- **The manifest and historical archive contain duplicate bytes** → Treat the stable copy as the current runtime contract and the dated copy as immutable change history; compare them mechanically.
- **Current and historical operator documents can be confused** → Put an explicit historical notice in the archived copy and test only the stable current document as executable guidance.
- **A future canary family may need discovery** → Keep the explicit path for v0 and revisit only when a second family creates measured duplication.

## Rollback

Before real activation, revert the PR commits. There is no real evidence to migrate or external system to clean up. Do not delete or rewrite evidence if activation ever occurs; that requires a separately designed migration.

## Open Questions

None.
