package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	openspecadapter "github.com/heurema/goalrail/internal/adapters/openspec"
	"github.com/heurema/goalrail/internal/admission"
	"github.com/heurema/goalrail/internal/admissiongit"
	"github.com/heurema/goalrail/internal/ambient"
	projectdoctor "github.com/heurema/goalrail/internal/doctor"
	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/githubadmission"
	"github.com/heurema/goalrail/internal/harness"
	lineagestore "github.com/heurema/goalrail/internal/lineage"
	"github.com/heurema/goalrail/internal/localrun"
	projectstate "github.com/heurema/goalrail/internal/project"
)

// TestProjectLineageAdmissionScratchCanary is the single no-provider dogfood
// path for the v0.2 project boundary. Every repository, home, state directory,
// clone, worktree, provider observation, and shared-check invocation is local
// to the test. Git commits are immutable fixture inputs only.
func TestProjectLineageAdmissionScratchCanary(t *testing.T) {
	ctx := context.Background()
	primary := scratchRepository(t)
	stdout, _, err := runCommand(t, "init", "--repo", primary, "--scaffold", "codex")
	if err != nil {
		t.Fatal(err)
	}
	var initialized projectstate.InitializeReport
	if err := json.Unmarshal([]byte(stdout), &initialized); err != nil {
		t.Fatal(err)
	}
	if !initialized.Managed || initialized.ProjectID == "" || !initialized.CommitRequired {
		t.Fatalf("initialization report = %+v", initialized)
	}
	installCanaryPolicy(t, primary)
	copyCanaryChange(t, primary)
	gitCommand(t, primary, "add", "-A")
	gitCommand(t, primary, "commit", "-m", "initialize managed canary fixture")
	base := strings.TrimSpace(gitOutput(t, primary, "rev-parse", "HEAD"))

	cloneOne := canaryClone(t, primary, "clone-one")
	cloneTwo := canaryClone(t, primary, "clone-two")
	var worktrees []string
	for index := 1; index <= 3; index++ {
		path := filepath.Join(t.TempDir(), "worktree-"+strconv.Itoa(index))
		gitCommand(t, primary, "worktree", "add", "--detach", path, "HEAD")
		worktrees = append(worktrees, path)
	}
	for _, repository := range append([]string{primary, cloneOne, cloneTwo}, worktrees...) {
		inspection, err := projectstate.Inspect(ctx, repository)
		if err != nil {
			t.Fatal(err)
		}
		if inspection.State != projectstate.ClaimManaged || inspection.Declaration.ProjectID != initialized.ProjectID {
			t.Fatalf("portable project identity at %s = %+v", repository, inspection)
		}
		if _, err := os.Lstat(filepath.Join(repository, ".goalrail", "ambient.json")); !os.IsNotExist(err) {
			t.Fatalf("clone/worktree required a legacy marker at %s: %v", repository, err)
		}
	}

	diagnosis, err := projectdoctor.Diagnose(ctx, projectdoctor.DiagnoseInput{
		RepositoryRoot: cloneTwo,
		Home:           t.TempDir(),
		Scaffolds:      []ambient.Scaffold{ambient.ScaffoldCodex},
		LatestRelease: func(context.Context) (string, time.Time, error) {
			return harness.Version, canaryTime(), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !diagnosis.Managed || !diagnosis.SetupRequired || diagnosis.Working || diagnosis.Category != projectdoctor.CategorySetupRequired {
		t.Fatalf("clean-clone diagnosis = %+v", diagnosis)
	}

	beforePlan := snapshotCanaryTree(t, cloneTwo)
	var planOutput bytes.Buffer
	if err := runSetup(ctx, []string{
		"plan", "--repo", cloneTwo, "--home", t.TempDir(), "--scaffold", "codex",
	}, &planOutput, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	plan, err := domain.DecodeSetupPlan(bytes.NewReader(planOutput.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if plan.ProjectID != initialized.ProjectID || plan.ProjectCodeWrites != 0 || plan.State != domain.SetupPlanIncomplete {
		t.Fatalf("clean-machine setup simulation = %+v", plan)
	}
	if afterPlan := snapshotCanaryTree(t, cloneTwo); !reflect.DeepEqual(beforePlan, afterPlan) {
		t.Fatal("read-only setup planning changed the clean clone")
	}

	beginReceipt, err := lineagestore.Begin(ctx, lineagestore.BeginOptions{
		Repository: cloneOne,
		ChangeID:   "project-lineage-admission-v0",
		ActorRef:   "role:repository-owner",
		Now:        canaryTime,
		NewWorkUnitID: func() (domain.WorkUnitID, error) {
			return "wu_11111111111111111111111111111111", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	lineage, err := lineagestore.NewStore(cloneOne)
	if err != nil {
		t.Fatal(err)
	}
	unit, _, err := lineage.LoadWorkUnit(beginReceipt.WorkUnitID)
	if err != nil {
		t.Fatal(err)
	}
	inspection, err := projectstate.Inspect(ctx, cloneOne)
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := openspecadapter.LoadChange(filepath.Join(cloneOne, "openspec", "changes", "project-lineage-admission-v0"))
	if err != nil {
		t.Fatal(err)
	}
	workSpec := domain.WorkSpec{
		Schema:  domain.WorkSpecSchemaV1,
		ID:      "work-spec-scratch-canary",
		Version: 1,
		Project: &domain.WorkSpecProjectReference{
			ID: initialized.ProjectID, ArtifactRef: domain.ProjectDeclarationPath, Digest: inspection.DeclarationDigest,
		},
		Policy: &domain.WorkSpecPolicyReference{
			ArtifactRef: inspection.Declaration.Policy.Path, Digest: inspection.Declaration.Policy.Digest,
		},
		Change: &domain.WorkSpecChangeReference{
			ID: "project-lineage-admission-v0", ArtifactRef: "openspec/changes/project-lineage-admission-v0",
			Digest: unit.ChangeRef.Digest, IntentID: compiled.Intent.ID, IntentVersion: compiled.Intent.Version,
			IntentDigest: unit.IntentRef.Digest,
		},
		WorkUnit: &domain.WorkSpecWorkUnitReference{
			ID: beginReceipt.WorkUnitID, ArtifactRef: beginReceipt.AnchorRef, Digest: beginReceipt.AnchorDigest,
		},
		Repository: domain.WorkSpecRepository{Root: inspection.WorktreeRoot, BaseRevision: base},
		Intent: domain.WorkSpecIntentReference{
			ID: compiled.Intent.ID, Version: compiled.Intent.Version,
			ArtifactRef: strings.TrimPrefix(unit.IntentRef.SourceRef, "repo:root/"), Digest: string(unit.IntentRef.Digest),
		},
		Task:   "Create one bounded scratch feature through the fixture adapter.",
		Paths:  []string{"internal/app.go"},
		Checks: []domain.WorkSpecCheck{{ID: "canary-check", Argv: []string{"fixture", "check"}}},
		StopConditions: []domain.WorkSpecStopCondition{{
			ID: "scope-drift", Description: "Stop if any path outside internal/app.go changes.",
		}},
		Posture: domain.PostureTrustedLocalProviderEnforcedV1,
	}
	frozenWorkSpec, err := domain.FreezeWorkSpec(workSpec)
	if err != nil {
		t.Fatal(err)
	}
	localState, err := localrun.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service, err := localrun.NewFixtureService(
		localState,
		localrun.GitObserver{},
		openspecadapter.IntentResolver{},
		localrun.FixtureAdapter{Result: localrun.ProviderObservation{
			Outcome: localrun.ProviderCompleted, IdentityStatus: localrun.IdentityVerified,
			RootSessionRef: "session-scratch-canary",
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := service.Prepare(ctx, bytes.NewReader(frozenWorkSpec.CanonicalJSON()))
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cloneOne, "internal", "app.go"), "package internal\n")
	started, err := service.Start(ctx, localrun.StartInput{WorkSpecDigest: prepared.WorkSpec.Digest(), Adapter: "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	if started.State != localrun.StateAwaitingVerification {
		t.Fatalf("fixture start = %+v", started)
	}
	terminalReceipt, err := service.Finish(ctx, localrun.FinishInput{
		RunID: started.Claim.RunID,
		Results: []localrun.CheckResult{{
			ID: "canary-check", State: domain.CheckResultPass,
			EvidenceRef: "local:scratch-canary-check", EvidenceDigest: string(domain.DigestCanonicalJSON([]byte("pass"))),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if terminalReceipt.Status != localrun.StatePassed || terminalReceipt.RootSessionRef != "session-scratch-canary" {
		t.Fatalf("fixture terminal receipt = %+v", terminalReceipt)
	}

	gitCommand(t, cloneOne, "add", "internal/app.go")
	gitCommand(t, cloneOne, "commit", "-m", "feat: add scratch canary code\n\nGoalrail-Work-Unit: "+string(beginReceipt.WorkUnitID))
	codeCommit := strings.TrimSpace(gitOutput(t, cloneOne, "rev-parse", "HEAD"))
	unitReference := domain.ContentAddressedEvidenceReference{
		ArtifactKind: "work_unit", Identity: "work-unit:" + string(beginReceipt.WorkUnitID), Version: "1",
		Digest: beginReceipt.AnchorDigest, SourceRef: "repo:root/" + beginReceipt.AnchorRef, AdapterID: "goalrail",
	}
	observedAt := canaryTime().Add(time.Minute)
	attachCanaryReplica(t, cloneOne, unitReference, domain.LineageWorkSpec, domain.RelationSingle,
		"work_spec", "work-spec:"+string(workSpec.ID), frozenWorkSpec.CanonicalJSON(), domain.WorkSpecSchemaV1,
		"role:repository-owner", observedAt)
	attachCanaryReplica(t, cloneOne, unitReference, domain.LineageRunSession, domain.RelationSingle,
		"run_session", "run:"+string(started.Claim.RunID), canaryFixtureEvidence(t, "run:"+string(started.Claim.RunID)), canaryFixtureSchema,
		"role:repository-owner", observedAt.Add(time.Minute))
	attachCanaryEvent(t, cloneOne, domain.LineageEvent{
		Schema: domain.LineageEventSchemaV1, WorkUnitID: beginReceipt.WorkUnitID,
		Relation: domain.LineageCommit, Cardinality: domain.RelationSet,
		Sources: []domain.ContentAddressedEvidenceReference{unitReference},
		Targets: []domain.ContentAddressedEvidenceReference{{
			ArtifactKind: "commit", Identity: "git-commit:" + codeCommit, Version: "1",
			Digest: domain.DigestCanonicalJSON([]byte(codeCommit)), SourceRef: "git:" + codeCommit, AdapterID: "git",
		}},
		ActorRef: "role:repository-owner", AdapterID: "git", ObservedAt: observedAt.Add(2 * time.Minute),
	}, nil)
	for index, relation := range []struct {
		relation domain.LineageRelation
		kind     string
		identity string
	}{
		{domain.LineagePullRequest, "pull_request", "pull-request:fixture-local"},
		{domain.LineageReviewIndex, "review_index", "review-index:fixture-local"},
		{domain.LineageCheckSet, "check_set", "check-set:fixture-local"},
	} {
		attachCanaryReplica(t, cloneOne, unitReference, relation.relation, domain.RelationSingle,
			relation.kind, relation.identity, canaryFixtureEvidence(t, relation.identity), canaryFixtureSchema,
			"role:repository-owner", observedAt.Add(time.Duration(index+3)*time.Minute))
	}
	canonicalReceipt, err := localrun.CanonicalTerminalReceipt(terminalReceipt)
	if err != nil {
		t.Fatal(err)
	}
	attachCanaryReplica(t, cloneOne, unitReference, domain.LineageTerminalReceipt, domain.RelationSingle,
		"terminal_receipt", "terminal-receipt:"+string(started.Claim.RunID), canonicalReceipt, localrun.TerminalReceiptSchemaV1,
		"role:repository-owner", observedAt.Add(6*time.Minute))
	attachCanaryReplica(t, cloneOne, unitReference, domain.LineageOwnerDecision, domain.RelationSingle,
		"owner_decision", "owner-decision:fixture-allow", canaryFixtureEvidence(t, "owner-decision:fixture-allow"), canaryFixtureSchema,
		"role:repository-owner", observedAt.Add(7*time.Minute))

	gitCommand(t, cloneOne, "add", ".goalrail/work-units", ".goalrail/evidence")
	gitCommand(t, cloneOne, "commit", "-m", "chore: attach scratch canary evidence\n\nGoalrail-Work-Unit: "+string(beginReceipt.WorkUnitID))
	validHead := strings.TrimSpace(gitOutput(t, cloneOne, "rev-parse", "HEAD"))
	localValid := verifyCanaryRange(t, cloneOne, base, validHead, nil)
	sharedClone := canaryClone(t, cloneOne, "shared-check")
	sharedValid := verifyCanaryRange(t, sharedClone, base, validHead, nil)
	if !bytes.Equal(localValid, sharedValid) {
		t.Fatalf("local/shared canonical verdicts differ\nlocal:  %s\nshared: %s", localValid, sharedValid)
	}
	// Committed lineage alone is advisory: the owner boundary needs a live
	// authenticated observation that no repository content can forge.
	assertCanaryAdmission(t, localValid, domain.AdmissionMissing, domain.AdmissionDeny, domain.ReasonOwnerDecisionMissing)

	providerObservedAt := canaryTime().Add(10 * time.Minute)
	localProvider := verifyCanaryRangeWithProvider(t, cloneOne, base, validHead, providerObservedAt)
	sharedProvider := verifyCanaryRangeWithProvider(t, sharedClone, base, validHead, providerObservedAt)
	if !bytes.Equal(localProvider, sharedProvider) {
		t.Fatalf("local/shared provider verdicts differ\nlocal:  %s\nshared: %s", localProvider, sharedProvider)
	}
	assertCanaryAdmission(t, localProvider, domain.AdmissionValid, domain.AdmissionAllow, "")

	evaluationTime := canaryTime().Add(30 * time.Minute)
	exceptionRaw, err := json.Marshal(map[string]any{
		"schema": lineagestore.LineageExceptionSchemaV1, "authority_id": "owner-break-glass",
		"class": domain.ExceptionBreakGlass, "actor_ref": "role:repository-owner",
		"path_prefixes": []string{"internal"}, "effect_scopes": []string{"material_change"},
		"issued_at": canaryTime().Add(20 * time.Minute), "expires_at": canaryTime().Add(50 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	attachCanaryReplica(t, cloneOne, unitReference, domain.LineageException, domain.RelationSet,
		"exception", "exception:owner-break-glass", exceptionRaw, lineagestore.LineageExceptionSchemaV1,
		"role:repository-owner", canaryTime().Add(20*time.Minute))
	gitCommand(t, cloneOne, "add", ".goalrail/work-units", ".goalrail/evidence")
	gitCommand(t, cloneOne, "commit", "-m", "chore: expose scratch canary exception\n\nGoalrail-Work-Unit: "+string(beginReceipt.WorkUnitID))
	exceptionHead := strings.TrimSpace(gitOutput(t, cloneOne, "rev-parse", "HEAD"))
	enrichException := func(packet *domain.AdmissionPacket) {
		packet.EvaluationTime = &evaluationTime
		packet.TimeAuthorityRef = "fixture:trusted-evaluation-time"
		packet.Provenance = []domain.AdmissionProviderProvenance{{
			AdapterID: "fixture", ProviderRef: packet.TimeAuthorityRef,
			EvidenceDigest: domain.DigestCanonicalJSON([]byte("trusted-time")),
			ObservedAt:     evaluationTime, Authenticated: true,
		}}
	}
	localException := verifyCanaryRange(t, cloneOne, base, exceptionHead, enrichException)
	exceptionSharedClone := canaryClone(t, cloneOne, "shared-exception-check")
	sharedException := verifyCanaryRange(t, exceptionSharedClone, base, exceptionHead, enrichException)
	if !bytes.Equal(localException, sharedException) {
		t.Fatalf("local/shared exception verdicts differ\nlocal:  %s\nshared: %s", localException, sharedException)
	}
	assertCanaryAdmission(t, localException, domain.AdmissionMissing, domain.AdmissionDeny, domain.ReasonOwnerDecisionMissing)

	providerException := verifyCanaryRangeWithProvider(t, cloneOne, base, exceptionHead, evaluationTime)
	assertCanaryAdmission(t, providerException, domain.AdmissionBreakGlass, domain.AdmissionAllow, domain.ReasonExceptionApplied)

	legacy := scratchRepository(t)
	installCanaryLegacyV018(t, legacy)
	writeFile(t, filepath.Join(legacy, ".goalrail", "ambient.json"), `{"schema":"goalrail.ambient-marker/v0","initialized_at":"2026-08-04T12:00:00Z"}`)
	writeFile(t, filepath.Join(legacy, ".goalrail", "runs", "receipt.json"), "retained receipt bytes\n")
	beforeMigration := snapshotCanaryTree(t, legacy)
	migrationOutput, _, err := runCommand(t, "migrate", "--repo", legacy, "--scaffold", "codex")
	if err != nil {
		t.Fatal(err)
	}
	var migration projectstate.InitializeReport
	if err := json.Unmarshal([]byte(migrationOutput), &migration); err != nil {
		t.Fatal(err)
	}
	if !migration.Migration || !migration.CommitRequired {
		t.Fatalf("migration report = %+v", migration)
	}
	restoreCanaryTree(t, legacy, beforeMigration)
	if afterRollback := snapshotCanaryTree(t, legacy); !reflect.DeepEqual(beforeMigration, afterRollback) {
		t.Fatal("pre-commit migration rollback did not restore exact legacy bytes")
	}
	legacyInspection, err := projectstate.Inspect(ctx, legacy)
	if err != nil {
		t.Fatal(err)
	}
	legacyEvidence, err := projectstate.DetectLegacyV018(ctx, legacy)
	if err != nil {
		t.Fatal(err)
	}
	if legacyInspection.State != projectstate.ClaimUnmanaged || legacyEvidence.Marker != projectstate.LegacyMarkerValid || legacyEvidence.Overlay != projectstate.LegacyOverlayComplete {
		t.Fatalf("legacy state after rollback: inspection=%+v evidence=%+v", legacyInspection, legacyEvidence)
	}
}

const canaryFixtureSchema = "goalrail.fixture-evidence/v1"

func canaryTime() time.Time {
	return time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
}

func installCanaryPolicy(t *testing.T, repository string) {
	t.Helper()
	policyPath := filepath.Join(repository, filepath.FromSlash(domain.DefaultProjectPolicyPath))
	policyRaw, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := domain.DecodeProjectPolicy(bytes.NewReader(policyRaw))
	if err != nil {
		t.Fatal(err)
	}
	policy.ExceptionAuthorities = []domain.PolicyExceptionAuthority{{
		ID: "owner-break-glass", Class: domain.ExceptionBreakGlass,
		ActorRefs: []string{"role:repository-owner"}, PathPrefixes: []string{"internal"},
		EffectScopes: []string{"material_change"}, MaxDurationSeconds: 3600, OwnerDecisionRequired: true,
	}}
	policyArtifact, err := domain.FreezeProjectPolicy(policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(policyPath, policyArtifact.CanonicalJSON(), 0o644); err != nil {
		t.Fatal(err)
	}
	declarationPath := filepath.Join(repository, filepath.FromSlash(domain.ProjectDeclarationPath))
	declarationRaw, err := os.ReadFile(declarationPath)
	if err != nil {
		t.Fatal(err)
	}
	declaration, err := domain.DecodeProjectDeclaration(bytes.NewReader(declarationRaw))
	if err != nil {
		t.Fatal(err)
	}
	declaration.Policy.Digest = policyArtifact.Digest()
	declarationArtifact, err := domain.FreezeProjectDeclaration(declaration)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(declarationPath, declarationArtifact.CanonicalJSON(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func copyCanaryChange(t *testing.T, repository string) {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve canary test path")
	}
	source := filepath.Join(
		filepath.Dir(currentFile),
		"..", "..", "openspec", "changes", "archive", "2026-08-05-project-lineage-admission-v0",
	)
	target := filepath.Join(repository, "openspec", "changes", "project-lineage-admission-v0")
	if err := filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, raw, 0o644)
	}); err != nil {
		t.Fatal(err)
	}
}

func canaryClone(t *testing.T, source, name string) string {
	t.Helper()
	target := filepath.Join(t.TempDir(), name)
	command := exec.Command("git", "clone", "--quiet", source, target)
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git clone: %v: %s", err, output)
	}
	gitCommand(t, target, "config", "user.email", "probe@localhost")
	gitCommand(t, target, "config", "user.name", "probe")
	gitCommand(t, target, "config", "core.excludesFile", os.DevNull)
	return target
}

func attachCanaryReplica(
	t *testing.T,
	repository string,
	source domain.ContentAddressedEvidenceReference,
	relation domain.LineageRelation,
	cardinality domain.RelationCardinality,
	kind, identity string,
	raw []byte,
	schema string,
	actor string,
	observedAt time.Time,
) {
	t.Helper()
	digest := domain.DigestCanonicalJSON(raw)
	replica, err := lineagestore.PrepareReplica(bytes.NewReader(raw), digest, schema)
	if err != nil {
		t.Fatal(err)
	}
	attachCanaryEvent(t, repository, domain.LineageEvent{
		Schema: domain.LineageEventSchemaV1, WorkUnitID: domain.WorkUnitID(strings.TrimPrefix(source.Identity, "work-unit:")),
		Relation: relation, Cardinality: cardinality,
		Sources: []domain.ContentAddressedEvidenceReference{source},
		Targets: []domain.ContentAddressedEvidenceReference{{
			ArtifactKind: kind, Identity: identity, Version: "1", Digest: digest,
			SourceRef: "repo:root/" + replica.Reference, AdapterID: "goalrail",
		}},
		ActorRef: actor, AdapterID: "goalrail", ObservedAt: observedAt,
	}, &replica)
}

func attachCanaryEvent(t *testing.T, repository string, event domain.LineageEvent, replica *lineagestore.Replica) {
	t.Helper()
	if _, err := lineagestore.Attach(context.Background(), lineagestore.AttachOptions{
		Repository: repository, Event: event, Replica: replica,
	}); err != nil {
		t.Fatal(err)
	}
}

func canaryFixtureEvidence(t *testing.T, identity string) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]string{
		"identity": identity,
		"schema":   canaryFixtureSchema,
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func verifyCanaryRange(
	t *testing.T,
	repository, base, head string,
	enrich func(*domain.AdmissionPacket),
) []byte {
	t.Helper()
	packet, err := admissiongit.CollectPacketSeed(context.Background(), repository, base, head)
	if err != nil {
		t.Fatal(err)
	}
	if enrich != nil {
		enrich(&packet)
	}
	packetArtifact, err := domain.FreezeAdmissionPacket(packet)
	if err != nil {
		t.Fatal(err)
	}
	packetPath := filepath.Join(t.TempDir(), "packet.json")
	if err := os.WriteFile(packetPath, packetArtifact.CanonicalJSON(), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err = runVerifyLineage(context.Background(), []string{
		"--repo", repository, "--base", base, "--head", head, "--packet", packetPath, "--json",
	}, &stdout, &stderr)
	// A denied verdict is a canonical result, not a command failure. The
	// advisory packet path denies by contract whenever an owner-gated relation
	// needs a live provider observation.
	if err != nil && exitCodeOf(err) != admission.ExitDenied {
		t.Fatalf("verify lineage: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}
	return stdout.Bytes()
}

func exitCodeOf(err error) int {
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	return -1
}

// verifyCanaryRangeWithProvider is the authoritative path: one invocation
// observes the provider through the registered adapter and verifies the frozen
// range with those in-memory candidates.
func verifyCanaryRangeWithProvider(
	t *testing.T,
	repository, base, head string,
	observedAt time.Time,
) []byte {
	t.Helper()
	seed, err := admissiongit.CollectPacketSeed(context.Background(), repository, base, head)
	if err != nil {
		t.Fatal(err)
	}
	event := githubadmission.WorkflowEvent{
		Name: "pull_request", Action: "opened",
		Repository: githubadmission.Repository{Owner: "heurema", Name: "goalrail"},
		Number:     1, BaseSHA: base, HeadSHA: head,
	}
	collection, err := githubadmission.Collect(context.Background(), githubadmission.CollectRequest{
		Seed: seed, Event: event, Reader: canaryProviderReader{observedAt: observedAt, base: base, head: head},
	})
	if err != nil {
		t.Fatal(err)
	}
	packet, err := domain.DecodeAdmissionPacket(bytes.NewReader(collection.Artifact.CanonicalJSON()))
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := admissiongit.CollectFrozenRange(context.Background(), repository, base, head, packet)
	if err != nil {
		t.Fatal(err)
	}
	result, err := admission.Verify(admission.Input{Range: frozen, Packet: packet, Candidates: collection.Candidates})
	if err != nil {
		t.Fatal(err)
	}
	canonical, _, err := admission.CanonicalResult(result)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}

// canaryProviderReader stands in for the authenticated GitHub API read. It
// returns only bounded identities, exactly as the real reader must.
type canaryProviderReader struct {
	observedAt time.Time
	base, head string
}

func (reader canaryProviderReader) Read(context.Context, githubadmission.Repository, int64) (githubadmission.Snapshot, error) {
	return githubadmission.Snapshot{
		PullRequest: githubadmission.PullRequest{
			Number: 1, BaseSHA: reader.base, HeadSHA: reader.head, State: "open",
			Author: "contributor", UpdatedAt: reader.observedAt.Add(-time.Minute),
		},
		Reviews: []githubadmission.Review{{
			ID: 1, Actor: "owner", State: "APPROVED", CommitSHA: reader.head,
			SubmittedAt: reader.observedAt.Add(-time.Minute),
		}},
		Checks: []githubadmission.Check{{ID: 1, Name: "build", Status: "completed", Conclusion: "success", AppSlug: "github-actions"}},
		// The shape the real reader produces, not a role invented by the test.
		// Stubbing role:repository-owner here hid a seam: the adapter reports a
		// provider permission, and a policy that permits only a role can never
		// authorize a real approval.
		AuthorizedActors: map[string]string{"owner": "github-permission:admin"},

		AuthorityAvailable: true, ObservedAt: reader.observedAt, Authenticated: true,
	}, nil
}

func assertCanaryAdmission(
	t *testing.T,
	raw []byte,
	classification domain.AdmissionClassification,
	outcome domain.AdmissionOutcome,
	reason domain.AdmissionReasonCode,
) {
	t.Helper()
	var result domain.AdmissionResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result.Classification != classification || result.Outcome != outcome {
		t.Fatalf("admission result = %+v", result)
	}
	if reason == "" {
		if len(result.Reasons) != 0 {
			t.Fatalf("normal admission retained reasons: %+v", result.Reasons)
		}
		return
	}
	if len(result.Reasons) != 1 || result.Reasons[0].Code != reason {
		t.Fatalf("admission reason = %+v, want %s", result.Reasons, reason)
	}
}

type canarySnapshotEntry struct {
	Bytes []byte
	Mode  fs.FileMode
}

func snapshotCanaryTree(t *testing.T, root string) map[string]canarySnapshotEntry {
	t.Helper()
	result := make(map[string]canarySnapshotEntry)
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative == ".git" || strings.HasPrefix(filepath.ToSlash(relative), ".git/") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if relative == "." || entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result[filepath.ToSlash(relative)] = canarySnapshotEntry{Bytes: raw, Mode: info.Mode()}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return result
}

func restoreCanaryTree(t *testing.T, root string, snapshot map[string]canarySnapshotEntry) {
	t.Helper()
	current := snapshotCanaryTree(t, root)
	for relative := range current {
		if _, retained := snapshot[relative]; retained {
			continue
		}
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(relative))); err != nil {
			t.Fatal(err)
		}
	}
	for relative, entry := range snapshot {
		absolute := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absolute, entry.Bytes, entry.Mode.Perm()); err != nil {
			t.Fatal(err)
		}
	}
}

func installCanaryLegacyV018(t *testing.T, repository string) {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve canary test path")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	legacy := harness.LegacyV018Canon()
	for _, file := range legacy.Files {
		relative := strings.TrimPrefix(file.Path, harness.OverlayDirectory+"/")
		raw, err := os.ReadFile(filepath.Join(repositoryRoot, "internal", "harness", "testdata", "canon-v1", filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		if harness.Digest(raw) != file.Digest {
			t.Fatalf("legacy fixture digest mismatch for %s", file.Path)
		}
		absolute := filepath.Join(repository, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absolute, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
