package setup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/project"
	"github.com/heurema/goalrail/internal/releasebundle"
)

func TestGenerateProducesDeterministicCompletePlanWithoutWrites(t *testing.T) {
	repository := initializedRepository(t)
	home := t.TempDir()
	evidence := releaseEvidence(t, "darwin", "arm64")
	beforeRepository := snapshotTree(t, repository)
	beforeHome := snapshotTree(t, home)
	options := PlanOptions{
		RepositoryRoot: repository,
		Home:           home,
		OS:             "darwin",
		Arch:           "arm64",
		Scaffold:       ambient.ScaffoldCodex,
		Evidence:       &evidence,
	}

	first, err := Generate(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Generate(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if first.Plan.State != domain.SetupPlanComplete || len(first.Issues) != 0 {
		t.Fatalf("plan state = %s, issues = %#v", first.Plan.State, first.Issues)
	}
	if first.Plan.ProjectCodeWrites != 0 {
		t.Fatalf("project code writes = %d, want 0", first.Plan.ProjectCodeWrites)
	}
	if len(first.Plan.Components) != 4 || len(first.Plan.Mutations) != 3 || len(first.Plan.Rollback) != 3 || len(first.Plan.Verification) != 2 {
		t.Fatalf("plan inventory = components:%d mutations:%d rollback:%d verification:%d", len(first.Plan.Components), len(first.Plan.Mutations), len(first.Plan.Rollback), len(first.Plan.Verification))
	}
	if len(first.Plan.NetworkAccess) != 4 || len(first.Plan.Prerequisites) != 6 || len(first.Plan.TrustSteps) != 1 {
		t.Fatalf("plan environment = network:%d prerequisites:%d trust:%d", len(first.Plan.NetworkAccess), len(first.Plan.Prerequisites), len(first.Plan.TrustSteps))
	}
	if !bytes.Equal(first.Artifact.CanonicalJSON(), second.Artifact.CanonicalJSON()) || first.Artifact.Digest() != second.Artifact.Digest() {
		t.Fatal("equivalent inspection inputs did not produce a byte-identical setup plan")
	}
	if after := snapshotTree(t, repository); !reflect.DeepEqual(after, beforeRepository) {
		t.Fatalf("planner changed repository\nbefore: %v\nafter:  %v", beforeRepository, after)
	}
	if after := snapshotTree(t, home); !reflect.DeepEqual(after, beforeHome) {
		t.Fatalf("planner changed user home\nbefore: %v\nafter:  %v", beforeHome, after)
	}

	mutationIDs := make([]string, 0, len(first.Plan.Mutations))
	for _, mutation := range first.Plan.Mutations {
		mutationIDs = append(mutationIDs, mutation.ID)
	}
	sort.Strings(mutationIDs)
	if strings.Join(mutationIDs, ",") != "install-bundle,install-hook,select-executable" {
		t.Fatalf("mutations = %v", mutationIDs)
	}
	for _, component := range first.Plan.Components {
		if component.Scope != domain.SetupScopeUserLocal || !filepath.IsAbs(component.Destination) || component.SizeBytes == 0 || !domain.IsSHA256Digest(component.Integrity) {
			t.Fatalf("component is not exact and user-local: %#v", component)
		}
	}
}

func TestGenerateRecognizesAnExactExistingInstallationWithoutWrites(t *testing.T) {
	repository := initializedRepository(t)
	home := t.TempDir()
	evidence := releaseEvidence(t, "darwin", "arm64")
	versionRoot := filepath.Join(home, ".local", "share", "goalrail", "bundles", "v0.2.0", "darwin_arm64")
	writeFixtureFile(t, filepath.Join(versionRoot, "bin", "gr"), []byte("fixture gr\n"), 0o755)
	writeFixtureFile(t, filepath.Join(versionRoot, "compiler", "node_modules", "@fission-ai", "openspec", "bin", "openspec.js"), []byte("#!/usr/bin/env node\n"), 0o755)
	writeFixtureFile(t, filepath.Join(versionRoot, "compiler", "package-lock.json"), []byte("{}\n"), 0o644)
	writeFixtureFile(t, filepath.Join(versionRoot, "runtime", "node", "bin", "node"), []byte("fixture node\n"), 0o755)
	writeFixtureFile(t, filepath.Join(versionRoot, releasebundle.BundleManifestPath), evidence.ManifestRaw, 0o644)
	stableExecutable := filepath.Join(home, ".local", "bin", "gr")
	writeFixtureFile(t, stableExecutable, []byte("fixture gr\n"), 0o755)
	target, err := ambient.RegistrationTarget(ambient.ScaffoldCodex, home, repository)
	if err != nil {
		t.Fatal(err)
	}
	connection, err := ambient.PlanRegistration(target, stableExecutable)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ambient.Connect(connection); err != nil {
		t.Fatal(err)
	}
	beforeRepository := snapshotTree(t, repository)
	beforeHome := snapshotTree(t, home)

	result, err := Generate(context.Background(), PlanOptions{
		RepositoryRoot: repository,
		Home:           home,
		OS:             "darwin",
		Arch:           "arm64",
		Scaffold:       ambient.ScaffoldCodex,
		Evidence:       &evidence,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Plan.State != domain.SetupPlanComplete || len(result.Plan.Mutations) != 0 || len(result.Plan.Rollback) != 0 {
		t.Fatalf("exact installation was not recognized: state=%s mutations=%#v issues=%#v", result.Plan.State, result.Plan.Mutations, result.Issues)
	}
	if after := snapshotTree(t, repository); !reflect.DeepEqual(after, beforeRepository) {
		t.Fatalf("planner changed repository\nbefore: %v\nafter:  %v", beforeRepository, after)
	}
	if after := snapshotTree(t, home); !reflect.DeepEqual(after, beforeHome) {
		t.Fatalf("planner changed user home\nbefore: %v\nafter:  %v", beforeHome, after)
	}
}

func TestGenerateSelectsAnExecutableWhoseBytesMatchButModeDoesNot(t *testing.T) {
	repository := initializedRepository(t)
	home := t.TempDir()
	evidence := releaseEvidence(t, "darwin", "arm64")
	versionRoot := filepath.Join(home, ".local", "share", "goalrail", "bundles", "v0.2.0", "darwin_arm64")
	materializeFixtureBundle(t, versionRoot, evidence.ManifestRaw)
	stableExecutable := filepath.Join(home, ".local", "bin", "gr")
	writeFixtureFile(t, stableExecutable, []byte("fixture gr\n"), 0o644)

	result, err := Generate(context.Background(), PlanOptions{
		RepositoryRoot: repository,
		Home:           home,
		OS:             "darwin",
		Arch:           "arm64",
		Scaffold:       ambient.ScaffoldCodex,
		Evidence:       &evidence,
	})
	if err != nil {
		t.Fatal(err)
	}
	var selection *domain.SetupMutation
	for index := range result.Plan.Mutations {
		if result.Plan.Mutations[index].ID == "select-executable" {
			selection = &result.Plan.Mutations[index]
		}
	}
	if selection == nil || selection.ExpectedBeforeDigest == nil || *selection.ExpectedBeforeDigest != selection.DesiredDigest {
		t.Fatalf("mode-only executable repair was not planned: %#v", selection)
	}
}

func TestCTX8DifferentExecutableDigestIsPlanningConflict(t *testing.T) {
	repository := initializedRepository(t)
	home := t.TempDir()
	evidence := releaseEvidence(t, "darwin", "arm64")
	versionRoot := filepath.Join(home, ".local", "share", "goalrail", "bundles", "v0.2.0", "darwin_arm64")
	materializeFixtureBundle(t, versionRoot, evidence.ManifestRaw)
	stableExecutable := filepath.Join(home, ".local", "bin", "gr")
	desired := []byte("fixture gr\n")
	writeFixtureFile(t, stableExecutable, bytes.Repeat([]byte("x"), len(desired)), 0o755)
	beforeRepository := snapshotTree(t, repository)
	beforeHome := snapshotTree(t, home)

	result, err := Generate(context.Background(), PlanOptions{
		RepositoryRoot: repository,
		Home:           home,
		OS:             "darwin",
		Arch:           "arm64",
		Scaffold:       ambient.ScaffoldCodex,
		Evidence:       &evidence,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Plan.State != domain.SetupPlanIncomplete || !contains(result.Plan.IncompleteReasonIDs, "EXECUTABLE_DESTINATION_CONFLICT") {
		t.Fatalf("state = %s, reasons = %v, issues = %#v", result.Plan.State, result.Plan.IncompleteReasonIDs, result.Issues)
	}
	for _, mutation := range result.Plan.Mutations {
		if mutation.ID == "select-executable" {
			t.Fatalf("different-digest executable retained an authorizable mutation: %#v", mutation)
		}
	}
	if len(result.Plan.Rollback) != 0 || len(result.Plan.Verification) != 0 || result.Plan.ProjectCodeWrites != 0 {
		t.Fatalf("incomplete plan retained executable work: %#v", result.Plan)
	}
	if after := snapshotTree(t, repository); !reflect.DeepEqual(after, beforeRepository) {
		t.Fatalf("planner changed repository\nbefore: %v\nafter:  %v", beforeRepository, after)
	}
	if after := snapshotTree(t, home); !reflect.DeepEqual(after, beforeHome) {
		t.Fatalf("planner changed user home\nbefore: %v\nafter:  %v", beforeHome, after)
	}
}

func TestGenerateEmitsCanonicalIncompletePlansWithoutExecutableWork(t *testing.T) {
	repository := initializedRepository(t)
	home := t.TempDir()
	evidence := releaseEvidence(t, "darwin", "arm64")
	nonCanonicalMetadata := evidence
	nonCanonicalMetadata.MetadataRaw = append([]byte{' '}, evidence.MetadataRaw...)
	tamperedManifest := evidence
	tamperedManifest.ManifestRaw = append([]byte(nil), evidence.ManifestRaw...)
	tamperedManifest.ManifestRaw[len(tamperedManifest.ManifestRaw)-2] = ' '

	for _, test := range []struct {
		name       string
		options    PlanOptions
		prepare    func(t *testing.T)
		wantReason string
	}{
		{
			name:       "metadata unavailable",
			options:    PlanOptions{RepositoryRoot: repository, Home: home, OS: "darwin", Arch: "arm64", Scaffold: ambient.ScaffoldCodex},
			wantReason: "RELEASE_METADATA_UNAVAILABLE",
		},
		{
			name:       "platform unsupported",
			options:    PlanOptions{RepositoryRoot: repository, Home: home, OS: "linux", Arch: "s390x", Scaffold: ambient.ScaffoldCodex, Evidence: &evidence},
			wantReason: "PLATFORM_UNSUPPORTED",
		},
		{
			name:       "metadata noncanonical",
			options:    PlanOptions{RepositoryRoot: repository, Home: home, OS: "darwin", Arch: "arm64", Scaffold: ambient.ScaffoldCodex, Evidence: &nonCanonicalMetadata},
			wantReason: "RELEASE_METADATA_INVALID",
		},
		{
			name:       "manifest digest mismatch",
			options:    PlanOptions{RepositoryRoot: repository, Home: home, OS: "darwin", Arch: "arm64", Scaffold: ambient.ScaffoldCodex, Evidence: &tamperedManifest},
			wantReason: "SETUP_MANIFEST_DIGEST_MISMATCH",
		},
		{
			name:    "symlinked provider config",
			options: PlanOptions{RepositoryRoot: repository, Home: home, OS: "darwin", Arch: "arm64", Scaffold: ambient.ScaffoldCodex, Evidence: &evidence},
			prepare: func(t *testing.T) {
				configDirectory := filepath.Join(home, ".codex")
				if err := os.MkdirAll(configDirectory, 0o755); err != nil {
					t.Fatal(err)
				}
				owner := filepath.Join(home, "owner-config.toml")
				if err := os.WriteFile(owner, []byte("model = \"owner\"\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(owner, filepath.Join(configDirectory, "config.toml")); err != nil {
					t.Fatal(err)
				}
			},
			wantReason: "SCAFFOLD_TARGET_UNSAFE",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.prepare != nil {
				test.prepare(t)
			}
			beforeRepository := snapshotTree(t, repository)
			beforeHome := snapshotTree(t, home)
			result, err := Generate(context.Background(), test.options)
			if err != nil {
				t.Fatal(err)
			}
			if result.Plan.State != domain.SetupPlanIncomplete || !contains(result.Plan.IncompleteReasonIDs, test.wantReason) {
				t.Fatalf("state = %s, reasons = %v, issues = %#v", result.Plan.State, result.Plan.IncompleteReasonIDs, result.Issues)
			}
			if len(result.Plan.Mutations) != 0 || len(result.Plan.Rollback) != 0 || len(result.Plan.Verification) != 0 || result.Plan.ProjectCodeWrites != 0 {
				t.Fatalf("incomplete plan retained executable work: %#v", result.Plan)
			}
			if _, err := domain.DecodeSetupPlan(bytes.NewReader(result.Artifact.CanonicalJSON())); err != nil {
				t.Fatalf("incomplete plan is not a valid canonical contract: %v", err)
			}
			if after := snapshotTree(t, repository); !reflect.DeepEqual(after, beforeRepository) {
				t.Fatalf("planner changed repository\nbefore: %v\nafter:  %v", beforeRepository, after)
			}
			if after := snapshotTree(t, home); !reflect.DeepEqual(after, beforeHome) {
				t.Fatalf("planner changed user home\nbefore: %v\nafter:  %v", beforeHome, after)
			}
		})
	}
}

func TestGenerateRejectsUnmanagedCheckoutInsteadOfInventingIdentity(t *testing.T) {
	repository := t.TempDir()
	runGit(t, repository, "init", "--quiet")
	_, err := Generate(context.Background(), PlanOptions{
		RepositoryRoot: repository,
		Home:           t.TempDir(),
		OS:             "darwin",
		Arch:           "arm64",
		Scaffold:       ambient.ScaffoldCodex,
	})
	if err == nil || !strings.Contains(err.Error(), "managed project claim") {
		t.Fatalf("unmanaged planning error = %v", err)
	}
}

func initializedRepository(t *testing.T) string {
	t.Helper()
	repository := t.TempDir()
	runGit(t, repository, "init", "--quiet")
	if _, err := project.Initialize(context.Background(), repository, project.InitializeOptions{}); err != nil {
		t.Fatal(err)
	}
	return repository
}

func releaseEvidence(t *testing.T, goos, arch string) ReleaseEvidence {
	t.Helper()
	platform := releasebundle.Platform{OS: goos, Arch: arch}
	grRaw := []byte("fixture gr\n")
	compilerRaw := []byte("#!/usr/bin/env node\n")
	lockRaw := []byte("{}\n")
	nodeRaw := []byte("fixture node\n")
	files := []releasebundle.ManifestFile{
		{Path: "bin/gr", ComponentID: "goalrail", SizeBytes: int64(len(grRaw)), SHA256: digestString(grRaw), Mode: "0755"},
		{Path: "compiler/node_modules/@fission-ai/openspec/bin/openspec.js", ComponentID: "npm-fission-ai-openspec", SizeBytes: int64(len(compilerRaw)), SHA256: digestString(compilerRaw), Mode: "0755"},
		{Path: "compiler/package-lock.json", ComponentID: "compiler-lock", SizeBytes: int64(len(lockRaw)), SHA256: digestString(lockRaw), Mode: "0644"},
		{Path: "runtime/node/bin/node", ComponentID: "node", SizeBytes: int64(len(nodeRaw)), SHA256: digestString(nodeRaw), Mode: "0755"},
	}
	components := []releasebundle.ManifestComponent{
		{
			ID: "compiler-lock", Name: "goalrail-private-planning-runtime", Kind: "npm-lock", Version: "1",
			Integrity: digestString(lockRaw), LicenseRef: "manifest:compiler-lock/license", ProvenanceRef: "manifest:compiler-lock/provenance",
			Dependencies: []releasebundle.ManifestDependency{{Name: "@fission-ai/openspec", Requested: "1.6.0", ComponentID: "npm-fission-ai-openspec"}},
		},
		{ID: "goalrail", Name: "goalrail", Kind: "go-binary", Version: "v0.2.0", Integrity: digestString(grRaw), LicenseRef: "manifest:goalrail/license", ProvenanceRef: "manifest:goalrail/provenance", Dependencies: []releasebundle.ManifestDependency{}},
		{ID: "node", Name: "node", Kind: "runtime", Version: "22.18.0", Integrity: digestString(nodeRaw), LicenseRef: "manifest:node/license", ProvenanceRef: "manifest:node/provenance", Dependencies: []releasebundle.ManifestDependency{}},
		{ID: "npm-fission-ai-openspec", Name: "@fission-ai/openspec", Kind: "npm", Version: "1.6.0", Integrity: "sha512-fixture", LicenseRef: "manifest:openspec/license", ProvenanceRef: "manifest:openspec/provenance", Dependencies: []releasebundle.ManifestDependency{}},
	}
	identities := []releasebundle.BinaryIdentity{
		{ComponentID: "goalrail", Path: "bin/gr", Kind: "go-buildinfo", Version: "v0.2.0", SHA256: files[0].SHA256, SourceIntegrity: files[0].SHA256},
		{ComponentID: "npm-fission-ai-openspec", Path: "compiler/node_modules/@fission-ai/openspec/bin/openspec.js", Kind: "npm-bin", Version: "1.6.0", SHA256: files[1].SHA256, SourceIntegrity: "sha512-fixture"},
		{ComponentID: "node", Path: "runtime/node/bin/node", Kind: "node-distribution", Version: "22.18.0", SHA256: files[3].SHA256, SourceIntegrity: files[3].SHA256},
	}
	manifest := releasebundle.SetupBundleManifest{
		Schema: releasebundle.SetupManifestSchemaV1, ReleaseVersion: "v0.2.0", Platform: platform,
		ArchiveName: "goalrail-setup_v0.2.0_" + platform.Key() + ".tar.gz", ManifestPath: releasebundle.BundleManifestPath,
		CompilerLockDigest: digestString(lockRaw), CompilerInstallPolicy: "never-run-package-scripts",
		Components: components, BinaryIdentities: identities, Files: files,
	}
	manifestRaw := canonicalJSON(t, manifest)
	goalrailIdentity := identities[0]
	metadata := releasebundle.ReleaseMetadata{
		Schema: releasebundle.ReleaseMetadataSchema, ReleaseVersion: "v0.2.0", ChecksumArtifact: releasebundle.ChecksumsName,
		Compatibility: releasebundle.ReleaseCompatibility{
			GovernanceContract: "goalrail-governance-v1", SetupProfileSchema: domain.SetupProfileSchemaV1,
			SetupManifestSchema: releasebundle.SetupManifestSchemaV1, RuntimeID: "node", RuntimeVersion: "22.18.0",
			CompilerID: "@fission-ai/openspec", CompilerVersion: "1.6.0", CompilerInstallPolicy: "never-run-package-scripts",
		},
		SupportedPlatforms: []releasebundle.ReleasePlatformMetadata{{
			Platform: platform, GoalrailBinary: goalrailIdentity,
			MinimalArchive: releasebundle.PublishedArtifact{Name: "gr_v0.2.0_" + platform.Key() + ".tar.gz", SizeBytes: 11, SHA256: digestString([]byte("minimal"))},
			SetupArchive:   releasebundle.PublishedArtifact{Name: manifest.ArchiveName, SizeBytes: 99, SHA256: digestString([]byte("setup archive"))},
			SetupManifest:  releasebundle.PublishedArtifact{Name: "goalrail-setup_v0.2.0_" + platform.Key() + ".manifest.json", SizeBytes: int64(len(manifestRaw)), SHA256: digestString(manifestRaw)},
		}},
	}
	return ReleaseEvidence{MetadataRaw: canonicalJSON(t, metadata), ManifestRaw: manifestRaw}
}

func canonicalJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return append(raw, '\n')
}

func digestString(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func snapshotTree(t *testing.T, root string) []string {
	t.Helper()
	var entries []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		kind := "file"
		identity := ""
		switch {
		case entry.Type()&os.ModeSymlink != 0:
			kind = "symlink"
			identity, err = os.Readlink(path)
		case entry.IsDir():
			kind = "directory"
		default:
			raw, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			identity = digestString(raw)
		}
		if err != nil {
			return err
		}
		entries = append(entries, fmt.Sprintf("%s|%s|%04o|%s", filepath.ToSlash(relative), kind, info.Mode().Perm(), identity))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(entries)
	return entries
}

func runGit(t *testing.T, directory string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func writeFixtureFile(t *testing.T, path string, raw []byte, mode fs.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, mode); err != nil {
		t.Fatal(err)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
