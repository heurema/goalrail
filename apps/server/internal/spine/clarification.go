package spine

import "time"

type ClarificationRequestID string

type ClarificationQuestionID string

type ClarificationRequestState string

const ClarificationRequestStateOpen ClarificationRequestState = "open"

type ClarificationAnswerType string

const (
	ClarificationAnswerTypeText    ClarificationAnswerType = "text"
	ClarificationAnswerTypeChoice  ClarificationAnswerType = "choice"
	ClarificationAnswerTypeBoolean ClarificationAnswerType = "boolean"
)

type ClarificationMapsTo string

const (
	ClarificationMapsToGoalSummary        ClarificationMapsTo = "goal.summary"
	ClarificationMapsToGoalIntentOwner    ClarificationMapsTo = "goal.intent_owner"
	ClarificationMapsToGoalScopeHint      ClarificationMapsTo = "goal.scope_hint"
	ClarificationMapsToGoalAcceptanceHint ClarificationMapsTo = "goal.acceptance_hint"
)

type ClarificationTargetRole string

const (
	ClarificationTargetRoleRequestAuthor ClarificationTargetRole = "request_author"
	ClarificationTargetRoleIntentOwner   ClarificationTargetRole = "intent_owner"
	ClarificationTargetRoleDeliveryOwner ClarificationTargetRole = "delivery_owner"
	ClarificationTargetRoleRepoOwner     ClarificationTargetRole = "repo_owner"
	ClarificationTargetRolePolicyOwner   ClarificationTargetRole = "policy_owner"
)

type ClarificationRequest struct {
	ID          ClarificationRequestID    `json:"id"`
	GoalID      GoalID                    `json:"goal_id"`
	ReasonCodes []GoalReadinessReasonCode `json:"reason_codes"`
	Questions   []ClarificationQuestion   `json:"questions"`
	Target      ClarificationTarget       `json:"target"`
	State       ClarificationRequestState `json:"state"`
	CreatedAt   time.Time                 `json:"created_at"`
}

type ClarificationQuestion struct {
	ID         ClarificationQuestionID `json:"id"`
	Text       string                  `json:"text"`
	WhyNeeded  string                  `json:"why_needed"`
	AnswerType ClarificationAnswerType `json:"answer_type"`
	MapsTo     ClarificationMapsTo     `json:"maps_to"`
}

type ClarificationTarget struct {
	Role             ClarificationTargetRole `json:"role"`
	ActorRef         *ActorRef               `json:"actor_ref,omitempty"`
	PreferredSurface string                  `json:"preferred_surface,omitempty"`
}
