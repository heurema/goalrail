## Why

Archiving the completed intent-canary change removes the active directory that previously held the canary's default evidence path, while the retained operator guidance still names that removed path. Goalrail needs a stable runtime-owned location and current guidance that remain valid across the replaceable OpenSpec lifecycle.

## What Changes

- Establish `canary/intent-canary-v0` as the stable home for the frozen manifest, current operator guidance, and default append-only evidence path.
- Preserve the dated OpenSpec archive as historical planning evidence without resolving runtime paths through it.
- Add regression coverage that the command and current operator guidance use the same provider-neutral path even when no `openspec/` directory exists.
- Add the confirmed change ID to PR #5 traceability before merge.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Keep the frozen manifest and evidence path under a stable runtime root. | OUT-1, SIG-1, SIG-2, SIG-4 | NG-1, NG-2, NG-3 |
| Publish current operator guidance beside the runtime contract and remove current references to the retired active-change path. | OUT-2, SIG-1 | NG-1, NG-2 |
| Preserve the archived OpenSpec artifacts and validate the new confirmed change. | OUT-3, SIG-3, SIG-5 | NG-3, NG-4 |
| Cover the runtime/documentation boundary with focused regression tests. | OUT-1, OUT-2, SIG-1, SIG-2, SIG-4, SIG-5 | NG-1, NG-2, NG-4 |

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `intent-canary`: Require runtime-owned canary artifacts and current operator guidance to remain stable and consistent across development-provider archive operations.

## Impact

- `canary/intent-canary-v0`: current frozen manifest and operator guidance.
- `cmd/goalrail-canary`: default evidence-path behavior and regression coverage.
- `internal/domain`: manifest-artifact path coverage.
- OpenSpec planning records and PR #5 traceability.
- No new dependency, service, external API, credential, or deployment surface.

## Non-Goals

- Do not activate the real 15-change canary or allocate ordinal 1.
- Do not change the frozen manifest rules, rotation, evidence semantics, or verdict thresholds.
- Do not add an artifact registry, database, service, or provider abstraction.
- Do not implement admission evidence or check-set freezing in this repair.
