package admissiongit

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/admission"
	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/lineage"
	"github.com/heurema/goalrail/internal/localrun"
)

func TestCollectChangesCoversGitKindsAndIgnoresCheckoutPathAndEnvironment(t *testing.T) {
	repository := t.TempDir()
	initTestRepository(t, repository)
	writeTestPath(t, repository, "edit.txt", "before\n", 0o644)
	writeTestPath(t, repository, "delete.txt", "delete\n", 0o644)
	writeTestPath(t, repository, "rename-old.txt", "rename\n", 0o644)
	writeTestPath(t, repository, "mode.sh", "#!/bin/sh\n", 0o644)
	runTestGit(t, repository, "add", "-A")
	runTestGit(t, repository, "commit", "-qm", "base")
	base := strings.TrimSpace(runTestGit(t, repository, "rev-parse", "HEAD"))

	writeTestPath(t, repository, "edit.txt", "after\n", 0o644)
	writeTestPath(t, repository, "add.txt", "add\n", 0o644)
	if err := os.Remove(filepath.Join(repository, "delete.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(repository, "rename-old.txt"), filepath.Join(repository, "rename-new.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(repository, "mode.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, repository, "add", "-A")
	runTestGit(t, repository, "update-index", "--add", "--cacheinfo", "160000,"+base+",submodule")
	runTestGit(t, repository, "commit", "-qm", "all change kinds")
	head := strings.TrimSpace(runTestGit(t, repository, "rev-parse", "HEAD"))

	t.Setenv("GIT_EXTERNAL_DIFF", "/bin/false")
	t.Setenv("GIT_DIR", "/nonexistent")
	first, err := collectChanges(context.Background(), repository, base, head)
	if err != nil {
		t.Fatal(err)
	}
	clone := filepath.Join(t.TempDir(), "different-checkout")
	runTestGit(t, repository, "clone", "-q", repository, clone)
	second, err := collectChanges(context.Background(), clone, base, head)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("checkout path changed frozen diff\nfirst: %#v\nsecond: %#v", first, second)
	}
	want := map[string]domain.ChangeKind{
		"add.txt": domain.ChangeAdd, "delete.txt": domain.ChangeDelete, "edit.txt": domain.ChangeModify,
		"mode.sh": domain.ChangeMode, "rename-new.txt": domain.ChangeRename, "submodule": domain.ChangeSubmodule,
	}
	for _, change := range first {
		if filepath.IsAbs(change.Path) || strings.Contains(change.Path, repository) {
			t.Fatalf("frozen path contains checkout state: %+v", change)
		}
		if want[change.Path] != change.Kind {
			t.Fatalf("change %s = %s, want %s", change.Path, change.Kind, want[change.Path])
		}
		delete(want, change.Path)
		if change.Kind == domain.ChangeRename && change.PreviousPath != "rename-old.txt" {
			t.Fatalf("rename = %+v", change)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing Git change kinds: %#v", want)
	}
}

func TestFrozenCollectorUsesBasePolicyAndValidatesHeadPolicy(t *testing.T) {
	repository, base, head, packet := admissionRepository(t)
	before := runTestGit(t, repository, "status", "--porcelain=v1")
	frozen, err := CollectFrozenRange(context.Background(), repository, base, head, packet)
	if err != nil {
		t.Fatal(err)
	}
	if frozen.BasePolicy.Rules[0].Classification != domain.MaterialityMaterial || frozen.HeadPolicy.Rules[0].Classification != domain.MaterialityNonMaterial {
		t.Fatalf("base/head policy fixture was not preserved")
	}
	candidates := testCandidates(base, head, packet.EvaluationTime.UTC())
	result, err := admission.Verify(admission.Input{Range: frozen, Packet: packet, Candidates: candidates})
	if err != nil {
		t.Fatal(err)
	}
	if result.Classification != domain.AdmissionValid || !reflect.DeepEqual(result.MaterialPaths, []string{"internal/app.go"}) {
		t.Fatalf("candidate head weakened its own evaluation: %+v", result)
	}
	if after := runTestGit(t, repository, "status", "--porcelain=v1"); after != before {
		t.Fatalf("read-only verifier changed repository state: before %q after %q", before, after)
	}
	first, firstCode, err := admission.CanonicalResult(result)
	if err != nil {
		t.Fatal(err)
	}

	clone := filepath.Join(t.TempDir(), "clone")
	runTestGit(t, repository, "clone", "-q", repository, clone)
	cloned, err := CollectFrozenRange(context.Background(), clone, base, head, packet)
	if err != nil {
		t.Fatal(err)
	}
	cloneResult, err := admission.Verify(admission.Input{Range: cloned, Packet: packet, Candidates: candidates})
	if err != nil {
		t.Fatal(err)
	}
	second, secondCode, err := admission.CanonicalResult(cloneResult)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) || firstCode != secondCode {
		t.Fatalf("clone changed admission\nfirst: %s\nsecond: %s", first, second)
	}
}

func TestFrozenCollectorReturnsCanonicalInvalidForMalformedHeadPolicy(t *testing.T) {
	repository, base, _, packet := admissionRepository(t)
	writeTestPath(t, repository, domain.DefaultProjectPolicyPath, "{\"schema\":\"broken\"}", 0o644)
	runTestGit(t, repository, "add", "-A")
	runTestGit(t, repository, "commit", "-qm", "malform candidate policy")
	packet.HeadRevision = strings.TrimSpace(runTestGit(t, repository, "rev-parse", "HEAD"))
	frozen, err := CollectFrozenRange(context.Background(), repository, base, packet.HeadRevision, packet)
	if err != nil {
		t.Fatal(err)
	}
	if !frozen.HeadGovernanceInvalid {
		t.Fatal("malformed candidate policy was not retained as invalid frozen evidence")
	}
	result, err := admission.Verify(admission.Input{Range: frozen, Packet: packet})
	if err != nil {
		t.Fatal(err)
	}
	if result.Classification != domain.AdmissionInvalid || result.Reasons[0].Code != domain.ReasonDeclarationInvalid {
		t.Fatalf("malformed head result = %+v", result)
	}
	if _, code, err := admission.CanonicalResult(result); err != nil || code != admission.ExitDenied {
		t.Fatalf("canonical invalid result code = %d, err = %v", code, err)
	}
}

func TestCTX10EveryDeclarationBoundArtifactIsVerified(t *testing.T) {
	for _, fixture := range []struct {
		name string
		path string
		raw  []byte
	}{
		{name: "bootstrap", path: domain.DefaultProjectBootstrapPath, raw: []byte("# Substituted bootstrap\n")},
		{name: "setup-profile", path: domain.DefaultProjectSetupProfilePath, raw: []byte("{\"schema\":\"goalrail.setup-profile/v1\"}")},
	} {
		t.Run("head-"+fixture.name, func(t *testing.T) {
			repository, base, _, packet := admissionRepository(t)
			writeTestBytes(t, repository, fixture.path, fixture.raw, 0o644)
			runTestGit(t, repository, "add", "-A")
			runTestGit(t, repository, "commit", "-qm", "drift "+fixture.name)
			packet.HeadRevision = strings.TrimSpace(runTestGit(t, repository, "rev-parse", "HEAD"))

			frozen, err := CollectFrozenRange(context.Background(), repository, base, packet.HeadRevision, packet)
			if err != nil {
				t.Fatal(err)
			}
			if !frozen.HeadGovernanceInvalid {
				t.Fatalf("drifted %s was not retained as invalid head governance", fixture.name)
			}
			result, err := admission.Verify(admission.Input{
				Range: frozen, Packet: packet,
				Candidates: testCandidates(base, packet.HeadRevision, packet.EvaluationTime.UTC()),
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.Classification != domain.AdmissionInvalid || result.Reasons[0].Code != domain.ReasonDeclarationInvalid {
				t.Fatalf("drifted %s result = %+v", fixture.name, result)
			}
		})
	}

	t.Run("base-setup-profile", func(t *testing.T) {
		repository, base, head, packet := admissionRepository(t)
		frozen, err := CollectFrozenRange(context.Background(), repository, base, head, packet)
		if err != nil {
			t.Fatal(err)
		}
		if frozen.BaseGovernanceInvalid || frozen.BaseDeclaration == nil {
			t.Fatal("intact base governance was reported invalid")
		}
		// The same drift in the base revision must deny rather than fall back to
		// head-defined authority.
		frozen.BaseGovernanceInvalid = true
		result, err := admission.Verify(admission.Input{
			Range: frozen, Packet: packet,
			Candidates: testCandidates(base, head, packet.EvaluationTime.UTC()),
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.Classification != domain.AdmissionInvalid || result.Reasons[0].Code != domain.ReasonDeclarationInvalid {
			t.Fatalf("drifted base governance result = %+v", result)
		}
	})
}

func TestCTX6NonBootstrapProjectIDChangeIsInvalid(t *testing.T) {
	repository, base, _, packet := admissionRepository(t)
	frozen, err := CollectFrozenRange(context.Background(), repository, base, packet.HeadRevision, packet)
	if err != nil {
		t.Fatal(err)
	}
	replacement := *frozen.HeadDeclaration
	replacement.ProjectID = "prj_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	replacementPolicy := *frozen.HeadPolicy
	replacementPolicy.ProjectID = replacement.ProjectID
	policyArtifact, err := domain.FreezeProjectPolicy(replacementPolicy)
	if err != nil {
		t.Fatal(err)
	}
	replacement.Policy.Digest = policyArtifact.Digest()
	declarationArtifact, err := domain.FreezeProjectDeclaration(replacement)
	if err != nil {
		t.Fatal(err)
	}
	frozen.HeadDeclaration = &replacement
	frozen.HeadDeclarationDigest = declarationArtifact.Digest()
	frozen.HeadPolicy = &replacementPolicy
	frozen.HeadPolicyDigest = policyArtifact.Digest()

	result, err := admission.Verify(admission.Input{
		Range: frozen, Packet: packet,
		Candidates: testCandidates(base, packet.HeadRevision, packet.EvaluationTime.UTC()),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Classification != domain.AdmissionInvalid || result.Reasons[0].Code != domain.ReasonDeclarationInvalid {
		t.Fatalf("head-minted project identity result = %+v", result)
	}
}

func TestCTX7OnlyConfirmedIntentSatisfiesLineage(t *testing.T) {
	repository, base, head, packet := admissionRepositoryWith(t, fixtureOptions{
		IntentStatus: "candidate", ReceiptStatus: localrun.StatePassed,
	})
	frozen, err := CollectFrozenRange(context.Background(), repository, base, head, packet)
	if err != nil {
		t.Fatal(err)
	}
	if !hasProjection(frozen.Projections, domain.LineageConfirmedIntent, "candidate", false) {
		t.Fatalf("candidate intent projection = %+v", frozen.Projections)
	}
	result, err := admission.Verify(admission.Input{
		Range: frozen, Packet: packet,
		Candidates: testCandidates(base, head, packet.EvaluationTime.UTC()),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Classification != domain.AdmissionMissing || result.Reasons[0].Code != domain.ReasonIntentUnconfirmed {
		t.Fatalf("candidate intent result = %+v", result)
	}
}

func TestCTX5OnlyPassedTerminalReceiptSatisfiesLineage(t *testing.T) {
	for _, status := range []localrun.RunState{
		localrun.StateFailed, localrun.StateUnlinked, localrun.StateLaunchFailed, localrun.StateVerificationIncomplete,
	} {
		t.Run(string(status), func(t *testing.T) {
			repository, base, head, packet := admissionRepositoryWith(t, fixtureOptions{
				IntentStatus: "confirmed", ReceiptStatus: status,
			})
			frozen, err := CollectFrozenRange(context.Background(), repository, base, head, packet)
			if err != nil {
				t.Fatal(err)
			}
			if !hasProjection(frozen.Projections, domain.LineageTerminalReceipt, string(status), false) {
				t.Fatalf("%s receipt projection = %+v", status, frozen.Projections)
			}
			result, err := admission.Verify(admission.Input{
				Range: frozen, Packet: packet,
				Candidates: testCandidates(base, head, packet.EvaluationTime.UTC()),
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.Classification != domain.AdmissionMissing || result.Reasons[0].Code != domain.ReasonReceiptMissing {
				t.Fatalf("%s receipt result = %+v", status, result)
			}
		})
	}
}

func hasProjection(projections []admission.SemanticProjection, relation domain.LineageRelation, state string, valid bool) bool {
	for _, projection := range projections {
		if projection.Relation == relation && projection.State == state && projection.Valid == valid {
			return true
		}
	}
	return false
}

// fixtureOptions varies only the semantic state of the exact committed
// artifacts, so a projection test changes what the bytes mean without changing
// how they are bound.
type fixtureOptions struct {
	IntentStatus          string
	ReceiptStatus         localrun.RunState
	IntentDeclaresContext bool
}

func admissionRepository(t *testing.T) (string, string, string, domain.AdmissionPacket) {
	t.Helper()
	return admissionRepositoryWith(t, fixtureOptions{IntentStatus: "confirmed", ReceiptStatus: localrun.StatePassed})
}

func admissionRepositoryWith(t *testing.T, options fixtureOptions) (string, string, string, domain.AdmissionPacket) {
	t.Helper()
	repository := t.TempDir()
	initTestRepository(t, repository)
	basePolicy := testPolicy(t)
	basePolicyArtifact, err := domain.FreezeProjectPolicy(basePolicy)
	if err != nil {
		t.Fatal(err)
	}
	bootstrap := []byte("# Goalrail bootstrap\n")
	setup := testSetupProfile(t, basePolicy.ProjectID)
	baseDeclaration := domain.ProjectDeclaration{
		Schema: domain.ProjectSchemaV1, ProjectID: basePolicy.ProjectID, ContractVersion: domain.GovernanceContractV1,
		Policy:       domain.CommittedArtifactReference{Schema: domain.PolicySchemaV1, Path: domain.DefaultProjectPolicyPath, Digest: basePolicyArtifact.Digest()},
		Bootstrap:    domain.CommittedArtifactReference{Schema: domain.BootstrapSchemaV1, Path: domain.DefaultProjectBootstrapPath, Digest: domain.DigestCanonicalJSON(bootstrap)},
		SetupProfile: domain.CommittedArtifactReference{Schema: domain.SetupProfileSchemaV1, Path: domain.DefaultProjectSetupProfilePath, Digest: domain.DigestCanonicalJSON(setup)},
	}
	baseDeclarationArtifact, err := domain.FreezeProjectDeclaration(baseDeclaration)
	if err != nil {
		t.Fatal(err)
	}
	writeTestBytes(t, repository, domain.DefaultProjectPolicyPath, basePolicyArtifact.CanonicalJSON(), 0o644)
	writeTestBytes(t, repository, domain.DefaultProjectBootstrapPath, bootstrap, 0o644)
	writeTestBytes(t, repository, domain.DefaultProjectSetupProfilePath, setup, 0o644)
	writeTestBytes(t, repository, domain.ProjectDeclarationPath, baseDeclarationArtifact.CanonicalJSON(), 0o644)
	writeTestBytes(t, repository, "openspec/changes/git-fixture/intent.md", testIntentArtifactWithContext("git-fixture", options.IntentStatus, options.IntentDeclaresContext), 0o644)
	writeTestBytes(t, repository, "openspec/changes/git-fixture/tasks.md", []byte("- [x] bounded change\n"), 0o644)
	runTestGit(t, repository, "add", "-A")
	runTestGit(t, repository, "commit", "-qm", "initialize governance")
	base := strings.TrimSpace(runTestGit(t, repository, "rev-parse", "HEAD"))

	writeTestPath(t, repository, "internal/app.go", "package internal\n", 0o644)
	runTestGit(t, repository, "add", "-A")
	runTestGit(t, repository, "commit", "-qm", "add material code")
	codeCommit := strings.TrimSpace(runTestGit(t, repository, "rev-parse", "HEAD"))

	headPolicy := basePolicy
	headPolicy.Rules = append([]domain.PolicyPathRule(nil), basePolicy.Rules...)
	headPolicy.Rules[0].Classification = domain.MaterialityNonMaterial
	headPolicyArtifact, err := domain.FreezeProjectPolicy(headPolicy)
	if err != nil {
		t.Fatal(err)
	}
	headDeclaration := baseDeclaration
	headDeclaration.Policy.Digest = headPolicyArtifact.Digest()
	headDeclarationArtifact, err := domain.FreezeProjectDeclaration(headDeclaration)
	if err != nil {
		t.Fatal(err)
	}
	writeTestBytes(t, repository, domain.DefaultProjectPolicyPath, headPolicyArtifact.CanonicalJSON(), 0o644)
	writeTestBytes(t, repository, domain.ProjectDeclarationPath, headDeclarationArtifact.CanonicalJSON(), 0o644)

	intentRaw := testIntentArtifactWithContext("git-fixture", options.IntentStatus, options.IntentDeclaresContext)
	writeTestBytes(t, repository, "openspec/changes/git-fixture/intent.md", intentRaw, 0o644)
	writeTestBytes(t, repository, "openspec/changes/git-fixture/tasks.md", []byte("- [x] bounded change\n"), 0o644)
	changeDigest, err := lineage.DigestChangeSnapshot(filepath.Join(repository, "openspec", "changes", "git-fixture"))
	if err != nil {
		t.Fatal(err)
	}
	intentRef := domain.ContentAddressedEvidenceReference{
		ArtifactKind: "intent", Identity: "intent:git-fixture", Version: "1", Digest: domain.DigestCanonicalJSON(intentRaw),
		SourceRef: "repo:root/openspec/changes/git-fixture/intent.md", AdapterID: "goalrail",
	}
	changeRef := domain.ContentAddressedEvidenceReference{
		ArtifactKind: "change", Identity: "change:git-fixture", Version: "1", Digest: changeDigest,
		SourceRef: "repo:root/openspec/changes/git-fixture", AdapterID: "goalrail",
	}
	unit := domain.WorkUnit{
		Schema: domain.WorkUnitSchemaV1, ID: "wu_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ProjectID: basePolicy.ProjectID,
		DeclarationDigest: baseDeclarationArtifact.Digest(), PolicyDigest: basePolicyArtifact.Digest(),
		IntentRef: intentRef, ChangeRef: changeRef, CreatedAt: time.Date(2026, 8, 5, 9, 0, 0, 0, time.UTC),
		Lifecycle: domain.WorkUnitOpen, RequiredRelations: lineage.DefaultRequirements(),
	}
	unitArtifact, err := domain.FreezeWorkUnit(unit)
	if err != nil {
		t.Fatal(err)
	}
	unitRef := domain.ContentAddressedEvidenceReference{
		ArtifactKind: "work_unit", Identity: "work-unit:" + string(unit.ID), Version: "1", Digest: unitArtifact.Digest(),
		SourceRef: "repo:root/.goalrail/work-units/" + string(unit.ID) + "/unit.json", AdapterID: "goalrail",
	}
	events := testEvents(t, unit, unitRef, baseDeclaration, baseDeclarationArtifact.Digest(), basePolicyArtifact.Digest(), codeCommit)
	replicas := make(map[string][]byte)
	for eventIndex := range events {
		event := &events[eventIndex]
		if event.Relation == domain.LineageProjectPolicy {
			event.Targets[0].SourceRef = "git:" + base + "/.goalrail/project.json"
			event.Targets[1].SourceRef = "git:" + base + "/.goalrail/policy.json"
			refreezeEvent(t, event)
			continue
		}
		switch event.Relation {
		case domain.LineageWorkSpec, domain.LineageRunSession, domain.LineagePullRequest,
			domain.LineageReviewIndex, domain.LineageCheckSet, domain.LineageTerminalReceipt, domain.LineageOwnerDecision:
			var raw []byte
			var marshalErr error
			if event.Relation == domain.LineageTerminalReceipt {
				raw = testTerminalReceipt(t, options.ReceiptStatus, intentRef)
			} else {
				raw, marshalErr = json.Marshal(map[string]any{
					"identity": event.Targets[0].Identity,
					"schema":   "goalrail.fixture-evidence/v1",
				})
			}
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			digest := domain.DigestCanonicalJSON(raw)
			event.Targets[0].Digest = digest
			event.Targets[0].SourceRef = "repo:root/.goalrail/evidence/sha256/" + strings.TrimPrefix(string(digest), "sha256:")
			replicas[strings.TrimPrefix(event.Targets[0].SourceRef, "repo:root/")] = raw
			refreezeEvent(t, event)
		}
	}
	writeTestBytes(t, repository, ".goalrail/work-units/"+string(unit.ID)+"/unit.json", unitArtifact.CanonicalJSON(), 0o644)
	for _, event := range events {
		artifact, freezeErr := domain.FreezeLineageEvent(event)
		if freezeErr != nil {
			t.Fatal(freezeErr)
		}
		name := strings.TrimPrefix(string(event.SemanticDigest), "sha256:") + ".json"
		writeTestBytes(t, repository, ".goalrail/work-units/"+string(unit.ID)+"/events/"+name, artifact.CanonicalJSON(), 0o644)
	}
	for relative, raw := range replicas {
		writeTestBytes(t, repository, relative, raw, 0o644)
	}
	runTestGit(t, repository, "add", "-A")
	runTestGit(t, repository, "commit", "-qm", "attach frozen lineage evidence\n\nGoalrail-Work-Unit: "+string(unit.ID))
	head := strings.TrimSpace(runTestGit(t, repository, "rev-parse", "HEAD"))
	evaluationTime := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	packet := domain.AdmissionPacket{
		Schema: domain.AdmissionPacketSchemaV1, ProjectID: basePolicy.ProjectID,
		DeclarationDigest: baseDeclarationArtifact.Digest(), PolicyDigest: basePolicyArtifact.Digest(),
		BaseRevision: base, HeadRevision: head, WorkUnitRef: unitRef,
		EvaluationTime: &evaluationTime, TimeAuthorityRef: "provider:trusted-evaluation-time",
		Evidence: []domain.ContentAddressedEvidenceReference{}, Provenance: []domain.AdmissionProviderProvenance{},
	}
	return repository, base, head, packet
}

func TestCollectPacketSeedUsesTrailerOnlyAsAWorkUnitIndex(t *testing.T) {
	repository, base, head, want := admissionRepository(t)
	seed, err := CollectPacketSeed(context.Background(), repository, base, head)
	if err != nil {
		t.Fatal(err)
	}
	if seed.ProjectID != want.ProjectID || seed.DeclarationDigest != want.DeclarationDigest ||
		seed.PolicyDigest != want.PolicyDigest || seed.WorkUnitRef != want.WorkUnitRef {
		t.Fatalf("packet seed differs from frozen repository authority\nseed: %+v\nwant: %+v", seed, want)
	}
	if seed.EvaluationTime != nil || len(seed.Evidence) != 0 || len(seed.Provenance) != 0 {
		t.Fatalf("repository seed invented provider evidence: %+v", seed)
	}
}

func initTestRepository(t *testing.T, repository string) {
	t.Helper()
	runTestGit(t, repository, "init", "-q")
	runTestGit(t, repository, "config", "user.name", "Goalrail Test")
	runTestGit(t, repository, "config", "user.email", "goalrail-test@example.invalid")
	runTestGit(t, repository, "config", "core.fileMode", "true")
}

func runTestGit(t *testing.T, repository string, arguments ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repository}, arguments...)...)
	command.Env = []string{
		"PATH=" + os.Getenv("PATH"), "HOME=" + repository, "LC_ALL=C",
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0",
	}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(arguments, " "), err, output)
	}
	return string(output)
}

func writeTestPath(t *testing.T, repository, relative, content string, mode os.FileMode) {
	t.Helper()
	writeTestBytes(t, repository, relative, []byte(content), mode)
}

func writeTestBytes(t *testing.T, repository, relative string, content []byte, mode os.FileMode) {
	t.Helper()
	absolute := filepath.Join(repository, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absolute, content, mode); err != nil {
		t.Fatal(err)
	}
}

func testSetupProfile(t *testing.T, project domain.ProjectID) []byte {
	t.Helper()
	profile := domain.SetupProfile{
		Schema: domain.SetupProfileSchemaV1, ProjectID: project,
		CompatibleGoalrailBundle: ">=0.2.0 <0.3.0",
		Planning: domain.SetupPlanningAdapter{
			Adapter: domain.SetupAdapterPin{
				ID: "openspec", Version: "1.6.0", SourceRef: "npm:fission-ai/openspec@1.6.0",
				Integrity: domain.DigestCanonicalJSON([]byte("planning")),
			},
			Runtime: "node", RuntimeVersion: "22.18.0", Compiler: "openspec", CompilerVersion: "1.6.0",
		},
		ScaffoldAdapters: []domain.SetupAdapterPin{{
			ID: "codex", Version: "1.0.0", SourceRef: "bundle:codex-v1",
			Integrity: domain.DigestCanonicalJSON([]byte("codex")),
		}},
		SharedAdmissionAdapter: &domain.SetupAdapterPin{
			ID: "github-actions", Version: "1.0.0", SourceRef: "release:github-actions-v1",
			Integrity: domain.DigestCanonicalJSON([]byte("github-actions")),
		},
	}
	artifact, err := domain.FreezeSetupProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	return artifact.CanonicalJSON()
}

// testIntentArtifact writes the exact artifact the intent adapter parses. A
// placeholder heading is no longer enough: the collector must be able to reach
// a confirmed snapshot from these bytes.
func testIntentArtifact(id, status string) []byte {
	return testIntentArtifactWithContext(id, status, false)
}

func testIntentArtifactWithContext(id, status string, declaresContext bool) []byte {
	contextDeclaration := ""
	if declaresContext {
		contextDeclaration = "\n- **Context Pack:** " + id + " version 1"
	}
	confirmation := "- **Confirmed by:** pending\n- **Confirmed at:** pending\n- **Verification action:** pending"
	if status == "confirmed" {
		confirmation = "- **Confirmed by:** owner\n- **Confirmed at:** 2026-08-05\n- **Verification action:** owner-reviewed-three-groups"
	}
	return []byte(`# Intent Snapshot

- **Intent ID:** ` + id + `
- **Version:** 1` + contextDeclaration + `
- **Status:** ` + status + `
- **Owner:** owner

## Source Evidence

- **SE-1 — Owner statement:** The owner asked for a bounded result.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Produce the bounded result. | Inspect it. | SE-1 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | Do not publish. | SE-1 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | The result is inspectable. | One local artifact exists. | SE-1 |

## Ambiguities and Unknowns

None.

## Confirmation

` + confirmation + `
`)
}

func testTerminalReceipt(t *testing.T, status localrun.RunState, intent domain.ContentAddressedEvidenceReference) []byte {
	t.Helper()
	moment := time.Date(2026, 8, 5, 11, 0, 0, 0, time.UTC)
	receipt := localrun.TerminalReceipt{
		Schema: localrun.TerminalReceiptSchemaV1, WorkSpecID: "ws-git-fixture", WorkSpecVersion: 1,
		WorkSpecDigest: domain.WorkSpecDigest(domain.DigestCanonicalJSON([]byte("work-spec"))),
		Intent: &localrun.ReceiptIntentReference{
			ID: domain.IntentID(strings.TrimPrefix(intent.Identity, "intent:")), Version: 1,
			Digest: string(intent.Digest),
		},
		RunID: "run-git-fixture", Adapter: "codex", AdapterVersion: "1.0.0",
		BaseRevision: strings.Repeat("a", 40), TerminalHead: strings.Repeat("b", 40),
		EffectivePaths: []string{"internal/app.go"}, Checks: []domain.WorkSpecCheck{}, CheckResults: []localrun.CheckResult{},
		Status: status, PreparedAt: moment, LaunchAttemptedAt: moment.Add(time.Minute),
		ProviderObservedAt: moment.Add(2 * time.Minute), TerminalAt: moment.Add(3 * time.Minute),
	}
	raw, err := localrun.CanonicalTerminalReceipt(receipt)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// testCandidates is the same-invocation provider observation a shared check
// supplies. No repository content can produce it.
func testCandidates(base, head string, evaluationTime time.Time) []admission.ProviderCandidate {
	observedAt := evaluationTime.Add(-time.Minute)
	candidates := []admission.ProviderCandidate{}
	for _, relation := range []domain.LineageRelation{
		domain.LineagePullRequest, domain.LineageReviewIndex, domain.LineageCheckSet,
	} {
		candidates = append(candidates, admission.ProviderCandidate{
			Relation: relation, Identity: "github:fixture/" + string(relation),
			Digest: domain.DigestCanonicalJSON([]byte(relation)), AdapterID: "github-actions",
			ProviderRef: "github:fixture/" + string(relation), BaseRevision: base, HeadRevision: head,
			ObservedAt: observedAt, Authenticated: true,
		})
	}
	return append(candidates, admission.ProviderCandidate{
		Relation: domain.LineageOwnerDecision, Identity: "github:fixture/owner-decision",
		Digest: domain.DigestCanonicalJSON([]byte("owner-decision")), AdapterID: "github-actions",
		ProviderRef: "github:fixture/owner-decision", BaseRevision: base, HeadRevision: head,
		ActorRef: "github-user:owner", AuthorityRef: "user:owner", Outcome: domain.OwnerDecisionAllow,
		ObservedAt: observedAt, Authenticated: true,
	})
}

func testPolicy(t *testing.T) domain.ProjectPolicy {
	t.Helper()
	return domain.ProjectPolicy{
		Schema: domain.PolicySchemaV1, ProjectID: "prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Version: 1,
		Rules: []domain.PolicyPathRule{
			{
				ID: "material-source", Matcher: domain.PolicyPathMatcher{Kind: domain.PathMatcherPrefix, Path: "internal"},
				ChangeKinds: []domain.ChangeKind{domain.ChangeAdd, domain.ChangeModify, domain.ChangeDelete, domain.ChangeRename, domain.ChangeMode, domain.ChangeSubmodule},
				Priority:    200, Classification: domain.MaterialityMaterial,
			},
			{
				ID: "goalrail-evidence", Matcher: domain.PolicyPathMatcher{Kind: domain.PathMatcherPrefix, Path: ".goalrail"},
				ChangeKinds: []domain.ChangeKind{domain.ChangeAdd, domain.ChangeModify, domain.ChangeDelete, domain.ChangeRename, domain.ChangeMode, domain.ChangeSubmodule},
				Priority:    100, Classification: domain.MaterialityEvidence,
			},
		},
		ExceptionAuthorities: []domain.PolicyExceptionAuthority{{
			ID: "owner-break-glass", Class: domain.ExceptionBreakGlass,
			ActorRefs: []string{"user:owner"}, PathPrefixes: []string{"internal"}, EffectScopes: []string{"material_change"},
			MaxDurationSeconds: 3600, OwnerDecisionRequired: true,
		}},
		OwnerDecision: domain.PolicyOwnerDecision{
			Required: true, AuthorityRefs: []string{"user:owner"},
			Outcomes: []domain.OwnerDecisionOutcome{domain.OwnerDecisionAllow, domain.OwnerDecisionReject},
		},
	}
}

func testEvents(t *testing.T, unit domain.WorkUnit, unitRef domain.ContentAddressedEvidenceReference, declaration domain.ProjectDeclaration, declarationDigest, policyDigest domain.SHA256Digest, commit string) []domain.LineageEvent {
	t.Helper()
	var events []domain.LineageEvent
	for _, requirement := range lineage.DefaultRequirements() {
		if requirement.Relation == domain.LineageClosure {
			continue
		}
		var targets []domain.ContentAddressedEvidenceReference
		switch requirement.Relation {
		case domain.LineageProjectPolicy:
			targets = []domain.ContentAddressedEvidenceReference{
				{ArtifactKind: "project_declaration", Identity: "project:" + string(declaration.ProjectID), Version: "1", Digest: declarationDigest, SourceRef: "repo:root/.goalrail/project.json", AdapterID: "goalrail"},
				{ArtifactKind: "project_policy", Identity: "policy:" + string(declaration.ProjectID), Version: "1", Digest: policyDigest, SourceRef: "repo:root/.goalrail/policy.json", AdapterID: "goalrail"},
			}
		case domain.LineageConfirmedIntent:
			targets = []domain.ContentAddressedEvidenceReference{unit.IntentRef}
		case domain.LineageChange:
			targets = []domain.ContentAddressedEvidenceReference{unit.ChangeRef}
		case domain.LineageCommit:
			targets = []domain.ContentAddressedEvidenceReference{{
				ArtifactKind: "commit", Identity: "git-commit:" + commit, Version: "1",
				Digest: domain.DigestCanonicalJSON([]byte(commit)), SourceRef: "git:" + commit, AdapterID: "git",
			}}
		default:
			kind := relationArtifactKind(requirement.Relation)
			targets = []domain.ContentAddressedEvidenceReference{{
				ArtifactKind: kind, Identity: kind + ":fixture", Version: "1",
				Digest: domain.DigestCanonicalJSON([]byte(kind + ":fixture")), SourceRef: "repo:root/.goalrail/index/" + kind + ".json", AdapterID: "goalrail",
			}}
		}
		event := domain.LineageEvent{
			Schema: domain.LineageEventSchemaV1, WorkUnitID: unit.ID, Relation: requirement.Relation, Cardinality: requirement.Cardinality,
			Sources: []domain.ContentAddressedEvidenceReference{unitRef}, Targets: targets,
			ActorRef: "user:owner", AdapterID: "goalrail", ObservedAt: time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC),
		}
		refreezeEvent(t, &event)
		events = append(events, event)
	}
	return events
}

func relationArtifactKind(relation domain.LineageRelation) string {
	switch relation {
	case domain.LineageWorkSpec:
		return "work_spec"
	case domain.LineageRunSession:
		return "run_session"
	case domain.LineagePullRequest:
		return "pull_request"
	case domain.LineageReviewIndex:
		return "review_index"
	case domain.LineageCheckSet:
		return "check_set"
	case domain.LineageTerminalReceipt:
		return "terminal_receipt"
	case domain.LineageOwnerDecision:
		return "owner_decision"
	default:
		return string(relation)
	}
}

func refreezeEvent(t *testing.T, event *domain.LineageEvent) {
	t.Helper()
	event.SemanticDigest = ""
	artifact, err := domain.FreezeLineageEvent(*event)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := domain.DecodeLineageEvent(bytes.NewReader(artifact.CanonicalJSON()))
	if err != nil {
		t.Fatal(err)
	}
	*event = decoded
}

func TestFullGitOIDRejectsRefsAndAbbreviations(t *testing.T) {
	values := []string{"main", "HEAD", "abc1234", strings.Repeat("A", 40), strings.Repeat("a", 39), strings.Repeat("g", 40)}
	for _, value := range values {
		if fullGitOID(value) {
			t.Fatalf("accepted mutable or malformed revision %q", value)
		}
	}
	valid := []string{strings.Repeat("a", 40), strings.Repeat("b", 64)}
	sort.Strings(valid)
	for _, value := range valid {
		if !fullGitOID(value) {
			t.Fatalf("rejected full object ID %q", value)
		}
	}
}

func TestPortableReplicaRejectsProhibitedBodiesAndSecrets(t *testing.T) {
	for _, raw := range [][]byte{
		[]byte(`{"body":"raw review text","schema":"goalrail.fixture/v1"}`),
		[]byte(`{"schema":"goalrail.fixture/v1","token":"ghp_abcdefghijklmnopqrstuvwxyz123456"}`),
	} {
		if err := validatePortableReplica(raw); err == nil {
			t.Fatalf("prohibited replica was accepted: %s", raw)
		}
	}
}

func TestPortableReplicaAcceptsKnownCanonicalArtifact(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "domain", "testdata", "contracts-v1", "work-unit.json"))
	if err != nil {
		t.Fatal(err)
	}
	unit, err := domain.DecodeWorkUnit(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := domain.FreezeWorkUnit(unit)
	if err != nil {
		t.Fatal(err)
	}
	if err := validatePortableReplica(artifact.CanonicalJSON()); err != nil {
		t.Fatalf("known canonical artifact was rejected: %v", err)
	}
}

// An intent that names its context pack is only readable as a pair. Deleting
// the context at head must not turn it into a readable legacy snapshot, or
// admission would pass because required evidence was removed.
func TestAnIntentNamingItsContextPackNeedsThatContext(t *testing.T) {
	repository, base, head, packet := admissionRepositoryWith(t, fixtureOptions{
		IntentStatus: "confirmed", ReceiptStatus: localrun.StatePassed, IntentDeclaresContext: true,
	})
	frozen, err := CollectFrozenRange(context.Background(), repository, base, head, packet)
	if err != nil {
		t.Fatal(err)
	}
	if !hasProjection(frozen.Projections, domain.LineageConfirmedIntent, admission.ProjectionStateInvalid, false) {
		t.Fatalf("an intent missing its declared context still projected: %+v", frozen.Projections)
	}
	result, err := admission.Verify(admission.Input{
		Range: frozen, Packet: packet,
		Candidates: testCandidates(base, head, packet.EvaluationTime.UTC()),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Classification != domain.AdmissionMissing || result.Reasons[0].Code != domain.ReasonIntentUnconfirmed {
		t.Fatalf("intent without its declared context = %+v", result)
	}
}

func TestAContextPackDeclarationIsReadFromThePreambleOnly(t *testing.T) {
	for _, fixture := range []struct {
		name     string
		raw      string
		declares bool
	}{
		{name: "declared", raw: "# Intent\n- **Context Pack:** pack-1 version 1\n\n## Body\n", declares: true},
		{name: "pending", raw: "# Intent\n- **Context Pack:** pending\n\n## Body\n"},
		{name: "absent", raw: "# Intent\n- **Version:** 1\n\n## Body\n"},
		{name: "mentioned in the body only", raw: "# Intent\n- **Version:** 1\n\n## Body\n- **Context Pack:** pack-1 version 1\n"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			if got := declaresContextPack([]byte(fixture.raw)); got != fixture.declares {
				t.Fatalf("declaresContextPack = %v, want %v", got, fixture.declares)
			}
		})
	}
}
