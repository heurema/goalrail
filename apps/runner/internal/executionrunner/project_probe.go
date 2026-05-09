package executionrunner

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxProjectProbeManifestBytes int64 = 256 * 1024

type projectProbeManifestSpec struct {
	Path string
	Kind string
}

var projectProbeManifestAllowlist = []projectProbeManifestSpec{
	{Path: "package.json", Kind: "node_package_manifest"},
	{Path: "pnpm-workspace.yaml", Kind: "pnpm_workspace_manifest"},
	{Path: "pnpm-lock.yaml", Kind: "pnpm_lockfile"},
	{Path: "package-lock.json", Kind: "npm_lockfile"},
	{Path: "yarn.lock", Kind: "yarn_lockfile"},
	{Path: "bun.lock", Kind: "bun_lockfile"},
	{Path: "bun.lockb", Kind: "bun_lockfile"},
	{Path: "go.mod", Kind: "go_module_manifest"},
	{Path: "go.work", Kind: "go_workspace_manifest"},
	{Path: "pyproject.toml", Kind: "python_project_manifest"},
	{Path: "pytest.ini", Kind: "pytest_config"},
	{Path: "tox.ini", Kind: "tox_config"},
}

func detectDeclaredTestTargets(workspaceRoot string, plan executionCommandPlan) (projectProbeMetadata, error) {
	if plan.WorkingDirectory != "." {
		return projectProbeMetadata{}, fmt.Errorf("project probe working directory %q is not supported", plan.WorkingDirectory)
	}
	if len(plan.PathScope) != 1 || plan.PathScope[0] != "." {
		return projectProbeMetadata{}, fmt.Errorf("project probe path scope %#v is not supported", plan.PathScope)
	}
	root, err := safeWorkspaceRoot(workspaceRoot)
	if err != nil {
		return projectProbeMetadata{}, err
	}
	metadata := projectProbeMetadata{
		DetectedManifests:            []projectProbeManifest{},
		PackageManagerCandidates:     []projectProbePackageManagerCandidate{},
		DeclaredTestTargetCandidates: []projectProbeTestTargetCandidate{},
		UnsupportedOrUnknowns:        []string{},
		PartialityReasons: []string{
			"probe reads only allowlisted manifest files under path_scope",
		},
	}
	for _, spec := range projectProbeManifestAllowlist {
		fullPath := filepath.Join(root, filepath.FromSlash(spec.Path))
		info, err := os.Stat(fullPath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			metadata.UnsupportedOrUnknowns = append(metadata.UnsupportedOrUnknowns, spec.Path+" could not be inspected")
			continue
		}
		if info.IsDir() {
			metadata.UnsupportedOrUnknowns = append(metadata.UnsupportedOrUnknowns, spec.Path+" is a directory")
			continue
		}
		metadata.DetectedManifests = append(metadata.DetectedManifests, projectProbeManifest{
			Path: spec.Path,
			Kind: spec.Kind,
		})
		if info.Size() > maxProjectProbeManifestBytes {
			metadata.PartialityReasons = append(metadata.PartialityReasons, spec.Path+" exceeded probe size limit")
			continue
		}
		switch spec.Path {
		case "package.json":
			probePackageJSON(fullPath, &metadata)
		case "pnpm-workspace.yaml", "pnpm-lock.yaml":
			addPackageManagerCandidate(&metadata, "pnpm", spec.Path)
		case "package-lock.json":
			addPackageManagerCandidate(&metadata, "npm", spec.Path)
		case "yarn.lock":
			addPackageManagerCandidate(&metadata, "yarn", spec.Path)
		case "bun.lock", "bun.lockb":
			addPackageManagerCandidate(&metadata, "bun", spec.Path)
		case "go.mod":
			addPackageManagerCandidate(&metadata, "go", spec.Path)
			addTestTargetCandidate(&metadata, "go_package_tests", spec.Path, "go_module_manifest")
		case "go.work":
			addPackageManagerCandidate(&metadata, "go", spec.Path)
			addTestTargetCandidate(&metadata, "go_workspace_tests", spec.Path, "go_workspace_manifest")
		case "pyproject.toml":
			probeTextManifest(fullPath, spec.Path, "python", "python_project_manifest", &metadata)
		case "pytest.ini":
			addPackageManagerCandidate(&metadata, "python", spec.Path)
			addTestTargetCandidate(&metadata, "python_pytest", spec.Path, "pytest_config")
		case "tox.ini":
			addPackageManagerCandidate(&metadata, "python", spec.Path)
			addTestTargetCandidate(&metadata, "python_tox", spec.Path, "tox_config")
		}
	}
	if len(metadata.DetectedManifests) == 0 {
		metadata.UnsupportedOrUnknowns = append(metadata.UnsupportedOrUnknowns, "no allowlisted manifest files detected")
	}
	sortProjectProbeMetadata(&metadata)
	return metadata, nil
}

func safeWorkspaceRoot(workspaceRoot string) (string, error) {
	root := strings.TrimSpace(workspaceRoot)
	if root == "" {
		return "", errors.New("workspace root is required for project probe")
	}
	if !filepath.IsAbs(root) {
		return "", errors.New("workspace root must be absolute")
	}
	clean := filepath.Clean(root)
	info, err := os.Stat(clean)
	if err != nil {
		return "", fmt.Errorf("inspect workspace root: %w", err)
	}
	if !info.IsDir() {
		return "", errors.New("workspace root must be a directory")
	}
	return clean, nil
}

func probePackageJSON(path string, metadata *projectProbeMetadata) {
	payload, err := os.ReadFile(path)
	if err != nil {
		metadata.UnsupportedOrUnknowns = append(metadata.UnsupportedOrUnknowns, "package.json could not be read")
		return
	}
	var parsed struct {
		PackageManager string            `json:"packageManager"`
		Scripts        map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		metadata.UnsupportedOrUnknowns = append(metadata.UnsupportedOrUnknowns, "package.json could not be parsed as JSON")
		return
	}
	if manager := packageManagerName(parsed.PackageManager); manager != "" {
		addPackageManagerCandidate(metadata, manager, "package.json")
	} else {
		addPackageManagerCandidate(metadata, "npm", "package.json")
	}
	for name := range parsed.Scripts {
		if isTestScriptName(name) {
			addTestTargetCandidate(metadata, name, "package.json", "package_json_script")
		}
	}
}

func probeTextManifest(path string, sourcePath string, manager string, sourceKind string, metadata *projectProbeMetadata) {
	payload, err := os.ReadFile(path)
	if err != nil {
		metadata.UnsupportedOrUnknowns = append(metadata.UnsupportedOrUnknowns, sourcePath+" could not be read")
		return
	}
	addPackageManagerCandidate(metadata, manager, sourcePath)
	text := string(payload)
	if strings.Contains(text, "[tool.pytest") || strings.Contains(text, "pytest") {
		addTestTargetCandidate(metadata, "python_pytest", sourcePath, sourceKind)
		return
	}
	metadata.PartialityReasons = append(metadata.PartialityReasons, sourcePath+" test target detection is partial")
}

func packageManagerName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if before, _, ok := strings.Cut(value, "@"); ok && before != "" {
		return before
	}
	return value
}

func isTestScriptName(name string) bool {
	name = strings.TrimSpace(name)
	return name == "test" || strings.HasPrefix(name, "test:")
}

func addPackageManagerCandidate(metadata *projectProbeMetadata, name string, sourcePath string) {
	for _, existing := range metadata.PackageManagerCandidates {
		if existing.Name == name && existing.SourcePath == sourcePath {
			return
		}
	}
	metadata.PackageManagerCandidates = append(metadata.PackageManagerCandidates, projectProbePackageManagerCandidate{
		Name:       name,
		SourcePath: sourcePath,
	})
}

func addTestTargetCandidate(metadata *projectProbeMetadata, name string, sourcePath string, sourceKind string) {
	for _, existing := range metadata.DeclaredTestTargetCandidates {
		if existing.Name == name && existing.SourcePath == sourcePath && existing.SourceKind == sourceKind {
			return
		}
	}
	metadata.DeclaredTestTargetCandidates = append(metadata.DeclaredTestTargetCandidates, projectProbeTestTargetCandidate{
		Name:       name,
		SourcePath: sourcePath,
		SourceKind: sourceKind,
	})
}

func sortProjectProbeMetadata(metadata *projectProbeMetadata) {
	sort.Slice(metadata.DetectedManifests, func(i, j int) bool {
		return metadata.DetectedManifests[i].Path < metadata.DetectedManifests[j].Path
	})
	sort.Slice(metadata.PackageManagerCandidates, func(i, j int) bool {
		if metadata.PackageManagerCandidates[i].Name == metadata.PackageManagerCandidates[j].Name {
			return metadata.PackageManagerCandidates[i].SourcePath < metadata.PackageManagerCandidates[j].SourcePath
		}
		return metadata.PackageManagerCandidates[i].Name < metadata.PackageManagerCandidates[j].Name
	})
	sort.Slice(metadata.DeclaredTestTargetCandidates, func(i, j int) bool {
		if metadata.DeclaredTestTargetCandidates[i].SourcePath == metadata.DeclaredTestTargetCandidates[j].SourcePath {
			return metadata.DeclaredTestTargetCandidates[i].Name < metadata.DeclaredTestTargetCandidates[j].Name
		}
		return metadata.DeclaredTestTargetCandidates[i].SourcePath < metadata.DeclaredTestTargetCandidates[j].SourcePath
	})
	sort.Strings(metadata.UnsupportedOrUnknowns)
	sort.Strings(metadata.PartialityReasons)
}
