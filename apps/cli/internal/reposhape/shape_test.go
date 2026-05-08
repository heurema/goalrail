package reposhape

import (
	"slices"
	"testing"
)

func TestWorkspaceManifestPaths(t *testing.T) {
	want := []string{"Cargo.toml", "Gemfile", "composer.json", "go.mod", "go.work", "package.json", "pyproject.toml", "requirements.txt"}
	got := WorkspaceManifestPaths()
	if !slices.Equal(got, want) {
		t.Fatalf("WorkspaceManifestPaths() = %#v, want %#v", got, want)
	}
	for _, name := range want {
		if !IsWorkspaceManifest(name) {
			t.Fatalf("IsWorkspaceManifest(%q) = false, want true", name)
		}
	}
	if IsWorkspaceManifest("pnpm-lock.yaml") {
		t.Fatal("IsWorkspaceManifest(pnpm-lock.yaml) = true, want false")
	}
}

func TestInventoryPathListsReturnCopies(t *testing.T) {
	for name, list := range map[string]func() []string{
		"WorkspaceManifestPaths":  WorkspaceManifestPaths,
		"RootInventoryPaths":      RootInventoryPaths,
		"WorkspaceInventoryPaths": WorkspaceInventoryPaths,
	} {
		got := list()
		if len(got) == 0 {
			t.Fatalf("%s() returned empty list", name)
		}
		first := got[0]
		got[0] = "changed"
		if list()[0] != first {
			t.Fatalf("%s() returned mutable package state", name)
		}
	}
	if !containsString(RootInventoryPaths(), "Cargo.lock") || !containsString(WorkspaceInventoryPaths(), "Gemfile.lock") {
		t.Fatalf("inventory paths missing shared package-manager metadata: root=%#v workspace=%#v", RootInventoryPaths(), WorkspaceInventoryPaths())
	}
}

func TestToolchainsForPath(t *testing.T) {
	tests := map[string][]string{
		"apps/api/go.mod":           {"go"},
		"go.work":                   {"go"},
		"packages/web/package.json": {"node"},
		"Cargo.toml":                {"rust"},
		"pyproject.toml":            {"python"},
		"requirements.txt":          {"python"},
		"Gemfile":                   {"ruby"},
		"composer.json":             {"php"},
		"Dockerfile":                {"docker"},
		"docker-compose.yml":        {"docker"},
		"compose.yml":               {"docker"},
	}
	for path, want := range tests {
		if got := ToolchainsForPath(path); !slices.Equal(got, want) {
			t.Fatalf("ToolchainsForPath(%q) = %#v, want %#v", path, got, want)
		}
	}
	if got := ToolchainsForPath("README.md"); len(got) != 0 {
		t.Fatalf("ToolchainsForPath(README.md) = %#v, want empty", got)
	}
}

func TestPackageManagersForPath(t *testing.T) {
	tests := map[string][]string{
		"pnpm-lock.yaml":      {"pnpm"},
		"package-lock.json":   {"npm"},
		"yarn.lock":           {"yarn"},
		"bun.lock":            {"bun"},
		"bun.lockb":           {"bun"},
		"Cargo.lock":          {"cargo"},
		"poetry.lock":         {"poetry"},
		"uv.lock":             {"uv"},
		"requirements.txt":    {"pip"},
		"Gemfile.lock":        {"bundler"},
		"composer.lock":       {"composer"},
		"apps/api/Cargo.lock": {"cargo"},
	}
	for path, want := range tests {
		if got := PackageManagersForPath(path); !slices.Equal(got, want) {
			t.Fatalf("PackageManagersForPath(%q) = %#v, want %#v", path, got, want)
		}
	}
	if got := PackageManagersForPath("package.json"); len(got) != 0 {
		t.Fatalf("PackageManagersForPath(package.json) = %#v, want empty", got)
	}
}

func TestCIWorkflowPath(t *testing.T) {
	if !IsCIWorkflowPath(".github/workflows/ci.yml") {
		t.Fatal("IsCIWorkflowPath(.github/workflows/ci.yml) = false, want true")
	}
	if IsCIWorkflowPath(".github/workflow/ci.yml") {
		t.Fatal("IsCIWorkflowPath(.github/workflow/ci.yml) = true, want false")
	}
	if IsCIWorkflowPath(".github/workflows") {
		t.Fatal("IsCIWorkflowPath(.github/workflows) = true, want false")
	}
}

func TestEntrypointCandidate(t *testing.T) {
	for _, path := range []string{"Dockerfile", "docker-compose.yml", "compose.yml", "Makefile", "Taskfile.yml", "cmd/server/main.go"} {
		if !IsEntrypointCandidate(path) {
			t.Fatalf("IsEntrypointCandidate(%q) = false, want true", path)
		}
	}
	if IsEntrypointCandidate("internal/server/main.go") {
		t.Fatal("IsEntrypointCandidate(internal/server/main.go) = true, want false")
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
