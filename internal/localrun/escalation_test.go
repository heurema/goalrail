package localrun

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/heurema/goalrail/internal/domain"
)

const validEscalationPayload = `---
schema: goalrail.escalation/v0
block_class: conflicting-requirements
question: Which retention period governs when the two documents disagree?
---
`

func TestPrepareRejectsPreexistingEscalationArtifact(t *testing.T) {
	adapter := &countingFixtureAdapter{result: completedObservation()}
	service, spec, store, _ := fixtureService(t, adapter, []WorktreeObservation{
		observationWith("base", escalationWorktreeEntry(validEscalationPayload)),
	})
	writeWorktreeEscalation(t, spec.Repository.Root, validEscalationPayload)

	_, err := service.Prepare(context.Background(), specReader(t, spec))
	if !errors.Is(err, ErrEscalationArtifactPresent) {
		t.Fatalf("expected the preparation gate to reject a pre-populated reserved path, got %v", err)
	}
	// A stale question must not attach itself to a new run, so no prepared state
	// may exist afterwards.
	frozen, freezeErr := domain.FreezeWorkSpec(spec)
	if freezeErr != nil {
		t.Fatal(freezeErr)
	}
	exists, existsErr := store.Exists(preparedPath(frozen.Digest(), "preparation.json"))
	if existsErr != nil {
		t.Fatal(existsErr)
	}
	if exists {
		t.Fatal("rejected preparation persisted prepared state")
	}
	if adapter.calls.Load() != 0 {
		t.Fatal("rejected preparation reached the provider")
	}
}

func TestPrepareAllowsUnrelatedContentBesideTheReservedPath(t *testing.T) {
	// The gate applies to the exact reserved file. The .goalrail/ namespace
	// already holds unrelated local content, and rejecting the directory would
	// make preparation unusable in any repository that uses it.
	adapter := &countingFixtureAdapter{result: completedObservation()}
	service, spec, _, _ := fixtureService(t, adapter, []WorktreeObservation{
		observationWith("base", WorktreeEntry{
			Path:   ".goalrail/activate-dogfood-run-v1/contract.go",
			Status: "!!",
			Digest: digestOf("capsule"),
		}),
		observationWith("terminal"),
	})

	if _, err := service.Prepare(context.Background(), specReader(t, spec)); err != nil {
		t.Fatalf("unrelated content in the reserved directory blocked preparation: %v", err)
	}
}

func TestStartRejectsAnArtifactCreatedAfterPreparation(t *testing.T) {
	// Preparation gates the frozen baseline, but a prepared run is reused rather
	// than re-observed and can be started much later. Without a pre-launch gate,
	// an artifact created in that window would be attributed to a provider that
	// had not run yet.
	adapter := &countingFixtureAdapter{result: completedObservation()}
	service, spec, _, _ := fixtureService(t, adapter, []WorktreeObservation{
		observationWith("base"),
		observationWith("terminal", escalationWorktreeEntry(validEscalationPayload)),
	})
	prepared := prepareFixture(t, service, spec)
	writeWorktreeEscalation(t, spec.Repository.Root, validEscalationPayload)
	service.newRunID = func() (domain.RunID, error) { return "run-stale-artifact", nil }

	_, err := service.Start(context.Background(), StartInput{
		WorkSpecDigest: prepared.WorkSpec.Digest(),
		Adapter:        "fixture",
	})
	if !errors.Is(err, ErrEscalationArtifactPresent) {
		t.Fatalf("expected the pre-launch gate to reject a stale artifact, got %v", err)
	}
	if adapter.calls.Load() != 0 {
		t.Fatal("a stale artifact reached the provider")
	}
}

func TestPrepareRejectsASymlinkedReservedDirectory(t *testing.T) {
	// Git reports only the `.goalrail` entry for a symlinked directory, so an
	// exact-path check alone would accept preparation. The provider would then
	// write the question outside the repository, where observation never sees
	// it, and the channel would be silently dead.
	adapter := &countingFixtureAdapter{result: completedObservation()}
	service, spec, _, _ := fixtureService(t, adapter, []WorktreeObservation{
		observationWith("base"),
		observationWith("terminal"),
	})
	external := t.TempDir()
	link := filepath.Join(spec.Repository.Root, ".goalrail")
	if err := os.Symlink(external, link); err != nil {
		t.Fatal(err)
	}

	if _, err := service.Prepare(context.Background(), specReader(t, spec)); err == nil {
		t.Fatal("preparation accepted a reserved directory pointing outside the repository")
	}
}

func TestRetentionMismatchIsRecordedRatherThanBound(t *testing.T) {
	// The observation hashes the artifact and retention reads it again. If the
	// file changed between the two, binding the receipt to the second read would
	// silently break the single-snapshot evidence chain.
	receipt := runEscalationLifecycle(t, escalationLifecycle{
		payload:      validEscalationPayload,
		terminal:     observationWith("terminal", escalationWorktreeEntry("a different question\n")),
		expectStatus: StateFailed,
	})

	if receipt.Status != StateFailed {
		t.Fatalf("status = %q, want failed", receipt.Status)
	}
	if receipt.Escalation == nil || receipt.Escalation.Valid {
		t.Fatalf("escalation = %+v, want an invalid record", receipt.Escalation)
	}
	if receipt.Escalation.Reason != "ESCALATION_ARTIFACT_CHANGED" {
		t.Fatalf("reason = %q, want ESCALATION_ARTIFACT_CHANGED", receipt.Escalation.Reason)
	}
	if receipt.Escalation.RetainedRef != "" {
		t.Fatal("a mismatched artifact was retained as if it were the observed one")
	}
}

func TestFinishFailsClosedWhenAnObservedQuestionWasNotRetained(t *testing.T) {
	// Reachable when retention fails after the observation exists. Treating the
	// missing record as "no question" would let the run finish as passed.
	adapter := &escalatingFixtureAdapter{result: completedObservation()}
	service, spec, store, _ := fixtureService(t, adapter, []WorktreeObservation{
		observationWith("base"),
		observationWith("terminal", escalationWorktreeEntry(validEscalationPayload)),
	})
	adapter.onLaunch = writeEscalationDuringLaunch(t, spec.Repository.Root, validEscalationPayload)
	prepared := prepareFixture(t, service, spec)
	service.newRunID = func() (domain.RunID, error) { return "run-unretained", nil }
	if _, err := service.Start(context.Background(), StartInput{
		WorkSpecDigest: prepared.WorkSpec.Digest(),
		Adapter:        "fixture",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(
		store.Root(),
		filepath.FromSlash(runPath("run-unretained", "escalation.json")),
	)); err != nil {
		t.Fatal(err)
	}

	_, err := service.Finish(context.Background(), FinishInput{
		RunID: "run-unretained",
		Results: []CheckResult{{
			ID:             "test",
			State:          domain.CheckResultPass,
			EvidenceRef:    "local:test-log",
			EvidenceDigest: "sha256:" + strings.Repeat("d", 64),
		}},
	})
	if err == nil {
		t.Fatal("finish accepted a run whose observed question was never retained")
	}
}

func TestValidEscalationWithCleanScopeYieldsBlockedReceipt(t *testing.T) {
	receipt := runEscalationLifecycle(t, escalationLifecycle{
		payload:  validEscalationPayload,
		terminal: observationWith("terminal", escalationWorktreeEntry(validEscalationPayload)),
	})

	if receipt.Status != StateBlocked {
		t.Fatalf("status = %q, want blocked", receipt.Status)
	}
	if receipt.Schema != TerminalReceiptSchemaV1 {
		t.Fatalf("schema = %q, want %q", receipt.Schema, TerminalReceiptSchemaV1)
	}
	if receipt.Intent == nil || receipt.Intent.ID != "intent-dogfood" || receipt.Intent.Version != 3 {
		t.Fatalf("receipt intent reference = %+v", receipt.Intent)
	}
	if receipt.Escalation == nil || !receipt.Escalation.Valid {
		t.Fatalf("escalation = %+v", receipt.Escalation)
	}
	if receipt.Escalation.Path != ReservedEscalationPath {
		t.Fatalf("escalation path = %q", receipt.Escalation.Path)
	}
	if !strings.HasPrefix(receipt.Escalation.Digest, "sha256:") ||
		receipt.Escalation.RetainedRef == "" {
		t.Fatalf("escalation lacks retained evidence: %+v", receipt.Escalation)
	}
	if !hasReason(receipt.Reasons, "ESCALATION_PENDING") {
		t.Fatalf("reasons = %v, want ESCALATION_PENDING", receipt.Reasons)
	}
	// The artifact stays visible in the delta: excluding it would make the
	// reserved path the only unauditable write channel in the contract.
	if !containsPath(receipt.WorktreeDelta.ChangedPaths, ReservedEscalationPath) {
		t.Fatalf("changed paths = %v, want the reserved path", receipt.WorktreeDelta.ChangedPaths)
	}
	if len(receipt.WorktreeDelta.ScopeViolations) != 0 {
		t.Fatalf("scope violations = %v, want none", receipt.WorktreeDelta.ScopeViolations)
	}
	// The receipt references the retained bytes; it never carries the payload.
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("Which retention period governs")) {
		t.Fatal("the receipt embedded the escalation payload instead of referencing it")
	}
}

func TestEscalationWithInScopeEditsYieldsFailed(t *testing.T) {
	receipt := runEscalationLifecycle(t, escalationLifecycle{
		payload: validEscalationPayload,
		terminal: observationWith(
			"terminal",
			escalationWorktreeEntry(validEscalationPayload),
			WorktreeEntry{Path: "inside/patch.go", Status: " M", Digest: digestOf("patch")},
		),
	})

	if receipt.Status != StateFailed {
		t.Fatalf("status = %q, want failed: a hedge must not be able to claim blocked", receipt.Status)
	}
	if !hasReason(receipt.Reasons, "BLOCKED_WITH_EDITS") {
		t.Fatalf("reasons = %v, want BLOCKED_WITH_EDITS", receipt.Reasons)
	}
	if receipt.Escalation == nil || !receipt.Escalation.Valid {
		t.Fatal("the artifact must still be retained on a failed hedge")
	}
}

func TestReservedPathIsNotAnInScopeEditUnderABroadScope(t *testing.T) {
	spec := func(base domain.WorkSpec) domain.WorkSpec {
		base.Paths = []string{"."}
		return base
	}
	receipt := runEscalationLifecycle(t, escalationLifecycle{
		payload:      validEscalationPayload,
		terminal:     observationWith("terminal", escalationWorktreeEntry(validEscalationPayload)),
		mutateSpec:   spec,
		expectStatus: StateBlocked,
	})

	// With a scope of ".", pathInScope matches everything, so without the
	// explicit rule the reserved path would count as an in-scope edit and every
	// escalation would fail.
	if receipt.Status != StateBlocked {
		t.Fatalf("status = %q, want blocked under a broad scope", receipt.Status)
	}
}

func TestInvalidEscalationPayloadYieldsFailedAndNeverBlocked(t *testing.T) {
	for name, payload := range map[string]string{
		"empty": "   \n",
		// Secret-shaped detection reuses the retained-text patterns rather than
		// introducing a second family, so it recognises exactly what those
		// patterns recognise.
		"secret shaped": "The blocked step needs api_key: 8f3ca11b9d0e4c72 to proceed.\n",
		"oversized":     strings.Repeat("x", domain.MaxEscalationBytes+1),
		"control byte":  "question\x00terminator\n",
	} {
		t.Run(name, func(t *testing.T) {
			receipt := runEscalationLifecycle(t, escalationLifecycle{
				payload:      payload,
				terminal:     observationWith("terminal", escalationWorktreeEntry(payload)),
				expectStatus: StateFailed,
			})
			if receipt.Status != StateFailed {
				t.Fatalf("status = %q, want failed", receipt.Status)
			}
			if receipt.Status == StateBlocked || receipt.Status == StatePassed {
				t.Fatalf("invalid payload produced %q", receipt.Status)
			}
			if receipt.Escalation == nil || receipt.Escalation.Valid {
				t.Fatalf("escalation = %+v, want an invalid retained record", receipt.Escalation)
			}
		})
	}
}

func TestUnattributableOutcomesTakePrecedenceOverBlocked(t *testing.T) {
	for name, testCase := range map[string]struct {
		observation ProviderObservation
		wantStatus  RunState
	}{
		"unlinked": {
			observation: ProviderObservation{
				Outcome:        ProviderCompleted,
				IdentityStatus: IdentityUnlinked,
				Reason:         "MISSING_LAUNCH_RECEIPT",
			},
			wantStatus: StateUnlinked,
		},
		"denied": {
			observation: ProviderObservation{
				Outcome:        ProviderDenied,
				IdentityStatus: IdentityVerified,
				RootSessionRef: "session-root",
			},
			wantStatus: StateFailed,
		},
		"launch failed": {
			observation: ProviderObservation{
				Outcome:        ProviderLaunchFailed,
				IdentityStatus: IdentityVerified,
				RootSessionRef: "session-root",
			},
			wantStatus: StateLaunchFailed,
		},
	} {
		t.Run(name, func(t *testing.T) {
			adapter := &escalatingFixtureAdapter{result: testCase.observation}
			service, spec, _, _ := fixtureService(t, adapter, []WorktreeObservation{
				observationWith("base"),
				observationWith("terminal", escalationWorktreeEntry(validEscalationPayload)),
			})
			adapter.onLaunch = writeEscalationDuringLaunch(t, spec.Repository.Root, validEscalationPayload)
			prepared := prepareFixture(t, service, spec)
			service.newRunID = func() (domain.RunID, error) { return "run-precedence", nil }

			started, err := service.Start(context.Background(), StartInput{
				WorkSpecDigest: prepared.WorkSpec.Digest(),
				Adapter:        "fixture",
			})
			if err != nil {
				t.Fatal(err)
			}
			if started.Receipt == nil {
				t.Fatal("an unattributable outcome must produce an eager terminal receipt")
			}
			if started.Receipt.Status != testCase.wantStatus {
				t.Fatalf("status = %q, want %q", started.Receipt.Status, testCase.wantStatus)
			}
			if started.Receipt.Status == StateBlocked {
				t.Fatal("an unattributable session minted an authoritative question")
			}
			// The question is still evidence even when it cannot be authoritative.
			if started.Receipt.Escalation == nil ||
				!strings.HasPrefix(started.Receipt.Escalation.Digest, "sha256:") {
				t.Fatalf("escalation evidence = %+v", started.Receipt.Escalation)
			}
			if !hasReason(started.Receipt.Reasons, "ESCALATION_RECORDED") {
				t.Fatalf("reasons = %v, want ESCALATION_RECORDED", started.Receipt.Reasons)
			}
		})
	}
}

func TestFailingCheckKeepsFailedWhenAnEscalationExists(t *testing.T) {
	receipt := runEscalationLifecycle(t, escalationLifecycle{
		payload:  validEscalationPayload,
		terminal: observationWith("terminal", escalationWorktreeEntry(validEscalationPayload)),
		results: []CheckResult{{
			ID:             "test",
			State:          domain.CheckResultFail,
			EvidenceRef:    "local:test-log",
			EvidenceDigest: "sha256:" + strings.Repeat("d", 64),
		}},
		expectStatus: StateFailed,
	})

	// Blocked must not become a channel for converting red checks into a softer
	// outcome, and the failing result must survive verbatim.
	if receipt.Status != StateFailed {
		t.Fatalf("status = %q, want failed", receipt.Status)
	}
	if len(receipt.CheckResults) != 1 || receipt.CheckResults[0].State != domain.CheckResultFail {
		t.Fatalf("check results = %+v, want the failing result verbatim", receipt.CheckResults)
	}
	if !hasReason(receipt.Reasons, "CHECK_FAILED") {
		t.Fatalf("reasons = %v, want CHECK_FAILED preserved", receipt.Reasons)
	}
}

func TestPassingChecksNeverYieldPassedWhileAnEscalationIsRetained(t *testing.T) {
	receipt := runEscalationLifecycle(t, escalationLifecycle{
		payload:  validEscalationPayload,
		terminal: observationWith("terminal", escalationWorktreeEntry(validEscalationPayload)),
		results: []CheckResult{{
			ID:             "test",
			State:          domain.CheckResultPass,
			EvidenceRef:    "local:test-log",
			EvidenceDigest: "sha256:" + strings.Repeat("d", 64),
		}},
	})

	if receipt.Status == StatePassed {
		t.Fatal("a retained escalation must make passed unreachable")
	}
	if receipt.Status != StateBlocked {
		t.Fatalf("status = %q, want blocked", receipt.Status)
	}
	if receipt.CheckResults[0].State != domain.CheckResultPass {
		t.Fatalf("check result = %+v, want the passing result verbatim", receipt.CheckResults[0])
	}
}

func TestRetainedEscalationSurvivesWorktreeMutation(t *testing.T) {
	adapter := &escalatingFixtureAdapter{result: completedObservation()}
	service, spec, store, _ := fixtureService(t, adapter, []WorktreeObservation{
		observationWith("base"),
		observationWith("terminal", escalationWorktreeEntry(validEscalationPayload)),
	})
	adapter.onLaunch = writeEscalationDuringLaunch(t, spec.Repository.Root, validEscalationPayload)
	prepared := prepareFixture(t, service, spec)
	service.newRunID = func() (domain.RunID, error) { return "run-retained", nil }

	if _, err := service.Start(context.Background(), StartInput{
		WorkSpecDigest: prepared.WorkSpec.Digest(),
		Adapter:        "fixture",
	}); err != nil {
		t.Fatal(err)
	}

	// `rm .goalrail/blocked.md && gr finish` must not convert the run.
	artifact := filepath.Join(spec.Repository.Root, filepath.FromSlash(ReservedEscalationPath))
	if err := os.Remove(artifact); err != nil {
		t.Fatal(err)
	}

	receipt, err := service.Finish(context.Background(), FinishInput{RunID: "run-retained"})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != StateBlocked {
		t.Fatalf("status = %q, want blocked after the worktree file was deleted", receipt.Status)
	}
	retained, err := store.ReadBytes(receipt.Escalation.RetainedRef)
	if err != nil {
		t.Fatal(err)
	}
	if string(retained) != validEscalationPayload {
		t.Fatal("retained bytes changed with the worktree file")
	}
}

func TestFinishUsesTheObservationTakenAtStart(t *testing.T) {
	adapter := &escalatingFixtureAdapter{result: completedObservation()}
	service, spec, _, _ := fixtureService(t, adapter, []WorktreeObservation{
		observationWith("base"),
		observationWith("terminal", escalationWorktreeEntry(validEscalationPayload)),
		// A third observation would only be reached if Finish observed again.
		observationWith("late"),
	})
	adapter.onLaunch = writeEscalationDuringLaunch(t, spec.Repository.Root, validEscalationPayload)
	prepared := prepareFixture(t, service, spec)
	service.newRunID = func() (domain.RunID, error) { return "run-single-observation", nil }

	if _, err := service.Start(context.Background(), StartInput{
		WorkSpecDigest: prepared.WorkSpec.Digest(),
		Adapter:        "fixture",
	}); err != nil {
		t.Fatal(err)
	}
	observer := service.observer.(*fixtureObserver)
	if calls := observer.calls.Load(); calls != 2 {
		t.Fatalf("observations after start = %d, want 2 (prepare and start)", calls)
	}

	receipt, err := service.Finish(context.Background(), FinishInput{RunID: "run-single-observation"})
	if err != nil {
		t.Fatal(err)
	}
	if calls := observer.calls.Load(); calls != 2 {
		t.Fatalf("observations after finish = %d, want 2: finish must reuse the start observation", calls)
	}
	if receipt.Status != StateBlocked {
		t.Fatalf("status = %q, want blocked", receipt.Status)
	}
}

func TestFinishFallsBackToObservingWhenNoStartObservationExists(t *testing.T) {
	adapter := &countingFixtureAdapter{result: completedObservation()}
	service, spec, store, _ := fixtureService(t, adapter, []WorktreeObservation{
		observationWith("base"),
		observationWith("terminal"),
		observationWith("late"),
	})
	prepared := prepareFixture(t, service, spec)
	service.newRunID = func() (domain.RunID, error) { return "run-legacy", nil }
	if _, err := service.Start(context.Background(), StartInput{
		WorkSpecDigest: prepared.WorkSpec.Digest(),
		Adapter:        "fixture",
	}); err != nil {
		t.Fatal(err)
	}
	// Simulate a run prepared before the start-time observation existed.
	if err := os.Remove(filepath.Join(
		store.Root(),
		filepath.FromSlash(runPath("run-legacy", "terminal-observation.json")),
	)); err != nil {
		t.Fatal(err)
	}

	observer := service.observer.(*fixtureObserver)
	before := observer.calls.Load()
	if _, err := service.Finish(context.Background(), FinishInput{RunID: "run-legacy"}); err != nil {
		t.Fatal(err)
	}
	if observer.calls.Load() != before+1 {
		t.Fatal("finish did not fall back to observing when no start observation existed")
	}
}

func TestReservedPathIsNeverAScopeViolation(t *testing.T) {
	baseline := observationWith("base")
	terminal := observationWith("terminal", escalationWorktreeEntry(validEscalationPayload))
	delta := CompareWorktrees(baseline, terminal, []string{"inside"})

	if len(delta.ScopeViolations) != 0 {
		t.Fatalf("scope violations = %v, want none", delta.ScopeViolations)
	}
	if !containsPath(delta.ChangedPaths, ReservedEscalationPath) {
		t.Fatalf("changed paths = %v, want the reserved path retained", delta.ChangedPaths)
	}
	if hasInScopeEdits(delta, []string{"inside"}) {
		t.Fatal("the reserved path counted as an in-scope edit")
	}
	if hasInScopeEdits(delta, []string{"."}) {
		t.Fatal("the reserved path counted as an in-scope edit under a broad scope")
	}
}

func TestReceiptSchemaRulesAcceptLegacyAndRejectUnknown(t *testing.T) {
	base := validFixtureReceipt()

	legacy := base
	legacy.Schema = ""
	if err := validateTerminalReceipt(legacy); err != nil {
		t.Fatalf("a receipt written before the schema identifier must stay readable: %v", err)
	}

	unknown := base
	unknown.Schema = "goalrail.terminal-receipt/v9"
	if err := validateTerminalReceipt(unknown); err == nil {
		t.Fatal("an unsupported receipt schema was accepted")
	}

	// A v1 receipt promises the run-to-intent chain. A missing or malformed
	// reference would make that chain vanish or point at the wrong decision.
	withoutIntent := base
	withoutIntent.Intent = nil
	if err := validateTerminalReceipt(withoutIntent); err == nil {
		t.Fatal("a v1 receipt was accepted without a frozen intent reference")
	}
	for name, reference := range map[string]ReceiptIntentReference{
		"non-canonical id": {ID: "Intent One", Version: 1, Digest: "sha256:" + strings.Repeat("b", 64)},
		"zero version":     {ID: "intent-dogfood", Version: 0, Digest: "sha256:" + strings.Repeat("b", 64)},
		"malformed digest": {ID: "intent-dogfood", Version: 1, Digest: "sha256:short"},
	} {
		malformed := base
		malformed.Intent = &reference
		if err := validateTerminalReceipt(malformed); err == nil {
			t.Fatalf("a v1 receipt was accepted with a %s intent reference", name)
		}
	}
	// The relaxed path stays available only to receipts predating the identifier.
	legacyWithoutIntent := base
	legacyWithoutIntent.Schema = ""
	legacyWithoutIntent.Intent = nil
	if err := validateTerminalReceipt(legacyWithoutIntent); err != nil {
		t.Fatalf("a legacy receipt was required to carry an intent reference: %v", err)
	}

	blockedWithoutEvidence := base
	blockedWithoutEvidence.Status = StateBlocked
	if err := validateTerminalReceipt(blockedWithoutEvidence); err == nil {
		t.Fatal("blocked was accepted without a retained escalation")
	}

	passedWithEscalation := base
	passedWithEscalation.Escalation = &EscalationRecord{
		Path:        ReservedEscalationPath,
		Digest:      "sha256:" + strings.Repeat("c", 64),
		RetainedRef: "runs/run-one/escalation/payload.md",
		Valid:       true,
	}
	if err := validateTerminalReceipt(passedWithEscalation); err == nil {
		t.Fatal("a passing receipt was accepted while an escalation was retained")
	}

	foreignPath := base
	foreignPath.Status = StateFailed
	foreignPath.Escalation = &EscalationRecord{Path: "docs/question.md", Valid: false}
	if err := validateTerminalReceipt(foreignPath); err == nil {
		t.Fatal("an escalation naming an unexpected path was accepted")
	}
}

type escalationLifecycle struct {
	payload      string
	terminal     WorktreeObservation
	results      []CheckResult
	mutateSpec   func(domain.WorkSpec) domain.WorkSpec
	expectStatus RunState
}

func runEscalationLifecycle(t *testing.T, lifecycle escalationLifecycle) TerminalReceipt {
	t.Helper()
	adapter := &escalatingFixtureAdapter{result: completedObservation()}
	service, spec, _, _ := fixtureService(t, adapter, []WorktreeObservation{
		observationWith("base"),
		lifecycle.terminal,
	})
	if lifecycle.mutateSpec != nil {
		spec = lifecycle.mutateSpec(spec)
	}
	adapter.onLaunch = writeEscalationDuringLaunch(t, spec.Repository.Root, lifecycle.payload)
	prepared := prepareFixture(t, service, spec)
	service.newRunID = func() (domain.RunID, error) { return "run-escalation", nil }

	if _, err := service.Start(context.Background(), StartInput{
		WorkSpecDigest: prepared.WorkSpec.Digest(),
		Adapter:        "fixture",
	}); err != nil {
		t.Fatal(err)
	}
	receipt, err := service.Finish(context.Background(), FinishInput{
		RunID:   "run-escalation",
		Results: lifecycle.results,
	})
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

// escalatingFixtureAdapter writes the reserved path while it "runs", which is
// when a provider would write it. Writing it before start is now rejected by
// the pre-launch gate, and rightly so: an artifact that predates the launch
// belongs to no provider.
type escalatingFixtureAdapter struct {
	result   ProviderObservation
	onLaunch func()
	calls    atomic.Int32
}

func (*escalatingFixtureAdapter) Name() string    { return "fixture" }
func (*escalatingFixtureAdapter) Version() string { return "v0" }

func (adapter *escalatingFixtureAdapter) Launch(
	_ context.Context,
	_ LaunchRequest,
) ProviderObservation {
	adapter.calls.Add(1)
	if adapter.onLaunch != nil {
		adapter.onLaunch()
	}
	return adapter.result
}

func specReader(t *testing.T, spec domain.WorkSpec) io.Reader {
	t.Helper()
	raw, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(raw)
}

// writeEscalationDuringLaunch returns a launch hook that writes the reserved
// path, matching when a real provider would create it. Writing it earlier now
// trips the pre-launch gate.
func writeEscalationDuringLaunch(t *testing.T, root, payload string) func() {
	t.Helper()
	return func() {
		writeWorktreeEscalation(t, root, payload)
	}
}

func writeWorktreeEscalation(t *testing.T, root, payload string) {
	t.Helper()
	artifact := filepath.Join(root, filepath.FromSlash(ReservedEscalationPath))
	if err := os.MkdirAll(filepath.Dir(artifact), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifact, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
}

// escalationWorktreeEntry mirrors what the real observer records: the digest of
// the artifact's actual bytes. Retention compares its own read against this
// digest, so a fixture that invented one would hide the mismatch it guards.
func escalationWorktreeEntry(payload string) WorktreeEntry {
	sum := sha256.Sum256([]byte(payload))
	return WorktreeEntry{
		Path:   ReservedEscalationPath,
		Status: "!!",
		Digest: "sha256:" + hex.EncodeToString(sum[:]),
	}
}

func observationWith(label string, entries ...WorktreeEntry) WorktreeObservation {
	observation := fixtureObservation(label)
	observation.Digest = digestOf(label)
	observation.Entries = append([]WorktreeEntry(nil), entries...)
	return observation
}

func digestOf(label string) string {
	fill := "a"
	if len(label) > 0 {
		fill = string(label[0])
	}
	return "sha256:" + strings.Repeat(fill, 64)
}

func containsPath(paths []string, wanted string) bool {
	for _, path := range paths {
		if path == wanted {
			return true
		}
	}
	return false
}

func hasReason(reasons []domain.EvidenceReasonCode, wanted domain.EvidenceReasonCode) bool {
	for _, reason := range reasons {
		if reason == wanted {
			return true
		}
	}
	return false
}

func validFixtureReceipt() TerminalReceipt {
	now := fixedClock()
	return TerminalReceipt{
		Schema:          TerminalReceiptSchemaV1,
		WorkSpecID:      "work-dogfood",
		WorkSpecVersion: 1,
		WorkSpecDigest:  domain.WorkSpecDigest("sha256:" + strings.Repeat("a", 64)),
		Intent: &ReceiptIntentReference{
			ID:      "intent-dogfood",
			Version: 3,
			Digest:  "sha256:" + strings.Repeat("b", 64),
		},
		RunID:              "run-one",
		Adapter:            "fixture",
		AdapterVersion:     "v0",
		BaseRevision:       strings.Repeat("a", 40),
		TerminalHead:       strings.Repeat("a", 40),
		EffectivePaths:     []string{"inside"},
		Status:             StatePassed,
		PreparedAt:         now(),
		LaunchAttemptedAt:  now(),
		ProviderObservedAt: now(),
		TerminalAt:         now(),
	}
}

func completedObservation() ProviderObservation {
	return ProviderObservation{
		Outcome:        ProviderCompleted,
		IdentityStatus: IdentityVerified,
		RootSessionRef: "session-root",
	}
}
