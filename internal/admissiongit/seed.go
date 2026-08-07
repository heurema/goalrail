package admissiongit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/heurema/goalrail/internal/admissionlocal"
	"github.com/heurema/goalrail/internal/domain"
)

// CollectPacketSeed freezes repository-owned packet identities without any
// provider access. The commit trailer is used only to find the canonical work
// unit anchor; the verifier still resolves and validates every lineage edge.
func CollectPacketSeed(ctx context.Context, repository, base, head string) (domain.AdmissionPacket, error) {
	if !fullGitOID(base) || !fullGitOID(head) || base == head {
		return domain.AdmissionPacket{}, fmt.Errorf("base and head must be distinct full lowercase Git commit IDs")
	}
	root, err := filepath.Abs(repository)
	if err != nil {
		return domain.AdmissionPacket{}, fmt.Errorf("resolve repository path: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return domain.AdmissionPacket{}, fmt.Errorf("resolve repository path: %w", err)
	}
	for _, revision := range []string{base, head} {
		if _, err := runGit(ctx, root, 1024, "cat-file", "-e", revision+"^{commit}"); err != nil {
			return domain.AdmissionPacket{}, fmt.Errorf("resolve immutable commit %s: %w", revision, err)
		}
	}

	workUnitID, err := collectTrailerIndex(ctx, root, base, head)
	if err != nil {
		return domain.AdmissionPacket{}, err
	}
	unitPath := path.Join(".goalrail", "work-units", string(workUnitID), "unit.json")
	unitRaw, err := readGitObject(ctx, root, head, unitPath, domain.MaxWorkUnitBytes)
	if err != nil {
		return domain.AdmissionPacket{}, fmt.Errorf("read trailer-indexed work unit: %w", err)
	}
	unit, err := domain.DecodeWorkUnit(bytes.NewReader(unitRaw))
	if err != nil {
		return domain.AdmissionPacket{}, fmt.Errorf("decode trailer-indexed work unit: %w", err)
	}
	unitArtifact, err := domain.FreezeWorkUnit(unit)
	if err != nil || !bytes.Equal(unitRaw, unitArtifact.CanonicalJSON()) || unit.ID != workUnitID {
		return domain.AdmissionPacket{}, fmt.Errorf("trailer-indexed work unit is not canonical")
	}

	declaration, declarationDigest, _, policyDigest, authorityErr := readGovernance(ctx, root, base)
	if errors.Is(authorityErr, errDeclarationMissing) {
		declaration, declarationDigest, _, policyDigest, authorityErr = readGovernance(ctx, root, head)
	}
	if authorityErr != nil {
		return domain.AdmissionPacket{}, fmt.Errorf("read packet authority: %w", authorityErr)
	}
	packet := domain.AdmissionPacket{
		Schema:            domain.AdmissionPacketSchemaV1,
		ProjectID:         declaration.ProjectID,
		DeclarationDigest: declarationDigest,
		PolicyDigest:      policyDigest,
		BaseRevision:      base,
		HeadRevision:      head,
		WorkUnitRef: domain.ContentAddressedEvidenceReference{
			ArtifactKind: "work_unit",
			Identity:     "work-unit:" + string(workUnitID),
			Version:      "1",
			Digest:       unitArtifact.Digest(),
			SourceRef:    "repo:root/" + unitPath,
			AdapterID:    "goalrail",
		},
		Evidence:   []domain.ContentAddressedEvidenceReference{},
		Provenance: []domain.AdmissionProviderProvenance{},
	}
	if err := domain.ValidateAdmissionPacket(packet); err != nil {
		return domain.AdmissionPacket{}, fmt.Errorf("validate repository packet seed: %w", err)
	}
	return packet, nil
}

func collectTrailerIndex(ctx context.Context, repository, base, head string) (domain.WorkUnitID, error) {
	commits, _, err := collectCommits(ctx, repository, base, head)
	if err != nil {
		return "", err
	}
	var selected domain.WorkUnitID
	for _, revision := range commits {
		message, err := runGit(ctx, repository, 128<<10, "show", "-s", "--format=%B", revision)
		if err != nil {
			return "", fmt.Errorf("read commit message %s: %w", revision, err)
		}
		workUnitID, err := admissionlocal.ValidateCommitMessage(bytes.NewReader(message))
		if errors.Is(err, admissionlocal.ErrTrailerMissing) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("validate commit message %s: %w", revision, err)
		}
		if selected != "" && selected != workUnitID {
			return "", errors.New("commit range contains conflicting Goalrail work-unit trailers")
		}
		selected = workUnitID
	}
	if selected == "" {
		return "", admissionlocal.ErrTrailerMissing
	}
	if !domain.IsCanonicalID(string(selected)) || !strings.HasPrefix(string(selected), "wu_") {
		return "", errors.New("commit range work-unit index is not canonical")
	}
	return selected, nil
}
