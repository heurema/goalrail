## 1. Traceable failing-before regressions

- [x] 1.1 Create `proof.md` with one row for each CTX-3 through CTX-11 finding, including the PR discussion URL, owning package, focused test name, expected stable reason, merge-commit result, remediation result, and implementation reference.
- [x] 1.2 Add focused `internal/admission` regressions for forged branch owner authority, non-`passed` terminal receipts, candidate or malformed intent, nil-time provenance, and repository-root `.` matching; keep each test runnable against the reviewed public behavior.
- [x] 1.3 Add focused `internal/admissiongit` and `internal/githubadmission` regressions for packet evidence omitted from completeness, non-bootstrap project-ID replacement, and unverified bootstrap/setup-profile bindings.
- [x] 1.4 Add a focused `internal/setup` regression proving a same-size different-digest executable destination produces an incomplete conflict plan and no `select-executable` mutation while current and mode-only cases remain supported.
- [x] 1.5 In a temporary detached worktree at `6a896355958874a4989eec131c6cc20a24921995`, apply only the new regression-test changes, run every named test, and record in `proof.md` that all nine fail for the reviewed reason before removing the temporary worktree.

## 2. Total core inputs and semantic evidence

- [x] 2.1 Add bounded provider-neutral semantic-projection and provider-candidate inputs to the admission composition contract; keep them non-persistent and non-serializable, validate closed kinds and enums, and canonically sort or reject duplicates and single-valued conflicts.
- [x] 2.2 Preserve the admission purity boundary with tests proving the core still imports no filesystem, network, credential, provider, OpenSpec-adapter, project, or local-run package and never reads wall-clock time.
- [x] 2.3 Extend immutable Git collection to read the exact intent bytes, verify the raw-byte digest and identity/version, parse with the current Goalrail intent adapter, run `domain.ValidateIntentSnapshot`, and emit a satisfying projection only for an actively confirmed snapshot.
- [x] 2.4 Extend immutable Git collection to decode and canonically validate the exact terminal-receipt replica, rebind its digest and relation, and emit a satisfying projection only when status is exactly `passed`.
- [x] 2.5 Change completeness to consume the ephemeral effective view and require satisfying intent and receipt projections; add malformed, mismatched, unsupported-codec, duplicate, candidate, failed, blocked, unlinked, launch-failed, launch-attempted-unknown, verification-incomplete, and passed fixtures with existing stable reason codes.
- [x] 2.6 Require non-nil, non-zero trusted evaluation time whenever packet provenance exists, defensively guard every verifier time access, and add nil, zero, stale, future, untrusted, provider-free, and repeated-canonical-result tests that prove no accepted input can panic.

## 3. Declaration-bound governance and policy matching

- [x] 3.1 Refactor admission Git governance collection to load policy, bootstrap, and setup profile from both immutable revisions and verify every declared path, bound, schema, exact digest, canonical encoding where required, and project binding before returning a governance state.
- [x] 3.2 Add base/head fixtures that independently remove or mutate policy, bootstrap, and setup profile bytes and prove each mismatch returns canonical invalid governance without falling back to unmanaged or head-defined authority.
- [x] 3.3 Reject every non-bootstrap base/head `project_id` mismatch before selecting authority, retain the explicit no-base-declaration bootstrap path, and test both denial and valid unchanged/bootstrap cases.
- [x] 3.4 Treat normalized prefix matcher `.` as the repository-root sentinel before ordinary prefix matching; cover root and nested add, modify, rename, delete, mode, and submodule changes while preserving rule ID, priority, classification, required evidence kinds, and neighboring-prefix boundaries.

## 4. Same-invocation provider evidence and owner authority

- [x] 4.1 Extend the GitHub adapter to return a bounded in-memory candidate set alongside the canonical audit packet, with exact repository, pull request, base/head, adapter, digest, observation-time, actor, authority, and outcome bindings and no raw body or secret-shaped content.
- [x] 4.2 Canonicalize latest-per-actor current-head review semantics so later dismissal, neutral state, or `CHANGES_REQUESTED` shadows earlier approval; retain explicit empty review/comment indexes and add stale-head, authority-unavailable, and actor-permission fixtures.
- [x] 4.3 Build the closed ephemeral mapping for pull request, review index, check set, owner decision, and closure candidates; reject unknown, stale, duplicate, conflicting, or range-mismatched candidates and prove the view is never persisted, cached, or emitted as lineage events.
- [x] 4.4 Replace branch-event owner authorization with same-invocation provider authorization against the valid base policy; require exact current head, authenticated actor and authority, supported outcome, and `allow`, while treating `reject`, branch labels, repository files, and serialized `authenticated=true` claims as non-authorizing.
- [x] 4.5 Make the prepared GitHub shared-admission path collect and verify within one trusted process invocation, keep standalone packet and local/hook entrypoints candidate-empty and advisory, and preserve canonical audit output without treating it as authority.
- [x] 4.6 Add an end-to-end fixture where complete committed semantics plus current authenticated GitHub candidates reaches `VALID`, while missing, stale, conflicting, packet-only, forged-owner, dismissed-review, rejected-owner, and hook-bypassed variants return the specified canonical non-valid results without repository writes.

## 5. Executable setup-plan honesty

- [x] 5.1 Change executable-destination planning so any existing different digest, including the same-size case, emits `EXECUTABLE_DESTINATION_CONFLICT`, marks the plan incomplete, and emits no authorizable replacement mutation.
- [x] 5.2 Preserve absent-destination selection, already-current no-op, and same-digest mode-only repair with exact pre-state binding; run plan/apply/rollback tests proving every complete plan remains executable and recoverable.

## 6. Integrated proof and owner stop gate

- [x] 6.1 Run `gofmt` on changed Go files and the focused suites `go test ./internal/admission ./internal/admissiongit ./internal/githubadmission ./internal/setup`; fix only deterministic failures inside the confirmed slice.
- [x] 6.2 Run the canonical admission corpus and the new effective-view fixtures at least twice, requiring byte-identical results independent of candidate order, map order, checkout path, local-hook presence, and serialized packet order.
- [x] 6.3 Run `go test ./...` and `go vet ./...`; stop at the first persistent deterministic failure rather than widening the change or weakening a required relation.
- [x] 6.4 Run `OPENSPEC_TELEMETRY=0 npx --yes @fission-ai/openspec@1.6.0 validate admission-hardening-after-pr62 --strict` after implementation and keep every delta requirement and task reference coherent.
- [x] 6.5 Run `git diff --check`, inspect the final diff for unrelated, generated, dependency, or infrastructure changes, and verify the prepared workflow remains inactive with no required-check, ruleset, credential, release, installation, deploy, or Deltacue action.
- [x] 6.6 Complete `proof.md` with failing-before and passing-after receipts for all nine findings, exact commands actually run, untested scope and remaining risks, then stop for owner review before staging, commit, push, PR creation, review-thread mutation, activation, release, or installation.
