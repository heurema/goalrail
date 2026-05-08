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
	ID                     string   `json:"id"`
	ProjectID              string   `json:"project_id"`
	RepoBindingID          string   `json:"repo_binding_id"`
	TaskID                 string   `json:"task_id"`
	CheckoutReceiptID      string   `json:"checkout_receipt_id"`
	ExecutionJobID         string   `json:"execution_job_id"`
	RunID                  string   `json:"run_id"`
	CommandKind            string   `json:"command_kind"`
	Action                 string   `json:"action"`
	ShellAllowed           bool     `json:"shell_allowed"`
	Argv                   []string `json:"argv"`
	WorkingDirectory       string   `json:"working_directory"`
	PathScope              []string `json:"path_scope"`
	TimeoutSeconds         int      `json:"timeout_seconds"`
	MaxStdoutBytes         int      `json:"max_stdout_bytes"`
	MaxStderrBytes         int      `json:"max_stderr_bytes"`
	AllowedArtifactKinds   []string `json:"allowed_artifact_kinds"`
	RawSourceUploadAllowed bool     `json:"raw_source_upload_allowed"`
	State                  string   `json:"state"`
}

type executionReceiptRequest struct {
	ExecutionJobID      string     `json:"execution_job_id"`
	LeaseID             string     `json:"lease_id"`
	LeaseToken          string     `json:"lease_token"`
	RunnerID            string     `json:"runner_id"`
	WorkspaceRef        string     `json:"workspace_ref"`
	CommitSHA           string     `json:"commit_sha"`
	BaselineID          string     `json:"baseline_id,omitempty"`
	OverlayID           string     `json:"overlay_id,omitempty"`
	ExecutionMode       string     `json:"execution_mode"`
	CommandPlanID       string     `json:"command_plan_id,omitempty"`
	CommandKind         string     `json:"command_kind,omitempty"`
	Action              string     `json:"action,omitempty"`
	ProcessStatus       string     `json:"process_status"`
	ArtifactRefs        []string   `json:"artifact_refs"`
	ChangedPathsSummary []string   `json:"changed_paths_summary"`
	RawSourceUploaded   bool       `json:"raw_source_uploaded"`
	RunnerStartedAt     *time.Time `json:"runner_started_at,omitempty"`
	RunnerFinishedAt    *time.Time `json:"runner_finished_at,omitempty"`
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
}
