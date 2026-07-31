package updatecheck

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The guarantee this file holds used to be free: `net/http` was absent from the
// binary's dependency graph entirely, so `gr` could not open a connection at all.
// The update check ends that, and these tests are what replaces the property.
//
// They are deliberately not written as import-graph assertions about commands.
// Go's import graph is package-granular, and the diagnosis and the update command
// live in the same package — as do the run functions for every command in
// `cmd/gr`. An assertion at that granularity would pass while the requirement was
// being violated. What actually confines the network is that the transport is
// handed to the diagnosis as a value, constructed in one place; these tests pin
// both halves.

// TestOnlyTheTransportPackageImportsAnHTTPClient pins the first half at the
// granularity Go actually has.
func TestOnlyTheTransportPackageImportsAnHTTPClient(t *testing.T) {
	importers := packagesImporting(t, "net/http")
	for _, importer := range importers {
		switch importer {
		case "internal/updatecheck":
			// This package. The one that may.
		case "internal/adapters/langfuse", "cmd/goalrail-canary":
			// The separate canary binary and its read-only telemetry client,
			// neither of which `gr` links.
		default:
			t.Errorf("%s imports net/http; the transport belongs in internal/updatecheck alone", importer)
		}
	}
}

// TestOnlyTwoPackagesImportTheTransport pins the second half. Two is the honest
// number and both are named rather than counted: the diagnosis consumes the
// check's type and its version comparison, and the command constructs the real
// one. A third importer means the surface grew.
func TestOnlyTwoPackagesImportTheTransport(t *testing.T) {
	allowed := map[string]bool{
		"internal/updatecheck": true,
		"internal/harness":     true, // consumes the type; constructs nothing
		"cmd/gr":               true, // the one construction site
	}
	for _, importer := range packagesImporting(t, "github.com/heurema/goalrail/internal/updatecheck") {
		if !allowed[importer] {
			t.Errorf("%s imports the transport; only the diagnosis and the command it is constructed in may", importer)
		}
	}
}

// TestTheRealCheckIsConstructedOnce is the assertion that carries the weight the
// import graph cannot. Whoever calls New holds the network; nobody else can reach
// it, whatever package they share.
//
// The first version of this test grepped for the literal "updatecheck.New(",
// which an import alias defeats without malice. This one resolves the import:
// whatever local name this package is given — alias, default, or a dot import,
// which is rejected outright because it binds no selector at all — any
// reference to its New, as a call or as a value, is a site. And the one site is
// named, not counted: the command constructs it, the diagnosis only receives it.
func TestTheRealCheckIsConstructedOnce(t *testing.T) {
	root := moduleRoot(t)
	const importPath = "github.com/heurema/goalrail/internal/updatecheck"
	var sites []string
	walkGoFiles(t, root, func(filePath string, contents []byte) {
		if strings.Contains(filepath.ToSlash(filepath.Dir(filePath)), "internal/updatecheck") {
			return // the package itself
		}
		file, err := parser.ParseFile(token.NewFileSet(), filePath, contents, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", filePath, err)
		}
		localName := ""
		for _, imported := range file.Imports {
			value, unquoteErr := strconv.Unquote(imported.Path.Value)
			if unquoteErr != nil || value != importPath {
				continue
			}
			switch {
			case imported.Name == nil:
				localName = "updatecheck"
			case imported.Name.Name == ".":
				relative, _ := filepath.Rel(root, filePath)
				t.Errorf("%s dot-imports the transport, which no reference scan can follow", relative)
				return
			default:
				localName = imported.Name.Name
			}
		}
		if localName == "" {
			return
		}
		ast.Inspect(file, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			identifier, ok := selector.X.(*ast.Ident)
			if !ok || identifier.Name != localName || selector.Sel.Name != "New" {
				return true
			}
			relative, _ := filepath.Rel(root, filePath)
			sites = append(sites, filepath.ToSlash(relative))
			return true
		})
	})
	if len(sites) != 1 || sites[0] != "cmd/gr/harness.go" {
		t.Errorf("the real check is referenced at %v; the one construction site is cmd/gr/harness.go", sites)
	}
}

// packagesImporting returns every package in this module that imports the given
// path, as module-relative directories.
func packagesImporting(t *testing.T, target string) []string {
	t.Helper()
	root := moduleRoot(t)
	seen := map[string]bool{}
	walkGoFiles(t, root, func(path string, contents []byte) {
		if strings.HasSuffix(path, "_test.go") {
			return // tests may reach for anything; the shipped binary is the subject
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, contents, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, imported := range file.Imports {
			value, err := strconv.Unquote(imported.Path.Value)
			if err != nil || value != target {
				continue
			}
			relative, _ := filepath.Rel(root, filepath.Dir(path))
			seen[filepath.ToSlash(relative)] = true
		}
	})
	var packages []string
	for name := range seen {
		packages = append(packages, name)
	}
	return packages
}

func walkGoFiles(t *testing.T, root string, visit func(path string, contents []byte)) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "openspec", "docs", "assets", "probes", "canary":
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		visit(path, contents)
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			t.Fatal("no go.mod above the working directory")
		}
		directory = parent
	}
}
