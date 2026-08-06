package review

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/ambient"
)

// Goalrail ships no integration name, and this is the check behind that claim.
//
// The removal syntax is Goalrail's to know; the thing removed is the caller's.
// A list of known-hostile integrations would age into a wrong answer nobody
// revisits, and asserting about other people's machines is not this
// repository's business. The rendering site is allowed to contain the syntax
// because that is what renders it; nothing else may.
func TestNoIntegrationIsNamedInShippedContent(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file")
	}
	repository := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))

	// Where the syntax may appear: the code that renders it, and the tests that
	// assert what it renders.
	allowed := map[string]struct{}{
		filepath.Join("internal", "review", "run.go"):                  {},
		filepath.Join("internal", "review", "stall_test.go"):           {},
		filepath.Join("internal", "review", "isolation_scope_test.go"): {},
	}
	const syntax = "mcp_servers"

	err := filepath.WalkDir(repository, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, relErr := filepath.Rel(repository, path)
		if relErr != nil {
			return relErr
		}
		if entry.IsDir() {
			switch relative {
			case ".git", filepath.Join("openspec", "changes"):
				// Change artifacts carry the investigation's evidence, including
				// the integration that blocked a reviewer. Evidence is where a
				// name belongs; shipped behaviour is not.
				return filepath.SkipDir
			}
			return nil
		}
		switch filepath.Ext(relative) {
		case ".go", ".md", ".json", ".yaml", ".yml", ".tmpl":
		default:
			return nil
		}
		if _, permitted := allowed[relative]; permitted {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(raw), syntax) {
			t.Errorf("shipped content names a provider integration surface: %s", relative)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Absent a caller's name, nothing about integrations reaches the invocation.
func TestWithoutACallerNameNoIsolationArgumentIsRendered(t *testing.T) {
	for _, reviewer := range []ambient.Scaffold{ambient.ScaffoldCodex, ambient.ScaffoldClaudeCode} {
		_, arguments, _, err := reviewCommand(reviewer, "base..head", "medium", "", []byte("x"), nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, argument := range arguments {
			if strings.Contains(argument, "mcp") {
				t.Fatalf("%s was invoked with an integration argument nobody asked for: %q", reviewer, argument)
			}
		}
	}
}
