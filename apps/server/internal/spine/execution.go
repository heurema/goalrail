package spine

import "time"

type ExecutionJobID string

type ExecutionLeaseID string

type RunID string

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
	ExecutionReceiptStatusNotExecuted      = "not_executed"
	ExecutionReceiptStatusMetadataOnly     = "metadata_only"
	ExecutionReceiptNextActionGateReview   = "gate_review"
	ExecutionReceiptNextActionPlannedSlice = "I"
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

type ExecutionReceiptSubmitRequest struct {
	ExecutionJobID      ExecutionJobID `json:"execution_job_id"`
	LeaseToken          string         `json:"lease_token"`
	RunnerID            string         `json:"runner_id"`
	WorkspaceRef        string         `json:"workspace_ref"`
	CommitSHA           string         `json:"commit_sha"`
	BaselineID          string         `json:"baseline_id,omitempty"`
	OverlayID           string         `json:"overlay_id,omitempty"`
	ExecutionMode       string         `json:"execution_mode"`
	ProcessStatus       string         `json:"process_status"`
	ExitCode            *int           `json:"exit_code,omitempty"`
	ArtifactRefs        []string       `json:"artifact_refs,omitempty"`
	ChangedPathsSummary []string       `json:"changed_paths_summary,omitempty"`
	RawSourceUploaded   bool           `json:"raw_source_uploaded"`
}

type ExecutionReceipt struct {
	ID                  ExecutionReceiptID  `json:"id"`
	RunID               RunID               `json:"run_id"`
	ExecutionJobID      ExecutionJobID      `json:"execution_job_id"`
	ExecutionLeaseID    ExecutionLeaseID    `json:"execution_lease_id"`
	TaskID              WorkItemID          `json:"task_id"`
	CheckoutReceiptID   CheckoutReceiptID   `json:"checkout_receipt_id"`
	RepoBindingID       RepoBindingID       `json:"repo_binding_id"`
	RunnerID            string              `json:"runner_id"`
	WorkspaceRef        string              `json:"workspace_ref"`
	CommitSHA           string              `json:"commit_sha"`
	BaselineID          string              `json:"baseline_id,omitempty"`
	OverlayID           string              `json:"overlay_id,omitempty"`
	ExecutionMode       string              `json:"execution_mode"`
	ProcessStatus       string              `json:"process_status"`
	ExitCode            *int                `json:"exit_code,omitempty"`
	ArtifactRefs        []string            `json:"artifact_refs"`
	ChangedPathsSummary []string            `json:"changed_paths_summary"`
	RawSourceUploaded   bool                `json:"raw_source_uploaded"`
	StartedAt           time.Time           `json:"started_at"`
	FinishedAt          time.Time           `json:"finished_at"`
	CreatedAt           time.Time           `json:"created_at"`
	UpdatedAt           time.Time           `json:"updated_at"`
	NextAction          ExecutionNextAction `json:"next_action"`
}

type ExecutionNextAction struct {
	Kind         string `json:"kind"`
	Blocking     bool   `json:"blocking"`
	Available    bool   `json:"available"`
	PlannedSlice string `json:"planned_slice,omitempty"`
}
