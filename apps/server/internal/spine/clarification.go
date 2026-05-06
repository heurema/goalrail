package spine

import "time"

type ClarificationRequestID string

type ClarificationQuestionID string

type ClarificationRequestState string

const (
	ClarificationRequestStateOpen       ClarificationRequestState = "open"
	ClarificationRequestStateAnswered   ClarificationRequestState = "answered"
	ClarificationRequestStateCancelled  ClarificationRequestState = "cancelled"
	ClarificationRequestStateSuperseded ClarificationRequestState = "superseded"
)

type ClarificationAnswerID string

type ClarificationAnswerState string

const ClarificationAnswerStateRecorded ClarificationAnswerState = "recorded"

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

type ClarificationAnswer struct {
	ID          ClarificationAnswerID     `json:"id"`
	RequestID   ClarificationRequestID    `json:"request_id"`
	GoalID      GoalID                    `json:"goal_id"`
	Answers     []ClarificationAnswerItem `json:"answers"`
	SubmittedBy ActorRef                  `json:"submitted_by"`
	State       ClarificationAnswerState  `json:"state"`
	CreatedAt   time.Time                 `json:"created_at"`
}

type ClarificationAnswerItem struct {
	QuestionID ClarificationQuestionID `json:"question_id"`
	Value      string                  `json:"value"`
	ActorRef   *ActorRef               `json:"actor_ref,omitempty"`
}

type ClarificationAnswerSubmission struct {
	Answers     []ClarificationAnswerItem `json:"answers"`
	SubmittedBy ActorRef                  `json:"submitted_by"`
}

type ClarificationAnswerApplicationRequest struct {
	AppliedBy ActorRef `json:"applied_by"`
}

type ClarificationAnswerAppliedMapping struct {
	QuestionID ClarificationQuestionID `json:"question_id"`
	MapsTo     ClarificationMapsTo     `json:"maps_to"`
	OldValue   string                  `json:"old_value,omitempty"`
	NewValue   string                  `json:"new_value"`
}

type ClarificationAnswerApplicationResult struct {
	AnswerID        ClarificationAnswerID               `json:"answer_id"`
	GoalID          GoalID                              `json:"goal_id"`
	AppliedBy       ActorRef                            `json:"applied_by"`
	AppliedMappings []ClarificationAnswerAppliedMapping `json:"applied_mappings"`
	AppliedAt       time.Time                           `json:"applied_at"`
}
