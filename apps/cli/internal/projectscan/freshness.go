package projectscan

func EvaluateFreshness(currentHeadSHA string, baseline *RepositoryBaselineProfile, overlay WorkspaceOverlay) FreshnessResult {
	if overlay.State == OverlayStateUnmerged || len(overlay.UnmergedPaths) > 0 {
		return FreshnessResult{
			Status:                 FreshnessUnmergedBlocking,
			Reasons:                []string{"unmerged_paths"},
			BlocksExecutionOrProof: true,
		}
	}
	if baseline == nil || baseline.RepositoryBaselineProfileID == "" {
		return FreshnessResult{
			Status:                     FreshnessMissingBaseline,
			Reasons:                    []string{"baseline_missing"},
			BaselineRebuildRecommended: true,
		}
	}
	if baseline.SchemaVersion != SchemaVersion {
		return FreshnessResult{
			Status:                     FreshnessSchemaMismatch,
			Reasons:                    []string{"schema_mismatch"},
			BaselineRebuildRecommended: true,
		}
	}
	if currentHeadSHA != "" && baseline.HeadSHA != currentHeadSHA {
		return FreshnessResult{
			Status:                     FreshnessStaleHead,
			Reasons:                    []string{"head_sha_mismatch"},
			BaselineRebuildRecommended: true,
		}
	}
	if len(overlay.ScanCriticalChangedPaths) > 0 {
		return FreshnessResult{
			Status:                      FreshnessScanCriticalDirty,
			Reasons:                     []string{"scan_critical_changed_paths"},
			StructuralRescanRecommended: true,
		}
	}
	if overlay.State == OverlayStateDirty || len(overlay.ChangedPaths) > 0 {
		return FreshnessResult{
			Status:  FreshnessDirtyOverlay,
			Reasons: []string{"workspace_dirty"},
		}
	}
	if baseline.Partiality.SparseCheckout || baseline.Partiality.ShallowRepository || baseline.Partiality.SubmodulesPresent || baseline.Partiality.Truncated || len(baseline.Partiality.Reasons) > 0 || overlay.State == OverlayStatePartial || len(overlay.PartialityReasons) > 0 {
		reasons := append([]string{}, baseline.Partiality.Reasons...)
		reasons = append(reasons, overlay.PartialityReasons...)
		if len(reasons) == 0 {
			reasons = append(reasons, "partial_repository_state")
		}
		return FreshnessResult{
			Status:  FreshnessPartial,
			Reasons: uniqueSorted(reasons),
		}
	}
	return FreshnessResult{
		Status:  FreshnessFresh,
		Reasons: []string{},
	}
}
