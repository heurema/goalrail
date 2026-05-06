package projectscan

import (
	"context"
	"strings"
	"time"
)

type OverlayOptions struct {
	Now func() time.Time
}

func BuildOverlay(ctx context.Context, workDir string, repoBindingID string, baseline *RepositoryBaselineProfile, options OverlayOptions) (WorkspaceOverlay, string, error) {
	facts, err := DiscoverGit(ctx, workDir)
	if err != nil {
		return WorkspaceOverlay{}, "", err
	}
	rawStatus, err := gitStatusPorcelainV2(ctx, facts.CanonicalRepoRoot)
	if err != nil {
		return WorkspaceOverlay{}, "", err
	}
	if rawStatus != "" && !strings.HasSuffix(rawStatus, "\n") {
		rawStatus += "\n"
	}

	parsed := parsePorcelainV2(rawStatus)
	partialityReasons := []string{}
	if facts.SparseCheckout {
		partialityReasons = append(partialityReasons, "sparse_checkout")
	}
	if facts.ShallowRepository {
		partialityReasons = append(partialityReasons, "shallow_repository")
	}
	if facts.SubmodulesPresent {
		partialityReasons = append(partialityReasons, "submodules_present")
	}
	partialityReasons = uniqueSorted(partialityReasons)

	state := OverlayStateClean
	switch {
	case len(parsed.unmergedPaths) > 0:
		state = OverlayStateUnmerged
	case len(parsed.changedPaths) > 0:
		state = OverlayStateDirty
	case len(partialityReasons) > 0:
		state = OverlayStatePartial
	}

	baselineID := ""
	baseHeadSHA := facts.HeadSHA
	if baseline != nil {
		baselineID = baseline.RepositoryBaselineProfileID
		baseHeadSHA = baseline.HeadSHA
	}
	now := time.Now
	if options.Now != nil {
		now = options.Now
	}
	statusHash := hashString(rawStatus)
	overlay := WorkspaceOverlay{
		WorkspaceOverlayID:          workspaceOverlayID(repoBindingID, baselineID, facts.CanonicalRepoRoot, baseHeadSHA, statusHash),
		RepositoryBaselineProfileID: baselineID,
		RepoBindingID:               repoBindingID,
		CanonicalRepoRoot:           facts.CanonicalRepoRoot,
		BaseHeadSHA:                 baseHeadSHA,
		CreatedAt:                   now().UTC().Format(time.RFC3339),
		State:                       state,
		ChangedPaths:                parsed.changedPaths,
		ScanCriticalChangedPaths:    ScanCriticalChangedPaths(parsed.changedPaths),
		UnmergedPaths:               parsed.unmergedPaths,
		UntrackedVisibility:         VisibilityNotChecked,
		IgnoredVisibility:           VisibilityNotChecked,
		SubmoduleFlags:              parsed.submoduleFlags,
		PartialityReasons:           partialityReasons,
		RawStatusReceiptRef:         statusReceiptFile,
	}
	return overlay, rawStatus, nil
}

type porcelainStatus struct {
	changedPaths   []string
	unmergedPaths  []string
	submoduleFlags []string
}

func parsePorcelainV2(raw string) porcelainStatus {
	changed := []string{}
	unmerged := []string{}
	submodules := []string{}
	for _, line := range strings.Split(raw, "\n") {
		if line == "" || strings.HasPrefix(line, "# ") {
			continue
		}
		switch line[0] {
		case '1':
			path, submodule := parseOrdinaryPorcelainLine(line)
			if path != "" {
				changed = append(changed, path)
			}
			if submodule != "" {
				submodules = append(submodules, path+":"+submodule)
			}
		case '2':
			path, submodule := parseRenamedPorcelainLine(line)
			if path != "" {
				changed = append(changed, path)
			}
			if submodule != "" {
				submodules = append(submodules, path+":"+submodule)
			}
		case 'u':
			path := parseUnmergedPorcelainLine(line)
			if path != "" {
				changed = append(changed, path)
				unmerged = append(unmerged, path)
			}
		case '?', '!':
			// Untracked and ignored paths are deliberately not requested in v0.
			continue
		}
	}
	return porcelainStatus{
		changedPaths:   uniqueSorted(changed),
		unmergedPaths:  uniqueSorted(unmerged),
		submoduleFlags: uniqueSorted(submodules),
	}
}

func parseOrdinaryPorcelainLine(line string) (string, string) {
	fields := strings.SplitN(line, " ", 9)
	if len(fields) < 9 {
		return "", ""
	}
	return normalizePorcelainPath(fields[8]), normalizeSubmoduleFlag(fields[2])
}

func parseRenamedPorcelainLine(line string) (string, string) {
	fields := strings.SplitN(line, " ", 10)
	if len(fields) < 10 {
		return "", ""
	}
	pathPart, _, _ := strings.Cut(fields[9], "\t")
	return normalizePorcelainPath(pathPart), normalizeSubmoduleFlag(fields[2])
}

func parseUnmergedPorcelainLine(line string) string {
	fields := strings.SplitN(line, " ", 11)
	if len(fields) < 11 {
		return ""
	}
	return normalizePorcelainPath(fields[10])
}

func normalizePorcelainPath(value string) string {
	value, _, _ = strings.Cut(value, "\t")
	return normalizeRelativePath(value)
}

func normalizeSubmoduleFlag(value string) string {
	if value == "" || value == "N..." {
		return ""
	}
	return value
}
