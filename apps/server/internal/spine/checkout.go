package spine

import "time"

type CheckoutJobID string

type CheckoutReceiptID string

type CheckoutJobState string

const (
	CheckoutJobStateQueued           CheckoutJobState = "queued"
	CheckoutJobStateLeased           CheckoutJobState = "leased"
	CheckoutJobStateReceiptSubmitted CheckoutJobState = "receipt_submitted"
)

type CheckoutInstruction struct {
	JobID              CheckoutJobID         `json:"job_id"`
	TaskID             WorkItemID            `json:"task_id"`
	RepoBindingID      RepoBindingID         `json:"repo_binding_id"`
	AccessMode         RepoBindingAccessMode `json:"access_mode"`
	Provider           string                `json:"provider"`
	RepositoryFullName string                `json:"repository_full_name"`
	RepositoryURL      string                `json:"repository_url"`
	WorkflowBaseBranch string                `json:"workflow_base_branch"`
	PathScope          string                `json:"path_scope"`
	SourceRef          SourceRef             `json:"source_ref"`
	RawSourceUploaded  bool                  `json:"raw_source_uploaded"`
}

type CheckoutJob struct {
	ID                 CheckoutJobID          `json:"id"`
	OrganizationID     OrganizationID         `json:"-"`
	ProjectID          ProjectID              `json:"-"`
	TaskID             WorkItemID             `json:"task_id"`
	ContractID         ContractID             `json:"contract_id"`
	ApprovedContractID ApprovedContractID     `json:"approved_contract_id"`
	PlanID             WorkItemPlanID         `json:"plan_id"`
	ProposalID         WorkItemPlanProposalID `json:"proposal_id"`
	RepoBindingID      RepoBindingID          `json:"repo_binding_id"`
	State              CheckoutJobState       `json:"state"`
	RequestedBy        ActorRef               `json:"requested_by"`
	Instruction        CheckoutInstruction    `json:"instruction"`
	CurrentRunnerID    string                 `json:"current_runner_id,omitempty"`
	LeaseTokenHash     string                 `json:"-"`
	LeaseExpiresAt     *time.Time             `json:"lease_expires_at,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

type CheckoutJobCreateRequest struct {
	ProjectID     ProjectID     `json:"project_id,omitempty"`
	RepoBindingID RepoBindingID `json:"repo_binding_id,omitempty"`
	RequestedBy   ActorRef      `json:"requested_by,omitempty"`
}

type CheckoutJobLeaseCreateRequest struct {
	ProjectID     ProjectID     `json:"project_id,omitempty"`
	RepoBindingID RepoBindingID `json:"repo_binding_id,omitempty"`
	RunnerID      string        `json:"runner_id"`
	TTLSeconds    int           `json:"ttl_seconds,omitempty"`
}

type CheckoutJobLeaseCreated struct {
	JobID          CheckoutJobID       `json:"job_id"`
	TaskID         WorkItemID          `json:"task_id"`
	State          CheckoutJobState    `json:"state"`
	RunnerID       string              `json:"runner_id"`
	LeaseToken     string              `json:"lease_token"`
	LeaseExpiresAt time.Time           `json:"lease_expires_at"`
	Instruction    CheckoutInstruction `json:"instruction"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
}

type CheckoutReceipt struct {
	ID                CheckoutReceiptID `json:"id"`
	JobID             CheckoutJobID     `json:"job_id"`
	TaskID            WorkItemID        `json:"task_id"`
	RepoBindingID     RepoBindingID     `json:"repo_binding_id"`
	RunnerID          string            `json:"runner_id"`
	WorkspaceRef      string            `json:"workspace_ref"`
	CommitSHA         string            `json:"commit_sha"`
	BaselineID        string            `json:"baseline_id,omitempty"`
	OverlayID         string            `json:"overlay_id,omitempty"`
	Dirty             bool              `json:"dirty"`
	Partial           bool              `json:"partial"`
	PartialReasons    []string          `json:"partial_reasons,omitempty"`
	RawSourceUploaded bool              `json:"raw_source_uploaded"`
	CreatedAt         time.Time         `json:"created_at"`
}

type CheckoutReceiptSubmitRequest struct {
	LeaseToken        string   `json:"lease_token"`
	RunnerID          string   `json:"runner_id"`
	WorkspaceRef      string   `json:"workspace_ref"`
	CommitSHA         string   `json:"commit_sha"`
	BaselineID        string   `json:"baseline_id,omitempty"`
	OverlayID         string   `json:"overlay_id,omitempty"`
	Dirty             bool     `json:"dirty"`
	Partial           bool     `json:"partial"`
	PartialReasons    []string `json:"partial_reasons,omitempty"`
	RawSourceUploaded bool     `json:"raw_source_uploaded"`
}
