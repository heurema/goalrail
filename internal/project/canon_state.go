package project

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/heurema/goalrail/internal/domain"
)

type CanonPathState string

const (
	CanonPathCurrent CanonPathState = "current"
	CanonPathMissing CanonPathState = "missing"
	CanonPathBehind  CanonPathState = "behind"
	CanonPathDrift   CanonPathState = "drift"
)

type CanonPathFinding struct {
	Path           string              `json:"path"`
	Ownership      Ownership           `json:"ownership"`
	State          CanonPathState      `json:"state"`
	ExpectedDigest domain.SHA256Digest `json:"expected_digest"`
	ObservedDigest domain.SHA256Digest `json:"observed_digest,omitempty"`
	Detail         string              `json:"detail,omitempty"`
}

type CanonState struct {
	Canon         domain.SHA256Digest `json:"canon"`
	Current       bool                `json:"current"`
	Files         []CanonPathFinding  `json:"files"`
	ManagedBlocks []CanonPathFinding  `json:"managed_blocks"`
}

// InspectCurrentCanon compares a managed worktree with the canon carried by
// this binary. It does not interpret repository-owned policy or setup values;
// InspectGoverningArtifacts owns that independent question.
func InspectCurrentCanon(inspection Inspection) (CanonState, error) {
	if inspection.State == ClaimUnmanaged {
		return CanonState{}, ErrClaimNotManaged
	}
	if inspection.State != ClaimManaged {
		return CanonState{}, ErrClaimInvalid
	}
	canon, err := CurrentProjectCanon()
	if err != nil {
		return CanonState{}, err
	}
	rendered, err := RenderProjectCanon(inspection.Declaration.ProjectID)
	if err != nil {
		return CanonState{}, err
	}
	state := CanonState{Canon: canon.ID, Current: true}
	for _, file := range rendered {
		if file.Ownership != OwnershipCanon {
			continue
		}
		raw, exists, err := readProjectUpdatePath(inspection.WorktreeRoot, file.Path)
		if err != nil {
			return CanonState{}, err
		}
		finding := CanonPathFinding{
			Path: file.Path, Ownership: OwnershipCanon, ExpectedDigest: file.Digest,
		}
		switch {
		case !exists:
			finding.State = CanonPathMissing
			finding.Detail = "canon-owned file is absent"
		case bytes.Equal(raw, file.Content):
			finding.State = CanonPathCurrent
			finding.ObservedDigest = file.Digest
		default:
			finding.State = CanonPathDrift
			finding.ObservedDigest = domain.DigestCanonicalJSON(raw)
			finding.Detail = "bytes differ from every retained project canon"
		}
		if finding.State != CanonPathCurrent {
			state.Current = false
		}
		state.Files = append(state.Files, finding)
	}

	desired := contentFor(rendered, AgentsSnippetPath)
	for _, path := range []string{AgentsRootPath, ClaudeRootPath} {
		raw, exists, err := readProjectUpdatePath(inspection.WorktreeRoot, path)
		if err != nil {
			return CanonState{}, err
		}
		finding := CanonPathFinding{
			Path: path, Ownership: OwnershipManaged,
			ExpectedDigest: domain.DigestCanonicalJSON(desired),
		}
		plan := PlanManagedBlock(path, raw, exists, desired)
		switch plan.Action {
		case ManagedBlockUnchanged:
			finding.State = CanonPathCurrent
			finding.ObservedDigest = domain.DigestCanonicalJSON(desired)
		case ManagedBlockCreated:
			finding.State = CanonPathMissing
			finding.Detail = "owner instruction file is absent"
		default:
			finding.State = CanonPathDrift
			finding.Detail = plan.Reason
			if exists {
				finding.ObservedDigest = domain.DigestCanonicalJSON(raw)
			}
		}
		if finding.State != CanonPathCurrent {
			state.Current = false
		}
		state.ManagedBlocks = append(state.ManagedBlocks, finding)
	}
	sort.Slice(state.Files, func(i, j int) bool { return state.Files[i].Path < state.Files[j].Path })
	sort.Slice(state.ManagedBlocks, func(i, j int) bool { return state.ManagedBlocks[i].Path < state.ManagedBlocks[j].Path })
	if len(state.Files) == 0 {
		return CanonState{}, fmt.Errorf("embedded project canon has no whole-file paths")
	}
	return state, nil
}
