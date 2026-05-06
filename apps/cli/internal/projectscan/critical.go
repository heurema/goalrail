package projectscan

import (
	"path"
	"strings"
)

func ScanCriticalChangedPaths(paths []string) []string {
	matches := []string{}
	for _, candidate := range paths {
		relativePath := normalizeRelativePath(candidate)
		if IsScanCriticalPath(relativePath) {
			matches = append(matches, relativePath)
		}
	}
	return uniqueSorted(matches)
}

func IsScanCriticalPath(relativePath string) bool {
	relativePath = normalizeRelativePath(relativePath)
	if relativePath == "" {
		return false
	}

	base := path.Base(relativePath)
	if isWorkspaceManifest(base) {
		return true
	}
	switch base {
	case "pnpm-lock.yaml", "package-lock.json", "yarn.lock", "bun.lock", "bun.lockb", "Cargo.lock", "poetry.lock", "uv.lock", "Gemfile.lock", "composer.lock":
		return true
	case "pnpm-workspace.yaml", "turbo.json", "nx.json", "lerna.json", "rush.json", "workspace.json":
		return true
	case "AGENTS.md", "CLAUDE.md", "CODEOWNERS":
		return true
	}

	switch {
	case strings.HasPrefix(relativePath, ".github/workflows/"):
		return true
	case relativePath == ".github/copilot-instructions.md":
		return true
	case strings.HasPrefix(relativePath, ".cursor/rules/"):
		return true
	case relativePath == ".github/CODEOWNERS":
		return true
	case relativePath == ".gitmodules":
		return true
	case relativePath == ".goalrail/project-scan.yml":
		return true
	case relativePath == ".goalrail/project-scan.yaml":
		return true
	case relativePath == "goalrail.project-scan.yml":
		return true
	case relativePath == "goalrail.project-scan.yaml":
		return true
	default:
		return false
	}
}
