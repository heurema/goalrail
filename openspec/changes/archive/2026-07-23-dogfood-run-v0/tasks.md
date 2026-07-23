## 1. WorkSpec Contract

- [x] 1.1 Add closed provider-neutral WorkSpec domain types, the fixed `trusted-local-provider-enforced-v0` posture, and explicit field/list size bounds in `internal/domain`.
- [x] 1.2 Implement strict JSON decoding and validation for required fields, unknown fields, prohibited provider/OpenSpec data, unsafe payload patterns, duplicate IDs, and unsupported authority; cover each rejection with focused tests.
- [x] 1.3 Implement repository and intent resolution that verifies the canonical Git root, full pinned commit, confirmed Intent Snapshot ID/version/digest, and repository-contained path scope; add local fixture tests for stale, missing, candidate, and escaping inputs.
- [x] 1.4 Implement canonical ordering, standard-library JSON serialization, SHA-256 identity, and frozen-content verification; add golden tests proving equivalent set order has one digest and material changes require a new version.

## 2. Local State and Worktree Evidence

- [x] 2.1 Add the external state-root resolver and owner-only write-once filesystem store with atomic creation, immutable reread verification, and no target-worktree storage; test duplicate, partial, and conflicting writes.
- [x] 2.2 Implement the bounded read-only Git observer for pinned `HEAD`, porcelain status metadata, dirty/untracked path hashes, file modes, and observation digest without retaining patches or source bodies; test limit and Git-read failures.
- [x] 2.3 Implement baseline-to-terminal comparison that distinguishes pre-existing dirty state, reports sorted run-time changed paths, and flags new or further-modified out-of-scope paths; cover clean, dirty, untracked, and scope-violation fixtures.

## 3. Preparation and One-Shot Lifecycle

- [x] 3.1 Implement preparation and inspection services that validate, freeze, baseline, persist, and display the exact canonical WorkSpec and digest without creating a run ID, invoking an adapter, or mutating the target worktree.
- [x] 3.2 Add provider-neutral run, launch-claim, provider-observation, check-result, worktree-delta, and terminal-receipt types with closed status and reason enums plus bounded-content validation.
- [x] 3.3 Add the adapter interface and deterministic fixture adapter, keeping provider configuration and rendered launch input outside canonical WorkSpec and receipt state.
- [x] 3.4 Implement the activation check and atomic launch claim so denial happens before run-ID generation, concurrent starts permit at most one fixture invocation, and any existing claim blocks retry or replacement.
- [x] 3.5 Implement provider-outcome handling for verified completion, denial, launch failure, unlinked identity, and orphaned `launch_attempted_unknown` state without retry, heuristic repair, or background continuation.
- [x] 3.6 Implement explicit finish processing that requires one result per frozen check, compares terminal worktree state with the baseline, prevents passing on missing checks or scope violations, and writes at most one bounded terminal receipt.

## 4. Codex Adapter Boundary

- [x] 4.1 Add a versioned adapter-local Codex context envelope binding WorkSpec digest, run ID, and canonical repository root without changing the existing canary `RunContext` schema.
- [x] 4.2 Reuse the existing lifecycle-correlation rules to accept only a matching provider-authoritative root-session receipt and preserve sticky missing, malformed, or conflicting identity outcomes; add concurrency and conflict tests.
- [x] 4.3 Implement the unwired Codex process adapter behind an injected launcher, preserving inherited I/O, native sandbox and approval behavior, bounded outcome categories, and zero retained prompt/transcript/hook/source content; test it without invoking real Codex.

## 5. `gr` Operator Surface and Activation Boundary

- [x] 5.1 Add `cmd/gr` parsing and output for `prepare` and `inspect`, including explicit state-root selection and exact canonical snapshot/digest display; add command tests proving no provider call or worktree write.
- [x] 5.2 Add `start` and `finish` command parsing with adapter arguments kept outside WorkSpec, exact structured check results, bounded evidence references, and no automatic check, Git, external, or follow-on execution.
- [x] 5.3 Wire the production command to reject every real adapter start with `ACTIVATION_REQUIRED` before a run ID or launch claim, with no flag, environment variable, fallback adapter, or fixture mode that can bypass the gate.

## 6. End-to-End Verification

- [x] 6.1 Add deterministic service and command fixtures covering prepare-to-receipt success, concurrent duplicate start, provider denial/failure, missing/conflicting identity, incomplete checks, pre-existing dirty files, and out-of-scope changes.
- [x] 6.2 Add assertions and test doubles proving zero real provider launches, retries, additional agents, credential use, external effects, Git writes, background work, and post-receipt actions across every fixture path.
- [x] 6.3 Document the inert v0 command contract and activation boundary, run the full Go test suite and strict OpenSpec validation, and stop with implementation evidence before any real dogfood run or Git lifecycle action.
