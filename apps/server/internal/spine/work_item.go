package spine

import "time"

type WorkItemID string

type WorkItemStatus string

const WorkItemStatusPlanned WorkItemStatus = "planned"

type WorkItem struct {
	ID                   WorkItemID         `json:"id"`
	OrganizationID       OrganizationID     `json:"-"`
	ProjectID            ProjectID          `json:"-"`
	ApprovedContractID   ApprovedContractID `json:"approved_contract_id"`
	RepoBindingID        RepoBindingID      `json:"repo_binding_id"`
	Title                string             `json:"title"`
	Summary              string             `json:"summary"`
	Scope                []string           `json:"scope"`
	AcceptanceRefs       []string           `json:"acceptance_refs"`
	ProofExpectationRefs []string           `json:"proof_expectation_refs"`
	Status               WorkItemStatus     `json:"status"`
	OwnerHint            string             `json:"owner_hint,omitempty"`
	OrderIndex           *int               `json:"order_index,omitempty"`
	SourceRefs           []SourceRef        `json:"source_refs"`
	CreatedAt            time.Time          `json:"created_at"`
}
