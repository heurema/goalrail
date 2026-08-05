package lineage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	openspecadapter "github.com/heurema/goalrail/internal/adapters/openspec"
	"github.com/heurema/goalrail/internal/boundedio"
	"github.com/heurema/goalrail/internal/domain"
	projectstate "github.com/heurema/goalrail/internal/project"
)

const (
	changeSnapshotMaxFiles = 512
	changeSnapshotMaxBytes = 8 << 20
	changeSnapshotFileMax  = 1 << 20
	lineageAdapterID       = "goalrail"
)

type BeginOptions struct {
	Repository        string
	ChangeID          string
	ActorRef          string
	RequiredRelations []domain.LineageRequirement
	Now               func() time.Time
	NewWorkUnitID     func() (domain.WorkUnitID, error)
}

func Begin(ctx context.Context, options BeginOptions) (BeginReceipt, error) {
	if !domain.IsCanonicalID(options.ChangeID) {
		return BeginReceipt{}, fmt.Errorf("change ID must be canonical")
	}
	if !domain.IsEvidenceReference(options.ActorRef) {
		return BeginReceipt{}, fmt.Errorf("actor reference must be a bounded provider-neutral reference")
	}
	inspection, err := projectstate.Inspect(ctx, options.Repository)
	if err != nil {
		return BeginReceipt{}, err
	}
	if inspection.State != projectstate.ClaimManaged {
		return BeginReceipt{}, fmt.Errorf("lineage begin requires one valid managed project: %s", inspection.Detail)
	}
	artifacts, err := projectstate.InspectGoverningArtifacts(inspection)
	if err != nil {
		return BeginReceipt{}, err
	}
	if !artifacts.PolicyReady() {
		return BeginReceipt{}, fmt.Errorf("lineage begin requires the committed project policy: %s", artifacts.Policy.Detail)
	}

	changeRef := filepath.ToSlash(filepath.Join("openspec", "changes", options.ChangeID))
	changeDir := filepath.Join(inspection.WorktreeRoot, filepath.FromSlash(changeRef))
	compiled, err := openspecadapter.LoadChange(changeDir)
	if err != nil {
		return BeginReceipt{}, fmt.Errorf("load current Goalrail change: %w", err)
	}
	intentRef := filepath.ToSlash(filepath.Join(changeRef, "intent.md"))
	intentRaw, err := boundedio.ReadRegularFile(
		filepath.Join(inspection.WorktreeRoot, filepath.FromSlash(intentRef)),
		"confirmed intent",
		openspecadapter.MaxResolvedIntentBytes,
	)
	if err != nil {
		return BeginReceipt{}, err
	}
	changeDigest, err := DigestChangeSnapshot(changeDir)
	if err != nil {
		return BeginReceipt{}, err
	}
	if err := inspection.Revalidate(); err != nil {
		return BeginReceipt{}, fmt.Errorf("managed project changed during lineage begin: %w", err)
	}

	now := options.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	newWorkUnitID := options.NewWorkUnitID
	if newWorkUnitID == nil {
		newWorkUnitID = randomWorkUnitID
	}
	workUnitID, err := newWorkUnitID()
	if err != nil {
		return BeginReceipt{}, err
	}
	requirements := append([]domain.LineageRequirement(nil), options.RequiredRelations...)
	if len(requirements) == 0 {
		requirements = DefaultRequirements()
	}
	intentReference := domain.ContentAddressedEvidenceReference{
		ArtifactKind: "intent",
		Identity:     "intent:" + string(compiled.Intent.ID),
		Version:      strconv.FormatUint(uint64(compiled.Intent.Version), 10),
		Digest:       domain.DigestCanonicalJSON(intentRaw),
		SourceRef:    repositorySourceRef(intentRef),
		AdapterID:    lineageAdapterID,
	}
	changeReference := domain.ContentAddressedEvidenceReference{
		ArtifactKind: "change",
		Identity:     "change:" + options.ChangeID,
		Version:      "1",
		Digest:       changeDigest,
		SourceRef:    repositorySourceRef(changeRef),
		AdapterID:    lineageAdapterID,
	}
	unit := domain.WorkUnit{
		Schema:            domain.WorkUnitSchemaV1,
		ID:                workUnitID,
		ProjectID:         inspection.Declaration.ProjectID,
		DeclarationDigest: inspection.DeclarationDigest,
		PolicyDigest:      inspection.Declaration.Policy.Digest,
		IntentRef:         intentReference,
		ChangeRef:         changeReference,
		CreatedAt:         now().UTC(),
		Lifecycle:         domain.WorkUnitOpen,
		RequiredRelations: requirements,
	}
	unitArtifact, err := domain.FreezeWorkUnit(unit)
	if err != nil {
		return BeginReceipt{}, err
	}
	unitReference := domain.ContentAddressedEvidenceReference{
		ArtifactKind: "work_unit",
		Identity:     "work-unit:" + string(workUnitID),
		Version:      "1",
		Digest:       unitArtifact.Digest(),
		SourceRef:    repositorySourceRef(filepath.ToSlash(filepath.Join(".goalrail", "work-units", string(workUnitID), "unit.json"))),
		AdapterID:    lineageAdapterID,
	}
	declarationReference := domain.ContentAddressedEvidenceReference{
		ArtifactKind: "project_declaration",
		Identity:     "project:" + string(inspection.Declaration.ProjectID),
		Version:      inspection.Declaration.ContractVersion,
		Digest:       inspection.DeclarationDigest,
		SourceRef:    repositorySourceRef(domain.ProjectDeclarationPath),
		AdapterID:    lineageAdapterID,
	}
	policyReference := domain.ContentAddressedEvidenceReference{
		ArtifactKind: "project_policy",
		Identity:     "policy:" + string(inspection.Declaration.ProjectID),
		Version:      strconv.FormatUint(uint64(artifacts.PolicyValue.Version), 10),
		Digest:       inspection.Declaration.Policy.Digest,
		SourceRef:    repositorySourceRef(inspection.Declaration.Policy.Path),
		AdapterID:    lineageAdapterID,
	}
	events, err := initialEvents(
		unit.ID,
		unitReference,
		declarationReference,
		policyReference,
		intentReference,
		changeReference,
		options.ActorRef,
		unit.CreatedAt,
	)
	if err != nil {
		return BeginReceipt{}, err
	}
	store, err := NewStore(inspection.WorktreeRoot)
	if err != nil {
		return BeginReceipt{}, err
	}
	return store.Begin(unit, events)
}

func repositorySourceRef(path string) string {
	return "repo:root/" + strings.TrimPrefix(filepath.ToSlash(path), "/")
}

func DefaultRequirements() []domain.LineageRequirement {
	return []domain.LineageRequirement{
		{Relation: domain.LineageProjectPolicy, Cardinality: domain.RelationSet},
		{Relation: domain.LineageConfirmedIntent, Cardinality: domain.RelationSingle},
		{Relation: domain.LineageChange, Cardinality: domain.RelationSingle},
		{Relation: domain.LineageWorkSpec, Cardinality: domain.RelationSingle},
		{Relation: domain.LineageRunSession, Cardinality: domain.RelationSingle},
		{Relation: domain.LineageCommit, Cardinality: domain.RelationSet},
		{Relation: domain.LineagePullRequest, Cardinality: domain.RelationSingle},
		{Relation: domain.LineageReviewIndex, Cardinality: domain.RelationSingle},
		{Relation: domain.LineageCheckSet, Cardinality: domain.RelationSingle},
		{Relation: domain.LineageTerminalReceipt, Cardinality: domain.RelationSingle},
		{Relation: domain.LineageOwnerDecision, Cardinality: domain.RelationSingle},
		{Relation: domain.LineageClosure, Cardinality: domain.RelationSingle},
	}
}

func initialEvents(
	workUnitID domain.WorkUnitID,
	unitReference domain.ContentAddressedEvidenceReference,
	declarationReference domain.ContentAddressedEvidenceReference,
	policyReference domain.ContentAddressedEvidenceReference,
	intentReference domain.ContentAddressedEvidenceReference,
	changeReference domain.ContentAddressedEvidenceReference,
	actorRef string,
	observedAt time.Time,
) ([]domain.LineageEvent, error) {
	events := []domain.LineageEvent{
		{
			Schema: domain.LineageEventSchemaV1, WorkUnitID: workUnitID,
			Relation: domain.LineageProjectPolicy, Cardinality: domain.RelationSet,
			Sources:  []domain.ContentAddressedEvidenceReference{unitReference},
			Targets:  []domain.ContentAddressedEvidenceReference{declarationReference, policyReference},
			ActorRef: actorRef, AdapterID: lineageAdapterID, ObservedAt: observedAt,
		},
		{
			Schema: domain.LineageEventSchemaV1, WorkUnitID: workUnitID,
			Relation: domain.LineageConfirmedIntent, Cardinality: domain.RelationSingle,
			Sources:  []domain.ContentAddressedEvidenceReference{unitReference},
			Targets:  []domain.ContentAddressedEvidenceReference{intentReference},
			ActorRef: actorRef, AdapterID: lineageAdapterID, ObservedAt: observedAt,
		},
		{
			Schema: domain.LineageEventSchemaV1, WorkUnitID: workUnitID,
			Relation: domain.LineageChange, Cardinality: domain.RelationSingle,
			Sources:  []domain.ContentAddressedEvidenceReference{unitReference},
			Targets:  []domain.ContentAddressedEvidenceReference{changeReference},
			ActorRef: actorRef, AdapterID: lineageAdapterID, ObservedAt: observedAt,
		},
	}
	for index := range events {
		digest, err := domain.LineageEventSemanticDigest(events[index])
		if err != nil {
			return nil, err
		}
		events[index].SemanticDigest = digest
	}
	return events, nil
}

type changeSnapshotEntry struct {
	Path   string              `json:"path"`
	Size   int64               `json:"size"`
	Digest domain.SHA256Digest `json:"digest"`
}

type changeSnapshotManifest struct {
	Schema  string                `json:"schema"`
	Entries []changeSnapshotEntry `json:"entries"`
}

func DigestChangeSnapshot(changeDir string) (domain.SHA256Digest, error) {
	var paths []string
	err := filepath.WalkDir(changeDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == changeDir {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("change snapshot cannot follow symlink %s", path)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("change snapshot contains non-regular file %s", path)
		}
		paths = append(paths, path)
		if len(paths) > changeSnapshotMaxFiles {
			return fmt.Errorf("change snapshot exceeds %d files", changeSnapshotMaxFiles)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(paths)
	manifest := changeSnapshotManifest{Schema: "goalrail.change-snapshot/v1"}
	var total int
	for _, path := range paths {
		raw, err := boundedio.ReadRegularFile(path, "Goalrail change artifact", changeSnapshotFileMax)
		if err != nil {
			return "", err
		}
		total += len(raw)
		if total > changeSnapshotMaxBytes {
			return "", fmt.Errorf("change snapshot exceeds %d bytes", changeSnapshotMaxBytes)
		}
		relative, err := filepath.Rel(changeDir, path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("change artifact escapes snapshot root")
		}
		manifest.Entries = append(manifest.Entries, changeSnapshotEntry{
			Path: filepath.ToSlash(relative), Size: int64(len(raw)), Digest: domain.DigestCanonicalJSON(raw),
		})
	}
	if len(manifest.Entries) == 0 {
		return "", errors.New("change snapshot contains no regular artifacts")
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	return domain.DigestCanonicalJSON(raw), nil
}

func randomWorkUnitID() (domain.WorkUnitID, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("create work-unit ID: %w", err)
	}
	return domain.WorkUnitID("wu_" + hex.EncodeToString(raw)), nil
}
