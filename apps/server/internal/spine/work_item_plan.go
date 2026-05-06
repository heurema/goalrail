package spine

import "time"

type WorkItemPlanID string

type WorkItemPlanProposalID string

type WorkItemPlanLeaseID string

type WorkItemPlanState string

const (
	WorkItemPlanStateQueued            WorkItemPlanState = "queued"
	WorkItemPlanStateLeased            WorkItemPlanState = "leased"
	WorkItemPlanStateProposalSubmitted WorkItemPlanState = "proposal_submitted"
	WorkItemPlanStateAccepted          WorkItemPlanState = "accepted"
)

type WorkItemPlanLeaseState string

const (
	WorkItemPlanLeaseStateActive    WorkItemPlanLeaseState = "active"
	WorkItemPlanLeaseStateCompleted WorkItemPlanLeaseState = "completed"
	WorkItemPlanLeaseStateExpired   WorkItemPlanLeaseState = "expired"
)

type WorkItemProposalState string

const (
	WorkItemProposalStateSubmitted WorkItemProposalState = "submitted"
	WorkItemProposalStateAccepted  WorkItemProposalState = "accepted"
)

type WorkItemPlan struct {
	ID                 WorkItemPlanID       `json:"id"`
	OrganizationID     OrganizationID       `json:"-"`
	ProjectID          ProjectID            `json:"-"`
	ContractID         ContractID           `json:"contract_id"`
	ApprovedContractID ApprovedContractID   `json:"approved_contract_id"`
	RepoBindingID      RepoBindingID        `json:"repo_binding_id"`
	State              WorkItemPlanState    `json:"state"`
	RequestedBy        ActorRef             `json:"requested_by"`
	CurrentLeaseID     *WorkItemPlanLeaseID `json:"current_lease_id,omitempty"`
	LeasedBy           *ActorRef            `json:"leased_by,omitempty"`
	LeaseExpiresAt     *time.Time           `json:"lease_expires_at,omitempty"`
	CreatedAt          time.Time            `json:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at"`
}

type WorkItemPlanLease struct {
	ID                 WorkItemPlanLeaseID    `json:"id"`
	PlanID             WorkItemPlanID         `json:"plan_id"`
	ContractID         ContractID             `json:"contract_id"`
	ApprovedContractID ApprovedContractID     `json:"approved_contract_id"`
	RepoBindingID      RepoBindingID          `json:"repo_binding_id"`
	LeasedBy           ActorRef               `json:"leased_by"`
	State              WorkItemPlanLeaseState `json:"state"`
	LeaseTokenHash     string                 `json:"-"`
	ExpiresAt          time.Time              `json:"expires_at"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

type WorkItemPlanLeaseCreated struct {
	ID                 WorkItemPlanLeaseID    `json:"id"`
	PlanID             WorkItemPlanID         `json:"plan_id"`
	ContractID         ContractID             `json:"contract_id"`
	ApprovedContractID ApprovedContractID     `json:"approved_contract_id"`
	RepoBindingID      RepoBindingID          `json:"repo_binding_id"`
	LeasedBy           ActorRef               `json:"leased_by"`
	State              WorkItemPlanLeaseState `json:"state"`
	LeaseToken         string                 `json:"lease_token"`
	ExpiresAt          time.Time              `json:"expires_at"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

type WorkItemPlanProposal struct {
	ID                 WorkItemPlanProposalID `json:"id"`
	PlanID             WorkItemPlanID         `json:"plan_id"`
	OrganizationID     OrganizationID         `json:"-"`
	ProjectID          ProjectID              `json:"-"`
	ContractID         ContractID             `json:"contract_id"`
	ApprovedContractID ApprovedContractID     `json:"approved_contract_id"`
	RepoBindingID      RepoBindingID          `json:"repo_binding_id"`
	State              WorkItemProposalState  `json:"state"`
	SubmittedBy        ActorRef               `json:"submitted_by"`
	Planner            map[string]any         `json:"planner"`
	SourceSnapshotRefs []SourceRef            `json:"source_snapshot_refs"`
	Rationale          string                 `json:"rationale"`
	ProposedTasks      []ProposedWorkItem     `json:"proposed_tasks"`
	AcceptedBy         *ActorRef              `json:"accepted_by,omitempty"`
	AcceptedAt         *time.Time             `json:"accepted_at,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

type ProposedWorkItem struct {
	Title                string      `json:"title"`
	Summary              string      `json:"summary"`
	Scope                []string    `json:"scope"`
	AcceptanceRefs       []string    `json:"acceptance_refs"`
	ProofExpectationRefs []string    `json:"proof_expectation_refs"`
	OwnerHint            string      `json:"owner_hint,omitempty"`
	OrderIndex           *int        `json:"order_index,omitempty"`
	SourceRefs           []SourceRef `json:"source_refs"`
}

type WorkItemPlanCreateRequest struct {
	RequestedBy ActorRef `json:"requested_by"`
}

type WorkItemPlanLeaseCreateRequest struct {
	LeasedBy   ActorRef `json:"leased_by"`
	TTLSeconds int      `json:"ttl_seconds,omitempty"`
}

type WorkItemPlanLeaseRenewRequest struct {
	LeaseToken string `json:"lease_token"`
	TTLSeconds int    `json:"ttl_seconds,omitempty"`
}

type WorkItemPlanProposalSubmitRequest struct {
	LeaseID            WorkItemPlanLeaseID `json:"lease_id"`
	LeaseToken         string              `json:"lease_token"`
	SubmittedBy        ActorRef            `json:"submitted_by"`
	Planner            map[string]any      `json:"planner"`
	SourceSnapshotRefs []SourceRef         `json:"source_snapshot_refs"`
	Rationale          string              `json:"rationale"`
	ProposedTasks      []ProposedWorkItem  `json:"proposed_tasks"`
}

type WorkItemPlanAcceptanceRequest struct {
	AcceptedBy ActorRef `json:"accepted_by"`
}

type WorkItemPlanAcceptanceResult struct {
	ProposalID     WorkItemPlanProposalID `json:"proposal_id"`
	PlanID         WorkItemPlanID         `json:"plan_id"`
	ContractID     ContractID             `json:"contract_id"`
	State          WorkItemProposalState  `json:"state"`
	AcceptedBy     ActorRef               `json:"accepted_by"`
	AcceptedAt     time.Time              `json:"accepted_at"`
	CreatedTaskIDs []WorkItemID           `json:"created_task_ids"`
}
