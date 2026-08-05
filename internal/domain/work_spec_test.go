package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func validWorkSpec() WorkSpec {
	return WorkSpec{
		Schema:  WorkSpecSchemaV0,
		ID:      "work-dogfood",
		Version: 1,
		Repository: WorkSpecRepository{
			Root:         "/tmp/goalrail",
			BaseRevision: strings.Repeat("a", 40),
		},
		Intent: WorkSpecIntentReference{
			ID:          "intent-dogfood",
			Version:     3,
			ArtifactRef: "changes/dogfood/intent.md",
			Digest:      "sha256:" + strings.Repeat("b", 64),
		},
		Task:  "Implement one bounded local change.",
		Paths: []string{"internal/localrun", "cmd/gr"},
		Checks: []WorkSpecCheck{
			{ID: "test", Argv: []string{"go", "test", "./..."}},
			{ID: "vet", Argv: []string{"go", "vet", "./..."}},
		},
		StopConditions: []WorkSpecStopCondition{
			{ID: "receipt", Description: "Stop after the terminal receipt."},
			{ID: "scope", Description: "Stop on an out-of-scope change."},
		},
		Posture: PostureTrustedLocalProviderEnforcedV0,
	}
}

func validManagedWorkSpec() WorkSpec {
	spec := validWorkSpec()
	spec.Schema = WorkSpecSchemaV1
	spec.Project = &WorkSpecProjectReference{
		ID:          "project-goalrail",
		ArtifactRef: ".goalrail/project.json",
		Digest:      SHA256Digest("sha256:" + strings.Repeat("c", 64)),
	}
	spec.Policy = &WorkSpecPolicyReference{
		ArtifactRef: ".goalrail/policy.json",
		Digest:      SHA256Digest("sha256:" + strings.Repeat("d", 64)),
	}
	spec.Change = &WorkSpecChangeReference{
		ID:            "project-lineage-admission-v0",
		ArtifactRef:   "openspec/changes/project-lineage-admission-v0",
		Digest:        SHA256Digest("sha256:" + strings.Repeat("e", 64)),
		IntentID:      spec.Intent.ID,
		IntentVersion: spec.Intent.Version,
		IntentDigest:  SHA256Digest(spec.Intent.Digest),
	}
	spec.WorkUnit = &WorkSpecWorkUnitReference{
		ID:          "wu_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ArtifactRef: ".goalrail/work-units/wu_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/unit.json",
		Digest:      SHA256Digest("sha256:" + strings.Repeat("f", 64)),
	}
	spec.Posture = PostureTrustedLocalProviderEnforcedV1
	return spec
}

func TestFreezeWorkSpecCanonicalizesSetOrder(t *testing.T) {
	left := validWorkSpec()
	right := validWorkSpec()
	right.Paths[0], right.Paths[1] = right.Paths[1], right.Paths[0]
	right.Checks[0], right.Checks[1] = right.Checks[1], right.Checks[0]
	right.StopConditions[0], right.StopConditions[1] = right.StopConditions[1], right.StopConditions[0]

	leftFrozen, err := FreezeWorkSpec(left)
	if err != nil {
		t.Fatal(err)
	}
	rightFrozen, err := FreezeWorkSpec(right)
	if err != nil {
		t.Fatal(err)
	}
	if leftFrozen.Digest() != rightFrozen.Digest() {
		t.Fatalf("equivalent WorkSpecs differ: %s != %s", leftFrozen.Digest(), rightFrozen.Digest())
	}
	if !bytes.Equal(leftFrozen.CanonicalJSON(), rightFrozen.CanonicalJSON()) {
		t.Fatal("equivalent WorkSpecs did not produce identical canonical JSON")
	}

	changed := rightFrozen.Spec()
	changed.Task = "A materially different task."
	changedFrozen, err := FreezeWorkSpec(changed)
	if err != nil {
		t.Fatal(err)
	}
	if changedFrozen.Digest() == leftFrozen.Digest() {
		t.Fatal("material WorkSpec change retained the old digest")
	}
}

func TestFreezeManagedWorkSpecBindsAllAuthorityFields(t *testing.T) {
	baseline, err := FreezeWorkSpec(validManagedWorkSpec())
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]func(*WorkSpec){
		"project":   func(spec *WorkSpec) { spec.Project.Digest = SHA256Digest("sha256:" + strings.Repeat("0", 64)) },
		"policy":    func(spec *WorkSpec) { spec.Policy.Digest = SHA256Digest("sha256:" + strings.Repeat("1", 64)) },
		"change":    func(spec *WorkSpec) { spec.Change.Digest = SHA256Digest("sha256:" + strings.Repeat("2", 64)) },
		"work unit": func(spec *WorkSpec) { spec.WorkUnit.Digest = SHA256Digest("sha256:" + strings.Repeat("3", 64)) },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			changed := validManagedWorkSpec()
			mutate(&changed)
			frozen, err := FreezeWorkSpec(changed)
			if err != nil {
				t.Fatal(err)
			}
			if frozen.Digest() == baseline.Digest() {
				t.Fatal("authority change retained the old WorkSpec digest")
			}
		})
	}
}

func TestValidateManagedWorkSpecRequiresCoherentAuthority(t *testing.T) {
	tests := map[string]func(*WorkSpec){
		"missing project":        func(spec *WorkSpec) { spec.Project = nil },
		"missing policy":         func(spec *WorkSpec) { spec.Policy = nil },
		"missing change":         func(spec *WorkSpec) { spec.Change = nil },
		"missing work unit":      func(spec *WorkSpec) { spec.WorkUnit = nil },
		"change intent mismatch": func(spec *WorkSpec) { spec.Change.IntentVersion++ },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			spec := validManagedWorkSpec()
			mutate(&spec)
			if _, err := FreezeWorkSpec(spec); err == nil {
				t.Fatal("expected managed WorkSpec rejection")
			}
		})
	}
}

func TestLegacyWorkSpecCanonicalBytesDoNotGainManagedBindings(t *testing.T) {
	frozen, err := FreezeWorkSpec(validWorkSpec())
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"project"`, `"policy"`, `"change"`, `"work_unit"`} {
		if bytes.Contains(frozen.CanonicalJSON(), []byte(field)) {
			t.Fatalf("legacy WorkSpec unexpectedly contains %s", field)
		}
	}
}

func TestLegacyWorkSpecBootstrapClassificationIsBoundedAndNeverValid(t *testing.T) {
	legacy := validWorkSpecForBootstrapFixture()
	before, err := FreezeWorkSpec(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if got := ClassifyLegacyWorkSpecBootstrap(legacy, []string{".goalrail"}); got != AdmissionBootstrap {
		t.Fatalf("classification = %s, want BOOTSTRAP", got)
	}
	legacy.Paths = append(legacy.Paths, "internal/domain")
	if got := ClassifyLegacyWorkSpecBootstrap(legacy, []string{".goalrail"}); got != AdmissionInvalid {
		t.Fatalf("out-of-bound classification = %s, want INVALID", got)
	}
	after, err := FreezeWorkSpec(validWorkSpecForBootstrapFixture())
	if err != nil {
		t.Fatal(err)
	}
	if before.Digest() != after.Digest() || !bytes.Equal(before.CanonicalJSON(), after.CanonicalJSON()) {
		t.Fatal("bootstrap classification rewrote legacy authority")
	}
}

func validWorkSpecForBootstrapFixture() WorkSpec {
	spec := validWorkSpec()
	spec.Paths = []string{".goalrail/setup", ".goalrail/project.json"}
	return spec
}

func TestDecodeWorkSpecRejectsUnknownAndTrailingFields(t *testing.T) {
	spec := validWorkSpec()
	raw, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	raw = bytes.Replace(raw, []byte(`"version":1`), []byte(`"version":1,"provider":"codex"`), 1)
	if _, err := DecodeWorkSpec(bytes.NewReader(raw)); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field rejection, got %v", err)
	}

	valid, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeWorkSpec(bytes.NewReader(append(valid, []byte(` {}`)...))); err == nil {
		t.Fatal("expected trailing JSON rejection")
	}
}

func TestValidateWorkSpecRejectsUnsafeScopePayloadAndAuthority(t *testing.T) {
	tests := map[string]func(*WorkSpec){
		"path escape": func(spec *WorkSpec) {
			spec.Paths = []string{"../outside"}
		},
		"secret": func(spec *WorkSpec) {
			spec.Task = "api_key=secret-value"
		},
		"shell fragment": func(spec *WorkSpec) {
			spec.Checks = []WorkSpecCheck{{ID: "unsafe", Argv: []string{"sh", "-c", "go test ./..."}}}
		},
		"unsupported posture": func(spec *WorkSpec) {
			spec.Posture = "credentials-and-retry"
		},
		"missing checks": func(spec *WorkSpec) {
			spec.Checks = nil
		},
		"missing stops": func(spec *WorkSpec) {
			spec.StopConditions = nil
		},
		"duplicate path": func(spec *WorkSpec) {
			spec.Paths = []string{"cmd/gr", "cmd/gr"}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			spec := validWorkSpec()
			mutate(&spec)
			if _, err := FreezeWorkSpec(spec); err == nil {
				t.Fatal("expected WorkSpec rejection")
			} else {
				var validationErr *ValidationError
				if !errors.As(err, &validationErr) {
					t.Fatalf("expected ValidationError, got %T: %v", err, err)
				}
			}
		})
	}
}

func TestOpenFrozenWorkSpecRejectsChangedOrNonCanonicalContent(t *testing.T) {
	frozen, err := FreezeWorkSpec(validWorkSpec())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OpenFrozenWorkSpec(frozen.CanonicalJSON(), frozen.Digest()); err != nil {
		t.Fatal(err)
	}

	pretty := new(bytes.Buffer)
	if err := json.Indent(pretty, frozen.CanonicalJSON(), "", "  "); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenFrozenWorkSpec(pretty.Bytes(), frozen.Digest()); err == nil {
		t.Fatal("expected non-canonical stored JSON rejection")
	}
	if _, err := OpenFrozenWorkSpec(frozen.CanonicalJSON(), WorkSpecDigest("sha256:"+strings.Repeat("0", 64))); err == nil {
		t.Fatal("expected digest mismatch")
	}
}
