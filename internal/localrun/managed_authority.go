package localrun

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	openspecadapter "github.com/heurema/goalrail/internal/adapters/openspec"
	"github.com/heurema/goalrail/internal/boundedio"
	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/project"
)

const lineageAdapterID = "goalrail"

// ManagedAuthorityVerifier resolves repository-owned authority immediately
// before a managed WorkSpec can become prepared state.
type ManagedAuthorityVerifier interface {
	Verify(context.Context, string, domain.WorkSpec) error
}

type repositoryManagedAuthorityVerifier struct {
	inspectProject   func(context.Context, string) (project.Inspection, error)
	inspectArtifacts func(project.Inspection) (project.GoverningArtifacts, error)
	loadChangeIntent func(string) (domain.IntentSnapshot, error)
	readWorkUnit     func(string, domain.WorkSpecWorkUnitReference) (domain.WorkUnit, domain.CanonicalArtifact, error)
}

func newRepositoryManagedAuthorityVerifier() ManagedAuthorityVerifier {
	return repositoryManagedAuthorityVerifier{
		inspectProject:   project.Inspect,
		inspectArtifacts: project.InspectGoverningArtifacts,
		loadChangeIntent: func(changeDir string) (domain.IntentSnapshot, error) {
			change, err := openspecadapter.LoadChange(changeDir)
			return change.Intent, err
		},
		readWorkUnit: readCanonicalWorkUnit,
	}
}

func (verifier repositoryManagedAuthorityVerifier) Verify(
	ctx context.Context,
	repositoryRoot string,
	spec domain.WorkSpec,
) error {
	if spec.Schema != domain.WorkSpecSchemaV1 || spec.Project == nil || spec.Policy == nil || spec.Change == nil || spec.WorkUnit == nil {
		return fmt.Errorf("managed authority verification requires a complete WorkSpec v1")
	}
	inspection, err := verifier.inspectProject(ctx, repositoryRoot)
	if err != nil {
		return fmt.Errorf("inspect managed project: %w", err)
	}
	if inspection.State != project.ClaimManaged {
		return fmt.Errorf("managed project claim is %s: %s", inspection.State, inspection.Detail)
	}
	if inspection.WorktreeRoot != repositoryRoot {
		return fmt.Errorf("managed project root %q differs from the resolved WorkSpec root %q", inspection.WorktreeRoot, repositoryRoot)
	}
	if spec.Project.ArtifactRef != domain.ProjectDeclarationPath ||
		spec.Project.ID != inspection.Declaration.ProjectID ||
		spec.Project.Digest != inspection.DeclarationDigest {
		return fmt.Errorf("WorkSpec project binding does not match the committed declaration")
	}

	artifacts, err := verifier.inspectArtifacts(inspection)
	if err != nil {
		return fmt.Errorf("inspect governing artifacts: %w", err)
	}
	if !artifacts.PolicyReady() {
		return fmt.Errorf("committed project policy is not current: %s", artifacts.Policy.Detail)
	}
	if spec.Policy.ArtifactRef != inspection.Declaration.Policy.Path ||
		spec.Policy.Digest != inspection.Declaration.Policy.Digest ||
		spec.Policy.Digest != artifacts.Policy.ObservedDigest {
		return fmt.Errorf("WorkSpec policy binding does not match the committed project policy")
	}

	expectedChangePath := filepath.ToSlash(filepath.Join("openspec", "changes", spec.Change.ID))
	if spec.Change.ArtifactRef != expectedChangePath {
		return fmt.Errorf("WorkSpec change must identify the current Goalrail change at %s", expectedChangePath)
	}
	changeIntent, err := verifier.loadChangeIntent(filepath.Join(repositoryRoot, filepath.FromSlash(spec.Change.ArtifactRef)))
	if err != nil {
		return fmt.Errorf("load current Goalrail change: %w", err)
	}
	if changeIntent.Status != domain.IntentConfirmed ||
		changeIntent.ID != spec.Intent.ID ||
		changeIntent.Version != spec.Intent.Version {
		return fmt.Errorf("current Goalrail change does not compile from the exact confirmed WorkSpec intent")
	}

	workUnit, workUnitArtifact, err := verifier.readWorkUnit(repositoryRoot, *spec.WorkUnit)
	if err != nil {
		return fmt.Errorf("read managed work unit: %w", err)
	}
	if workUnitArtifact.Digest() != spec.WorkUnit.Digest || workUnit.ID != spec.WorkUnit.ID {
		return fmt.Errorf("WorkSpec work-unit binding does not match the canonical anchor")
	}
	if workUnit.Lifecycle != domain.WorkUnitOpen {
		return fmt.Errorf("work unit %s is not open", workUnit.ID)
	}
	if workUnit.ProjectID != spec.Project.ID ||
		workUnit.DeclarationDigest != spec.Project.Digest ||
		workUnit.PolicyDigest != spec.Policy.Digest {
		return fmt.Errorf("work unit does not bind the WorkSpec project and policy")
	}
	if workUnit.IntentRef != expectedIntentReference(spec.Intent) {
		return fmt.Errorf("work unit does not bind the exact confirmed WorkSpec intent")
	}
	if workUnit.ChangeRef != expectedChangeReference(*spec.Change) {
		return fmt.Errorf("work unit does not bind the current WorkSpec change")
	}
	return nil
}

func readCanonicalWorkUnit(
	repositoryRoot string,
	reference domain.WorkSpecWorkUnitReference,
) (domain.WorkUnit, domain.CanonicalArtifact, error) {
	expectedPath := filepath.ToSlash(filepath.Join(".goalrail", "work-units", string(reference.ID), "unit.json"))
	if reference.ArtifactRef != expectedPath {
		return domain.WorkUnit{}, domain.CanonicalArtifact{}, fmt.Errorf("work-unit anchor must be %s", expectedPath)
	}
	if err := validateScopedPathBoundaries(repositoryRoot, []string{reference.ArtifactRef}); err != nil {
		return domain.WorkUnit{}, domain.CanonicalArtifact{}, err
	}
	raw, err := boundedio.ReadRegularFile(
		filepath.Join(repositoryRoot, filepath.FromSlash(reference.ArtifactRef)),
		"work-unit anchor",
		domain.MaxWorkUnitBytes,
	)
	if err != nil {
		return domain.WorkUnit{}, domain.CanonicalArtifact{}, err
	}
	workUnit, err := domain.DecodeWorkUnit(bytes.NewReader(raw))
	if err != nil {
		return domain.WorkUnit{}, domain.CanonicalArtifact{}, err
	}
	artifact, err := domain.FreezeWorkUnit(workUnit)
	if err != nil {
		return domain.WorkUnit{}, domain.CanonicalArtifact{}, err
	}
	if !bytes.Equal(raw, artifact.CanonicalJSON()) {
		return domain.WorkUnit{}, domain.CanonicalArtifact{}, fmt.Errorf("work-unit anchor is not canonical JSON")
	}
	return workUnit, artifact, nil
}

func expectedIntentReference(reference domain.WorkSpecIntentReference) domain.ContentAddressedEvidenceReference {
	return domain.ContentAddressedEvidenceReference{
		ArtifactKind: "intent",
		Identity:     "intent:" + string(reference.ID),
		Version:      strconv.FormatUint(uint64(reference.Version), 10),
		Digest:       domain.SHA256Digest(reference.Digest),
		SourceRef:    "repo:root/" + strings.TrimPrefix(filepath.ToSlash(reference.ArtifactRef), "/"),
		AdapterID:    lineageAdapterID,
	}
}

func expectedChangeReference(reference domain.WorkSpecChangeReference) domain.ContentAddressedEvidenceReference {
	return domain.ContentAddressedEvidenceReference{
		ArtifactKind: "change",
		Identity:     "change:" + reference.ID,
		Version:      "1",
		Digest:       reference.Digest,
		SourceRef:    "repo:root/" + strings.TrimPrefix(filepath.ToSlash(reference.ArtifactRef), "/"),
		AdapterID:    lineageAdapterID,
	}
}
