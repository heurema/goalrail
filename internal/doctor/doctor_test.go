package doctor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/harness"
	"github.com/heurema/goalrail/internal/project"
)

func TestDoctorV2KeepsManagedIdentityWhenCleanCloneNeedsSetup(t *testing.T) {
	repository := initializedRepository(t)
	git(t, repository, "add", ".")
	git(t, repository, "commit", "-qm", "initialize Goalrail")
	clone := filepath.Join(t.TempDir(), "clone")
	git(t, repository, "clone", "-q", repository, clone)
	linked := filepath.Join(t.TempDir(), "linked")
	git(t, repository, "worktree", "add", "-q", "--detach", linked, "HEAD")

	original, err := project.Inspect(context.Background(), repository)
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{clone, filepath.Join(linked, ".goalrail")} {
		diagnosis, err := Diagnose(context.Background(), DiagnoseInput{
			RepositoryRoot: root,
			Home:           t.TempDir(),
			Scaffolds:      []ambient.Scaffold{ambient.ScaffoldClaudeCode},
			PlanningObserver: PlanningObserverFunc(func(context.Context, domain.SetupProfile) PlanningObservation {
				return PlanningObservation{
					Bundle:   readyComponent("goalrail_bundle", "goalrail"),
					Runtime:  ComponentReadiness{Kind: "runtime", ID: "node", RequiredVersion: "22.18.0", State: ComponentMissing},
					Compiler: readyComponent("compiler", "@fission-ai/openspec"),
				}
			}),
		})
		if err != nil {
			t.Fatal(err)
		}
		if diagnosis.Schema != SchemaV2 || !diagnosis.Managed || !diagnosis.Initialized ||
			!diagnosis.InitializedDeprecated || diagnosis.Working || !diagnosis.SetupRequired {
			t.Fatalf("clean checkout diagnosis = %#v", diagnosis)
		}
		if diagnosis.ProjectID != original.Declaration.ProjectID || diagnosis.Claim.State != project.ClaimManaged {
			t.Fatalf("portable identity = %s/%s, want %s", diagnosis.ProjectID, diagnosis.Claim.State, original.Declaration.ProjectID)
		}
		if len(diagnosis.Attachments) != 1 || diagnosis.Attachments[0].ProjectID != string(original.Declaration.ProjectID) ||
			diagnosis.Attachments[0].EnforcementScope != "local_advisory_only" {
			t.Fatalf("attachment health lost committed identity or local scope: %#v", diagnosis.Attachments)
		}
		if diagnosis.Planning.Runtime.State != ComponentMissing || !hasReason(diagnosis, "GOALRAIL_SETUP_REQUIRED") {
			t.Fatalf("missing runtime did not route through setup: %#v", diagnosis.Planning)
		}
	}
}

func TestDoctorDoesNotObserveUnmanagedOrInvalidRepositories(t *testing.T) {
	observerCalls, reviewCalls, releaseCalls := 0, 0, 0
	input := func(root string) DiagnoseInput {
		return DiagnoseInput{
			RepositoryRoot: root, Home: t.TempDir(),
			PlanningObserver: PlanningObserverFunc(func(context.Context, domain.SetupProfile) PlanningObservation {
				observerCalls++
				return PlanningObservation{}
			}),
			ActivationObserver: ActivationObserverFunc(func(context.Context, ActivationRequest) ActivationEvidence {
				observerCalls++
				return ActivationEvidence{}
			}),
			LatestRelease: func(context.Context) (string, time.Time, error) {
				releaseCalls++
				return "", time.Time{}, nil
			},
		}
	}

	unmanaged := gitRepository(t)
	unmanagedInput := input(unmanaged)
	unmanagedInput.Review = func() harness.ReviewState { reviewCalls++; return harness.ReviewState{} }
	diagnosis, err := Diagnose(context.Background(), unmanagedInput)
	if err != nil {
		t.Fatal(err)
	}
	if diagnosis.Category != CategoryUnmanaged || len(diagnosis.Attachments) != 0 || diagnosis.Managed {
		t.Fatalf("unmanaged diagnosis = %#v", diagnosis)
	}

	invalid := gitRepository(t)
	write(t, filepath.Join(invalid, ".goalrail", "project.json"), []byte("{"))
	invalidInput := input(invalid)
	invalidInput.Review = func() harness.ReviewState { reviewCalls++; return harness.ReviewState{} }
	diagnosis, err = Diagnose(context.Background(), invalidInput)
	if err != nil {
		t.Fatal(err)
	}
	if diagnosis.Category != CategoryDeclaredInvalid || diagnosis.Claim.State != project.ClaimDeclaredInvalid || len(diagnosis.Attachments) != 0 {
		t.Fatalf("invalid diagnosis = %#v", diagnosis)
	}
	if observerCalls != 0 || reviewCalls != 0 || releaseCalls != 0 {
		t.Fatalf("claim-first boundary leaked observations: planning/activation=%d review=%d release=%d", observerCalls, reviewCalls, releaseCalls)
	}
}

func TestDoctorSeparatesLocalReadinessFromSharedActivation(t *testing.T) {
	repository := initializedRepository(t)
	home := t.TempDir()
	attachClaude(t, repository, home)

	base := DiagnoseInput{
		RepositoryRoot: repository, Home: home,
		Scaffolds:        []ambient.Scaffold{ambient.ScaffoldClaudeCode},
		PlanningObserver: readyPlanningObserver(),
	}
	preparedOnly, err := Diagnose(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	if !preparedOnly.PreparedAdmission.Prepared || !preparedOnly.LocallyReady || preparedOnly.SharedAdmissionActive ||
		preparedOnly.SharedAdmission.State != ActivationUnknown || preparedOnly.Working {
		t.Fatalf("workflow-only diagnosis collapsed activation: %#v", preparedOnly)
	}

	base.ActivationObserver = ActivationObserverFunc(func(context.Context, ActivationRequest) ActivationEvidence {
		return ActivationEvidence{
			State: ActivationActive, Provider: "github", Boundary: "branch-protection",
			RequiredCheck: requiredAdmissionCheck, ObservedTarget: "refs/heads/main",
			EvidenceSource: "github-ruleset-read", ObservedAt: time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
			Freshness: ActivationFreshCurrent,
		}
	})
	active, err := Diagnose(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	if !active.LocallyReady || !active.LineageReady || !active.SharedAdmissionActive || !active.Working || active.Category != CategoryWorking {
		t.Fatalf("evidence-backed working diagnosis = %#v", active)
	}

	base.ActivationObserver = ActivationObserverFunc(func(context.Context, ActivationRequest) ActivationEvidence {
		return ActivationEvidence{
			State: ActivationActive, Provider: "github", Boundary: "branch-protection",
			RequiredCheck: requiredAdmissionCheck, ObservedTarget: "refs/heads/main",
			EvidenceSource: "expired-receipt", ObservedAt: time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
			Freshness: ActivationFreshStale,
		}
	})
	stale, err := Diagnose(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	if stale.SharedAdmission.State != ActivationStale || stale.SharedAdmissionActive || stale.Working || stale.Category != CategoryAdvisory {
		t.Fatalf("stale evidence was promoted: %#v", stale.SharedAdmission)
	}
}

func TestActivationEvidenceStatesNeverPromoteWithoutCurrentProof(t *testing.T) {
	request := ActivationRequest{
		ProjectID: "prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		AdapterID: "github-actions", RequiredCheck: requiredAdmissionCheck,
		PreparedPath: project.PreparedAdmissionPath,
	}
	for _, state := range []ActivationState{ActivationInactive, ActivationUnknown, ActivationStale, ActivationAccessDenied} {
		evidence := activationEvidence(context.Background(), ActivationObserverFunc(func(context.Context, ActivationRequest) ActivationEvidence {
			return ActivationEvidence{State: state, RequiredCheck: requiredAdmissionCheck, Freshness: ActivationFreshUnverified}
		}), request)
		if evidence.State != state || evidence.State == ActivationActive {
			t.Fatalf("state %s was rewritten as %s", state, evidence.State)
		}
	}
	insufficient := activationEvidence(context.Background(), ActivationObserverFunc(func(context.Context, ActivationRequest) ActivationEvidence {
		return ActivationEvidence{State: ActivationActive, RequiredCheck: requiredAdmissionCheck, Freshness: ActivationFreshCurrent}
	}), request)
	if insufficient.State != ActivationStale {
		t.Fatalf("provenance-free active claim = %s, want stale", insufficient.State)
	}
}

func TestDoctorRetainsTrustAndGovernanceBoundaries(t *testing.T) {
	repository := initializedRepository(t)
	home := t.TempDir()
	executable := executableFixture(t)
	plan, err := ambient.PlanConnection(ambient.ScaffoldCodex, home, executable)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ambient.Connect(plan); err != nil {
		t.Fatal(err)
	}
	diagnosis, err := Diagnose(context.Background(), DiagnoseInput{
		RepositoryRoot: repository, Home: home,
		Scaffolds:          []ambient.Scaffold{ambient.ScaffoldCodex},
		PlanningObserver:   readyPlanningObserver(),
		ActivationObserver: activeObserver(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if diagnosis.Attachments[0].Trust != ambient.TrustPending || diagnosis.Attachments[0].Working || diagnosis.LocallyReady ||
		!diagnosis.SharedAdmissionActive || diagnosis.Attachments[0].EnforcementScope != "local_advisory_only" {
		t.Fatalf("untrusted local attachment boundary = %#v", diagnosis)
	}

	policyPath := filepath.Join(repository, filepath.FromSlash(domain.DefaultProjectPolicyPath))
	raw, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatal(err)
	}
	write(t, policyPath, append(raw, ' '))
	diagnosis, err = Diagnose(context.Background(), DiagnoseInput{RepositoryRoot: repository, Home: home})
	if err != nil {
		t.Fatal(err)
	}
	if !diagnosis.Managed || !diagnosis.Initialized || diagnosis.LineageReady || diagnosis.Category != CategoryGovernanceInvalid ||
		!hasReason(diagnosis, "GOVERNING_ARTIFACT_INVALID") {
		t.Fatalf("policy mismatch erased identity or passed lineage: %#v", diagnosis)
	}
}

func initializedRepository(t *testing.T) string {
	t.Helper()
	repository := gitRepository(t)
	if _, err := project.Initialize(context.Background(), repository, project.InitializeOptions{}); err != nil {
		t.Fatal(err)
	}
	return repository
}

func gitRepository(t *testing.T) string {
	t.Helper()
	repository := filepath.Join(t.TempDir(), "repository")
	if err := os.MkdirAll(repository, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, repository, "init", "-q")
	git(t, repository, "config", "user.email", "doctor@localhost")
	git(t, repository, "config", "user.name", "doctor")
	git(t, repository, "config", "core.excludesFile", os.DevNull)
	return repository
}

func git(t *testing.T, repository string, arguments ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repository}, arguments...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", arguments, err, output)
	}
}

func write(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func executableFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "gr")
	write(t, path, []byte("#!/bin/sh\nexit 0\n"))
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func attachClaude(t *testing.T, repository, home string) {
	t.Helper()
	target, err := ambient.RegistrationTarget(ambient.ScaffoldClaudeCode, home, repository)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := ambient.PlanRegistration(target, executableFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ambient.Connect(plan); err != nil {
		t.Fatal(err)
	}
}

func readyPlanningObserver() PlanningObserver {
	return PlanningObserverFunc(func(context.Context, domain.SetupProfile) PlanningObservation {
		return PlanningObservation{
			Bundle:   readyComponent("goalrail_bundle", "goalrail"),
			Runtime:  readyComponent("runtime", "node"),
			Compiler: readyComponent("compiler", "@fission-ai/openspec"),
		}
	})
}

func readyComponent(kind, id string) ComponentReadiness {
	return ComponentReadiness{Kind: kind, ID: id, State: ComponentReady}
}

func activeObserver() ActivationObserver {
	return ActivationObserverFunc(func(context.Context, ActivationRequest) ActivationEvidence {
		return ActivationEvidence{
			State: ActivationActive, Provider: "github", Boundary: "branch-protection",
			RequiredCheck: requiredAdmissionCheck, ObservedTarget: "refs/heads/main",
			EvidenceSource: "github-ruleset-read", ObservedAt: time.Now().UTC(), Freshness: ActivationFreshCurrent,
		}
	})
}

func hasReason(diagnosis Diagnosis, code string) bool {
	for _, reason := range diagnosis.Reasons {
		if reason.Code == code {
			return true
		}
	}
	return false
}
