package setup

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/domain"
)

func TestApplyInstallsOnlyTheVerifiedDurableSetupAndLeavesTrustPending(t *testing.T) {
	fixture := newVerifyPlanFixture(t)
	verified, err := VerifyPlan(context.Background(), fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	beforeRepository := snapshotTree(t, fixture.repository)

	result, err := Apply(context.Background(), verified)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Applied || !result.Changed || result.Schema != ApplyResultSchemaV1 {
		t.Fatalf("apply result = %#v", result)
	}
	wantMutations := []string{"install-bundle", "select-executable", "install-hook"}
	if !reflect.DeepEqual(result.AppliedMutationIDs, wantMutations) {
		t.Fatalf("applied mutations = %v, want %v", result.AppliedMutationIDs, wantMutations)
	}
	if !reflect.DeepEqual(result.PendingTrustStepIDs, []string{"provider-trust"}) {
		t.Fatalf("pending trust steps = %v", result.PendingTrustStepIDs)
	}
	if result.PlanDigest != fixture.plan.Artifact.Digest() {
		t.Fatalf("applied plan digest = %s, want %s", result.PlanDigest, fixture.plan.Artifact.Digest())
	}

	versionRoot := fixture.plan.Plan.Components[0].Destination
	if err := verifyInstalledBundle(versionRoot, verified.manifest, verified.manifestRaw); err != nil {
		t.Fatalf("durable bundle: %v", err)
	}
	stableExecutable := filepath.Join(fixture.home, ".local", "bin", "gr")
	raw, err := os.ReadFile(stableExecutable)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, []byte("fixture gr\n")) {
		t.Fatalf("selected executable bytes = %q", raw)
	}
	info, err := os.Stat(stableExecutable)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("selected executable mode = %04o", info.Mode().Perm())
	}
	target, err := ambient.RegistrationTarget(ambient.ScaffoldCodex, fixture.home, fixture.repository)
	if err != nil {
		t.Fatal(err)
	}
	connection, err := ambient.PlanRegistration(target, stableExecutable)
	if err != nil {
		t.Fatal(err)
	}
	if !connection.AlreadyPresent {
		t.Fatal("scaffold registration does not select the durable executable")
	}
	if after := snapshotTree(t, fixture.repository); !reflect.DeepEqual(after, beforeRepository) {
		t.Fatalf("user-local setup changed repository bytes\nbefore: %v\nafter:  %v", beforeRepository, after)
	}

	if err := os.RemoveAll(fixture.bundleRoot); err != nil {
		t.Fatal(err)
	}
	if raw, err := os.ReadFile(stableExecutable); err != nil || !bytes.Equal(raw, []byte("fixture gr\n")) {
		t.Fatalf("temporary bundle cleanup broke selected executable: raw=%q err=%v", raw, err)
	}
}

func TestApplyRejectsBundleTamperingBeforeAnyDurableWrite(t *testing.T) {
	fixture := newVerifyPlanFixture(t)
	verified, err := VerifyPlan(context.Background(), fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, filepath.Join(fixture.bundleRoot, "bin", "gr"), []byte("tampered\n"), 0o755)
	beforeRepository := snapshotTree(t, fixture.repository)
	beforeHome := snapshotTree(t, fixture.home)

	if _, err := Apply(context.Background(), verified); !errors.Is(err, ErrApplyStateChanged) {
		t.Fatalf("Apply() error = %v, want %v", err, ErrApplyStateChanged)
	}
	assertTreesUnchanged(t, fixture.repository, fixture.home, beforeRepository, beforeHome)
}

func TestApplyUsesTheBoundedRepositoryScopeForClaudeWithoutForgingTrust(t *testing.T) {
	fixture := newVerifyPlanFixtureForScaffold(t, ambient.ScaffoldClaudeCode)
	verified, err := VerifyPlan(context.Background(), fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Apply(context.Background(), verified)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.PendingTrustStepIDs) != 0 {
		t.Fatalf("Claude setup invented a trust step: %v", result.PendingTrustStepIDs)
	}
	target, err := ambient.RegistrationTarget(ambient.ScaffoldClaudeCode, fixture.home, fixture.repository)
	if err != nil {
		t.Fatal(err)
	}
	if !pathWithin(fixture.repository, target.Path) {
		t.Fatalf("Claude target escaped repository: %s", target.Path)
	}
	connection, err := ambient.PlanRegistration(target, filepath.Join(fixture.home, ".local", "bin", "gr"))
	if err != nil {
		t.Fatal(err)
	}
	if !connection.AlreadyPresent {
		t.Fatal("Claude repository-scoped registration is not current")
	}
}

func TestApplyPreservesAuthorizedOwnerConfigAndRepairsExecutableMode(t *testing.T) {
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
	if _, err := Apply(context.Background(), verified); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(target.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(raw), string(owner)) {
		t.Fatalf("owner configuration was not preserved: %q", raw)
	}
	configInfo, err := os.Stat(target.Path)
	if err != nil {
		t.Fatal(err)
	}
	if configInfo.Mode().Perm() != 0o600 {
		t.Fatalf("owner configuration mode = %04o, want 0600", configInfo.Mode().Perm())
	}
	executableInfo, err := os.Stat(stableExecutable)
	if err != nil {
		t.Fatal(err)
	}
	if executableInfo.Mode().Perm() != 0o755 {
		t.Fatalf("repaired executable mode = %04o, want 0755", executableInfo.Mode().Perm())
	}
}

func TestApplyPreservesAConfigurationChangedAtTheJustInTimeCheck(t *testing.T) {
	fixture := newVerifyPlanFixture(t)
	verified, err := VerifyPlan(context.Background(), fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	target, err := ambient.RegistrationTarget(ambient.ScaffoldCodex, fixture.home, fixture.repository)
	if err != nil {
		t.Fatal(err)
	}
	concurrent := []byte("owner_change = true\n")

	_, err = applyVerified(context.Background(), verified, applyHooks{
		beforeMutation: func(mutationID string) error {
			if mutationID == "install-hook" {
				writeFixtureFile(t, target.Path, concurrent, 0o600)
			}
			return nil
		},
	})
	if !errors.Is(err, ErrApplyStateChanged) {
		t.Fatalf("Apply() error = %v, want %v", err, ErrApplyStateChanged)
	}
	raw, readErr := os.ReadFile(target.Path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(raw, concurrent) {
		t.Fatalf("concurrent owner configuration was overwritten: %q", raw)
	}
}

func TestApplyDoesNotRemoveABundleDestinationThatAppearsConcurrently(t *testing.T) {
	fixture := newVerifyPlanFixture(t)
	verified, err := VerifyPlan(context.Background(), fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	destination := fixture.plan.Plan.Components[0].Destination
	foreign := filepath.Join(destination, "owner-file")

	_, err = applyVerified(context.Background(), verified, applyHooks{
		beforeMutation: func(mutationID string) error {
			if mutationID == "install-bundle" {
				writeFixtureFile(t, foreign, []byte("owner\n"), 0o600)
			}
			return nil
		},
	})
	if !errors.Is(err, ErrApplyStateChanged) {
		t.Fatalf("Apply() error = %v, want %v", err, ErrApplyStateChanged)
	}
	raw, readErr := os.ReadFile(foreign)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(raw) != "owner\n" {
		t.Fatalf("concurrent bundle destination was changed: %q", raw)
	}
}

func reauthorizeVerifyPlanFixture(t *testing.T, fixture *verifyPlanFixture) {
	t.Helper()
	metadataRaw, err := os.ReadFile(fixture.options.ReleaseMetadataPath)
	if err != nil {
		t.Fatal(err)
	}
	manifestRaw, err := os.ReadFile(filepath.Join(fixture.bundleRoot, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	evidence := ReleaseEvidence{MetadataRaw: metadataRaw, ManifestRaw: manifestRaw}
	options := authorizationPlanOptions(fixture.repository, fixture.home, &evidence)
	options.OS = runtime.GOOS
	options.Arch = runtime.GOARCH
	options.Scaffold = fixture.options.Scaffold
	plan := mustGeneratePlan(t, options)
	writeFixtureFile(t, fixture.options.PlanPath, plan.Artifact.CanonicalJSON(), 0o644)
	authorization, err := domain.FreezePlanAuthorizationReference(*exactConsent(plan).Authorization)
	if err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, fixture.options.AuthorizationPath, authorization.CanonicalJSON(), 0o644)
	fixture.plan = plan
}
