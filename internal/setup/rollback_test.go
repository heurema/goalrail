package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/heurema/goalrail/internal/ambient"
)

func TestApplyReportsPartialProgressAndRollbackUsesOnlyCompletedMutations(t *testing.T) {
	fixture := newVerifyPlanFixture(t)
	verified, err := VerifyPlan(context.Background(), fixture.options)
	if err != nil {
		t.Fatal(err)
	}
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
		t.Fatalf("Apply() error = %v, want fixture stop", err)
	}
	if result.Applied || !result.Changed || result.FailedMutationID != "install-hook" {
		t.Fatalf("partial apply result = %#v", result)
	}
	if !reflect.DeepEqual(result.AppliedMutationIDs, []string{"install-bundle", "select-executable"}) || !reflect.DeepEqual(result.PendingMutationIDs, []string{"install-hook"}) {
		t.Fatalf("partial mutation inventory = applied:%v pending:%v", result.AppliedMutationIDs, result.PendingMutationIDs)
	}
	assertPrivateRecoveryFile(t, result.RecoveryPath)

	rollback, err := Rollback(context.Background(), RollbackOptions{
		Home: fixture.home, RepositoryRoot: fixture.repository, RecoveryPath: result.RecoveryPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.RolledBack || !rollback.Changed || len(rollback.PendingMutationIDs) != 0 {
		t.Fatalf("rollback result = %#v", rollback)
	}
	if !reflect.DeepEqual(rollback.RolledBackMutationIDs, []string{"select-executable", "install-bundle"}) {
		t.Fatalf("rolled back mutations = %v", rollback.RolledBackMutationIDs)
	}
	if _, err := os.Lstat(filepath.Join(fixture.home, ".local", "bin", "gr")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("selected executable survived rollback: %v", err)
	}
	if _, err := os.Lstat(fixture.plan.Plan.Components[0].Destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("versioned bundle survived rollback: %v", err)
	}
	target, err := ambient.RegistrationTarget(ambient.ScaffoldCodex, fixture.home, fixture.repository)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(target.Path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unperformed hook target was changed: %v", err)
	}
}

func TestRollbackRestoresExactPriorConfigurationAndExecutableMode(t *testing.T) {
	fixture := newVerifyPlanFixture(t)
	target, err := ambient.RegistrationTarget(ambient.ScaffoldCodex, fixture.home, fixture.repository)
	if err != nil {
		t.Fatal(err)
	}
	owner := []byte("model = \"owner\"\n")
	writeFixtureFile(t, target.Path, owner, 0o600)
	stableExecutable := filepath.Join(fixture.home, ".local", "bin", "gr")
	writeFixtureFile(t, stableExecutable, []byte("fixture gr\n"), 0o644)
	reauthorizeVerifyPlanFixture(t, &fixture)
	verified, err := VerifyPlan(context.Background(), fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Apply(context.Background(), verified)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, owner) {
		t.Fatalf("apply result exposed prior configuration: %s", encoded)
	}

	rollback, err := Rollback(context.Background(), RollbackOptions{
		Home: fixture.home, RepositoryRoot: fixture.repository, RecoveryPath: result.RecoveryPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !rollback.RolledBack || !rollback.Changed {
		t.Fatalf("rollback result = %#v", rollback)
	}
	raw, err := os.ReadFile(target.Path)
	if err != nil || !bytes.Equal(raw, owner) {
		t.Fatalf("restored config = %q, err=%v", raw, err)
	}
	configInfo, err := os.Stat(target.Path)
	if err != nil || configInfo.Mode().Perm() != 0o600 {
		t.Fatalf("restored config mode = %v, err=%v", configInfo.Mode(), err)
	}
	executableRaw, err := os.ReadFile(stableExecutable)
	if err != nil || !bytes.Equal(executableRaw, []byte("fixture gr\n")) {
		t.Fatalf("restored executable = %q, err=%v", executableRaw, err)
	}
	executableInfo, err := os.Stat(stableExecutable)
	if err != nil || executableInfo.Mode().Perm() != 0o644 {
		t.Fatalf("restored executable mode = %v, err=%v", executableInfo.Mode(), err)
	}
	if _, err := os.Lstat(fixture.plan.Plan.Components[0].Destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("owned bundle survived rollback: %v", err)
	}
	assertPrivateRecoveryFile(t, result.RecoveryPath)
}

func TestRollbackRefusesAConfigurationChangedAfterApply(t *testing.T) {
	fixture := newVerifyPlanFixture(t)
	verified, err := VerifyPlan(context.Background(), fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Apply(context.Background(), verified)
	if err != nil {
		t.Fatal(err)
	}
	target, err := ambient.RegistrationTarget(ambient.ScaffoldCodex, fixture.home, fixture.repository)
	if err != nil {
		t.Fatal(err)
	}
	foreign := []byte("owner_changed_after_apply = true\n")
	writeFixtureFile(t, target.Path, foreign, 0o600)

	rollback, err := Rollback(context.Background(), RollbackOptions{
		Home: fixture.home, RepositoryRoot: fixture.repository, RecoveryPath: result.RecoveryPath,
	})
	if !errors.Is(err, ErrRollbackStateChanged) {
		t.Fatalf("Rollback() error = %v, want %v", err, ErrRollbackStateChanged)
	}
	if rollback.Changed || rollback.FailedMutationID != "install-hook" {
		t.Fatalf("rollback result = %#v", rollback)
	}
	raw, readErr := os.ReadFile(target.Path)
	if readErr != nil || !bytes.Equal(raw, foreign) {
		t.Fatalf("foreign config was changed: %q, err=%v", raw, readErr)
	}
	if _, err := os.Stat(filepath.Join(fixture.home, ".local", "bin", "gr")); err != nil {
		t.Fatalf("rollback continued past conflict and removed executable: %v", err)
	}
}

func TestRollbackNeverDeletesABundleWithUnrelatedContent(t *testing.T) {
	fixture := newVerifyPlanFixture(t)
	verified, err := VerifyPlan(context.Background(), fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Apply(context.Background(), verified)
	if err != nil {
		t.Fatal(err)
	}
	bundleRoot := fixture.plan.Plan.Components[0].Destination
	foreignPath := filepath.Join(bundleRoot, "owner-file")
	writeFixtureFile(t, foreignPath, []byte("owner\n"), 0o600)

	rollback, err := Rollback(context.Background(), RollbackOptions{
		Home: fixture.home, RepositoryRoot: fixture.repository, RecoveryPath: result.RecoveryPath,
	})
	if !errors.Is(err, ErrRollbackStateChanged) {
		t.Fatalf("Rollback() error = %v, want %v", err, ErrRollbackStateChanged)
	}
	if rollback.FailedMutationID != "install-bundle" {
		t.Fatalf("rollback result = %#v", rollback)
	}
	if raw, readErr := os.ReadFile(foreignPath); readErr != nil || string(raw) != "owner\n" {
		t.Fatalf("unrelated bundle content was deleted: %q, err=%v", raw, readErr)
	}
	if raw, readErr := os.ReadFile(filepath.Join(bundleRoot, "bin", "gr")); readErr != nil || string(raw) != "fixture gr\n" {
		t.Fatalf("verified bundle was partially deleted before refusal: %q, err=%v", raw, readErr)
	}
}

func assertPrivateRecoveryFile(t *testing.T, path string) {
	t.Helper()
	if path == "" {
		t.Fatal("recovery path is empty")
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		t.Fatalf("recovery mode = %v, want private regular 0600", info.Mode())
	}
}
