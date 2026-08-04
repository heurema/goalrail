package openspec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadChangeSchemaHandlesQuotedAndCommentedValues(t *testing.T) {
	tests := map[string]string{
		"unquoted comment": "schema: intent-driven # legacy\n",
		"tabbed comment":   "schema: intent-driven\t# legacy\n",
		"double quoted":    "schema: \"intent-driven\" # legacy\n",
		"single quoted":    "schema: 'intent-driven' # legacy\n",
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), ".openspec.yaml")
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			schema, err := readChangeSchema(path)
			if err != nil {
				t.Fatal(err)
			}
			if schema != "intent-driven" {
				t.Fatalf("schema = %q", schema)
			}
		})
	}
}

func TestCountSchemaPinsSeparatesActiveAndArchivedChanges(t *testing.T) {
	root := t.TempDir()
	writePin := func(relative, content string) {
		path := filepath.Join(root, filepath.FromSlash(relative), ".openspec.yaml")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writePin("openspec/changes/current", "schema: intent-driven # active\n")
	writePin("openspec/changes/other", "schema: goalrail-intent\n")
	writePin("openspec/changes/archive/finished", "schema: 'intent-driven' # archived\n")

	counts, err := CountSchemaPins(root, "intent-driven")
	if err != nil {
		t.Fatal(err)
	}
	if counts.Active != 1 || counts.Archived != 1 || counts.Total() != 2 {
		t.Fatalf("counts = %#v", counts)
	}
}

func TestCountSchemaPinsAcceptsMissingOrArchiveOnlyChanges(t *testing.T) {
	empty, err := CountSchemaPins(t.TempDir(), "intent-driven")
	if err != nil || empty.Total() != 0 {
		t.Fatalf("missing changes = %#v, %v", empty, err)
	}

	root := t.TempDir()
	path := filepath.Join(root, "openspec", "changes", "archive", "finished", ".openspec.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("schema: intent-driven\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	archiveOnly, err := CountSchemaPins(root, "intent-driven")
	if err != nil || archiveOnly.Active != 0 || archiveOnly.Archived != 1 {
		t.Fatalf("archive-only changes = %#v, %v", archiveOnly, err)
	}
}
