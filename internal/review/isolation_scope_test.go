package review

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/ambient"
)

// The removal syntax appears only where it is rendered.
//
// This is narrower than "no integration is named anywhere", and deliberately
// so: a check cannot verify that claim, because a name is an arbitrary string
// and nothing distinguishes one in prose. What it does verify is that the
// provider configuration surfaces this repository knows how to write are
// confined to the code that writes them, so no default, example, or help text
// can quietly acquire a machine's integration. The earlier version of this test
// claimed the broader thing while checking this narrower one, in a subset of
// file extensions; the claim now matches the check, and the scan covers every
// text file.
func TestTheRemovalSyntaxAppearsOnlyWhereItIsRendered(t *testing.T) {
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
	syntaxes := []string{"mcp_servers", "deniedMcpServers"}

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
		if _, permitted := allowed[relative]; permitted {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		// Every file, not a chosen set of extensions: a shell script or a
		// configuration sample is exactly where such a surface would hide, and
		// choosing extensions is choosing not to look.
		if bytes.ContainsRune(raw, 0) {
			return nil
		}
		for _, surface := range syntaxes {
			if strings.Contains(string(raw), surface) {
				t.Errorf("shipped content carries the %q provider surface: %s", surface, relative)
			}
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
