package projectscan

import (
	"bytes"
	"context"
	"path"
	"strings"
)

const (
	defaultMaxFilesScanned                 = 5000
	defaultMaxBytesRead                    = 1 << 20
	defaultMaxChildDirectoriesPerWorkspace = 50
)

type BuildOptions struct {
	MaxFilesScanned                 int
	MaxBytesRead                    int
	MaxChildDirectoriesPerWorkspace int
}

func DefaultBuildOptions() BuildOptions {
	return BuildOptions{
		MaxFilesScanned:                 defaultMaxFilesScanned,
		MaxBytesRead:                    defaultMaxBytesRead,
		MaxChildDirectoriesPerWorkspace: defaultMaxChildDirectoriesPerWorkspace,
	}
}

func BuildBaseline(ctx context.Context, workDir string, repoBindingID string, options BuildOptions) (RepositoryBaselineProfile, error) {
	if options.MaxFilesScanned <= 0 {
		options.MaxFilesScanned = defaultMaxFilesScanned
	}
	if options.MaxBytesRead <= 0 {
		options.MaxBytesRead = defaultMaxBytesRead
	}
	if options.MaxChildDirectoriesPerWorkspace <= 0 {
		options.MaxChildDirectoriesPerWorkspace = defaultMaxChildDirectoriesPerWorkspace
	}

	facts, err := DiscoverGit(ctx, workDir)
	if err != nil {
		return RepositoryBaselineProfile{}, err
	}
	trackedPaths, err := gitTrackedPaths(ctx, facts.CanonicalRepoRoot)
	if err != nil {
		return RepositoryBaselineProfile{}, err
	}

	scan := baselineScanner{
		ctx:          ctx,
		root:         facts.CanonicalRepoRoot,
		options:      options,
		skippedDirs:  map[string]struct{}{},
		workspaceSet: map[string]struct{}{},
	}
	scan.scanPaths(trackedPaths)
	scan.detectPackageJSONTests()
	workspaces := scan.limitedWorkspaces()

	partiality := Partiality{
		SparseCheckout:    facts.SparseCheckout,
		ShallowRepository: facts.ShallowRepository,
		SubmodulesPresent: facts.SubmodulesPresent || scan.hasPath(".gitmodules"),
		Truncated:         scan.truncated,
	}
	if partiality.SparseCheckout {
		partiality.Reasons = append(partiality.Reasons, "sparse_checkout")
	}
	if partiality.ShallowRepository {
		partiality.Reasons = append(partiality.Reasons, "shallow_repository")
	}
	if partiality.SubmodulesPresent {
		partiality.Reasons = append(partiality.Reasons, "submodules_present")
	}
	if partiality.Truncated {
		partiality.Reasons = append(partiality.Reasons, "scan_budget_truncated")
	}
	partiality.Reasons = uniqueSorted(partiality.Reasons)

	status := BaselineStatusQuick
	if len(partiality.Reasons) > 0 {
		status = BaselineStatusPartial
	}

	scanned := sortedKeys(scan.scannedSet)
	hashes := []HashReceipt{
		hashStrings("scanned_paths", scanned),
		hashStrings("readiness_tests", sortedKeys(scan.tests)),
		hashStrings("ci_paths", sortedKeys(scan.ci)),
		hashStrings("agent_rule_paths", sortedKeys(scan.agentRules)),
		hashStrings("codeowners_paths", sortedKeys(scan.codeowners)),
	}

	return RepositoryBaselineProfile{
		RepositoryBaselineProfileID: BaselineProfileID(repoBindingID, facts.CanonicalRepoRoot, facts.HeadSHA, SchemaVersion),
		RepoBindingID:               repoBindingID,
		CanonicalRepoRoot:           facts.CanonicalRepoRoot,
		HeadSHA:                     facts.HeadSHA,
		SchemaVersion:               SchemaVersion,
		Status:                      status,
		ScanBudget: ScanBudget{
			ElapsedMS:                             0,
			FileLimit:                             options.MaxFilesScanned,
			ByteLimit:                             options.MaxBytesRead,
			MaxChildDirectoriesPerWorkspaceParent: options.MaxChildDirectoriesPerWorkspace,
			FilesScanned:                          len(scanned),
			BytesRead:                             scan.bytesRead,
		},
		Shape: RepositoryShape{
			Workspaces:           workspaces,
			Toolchains:           sortedKeys(scan.toolchains),
			PackageManagers:      sortedKeys(scan.packageManagers),
			EntrypointCandidates: sortedKeys(scan.entrypoints),
		},
		ReadinessSignals: ReadinessSignals{
			Tests:        sortedKeys(scan.tests),
			CI:           sortedKeys(scan.ci),
			AgentRules:   sortedKeys(scan.agentRules),
			Codeowners:   sortedKeys(scan.codeowners),
			ProofSurface: proofSurface(len(scan.tests) > 0, len(scan.ci) > 0),
		},
		Partiality: partiality,
		Receipts: ScanReceipts{
			Scanned: scanned,
			Skipped: scan.skipped,
			Hashes:  hashes,
		},
	}, nil
}

type baselineScanner struct {
	ctx             context.Context
	root            string
	options         BuildOptions
	truncated       bool
	bytesRead       int
	scannedSet      map[string]struct{}
	skipped         []SkipReceipt
	skippedDirs     map[string]struct{}
	workspaceSet    map[string]struct{}
	toolchains      map[string]struct{}
	packageManagers map[string]struct{}
	entrypoints     map[string]struct{}
	tests           map[string]struct{}
	ci              map[string]struct{}
	agentRules      map[string]struct{}
	codeowners      map[string]struct{}
}

func (s *baselineScanner) scanPaths(paths []string) {
	s.scannedSet = map[string]struct{}{}
	s.toolchains = map[string]struct{}{}
	s.packageManagers = map[string]struct{}{}
	s.entrypoints = map[string]struct{}{}
	s.tests = map[string]struct{}{}
	s.ci = map[string]struct{}{}
	s.agentRules = map[string]struct{}{}
	s.codeowners = map[string]struct{}{}

	for _, relativePath := range paths {
		if skippedDir := skippedDirectory(relativePath); skippedDir != "" {
			if _, ok := s.skippedDirs[skippedDir]; !ok {
				s.skippedDirs[skippedDir] = struct{}{}
				s.skipped = append(s.skipped, SkipReceipt{Path: skippedDir, Reason: "skipped_directory"})
			}
			continue
		}
		if len(s.scannedSet) >= s.options.MaxFilesScanned {
			s.truncated = true
			if !hasSkipReason(s.skipped, "scan_budget_file_limit") {
				s.skipped = append(s.skipped, SkipReceipt{Path: "*", Reason: "scan_budget_file_limit"})
			}
			continue
		}
		s.scannedSet[relativePath] = struct{}{}
		s.collectSignals(relativePath)
	}
}

func (s *baselineScanner) collectSignals(relativePath string) {
	base := path.Base(relativePath)
	dir := path.Dir(relativePath)
	if dir == "." {
		dir = "."
	}

	if isWorkspaceManifest(base) {
		appendUnique(s.workspaceSet, dir)
	}
	switch base {
	case "go.mod", "go.work":
		appendUnique(s.toolchains, "go")
	case "package.json":
		appendUnique(s.toolchains, "node")
	case "Cargo.toml":
		appendUnique(s.toolchains, "rust")
	case "pyproject.toml", "requirements.txt":
		appendUnique(s.toolchains, "python")
	case "Gemfile":
		appendUnique(s.toolchains, "ruby")
	case "composer.json":
		appendUnique(s.toolchains, "php")
	}

	switch base {
	case "pnpm-lock.yaml":
		appendUnique(s.packageManagers, "pnpm")
	case "package-lock.json":
		appendUnique(s.packageManagers, "npm")
	case "yarn.lock":
		appendUnique(s.packageManagers, "yarn")
	case "bun.lock", "bun.lockb":
		appendUnique(s.packageManagers, "bun")
	case "Cargo.lock":
		appendUnique(s.packageManagers, "cargo")
	case "poetry.lock":
		appendUnique(s.packageManagers, "poetry")
	case "uv.lock":
		appendUnique(s.packageManagers, "uv")
	case "requirements.txt":
		appendUnique(s.packageManagers, "pip")
	case "Gemfile.lock":
		appendUnique(s.packageManagers, "bundler")
	case "composer.lock":
		appendUnique(s.packageManagers, "composer")
	}

	if strings.HasSuffix(relativePath, "_test.go") || base == "pytest.ini" {
		appendUnique(s.tests, relativePath)
	}
	if strings.Contains(relativePath, "/tests/") || strings.HasPrefix(relativePath, "tests/") {
		appendUnique(s.tests, firstPathSegment(relativePath)+"/tests")
	}
	if strings.HasPrefix(relativePath, ".github/workflows/") {
		appendUnique(s.ci, relativePath)
	}
	if relativePath == "AGENTS.md" || relativePath == "CLAUDE.md" || relativePath == ".github/copilot-instructions.md" || strings.HasPrefix(relativePath, ".cursor/rules/") {
		appendUnique(s.agentRules, relativePath)
	}
	if relativePath == "CODEOWNERS" || relativePath == ".github/CODEOWNERS" {
		appendUnique(s.codeowners, relativePath)
	}
	if isEntrypointCandidate(relativePath) {
		appendUnique(s.entrypoints, relativePath)
	}
}

func (s *baselineScanner) detectPackageJSONTests() {
	for _, relativePath := range sortedKeys(s.scannedSet) {
		if path.Base(relativePath) != "package.json" {
			continue
		}
		size, err := gitBlobSize(s.ctx, s.root, relativePath)
		if err != nil {
			continue
		}
		if size > s.options.MaxBytesRead-s.bytesRead {
			s.truncated = true
			s.skipped = append(s.skipped, SkipReceipt{Path: relativePath, Reason: "scan_budget_byte_limit"})
			continue
		}
		raw, err := gitBlob(s.ctx, s.root, relativePath)
		if err != nil {
			continue
		}
		s.bytesRead += len(raw)
		if bytes.Contains(raw, []byte(`"test"`)) {
			appendUnique(s.tests, relativePath)
		}
	}
}

func (s *baselineScanner) limitedWorkspaces() []string {
	workspaces := sortedKeys(s.workspaceSet)
	byParent := map[string][]string{}
	for _, workspace := range workspaces {
		parent := firstPathSegment(workspace)
		if parent == "." || (parent != "apps" && parent != "packages" && parent != "services") {
			continue
		}
		byParent[parent] = append(byParent[parent], workspace)
	}
	allowed := map[string]struct{}{}
	for _, workspace := range workspaces {
		allowed[workspace] = struct{}{}
	}
	for parent, children := range byParent {
		sortStrings(children)
		if len(children) <= s.options.MaxChildDirectoriesPerWorkspace {
			continue
		}
		s.truncated = true
		for _, skipped := range children[s.options.MaxChildDirectoriesPerWorkspace:] {
			delete(allowed, skipped)
		}
		s.skipped = append(s.skipped, SkipReceipt{Path: parent, Reason: "workspace_child_directory_limit"})
	}
	return sortedKeys(allowed)
}

func (s *baselineScanner) hasPath(relativePath string) bool {
	_, ok := s.scannedSet[relativePath]
	return ok
}

func skippedDirectory(relativePath string) string {
	parts := strings.Split(relativePath, "/")
	for i, part := range parts {
		switch part {
		case ".git", "node_modules", "dist", "build", ".build", "coverage":
			return strings.Join(parts[:i+1], "/")
		}
	}
	return ""
}

func hasSkipReason(receipts []SkipReceipt, reason string) bool {
	for _, receipt := range receipts {
		if receipt.Reason == reason {
			return true
		}
	}
	return false
}

func isWorkspaceManifest(base string) bool {
	switch base {
	case "go.mod", "go.work", "package.json", "Cargo.toml", "pyproject.toml", "requirements.txt", "Gemfile", "composer.json":
		return true
	default:
		return false
	}
}

func isEntrypointCandidate(relativePath string) bool {
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

func firstPathSegment(relativePath string) string {
	if relativePath == "." {
		return "."
	}
	if before, _, ok := strings.Cut(relativePath, "/"); ok {
		return before
	}
	return relativePath
}

func proofSurface(hasTests bool, hasCI bool) string {
	switch {
	case hasTests && hasCI:
		return "strong"
	case hasTests || hasCI:
		return "partial"
	default:
		return "none"
	}
}
