package admission

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/lineage"
)

func TestCorePackageHasNoCollectorProviderNetworkOrCredentialImports(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	banned := []string{
		"net", "net/", "os", "os/", "path/filepath",
		"github.com/heurema/goalrail/internal/adapters", "github.com/heurema/goalrail/internal/admissiongit",
		"github.com/heurema/goalrail/internal/githubadmission", "github.com/heurema/goalrail/internal/localrun",
		"github.com/heurema/goalrail/internal/project",
	}
	files := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(files, filepath.Clean(entry.Name()), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range parsed.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range general.Specs {
				importSpec, ok := spec.(*ast.ImportSpec)
				if !ok {
					continue
				}
				importPath, err := strconv.Unquote(importSpec.Path.Value)
				if err != nil {
					t.Fatal(err)
				}
				for _, prefix := range banned {
					if importPath == prefix || strings.HasPrefix(importPath, prefix) {
						t.Fatalf("pure admission core imports %q from %s", importPath, entry.Name())
					}
				}
			}
		}
	}
}

// TestCorePackageNeverReadsWallClockTime keeps the composition of provider
// candidates deterministic. Every time value must arrive as an explicit input,
// so two runs over identical frozen evidence cannot diverge.
func TestCorePackageNeverReadsWallClockTime(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		raw, err := os.ReadFile(filepath.Clean(entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"time.Now(", "time.Since(", "time.Until("} {
			if strings.Contains(string(raw), forbidden) {
				t.Fatalf("pure admission core reads the wall clock via %s in %s", forbidden, entry.Name())
			}
		}
	}
}

func TestRequiredSpineMatchesAppendOnlyLineageContract(t *testing.T) {
	core := RequiredSpine()
	store := lineage.DefaultRequirements()
	if len(core) != len(store) {
		t.Fatalf("required spine length = %d, lineage contract = %d", len(core), len(store))
	}
	for index := range core {
		if core[index] != store[index] {
			t.Fatalf("required spine differs at %d: %+v != %+v", index, core[index], store[index])
		}
	}
}
