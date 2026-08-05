package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestOperatorSurfacesDoNotOverclaimEnforcement(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file")
	}
	repository := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	paths := []string{
		"README.md",
		"docs/contracts-v0.2.md",
		"docs/install.md",
		"docs/migration-v0.2.md",
		"docs/project-lifecycle.md",
		"docs/schemas/goalrail-doctor-v2.md",
		"internal/project/canon/bootstrap.md",
		"internal/project/canon/agent-block.md",
	}
	sources := make(map[string]string, len(paths)+1)
	var combined strings.Builder
	for _, path := range paths {
		raw, err := os.ReadFile(filepath.Join(repository, filepath.FromSlash(path)))
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		sources[path] = string(raw)
		combined.Write(raw)
		combined.WriteByte('\n')
	}
	var help bytes.Buffer
	if err := writeHelp(&help); err != nil {
		t.Fatal(err)
	}
	sources["gr help"] = help.String()
	combined.WriteString(help.String())

	for _, forbidden := range []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{name: "no Goalrail lifecycle command", pattern: regexp.MustCompile(`(?i)(?:you run|ordinary (?:code )?work (?:needs|requires)) no goalrail command`)},
		{name: "init is the whole attachment", pattern: regexp.MustCompile(`(?i)gr init (?:is|does) the whole attachment`)},
		{name: "local hook claimed as enforcement", pattern: regexp.MustCompile(`(?i)local hooks? (?:are|is|provide|provides|guarantee|guarantees) (?:authoritative|unbypassable|protected|enforced|enforcement)`)},
		{name: "workflow file claimed as activation", pattern: regexp.MustCompile(`(?i)(?:prepared |committed )?workflow file (?:proves|establishes|means|is) (?:active|activation|protected|enforced|enforcement)`)},
		{name: "managed identity claimed as protection", pattern: regexp.MustCompile(`(?i)managed project identity (?:proves|establishes|means|is) (?:active|activation|protected|enforced|enforcement)`)},
	} {
		for source, text := range sources {
			if match := forbidden.pattern.FindString(text); match != "" {
				t.Fatalf("%s in %s: %q", forbidden.name, source, match)
			}
		}
	}

	all := strings.ToLower(combined.String())
	for _, required := range []string{
		"local hooks are advisory",
		"a prepared workflow file is not activation evidence",
		"managed project identity does not prove",
		"protected shared admission",
		"independently verified",
	} {
		if !strings.Contains(all, required) {
			t.Fatalf("operator surfaces omit enforcement boundary %q", required)
		}
	}
}
