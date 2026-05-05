package initcmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectRepositoryInventoryDetectsWorkspaceManifests(t *testing.T) {
	root := t.TempDir()
	writeInventoryFile(t, root, filepath.Join("apps", "cli", "go.mod"), "module github.com/heurema/goalrail/apps/cli\n")
	writeInventoryFile(t, root, filepath.Join("apps", "server", "go.mod"), "module github.com/heurema/goalrail/apps/server\n")
	writeInventoryFile(t, root, filepath.Join("apps", "web", "package.json"), `{"name":"goalrail-web"}`)
	writeInventoryFile(t, root, filepath.Join("apps", "web", "package-lock.json"), `{"lockfileVersion":3}`)

	inventory, err := collectRepositoryInventory(root)
	if err != nil {
		t.Fatalf("collectRepositoryInventory() error = %v", err)
	}

	for _, want := range []string{"apps/cli/go.mod", "apps/server/go.mod", "apps/web/package.json", "apps/web/package-lock.json"} {
		if !stringSliceContains(inventory.detectedPaths, want) {
			t.Fatalf("detected paths = %#v, want %q", inventory.detectedPaths, want)
		}
	}
	for _, want := range []string{"apps/cli", "apps/server", "apps/web"} {
		if !stringSliceContains(inventory.workspaceCandidates, want) {
			t.Fatalf("workspace candidates = %#v, want %q", inventory.workspaceCandidates, want)
		}
	}
	if !stringSliceContains(inventory.detectedToolchains, "go") || !stringSliceContains(inventory.detectedToolchains, "node") {
		t.Fatalf("toolchains = %#v, want go and node from workspace manifests", inventory.detectedToolchains)
	}
	if !stringSliceContains(inventory.detectedPackageManagers, "npm") {
		t.Fatalf("package managers = %#v, want npm from workspace lockfile", inventory.detectedPackageManagers)
	}
}

func writeInventoryFile(t *testing.T, root string, relative string, content string) {
	t.Helper()

	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create %s parent: %v", relative, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", relative, err)
	}
}
