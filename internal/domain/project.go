package domain

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"io"
	"strings"
)

type ProjectID string

const (
	ProjectSchemaV1                = "goalrail.project/v1"
	BootstrapSchemaV1              = "goalrail.bootstrap/v1"
	ProjectDeclarationPath         = ".goalrail/project.json"
	GovernanceContractV1           = "goalrail-governance-v1"
	DefaultProjectPolicyPath       = ".goalrail/policy.json"
	DefaultProjectBootstrapPath    = ".goalrail/bootstrap.md"
	DefaultProjectSetupProfilePath = ".goalrail/setup-profile.json"
	MaxProjectDeclarationBytes     = 16 << 10
	MaxProjectArtifactPathBytes    = 512
	projectIdentityRandomBytes     = 20
)

// CommittedArtifactReference binds a repository-relative committed path to
// the exact bytes that govern the project.
type CommittedArtifactReference struct {
	Schema string       `json:"schema"`
	Path   string       `json:"path"`
	Digest SHA256Digest `json:"digest"`
}

// ProjectDeclaration is the committed, checkout-independent project identity.
// It intentionally contains no user, provider, credential, or local readiness
// fields.
type ProjectDeclaration struct {
	Schema          string                     `json:"schema"`
	ProjectID       ProjectID                  `json:"project_id"`
	ContractVersion string                     `json:"contract_version"`
	Policy          CommittedArtifactReference `json:"policy"`
	Bootstrap       CommittedArtifactReference `json:"bootstrap"`
	// SetupProfile binds both the planning compiler and the prepared shared-
	// admission adapter declared by goalrail.setup-profile/v1.
	SetupProfile CommittedArtifactReference `json:"setup_profile"`
}

// NewProjectID creates an anonymous project identity from crypto/rand. No
// machine, user, remote, or checkout identity contributes to the value.
func NewProjectID() (ProjectID, error) {
	return newProjectID(rand.Reader)
}

func newProjectID(random io.Reader) (ProjectID, error) {
	raw := make([]byte, projectIdentityRandomBytes)
	if _, err := io.ReadFull(random, raw); err != nil {
		return "", fmt.Errorf("generate project ID: %w", err)
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
	return ProjectID("prj_" + strings.ToLower(encoded)), nil
}

func DecodeProjectDeclaration(reader io.Reader) (ProjectDeclaration, error) {
	declaration, err := decodeStrictBoundedJSON[ProjectDeclaration](
		reader,
		MaxProjectDeclarationBytes,
		"project declaration",
	)
	if err != nil {
		return ProjectDeclaration{}, err
	}
	if err := ValidateProjectDeclaration(declaration); err != nil {
		return ProjectDeclaration{}, err
	}
	return declaration, nil
}

func FreezeProjectDeclaration(declaration ProjectDeclaration) (CanonicalArtifact, error) {
	if err := ValidateProjectDeclaration(declaration); err != nil {
		return CanonicalArtifact{}, err
	}
	return newCanonicalArtifact(declaration)
}

func ValidateProjectDeclaration(declaration ProjectDeclaration) error {
	v := &validator{}
	if declaration.Schema != ProjectSchemaV1 {
		v.add("project.schema.invalid", "schema", "unsupported project declaration schema")
	}
	if !IsCanonicalID(string(declaration.ProjectID)) || !strings.HasPrefix(string(declaration.ProjectID), "prj_") {
		v.add("project.id.invalid", "project_id", "project ID must be a canonical anonymous Goalrail project ID")
	}
	if declaration.ContractVersion != GovernanceContractV1 {
		v.add("project.contract.invalid", "contract_version", "unsupported governance contract version")
	}
	validateCommittedArtifactReference(v, "policy", declaration.Policy)
	validateCommittedArtifactReference(v, "bootstrap", declaration.Bootstrap)
	validateCommittedArtifactReference(v, "setup_profile", declaration.SetupProfile)
	validateExpectedProjectReference(v, "policy", declaration.Policy, PolicySchemaV1, DefaultProjectPolicyPath)
	validateExpectedProjectReference(v, "bootstrap", declaration.Bootstrap, BootstrapSchemaV1, DefaultProjectBootstrapPath)
	validateExpectedProjectReference(v, "setup_profile", declaration.SetupProfile, SetupProfileSchemaV1, DefaultProjectSetupProfilePath)
	return v.result()
}

func validateExpectedProjectReference(v *validator, field string, reference CommittedArtifactReference, schema, path string) {
	if reference.Schema != schema {
		v.add("project.reference.schema_mismatch", field+".schema", "artifact reference does not name the required schema")
	}
	if reference.Path != path {
		v.add("project.reference.path_mismatch", field+".path", "artifact reference does not name the canonical project path")
	}
}

func validateCommittedArtifactReference(v *validator, path string, reference CommittedArtifactReference) {
	if reference.Schema == "" || len(reference.Schema) > MaxReferenceBytes || hasSecretShapedContent(reference.Schema) {
		v.add("project.reference.schema_invalid", path+".schema", "artifact schema must be bounded and non-secret")
	}
	if len(reference.Path) > MaxProjectArtifactPathBytes {
		v.add("project.reference.path_too_large", path+".path", "artifact path exceeds the project bound")
	} else if err := validateRepositoryRelativePath(reference.Path); err != nil {
		v.add("project.reference.path_invalid", path+".path", err.Error())
	} else if reference.Path == "." || reference.Path != normalizeRelativePath(reference.Path) {
		v.add("project.reference.path_noncanonical", path+".path", "artifact path must be a normalized repository-relative path")
	}
	if !IsSHA256Digest(reference.Digest) {
		v.add("project.reference.digest_invalid", path+".digest", "artifact digest must be a complete lowercase SHA-256 reference")
	}
}
