package setup

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/domain"
)

func TestCleanMachineSetupMatrix(t *testing.T) {
	t.Run("refusal performs no writes", func(t *testing.T) {
		fixture := newCleanMachineFixture(t)
		beforeRepository := snapshotTree(t, fixture.repository)
		beforeHome := snapshotTree(t, fixture.home)

		emission, err := EmitNoWriteReceipt(NoWriteReceiptOptions{
			PlanPath:        fixture.options.PlanPath,
			ContinuationRef: "task:clean-machine-refusal",
			Reason:          NoWriteRefused,
		})
		if err != nil {
			t.Fatal(err)
		}
		assertTreesUnchanged(t, fixture.repository, fixture.home, beforeRepository, beforeHome)
		if emission.Receipt.Status != domain.SetupReceiptRefused || emission.Receipt.Authorization != nil || emission.RecoveryPath != "" {
			t.Fatalf("refusal emission = %#v", emission)
		}
		if !contains(emission.Receipt.BlockerRefs, "setup-authorization:refused") {
			t.Fatalf("refusal blockers = %v", emission.Receipt.BlockerRefs)
		}
	})

	t.Run("successful installation is durable and repository read only", func(t *testing.T) {
		fixture := newCleanMachineFixture(t)
		verified := mustVerifyCleanMachinePlan(t, fixture)
		beforeRepository := snapshotTree(t, fixture.repository)

		result, err := Apply(context.Background(), verified)
		if err != nil {
			t.Fatal(err)
		}
		wantMutations := []string{"install-bundle", "select-executable", "install-hook"}
		if !result.Applied || !reflect.DeepEqual(result.AppliedMutationIDs, wantMutations) {
			t.Fatalf("successful apply result = %#v", result)
		}
		if err := verifyInstalledBundle(fixture.plan.Plan.Components[0].Destination, verified.manifest, verified.manifestRaw); err != nil {
			t.Fatalf("durable bundle: %v", err)
		}
		if after := snapshotTree(t, fixture.repository); !reflect.DeepEqual(after, beforeRepository) {
			t.Fatalf("setup changed repository bytes\nbefore: %v\nafter:  %v", beforeRepository, after)
		}
	})

	t.Run("temporary directory cleanup preserves installed runtime", func(t *testing.T) {
		fixture := newCleanMachineFixture(t)
		verified := mustVerifyCleanMachinePlan(t, fixture)
		if _, err := Apply(context.Background(), verified); err != nil {
			t.Fatal(err)
		}

		stableExecutable := filepath.Join(fixture.home, ".local", "bin", "gr")
		durableRuntime := filepath.Join(fixture.plan.Plan.Components[0].Destination, "runtime", "node", "bin", "node")
		if err := os.RemoveAll(fixture.bundleRoot); err != nil {
			t.Fatal(err)
		}
		if raw, err := os.ReadFile(stableExecutable); err != nil || !bytes.Equal(raw, []byte("fixture gr\n")) {
			t.Fatalf("temporary cleanup broke durable executable: raw=%q err=%v", raw, err)
		}
		if raw, err := os.ReadFile(durableRuntime); err != nil || !bytes.Equal(raw, []byte("fixture node\n")) {
			t.Fatalf("temporary cleanup broke durable runtime: raw=%q err=%v", raw, err)
		}
	})

	t.Run("unknown dependency stops before durable writes", func(t *testing.T) {
		fixture := newCleanMachineFixture(t)
		writeFixtureFile(t, filepath.Join(fixture.bundleRoot, "runtime", "unlisted", "bin", "tool"), []byte("unlisted dependency\n"), 0o755)
		beforeRepository := snapshotTree(t, fixture.repository)
		beforeHome := snapshotTree(t, fixture.home)

		emission, err := ExecuteApply(context.Background(), fixture.options, "task:clean-machine-unknown-dependency")
		if !errors.Is(err, ErrBundleInvalid) || !strings.Contains(err.Error(), "unmanifested directory runtime/unlisted") {
			t.Fatalf("ExecuteApply() error = %v, want exact unlisted dependency blocker", err)
		}
		assertTreesUnchanged(t, fixture.repository, fixture.home, beforeRepository, beforeHome)
		if emission.Receipt.Status != domain.SetupReceiptFailure || emission.Receipt.Authorization == nil || emission.RecoveryPath != "" {
			t.Fatalf("unknown dependency emission = %#v", emission)
		}
	})

	t.Run("concurrent target change is preserved", func(t *testing.T) {
		fixture := newCleanMachineFixture(t)
		verified := mustVerifyCleanMachinePlan(t, fixture)
		target, err := ambient.RegistrationTarget(ambient.ScaffoldCodex, fixture.home, fixture.repository)
		if err != nil {
			t.Fatal(err)
		}
		concurrent := []byte("owner_change = true\n")

		result, err := applyVerified(context.Background(), verified, applyHooks{
			beforeMutation: func(mutationID string) error {
				if mutationID == "install-hook" {
					writeFixtureFile(t, target.Path, concurrent, 0o600)
				}
				return nil
			},
		})
		if !errors.Is(err, ErrApplyStateChanged) || result.FailedMutationID != "install-hook" {
			t.Fatalf("concurrent apply result = %#v, error = %v", result, err)
		}
		raw, readErr := os.ReadFile(target.Path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if !bytes.Equal(raw, concurrent) {
			t.Fatalf("concurrent owner bytes were overwritten: %q", raw)
		}
	})

	t.Run("partial failure rolls back only completed mutations", func(t *testing.T) {
		fixture := newCleanMachineFixture(t)
		verified := mustVerifyCleanMachinePlan(t, fixture)
		stop := errors.New("fixture stop before hook")

		result, err := applyVerified(context.Background(), verified, applyHooks{
			beforeMutation: func(mutationID string) error {
				if mutationID == "install-hook" {
					return stop
				}
				return nil
			},
		})
		if !errors.Is(err, stop) {
			t.Fatalf("partial Apply() error = %v, want fixture stop", err)
		}
		if !reflect.DeepEqual(result.AppliedMutationIDs, []string{"install-bundle", "select-executable"}) || !reflect.DeepEqual(result.PendingMutationIDs, []string{"install-hook"}) {
			t.Fatalf("partial mutation inventory = %#v", result)
		}

		rollback, err := Rollback(context.Background(), RollbackOptions{
			Home: fixture.home, RepositoryRoot: fixture.repository, RecoveryPath: result.RecoveryPath,
		})
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(rollback.RolledBackMutationIDs, []string{"select-executable", "install-bundle"}) || len(rollback.PendingMutationIDs) != 0 {
			t.Fatalf("rollback result = %#v", rollback)
		}
		if _, err := os.Lstat(fixture.plan.Plan.Components[0].Destination); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("durable bundle survived rollback: %v", err)
		}
	})

	t.Run("provider trust remains an explicit user action", func(t *testing.T) {
		fixture := newCleanMachineFixture(t)
		verified := mustVerifyCleanMachinePlan(t, fixture)

		result, err := Apply(context.Background(), verified)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(result.PendingTrustStepIDs, []string{"provider-trust"}) {
			t.Fatalf("pending trust steps = %v", result.PendingTrustStepIDs)
		}
		if len(fixture.plan.Plan.TrustSteps) != 1 || fixture.plan.Plan.TrustSteps[0].ID != "provider-trust" || !fixture.plan.Plan.TrustSteps[0].Interactive {
			t.Fatalf("provider trust plan = %#v", fixture.plan.Plan.TrustSteps)
		}
	})

	t.Run("shared admission remains unknown after local setup", func(t *testing.T) {
		fixture := newCleanMachineFixture(t)

		emission, err := ExecuteApply(context.Background(), fixture.options, "task:clean-machine-shared-unknown")
		if !errors.Is(err, ErrDiagnosisNotReady) {
			t.Fatalf("ExecuteApply() error = %v, want post-apply readiness blocker", err)
		}
		if emission.Receipt.Status != domain.SetupReceiptFailure || emission.Diagnosis == nil {
			t.Fatalf("post-apply clean-machine emission = %#v", emission)
		}
		if emission.Diagnosis.SharedAdmissionActive || emission.Diagnosis.SharedAdmissionState != "unknown" {
			t.Fatalf("shared admission was overstated: %#v", emission.Diagnosis)
		}
	})
}

func newCleanMachineFixture(t *testing.T) verifyPlanFixture {
	t.Helper()
	fixture := newVerifyPlanFixture(t)
	if entries := snapshotTree(t, fixture.home); len(entries) != 0 {
		t.Fatalf("clean-machine home is not empty: %v", entries)
	}
	return fixture
}

func mustVerifyCleanMachinePlan(t *testing.T, fixture verifyPlanFixture) VerifiedPlan {
	t.Helper()
	verified, err := VerifyPlan(context.Background(), fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	return verified
}
