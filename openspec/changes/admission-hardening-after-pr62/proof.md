# Remediation Proof

- **Change:** `admission-hardening-after-pr62`
- **Reviewed merge:** `6a896355958874a4989eec131c6cc20a24921995`
- **Status:** implementation complete; stopped at the owner gate before staging, commit, push, PR, activation, release, or installation

| Finding | Review discussion | Owning package | Focused test | Expected stable result | Merge-commit result | Remediation result | Implementation reference |
|---|---|---|---|---|---|---|---|
| CTX-3 | https://github.com/heurema/goalrail/pull/62#discussion_r3722108655 | `internal/admission` | `TestCTX3BranchOwnerDecisionCannotAuthorizeAdmission` | `owner_decision_missing` | failed: a committed owner-decision event with an allowed actor reached `VALID` | passed | `buildEffectiveView`; `evaluateCandidates` |
| CTX-4 | https://github.com/heurema/goalrail/pull/62#discussion_r3722108695 | `internal/admission`, `internal/githubadmission` | `TestCTX4AuthenticatedProviderCandidatesCompleteEffectiveView`; `TestCTX4CandidatesBindTheExactRangeAndAuthenticatedAuthority`; `TestCTX4LatestCurrentHeadReviewPerActorDecidesTheOwnerBoundary` | `VALID` only with current authenticated candidates | failed: authenticated packet evidence for the pull request, review index, and check set left the work unit `MISSING` | passed | `EffectiveView`; `EvaluateCompleteness`; `providerCandidates`; `selectOwnerDecision` |
| CTX-5 | https://github.com/heurema/goalrail/pull/62#discussion_r3722108709 | `internal/admission`, `internal/admissiongit` | `TestCTX5OnlyPassedTerminalReceiptSatisfiesLineage` (both packages) | `receipt_missing` | failed: a terminal-receipt target that decodes to a non-receipt fixture object reached `VALID` | passed | `projectTerminalReceipt`; `evaluateProjections` |
| CTX-6 | https://github.com/heurema/goalrail/pull/62#discussion_r3722108716 | `internal/admission`, `internal/admissiongit` | `TestCTX6NonBootstrapProjectIDChangeIsInvalid` (both packages) | `declaration_invalid` | failed: a head declaration naming a different `project_id` reached `VALID` | passed | `resolveAuthority` |
| CTX-7 | https://github.com/heurema/goalrail/pull/62#discussion_r3722108733 | `internal/admission`, `internal/admissiongit` | `TestCTX7OnlyConfirmedIntentSatisfiesLineage` (both packages) | `intent_unconfirmed` | failed: an intent target that does not parse as a confirmed snapshot reached `VALID` | passed | `projectIntent`; `readIntentArtifact`; `evaluateProjections` |
| CTX-8 | https://github.com/heurema/goalrail/pull/62#discussion_r3722108662 | `internal/setup` | `TestCTX8DifferentExecutableDigestIsPlanningConflict` | `EXECUTABLE_DESTINATION_CONFLICT` | failed: same-size different-digest destination produced a complete plan with no issue | passed | `inspectExecutableDestination` |
| CTX-9 | https://github.com/heurema/goalrail/pull/62#discussion_r3722108670 | `internal/domain`, `internal/admission` | `TestCTX9AdmissionPacketRejectsProvenanceWithoutEvaluationTime`; `TestCTX9ProvenanceWithoutEvaluationTimeNeverPanics`; `TestCTX9FutureAndUnauthenticatedProvenanceAreUntrusted`; `TestCTX9ExpiredTimeBoundEvidenceIsStale`; `TestCTX9ProviderFreeEvaluationNeedsNoTrustedTime` | validation rejection or `trusted_time_missing`; never panic | failed: validation accepted nil-time provenance and verification panicked at `evaluate.go:413` | passed | `ValidateAdmissionPacket`; `verifyPacketEvidence`; `hasBoundedEvaluationTime` |
| CTX-10 | https://github.com/heurema/goalrail/pull/62#discussion_r3722108721 | `internal/admissiongit` | `TestCTX10EveryDeclarationBoundArtifactIsVerified` | `declaration_invalid` | failed: substituted bootstrap bytes were not retained as invalid head governance | passed | `verifyDeclarationBoundArtifacts`; `readDeclaredArtifact` |
| CTX-11 | https://github.com/heurema/goalrail/pull/62#discussion_r3722108727 | `internal/admission` | `TestCTX11RepositoryRootPrefixMatchesEveryNormalizedChangeKind`; `TestCTX11NeighborPrefixStillUsesPathBoundary` | root rule classification, priority, and required evidence preserved | failed: normalized paths had no root rule ID; neighboring boundary case also fell through | passed | `ruleMatches` |

## Failing-before method

CTX-8, CTX-9, and CTX-11 were run at the reviewed merge commit as written, in a
temporary detached worktree that was removed afterwards.

CTX-3 through CTX-7 and CTX-10 assert behavior that the merge commit has no API
for: `admission.Input` there carries no provider candidates and
`admission.FrozenRange` carries no semantic projections, so the shipped tests
cannot compile against it. Each claim was therefore restated against the
reviewed public behavior in one throwaway file per package
(`TestCTXBefore3…`, `TestCTXBefore4…`, `TestCTXBefore5And7…`,
`TestCTXBefore6…`, `TestCTXBefore10…`), run in a temporary detached worktree at
`6a896355958874a4989eec131c6cc20a24921995`, and the worktree was removed. Each
one failed for the reviewed reason; the observed merge-commit results are in the
table above. The restatement is the same claim in the old vocabulary, not a
weaker one — every restated test asserts the exact classification and reason the
shipped test asserts.

## Commands actually run

- Baseline before new regressions: `go test ./internal/domain ./internal/admission` — passed.
- Failing-before at the exact reviewed merge in a temporary detached worktree: `go test ./internal/domain ./internal/admission -run 'TestCTX(9|11)' -count=1` — failed for CTX-9 with validation acceptance and a nil-pointer panic, and for CTX-11 with missing root rule IDs; the temporary worktree was removed.
- CTX-8 failing-before at the exact reviewed merge in a temporary detached worktree: `go test ./internal/setup -run '^TestCTX8DifferentExecutableDigestIsPlanningConflict$' -count=1` — failed because the plan state was `complete`; the temporary worktree was removed.
- Failing-before for CTX-3 through CTX-7 and CTX-10 at the exact reviewed merge in a temporary detached worktree: `go test ./internal/admission ./internal/admissiongit -run 'TestCTXBefore' -count=1` — all five restated tests failed for the reviewed reasons; the temporary worktree was removed.
- Focused remediation suites: `go test ./internal/admission ./internal/admissiongit ./internal/githubadmission ./internal/setup -count=1` — passed.
- Repeated canonical-result equality and candidate/projection order independence: `go test ./internal/admission -count=2` — passed (`runSerializedEntrypoint` drops the ephemeral evidence, then re-supplies it in reverse order).
- Repository-wide: `go test ./...` — passed. `go vet ./...` — passed. `gofmt -l internal cmd` — empty.
- Shuffled repeats: `go test -count=2 -shuffle=on ./internal/admission ./internal/admissiongit ./internal/githubadmission ./internal/setup ./cmd/gr` — passed.
- OpenSpec validation: `OPENSPEC_TELEMETRY=0 npx --yes @fission-ai/openspec@1.6.0 validate admission-hardening-after-pr62 --strict` — passed.
- Whitespace validation: `git diff --check` — passed.

## Behavior changes an owner should read before merge

- `gr verify-lineage` over a packet at rest is now advisory for every
  provider-owned relation. It returns `MISSING` / `OWNER_DECISION_MISSING` where
  a live authenticated approval is required. This is the intended compatibility
  break from CTX-3: a serialized `authenticated=true` claim is not authority.
- The authoritative shared path is the new internal `gr github-verify`
  command, which observes the provider and verifies the frozen range inside one
  invocation. The prepared workflow now calls it instead of the previous
  `github-collect` + `verify-lineage` pair; the emitted packet remains audit
  evidence only. The workflow stays inactive and unrequired.
- `TestProjectLineageAdmissionScratchCanary` now asserts both paths: the
  packet-only path denies the owner relation, and the same range reaches `VALID`
  (and `BREAK_GLASS` for the exception head) through a stubbed authenticated
  provider observation.
- Base governance that is missing, malformed, or drifted now denies with
  `declaration_invalid` rather than `policy_conflict`, matching the design's
  canonical reason mapping.

## Deviation from the design

Design decision 9 lists materiality classification after provider-candidate
validation. Classification is still computed before graph binding, because graph
binding consumes the material path set. The denial precedence the design cares
about is unchanged: the new boundaries (semantic projections, then provider
candidates with the owner decision last) are evaluated after structural,
governance, and graph-binding validation, and no later evidence repairs an
earlier failure.

## Untested scope and remaining risks

- No real GitHub API call, workflow activation, required-check or ruleset
  change, release, installation, deployment, or Deltacue mutation was performed.
  The end-to-end provider fixtures use an in-process stub reader.
- The prepared workflow's new `github-verify` invocation has not been executed
  on a real runner; only its rendered text and the underlying command path are
  covered by tests.
- `readIntentArtifact` falls back to single-artifact intent parsing when the
  change directory has no `context.md` at the evaluated revision. A change whose
  schema requires a context pack but whose `context.md` was deleted at head
  therefore projects from the intent alone; the pair contract is still enforced
  wherever `context.md` exists.
