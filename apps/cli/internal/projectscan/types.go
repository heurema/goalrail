package projectscan

const SchemaVersion = 1

const (
	BaselineStatusQuick   = "quick"
	BaselineStatusPartial = "partial"
	BaselineStatusStale   = "stale"
	BaselineStatusError   = "error"

	OverlayStateClean    = "clean"
	OverlayStateDirty    = "dirty"
	OverlayStateUnmerged = "unmerged"
	OverlayStatePartial  = "partial"
	OverlayStateUnknown  = "unknown"

	VisibilityNotChecked = "not_checked"
	VisibilityChecked    = "checked"
	VisibilityPartial    = "partial"

	FreshnessFresh             = "fresh"
	FreshnessMissingBaseline   = "missing_baseline"
	FreshnessStaleHead         = "stale_head"
	FreshnessSchemaMismatch    = "schema_mismatch"
	FreshnessDirtyOverlay      = "dirty_overlay"
	FreshnessScanCriticalDirty = "scan_critical_dirty"
	FreshnessUnmergedBlocking  = "unmerged_blocking"
	FreshnessPartial           = "partial"
)

type RepositoryBaselineProfile struct {
	RepositoryBaselineProfileID string           `json:"repository_baseline_profile_id"`
	RepoBindingID               string           `json:"repo_binding_id"`
	CanonicalRepoRoot           string           `json:"canonical_repo_root"`
	HeadSHA                     string           `json:"head_sha"`
	SchemaVersion               int              `json:"schema_version"`
	Status                      string           `json:"status"`
	ScanBudget                  ScanBudget       `json:"scan_budget"`
	Shape                       RepositoryShape  `json:"shape"`
	ReadinessSignals            ReadinessSignals `json:"readiness_signals"`
	Partiality                  Partiality       `json:"partiality"`
	Receipts                    ScanReceipts     `json:"receipts"`
}

type ScanBudget struct {
	ElapsedMS                             int `json:"elapsed_ms"`
	FileLimit                             int `json:"file_limit"`
	ByteLimit                             int `json:"byte_limit"`
	MaxChildDirectoriesPerWorkspaceParent int `json:"max_child_directories_per_workspace_parent"`
	FilesScanned                          int `json:"files_scanned"`
	BytesRead                             int `json:"bytes_read"`
}

type RepositoryShape struct {
	Workspaces           []string `json:"workspaces"`
	Toolchains           []string `json:"toolchains"`
	PackageManagers      []string `json:"package_managers"`
	EntrypointCandidates []string `json:"entrypoint_candidates"`
}

type ReadinessSignals struct {
	Tests        []string `json:"tests"`
	CI           []string `json:"ci"`
	AgentRules   []string `json:"agent_rules"`
	Codeowners   []string `json:"codeowners"`
	ProofSurface string   `json:"proof_surface"`
}

type Partiality struct {
	SparseCheckout    bool     `json:"sparse_checkout"`
	ShallowRepository bool     `json:"shallow_repository"`
	SubmodulesPresent bool     `json:"submodules_present"`
	Truncated         bool     `json:"truncated"`
	Reasons           []string `json:"reasons"`
}

type ScanReceipts struct {
	Scanned []string      `json:"scanned"`
	Skipped []SkipReceipt `json:"skipped"`
	Hashes  []HashReceipt `json:"hashes"`
}

type SkipReceipt struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type HashReceipt struct {
	Name      string `json:"name"`
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
}

type WorkspaceOverlay struct {
	WorkspaceOverlayID          string   `json:"workspace_overlay_id"`
	RepositoryBaselineProfileID string   `json:"repository_baseline_profile_id"`
	RepoBindingID               string   `json:"repo_binding_id"`
	CanonicalRepoRoot           string   `json:"canonical_repo_root"`
	BaseHeadSHA                 string   `json:"base_head_sha"`
	CreatedAt                   string   `json:"created_at"`
	State                       string   `json:"state"`
	ChangedPaths                []string `json:"changed_paths"`
	ScanCriticalChangedPaths    []string `json:"scan_critical_changed_paths"`
	UnmergedPaths               []string `json:"unmerged_paths"`
	UntrackedVisibility         string   `json:"untracked_visibility"`
	IgnoredVisibility           string   `json:"ignored_visibility"`
	SubmoduleFlags              []string `json:"submodule_flags"`
	PartialityReasons           []string `json:"partiality_reasons"`
	RawStatusReceiptRef         string   `json:"raw_status_receipt_ref"`
}

type FreshnessResult struct {
	Status                      string   `json:"status"`
	Reasons                     []string `json:"reasons"`
	StructuralRescanRecommended bool     `json:"structural_rescan_recommended"`
	BaselineRebuildRecommended  bool     `json:"baseline_rebuild_recommended"`
	BlocksExecutionOrProof      bool     `json:"blocks_execution_or_proof"`
}

type GitFacts struct {
	CanonicalRepoRoot string
	CanonicalGitDir   string
	HeadSHA           string
	Branch            string
	Detached          bool
	ShallowRepository bool
	SparseCheckout    bool
	SubmodulesPresent bool
}
