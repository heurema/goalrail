package localrun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/project"
)

type fixtureManagedAuthorityVerifier struct {
	calls atomic.Int32
	err   error
}

func (verifier *fixtureManagedAuthorityVerifier) Verify(
	context.Context,
	string,
	domain.WorkSpec,
) error {
	verifier.calls.Add(1)
	return verifier.err
}

func TestPrepareManagedWorkSpecRequiresAuthorityBeforeState(t *testing.T) {
	service, spec, store, _ := productionServiceFixture(t)
	spec = managedFixtureWorkSpec(spec)
	verifier := &fixtureManagedAuthorityVerifier{err: errors.New("authority rejected")}
	service.authority = verifier

	raw, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Prepare(context.Background(), bytes.NewReader(raw)); err == nil || !strings.Contains(err.Error(), "authority rejected") {
		t.Fatalf("expected authority rejection, got %v", err)
	}
	if verifier.calls.Load() != 1 {
		t.Fatalf("authority verifier calls = %d, want 1", verifier.calls.Load())
	}
	entries, err := os.ReadDir(store.Root())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("authority rejection left prepared state: %+v", entries)
	}
}

func TestLegacyWorkSpecIsInspectableButCannotPrepareNormalRun(t *testing.T) {
	service, managed, store, _ := productionServiceFixture(t)
	legacy := fixtureWorkSpec(managed.Repository.Root, managed.Repository.BaseRevision)
	raw, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Prepare(context.Background(), bytes.NewReader(raw)); !errors.Is(err, ErrLegacyWorkSpecAuthority) {
		t.Fatalf("legacy normal prepare error = %v, want ErrLegacyWorkSpecAuthority", err)
	}

	frozen, err := domain.FreezeWorkSpec(legacy)
	if err != nil {
		t.Fatal(err)
	}
	preparation := Preparation{
		WorkSpecDigest: frozen.Digest(),
		PreparedAt:     time.Date(2026, 8, 5, 9, 0, 0, 0, time.UTC),
		Baseline:       fixtureObservation("legacy-baseline"),
		State:          StatePrepared,
	}
	if err := store.WriteBytesOnce(preparedPath(frozen.Digest(), "work-spec.json"), frozen.CanonicalJSON(), true); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteJSONOnce(preparedPath(frozen.Digest(), "preparation.json"), preparation, false); err != nil {
		t.Fatal(err)
	}
	inspected, err := service.InspectPrepared(frozen.Digest())
	if err != nil {
		t.Fatal(err)
	}
	if inspected.WorkSpec.Spec().Schema != domain.WorkSpecSchemaV0 ||
		!bytes.Equal(inspected.WorkSpec.CanonicalJSON(), frozen.CanonicalJSON()) {
		t.Fatal("legacy WorkSpec was not preserved byte-for-byte for inspection")
	}
	stored, err := store.ReadBytes(preparedPath(frozen.Digest(), "work-spec.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, frozen.CanonicalJSON()) {
		t.Fatal("legacy WorkSpec history was rewritten during inspection")
	}
}

func TestRepositoryManagedAuthorityVerifierRequiresOneCoherentOpenUnit(t *testing.T) {
	root := t.TempDir()
	spec := managedFixtureWorkSpec(fixtureWorkSpec(root, strings.Repeat("a", 40)))
	projectID := spec.Project.ID
	declaration := domain.ProjectDeclaration{
		ProjectID: projectID,
		Policy: domain.CommittedArtifactReference{
			Path:   spec.Policy.ArtifactRef,
			Digest: spec.Policy.Digest,
		},
	}
	inspection := project.Inspection{
		State:             project.ClaimManaged,
		WorktreeRoot:      root,
		Declaration:       declaration,
		DeclarationDigest: spec.Project.Digest,
	}
	artifacts := project.GoverningArtifacts{Policy: project.ArtifactFinding{
		State:          project.ArtifactCurrent,
		ObservedDigest: spec.Policy.Digest,
	}}
	changeIntent := domain.IntentSnapshot{
		ID:      spec.Intent.ID,
		Version: spec.Intent.Version,
		Status:  domain.IntentConfirmed,
	}
	unit := domain.WorkUnit{
		Schema:            domain.WorkUnitSchemaV1,
		ID:                spec.WorkUnit.ID,
		ProjectID:         projectID,
		DeclarationDigest: spec.Project.Digest,
		PolicyDigest:      spec.Policy.Digest,
		IntentRef:         expectedIntentReference(spec.Intent),
		ChangeRef:         expectedChangeReference(*spec.Change),
		CreatedAt:         time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC),
		Lifecycle:         domain.WorkUnitOpen,
		RequiredRelations: []domain.LineageRequirement{{
			Relation:    domain.LineageTerminalReceipt,
			Cardinality: domain.RelationSingle,
		}},
	}
	unitArtifact, err := domain.FreezeWorkUnit(unit)
	if err != nil {
		t.Fatal(err)
	}
	spec.WorkUnit.Digest = unitArtifact.Digest()

	verifier := repositoryManagedAuthorityVerifier{
		inspectProject: func(context.Context, string) (project.Inspection, error) {
			return inspection, nil
		},
		inspectArtifacts: func(project.Inspection) (project.GoverningArtifacts, error) {
			return artifacts, nil
		},
		loadChangeIntent: func(string) (domain.IntentSnapshot, error) {
			return changeIntent, nil
		},
		readWorkUnit: func(string, domain.WorkSpecWorkUnitReference) (domain.WorkUnit, domain.CanonicalArtifact, error) {
			return unit, unitArtifact, nil
		},
	}
	if err := verifier.Verify(context.Background(), root, spec); err != nil {
		t.Fatal(err)
	}

	tests := map[string]func(*domain.WorkSpec){
		"project mismatch": func(candidate *domain.WorkSpec) {
			candidate.Project.Digest = domain.SHA256Digest("sha256:" + strings.Repeat("0", 64))
		},
		"policy mismatch": func(candidate *domain.WorkSpec) {
			candidate.Policy.Digest = domain.SHA256Digest("sha256:" + strings.Repeat("1", 64))
		},
		"non-current change": func(candidate *domain.WorkSpec) {
			candidate.Change.ArtifactRef = "openspec/changes/archive/2026-08-05-project-lineage-admission-v0"
		},
		"work-unit mismatch": func(candidate *domain.WorkSpec) {
			candidate.WorkUnit.Digest = domain.SHA256Digest("sha256:" + strings.Repeat("2", 64))
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := managedFixtureWorkSpec(spec)
			mutate(&candidate)
			if err := verifier.Verify(context.Background(), root, candidate); err == nil {
				t.Fatal("expected managed authority rejection")
			}
		})
	}

	closed := unit
	closed.Lifecycle = domain.WorkUnitClosed
	closedArtifact, err := domain.FreezeWorkUnit(closed)
	if err != nil {
		t.Fatal(err)
	}
	closedSpec := managedFixtureWorkSpec(spec)
	closedSpec.WorkUnit.Digest = closedArtifact.Digest()
	verifier.readWorkUnit = func(string, domain.WorkSpecWorkUnitReference) (domain.WorkUnit, domain.CanonicalArtifact, error) {
		return closed, closedArtifact, nil
	}
	if err := verifier.Verify(context.Background(), root, closedSpec); err == nil || !strings.Contains(err.Error(), "not open") {
		t.Fatalf("expected closed work-unit rejection, got %v", err)
	}
}

func managedFixtureWorkSpec(spec domain.WorkSpec) domain.WorkSpec {
	projectDigest := domain.SHA256Digest("sha256:" + strings.Repeat("c", 64))
	policyDigest := domain.SHA256Digest("sha256:" + strings.Repeat("d", 64))
	changeDigest := domain.SHA256Digest("sha256:" + strings.Repeat("e", 64))
	spec.Schema = domain.WorkSpecSchemaV1
	spec.Project = &domain.WorkSpecProjectReference{
		ID:          "prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ArtifactRef: domain.ProjectDeclarationPath,
		Digest:      projectDigest,
	}
	spec.Policy = &domain.WorkSpecPolicyReference{
		ArtifactRef: ".goalrail/policy.json",
		Digest:      policyDigest,
	}
	spec.Change = &domain.WorkSpecChangeReference{
		ID:            "project-lineage-admission-v0",
		ArtifactRef:   "openspec/changes/project-lineage-admission-v0",
		Digest:        changeDigest,
		IntentID:      spec.Intent.ID,
		IntentVersion: spec.Intent.Version,
		IntentDigest:  domain.SHA256Digest(spec.Intent.Digest),
	}
	spec.WorkUnit = &domain.WorkSpecWorkUnitReference{
		ID:          "wu_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ArtifactRef: ".goalrail/work-units/wu_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/unit.json",
		Digest:      domain.SHA256Digest("sha256:" + strings.Repeat("f", 64)),
	}
	spec.Posture = domain.PostureTrustedLocalProviderEnforcedV1
	return spec
}
