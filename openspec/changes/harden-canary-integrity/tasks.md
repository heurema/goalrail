## 1. Cross-Process Evidence Transactions

- [x] 1.1 Add the minimal macOS/Linux advisory-lock adapter and refactor evidence append, read, and verification to hold the file lock across each complete transaction without changing the JSONL format.
- [x] 1.2 Add a synchronized subprocess regression test proving distinct concurrent events remain present exactly once and the digest chain verifies; add an explicit unsupported-platform path and Linux cross-compilation check.
- [x] 1.3 Acquire the reader lock before the first evidence-path existence check and add a regression proving a reader waits for an in-progress first append.

## 2. Wrong-Join Projection

- [x] 2.1 Project terminal `SESSION_CONFLICT` lineage as `CanaryLineageWrong` while preserving existing pending and unresolved outcomes for other reason codes.
- [x] 2.2 Add operator/report regression tests proving one session conflict produces `WrongJoins == 1` and `STOP`, while one ordinary unresolved link remains below its threshold.
- [x] 2.3 Preserve a detected session conflict across later lineage retries and add a regression proving resolution exhaustion cannot erase the wrong-join `STOP`.

## 3. Wording-Only Provenance

- [x] 3.1 Require wording-only amendments to preserve source-evidence values and each stable semantic item's evidence-reference set while allowing harmless order and statement changes.
- [x] 3.2 Add validation tests for source-evidence replacement, item-reference mutation, accepted wording changes, and accepted reordering.

## 4. Verification

- [x] 4.1 Run formatting, focused tests, `go vet ./...`, `go test -race ./...`, and Linux cross-compilation without adding a dependency.
- [x] 4.2 Run the Python correlation tests and strict OpenSpec validation, confirm all tasks complete, and verify that no real canary evidence or activation action was created.
- [x] 4.3 Re-run focused regressions, the full race suite, Python correlation tests, cross-compilation, and strict OpenSpec validation after the PR #4 review fixes.
