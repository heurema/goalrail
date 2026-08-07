package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/doctor"
	"github.com/heurema/goalrail/internal/domain"
)

func TestSuccessReceiptRetainsExactAuthorityAndSeparatesDoctorLayers(t *testing.T) {
	fixture := newVerifyPlanFixture(t)
	verified, err := VerifyPlan(context.Background(), fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	plan, authorization, err := receiptAuthority(verified)
	if err != nil {
		t.Fatal(err)
	}
	applied := make([]string, 0, len(plan.Mutations))
	for _, mutation := range plan.Mutations {
		applied = append(applied, mutation.ID)
	}
	diagnosis := doctor.Diagnosis{
		Category: doctor.CategoryAdvisory, Managed: true, ProjectID: plan.ProjectID,
		LocallyReady: true, LineageReady: true,
		SharedAdmission: doctor.ActivationEvidence{State: doctor.ActivationUnknown},
	}
	emission, err := buildApplyEmission(plan, authorization, ApplyResult{
		Applied: true, AppliedMutationIDs: applied, RecoveryDigest: authorization.PlanDigest,
		RecoveryPath: "/private/setup-recovery.json",
	}, &diagnosis, "task:feature-123")
	if err != nil {
		t.Fatal(err)
	}

	if emission.Schema != ReceiptEmissionSchemaV1 || emission.Receipt.Status != domain.SetupReceiptSuccess {
		t.Fatalf("success emission = %#v", emission)
	}
	if emission.Receipt.Authorization == nil || *emission.Receipt.Authorization != authorization {
		t.Fatal("receipt did not retain the exact plan authorization")
	}
	if emission.Receipt.PlanDigest != verified.Report().PlanDigest || emission.Receipt.ContinuationRef != "task:feature-123" {
		t.Fatalf("receipt lineage = %#v", emission.Receipt)
	}
	if emission.Diagnosis == nil || !emission.Diagnosis.LocallyReady || emission.Diagnosis.SharedAdmissionActive || emission.Diagnosis.SharedAdmissionState != string(doctor.ActivationUnknown) {
		t.Fatalf("doctor layers = %#v", emission.Diagnosis)
	}
	if !strings.HasPrefix(emission.Receipt.DiagnosisRef, "doctor:sha256:") || !strings.HasPrefix(emission.Receipt.RollbackRef, "setup-recovery:sha256:") {
		t.Fatalf("receipt references = %#v", emission.Receipt)
	}
	if len(emission.Receipt.BlockerRefs) != 0 {
		t.Fatalf("successful local setup retained blockers: %v", emission.Receipt.BlockerRefs)
	}
	for _, component := range emission.Receipt.Components {
		if component.Status != domain.SetupActionApplied {
			t.Fatalf("component %s status = %s", component.ID, component.Status)
		}
	}
	for _, mutation := range emission.Receipt.Mutations {
		if mutation.Status != domain.SetupActionApplied || mutation.AfterDigest == nil {
			t.Fatalf("mutation result = %#v", mutation)
		}
	}
	assertEmissionContainsNoRequestPayload(t, emission)
}

func TestPartialFailureReceiptListsAppliedFailedAndPendingMutations(t *testing.T) {
	fixture := newVerifyPlanFixture(t)
	verified, err := VerifyPlan(context.Background(), fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	plan, authorization, err := receiptAuthority(verified)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Mutations) < 3 {
		t.Fatalf("fixture has only %d mutations", len(plan.Mutations))
	}
	result := ApplyResult{
		AppliedMutationIDs: []string{plan.Mutations[0].ID},
		FailedMutationID:   plan.Mutations[1].ID,
		PendingMutationIDs: []string{plan.Mutations[1].ID, plan.Mutations[2].ID},
		FailureStage:       "apply-mutation",
		RecoveryDigest:     authorization.PlanDigest,
		RecoveryPath:       "/private/setup-recovery.json",
	}
	emission, err := buildApplyEmission(plan, authorization, result, nil, "task:feature-456")
	if err != nil {
		t.Fatal(err)
	}
	if emission.Receipt.Status != domain.SetupReceiptFailure || emission.Diagnosis != nil {
		t.Fatalf("failure emission = %#v", emission)
	}
	statuses := map[string]domain.SetupActionStatus{}
	for _, mutation := range emission.Receipt.Mutations {
		statuses[mutation.ID] = mutation.Status
	}
	if statuses[plan.Mutations[0].ID] != domain.SetupActionApplied ||
		statuses[plan.Mutations[1].ID] != domain.SetupActionFailed ||
		statuses[plan.Mutations[2].ID] != domain.SetupActionSkipped {
		t.Fatalf("mutation statuses = %v", statuses)
	}
	for _, ref := range []string{"setup-stage:apply-mutation", "setup-mutation:" + plan.Mutations[1].ID} {
		if !contains(emission.Receipt.BlockerRefs, ref) {
			t.Fatalf("failure receipt lacks %s: %v", ref, emission.Receipt.BlockerRefs)
		}
	}
	assertEmissionContainsNoRequestPayload(t, emission)
}

func TestNoWriteReceiptsAreCanonicalAndPerformZeroMutations(t *testing.T) {
	for _, test := range []struct {
		name       string
		reason     NoWriteReason
		wantStatus domain.SetupReceiptStatus
		wantRef    string
	}{
		{name: "refused", reason: NoWriteRefused, wantStatus: domain.SetupReceiptRefused, wantRef: "setup-authorization:refused"},
		{name: "missing authorization", reason: NoWriteMissingAuthorization, wantStatus: domain.SetupReceiptNoWrite, wantRef: "setup-authorization:missing"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newVerifyPlanFixture(t)
			beforeRepository := snapshotTree(t, fixture.repository)
			beforeHome := snapshotTree(t, fixture.home)
			emission, err := EmitNoWriteReceipt(NoWriteReceiptOptions{
				PlanPath: fixture.options.PlanPath, ContinuationRef: "task:feature-789", Reason: test.reason,
			})
			if err != nil {
				t.Fatal(err)
			}
			assertTreesUnchanged(t, fixture.repository, fixture.home, beforeRepository, beforeHome)
			if emission.Receipt.Status != test.wantStatus || emission.Receipt.Authorization != nil || emission.RecoveryPath != "" {
				t.Fatalf("no-write emission = %#v", emission)
			}
			if !contains(emission.Receipt.BlockerRefs, test.wantRef) || emission.Receipt.DiagnosisRef != "doctor:not-run-no-write" {
				t.Fatalf("no-write references = %#v", emission.Receipt)
			}
			for _, component := range emission.Receipt.Components {
				if component.Status != domain.SetupActionSkipped {
					t.Fatalf("no-write component status = %s", component.Status)
				}
			}
			for _, mutation := range emission.Receipt.Mutations {
				if mutation.Status != domain.SetupActionSkipped || mutation.BeforeDigest != nil || mutation.AfterDigest != nil {
					t.Fatalf("no-write mutation = %#v", mutation)
				}
			}
			assertEmissionContainsNoRequestPayload(t, emission)
		})
	}
}

func TestVerificationFailureEmitsAuthorizedNoMutationReceipt(t *testing.T) {
	fixture := newVerifyPlanFixture(t)
	writeFixtureFile(t, filepath.Join(fixture.bundleRoot, "bin", "gr"), []byte("tampered\n"), 0o755)
	beforeRepository := snapshotTree(t, fixture.repository)
	beforeHome := snapshotTree(t, fixture.home)

	emission, err := ExecuteApply(context.Background(), fixture.options, "task:feature-verify-failure")
	if !errors.Is(err, ErrBundleInvalid) {
		t.Fatalf("ExecuteApply() error = %v, want %v", err, ErrBundleInvalid)
	}
	assertTreesUnchanged(t, fixture.repository, fixture.home, beforeRepository, beforeHome)
	if emission.Receipt.Status != domain.SetupReceiptFailure || emission.Receipt.Authorization == nil {
		t.Fatalf("verification failure emission = %#v", emission)
	}
	if !contains(emission.Receipt.BlockerRefs, "setup-stage:verify-plan") || emission.Receipt.RollbackRef != "setup-rollback:none-required" {
		t.Fatalf("verification failure references = %#v", emission.Receipt)
	}
	for _, mutation := range emission.Receipt.Mutations {
		if mutation.Status != domain.SetupActionSkipped {
			t.Fatalf("verification failure mutation = %#v", mutation)
		}
	}
}

func TestReceiptEmissionRejectsRawContinuationContent(t *testing.T) {
	fixture := newVerifyPlanFixture(t)
	beforeRepository := snapshotTree(t, fixture.repository)
	beforeHome := snapshotTree(t, fixture.home)
	_, err := EmitNoWriteReceipt(NoWriteReceiptOptions{
		PlanPath:        fixture.options.PlanPath,
		ContinuationRef: "please implement the raw feature prompt",
		Reason:          NoWriteRefused,
	})
	if !errors.Is(err, ErrContinuationRequired) {
		t.Fatalf("raw continuation error = %v", err)
	}
	assertTreesUnchanged(t, fixture.repository, fixture.home, beforeRepository, beforeHome)
}

func assertEmissionContainsNoRequestPayload(t *testing.T, emission ReceiptEmission) {
	t.Helper()
	raw, err := json.Marshal(emission)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range [][]byte{[]byte(`"prompt"`), []byte(`"request"`), []byte(`"conversation"`)} {
		if bytes.Contains(raw, forbidden) {
			t.Fatalf("receipt emission retained prohibited field %q: %s", forbidden, raw)
		}
	}
	artifact, err := domain.FreezeSetupReceipt(emission.Receipt)
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Digest() != emission.ReceiptDigest {
		t.Fatalf("receipt digest = %s, want %s", emission.ReceiptDigest, artifact.Digest())
	}
}

func TestNoWriteReceiptDoesNotDependOnProcessHome(t *testing.T) {
	fixture := newVerifyPlanFixture(t)
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() { _ = os.Setenv("HOME", oldHome) })
	if err := os.Setenv("HOME", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	first, err := EmitNoWriteReceipt(NoWriteReceiptOptions{
		PlanPath: fixture.options.PlanPath, ContinuationRef: "task:stable-no-write", Reason: NoWriteRefused,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("HOME", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	second, err := EmitNoWriteReceipt(NoWriteReceiptOptions{
		PlanPath: fixture.options.PlanPath, ContinuationRef: "task:stable-no-write", Reason: NoWriteRefused,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("no-write receipt changed with process HOME\nfirst:  %#v\nsecond: %#v", first, second)
	}
}

// A digest names its own algorithm. SHA256Digest is a complete prefixed
// reference by the domain's own rule, so a scheme that adds the algorithm again
// produces setup-recovery:sha256:sha256:… — a reference that passes the shape
// check while saying the same thing twice.
func TestReceiptReferencesNameTheirDigestOnce(t *testing.T) {
	digest := domain.DigestCanonicalJSON([]byte("{}"))
	if !strings.HasPrefix(string(digest), "sha256:") {
		t.Fatalf("the domain digest no longer carries its algorithm: %q", digest)
	}

	reference := diagnosisReference(SetupDiagnosisSummary{Schema: "goalrail.setup-diagnosis-summary/v1"})
	if strings.Contains(reference, "sha256:sha256:") {
		t.Fatalf("the diagnosis reference names its algorithm twice: %q", reference)
	}
	if !domain.IsEvidenceReference(reference) {
		t.Fatalf("the diagnosis reference is not a bounded evidence reference: %q", reference)
	}
	if !strings.HasPrefix(reference, "doctor:sha256:") {
		t.Fatalf("the diagnosis reference lost its scheme or its digest: %q", reference)
	}
}
