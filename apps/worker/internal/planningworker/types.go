package planningworker

import "time"

const plannerVersion = "0.2.0"

type actorRef struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type sourceRef struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type leaseCreateRequest struct {
	LeasedBy   actorRef `json:"leased_by"`
	TTLSeconds int      `json:"ttl_seconds,omitempty"`
}

type planLease struct {
	ID                 string    `json:"id"`
	PlanID             string    `json:"plan_id"`
	ContractID         string    `json:"contract_id"`
	ApprovedContractID string    `json:"approved_contract_id"`
	RepoBindingID      string    `json:"repo_binding_id"`
	State              string    `json:"state"`
	LeaseToken         string    `json:"lease_token"`
	ExpiresAt          time.Time `json:"expires_at"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type workItemPlan struct {
	ID                 string                    `json:"id"`
	ContractID         string                    `json:"contract_id"`
	ApprovedContractID string                    `json:"approved_contract_id"`
	RepoBindingID      string                    `json:"repo_binding_id"`
	State              string                    `json:"state"`
	CurrentLeaseID     string                    `json:"current_lease_id,omitempty"`
	LeasedBy           *actorRef                 `json:"leased_by,omitempty"`
	LeaseExpiresAt     *time.Time                `json:"lease_expires_at,omitempty"`
	ApprovedContract   *approvedContractSnapshot `json:"approved_contract,omitempty"`
	CreatedAt          time.Time                 `json:"created_at"`
	UpdatedAt          time.Time                 `json:"updated_at"`
}

type approvedContractSnapshot struct {
	ID                 string   `json:"id"`
	ContractID         string   `json:"contract_id"`
	ContractDraftID    string   `json:"contract_draft_id"`
	ContractSeedID     string   `json:"contract_seed_id"`
	GoalID             string   `json:"goal_id"`
	RepoBindingID      string   `json:"repo_binding_id"`
	Title              string   `json:"title"`
	IntentSummary      string   `json:"intent_summary"`
	Scope              []string `json:"scope"`
	NonGoals           []string `json:"non_goals"`
	Constraints        []string `json:"constraints"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	ExpectedChecks     []string `json:"expected_checks"`
	ProofExpectations  []string `json:"proof_expectations"`
	RiskHints          []string `json:"risk_hints"`
	State              string   `json:"state"`
}

type proposalSubmitRequest struct {
	LeaseID            string             `json:"lease_id"`
	LeaseToken         string             `json:"lease_token"`
	SubmittedBy        actorRef           `json:"submitted_by"`
	Planner            map[string]any     `json:"planner"`
	SourceSnapshotRefs []sourceRef        `json:"source_snapshot_refs"`
	Rationale          string             `json:"rationale"`
	ProposedTasks      []proposedWorkItem `json:"proposed_tasks"`
}

type proposedWorkItem struct {
	Title                string      `json:"title"`
	Summary              string      `json:"summary"`
	Scope                []string    `json:"scope"`
	AcceptanceRefs       []string    `json:"acceptance_refs"`
	ProofExpectationRefs []string    `json:"proof_expectation_refs"`
	OwnerHint            string      `json:"owner_hint,omitempty"`
	OrderIndex           *int        `json:"order_index,omitempty"`
	SourceRefs           []sourceRef `json:"source_refs"`
}

type planProposal struct {
	ID                 string             `json:"id"`
	PlanID             string             `json:"plan_id"`
	ContractID         string             `json:"contract_id"`
	ApprovedContractID string             `json:"approved_contract_id"`
	RepoBindingID      string             `json:"repo_binding_id"`
	State              string             `json:"state"`
	SubmittedBy        actorRef           `json:"submitted_by"`
	Planner            map[string]any     `json:"planner"`
	SourceSnapshotRefs []sourceRef        `json:"source_snapshot_refs"`
	Rationale          string             `json:"rationale"`
	ProposedTasks      []proposedWorkItem `json:"proposed_tasks"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}
