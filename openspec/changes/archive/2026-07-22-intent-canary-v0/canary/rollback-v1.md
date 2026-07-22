# Synthetic Rollback Receipt v1

## Scope

- Exercise: disable `intent-canary-v0` in an isolated temporary evidence store
- External effects: none
- Default repository canary store changed: no
- Real 15-change canary started or stopped: no
- Codex hook, Langfuse config, environment, or credentials changed: no

The live manifest remains `frozen_not_activated`. This receipt proves the
rollback mechanism without treating a synthetic assignment as a real ordinal.

## Verified behavior

1. `disable` appends one `canary_stopped` event after the complete existing
   JSONL bytes; the prior chain remains an exact byte prefix.
2. Service-level and store-level guards reject every new assignment after that
   event, and a rejected attempt writes no bytes.
3. Previously assigned change evidence remains readable and can still receive
   terminal evidence.
4. The verified aggregate report returns `STOP` with
   `assignments_stopped=true` while preserving existing counts.
5. A repository-local `.codex/langfuse.json` sentinel remains byte-identical.
   Independent Codex run-context and Langfuse-reference adapters remain usable
   after the selected store is stopped.

The fifth check proves code-path isolation without calling or mutating any live
external service. Goalrail has no persistent hook installer or global Langfuse
switch in this slice.

## Test evidence

- `internal/evidence.TestStoreStopMarkerBlocksDirectAssignmentAppend`
- `internal/operator.TestDisableStopsNewAssignmentsAndPreservesExistingEvidence`
- `cmd/goalrail-canary.TestCommandRunsSyntheticLifecycleWithoutManualSessionID`
- `internal/integration.TestDisableIsRepositoryLocalAndLeavesUnrelatedAdaptersUsable`

## Reproduction

```sh
mkdir -p /tmp/goalrail-rollback-cache /tmp/goalrail-rollback-tmp
GOCACHE=/tmp/goalrail-rollback-cache \
  GOTMPDIR=/tmp/goalrail-rollback-tmp \
  go test -v \
  ./internal/evidence \
  ./internal/operator \
  ./cmd/goalrail-canary \
  ./internal/integration \
  -run 'Test(StoreStopMarkerBlocksDirectAssignmentAppend|DisableStopsNewAssignmentsAndPreservesExistingEvidence|CommandRunsSyntheticLifecycleWithoutManualSessionID|DisableIsRepositoryLocalAndLeavesUnrelatedAdaptersUsable)$'
```
