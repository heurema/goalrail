package spine

import "time"

type GoalID string

type GoalState string

const (
	GoalStateCreated              GoalState = "created"
	GoalStateNeedsClarification   GoalState = "needs_clarification"
	GoalStateReadyForContractSeed GoalState = "ready_for_contract_seed"
	GoalStateRejected             GoalState = "rejected"
)

type GoalReadinessReasonCode string

const (
	GoalReadinessReasonMissingGoalSummary    GoalReadinessReasonCode = "missing_goal_summary"
	GoalReadinessReasonMissingIntentOwner    GoalReadinessReasonCode = "missing_intent_owner"
	GoalReadinessReasonMissingScopeHint      GoalReadinessReasonCode = "missing_scope_hint"
	GoalReadinessReasonMissingAcceptanceHint GoalReadinessReasonCode = "missing_acceptance_hint"
	GoalReadinessReasonPolicyRejected        GoalReadinessReasonCode = "policy_rejected"
)

type SourceRef struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type Goal struct {
	ID                       GoalID                    `json:"id"`
	IntakeID                 IntakeID                  `json:"intake_id"`
	OrganizationID           OrganizationID            `json:"organization_id"`
	ProjectID                ProjectID                 `json:"project_id"`
	RepoBindingID            RepoBindingID             `json:"repo_binding_id"`
	Title                    string                    `json:"title"`
	Summary                  string                    `json:"summary"`
	ScopeHint                string                    `json:"scope_hint,omitempty"`
	AcceptanceHint           string                    `json:"acceptance_hint,omitempty"`
	SourceRefs               []SourceRef               `json:"source_refs"`
	RequestAuthor            ActorRef                  `json:"request_author"`
	IntentOwner              ActorRef                  `json:"intent_owner"`
	State                    GoalState                 `json:"state"`
	LastReadinessReasonCodes []GoalReadinessReasonCode `json:"last_readiness_reason_codes,omitempty"`
	CreatedAt                time.Time                 `json:"created_at"`
}

type GoalReadinessResult struct {
	GoalID         GoalID                    `json:"goal_id"`
	OrganizationID OrganizationID            `json:"organization_id,omitempty"`
	ProjectID      ProjectID                 `json:"project_id,omitempty"`
	RepoBindingID  RepoBindingID             `json:"repo_binding_id,omitempty"`
	State          GoalState                 `json:"state"`
	Ready          bool                      `json:"ready"`
	ReasonCodes    []GoalReadinessReasonCode `json:"reason_codes"`
	Message        string                    `json:"message"`
	CheckedAt      time.Time                 `json:"checked_at"`
}

type GoalHintUpdate struct {
	Summary        *string
	ScopeHint      *string
	AcceptanceHint *string
	IntentOwner    *ActorRef
}
