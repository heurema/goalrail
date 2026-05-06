package spine

type GoalContinuation struct {
	GoalID               GoalID                `json:"goal_id"`
	State                GoalState             `json:"state"`
	Readiness            *GoalReadinessResult  `json:"readiness,omitempty"`
	Goal                 *Goal                 `json:"goal,omitempty"`
	ClarificationRequest *ClarificationRequest `json:"clarification_request,omitempty"`
}

type WorkAnswerSubmission struct {
	Answers []ClarificationAnswerItem `json:"answers"`
}
