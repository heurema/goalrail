package initcmd

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/heurema/goalrail/apps/cli/internal/spine"
)

const repositoryContextSnapshotSource = "goalrail_cli_init"

func buildRepositoryContextSnapshot(gitRoot string, output spine.RepositoryContextInitOutput, draft spine.RepoBindingDraft) (spine.RepositoryContextSnapshotRequest, error) {
	inventory, err := collectRepositoryInventory(gitRoot)
	if err != nil {
		return spine.RepositoryContextSnapshotRequest{}, err
	}
	return spine.RepositoryContextSnapshotRequest{
		Source:        repositoryContextSnapshotSource,
		SchemaVersion: 1,
		Repository: spine.RepositoryContextSnapshotRepository{
			Provider:              output.Provider,
			FullName:              output.RepositoryFullName,
			URL:                   output.RepositoryURL,
			ProviderDefaultBranch: output.ProviderDefaultBranch,
			WorkflowBaseBranch:    output.WorkflowBaseBranch,
			RemoteName:            draft.RemoteName,
			HeadSHA:               draft.HeadSHA,
		},
		DetectedPaths:           inventory.detectedPaths,
		DetectedToolchains:      inventory.detectedToolchains,
		DetectedPackageManagers: inventory.detectedPackageManagers,
		WorkspaceCandidates:     inventory.workspaceCandidates,
	}, nil
}

type repositoryInventory struct {
	detectedPaths           []string
	detectedToolchains      []string
	detectedPackageManagers []string
	workspaceCandidates     []string
}

func collectRepositoryInventory(gitRoot string) (repositoryInventory, error) {
	pathSet := map[string]struct{}{}
	toolchains := map[string]struct{}{}
	packageManagers := map[string]struct{}{}
	workspaces := map[string]struct{}{}

	for _, candidate := range knownRepositoryPaths() {
		if exists, isDir, err := relativePathExists(gitRoot, candidate); err != nil {
			return repositoryInventory{}, err
		} else if exists {
			path := candidate
			if isDir {
				path += "/"
			}
			pathSet[path] = struct{}{}
		}
	}

	for _, parent := range []string{"apps", "packages", "services"} {
		children, err := immediateChildDirectories(gitRoot, parent, 25)
		if err != nil {
			return repositoryInventory{}, err
		}
		for _, child := range children {
			pathSet[child+"/"] = struct{}{}
			workspaces[child] = struct{}{}
			if err := collectWorkspaceManifestPaths(gitRoot, child, pathSet); err != nil {
				return repositoryInventory{}, err
			}
		}
	}

	workflowFiles, err := immediateChildFiles(gitRoot, filepath.Join(".github", "workflows"), 25)
	if err != nil {
		return repositoryInventory{}, err
	}
	for _, workflow := range workflowFiles {
		pathSet[workflow] = struct{}{}
	}

	addInventorySignals(pathSet, toolchains, packageManagers)

	return repositoryInventory{
		detectedPaths:           sortedKeys(pathSet),
		detectedToolchains:      sortedKeys(toolchains),
		detectedPackageManagers: sortedKeys(packageManagers),
		workspaceCandidates:     sortedKeys(workspaces),
	}, nil
}

func knownRepositoryPaths() []string {
	return []string{
		"README.md",
		"README",
		"go.mod",
		"go.work",
		"package.json",
		"pnpm-lock.yaml",
		"package-lock.json",
		"yarn.lock",
		"bun.lock",
		"bun.lockb",
		"Cargo.toml",
		"pyproject.toml",
		"requirements.txt",
		"poetry.lock",
		"uv.lock",
		"Gemfile",
		"composer.json",
		"Dockerfile",
		"docker-compose.yml",
		"compose.yml",
		"Makefile",
		"Taskfile.yml",
		filepath.Join(".github", "workflows"),
	}
}

func collectWorkspaceManifestPaths(root string, workspace string, pathSet map[string]struct{}) error {
	for _, manifest := range workspaceManifestPaths() {
		path := filepath.ToSlash(filepath.Join(workspace, manifest))
		exists, isDir, err := relativePathExists(root, path)
		if err != nil {
			return err
		}
		if exists && !isDir {
			pathSet[path] = struct{}{}
		}
	}
	return nil
}

func workspaceManifestPaths() []string {
	return []string{
		"go.mod",
		"go.work",
		"package.json",
		"pnpm-lock.yaml",
		"package-lock.json",
		"yarn.lock",
		"bun.lock",
		"bun.lockb",
		"Cargo.toml",
		"pyproject.toml",
		"requirements.txt",
		"poetry.lock",
		"uv.lock",
		"Gemfile",
		"composer.json",
		"Dockerfile",
		"docker-compose.yml",
		"compose.yml",
		"Makefile",
		"Taskfile.yml",
	}
}

func addInventorySignals(pathSet map[string]struct{}, toolchains map[string]struct{}, packageManagers map[string]struct{}) {
	for path := range pathSet {
		name := filepath.Base(strings.TrimSuffix(path, "/"))
		switch name {
		case "go.mod", "go.work":
			toolchains["go"] = struct{}{}
		case "package.json":
			toolchains["node"] = struct{}{}
		case "Cargo.toml":
			toolchains["rust"] = struct{}{}
		case "pyproject.toml", "requirements.txt":
			toolchains["python"] = struct{}{}
		case "Dockerfile", "docker-compose.yml", "compose.yml":
			toolchains["docker"] = struct{}{}
		}

		switch name {
		case "pnpm-lock.yaml":
			packageManagers["pnpm"] = struct{}{}
		case "package-lock.json":
			packageManagers["npm"] = struct{}{}
		case "yarn.lock":
			packageManagers["yarn"] = struct{}{}
		case "bun.lock", "bun.lockb":
			packageManagers["bun"] = struct{}{}
		case "poetry.lock":
			packageManagers["poetry"] = struct{}{}
		case "uv.lock":
			packageManagers["uv"] = struct{}{}
		}
	}
}

func relativePathExists(root string, relative string) (bool, bool, error) {
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		if os.IsNotExist(err) {
			return false, false, nil
		}
		return false, false, err
	}
	return true, info.IsDir(), nil
}

func immediateChildDirectories(root string, relative string, limit int) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		paths = append(paths, filepath.ToSlash(filepath.Join(relative, name)))
	}
	sort.Strings(paths)
	if len(paths) > limit {
		paths = paths[:limit]
	}
	return paths, nil
}

func immediateChildFiles(root string, relative string, limit int) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		paths = append(paths, filepath.ToSlash(filepath.Join(relative, name)))
	}
	sort.Strings(paths)
	if len(paths) > limit {
		paths = paths[:limit]
	}
	return paths, nil
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
