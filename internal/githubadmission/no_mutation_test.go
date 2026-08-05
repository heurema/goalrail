package githubadmission

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The only provider-capable production package in this slice is this adapter.
// Keeping its HTTP method surface at GET makes init/setup/update/doctor/core
// incapable of enabling protection, commenting, committing, pushing, or merging.
func TestProviderAdapterHasNoExternalMutationMethod(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	files := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(files, filepath.Clean(entry.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok || !strings.HasPrefix(selector.Sel.Name, "Method") {
				return true
			}
			identifier, ok := selector.X.(*ast.Ident)
			if ok && identifier.Name == "http" && selector.Sel.Name != "MethodGet" {
				t.Errorf("provider adapter references mutating HTTP method %s in %s", selector.Sel.Name, entry.Name())
			}
			return true
		})
		ast.Inspect(parsed, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(literal.Value)
			if err != nil {
				return true
			}
			for _, forbidden := range []string{"/merges", "/comments/", "/statuses/", "/check-runs/"} {
				if strings.Contains(value, forbidden) {
					t.Errorf("provider adapter contains external mutation route %q in %s", forbidden, entry.Name())
				}
			}
			return true
		})
	}
}
