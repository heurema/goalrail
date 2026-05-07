package spine

import (
	"encoding/json"
	"time"
)

type ContractDraftID string

type ContractDraftState string

const ContractDraftStateDraft ContractDraftState = "draft"
const ContractDraftStateReadyForApproval ContractDraftState = "ready_for_approval"

type ContractDraft struct {
	ID                         ContractDraftID    `json:"id"`
	OrganizationID             OrganizationID     `json:"-"`
	ProjectID                  ProjectID          `json:"-"`
	ContractID                 ContractID         `json:"contract_id"`
	ContractSeedID             ContractSeedID     `json:"contract_seed_id"`
	GoalID                     GoalID             `json:"goal_id"`
	RepoBindingID              RepoBindingID      `json:"repo_binding_id"`
	Title                      string             `json:"title"`
	IntentSummary              string             `json:"intent_summary"`
	ProposedScope              []string           `json:"proposed_scope"`
	ProposedNonGoals           []string           `json:"proposed_non_goals"`
	ProposedConstraints        []string           `json:"proposed_constraints"`
	ProposedAcceptanceCriteria []string           `json:"proposed_acceptance_criteria"`
	ProposedExpectedChecks     []string           `json:"proposed_expected_checks"`
	ProposedProofExpectations  []string           `json:"proposed_proof_expectations"`
	RiskHints                  []string           `json:"risk_hints"`
	SourceRefs                 []SourceRef        `json:"source_refs"`
	State                      ContractDraftState `json:"state"`
	CreatedAt                  time.Time          `json:"created_at"`
}

type ContractDraftUpdateRequest struct {
	ProjectID     ProjectID                  `json:"project_id,omitempty"`
	RepoBindingID RepoBindingID              `json:"repo_binding_id,omitempty"`
	UpdatedBy     ActorRef                   `json:"updated_by"`
	Changes       map[string]json.RawMessage `json:"changes"`
	ContextRefs   []ContractUpdateContextRef `json:"context_refs,omitempty"`
	Unknowns      []string                   `json:"unknowns,omitempty"`
}

type ContractUpdateContextRef struct {
	Kind       string `json:"kind"`
	ID         string `json:"id"`
	BaselineID string `json:"baseline_id,omitempty"`
	OverlayID  string `json:"overlay_id,omitempty"`
}

type ContractDraftReadyForApprovalRequest struct {
	MarkedBy ActorRef `json:"marked_by"`
}
