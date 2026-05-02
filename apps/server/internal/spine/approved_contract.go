package spine

import "time"

type ApprovedContractID string

type ApprovedContractState string

const ApprovedContractStateApproved ApprovedContractState = "approved"

type ApprovedContract struct {
	ID                 ApprovedContractID    `json:"id"`
	OrganizationID     OrganizationID        `json:"-"`
	ProjectID          ProjectID             `json:"-"`
	ContractID         ContractID            `json:"contract_id"`
	ContractDraftID    ContractDraftID       `json:"contract_draft_id"`
	ContractSeedID     ContractSeedID        `json:"contract_seed_id"`
	GoalID             GoalID                `json:"goal_id"`
	RepoBindingID      RepoBindingID         `json:"repo_binding_id"`
	Title              string                `json:"title"`
	IntentSummary      string                `json:"intent_summary"`
	Scope              []string              `json:"scope"`
	NonGoals           []string              `json:"non_goals"`
	Constraints        []string              `json:"constraints"`
	AcceptanceCriteria []string              `json:"acceptance_criteria"`
	ExpectedChecks     []string              `json:"expected_checks"`
	ProofExpectations  []string              `json:"proof_expectations"`
	RiskHints          []string              `json:"risk_hints"`
	ApprovedBy         ActorRef              `json:"approved_by"`
	ApprovedAt         time.Time             `json:"approved_at"`
	SourceRefs         []SourceRef           `json:"source_refs"`
	State              ApprovedContractState `json:"state"`
}

type ApproveContractDraftRequest struct {
	ApprovedBy ActorRef `json:"approved_by"`
}
