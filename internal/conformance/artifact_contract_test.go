package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/adapters/openspec"
	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/domain"
)

const (
	corpusRoot       = "testdata/artifact-contract-v1"
	manifestPath     = corpusRoot + "/manifest.json"
	manifestSchema   = "goalrail.artifact-conformance-corpus/v1"
	laneCompiler     = "compiler"
	laneLocalRun     = "local-run"
	laneAmbient      = "ambient"
	invalidConfirmed = "INVALID_CONFIRMED_INTENT"
)

type corpusManifest struct {
	Schema string       `json:"schema"`
	Cases  []corpusCase `json:"cases"`
}

type corpusCase struct {
	ID                   string            `json:"id"`
	Provenance           string            `json:"provenance"`
	Mode                 string            `json:"mode"`
	Fixtures             []fixtureRecord   `json:"fixtures"`
	Lanes                []string          `json:"lanes"`
	IntentRef            *intentReference  `json:"intent_ref,omitempty"`
	AmbientCoexistsValid bool              `json:"ambient_coexists_valid,omitempty"`
	Expectation          corpusExpectation `json:"expectation"`
}

type fixtureRecord struct {
	Role   string `json:"role"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type intentReference struct {
	ID      string `json:"id"`
	Version uint32 `json:"version"`
}

type corpusExpectation struct {
	Semantic   *semanticExpectation   `json:"semantic,omitempty"`
	Diagnostic *diagnosticExpectation `json:"diagnostic,omitempty"`
}

type semanticExpectation struct {
	ContractMode        string  `json:"contract_mode"`
	ContextID           string  `json:"context_id"`
	ContextVersion      uint32  `json:"context_version"`
	ContextPrevious     uint32  `json:"context_previous_version"`
	ContextItems        int     `json:"context_items"`
	FirstRecipe         *string `json:"first_recipe,omitempty"`
	FirstRecipeNonempty bool    `json:"first_recipe_nonempty,omitempty"`
	IntentID            string  `json:"intent_id"`
	IntentVersion       uint32  `json:"intent_version"`
	IntentPrevious      uint32  `json:"intent_previous_version"`
	DesiredOutcomes     int     `json:"desired_outcomes"`
	NonGoals            int     `json:"non_goals"`
	SuccessSignals      int     `json:"success_signals"`
	ResolvedID          string  `json:"resolved_id,omitempty"`
	ResolvedDigest      string  `json:"resolved_digest,omitempty"`
	Disposition         string  `json:"disposition,omitempty"`
}

type diagnosticExpectation struct {
	Code           string `json:"code"`
	Path           string `json:"path"`
	ArtifactKind   string `json:"artifact_kind"`
	ContractMode   string `json:"contract_mode"`
	FieldOrSection string `json:"field_or_section"`
	Observation    string `json:"observation"`
	Expectation    string `json:"expectation"`
	RepairHint     string `json:"repair_hint"`
}

type laneResult struct {
	Case string
	Lane string
}

func TestArtifactConformanceCorpus(t *testing.T) {
	manifest, fixtures := loadVerifiedCorpus(t)
	first := executeCorpus(t, manifest, fixtures)
	second := executeCorpus(t, manifest, fixtures)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("corpus matrix changed between runs:\nfirst=%+v\nsecond=%+v", first, second)
	}
	if err := validateExecutedMatrix(manifest, first); err != nil {
		t.Fatal(err)
	}
}

func TestCorpusManifestRejectsInvalidOracleDefinitions(t *testing.T) {
	base, err := decodeManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func(*corpusManifest)
	}{
		{name: "duplicate case ID", mutate: func(value *corpusManifest) { value.Cases[1].ID = value.Cases[0].ID }},
		{name: "unknown mode", mutate: func(value *corpusManifest) { value.Cases[0].Mode = "guessed" }},
		{name: "unknown lane", mutate: func(value *corpusManifest) { value.Cases[0].Lanes[0] = "copied-result" }},
		{name: "unsafe fixture path", mutate: func(value *corpusManifest) { value.Cases[0].Fixtures[0].Path = "../escape.md" }},
		{name: "missing provenance", mutate: func(value *corpusManifest) { value.Cases[0].Provenance = "" }},
		{name: "missing expectation", mutate: func(value *corpusManifest) { value.Cases[0].Expectation = corpusExpectation{} }},
		{name: "two expectations", mutate: func(value *corpusManifest) {
			value.Cases[0].Expectation.Semantic = base.Cases[2].Expectation.Semantic
		}},
		{name: "non-deterministic case order", mutate: func(value *corpusManifest) {
			value.Cases[0], value.Cases[1] = value.Cases[1], value.Cases[0]
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := cloneManifest(t, base)
			test.mutate(&candidate)
			if err := validateManifest(candidate); err == nil {
				t.Fatal("invalid manifest was accepted")
			}
		})
	}

	tampered := cloneManifest(t, base)
	tampered.Cases[0].Fixtures[0].SHA256 = "sha256:" + strings.Repeat("0", 64)
	if _, err := verifyFixtureDigests(tampered); err == nil {
		t.Fatal("tampered fixture digest was accepted")
	}

	fixtures, err := verifyFixtureDigests(base)
	if err != nil {
		t.Fatal(err)
	}
	_ = fixtures
	matrix := expectedMatrix(base)
	if err := validateExecutedMatrix(base, matrix[:len(matrix)-1]); err == nil {
		t.Fatal("a skipped declared lane was accepted")
	}
}

func loadVerifiedCorpus(t *testing.T) (corpusManifest, map[string][]byte) {
	t.Helper()
	manifest, err := decodeManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateManifest(manifest); err != nil {
		t.Fatalf("validate corpus manifest: %v", err)
	}
	fixtures, err := verifyFixtureDigests(manifest)
	if err != nil {
		t.Fatalf("verify corpus fixtures before lanes: %v", err)
	}
	return manifest, fixtures
}

func decodeManifest(path string) (corpusManifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return corpusManifest{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var manifest corpusManifest
	if err := decoder.Decode(&manifest); err != nil {
		return corpusManifest{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return corpusManifest{}, fmt.Errorf("manifest has trailing JSON")
	}
	return manifest, nil
}

func validateManifest(manifest corpusManifest) error {
	if manifest.Schema != manifestSchema {
		return fmt.Errorf("unknown manifest schema %q", manifest.Schema)
	}
	if len(manifest.Cases) == 0 {
		return fmt.Errorf("manifest has no cases")
	}
	seenCases := make(map[string]struct{}, len(manifest.Cases))
	previousID := ""
	for _, current := range manifest.Cases {
		if current.ID == "" || current.ID <= previousID {
			return fmt.Errorf("case IDs must be unique and strictly sorted: %q after %q", current.ID, previousID)
		}
		if _, exists := seenCases[current.ID]; exists {
			return fmt.Errorf("duplicate case ID %q", current.ID)
		}
		seenCases[current.ID] = struct{}{}
		previousID = current.ID
		if strings.TrimSpace(current.Provenance) == "" {
			return fmt.Errorf("case %s has no provenance", current.ID)
		}
		if current.Mode != "contract-v1" && current.Mode != "legacy-v0" {
			return fmt.Errorf("case %s has unknown mode %q", current.ID, current.Mode)
		}
		if (current.Expectation.Semantic == nil) == (current.Expectation.Diagnostic == nil) {
			return fmt.Errorf("case %s must have exactly one expectation", current.ID)
		}
		if len(current.Fixtures) == 0 || len(current.Lanes) == 0 {
			return fmt.Errorf("case %s has no fixtures or lanes", current.ID)
		}
		roles := make(map[string]struct{}, len(current.Fixtures))
		previousPath := ""
		for _, fixture := range current.Fixtures {
			if fixture.Role == "" || fixture.Path == "" || !validSHA256(fixture.SHA256) {
				return fmt.Errorf("case %s has an incomplete fixture record", current.ID)
			}
			if _, duplicate := roles[fixture.Role]; duplicate {
				return fmt.Errorf("case %s repeats fixture role %q", current.ID, fixture.Role)
			}
			roles[fixture.Role] = struct{}{}
			if fixture.Path <= previousPath {
				return fmt.Errorf("case %s fixture paths are not strictly sorted", current.ID)
			}
			previousPath = fixture.Path
			if !safeFixturePath(fixture.Path) {
				return fmt.Errorf("case %s fixture path escapes: %q", current.ID, fixture.Path)
			}
		}
		for _, required := range []string{"context", "intent"} {
			if _, exists := roles[required]; !exists {
				return fmt.Errorf("case %s has no %s fixture", current.ID, required)
			}
		}
		previousLane := -1
		seenLanes := make(map[string]struct{}, len(current.Lanes))
		for _, lane := range current.Lanes {
			index := laneIndex(lane)
			if index < 0 || index <= previousLane {
				return fmt.Errorf("case %s has unknown, repeated, or unordered lane %q", current.ID, lane)
			}
			seenLanes[lane] = struct{}{}
			previousLane = index
		}
		if _, compiler := seenLanes[laneCompiler]; compiler {
			if _, exists := roles["proposal"]; !exists {
				return fmt.Errorf("compiler case %s has no fixed Proposal fixture", current.ID)
			}
		}
		if _, local := seenLanes[laneLocalRun]; local && current.IntentRef == nil {
			return fmt.Errorf("local-run case %s has no pinned intent reference", current.ID)
		}
		if current.AmbientCoexistsValid {
			for _, required := range []string{"good-context", "good-intent"} {
				if _, exists := roles[required]; !exists {
					return fmt.Errorf("ambient coexistence case %s has no %s fixture", current.ID, required)
				}
			}
		}
	}
	return nil
}

func verifyFixtureDigests(manifest corpusManifest) (map[string][]byte, error) {
	result := make(map[string][]byte)
	for _, current := range manifest.Cases {
		for _, fixture := range current.Fixtures {
			if existing, ok := result[fixture.Path]; ok {
				if digest(existing) != fixture.SHA256 {
					return nil, fmt.Errorf("fixture %s repeats with a different digest", fixture.Path)
				}
				continue
			}
			raw, err := os.ReadFile(filepath.Join(corpusRoot, filepath.FromSlash(fixture.Path)))
			if err != nil {
				return nil, fmt.Errorf("read fixture %s: %w", fixture.Path, err)
			}
			if got := digest(raw); got != fixture.SHA256 {
				return nil, fmt.Errorf("fixture %s digest = %s, want %s", fixture.Path, got, fixture.SHA256)
			}
			result[fixture.Path] = raw
		}
	}
	return result, nil
}

func executeCorpus(t *testing.T, manifest corpusManifest, fixtures map[string][]byte) []laneResult {
	t.Helper()
	results := make([]laneResult, 0)
	for _, current := range manifest.Cases {
		for _, lane := range current.Lanes {
			t.Run(current.ID+"/"+lane, func(t *testing.T) {
				switch lane {
				case laneCompiler:
					runCompilerLane(t, current, fixtures)
				case laneLocalRun:
					runLocalRunLane(t, current, fixtures)
				case laneAmbient:
					runAmbientLane(t, current, fixtures)
				default:
					t.Fatalf("unimplemented lane %q", lane)
				}
			})
			results = append(results, laneResult{Case: current.ID, Lane: lane})
		}
	}
	return results
}

func runCompilerLane(t *testing.T, current corpusCase, fixtures map[string][]byte) {
	t.Helper()
	changeDir := t.TempDir()
	materializeRoles(t, changeDir, current, fixtures, map[string]string{
		"context": "context.md", "intent": "intent.md", "proposal": "proposal.md",
	})
	compiled, err := openspec.LoadChange(changeDir)
	if current.Expectation.Diagnostic != nil {
		assertDiagnostic(t, err, *current.Expectation.Diagnostic)
		return
	}
	if err != nil {
		t.Fatalf("LoadChange: %v", err)
	}
	if compiled.Pair == nil {
		t.Fatal("compiler returned no ConformedPair")
	}
	assertSemantic(t, *compiled.Pair, *current.Expectation.Semantic)
}

func runLocalRunLane(t *testing.T, current corpusCase, fixtures map[string][]byte) {
	t.Helper()
	root := t.TempDir()
	changeDir := filepath.Join(root, "openspec", "changes", current.ID)
	materializeRoles(t, changeDir, current, fixtures, map[string]string{"context": "context.md", "intent": "intent.md"})
	rawIntent := fixtureForRole(t, current, fixtures, "intent")
	reference := domain.WorkSpecIntentReference{
		ID:          domain.IntentID(current.IntentRef.ID),
		Version:     current.IntentRef.Version,
		ArtifactRef: filepath.ToSlash(filepath.Join("openspec", "changes", current.ID, "intent.md")),
		Digest:      digest(rawIntent),
	}
	resolved, err := (openspec.IntentResolver{}).Resolve(root, reference)
	if current.Expectation.Diagnostic != nil {
		assertDiagnostic(t, err, *current.Expectation.Diagnostic)
		return
	}
	if err != nil {
		t.Fatalf("IntentResolver.Resolve: %v", err)
	}
	if resolved.Pair == nil {
		t.Fatal("local-run returned no ConformedPair")
	}
	assertSemantic(t, *resolved.Pair, *current.Expectation.Semantic)
}

func runAmbientLane(t *testing.T, current corpusCase, fixtures map[string][]byte) {
	t.Helper()
	root := t.TempDir()
	changeDir := filepath.Join(root, "openspec", "changes", current.ID)
	materializeRoles(t, changeDir, current, fixtures, map[string]string{"context": "context.md", "intent": "intent.md"})
	if current.AmbientCoexistsValid {
		goodDir := filepath.Join(root, "openspec", "changes", "good-change")
		materializeRoles(t, goodDir, current, fixtures, map[string]string{
			"good-context": "context.md", "good-intent": "intent.md",
		})
	}
	resolution := (ambient.OpenSpecIntents{}).ActiveConfirmedIntent(root)
	assertAmbientResolution(t, current, resolution)
	materializeUnrelatedAmbientChanges(t, root)
	after := (ambient.OpenSpecIntents{}).ActiveConfirmedIntent(root)
	beforeJSON, beforeErr := json.Marshal(resolution)
	afterJSON, afterErr := json.Marshal(after)
	if beforeErr != nil || afterErr != nil || !reflect.DeepEqual(beforeJSON, afterJSON) {
		t.Fatalf("ambient result depends on unrelated active/archive changes:\nbefore=%s\nafter=%s\nerrors=%v %v", beforeJSON, afterJSON, beforeErr, afterErr)
	}
}

func assertAmbientResolution(t *testing.T, current corpusCase, resolution ambient.IntentResolution) {
	t.Helper()
	if current.Expectation.Diagnostic != nil {
		if resolution.Reference != nil || resolution.UnboundReason != invalidConfirmed {
			t.Fatalf("ambient resolution = %+v", resolution)
		}
		if len(resolution.BindingDiagnostics) != 1 || resolution.BindingDiagnostics[0].Change != current.ID {
			t.Fatalf("ambient diagnostics = %+v", resolution.BindingDiagnostics)
		}
		assertDiagnosticValue(t, resolution.BindingDiagnostics[0].Diagnostic, *current.Expectation.Diagnostic)
		return
	}
	if resolution.Reference == nil || resolution.UnboundReason != "" || len(resolution.BindingDiagnostics) != 0 {
		t.Fatalf("ambient resolution = %+v", resolution)
	}
	expected := current.Expectation.Semantic
	if string(resolution.Reference.ID) != expected.IntentID || resolution.Reference.Version != expected.IntentVersion || resolution.Reference.Change != current.ID {
		t.Fatalf("ambient reference = %+v", resolution.Reference)
	}
}

func materializeUnrelatedAmbientChanges(t *testing.T, root string) {
	t.Helper()
	for _, directory := range []string{
		filepath.Join(root, "openspec", "changes", "unrelated-wip"),
		filepath.Join(root, "openspec", "changes", "archive", "2026-08-03-unrelated"),
	} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "intent.md"), []byte("# incomplete unrelated artifact\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func materializeRoles(
	t *testing.T,
	directory string,
	current corpusCase,
	fixtures map[string][]byte,
	roles map[string]string,
) {
	t.Helper()
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	for role, name := range roles {
		raw := fixtureForRole(t, current, fixtures, role)
		if err := os.WriteFile(filepath.Join(directory, name), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func fixtureForRole(t *testing.T, current corpusCase, fixtures map[string][]byte, role string) []byte {
	t.Helper()
	for _, fixture := range current.Fixtures {
		if fixture.Role == role {
			return fixtures[fixture.Path]
		}
	}
	t.Fatalf("case %s has no fixture role %s", current.ID, role)
	return nil
}

func assertSemantic(t *testing.T, pair openspec.ConformedPair, expected semanticExpectation) {
	t.Helper()
	actual := pair.Intent
	if pair.Selection.Mode != expected.ContractMode || string(pair.Context.ID) != expected.ContextID ||
		pair.Context.Version != expected.ContextVersion || pair.Context.PreviousVersion != expected.ContextPrevious ||
		len(pair.Context.Items) != expected.ContextItems || string(actual.ID) != expected.IntentID ||
		actual.Version != expected.IntentVersion || actual.PreviousVersion != expected.IntentPrevious ||
		len(actual.DesiredOutcomes) != expected.DesiredOutcomes || len(actual.NonGoals) != expected.NonGoals ||
		len(actual.SuccessSignals) != expected.SuccessSignals {
		t.Fatalf("semantic projection mismatch:\npair=%+v\nexpected=%+v", pair, expected)
	}
	if expected.FirstRecipe != nil {
		if len(pair.Context.Items) == 0 || pair.Context.Items[0].VerificationRecipe != *expected.FirstRecipe {
			t.Fatalf("first recipe = %q, want %q", pair.Context.Items[0].VerificationRecipe, *expected.FirstRecipe)
		}
	}
	if expected.FirstRecipeNonempty && (len(pair.Context.Items) == 0 || pair.Context.Items[0].VerificationRecipe == "") {
		t.Fatal("first Verification recipe was lost")
	}
	if expected.ResolvedID == "" {
		if actual.ResolvedEscalation != nil {
			t.Fatalf("unexpected resolution = %+v", actual.ResolvedEscalation)
		}
	} else if actual.ResolvedEscalation == nil || actual.ResolvedEscalation.ResolvedID != expected.ResolvedID ||
		actual.ResolvedEscalation.EscalationDigest != expected.ResolvedDigest ||
		string(actual.ResolvedEscalation.Disposition) != expected.Disposition {
		t.Fatalf("resolution = %+v, expected = %+v", actual.ResolvedEscalation, expected)
	}
}

func assertDiagnostic(t *testing.T, err error, expected diagnosticExpectation) {
	t.Helper()
	var diagnostic *openspec.ArtifactDiagnostic
	if !errors.As(err, &diagnostic) || diagnostic == nil {
		t.Fatalf("error is not ArtifactDiagnostic: %v", err)
	}
	assertDiagnosticValue(t, *diagnostic, expected)
}

func assertDiagnosticValue(t *testing.T, diagnostic openspec.ArtifactDiagnostic, expected diagnosticExpectation) {
	t.Helper()
	actual := diagnosticExpectation{
		Code: string(diagnostic.Code), Path: diagnostic.Path, ArtifactKind: string(diagnostic.ArtifactKind),
		ContractMode: diagnostic.ContractMode, FieldOrSection: diagnostic.FieldOrSection,
		Observation: diagnostic.Observation, Expectation: diagnostic.Expectation, RepairHint: diagnostic.RepairHint,
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("diagnostic mismatch:\nactual=%+v\nexpected=%+v", actual, expected)
	}
}

func expectedMatrix(manifest corpusManifest) []laneResult {
	result := make([]laneResult, 0)
	for _, current := range manifest.Cases {
		for _, lane := range current.Lanes {
			result = append(result, laneResult{Case: current.ID, Lane: lane})
		}
	}
	return result
}

func validateExecutedMatrix(manifest corpusManifest, actual []laneResult) error {
	expected := expectedMatrix(manifest)
	if !reflect.DeepEqual(actual, expected) {
		return fmt.Errorf("executed case-lane matrix differs: got %+v, want %+v", actual, expected)
	}
	return nil
}

func cloneManifest(t *testing.T, value corpusManifest) corpusManifest {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var clone corpusManifest
	if err := json.Unmarshal(raw, &clone); err != nil {
		t.Fatal(err)
	}
	return clone
}

func laneIndex(value string) int {
	for index, lane := range []string{laneCompiler, laneLocalRun, laneAmbient} {
		if value == lane {
			return index
		}
	}
	return -1
}

func safeFixturePath(value string) bool {
	if value == "" || filepath.IsAbs(value) {
		return false
	}
	cleaned := filepath.ToSlash(filepath.Clean(value))
	return cleaned != "." && cleaned != ".." && !strings.HasPrefix(cleaned, "../") && cleaned == value
}

func validSHA256(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil && strings.ToLower(value) == value
}

func digest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}
