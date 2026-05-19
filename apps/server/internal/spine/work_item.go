package spine

import "time"

type WorkItemID string

type WorkItemStatus string

const WorkItemStatusPlanned WorkItemStatus = "planned"

type WorkItem struct {
	ID                   WorkItemID             `json:"id"`
	OrganizationID       OrganizationID         `json:"-"`
	ProjectID            ProjectID              `json:"-"`
	ContractID           ContractID             `json:"contract_id"`
	ApprovedContractID   ApprovedContractID     `json:"approved_contract_id"`
	PlanID               WorkItemPlanID         `json:"plan_id"`
	ProposalID           WorkItemPlanProposalID `json:"proposal_id"`
	RepoBindingID        RepoBindingID          `json:"repo_binding_id"`
	Title                string                 `json:"title"`
	Summary              string                 `json:"summary"`
	Scope                []string               `json:"scope"`
	AcceptanceRefs       []string               `json:"acceptance_refs"`
	ProofExpectationRefs []string               `json:"proof_expectation_refs"`
	Status               WorkItemStatus         `json:"status"`
	OwnerHint            string                 `json:"owner_hint,omitempty"`
	OrderIndex           *int                   `json:"order_index,omitempty"`
	SourceRefs           []SourceRef            `json:"source_refs"`
	CreatedAt            time.Time              `json:"created_at"`
}

type WorkItemDetailRequest struct {
	ProjectID     ProjectID     `json:"project_id,omitempty"`
	RepoBindingID RepoBindingID `json:"repo_binding_id,omitempty"`
}

type WorkItemNextAction struct {
	Kind      string `json:"kind"`
	Blocking  bool   `json:"blocking"`
	Command   string `json:"command,omitempty"`
	Available bool   `json:"available"`
}

type WorkItemDetail struct {
	ID                   WorkItemID             `json:"id"`
	WorkItemID           WorkItemID             `json:"work_item_id"`
	TaskID               WorkItemID             `json:"task_id"`
	ProjectID            ProjectID              `json:"project_id,omitempty"`
	GoalID               GoalID                 `json:"goal_id,omitempty"`
	ContractID           ContractID             `json:"contract_id"`
	ApprovedContractID   ApprovedContractID     `json:"approved_contract_id"`
	PlanID               WorkItemPlanID         `json:"plan_id"`
	ProposalID           WorkItemPlanProposalID `json:"proposal_id"`
	RepoBindingID        RepoBindingID          `json:"repo_binding_id"`
	Status               WorkItemStatus         `json:"status"`
	Title                string                 `json:"title"`
	Summary              string                 `json:"summary"`
	Scope                []string               `json:"scope"`
	AcceptanceRefs       []string               `json:"acceptance_refs"`
	ProofExpectationRefs []string               `json:"proof_expectation_refs"`
	SourceRefs           []SourceRef            `json:"source_refs"`
	OwnerHint            string                 `json:"owner_hint,omitempty"`
	OrderIndex           *int                   `json:"order_index,omitempty"`
	NextAction           WorkItemNextAction     `json:"next_action"`
}
