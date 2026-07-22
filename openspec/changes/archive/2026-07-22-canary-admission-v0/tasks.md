## 1. Canonical Evidence Model

- [x] 1.1 Add provider-neutral admission decision and frozen check-set payloads, event kinds, and inspect/report projections without adding a second candidate identity.
- [x] 1.2 Add focused domain tests for decision values, exclusion reporting, and unchanged assigned-change measurement denominators.

## 2. Append-Only Transition Enforcement

- [x] 2.1 Validate atomic eligible admission/assignment, ordinal-neutral exclusion, duplicate/conflicting decisions, stopped-canary behavior, and the pre-activation synthetic-only boundary inside the serialized store transaction.
- [x] 2.2 Validate initial check freeze, exact delivery-set equality, append-only superset correction before terminal evidence, and fail-closed removal or post-terminal correction.
- [x] 2.3 Add store tests proving rejected transitions leave bytes unchanged and concurrent accepted evidence retains a canonical hash chain.

## 3. Operator Vertical Slice

- [x] 3.1 Make `start` record eligible admission and add excluded admission, check freeze, and check correction service actions using the shared evidence model.
- [x] 3.2 Extend inspect/report projections and service tests so exclusions remain outside ordinals and denominators while effective checks remain visible.

## 4. CLI and Team Workflow

- [x] 4.1 Add bounded CLI flags/actions for eligible reason, exclusion, check freeze, and pre-result check correction, with no real activation path.
- [x] 4.2 Update current operator guidance and command tests to exercise exclusion, consecutive eligible ordinals, frozen checks, delivery, and explicit real-attempt rejection.

## 5. End-to-End Proof

- [x] 5.1 Update deterministic synthetic fixtures and integration tests for admission and check-freeze evidence while preserving automatic lineage and independent owner assessment.
- [x] 5.2 Run strict OpenSpec validation, `go vet ./...`, `go test -race ./...`, `git diff --check`, and verify the manifest remains `frozen_not_activated` with no repository default evidence file.
