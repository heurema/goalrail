# Intent Canary v0 Fixture Matrix

These synthetic fixtures verify the frozen v1 semantics without starting the
real 15-change canary. They use deterministic IDs and timestamps, write only to
test temporary directories, and consume no canary ordinal.

| Required outcome | Deterministic fixture | Required assertion |
|---|---|---|
| `wrong-but-green` | `internal/domain.TestCalculateCanaryReportCountsWrongButGreenIndependently` | A green `partial` or `miss` increments both non-match and wrong-but-green while preserving both facts. |
| Prevented material misunderstanding | `internal/operator.TestSyntheticFlowLifecycleKeepsOwnerAssessmentExplicitAndAppendOnly` | A pre-delivery material-correction event remains append-only and the later owner assessment projects `material_correction_before_delivery=true`. |
| Wording-only correction | `internal/operator.TestWordingOnlyFixtureDoesNotBecomeMaterialPrevention` | With no material-correction event, the owner assessment remains `material_correction_before_delivery=false` and the material count stays zero. |
| Process-caused abandonment | `internal/operator.TestOperatorRejectsRealStartAndPostTerminalMaterialCorrection` and `internal/domain.TestCalculateCanaryReportAppliesEveryHardStop/three_process-caused_flow_abandonments` | One flow abandonment retains its categorical process flag; three such abandonments trigger the exact hard-stop threshold. |
| Wrong join | `internal/domain.TestCalculateCanaryReportAppliesEveryHardStop/wrong_join` | One wrong join sets `WrongJoin` and produces `STOP`. |
| Unresolved links | `internal/domain.TestCalculateCanaryReportAppliesEveryHardStop/two_unresolved_links` | Two links unresolved after their bounded attempt set `UnresolvedLinks` and produce `STOP`. |
| Evidence rewrite | `internal/integration.TestEvidenceRewriteFixtureBecomesStopVerdict` | Rewriting an already appended event breaks digest verification; one integrity violation produces `STOP`. |
| `PASS` | `internal/domain.TestCalculateCanaryReportPassesCompleteEvidence` | Fifteen terminal assignments satisfying every pass signal produce `PASS`. |
| `STOP` | `internal/domain.TestCalculateCanaryReportAppliesEveryHardStop` | Each frozen hard-stop signal is evaluated before completion and produces `STOP`. |
| `RESHAPE` | `internal/domain.TestCalculateCanaryReportReshapesCompletedNonPass` | Complete evidence with no prevention, no lower non-match rate, and positive flow cost produces `RESHAPE`. |

The fixture matrix is evidence for task 4.3 only. Repository evidence privacy is
audited separately in task 4.4, and the synthetic end-to-end operator run remains
task 5.1.
