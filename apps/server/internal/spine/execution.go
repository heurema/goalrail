package spine

import (
	"encoding/json"
	"time"
)

type ExecutionJobID string

type ExecutionLeaseID string

type RunID string

type ExecutionCommandPlanID string

type ExecutionReceiptID string

type ExecutionJobState string

const (
	ExecutionJobStateQueued           ExecutionJobState = "queued"
	ExecutionJobStateLeased           ExecutionJobState = "leased"
	ExecutionJobStateRunStarted       ExecutionJobState = "run_started"
	ExecutionJobStateReceiptSubmitted ExecutionJobState = "receipt_submitted"
)

type ExecutionLeaseState string

const (
	ExecutionLeaseStateActive     ExecutionLeaseState = "active"
	ExecutionLeaseStateExpired    ExecutionLeaseState = "expired"
	ExecutionLeaseStateRunStarted ExecutionLeaseState = "run_started"
)

type RunState string

const (
	RunStateStarted          RunState = "started"
	RunStateReceiptSubmitted RunState = "receipt_submitted"
)

const (
	ExecutionReceiptModeNoCommand          = "no_command"
	ExecutionReceiptModeBuiltinDiagnostic  = "builtin_diagnostic"
	ExecutionReceiptModeProjectProbe       = "project_probe"
	ExecutionReceiptStatusNotExecuted      = "not_executed"
	ExecutionReceiptStatusMetadataOnly     = "metadata_only"
	ExecutionReceiptNextActionGateReview   = "gate_review"
	ExecutionReceiptNextActionPlannedSlice = "I"
)

const (
	ExecutionCommandPlanNextActionRunnerProjectTestRequired = "runner_project_test_required"
	ExecutionCommandPlanNextActionProjectTestPlannedSlice   = "H2.6.2"
)

type ExecutionCommandPlanState string

const (
	ExecutionCommandPlanStatePlanned ExecutionCommandPlanState = "planned"
)

const (
	ExecutionCommandKindBuiltinDiagnostic   = "builtin_diagnostic"
	ExecutionCommandActionWorkspaceStatus   = "workspace_status"
	ExecutionCommandKindProjectProbe        = "project_probe"
	ExecutionCommandActionDetectTestTargets = "detect_declared_test_targets"
	ExecutionCommandKindProjectTest         = "project_test"
	ExecutionCommandActionRunTestTarget     = "run_declared_test_target"
)

type ExecutionJob struct {
	ID                 ExecutionJobID         `json:"id"`
	OrganizationID     OrganizationID         `json:"-"`
	ProjectID          ProjectID              `json:"-"`
	TaskID             WorkItemID             `json:"task_id"`
	ContractID         ContractID             `json:"contract_id"`
	ApprovedContractID ApprovedContractID     `json:"approved_contract_id"`
	PlanID             WorkItemPlanID         `json:"plan_id"`
	ProposalID         WorkItemPlanProposalID `json:"proposal_id"`
	RepoBindingID      RepoBindingID          `json:"repo_binding_id"`
	CheckoutJobID      CheckoutJobID          `json:"checkout_job_id"`
	CheckoutReceiptID  CheckoutReceiptID      `json:"checkout_receipt_id"`
	State              ExecutionJobState      `json:"state"`
	RequestedBy        ActorRef               `json:"requested_by"`
	ExecutionMode      string                 `json:"execution_mode"`
	CurrentLeaseID     *ExecutionLeaseID      `json:"current_lease_id,omitempty"`
	CurrentRunnerID    string                 `json:"current_runner_id,omitempty"`
	LeaseTokenHash     string                 `json:"-"`
	LeaseExpiresAt     *time.Time             `json:"lease_expires_at,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

type ExecutionJobCreateRequest struct {
	CheckoutReceiptID CheckoutReceiptID `json:"checkout_receipt_id"`
	ProjectID         ProjectID         `json:"project_id,omitempty"`
	RepoBindingID     RepoBindingID     `json:"repo_binding_id,omitempty"`
	RequestedBy       ActorRef          `json:"requested_by,omitempty"`
}

type ExecutionJobLeaseCreateRequest struct {
	ProjectID     ProjectID     `json:"project_id,omitempty"`
	RepoBindingID RepoBindingID `json:"repo_binding_id,omitempty"`
	RunnerID      string        `json:"runner_id"`
	TTLSeconds    int           `json:"ttl_seconds,omitempty"`
}

type ExecutionLease struct {
	ID                ExecutionLeaseID    `json:"id"`
	ExecutionJobID    ExecutionJobID      `json:"execution_job_id"`
	TaskID            WorkItemID          `json:"task_id"`
	CheckoutReceiptID CheckoutReceiptID   `json:"checkout_receipt_id"`
	RepoBindingID     RepoBindingID       `json:"repo_binding_id"`
	RunnerID          string              `json:"runner_id"`
	State             ExecutionLeaseState `json:"state"`
	LeaseTokenHash    string              `json:"-"`
	ExpiresAt         time.Time           `json:"expires_at"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
}

type ExecutionJobLeaseCreated struct {
	ID                ExecutionLeaseID    `json:"id"`
	ExecutionJobID    ExecutionJobID      `json:"execution_job_id"`
	TaskID            WorkItemID          `json:"task_id"`
	CheckoutReceiptID CheckoutReceiptID   `json:"checkout_receipt_id"`
	RepoBindingID     RepoBindingID       `json:"repo_binding_id"`
	RunnerID          string              `json:"runner_id"`
	State             ExecutionLeaseState `json:"state"`
	LeaseToken        string              `json:"lease_token"`
	ExpiresAt         time.Time           `json:"expires_at"`
	ExecutionJob      ExecutionJob        `json:"execution_job"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
}

type RunStartRequest struct {
	LeaseID    ExecutionLeaseID `json:"lease_id"`
	LeaseToken string           `json:"lease_token"`
	RunnerID   string           `json:"runner_id"`
}

type Run struct {
	ID                RunID             `json:"id"`
	ExecutionJobID    ExecutionJobID    `json:"execution_job_id"`
	ExecutionLeaseID  ExecutionLeaseID  `json:"execution_lease_id"`
	TaskID            WorkItemID        `json:"task_id"`
	CheckoutReceiptID CheckoutReceiptID `json:"checkout_receipt_id"`
	RunnerID          string            `json:"runner_id"`
	State             RunState          `json:"state"`
	StartedAt         time.Time         `json:"started_at"`
	FinishedAt        *time.Time        `json:"finished_at,omitempty"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

type ExecutionCommandPlanCreateRequest struct {
	ProjectID             ProjectID          `json:"project_id,omitempty"`
	RepoBindingID         RepoBindingID      `json:"repo_binding_id,omitempty"`
	CommandKind           string             `json:"command_kind,omitempty"`
	Action                string             `json:"action,omitempty"`
	ProjectProbeReceiptID ExecutionReceiptID `json:"project_probe_receipt_id,omitempty"`
	SelectedTargetID      string             `json:"selected_target_id,omitempty"`
	Shell                 json.RawMessage    `json:"shell,omitempty"`
	ShellAllowed          json.RawMessage    `json:"shell_allowed,omitempty"`
	Argv                  json.RawMessage    `json:"argv,omitempty"`
	Command               string             `json:"command,omitempty"`
	CommandString         string             `json:"command_string,omitempty"`
	UserCommand           string             `json:"user_command,omitempty"`
	RunAllTests           json.RawMessage    `json:"run_all_tests,omitempty"`
	StdoutCapture         json.RawMessage    `json:"stdout_capture,omitempty"`
	StderrCapture         json.RawMessage    `json:"stderr_capture,omitempty"`
	ArtifactsAllowed      json.RawMessage    `json:"artifacts_allowed,omitempty"`
	ArtifactRefs          json.RawMessage    `json:"artifact_refs,omitempty"`
	AllowedArtifactKinds  json.RawMessage    `json:"allowed_artifact_kinds,omitempty"`
	ChangedPathsAllowed   json.RawMessage    `json:"changed_paths_allowed,omitempty"`
	ChangedPathsSummary   json.RawMessage    `json:"changed_paths_summary,omitempty"`
	RawSourceUpload       json.RawMessage    `json:"raw_source_upload,omitempty"`
	RawSourceUploaded     json.RawMessage    `json:"raw_source_uploaded,omitempty"`
	RawSourceAllowed      json.RawMessage    `json:"raw_source_upload_allowed,omitempty"`
	NetworkAllowed        json.RawMessage    `json:"network_allowed,omitempty"`
	WriteAllowed          json.RawMessage    `json:"write_allowed,omitempty"`
	WorkspaceWriteAllowed json.RawMessage    `json:"workspace_write_allowed,omitempty"`
	ScratchWriteAllowed   json.RawMessage    `json:"scratch_write_allowed,omitempty"`
}

type ExecutionCommandPlan struct {
	ID                          ExecutionCommandPlanID           `json:"id"`
	OrganizationID              OrganizationID                   `json:"-"`
	ProjectID                   ProjectID                        `json:"project_id"`
	RepoBindingID               RepoBindingID                    `json:"repo_binding_id"`
	TaskID                      WorkItemID                       `json:"task_id"`
	CheckoutReceiptID           CheckoutReceiptID                `json:"checkout_receipt_id"`
	ExecutionJobID              ExecutionJobID                   `json:"execution_job_id"`
	RunID                       RunID                            `json:"run_id"`
	CommandKind                 string                           `json:"command_kind"`
	Action                      string                           `json:"action"`
	SourceProjectProbeReceiptID *ExecutionReceiptID              `json:"source_project_probe_receipt_id,omitempty"`
	SelectedTargetID            string                           `json:"selected_target_id,omitempty"`
	DeclaredTestTarget          *ProjectProbeTestTargetCandidate `json:"declared_test_target,omitempty"`
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
	State                       ExecutionCommandPlanState        `json:"state"`
	CreatedAt                   time.Time                        `json:"created_at"`
	UpdatedAt                   time.Time                        `json:"updated_at"`
	NextAction                  *ExecutionNextAction             `json:"next_action,omitempty"`
}

type ExecutionReceiptSubmitRequest struct {
	ExecutionJobID       ExecutionJobID         `json:"execution_job_id"`
	LeaseID              ExecutionLeaseID       `json:"lease_id"`
	LeaseToken           string                 `json:"lease_token"`
	RunnerID             string                 `json:"runner_id"`
	WorkspaceRef         string                 `json:"workspace_ref"`
	CommitSHA            string                 `json:"commit_sha"`
	BaselineID           string                 `json:"baseline_id,omitempty"`
	OverlayID            string                 `json:"overlay_id,omitempty"`
	ExecutionMode        string                 `json:"execution_mode"`
	CommandPlanID        ExecutionCommandPlanID `json:"command_plan_id,omitempty"`
	CommandKind          string                 `json:"command_kind,omitempty"`
	Action               string                 `json:"action,omitempty"`
	ProcessStatus        string                 `json:"process_status"`
	ExitCode             *int                   `json:"exit_code,omitempty"`
	ArtifactRefs         []string               `json:"artifact_refs,omitempty"`
	ChangedPathsSummary  []string               `json:"changed_paths_summary,omitempty"`
	RawSourceUploaded    bool                   `json:"raw_source_uploaded"`
	RunnerStartedAt      *time.Time             `json:"runner_started_at,omitempty"`
	RunnerFinishedAt     *time.Time             `json:"runner_finished_at,omitempty"`
	ProjectProbeMetadata *ProjectProbeMetadata  `json:"project_probe_metadata,omitempty"`
}

type ExecutionReceipt struct {
	ID                   ExecutionReceiptID      `json:"id"`
	RunID                RunID                   `json:"run_id"`
	ExecutionJobID       ExecutionJobID          `json:"execution_job_id"`
	ExecutionLeaseID     ExecutionLeaseID        `json:"execution_lease_id"`
	TaskID               WorkItemID              `json:"task_id"`
	CheckoutReceiptID    CheckoutReceiptID       `json:"checkout_receipt_id"`
	RepoBindingID        RepoBindingID           `json:"repo_binding_id"`
	RunnerID             string                  `json:"runner_id"`
	WorkspaceRef         string                  `json:"workspace_ref"`
	CommitSHA            string                  `json:"commit_sha"`
	BaselineID           string                  `json:"baseline_id,omitempty"`
	OverlayID            string                  `json:"overlay_id,omitempty"`
	ExecutionMode        string                  `json:"execution_mode"`
	CommandPlanID        *ExecutionCommandPlanID `json:"command_plan_id,omitempty"`
	CommandKind          string                  `json:"command_kind,omitempty"`
	Action               string                  `json:"action,omitempty"`
	ProcessStatus        string                  `json:"process_status"`
	ExitCode             *int                    `json:"exit_code,omitempty"`
	ArtifactRefs         []string                `json:"artifact_refs"`
	ChangedPathsSummary  []string                `json:"changed_paths_summary"`
	RawSourceUploaded    bool                    `json:"raw_source_uploaded"`
	RunnerStartedAt      *time.Time              `json:"runner_started_at,omitempty"`
	RunnerFinishedAt     *time.Time              `json:"runner_finished_at,omitempty"`
	ProjectProbeMetadata *ProjectProbeMetadata   `json:"project_probe_metadata,omitempty"`
	StartedAt            time.Time               `json:"started_at"`
	FinishedAt           time.Time               `json:"finished_at"`
	CreatedAt            time.Time               `json:"created_at"`
	UpdatedAt            time.Time               `json:"updated_at"`
	NextAction           ExecutionNextAction     `json:"next_action"`
}

type ProjectProbeMetadata struct {
	DetectedManifests            []ProjectProbeManifest                `json:"detected_manifests"`
	PackageManagerCandidates     []ProjectProbePackageManagerCandidate `json:"package_manager_candidates"`
	DeclaredTestTargetCandidates []ProjectProbeTestTargetCandidate     `json:"declared_test_target_candidates"`
	UnsupportedOrUnknowns        []string                              `json:"unsupported_or_unknowns"`
	PartialityReasons            []string                              `json:"partiality_reasons"`
}

type ProjectProbeManifest struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
}

type ProjectProbePackageManagerCandidate struct {
	Name       string `json:"name"`
	SourcePath string `json:"source_path"`
}

type ProjectProbeTestTargetCandidate struct {
	Name       string `json:"name"`
	SourcePath string `json:"source_path"`
	SourceKind string `json:"source_kind"`
}

type ExecutionNextAction struct {
	Kind         string `json:"kind"`
	Blocking     bool   `json:"blocking"`
	Available    bool   `json:"available"`
	PlannedSlice string `json:"planned_slice,omitempty"`
}
