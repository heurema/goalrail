package projectscan

import (
	"strings"

	"github.com/heurema/goalrail/apps/cli/internal/reposhape"
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

	if reposhape.IsWorkspaceManifest(relativePath) {
		return true
	}
	if len(reposhape.PackageManagersForPath(relativePath)) > 0 {
		return true
	}
	switch lastPathSegment(relativePath) {
	case "pnpm-workspace.yaml", "turbo.json", "nx.json", "lerna.json", "rush.json", "workspace.json":
		return true
	case "AGENTS.md", "CLAUDE.md", "CODEOWNERS":
		return true
	}

	switch {
	case reposhape.IsCIWorkflowPath(relativePath):
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

func lastPathSegment(relativePath string) string {
	if idx := strings.LastIndex(relativePath, "/"); idx >= 0 {
		return relativePath[idx+1:]
	}
	return relativePath
}
