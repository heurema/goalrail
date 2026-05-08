package reposhape

import (
	"path"
	"sort"
	"strings"
)

var workspaceManifestPaths = []string{
	"Cargo.toml",
	"Gemfile",
	"composer.json",
	"go.mod",
	"go.work",
	"package.json",
	"pyproject.toml",
	"requirements.txt",
}

var rootInventoryPaths = []string{
	".github/workflows",
	"Cargo.lock",
	"Cargo.toml",
	"Dockerfile",
	"Gemfile",
	"Gemfile.lock",
	"Makefile",
	"README",
	"README.md",
	"Taskfile.yml",
	"bun.lock",
	"bun.lockb",
	"compose.yml",
	"composer.json",
	"composer.lock",
	"docker-compose.yml",
	"go.mod",
	"go.work",
	"package-lock.json",
	"package.json",
	"pnpm-lock.yaml",
	"poetry.lock",
	"pyproject.toml",
	"requirements.txt",
	"uv.lock",
	"yarn.lock",
}

var workspaceInventoryPaths = []string{
	"Cargo.lock",
	"Cargo.toml",
	"Dockerfile",
	"Gemfile",
	"Gemfile.lock",
	"Makefile",
	"Taskfile.yml",
	"bun.lock",
	"bun.lockb",
	"compose.yml",
	"composer.json",
	"composer.lock",
	"docker-compose.yml",
	"go.mod",
	"go.work",
	"package-lock.json",
	"package.json",
	"pnpm-lock.yaml",
	"poetry.lock",
	"pyproject.toml",
	"requirements.txt",
	"uv.lock",
	"yarn.lock",
}

// WorkspaceManifestPaths returns the file names that identify a workspace.
func WorkspaceManifestPaths() []string {
	return copyStrings(workspaceManifestPaths)
}

// RootInventoryPaths returns bounded root metadata paths for init-time inventory.
func RootInventoryPaths() []string {
	return copyStrings(rootInventoryPaths)
}

// WorkspaceInventoryPaths returns bounded metadata paths checked under workspace candidates.
func WorkspaceInventoryPaths() []string {
	return copyStrings(workspaceInventoryPaths)
}

// IsWorkspaceManifest reports whether name is a supported workspace manifest file name.
func IsWorkspaceManifest(name string) bool {
	switch baseName(name) {
	case "go.mod", "go.work", "package.json", "Cargo.toml", "pyproject.toml", "requirements.txt", "Gemfile", "composer.json":
		return true
	default:
		return false
	}
}

// ToolchainsForPath returns deterministic toolchain signals for a repository-relative path.
func ToolchainsForPath(relativePath string) []string {
	switch baseName(relativePath) {
	case "go.mod", "go.work":
		return []string{"go"}
	case "package.json":
		return []string{"node"}
	case "Cargo.toml":
		return []string{"rust"}
	case "pyproject.toml", "requirements.txt":
		return []string{"python"}
	case "Gemfile":
		return []string{"ruby"}
	case "composer.json":
		return []string{"php"}
	case "Dockerfile", "docker-compose.yml", "compose.yml":
		return []string{"docker"}
	default:
		return nil
	}
}

// PackageManagersForPath returns deterministic package-manager signals for a repository-relative path.
func PackageManagersForPath(relativePath string) []string {
	switch baseName(relativePath) {
	case "pnpm-lock.yaml":
		return []string{"pnpm"}
	case "package-lock.json":
		return []string{"npm"}
	case "yarn.lock":
		return []string{"yarn"}
	case "bun.lock", "bun.lockb":
		return []string{"bun"}
	case "Cargo.lock":
		return []string{"cargo"}
	case "poetry.lock":
		return []string{"poetry"}
	case "uv.lock":
		return []string{"uv"}
	case "requirements.txt":
		return []string{"pip"}
	case "Gemfile.lock":
		return []string{"bundler"}
	case "composer.lock":
		return []string{"composer"}
	default:
		return nil
	}
}

// IsCIWorkflowPath reports whether relativePath is inside .github/workflows/.
func IsCIWorkflowPath(relativePath string) bool {
	relativePath = normalizePath(relativePath)
	return strings.HasPrefix(relativePath, ".github/workflows/")
}

// IsEntrypointCandidate reports whether relativePath is a lightweight entrypoint marker.
func IsEntrypointCandidate(relativePath string) bool {
	relativePath = normalizePath(relativePath)
	base := path.Base(relativePath)
	switch base {
	case "Dockerfile", "docker-compose.yml", "compose.yml", "Makefile", "Taskfile.yml":
		return true
	}
	if strings.HasSuffix(relativePath, "/main.go") {
		parts := strings.Split(relativePath, "/")
		for i, part := range parts {
			if part == "cmd" && i+2 < len(parts) {
				return true
			}
		}
	}
	return false
}

func baseName(value string) string {
	relativePath := normalizePath(value)
	if relativePath == "" {
		return ""
	}
	return path.Base(relativePath)
}

func normalizePath(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" || value == "." {
		return ""
	}
	value = strings.TrimPrefix(value, "./")
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return ""
	}
	return cleaned
}

func copyStrings(values []string) []string {
	copied := append([]string(nil), values...)
	sort.Strings(copied)
	return copied
}
