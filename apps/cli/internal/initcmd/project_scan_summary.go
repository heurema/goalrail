package initcmd

import (
	"fmt"
	"strings"

	"github.com/heurema/goalrail/apps/cli/internal/projectscan"
)

const (
	projectScanArtifactCreated     = "created"
	projectScanArtifactRefreshed   = "refreshed"
	projectScanArtifactUnavailable = "unavailable"

	projectScanSignalDetected    = "detected"
	projectScanSignalNotDetected = "not_detected"
	projectScanSignalUnknown     = "unknown"

	projectScanFreshnessCurrentHead = "current_head"
	projectScanFreshnessStale       = "stale"
	projectScanFreshnessUnknown     = "unknown"
)

type initProjectScanSummary struct {
	Baseline        string
	Overlay         string
	Toolchains      []string
	PackageManagers []string
	Workspaces      []string
	Tests           string
	CI              string
	AgentRules      string
	Codeowners      string
	Partiality      []string
	Freshness       string
	Warnings        []string
}

func unavailableProjectScanSummary(warning string) initProjectScanSummary {
	summary := initProjectScanSummary{
		Baseline:        projectScanArtifactUnavailable,
		Overlay:         projectScanArtifactUnavailable,
		Tests:           projectScanSignalUnknown,
		CI:              projectScanSignalUnknown,
		AgentRules:      projectScanSignalUnknown,
		Codeowners:      projectScanSignalUnknown,
		Partiality:      []string{"not_checked"},
		Freshness:       projectScanFreshnessUnknown,
		Toolchains:      []string{projectScanSignalUnknown},
		PackageManagers: []string{projectScanSignalUnknown},
		Workspaces:      []string{projectScanSignalUnknown},
	}
	if strings.TrimSpace(warning) != "" {
		summary.Warnings = []string{strings.TrimSpace(warning)}
	}
	return summary
}

func summarizeInitProjectScan(
	baselineAction string,
	overlayAction string,
	baseline projectscan.RepositoryBaselineProfile,
	overlay projectscan.WorkspaceOverlay,
	freshness projectscan.FreshnessResult,
) initProjectScanSummary {
	partiality := projectScanPartiality(&baseline, &overlay)
	summary := initProjectScanSummary{
		Baseline:        valueOrUnknown(baselineAction),
		Overlay:         valueOrUnknown(overlayAction),
		Toolchains:      baseline.Shape.Toolchains,
		PackageManagers: baseline.Shape.PackageManagers,
		Workspaces:      baseline.Shape.Workspaces,
		Tests:           detectionStatus(baseline.ReadinessSignals.Tests),
		CI:              detectionStatus(baseline.ReadinessSignals.CI),
		AgentRules:      detectionStatus(baseline.ReadinessSignals.AgentRules),
		Codeowners:      detectionStatus(baseline.ReadinessSignals.Codeowners),
		Partiality:      partiality,
		Freshness:       projectScanFreshnessLabel(freshness),
	}
	summary.Warnings = projectScanSummaryWarnings(&baseline, &overlay, freshness, partiality)
	return summary
}

func writeProjectScanSummaryText(b *strings.Builder, summary initProjectScanSummary) {
	if summary.Baseline == "" && summary.Overlay == "" {
		return
	}
	b.WriteString("Project scan:\n")
	fmt.Fprintf(b, "  baseline: %s\n", valueOrUnknown(summary.Baseline))
	fmt.Fprintf(b, "  overlay: %s\n", valueOrUnknown(summary.Overlay))
	fmt.Fprintf(b, "  toolchains: %s\n", compactList(summary.Toolchains))
	fmt.Fprintf(b, "  package managers: %s\n", compactList(summary.PackageManagers))
	fmt.Fprintf(b, "  workspaces: %s\n", compactList(summary.Workspaces))
	fmt.Fprintf(b, "  tests: %s\n", valueOrUnknown(summary.Tests))
	fmt.Fprintf(b, "  ci: %s\n", valueOrUnknown(summary.CI))
	fmt.Fprintf(b, "  agent rules: %s\n", valueOrUnknown(summary.AgentRules))
	fmt.Fprintf(b, "  codeowners: %s\n", valueOrUnknown(summary.Codeowners))
	fmt.Fprintf(b, "  partiality: %s\n", compactList(summary.Partiality))
	fmt.Fprintf(b, "  freshness: %s\n", valueOrUnknown(summary.Freshness))
	if len(summary.Warnings) > 0 {
		fmt.Fprintf(b, "  warnings: %s\n", strings.Join(uniqueSortedStrings(summary.Warnings), "; "))
	}
}

func detectionStatus(values []string) string {
	if len(values) > 0 {
		return projectScanSignalDetected
	}
	return projectScanSignalNotDetected
}

func projectScanPartiality(baseline *projectscan.RepositoryBaselineProfile, overlay *projectscan.WorkspaceOverlay) []string {
	if baseline == nil && overlay == nil {
		return []string{"not_checked"}
	}
	var reasons []string
	if baseline != nil {
		if baseline.Partiality.ShallowRepository {
			reasons = append(reasons, "shallow_repo")
		}
		if baseline.Partiality.SparseCheckout {
			reasons = append(reasons, "sparse_checkout")
		}
		if baseline.Partiality.SubmodulesPresent {
			reasons = append(reasons, "submodules_present")
		}
		if baseline.Partiality.Truncated {
			reasons = append(reasons, "truncated")
		}
		reasons = appendMappedPartialityReasons(reasons, baseline.Partiality.Reasons)
	}
	if overlay != nil {
		if overlay.State == projectscan.OverlayStatePartial {
			reasons = append(reasons, "workspace_partial")
		}
		reasons = appendMappedPartialityReasons(reasons, overlay.PartialityReasons)
	}
	reasons = uniqueSortedStrings(reasons)
	if len(reasons) == 0 {
		return []string{"none"}
	}
	return reasons
}

func appendMappedPartialityReasons(out []string, reasons []string) []string {
	for _, reason := range reasons {
		switch strings.TrimSpace(reason) {
		case "":
			continue
		case "shallow_repository":
			out = append(out, "shallow_repo")
		case "scan_budget_truncated", "scan_budget_file_limit":
			out = append(out, "truncated")
		default:
			out = append(out, reason)
		}
	}
	return out
}

func projectScanFreshnessLabel(freshness projectscan.FreshnessResult) string {
	switch freshness.Status {
	case projectscan.FreshnessFresh,
		projectscan.FreshnessPartial,
		projectscan.FreshnessDirtyOverlay,
		projectscan.FreshnessScanCriticalDirty,
		projectscan.FreshnessUnmergedBlocking:
		return projectScanFreshnessCurrentHead
	case projectscan.FreshnessStaleHead, projectscan.FreshnessSchemaMismatch:
		return projectScanFreshnessStale
	case projectscan.FreshnessMissingBaseline, "":
		return projectScanFreshnessUnknown
	default:
		if freshness.BaselineRebuildRecommended {
			return projectScanFreshnessStale
		}
		return projectScanFreshnessUnknown
	}
}

func projectScanSummaryWarnings(
	baseline *projectscan.RepositoryBaselineProfile,
	overlay *projectscan.WorkspaceOverlay,
	freshness projectscan.FreshnessResult,
	partiality []string,
) []string {
	var warnings []string
	if len(partiality) > 0 && !(len(partiality) == 1 && partiality[0] == "none") {
		warnings = append(warnings, "partial scan: "+strings.Join(partiality, ", "))
	}
	switch freshness.Status {
	case projectscan.FreshnessStaleHead:
		warnings = append(warnings, "stale baseline: head_sha_mismatch")
	case projectscan.FreshnessSchemaMismatch:
		warnings = append(warnings, "stale baseline: schema_mismatch")
	case projectscan.FreshnessMissingBaseline:
		warnings = append(warnings, "baseline unavailable")
	case projectscan.FreshnessScanCriticalDirty:
		if overlay != nil && len(overlay.ScanCriticalChangedPaths) > 0 {
			warnings = append(warnings, "scan-critical changes: "+strings.Join(overlay.ScanCriticalChangedPaths, ", "))
		} else {
			warnings = append(warnings, "scan-critical changes present")
		}
	case projectscan.FreshnessUnmergedBlocking:
		if overlay != nil && len(overlay.UnmergedPaths) > 0 {
			warnings = append(warnings, "unmerged paths: "+strings.Join(overlay.UnmergedPaths, ", "))
		} else {
			warnings = append(warnings, "unmerged paths present")
		}
	case projectscan.FreshnessDirtyOverlay:
		warnings = append(warnings, "workspace has uncommitted changes")
	}
	if baseline != nil && baseline.Status == projectscan.BaselineStatusError {
		warnings = append(warnings, "baseline unavailable")
	}
	return uniqueSortedStrings(warnings)
}

func compactList(values []string) string {
	values = uniqueSortedStrings(values)
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func valueOrUnknown(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return projectScanSignalUnknown
	}
	return value
}

func uniqueSortedStrings(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
