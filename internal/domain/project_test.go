package domain

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestProjectDeclarationCanonicalJSONAndDigest(t *testing.T) {
	declaration := validProjectDeclaration()
	frozen, err := FreezeProjectDeclaration(declaration)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"schema":"goalrail.project/v1","project_id":"prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","contract_version":"goalrail-governance-v1","policy":{"schema":"goalrail.policy/v1","path":".goalrail/policy.json","digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"bootstrap":{"schema":"goalrail.bootstrap/v1","path":".goalrail/bootstrap.md","digest":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},"setup_profile":{"schema":"goalrail.setup-profile/v1","path":".goalrail/setup-profile.json","digest":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}}`
	if got := string(frozen.CanonicalJSON()); got != want {
		t.Fatalf("canonical declaration mismatch\nwant: %s\n got: %s", want, got)
	}
	if frozen.Digest() != DigestCanonicalJSON([]byte(want)) {
		t.Fatalf("digest does not identify canonical bytes: %s", frozen.Digest())
	}
}

func TestDecodeProjectDeclarationRejectsMalformedUnknownAndOversizedJSON(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
	}{
		{name: "malformed", raw: []byte(`{"schema":`)},
		{name: "unknown schema", raw: []byte(strings.Replace(
			string(mustFreezeProject(t, validProjectDeclaration()).CanonicalJSON()),
			ProjectSchemaV1,
			"goalrail.project/v99",
			1,
		))},
		{name: "unknown field", raw: []byte(`{"schema":"goalrail.project/v1","surprise":true}`)},
		{name: "oversized", raw: bytes.Repeat([]byte("x"), MaxProjectDeclarationBytes+1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := DecodeProjectDeclaration(bytes.NewReader(test.raw)); err == nil {
				t.Fatal("expected decoding to fail")
			}
		})
	}
}

func TestNewProjectIDUsesOnlyRandomBytesAndFailsClosed(t *testing.T) {
	first, err := newProjectID(bytes.NewReader(bytes.Repeat([]byte{0x01}, projectIdentityRandomBytes)))
	if err != nil {
		t.Fatal(err)
	}
	second, err := newProjectID(bytes.NewReader(bytes.Repeat([]byte{0x02}, projectIdentityRandomBytes)))
	if err != nil {
		t.Fatal(err)
	}
	if first == second || !IsCanonicalID(string(first)) || !strings.HasPrefix(string(first), "prj_") {
		t.Fatalf("unexpected generated IDs: %q %q", first, second)
	}
	if _, err := newProjectID(errorReader{}); err == nil {
		t.Fatal("random-source failure must stop project-ID generation")
	}
}

func TestValidateProjectDeclarationRejectsUnsafeReferences(t *testing.T) {
	declaration := validProjectDeclaration()
	declaration.Policy.Path = "/tmp/policy.json"
	declaration.Bootstrap.Digest = "sha256:NOT-A-DIGEST"
	if err := ValidateProjectDeclaration(declaration); err == nil {
		t.Fatal("unsafe project references were accepted")
	}
}

func validProjectDeclaration() ProjectDeclaration {
	return ProjectDeclaration{
		Schema:          ProjectSchemaV1,
		ProjectID:       "prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ContractVersion: GovernanceContractV1,
		Policy: CommittedArtifactReference{
			Schema: PolicySchemaV1,
			Path:   DefaultProjectPolicyPath,
			Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		Bootstrap: CommittedArtifactReference{
			Schema: "goalrail.bootstrap/v1",
			Path:   DefaultProjectBootstrapPath,
			Digest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},
		SetupProfile: CommittedArtifactReference{
			Schema: SetupProfileSchemaV1,
			Path:   DefaultProjectSetupProfilePath,
			Digest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		},
	}
}

func mustFreezeProject(t *testing.T, declaration ProjectDeclaration) CanonicalArtifact {
	t.Helper()
	frozen, err := FreezeProjectDeclaration(declaration)
	if err != nil {
		t.Fatal(err)
	}
	return frozen
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, errors.New("random source unavailable")
}
