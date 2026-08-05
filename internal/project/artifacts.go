package project

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/heurema/goalrail/internal/boundedio"
	"github.com/heurema/goalrail/internal/domain"
)

const MaxBootstrapBytes = 128 << 10

type ArtifactState string

const (
	ArtifactCurrent  ArtifactState = "current"
	ArtifactMissing  ArtifactState = "missing"
	ArtifactMismatch ArtifactState = "digest_mismatch"
	ArtifactInvalid  ArtifactState = "invalid"
)

// ArtifactFinding reports the identity of one declaration-bound repository
// artifact. Current means both canonical semantics and exact referenced bytes;
// a digest match alone is never promoted over a malformed schema.
type ArtifactFinding struct {
	Path           string              `json:"path"`
	Schema         string              `json:"schema"`
	Ownership      Ownership           `json:"ownership"`
	State          ArtifactState       `json:"state"`
	ExpectedDigest domain.SHA256Digest `json:"expected_digest"`
	ObservedDigest domain.SHA256Digest `json:"observed_digest,omitempty"`
	Detail         string              `json:"detail,omitempty"`
}

type GoverningArtifacts struct {
	Policy       ArtifactFinding      `json:"policy"`
	Bootstrap    ArtifactFinding      `json:"bootstrap"`
	SetupProfile ArtifactFinding      `json:"setup_profile"`
	PolicyValue  domain.ProjectPolicy `json:"-"`
	SetupValue   domain.SetupProfile  `json:"-"`
}

func (artifacts GoverningArtifacts) PolicyReady() bool {
	return artifacts.Policy.State == ArtifactCurrent
}

func (artifacts GoverningArtifacts) SetupReady() bool {
	return artifacts.SetupProfile.State == ArtifactCurrent
}

// InspectGoverningArtifacts reads only the three paths selected by a valid
// committed declaration. It neither searches for alternatives nor repairs a
// mismatch, so repository-owned policy remains the source of truth.
func InspectGoverningArtifacts(inspection Inspection) (GoverningArtifacts, error) {
	if inspection.State == ClaimUnmanaged {
		return GoverningArtifacts{}, ErrClaimNotManaged
	}
	if inspection.State != ClaimManaged {
		return GoverningArtifacts{}, ErrClaimInvalid
	}

	policyFinding, policyRaw := inspectReferencedArtifact(
		inspection,
		inspection.Declaration.Policy,
		OwnershipRepository,
		domain.MaxPolicyBytes,
	)
	bootstrapFinding, _ := inspectReferencedArtifact(
		inspection,
		inspection.Declaration.Bootstrap,
		OwnershipCanon,
		MaxBootstrapBytes,
	)
	setupFinding, setupRaw := inspectReferencedArtifact(
		inspection,
		inspection.Declaration.SetupProfile,
		OwnershipRepository,
		domain.MaxSetupProfileBytes,
	)

	result := GoverningArtifacts{
		Policy:       policyFinding,
		Bootstrap:    bootstrapFinding,
		SetupProfile: setupFinding,
	}
	if policyFinding.State == ArtifactCurrent {
		policy, err := domain.DecodeProjectPolicy(bytes.NewReader(policyRaw))
		if err != nil {
			result.Policy.State, result.Policy.Detail = ArtifactInvalid, err.Error()
		} else if policy.ProjectID != inspection.Declaration.ProjectID {
			result.Policy.State, result.Policy.Detail = ArtifactInvalid, "policy project ID differs from the declaration"
		} else if frozen, freezeErr := domain.FreezeProjectPolicy(policy); freezeErr != nil {
			result.Policy.State, result.Policy.Detail = ArtifactInvalid, freezeErr.Error()
		} else if !bytes.Equal(policyRaw, frozen.CanonicalJSON()) {
			result.Policy.State, result.Policy.Detail = ArtifactInvalid, "policy bytes are not canonical JSON"
		} else {
			result.PolicyValue = policy
		}
	}
	if setupFinding.State == ArtifactCurrent {
		profile, err := domain.DecodeSetupProfile(bytes.NewReader(setupRaw))
		if err != nil {
			result.SetupProfile.State, result.SetupProfile.Detail = ArtifactInvalid, err.Error()
		} else if profile.ProjectID != inspection.Declaration.ProjectID {
			result.SetupProfile.State, result.SetupProfile.Detail = ArtifactInvalid, "setup profile project ID differs from the declaration"
		} else if frozen, freezeErr := domain.FreezeSetupProfile(profile); freezeErr != nil {
			result.SetupProfile.State, result.SetupProfile.Detail = ArtifactInvalid, freezeErr.Error()
		} else if !bytes.Equal(setupRaw, frozen.CanonicalJSON()) {
			result.SetupProfile.State, result.SetupProfile.Detail = ArtifactInvalid, "setup profile bytes are not canonical JSON"
		} else {
			result.SetupValue = profile
		}
	}
	return result, nil
}

func inspectReferencedArtifact(
	inspection Inspection,
	reference domain.CommittedArtifactReference,
	ownership Ownership,
	limit int,
) (ArtifactFinding, []byte) {
	finding := ArtifactFinding{
		Path:           reference.Path,
		Schema:         reference.Schema,
		Ownership:      ownership,
		State:          ArtifactInvalid,
		ExpectedDigest: reference.Digest,
	}
	absolute := filepath.Join(inspection.WorktreeRoot, filepath.FromSlash(reference.Path))
	info, err := os.Lstat(absolute)
	if errors.Is(err, fs.ErrNotExist) {
		finding.State = ArtifactMissing
		finding.Detail = "referenced artifact is absent"
		return finding, nil
	}
	if err != nil {
		finding.Detail = err.Error()
		return finding, nil
	}
	if err := ensureSafeRegularPath(inspection.WorktreeRoot, absolute, info); err != nil {
		finding.Detail = err.Error()
		return finding, nil
	}
	raw, opened, err := boundedio.ReadRegularFileWithInfo(absolute, reference.Path, limit)
	if err != nil {
		finding.Detail = err.Error()
		return finding, nil
	}
	after, err := os.Lstat(absolute)
	if err != nil || !sameFileSnapshot(info, opened) || !sameFileSnapshot(opened, after) {
		if err == nil {
			err = errors.New("artifact changed while it was read")
		}
		finding.Detail = fmt.Sprintf("artifact identity is unstable: %v", err)
		return finding, nil
	}
	finding.ObservedDigest = domain.DigestCanonicalJSON(raw)
	if finding.ObservedDigest != reference.Digest {
		finding.State = ArtifactMismatch
		finding.Detail = "artifact bytes do not match the committed declaration reference"
		return finding, raw
	}
	finding.State = ArtifactCurrent
	return finding, raw
}
