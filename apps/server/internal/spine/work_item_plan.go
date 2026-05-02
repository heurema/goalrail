package spine

import "time"

type WorkItemPlanID string

type WorkItemPlanProposalID string

type WorkItemPlanState string

const (
	WorkItemPlanStateQueued            WorkItemPlanState = "queued"
	WorkItemPlanStateProposalSubmitted WorkItemPlanState = "proposal_submitted"
	WorkItemPlanStateAccepted          WorkItemPlanState = "accepted"
)

type WorkItemProposalState string

const (
	WorkItemProposalStateSubmitted WorkItemProposalState = "submitted"
	WorkItemProposalStateAccepted  WorkItemProposalState = "accepted"
)

type WorkItemPlan struct {
	ID                 WorkItemPlanID     `json:"id"`
	OrganizationID     OrganizationID     `json:"-"`
	ProjectID          ProjectID          `json:"-"`
	ContractID         ContractID         `json:"contract_id"`
	ApprovedContractID ApprovedContractID `json:"approved_contract_id"`
	RepoBindingID      RepoBindingID      `json:"repo_binding_id"`
	State              WorkItemPlanState  `json:"state"`
	RequestedBy        ActorRef           `json:"requested_by"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
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

type WorkItemPlanProposalSubmitRequest struct {
	SubmittedBy        ActorRef           `json:"submitted_by"`
	Planner            map[string]any     `json:"planner"`
	SourceSnapshotRefs []SourceRef        `json:"source_snapshot_refs"`
	Rationale          string             `json:"rationale"`
	ProposedTasks      []ProposedWorkItem `json:"proposed_tasks"`
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
