package setup

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/domain"
)

func TestExactAuthorizationRevalidatesOnceWithoutWrites(t *testing.T) {
	repository := initializedRepository(t)
	home := t.TempDir()
	evidence := releaseEvidence(t, "darwin", "arm64")
	options := authorizationPlanOptions(repository, home, &evidence)
	plan := mustGeneratePlan(t, options)
	beforeRepository := snapshotTree(t, repository)
	beforeHome := snapshotTree(t, home)

	binding, err := BindAuthorization(plan, options, exactConsent(plan))
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := binding.BeforeFirstMutation(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if attempt.PlanArtifact().Digest() != plan.Artifact.Digest() ||
		!bytes.Equal(attempt.PlanArtifact().CanonicalJSON(), plan.Artifact.CanonicalJSON()) {
		t.Fatal("authorized attempt did not preserve the exact plan artifact")
	}
	authorization, err := domain.DecodePlanAuthorizationReference(bytes.NewReader(attempt.AuthorizationArtifact().CanonicalJSON()))
	if err != nil {
		t.Fatal(err)
	}
	if authorization.PlanDigest != plan.Artifact.Digest() || authorization.ProjectID != plan.Plan.ProjectID {
		t.Fatalf("authorization does not bind the plan: %#v", authorization)
	}
	if _, err := binding.BeforeFirstMutation(context.Background()); !errors.Is(err, ErrAuthorizationConsumed) {
		t.Fatalf("second attempt error = %v, want %v", err, ErrAuthorizationConsumed)
	}
	assertTreesUnchanged(t, repository, home, beforeRepository, beforeHome)
}

func TestBindAuthorizationRejectsNonExactConsentAndInvalidPlans(t *testing.T) {
	repository := initializedRepository(t)
	home := t.TempDir()
	evidence := releaseEvidence(t, "darwin", "arm64")
	options := authorizationPlanOptions(repository, home, &evidence)
	plan := mustGeneratePlan(t, options)
	exact := exactConsent(plan)
	stale := exact
	staleAuthorization := *exact.Authorization
	staleAuthorization.PlanDigest = domain.SHA256Digest("sha256:" + strings.Repeat("0", 64))
	stale.Authorization = &staleAuthorization
	wrongProject := exact
	wrongProjectAuthorization := *exact.Authorization
	projectID := string(wrongProjectAuthorization.ProjectID)
	replacement := "0"
	if projectID[len(projectID)-1:] == replacement {
		replacement = "1"
	}
	wrongProjectAuthorization.ProjectID = domain.ProjectID(projectID[:len(projectID)-1] + replacement)
	wrongProject.Authorization = &wrongProjectAuthorization
	invalid := exact
	invalidAuthorization := *exact.Authorization
	invalidAuthorization.AuthorizedAt = time.Time{}
	invalid.Authorization = &invalidAuthorization

	for _, test := range []struct {
		name    string
		consent Consent
		want    error
	}{
		{name: "missing decision", consent: Consent{}, want: ErrAuthorizationMissing},
		{name: "missing reference", consent: Consent{Decision: ConsentGrant, Scope: ConsentExactPlan}, want: ErrAuthorizationMissing},
		{name: "refused", consent: Consent{Decision: ConsentRefuse, Scope: ConsentExactPlan, Authorization: exact.Authorization}, want: ErrAuthorizationRefused},
		{name: "general", consent: Consent{Decision: ConsentGrant, Scope: ConsentGeneral, Authorization: exact.Authorization}, want: ErrAuthorizationGeneral},
		{name: "stale digest", consent: stale, want: ErrAuthorizationStale},
		{name: "wrong project", consent: wrongProject, want: ErrAuthorizationStale},
		{name: "invalid reference", consent: invalid, want: ErrAuthorizationInvalid},
	} {
		t.Run(test.name, func(t *testing.T) {
			beforeRepository := snapshotTree(t, repository)
			beforeHome := snapshotTree(t, home)
			if _, err := BindAuthorization(plan, options, test.consent); !errors.Is(err, test.want) {
				t.Fatalf("BindAuthorization() error = %v, want %v", err, test.want)
			}
			assertTreesUnchanged(t, repository, home, beforeRepository, beforeHome)
		})
	}

	incomplete := mustGeneratePlan(t, authorizationPlanOptions(repository, home, nil))
	if _, err := BindAuthorization(incomplete, options, exact); !errors.Is(err, ErrPlanIncomplete) {
		t.Fatalf("incomplete plan error = %v, want %v", err, ErrPlanIncomplete)
	}
	tampered := plan
	tampered.Plan.Platform = "linux-amd64"
	if _, err := BindAuthorization(tampered, options, exact); !errors.Is(err, ErrPlanInvalid) {
		t.Fatalf("tampered plan error = %v, want %v", err, ErrPlanInvalid)
	}
}

func TestAuthorizationBecomesStaleWhenTargetStateChanges(t *testing.T) {
	repository := initializedRepository(t)
	home := t.TempDir()
	evidence := releaseEvidence(t, "darwin", "arm64")
	options := authorizationPlanOptions(repository, home, &evidence)
	plan := mustGeneratePlan(t, options)
	binding, err := BindAuthorization(plan, options, exactConsent(plan))
	if err != nil {
		t.Fatal(err)
	}

	writeFixtureFile(t, filepath.Join(home, ".local", "bin", "gr"), []byte("fixture gr\n"), 0o755)
	beforeRepository := snapshotTree(t, repository)
	beforeHome := snapshotTree(t, home)
	if _, err := binding.BeforeFirstMutation(context.Background()); !errors.Is(err, ErrAuthorizationStale) {
		t.Fatalf("changed target error = %v, want %v", err, ErrAuthorizationStale)
	}
	if _, err := binding.BeforeFirstMutation(context.Background()); !errors.Is(err, ErrAuthorizationConsumed) {
		t.Fatalf("retry error = %v, want %v", err, ErrAuthorizationConsumed)
	}
	assertTreesUnchanged(t, repository, home, beforeRepository, beforeHome)
}

func TestAuthorizationBecomesStaleWhenDeclarationCannotBeResolved(t *testing.T) {
	repository := initializedRepository(t)
	home := t.TempDir()
	evidence := releaseEvidence(t, "darwin", "arm64")
	options := authorizationPlanOptions(repository, home, &evidence)
	plan := mustGeneratePlan(t, options)
	binding, err := BindAuthorization(plan, options, exactConsent(plan))
	if err != nil {
		t.Fatal(err)
	}

	declarationPath := filepath.Join(repository, ".goalrail", "project.json")
	raw, err := os.ReadFile(declarationPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(declarationPath, append(raw, []byte("not-json")...), 0o644); err != nil {
		t.Fatal(err)
	}
	beforeRepository := snapshotTree(t, repository)
	beforeHome := snapshotTree(t, home)
	if _, err := binding.BeforeFirstMutation(context.Background()); !errors.Is(err, ErrAuthorizationStale) {
		t.Fatalf("changed declaration error = %v, want %v", err, ErrAuthorizationStale)
	}
	assertTreesUnchanged(t, repository, home, beforeRepository, beforeHome)
}

func TestAuthorizationPinsEvidenceBytesAgainstCallerMutation(t *testing.T) {
	repository := initializedRepository(t)
	home := t.TempDir()
	evidence := releaseEvidence(t, "darwin", "arm64")
	options := authorizationPlanOptions(repository, home, &evidence)
	plan := mustGeneratePlan(t, options)
	binding, err := BindAuthorization(plan, options, exactConsent(plan))
	if err != nil {
		t.Fatal(err)
	}

	evidence.MetadataRaw[0] = 'x'
	evidence.ManifestRaw[0] = 'x'
	if _, err := binding.BeforeFirstMutation(context.Background()); err != nil {
		t.Fatalf("caller-owned evidence changed bound authorization: %v", err)
	}
}

func TestAuthorizationAllowsAtMostOneConcurrentAttempt(t *testing.T) {
	repository := initializedRepository(t)
	home := t.TempDir()
	evidence := releaseEvidence(t, "darwin", "arm64")
	options := authorizationPlanOptions(repository, home, &evidence)
	plan := mustGeneratePlan(t, options)
	binding, err := BindAuthorization(plan, options, exactConsent(plan))
	if err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	errorsSeen := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for range 2 {
		go func() {
			ready.Done()
			<-start
			_, attemptErr := binding.BeforeFirstMutation(context.Background())
			errorsSeen <- attemptErr
		}()
	}
	ready.Wait()
	close(start)

	var successes, consumed int
	for range 2 {
		err := <-errorsSeen
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrAuthorizationConsumed):
			consumed++
		default:
			t.Fatalf("unexpected concurrent attempt error: %v", err)
		}
	}
	if successes != 1 || consumed != 1 {
		t.Fatalf("concurrent results: successes=%d consumed=%d", successes, consumed)
	}
}

func authorizationPlanOptions(repository, home string, evidence *ReleaseEvidence) PlanOptions {
	return PlanOptions{
		RepositoryRoot: repository,
		Home:           home,
		OS:             "darwin",
		Arch:           "arm64",
		Scaffold:       ambient.ScaffoldCodex,
		Evidence:       evidence,
	}
}

func mustGeneratePlan(t *testing.T, options PlanOptions) Result {
	t.Helper()
	result, err := Generate(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func exactConsent(plan Result) Consent {
	authorization := domain.PlanAuthorizationReference{
		Schema:       domain.PlanAuthorizationSchemaV1,
		ProjectID:    plan.Plan.ProjectID,
		PlanDigest:   plan.Artifact.Digest(),
		DecisionRef:  "decision:setup-plan-v0",
		ActorRef:     "owner:repository-owner",
		AuthorizedAt: time.Date(2026, time.August, 5, 12, 0, 0, 0, time.FixedZone("owner", 3*60*60)),
	}
	return Consent{Decision: ConsentGrant, Scope: ConsentExactPlan, Authorization: &authorization}
}

func assertTreesUnchanged(t *testing.T, repository, home string, beforeRepository, beforeHome []string) {
	t.Helper()
	if after := snapshotTree(t, repository); !reflect.DeepEqual(after, beforeRepository) {
		t.Fatalf("authorization changed repository\nbefore: %v\nafter:  %v", beforeRepository, after)
	}
	if after := snapshotTree(t, home); !reflect.DeepEqual(after, beforeHome) {
		t.Fatalf("authorization changed user home\nbefore: %v\nafter:  %v", beforeHome, after)
	}
}
