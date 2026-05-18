package executionrunner

import "time"

type executionLeaseCreateRequest struct {
	ProjectID     string `json:"project_id"`
	RepoBindingID string `json:"repo_binding_id"`
	RunnerID      string `json:"runner_id"`
	TTLSeconds    int    `json:"ttl_seconds,omitempty"`
}

type executionLease struct {
	ID                string       `json:"id"`
	ExecutionJobID    string       `json:"execution_job_id"`
	TaskID            string       `json:"task_id"`
	CheckoutReceiptID string       `json:"checkout_receipt_id"`
	RepoBindingID     string       `json:"repo_binding_id"`
	RunnerID          string       `json:"runner_id"`
	State             string       `json:"state"`
	LeaseToken        string       `json:"lease_token"`
	ExpiresAt         time.Time    `json:"expires_at"`
	ExecutionJob      executionJob `json:"execution_job"`
}

type executionJob struct {
	ID                string `json:"id"`
	TaskID            string `json:"task_id"`
	RepoBindingID     string `json:"repo_binding_id"`
	CheckoutReceiptID string `json:"checkout_receipt_id"`
	State             string `json:"state"`
}

type runStartRequest struct {
	LeaseID    string `json:"lease_id"`
	LeaseToken string `json:"lease_token"`
	RunnerID   string `json:"runner_id"`
}

type runStarted struct {
	ID                string `json:"id"`
	ExecutionJobID    string `json:"execution_job_id"`
	ExecutionLeaseID  string `json:"execution_lease_id"`
	TaskID            string `json:"task_id"`
	CheckoutReceiptID string `json:"checkout_receipt_id"`
	RunnerID          string `json:"runner_id"`
	State             string `json:"state"`
}

type executionCommandPlanRequest struct {
	ProjectID     string `json:"project_id"`
	RepoBindingID string `json:"repo_binding_id"`
	CommandKind   string `json:"command_kind,omitempty"`
	Action        string `json:"action,omitempty"`
}

type executionCommandPlan struct {
	ID                          string                           `json:"id"`
	ProjectID                   string                           `json:"project_id"`
	RepoBindingID               string                           `json:"repo_binding_id"`
	TaskID                      string                           `json:"task_id"`
	CheckoutReceiptID           string                           `json:"checkout_receipt_id"`
	ExecutionJobID              string                           `json:"execution_job_id"`
	RunID                       string                           `json:"run_id"`
	CommandKind                 string                           `json:"command_kind"`
	Action                      string                           `json:"action"`
	SourceProjectProbeReceiptID string                           `json:"source_project_probe_receipt_id,omitempty"`
	SelectedTargetID            string                           `json:"selected_target_id,omitempty"`
	DeclaredTestTarget          *projectProbeTestTargetCandidate `json:"declared_test_target,omitempty"`
	ShellAllowed                bool                             `json:"shell_allowed"`
	Argv                        []string                         `json:"argv"`
	WorkingDirectory            string                           `json:"working_directory"`
	PathScope                   []string                         `json:"path_scope"`
	TimeoutSeconds              int                              `json:"timeout_seconds"`
	NetworkAllowed              bool                             `json:"network_allowed"`
	WorkspaceWriteAllowed       bool                             `json:"workspace_write_allowed"`
	ScratchWriteAllowed         bool                             `json:"scratch_write_allowed"`
	MaxStdoutBytes              int                              `json:"max_stdout_bytes"`
	MaxStderrBytes              int                              `json:"max_stderr_bytes"`
	AllowedArtifactKinds        []string                         `json:"allowed_artifact_kinds"`
	ChangedPathsAllowed         bool                             `json:"changed_paths_allowed"`
	RawSourceUploadAllowed      bool                             `json:"raw_source_upload_allowed"`
	State                       string                           `json:"state"`
}

type executionReceiptRequest struct {
	ExecutionJobID       string                `json:"execution_job_id"`
	LeaseID              string                `json:"lease_id"`
	LeaseToken           string                `json:"lease_token"`
	RunnerID             string                `json:"runner_id"`
	WorkspaceRef         string                `json:"workspace_ref"`
	CommitSHA            string                `json:"commit_sha"`
	BaselineID           string                `json:"baseline_id,omitempty"`
	OverlayID            string                `json:"overlay_id,omitempty"`
	ExecutionMode        string                `json:"execution_mode"`
	CommandPlanID        string                `json:"command_plan_id,omitempty"`
	CommandKind          string                `json:"command_kind,omitempty"`
	Action               string                `json:"action,omitempty"`
	ProcessStatus        string                `json:"process_status"`
	ExitCode             *int                  `json:"exit_code,omitempty"`
	ArtifactRefs         []string              `json:"artifact_refs"`
	ChangedPathsSummary  []string              `json:"changed_paths_summary"`
	RawSourceUploaded    bool                  `json:"raw_source_uploaded"`
	RunnerStartedAt      *time.Time            `json:"runner_started_at,omitempty"`
	RunnerFinishedAt     *time.Time            `json:"runner_finished_at,omitempty"`
	ProjectProbeMetadata *projectProbeMetadata `json:"project_probe_metadata,omitempty"`
	EnforcementReport    *enforcementReport    `json:"enforcement_report,omitempty"`
}

type executionReceipt struct {
	ID             string `json:"id"`
	RunID          string `json:"run_id"`
	ExecutionJobID string `json:"execution_job_id"`
	RunnerID       string `json:"runner_id"`
	ExecutionMode  string `json:"execution_mode"`
	CommandPlanID  string `json:"command_plan_id,omitempty"`
	CommandKind    string `json:"command_kind,omitempty"`
	Action         string `json:"action,omitempty"`
	ProcessStatus  string `json:"process_status"`
	ExitCode       *int   `json:"exit_code,omitempty"`
}

type runnerCapabilityReportRequest struct {
	RunnerID                        string `json:"runner_id"`
	ProjectID                       string `json:"project_id"`
	RepoBindingID                   string `json:"repo_binding_id"`
	NetworkIsolationDeclared        bool   `json:"network_isolation_declared"`
	WorkspaceWriteIsolationDeclared bool   `json:"workspace_write_isolation_declared"`
	ProcessTreeControlDeclared      bool   `json:"process_tree_control_declared"`
	StdoutStderrPolicyDeclared      bool   `json:"stdout_stderr_policy_declared"`
	ArtifactPolicyDeclared          bool   `json:"artifact_policy_declared"`
	TrustState                      string `json:"trust_state"`
}

type runnerCapabilityReport struct {
	ID            string `json:"id"`
	RunnerID      string `json:"runner_id"`
	ProjectID     string `json:"project_id"`
	RepoBindingID string `json:"repo_binding_id"`
	TrustState    string `json:"trust_state"`
}

type enforcementReport struct {
	NetworkPolicy             string `json:"network_policy"`
	NetworkEnforcement        string `json:"network_enforcement"`
	WorkspaceWritePolicy      string `json:"workspace_write_policy"`
	WorkspaceWriteEnforcement string `json:"workspace_write_enforcement"`
	ProcessTreeEnforcement    string `json:"process_tree_enforcement"`
	ScratchWritePolicy        string `json:"scratch_write_policy,omitempty"`
	Decision                  string `json:"decision"`
	Reason                    string `json:"reason"`
}

type projectProbeMetadata struct {
	DetectedManifests            []projectProbeManifest                `json:"detected_manifests"`
	PackageManagerCandidates     []projectProbePackageManagerCandidate `json:"package_manager_candidates"`
	DeclaredTestTargetCandidates []projectProbeTestTargetCandidate     `json:"declared_test_target_candidates"`
	UnsupportedOrUnknowns        []string                              `json:"unsupported_or_unknowns"`
	PartialityReasons            []string                              `json:"partiality_reasons"`
}

type projectProbeManifest struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
}

type projectProbePackageManagerCandidate struct {
	Name       string `json:"name"`
	SourcePath string `json:"source_path"`
}

type projectProbeTestTargetCandidate struct {
	Name       string `json:"name"`
	SourcePath string `json:"source_path"`
	SourceKind string `json:"source_kind"`
}
