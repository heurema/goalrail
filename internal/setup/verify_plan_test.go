package setup

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"testing"

	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/releasebundle"
)

func TestVerifyPlanAcceptsExactTemporaryBundleWithoutWrites(t *testing.T) {
	fixture := newVerifyPlanFixture(t)
	beforeRepository := snapshotTree(t, fixture.repository)
	beforeHome := snapshotTree(t, fixture.home)
	beforeBundle := snapshotTree(t, fixture.bundleRoot)

	verified, err := VerifyPlan(context.Background(), fixture.options)
	if err != nil {
		t.Fatal(err)
	}
	report := verified.Report()
	if !report.Verified || report.Schema != PlanVerificationSchemaV1 {
		t.Fatalf("verification report = %#v", report)
	}
	if report.ProjectID != fixture.plan.Plan.ProjectID || report.PlanDigest != fixture.plan.Artifact.Digest() {
		t.Fatalf("verification lost authorized plan identity: %#v", report)
	}
	if report.ReleaseVersion != "v0.2.0" || report.Platform != runtime.GOOS+"-"+runtime.GOARCH {
		t.Fatalf("verification release identity = %#v", report)
	}
	resolvedBundle, err := filepath.EvalSymlinks(fixture.bundleRoot)
	if err != nil {
		t.Fatal(err)
	}
	if verified.bundleRoot != resolvedBundle || verified.attempt.PlanArtifact().Digest() != fixture.plan.Artifact.Digest() {
		t.Fatal("internal verified proof does not retain the exact bundle and plan")
	}
	assertTreesUnchanged(t, fixture.repository, fixture.home, beforeRepository, beforeHome)
	if after := snapshotTree(t, fixture.bundleRoot); !reflect.DeepEqual(after, beforeBundle) {
		t.Fatalf("verification changed temporary bundle\nbefore: %v\nafter:  %v", beforeBundle, after)
	}
}

func TestVerifyPlanRejectsTamperedIncompleteAndUnmanifestedBundles(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(t *testing.T, fixture verifyPlanFixture)
	}{
		{
			name: "tampered file",
			mutate: func(t *testing.T, fixture verifyPlanFixture) {
				writeFixtureFile(t, filepath.Join(fixture.bundleRoot, "bin", "gr"), []byte("fixture gx\n"), 0o755)
			},
		},
		{
			name: "missing file",
			mutate: func(t *testing.T, fixture verifyPlanFixture) {
				if err := os.Remove(filepath.Join(fixture.bundleRoot, "bin", "gr")); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "unmanifested file",
			mutate: func(t *testing.T, fixture verifyPlanFixture) {
				writeFixtureFile(t, filepath.Join(fixture.bundleRoot, "unexpected"), []byte("no\n"), 0o644)
			},
		},
		{
			name: "symbolic link",
			mutate: func(t *testing.T, fixture verifyPlanFixture) {
				target := filepath.Join(fixture.bundleRoot, "outside")
				writeFixtureFile(t, target, []byte("fixture gr\n"), 0o755)
				path := filepath.Join(fixture.bundleRoot, "bin", "gr")
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "manifest with unsafe mode",
			mutate: func(t *testing.T, fixture verifyPlanFixture) {
				if err := os.Chmod(filepath.Join(fixture.bundleRoot, releasebundle.BundleManifestPath), 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newVerifyPlanFixture(t)
			test.mutate(t, fixture)
			beforeRepository := snapshotTree(t, fixture.repository)
			beforeHome := snapshotTree(t, fixture.home)
			if _, err := VerifyPlan(context.Background(), fixture.options); !errors.Is(err, ErrBundleInvalid) {
				t.Fatalf("VerifyPlan() error = %v, want %v", err, ErrBundleInvalid)
			}
			assertTreesUnchanged(t, fixture.repository, fixture.home, beforeRepository, beforeHome)
		})
	}
}

func TestVerifyPlanRejectsAPlanThatIsNotSemanticallyCurrent(t *testing.T) {
	fixture := newVerifyPlanFixture(t)
	fixture.options.Scaffold = ambient.ScaffoldClaudeCode
	beforeRepository := snapshotTree(t, fixture.repository)
	beforeHome := snapshotTree(t, fixture.home)

	if _, err := VerifyPlan(context.Background(), fixture.options); !errors.Is(err, ErrAuthorizationStale) {
		t.Fatalf("semantic mismatch error = %v, want %v", err, ErrAuthorizationStale)
	}
	assertTreesUnchanged(t, fixture.repository, fixture.home, beforeRepository, beforeHome)
}

func TestVerifyPlanRejectsStateChangedAfterAuthorization(t *testing.T) {
	fixture := newVerifyPlanFixture(t)
	writeFixtureFile(t, filepath.Join(fixture.home, ".local", "bin", "gr"), []byte("fixture gr\n"), 0o755)
	beforeRepository := snapshotTree(t, fixture.repository)
	beforeHome := snapshotTree(t, fixture.home)

	if _, err := VerifyPlan(context.Background(), fixture.options); !errors.Is(err, ErrAuthorizationStale) {
		t.Fatalf("changed target error = %v, want %v", err, ErrAuthorizationStale)
	}
	assertTreesUnchanged(t, fixture.repository, fixture.home, beforeRepository, beforeHome)
}

func TestVerifyPlanRequiresCanonicalStoredAuthority(t *testing.T) {
	fixture := newVerifyPlanFixture(t)
	raw, err := os.ReadFile(fixture.options.PlanPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture.options.PlanPath, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyPlan(context.Background(), fixture.options); err == nil {
		t.Fatal("noncanonical stored plan was accepted")
	}
}

type verifyPlanFixture struct {
	repository string
	home       string
	bundleRoot string
	plan       Result
	options    VerifyPlanOptions
}

func newVerifyPlanFixture(t *testing.T) verifyPlanFixture {
	return newVerifyPlanFixtureForScaffold(t, ambient.ScaffoldCodex)
}

func newVerifyPlanFixtureForScaffold(t *testing.T, scaffold ambient.Scaffold) verifyPlanFixture {
	t.Helper()
	repository := initializedRepository(t)
	home := t.TempDir()
	evidence := releaseEvidence(t, runtime.GOOS, runtime.GOARCH)
	planOptions := authorizationPlanOptions(repository, home, &evidence)
	planOptions.OS = runtime.GOOS
	planOptions.Arch = runtime.GOARCH
	planOptions.Scaffold = scaffold
	plan := mustGeneratePlan(t, planOptions)

	artifacts := t.TempDir()
	planPath := filepath.Join(artifacts, "plan.json")
	writeFixtureFile(t, planPath, plan.Artifact.CanonicalJSON(), 0o644)
	authorizationArtifact, err := domain.FreezePlanAuthorizationReference(*exactConsent(plan).Authorization)
	if err != nil {
		t.Fatal(err)
	}
	authorizationPath := filepath.Join(artifacts, "authorization.json")
	writeFixtureFile(t, authorizationPath, authorizationArtifact.CanonicalJSON(), 0o644)
	metadataPath := filepath.Join(artifacts, "release.json")
	writeFixtureFile(t, metadataPath, evidence.MetadataRaw, 0o644)

	bundleRoot := t.TempDir()
	materializeFixtureBundle(t, bundleRoot, evidence.ManifestRaw)
	return verifyPlanFixture{
		repository: repository,
		home:       home,
		bundleRoot: bundleRoot,
		plan:       plan,
		options: VerifyPlanOptions{
			RepositoryRoot:      repository,
			Home:                home,
			BundleRoot:          bundleRoot,
			PlanPath:            planPath,
			AuthorizationPath:   authorizationPath,
			ReleaseMetadataPath: metadataPath,
			Scaffold:            scaffold,
		},
	}
}

func materializeFixtureBundle(t *testing.T, root string, manifestRaw []byte) {
	t.Helper()
	manifest, err := releasebundle.DecodeSetupBundleManifest(bytes.NewReader(manifestRaw))
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string][]byte{
		"bin/gr": []byte("fixture gr\n"),
		"compiler/node_modules/@fission-ai/openspec/bin/openspec.js": []byte("#!/usr/bin/env node\n"),
		"compiler/package-lock.json":                                 []byte("{}\n"),
		"runtime/node/bin/node":                                      []byte("fixture node\n"),
	}
	for _, record := range manifest.Files {
		raw, ok := contents[record.Path]
		if !ok {
			t.Fatalf("fixture has no bytes for manifest path %s", record.Path)
		}
		mode, err := strconv.ParseUint(record.Mode, 8, 32)
		if err != nil {
			t.Fatal(err)
		}
		writeFixtureFile(t, filepath.Join(root, filepath.FromSlash(record.Path)), raw, os.FileMode(mode))
	}
	writeFixtureFile(t, filepath.Join(root, releasebundle.BundleManifestPath), manifestRaw, 0o644)
}
