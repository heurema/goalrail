## 1. Runtime Guidance Boundary

- [x] 1.1 Publish the current operator flow under `canary/intent-canary-v0` with the executable default evidence path.
- [x] 1.2 Mark the archived operator flow as historical and point readers to the current runtime-owned guidance.
- [x] 1.3 Update the runtime-boundary decision record with the confirmed intent and selected source-of-truth boundary.

## 2. Regression Evidence

- [x] 2.1 Add deterministic coverage that current operator guidance matches the executable default and excludes the retired active-change path.
- [x] 2.2 Verify the frozen manifest remains identical, no repository evidence file exists, and real assignment remains rejected.

## 3. Validation and Review Traceability

- [x] 3.1 Run strict OpenSpec validation, `go vet ./...`, `go test -race ./...`, and `git diff --check`.
- [x] 3.2 Add `stabilize-canary-runtime-boundary` to PR #5 traceability, commit and push the repair, and re-read unresolved review state.
