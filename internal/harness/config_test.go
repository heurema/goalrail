package harness

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, root, content string) string {
	t.Helper()
	absolute := filepath.Join(root, filepath.FromSlash(ConfigPath))
	if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
		t.Fatalf("create config directory: %v", err)
	}
	if err := os.WriteFile(absolute, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return absolute
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func TestEnsureConfigCreatesAnAbsentConfiguration(t *testing.T) {
	root := t.TempDir()
	outcome, err := EnsureConfig(root, false)
	if err != nil {
		t.Fatalf("ensure config: %v", err)
	}
	if outcome.Action != ConfigCreated {
		t.Fatalf("action is %q, expected created", outcome.Action)
	}
	named, present, err := ConfiguredSchema(root)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	if !present || named != SchemaName {
		t.Fatalf("created configuration names %q (present=%t)", named, present)
	}
}

func TestEnsureConfigLeavesTheGoalrailSchemaByteIdentical(t *testing.T) {
	root := t.TempDir()
	original := "schema: " + SchemaName + "\n\ncontext: |\n  our own words\n"
	absolute := writeConfig(t, root, original)

	outcome, err := EnsureConfig(root, false)
	if err != nil {
		t.Fatalf("ensure config: %v", err)
	}
	if outcome.Action != ConfigUnchanged {
		t.Errorf("action is %q, expected unchanged", outcome.Action)
	}
	if readFile(t, absolute) != original {
		t.Error("a configuration already naming the Goalrail schema was rewritten")
	}
}

// TestEnsureConfigSwitchesTheStockSchemaAndKeepsEverythingElse pins that adoption
// preserves the project's own prose and comments: a parse-and-rewrite would
// reformat content this package does not own.
func TestEnsureConfigSwitchesTheStockSchemaAndKeepsEverythingElse(t *testing.T) {
	root := t.TempDir()
	original := "# our configuration\nschema: spec-driven\n\ncontext: |\n  a description we wrote\n\nrules:\n  proposal:\n    - keep it short\n"
	absolute := writeConfig(t, root, original)

	outcome, err := EnsureConfig(root, false)
	if err != nil {
		t.Fatalf("ensure config: %v", err)
	}
	if outcome.Action != ConfigSwitched {
		t.Fatalf("action is %q, expected switched", outcome.Action)
	}
	if outcome.PreviousSchema != "spec-driven" {
		t.Errorf("previous schema reported as %q", outcome.PreviousSchema)
	}
	expected := "# our configuration\nschema: " + SchemaName + "\n\ncontext: |\n  a description we wrote\n\nrules:\n  proposal:\n    - keep it short\n"
	if actual := readFile(t, absolute); actual != expected {
		t.Errorf("switching the schema changed more than the key:\n%q", actual)
	}
}

func TestEnsureConfigRefusesAForeignCustomSchema(t *testing.T) {
	root := t.TempDir()
	original := "schema: acme-workflow\n\ncontext: |\n  another team's workflow\n"
	absolute := writeConfig(t, root, original)

	_, err := EnsureConfig(root, false)
	if !errors.Is(err, ErrForeignSchema) {
		t.Fatalf("expected a foreign-schema refusal, got %v", err)
	}
	if readFile(t, absolute) != original {
		t.Error("a refused switch still modified the configuration")
	}

	outcome, err := EnsureConfig(root, true)
	if err != nil {
		t.Fatalf("confirmed switch: %v", err)
	}
	if outcome.Action != ConfigSwitched || outcome.PreviousSchema != "acme-workflow" {
		t.Errorf("confirmed switch reported %+v", outcome)
	}
	named, _, err := ConfiguredSchema(root)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	if named != SchemaName {
		t.Errorf("configuration names %q after a confirmed switch", named)
	}
}

func TestEnsureConfigAddsAMissingKey(t *testing.T) {
	root := t.TempDir()
	original := "context: |\n  a configuration with no schema key\n"
	absolute := writeConfig(t, root, original)

	outcome, err := EnsureConfig(root, false)
	if err != nil {
		t.Fatalf("ensure config: %v", err)
	}
	if outcome.Action != ConfigSwitched {
		t.Fatalf("action is %q, expected switched", outcome.Action)
	}
	expected := "schema: " + SchemaName + "\n" + original
	if actual := readFile(t, absolute); actual != expected {
		t.Errorf("adding the key changed the rest of the file:\n%q", actual)
	}
}

// TestEnsureConfigIgnoresANestedSchemaKey pins that only a top-level assignment
// counts: rewriting an indented one would change a different setting.
func TestEnsureConfigIgnoresANestedSchemaKey(t *testing.T) {
	root := t.TempDir()
	original := "schema: spec-driven\n\nrules:\n  specs:\n    schema: something-else\n"
	absolute := writeConfig(t, root, original)

	if _, err := EnsureConfig(root, false); err != nil {
		t.Fatalf("ensure config: %v", err)
	}
	expected := "schema: " + SchemaName + "\n\nrules:\n  specs:\n    schema: something-else\n"
	if actual := readFile(t, absolute); actual != expected {
		t.Errorf("a nested key was touched:\n%q", actual)
	}
}

func TestEnsureConfigReadsAQuotedValue(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "schema: \""+SchemaName+"\"\n")
	outcome, err := EnsureConfig(root, false)
	if err != nil {
		t.Fatalf("ensure config: %v", err)
	}
	if outcome.Action != ConfigUnchanged {
		t.Errorf("a quoted Goalrail schema was reported as %q", outcome.Action)
	}
}

// TestEnsureConfigReadsAValueWithATrailingComment covers this repository's own
// configuration shape, whose schema key is preceded by an explanatory comment.
func TestEnsureConfigReadsAValueWithATrailingComment(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "schema: "+SchemaName+" # pinned by AGENTS.md\n")
	outcome, err := EnsureConfig(root, false)
	if err != nil {
		t.Fatalf("ensure config: %v", err)
	}
	if outcome.Action != ConfigUnchanged {
		t.Errorf("a commented Goalrail schema was reported as %q", outcome.Action)
	}
}

func TestEnsureConfigPreservesWindowsLineEndings(t *testing.T) {
	root := t.TempDir()
	original := "schema: spec-driven\r\ncontext: |\r\n  ours\r\n"
	absolute := writeConfig(t, root, original)
	if _, err := EnsureConfig(root, false); err != nil {
		t.Fatalf("ensure config: %v", err)
	}
	expected := "schema: " + SchemaName + "\r\ncontext: |\r\n  ours\r\n"
	if actual := readFile(t, absolute); actual != expected {
		t.Errorf("line endings were not preserved:\n%q", actual)
	}
}

// TestEnsureConfigReadsAQuotedValueWithATrailingComment pins the combination the
// pre-PR review found: quoting resolved after comment-stripping left the closing
// quote inside the value, and a correctly configured repository was refused as
// running a foreign schema.
func TestEnsureConfigReadsAQuotedValueWithATrailingComment(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "schema: \""+SchemaName+"\" # pinned by AGENTS.md\n")
	outcome, err := EnsureConfig(root, false)
	if err != nil {
		t.Fatalf("ensure config: %v", err)
	}
	if outcome.Action != ConfigUnchanged {
		t.Errorf("a quoted, commented Goalrail schema was reported as %q", outcome.Action)
	}

	stock := t.TempDir()
	writeConfig(t, stock, "schema: \"spec-driven\" # the default\n")
	outcome, err = EnsureConfig(stock, false)
	if err != nil {
		t.Fatalf("a quoted, commented stock schema was refused: %v", err)
	}
	if outcome.Action != ConfigSwitched || outcome.PreviousSchema != "spec-driven" {
		t.Errorf("stock adoption reported %+v", outcome)
	}
}

func TestEnsureConfigTreatsACommentOnlyValueAsUnnamed(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "schema: # decided per team\n")
	outcome, err := EnsureConfig(root, false)
	if err != nil {
		t.Fatalf("a comment-only value was refused as a foreign schema: %v", err)
	}
	if outcome.Action != ConfigSwitched {
		t.Errorf("an unnamed schema was reported as %q", outcome.Action)
	}
	named, _, err := ConfiguredSchema(root)
	if err != nil {
		t.Fatal(err)
	}
	if named != SchemaName {
		t.Errorf("configuration names %q afterwards", named)
	}
}
