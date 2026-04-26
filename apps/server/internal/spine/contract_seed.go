package spine

import "time"

type ContractSeedID string

type ContractSeedState string

const ContractSeedStateCreated ContractSeedState = "created"

type ContractSeed struct {
	ID             ContractSeedID    `json:"id"`
	GoalID         GoalID            `json:"goal_id"`
	RepoBindingID  RepoBindingID     `json:"repo_binding_id"`
	Title          string            `json:"title"`
	IntentSummary  string            `json:"intent_summary"`
	IntentOwner    ActorRef          `json:"intent_owner"`
	ScopeHint      string            `json:"scope_hint"`
	AcceptanceHint string            `json:"acceptance_hint"`
	SourceRefs     []SourceRef       `json:"source_refs"`
	State          ContractSeedState `json:"state"`
	CreatedAt      time.Time         `json:"created_at"`
}
