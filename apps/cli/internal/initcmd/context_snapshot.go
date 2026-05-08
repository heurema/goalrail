package initcmd

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/heurema/goalrail/apps/cli/internal/reposhape"
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

	for _, candidate := range reposhape.RootInventoryPaths() {
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

	workflowFiles, err := immediateChildFiles(gitRoot, ".github/workflows", 25)
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

func collectWorkspaceManifestPaths(root string, workspace string, pathSet map[string]struct{}) error {
	for _, manifest := range reposhape.WorkspaceInventoryPaths() {
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

func addInventorySignals(pathSet map[string]struct{}, toolchains map[string]struct{}, packageManagers map[string]struct{}) {
	for path := range pathSet {
		cleanPath := strings.TrimSuffix(path, "/")
		for _, toolchain := range reposhape.ToolchainsForPath(cleanPath) {
			toolchains[toolchain] = struct{}{}
		}
		for _, packageManager := range reposhape.PackageManagersForPath(cleanPath) {
			packageManagers[packageManager] = struct{}{}
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
