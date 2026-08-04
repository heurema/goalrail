package harness

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSchemaDifferenceReadsOnlyBoundedStructure(t *testing.T) {
	replaced := []byte("name: old\r\nartifacts:\r\n  - id: 'intent' # stable id\r\n    instruction: |\r\n      Keep prose opaque.\r\n      - this is not an artifact\r\n    requires: []\r\n  # between entries\r\n  - id: design\r\n    instruction: >-\r\n      Explain design.   \r\n    requires:\r\n      - proposal\r\n")
	adopted := []byte("name: new\nartifacts:\n  - id: context\n    instruction: >\n      Gather evidence.\n    requires: []\n  - id: intent\n    instruction: |\n      Keep prose opaque.\n      - this is not an artifact\n    requires: []\n  - id: design\n    instruction: >\n      Explain a changed design.\n    requires:\n      - proposal\n      - specs\n")

	difference := compareSchemaBytes(replaced, adopted)
	if !difference.Comparable {
		t.Fatalf("schemas were not comparable: %s", difference.Reason)
	}
	if !reflect.DeepEqual(difference.OnlyInAdopted, []string{"context"}) {
		t.Fatalf("added artifacts = %#v", difference.OnlyInAdopted)
	}
	if !reflect.DeepEqual(difference.DependenciesChanged, []string{"design"}) {
		t.Fatalf("dependency changes = %#v", difference.DependenciesChanged)
	}
	if !reflect.DeepEqual(difference.InstructionsChanged, []string{"design"}) {
		t.Fatalf("instruction changes = %#v", difference.InstructionsChanged)
	}
	artifacts, err := readSchemaArtifacts(replaced)
	if err != nil {
		t.Fatal(err)
	}
	wantSpan := "    instruction: |\r\n      Keep prose opaque.\r\n      - this is not an artifact\r\n"
	if string(artifacts[0].instructionSpan) != wantSpan {
		t.Fatalf("instruction span = %q, want %q", artifacts[0].instructionSpan, wantSpan)
	}
}

func TestSchemaReaderRefusesShapesItCannotFollow(t *testing.T) {
	tests := map[string]string{
		"missing artifacts": "name: x\n",
		"missing id":        "artifacts:\n  - instruction: >\n      x\n",
		"duplicate id":      "artifacts:\n  - id: x\n    instruction: x\n  - id: x\n    instruction: y\n",
		"bad indentation":   "artifacts:\n   - id: x\n     instruction: x\n",
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			if difference := compareSchemaBytes([]byte(input), []byte(input)); difference.Comparable {
				t.Fatalf("unsupported schema was reported comparable: %#v", difference)
			}
		})
	}
}

func TestSchemaDifferenceComparesRequiresAsSets(t *testing.T) {
	left := []byte("artifacts:\n  - id: design\n    instruction: design\n    requires: [proposal, proposal]\n")
	right := []byte("artifacts:\n  - id: design\n    instruction: design\n    requires: [proposal]\n")
	difference := compareSchemaBytes(left, right)
	if !difference.Comparable || len(difference.DependenciesChanged) != 0 {
		t.Fatalf("duplicate set member changed the dependency set: %#v", difference)
	}
}

func TestInstructionDifferenceIgnoresCheckoutLineEndingsAndOuterComments(t *testing.T) {
	left := []byte("artifacts:\r\n  - id: intent\r\n    requires: []\r\n    instruction: |\r\n      Same instruction.\r\n  # left-side note\r\n  - id: design\r\n    requires: []\r\n    instruction: design\r\n")
	right := []byte("artifacts:\n  - id: intent\n    requires: []\n    instruction: |\n      Same instruction.\n  # right-side note\n  - id: design\n    requires: []\n    instruction: design\n")
	difference := compareSchemaBytes(left, right)
	if !difference.Comparable || len(difference.InstructionsChanged) != 0 {
		t.Fatalf("transport formatting changed an instruction: %#v", difference)
	}
}

func TestInstructionDifferenceIgnoresYAMLInsignificantTails(t *testing.T) {
	left := []byte("artifacts:\n  - id: intent\n    requires: []\n    instruction: |\n      Same instruction.\n\n  - id: design\n    requires: []\n    instruction: design   # source note\n\n")
	right := []byte("artifacts:\n  - id: intent\n    requires: []\n    instruction: |\n      Same instruction.\n  - id: design\n    requires: []\n    instruction: design\n")
	difference := compareSchemaBytes(left, right)
	if !difference.Comparable || len(difference.InstructionsChanged) != 0 {
		t.Fatalf("YAML-insignificant tails changed an instruction: %#v", difference)
	}
}

func TestInstructionDifferencePreservesKeepChompingTails(t *testing.T) {
	left := []byte("artifacts:\n  - id: intent\n    requires: []\n    instruction: |+\n      Same instruction.\n\n")
	right := []byte("artifacts:\n  - id: intent\n    requires: []\n    instruction: |+\n      Same instruction.\n")
	difference := compareSchemaBytes(left, right)
	if !difference.Comparable || !reflect.DeepEqual(difference.InstructionsChanged, []string{"intent"}) {
		t.Fatalf("keep-chomped tail was lost: %#v", difference)
	}
}

func TestRepositorySchemaComparisonUsesTheOverlayOnDisk(t *testing.T) {
	root := t.TempDir()
	writeSchema := func(name, content string) {
		path := filepath.Join(root, "openspec", "schemas", name, "schema.yaml")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeSchema("old", "artifacts:\n  - id: intent\n    instruction: old\n    requires: []\n")
	if _, err := Materialize(root, false); err != nil {
		t.Fatal(err)
	}
	writeSchema(SchemaName, "artifacts:\n  - id: context\n    instruction: edited overlay\n    requires: []\n")
	outcomes, err := Materialize(root, false)
	if err != nil {
		t.Fatal(err)
	}
	schemaKept := false
	for _, outcome := range outcomes {
		if outcome.Path == "openspec/schemas/"+SchemaName+"/schema.yaml" &&
			outcome.Action == ActionKept && outcome.State == FileEdited {
			schemaKept = true
		}
	}
	if !schemaKept {
		t.Fatalf("edited overlay was not kept: %#v", outcomes)
	}

	difference := CompareRepositorySchemas(root, "old")
	if !difference.Comparable || !reflect.DeepEqual(difference.OnlyInAdopted, []string{"context"}) {
		t.Fatalf("comparison did not follow the materialized overlay: %#v", difference)
	}
	missing := CompareRepositorySchemas(root, "spec-driven")
	if missing.Comparable || !strings.Contains(missing.Reason, "not a file in the repository") {
		t.Fatalf("package schema produced %#v", missing)
	}
}

func TestRulesSnapshotKeepsOnlyTheTopLevelBlock(t *testing.T) {
	input := "schema: old\r\ncontext:\r\n  rules:\r\n    - nested\r\nrules:\r\n# comment inside the block\r\n  intent:\r\n    - first rule\r\n    - |\r\n      multi-line rule\r\n      - prose, not an item\r\n\r\n# closing note\r\n"
	snapshot := extractRules([]byte(input))
	if !snapshot.Present || !snapshot.HasRules || !snapshot.Counted || snapshot.Count != 2 {
		t.Fatalf("rules snapshot = %#v", snapshot)
	}
	expected := "rules:\r\n# comment inside the block\r\n  intent:\r\n    - first rule\r\n    - |\r\n      multi-line rule\r\n      - prose, not an item\r\n"
	if snapshot.Text != expected {
		t.Fatalf("rules text = %q, want %q", snapshot.Text, expected)
	}
	if snapshot.Digest != digestBytes([]byte(expected)) {
		t.Fatal("rules digest did not cover the extracted span alone")
	}
}

func TestRulesSnapshotKeepsUncountableLiteralContentVerbatim(t *testing.T) {
	left := "schema: old\nrules:\n  intent:\n    - |\n      Use this template:\n      \tname: value\n    - keep evidence distinct\n"
	right := strings.Replace(left, "keep evidence distinct", "keep evidence separate", 1)
	snapshot := extractRules([]byte(left))
	if !snapshot.Present || !snapshot.HasRules || snapshot.Counted {
		t.Fatalf("literal-tab rules snapshot = %#v", snapshot)
	}
	if snapshot.Text != strings.TrimPrefix(left, "schema: old\n") ||
		!strings.Contains(snapshot.Text, "keep evidence distinct") {
		t.Fatalf("rules text was truncated: %q", snapshot.Text)
	}
	if snapshot.Digest == extractRules([]byte(right)).Digest {
		t.Fatal("an edit below uncountable literal content did not change the rules digest")
	}
}

func TestRulesSnapshotDistinguishesAbsentEmptyAndUncountable(t *testing.T) {
	absent := extractRules([]byte("schema: old\ncontext:\n  rules:\n    - nested\n"))
	if absent.Present || !absent.Counted || absent.Digest != digestBytes(nil) {
		t.Fatalf("absent rules = %#v", absent)
	}

	empty := extractRules([]byte("schema: old\nrules:\n\n# note\n"))
	if !empty.Present || empty.HasRules || !empty.Counted || empty.Count != 0 || empty.Text != "rules:\n" {
		t.Fatalf("empty rules = %#v", empty)
	}

	flow := extractRules([]byte("schema: old\nrules: {intent: [\"a\", \"b\"]}\n"))
	if !flow.Present || !flow.HasRules || flow.Counted || !strings.Contains(flow.CountReason, "inline") {
		t.Fatalf("flow rules = %#v", flow)
	}

	awkward := extractRules([]byte("schema: old\nrules:\n  intent:\n    [\n      \"ask exactly one question\"\n    ]\n"))
	if !awkward.Present || !awkward.HasRules || awkward.Counted || awkward.Text == "" || awkward.Digest == "" {
		t.Fatalf("awkward rules = %#v", awkward)
	}

	duplicate := extractRules([]byte("rules:\n  intent:\n    - first\ncontext: retained\nrules:\n  proposal:\n    - second\n"))
	if !duplicate.Present || !duplicate.HasRules || duplicate.Counted || duplicate.Digest == "" ||
		!strings.Contains(duplicate.Text, "first") || !strings.Contains(duplicate.Text, "second") ||
		strings.Contains(duplicate.Text, "context: retained") {
		t.Fatalf("duplicate rules = %#v", duplicate)
	}

	noNewline := extractRules([]byte("schema: old\nrules:\n  intent:\n    - one"))
	if !noNewline.Counted || noNewline.Count != 1 || noNewline.Text != "rules:\n  intent:\n    - one" {
		t.Fatalf("final rules block = %#v", noNewline)
	}
}

func TestEnsureConfigCarriesRulesOnlyWhenItReplacesANamedSchema(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, filepath.FromSlash(ConfigPath))
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	original := "schema: intent-driven\ncontext: keep\nrules:\n  intent:\n    - ask once\n"
	if err := os.WriteFile(configPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	outcome, err := EnsureConfig(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.PreviousSchema != "intent-driven" || outcome.Rules == nil || outcome.Rules.Count != 1 {
		t.Fatalf("config outcome = %#v", outcome)
	}
	want := strings.Replace(original, "schema: intent-driven", "schema: "+SchemaName, 1)
	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("configuration was reformatted:\n%s", got)
	}

	second, err := EnsureConfig(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if second.Rules != nil || second.PreviousSchema != "" {
		t.Fatalf("unchanged configuration carried adoption data: %#v", second)
	}
}
