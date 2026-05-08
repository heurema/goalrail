package spine

import "time"

type ExecutionJobID string

type ExecutionJobState string

const ExecutionJobStateQueued ExecutionJobState = "queued"

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
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

type ExecutionJobCreateRequest struct {
	CheckoutReceiptID CheckoutReceiptID `json:"checkout_receipt_id"`
	ProjectID         ProjectID         `json:"project_id,omitempty"`
	RepoBindingID     RepoBindingID     `json:"repo_binding_id,omitempty"`
	RequestedBy       ActorRef          `json:"requested_by,omitempty"`
}
