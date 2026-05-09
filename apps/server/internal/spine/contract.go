package spine

import "time"

type ContractID string

type ContractState string

const (
	ContractStateSeeded           ContractState = "seeded"
	ContractStateDraft            ContractState = "draft"
	ContractStateReadyForApproval ContractState = "ready_for_approval"
	ContractStateApproved         ContractState = "approved"
)

type Contract struct {
	ID                 ContractID          `json:"id"`
	OrganizationID     OrganizationID      `json:"-"`
	ProjectID          ProjectID           `json:"-"`
	RepoBindingID      RepoBindingID       `json:"repo_binding_id"`
	GoalID             GoalID              `json:"goal_id"`
	State              ContractState       `json:"state"`
	CurrentSeedID      *ContractSeedID     `json:"current_seed_id,omitempty"`
	CurrentDraftID     *ContractDraftID    `json:"current_draft_id,omitempty"`
	ApprovedSnapshotID *ApprovedContractID `json:"approved_snapshot_id,omitempty"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
}

type ContractCreateRequest struct {
	GoalID        GoalID        `json:"goal_id"`
	ProjectID     ProjectID     `json:"project_id,omitempty"`
	RepoBindingID RepoBindingID `json:"repo_binding_id,omitempty"`
}

type ContractListFilter struct {
	OrganizationID OrganizationID
	ProjectID      ProjectID
	RepoBindingID  RepoBindingID
	GoalID         GoalID
	State          ContractState
	Limit          int
}

type ContractList struct {
	Contracts []Contract `json:"contracts"`
	Limit     int        `json:"limit"`
}
