# Scenario-to-test map — harness-init-v0

Every scenario in every delta of this change, against the deterministic test that
pins it. Written by walking the deltas one scenario at a time rather than by
sampling, because a scenario with no test is invisible from the other direction:
the suite passes, the requirement is unverified, and nothing says so.

Nine gaps were found and closed while walking it. They are named at the end
rather than quietly filled, so the next reader can see what this pass was worth.

Scenarios that state a prohibition — an act that must not happen — are marked
`prohibition` and pinned by the closest observable consequence, since a test
cannot prove the absence of code. Where that is the whole verification, it says
so.

- 94 scenarios across five deltas.
- 137 test functions in `internal/harness`, `internal/ambient`, and `cmd/gr`.

## harness-init (23)

| Scenario | Test |
|---|---|
| A fresh repository is initialized | `TestMaterializeInstallsTheOverlayIntoAFreshDirectory`, `TestInitCommandIsExplicitAndReportsItself` |
| Initialization is repeated | `TestMaterializeIsIdempotent` |
| The materialized overlay is the one the checking CLI accepts | `TestPinnedCLIAcceptsTheEmbeddedCanon` |
| A stock OpenSpec root is adopted | `TestEnsureConfigSwitchesTheStockSchemaAndKeepsEverythingElse`, `TestMaterializeTouchesNothingOutsideTheOverlay` |
| The configuration names a foreign custom schema | `TestEnsureConfigRefusesAForeignCustomSchema` |
| The configuration already names the Goalrail schema | `TestEnsureConfigLeavesTheGoalrailSchemaByteIdentical` |
| A scaffold is detected | `TestDetectionReadsConfigurationPresenceNotEnvironment`, `TestInitRefusesToRegisterWhereACommitCouldCarryIt` |
| The scaffold is named explicitly | `TestInitSelectsTheScaffoldByFlagAndByDetection/named_explicitly` |
| No scaffold is detected | `TestInitSelectsTheScaffoldByFlagAndByDetection/nothing_detected` |
| The scaffold registers only at user scope | `TestInitSelectsTheScaffoldByFlagAndByDetection/user-scope_scaffold_names_its_own_command` |
| The project settings path is not ignored | `TestInitRefusesToRegisterWhereACommitCouldCarryIt`, `TestIgnoreStateAnswersTheQuestionItCanAnswer`, `TestIgnoreStateRefusesWhenItCannotCheck` |
| The user asks for the ignore entries to be added | `TestInitRefusesToRegisterWhereACommitCouldCarryIt`, `TestAddIgnoreEntriesIsAdditiveAndIdempotent` |
| The marker's ignore entry is missing | `TestInitSelectsTheScaffoldByFlagAndByDetection/marker_exposure_is_a_notice` |
| The scaffold's stop-like event fires once per turn | `TestRetentionIsRegisteredAgainstASessionEndingEvent` |
| Re-initialization meets a registration naming a per-turn event | `TestARegistrationOnThePerTurnEventIsRepaired` |
| Re-initialization meets a stale registration | `TestConnectRepairsAStaleExecutable` |
| Re-initialization meets a correct registration | `TestRepairIsNotTriggeredForTheCurrentExecutable`, `TestClaudeCodeScopedRegistrationIsIdempotent` |
| A foreign handler shares the event | `TestRepairPreservesAForeignHandlerForTheSameEvent`, `TestClaudeCodeConnectionPreservesForeignHooks` |
| User-level configuration would be modified | `TestInitNeverWritesUserLevelConfiguration` (prohibition; pinned by the file staying byte-identical, including when it holds a registration initialization has reason to want gone) |
| No Node runtime is present | `TestTheHarnessCommandsNeedNoNodeRuntime`, `TestAnAbsentToolchainIsAFactNotAFault` |
| A Goalrail command would invoke the checking CLI | `TestTheHarnessCommandsNeedNoNodeRuntime` (prohibition; the three commands complete with an empty executable lookup path, which no invocation of a package runner could survive) |
| Observability is already configured | `TestObservabilityConfigurationNeverEntersTheRepository` |
| Observability is absent | `TestObservabilityAbsenceIsOptionalAndNeverLeaks` |

## harness-doctor (15)

| Scenario | Test |
|---|---|
| The repository is not initialized | `TestDiagnosisReportsAnUninitializedRepository` |
| The overlay is missing | `TestDiagnosisReportsAnUninitializedRepository`, `TestInspectReportsAMissingOverlay` |
| An overlay file has drifted | `TestDiagnosisReportsDriftPerFile` |
| The overlay is behind the canon | `TestDiagnosisReportsAnOverlayBehindTheCanon`, `TestBehindIsDistinguishedFromEdited` |
| A registration exists in the superseded scope | `TestInitNeverWritesUserLevelConfiguration` |
| A registration names a superseded event | `TestDiagnosisNamesASupersededEventRegistration` |
| Everything is working | `TestObservabilityAbsenceIsOptionalAndNeverLeaks`, `TestDoctorSeparatesHealthyFromNotForAMachine` |
| The diagnosis is consumed by a machine | `TestDoctorSeparatesHealthyFromNotForAMachine` |
| A next action names a command that does not exist | `TestEveryNextActionNamesARealCommand` (prohibition; every printed action is parsed and matched against the commands the CLI accepts) |
| The version string alone changes | `TestDiagnosisIgnoresTheVersionWhenJudgingCurrency` |
| The files differ while the version agrees | `TestDiagnosisReportsDriftPerFile`, `TestDiagnosisIgnoresTheVersionWhenJudgingCurrency` |
| The runtime is absent | `TestAnAbsentToolchainIsAFactNotAFault`, `TestTheHarnessCommandsNeedNoNodeRuntime` |
| The runtime is present | `TestDiagnosisReportsDriftPerFile` and every other case built with a present runtime |
| No observability is configured | `TestObservabilityAbsenceIsOptionalAndNeverLeaks` |
| Observability is configured | `TestObservabilityAbsenceIsOptionalAndNeverLeaks`, `TestObservabilityConfigurationNeverEntersTheRepository` |

## harness-update (10)

| Scenario | Test |
|---|---|
| The overlay is behind the canon | `TestUpdateRestoresAFileFromAnEarlierCanon` |
| The overlay is already current | `TestUpdateReportsAnAlreadyCurrentOverlay` |
| An update would run implicitly | `TestMaterializeLeavesALocalEditAlone`, `TestInitCommandIsExplicitAndReportsItself` (prohibition; initialization leaves a differing file alone and reports it, so no other command performs an update's work) |
| The previous state is wanted back | `TestUpdateDiscardsOnlyWhenAskedAndSaysSo` (recovers the replaced content from the state-root backup) |
| No runtime or network is available | `TestTheHarnessCommandsNeedNoNodeRuntime` |
| A template was edited locally | `TestUpdateRefusesOverALocalEdit` |
| The user chooses to discard local edits | `TestUpdateDiscardsOnlyWhenAskedAndSaysSo` |
| Drift and behind-ness coincide | `TestUpdateStopsOnDriftEvenWhenSomethingIsAlsoBehind`, `TestAnEditWinsOverBehindness` |
| The user reads the command's help | `TestUpdateHelpSaysItDoesNotUpdateTheBinary`, `TestUpdateSaysItDoesNotUpdateTheBinary` |
| A release lookup would be attempted | `TestTheHarnessCommandsNeedNoNodeRuntime` (prohibition; the update completes with an empty lookup path and the package makes no network call) |

## operator-cli-help (5)

| Scenario | Test |
|---|---|
| Operator requests top-level help | `TestHelpDescribesLifecycleAndCommandFlagsWithoutService` |
| Operator looks for the background surface | `TestHelpPresentsBackgroundSurfaceAndHidesTheHook` |
| Operator cannot tell untrusted from broken | `TestHelpPresentsTheDiagnosisAndWhatItDistinguishes` |
| A superseded command name is used | `TestTheSupersededNameStillWorksAndNamesItsSuccessor` |
| Help would advertise the hook entry point | `TestHelpPresentsBackgroundSurfaceAndHidesTheHook` (prohibition; the usage line and the command entries are read, which is where help lists commands) |

## ambient-connect (41)

| Scenario | Test |
|---|---|
| User connects once | `TestCodexConnectionIsIdempotentAndResidueFree`, `TestConnectionRequiresConsentAndIsPlannedFirst` |
| Registration scope follows the scaffold | `TestRegistrationScopeFollowsTheScaffold` |
| Connection is invoked for a scaffold that registers per repository | `TestConnectionRefusesTheRepositoryScopeScaffold` |
| Consent would be given on someone else's behalf | `TestInitRefusesToRegisterWhereACommitCouldCarryIt`, `TestIgnoreStateRefusesWhenItCannotCheck` |
| A scaffold distinguishes session-start occurrences | `TestClaudeCodeRegistersOnlyTheOpeningOccurrence` |
| An earlier registration fires on every occurrence | `TestClaudeCodeRepairsAnUnscopedRegistration` |
| Removal spans every registered occurrence | `TestClaudeCodeRemovalSpansTheScopedRegistration` |
| Removal spans a repository-scope registration | `TestDisconnectSpansBothScopes`, `TestClaudeCodeConnectionPreservesForeignHooks` |
| A registration names an event that fires once per turn | `TestARegistrationOnThePerTurnEventIsRepaired` |
| Removal covers the event that was registered | `TestRemovalCoversTheSupersededEvent` |
| The registration names a moved executable | `TestConnectRepairsAStaleExecutable` |
| A repair replaces the stale registration | `TestConnectRepairsAStaleExecutable`, `TestRepairRemovesEveryManagedBlock` |
| The registration already names the current executable | `TestRepairIsNotTriggeredForTheCurrentExecutable` |
| A repair meets a handler it did not add | `TestRepairPreservesAForeignHandlerForTheSameEvent` |
| A repair meets an event that is already current | `TestRepairLeavesAnEventThatIsAlreadyCurrent` |
| A repaired session-start entry stays scoped | `TestRepairKeepsSessionStartScoped` |
| A health remedy names connection | `TestHealthReportsWorkingAfterTheRepair`, `TestEveryNextActionNamesARealCommand` |
| A limitation is recorded from one scaffold's evidence | `TestRegistrationScopeFollowsTheScaffold`, `TestRetentionIsRegisteredAgainstASessionEndingEvent` (the per-scaffold profile is what carries the attribution) |
| A user-scope registration remains from the earlier arrangement | `TestInitNeverWritesUserLevelConfiguration` |
| A scaffold has not been exercised live | `TestHealthNamesWhatItCouldNotVerify` |
| Part of a scaffold's behaviour has been observed | `TestHealthReportsNoTrustStepWhereNoneWasObserved` (reports working while still naming what the run did not capture) |
| An observation was made under conditions that do not settle a question | `TestHealthNamesWhatItCouldNotVerify`, `evidence/approval-probe-2026-07-30.md` |
| Connection discloses the trust step | `TestConnectionNoticeMatchesWhatWasActuallyObserved` |
| A repair discloses that review applies again | `TestRepairNoticeMatchesWhatWasActuallyObserved`, `TestRepairNoticeDoesNotSendTheUserToASurfaceThatContradictsIt` |
| The scaffold's trust behaviour is unverified | `TestConnectionNoticeMatchesWhatWasActuallyObserved` (the unknown branch), `TestHealthNamesWhatItCouldNotVerify` |
| The scaffold documents no approval step | `TestTheDocumentedAbsentDisclosureStatesItsOwnLimit` |
| Connection is repeated on a working attachment | `TestConnectDoesNotCallARepairedAttachmentActive`, `TestCodexConnectionIsIdempotentAndResidueFree` |
| User disconnects | `TestCodexConnectionIsIdempotentAndResidueFree`, `TestDisconnectRemovesAConfigurationFileConnectionCreated` |
| Commands are repeated | `TestCodexConnectionIsIdempotentAndResidueFree`, `TestDisconnectOnAnUntouchedConfigurationDoesNothing`, `TestAddIgnoreEntriesIsAdditiveAndIdempotent` |
| Configuration would change without consent | `TestConnectionRequiresConsentAndIsPlannedFirst`, `TestInspectionNeverWritesAnything`, `TestInitNeverWritesUserLevelConfiguration` (prohibition) |
| Attachment is not yet trusted | `TestHealthDistinguishesEveryState` |
| Repository is not initialized | `TestHealthDistinguishesEveryState`, `TestDiagnosisReportsAnUninitializedRepository` |
| Health is judged in the scope the scaffold uses | `TestHealthReportsNoTrustStepWhereNoneWasObserved`, `TestHealthSurfacesAnUnreadableConfiguration` |
| A registration survives in the superseded scope | `TestInitNeverWritesUserLevelConfiguration` |
| Only part of the registration survives | `TestHealthReportsAHalfRemovedRegistration`, `TestConnectRepairsAPartialClaudeRegistration` |
| The registered executable is gone | `TestHealthRefusesToReportWorkingWhenTheBinaryIsGone` |
| Scaffold configuration cannot be read | `TestHealthSurfacesAnUnreadableConfiguration` |
| No scaffold is named | `TestHealthWithoutScaffoldChecksAllOfThem` |
| Trust freshness cannot be established | `TestHealthNamesWhatItCouldNotVerify` |
| Attachment is working | `TestHealthDistinguishesEveryState`, `TestHealthReportsNoTrustStepWhereNoneWasObserved` |
| Trust would be written by Goalrail | `TestNoTrustRecordIsEverWritten`, `TestRepairWritesNoTrustRecord` (prohibition) |

## Gaps this pass found and closed

1. **The overlay-behind verdict was untested at the diagnosis level.** It was covered at the overlay level only, and its next action differs from an edit's — the plain update rather than the one that discards. `TestDiagnosisReportsAnOverlayBehindTheCanon`.
2. **The machine-readable contract was untested.** Structured output was exercised incidentally; the exit status separating healthy from not was not. `TestDoctorSeparatesHealthyFromNotForAMachine`.
3. **Explicit scaffold selection was untested.** Detection was covered, the flag override was not. `TestInitSelectsTheScaffoldByFlagAndByDetection/named_explicitly`.
4. **The no-scaffold case was untested.** Installing the harness must not depend on which agent environment is present, and the report has to name what attaches later. Same test, `nothing_detected`.
5. **The user-scope-only case was untested.** Initialization must name the remaining consented command rather than implying the attachment is complete. Same test, `user-scope_scaffold_names_its_own_command`.
6. **The marker's ignore notice was untested.** It is deliberately a notice rather than a refusal, and nothing pinned that distinction. Same test, `marker_exposure_is_a_notice`.
7. **Credential containment was untested.** That no key reaches a report was covered; that none reaches repository content was not. `TestObservabilityConfigurationNeverEntersTheRepository`.
8. **The update command's help was untested.** The report's note was covered; the help text a user reads before running it was not. `TestUpdateHelpSaysItDoesNotUpdateTheBinary`.
9. **The documented-absent disclosure branch was unreachable and untested.** No supported scaffold is in that state today — one gate was observed to exist, the other observed not to — so the branch had no exercise at all. A synthetic profile now reaches it the way production would. `TestTheDocumentedAbsentDisclosureStatesItsOwnLimit`.

## Addendum: the pre-PR adversarial review

After this map was written, an adversarial review ran over the branch — five
independent reviewers, thirty-two pooled findings, fourteen verified by agents
prompted to refute them. Thirteen were confirmed. The code defects and their
regression tests:

| Confirmed defect | Fix pinned by |
|---|---|
| The hook entry point handled only `SessionStart` and `Stop`, so the `SessionEnd` registration this change ships made retention silently never fire on the second scaffold — while the diagnosis reported the attachment active | `TestHookRetainsAQuestionAtSessionEnd`, plus a live re-check on a scratch stand |
| A quoted schema value with a trailing comment misparsed — quotes were trimmed before the comment was cut — so a correctly configured repository was refused as running a foreign schema | `TestEnsureConfigReadsAQuotedValueWithATrailingComment`, `TestEnsureConfigTreatsACommentOnlyValueAsUnnamed` |
| A superseded overlay file made every update fail its own verification forever, in a state re-materializing cannot change | `TestUpdateConvergesWithASupersededFilePresent` |
| The update report never named the canon it moved from, which the delta requires | `TestUpdateNamesTheCanonItMovedFrom`, including the backup manifest |
| The diagnosis exit-status contract (0 healthy / 1 problem / 2 the check did not run) was designed but not implemented | live exit-code check; `doctorFailure` in `cmd/gr/harness.go` |

Hardening taken from the unverified pool while the files were open: a
repository-scope registration path that is a symbolic link, or resolves outside
the repository, is refused (`TestRegistrationRefusesAPathThatResolvesOutsideTheRepository`);
an observability endpoint carrying URL userinfo is redacted before it is
reported (`TestObservabilityEndpointNeverEchoesEmbeddedCredentials`); a tracked
settings path no longer gets a useless ignore entry or advice naming a flag that
cannot help; an update whose verification fails still prints the report naming
the backup; the usage lines list every accepted command; the dead `fixIgnore`
parameter is gone.

Artifact corrections from the same review: the ambient-connect delta's
approval-question paragraph now records the interactive observation instead of
mandating the record stay "unverified", and the disclosure taxonomy names four
evidence states with a scenario for the observed-absent one
(`TestConnectionNoticeMatchesWhatWasActuallyObserved`,
`TestTheDocumentedAbsentDisclosureStatesItsOwnLimit`); the harness-doctor delta
allows a next action to name the user's own act where the tool deliberately has
no command; design decisions 6, 8, and 12 were reconciled with the code; tasks
6.13 and 8.3 were corrected to say what actually happened; and intent version 4
rewords OUT-11/SIG-13, whose "in either scope" overstated the shipped behaviour
against NG-6.
