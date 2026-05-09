package spine

import "time"

type QualificationLane string

const (
	QualificationLaneQualification QualificationLane = "qualification"
	QualificationLaneClarification QualificationLane = "clarification"
	QualificationLaneContract      QualificationLane = "contract"
	QualificationLaneBlocked       QualificationLane = "blocked"
)

type QualificationNextActionKind string

const (
	QualificationNextActionContinueGoal        QualificationNextActionKind = "continue_goal"
	QualificationNextActionAnswerClarification QualificationNextActionKind = "answer_clarification"
	QualificationNextActionDraftContract       QualificationNextActionKind = "draft_contract"
	QualificationNextActionUpdateContract      QualificationNextActionKind = "update_contract"
	QualificationNextActionApproveContract     QualificationNextActionKind = "approve_contract"
	QualificationNextActionPlanWork            QualificationNextActionKind = "plan_work"
	QualificationNextActionBlocked             QualificationNextActionKind = "blocked"
	QualificationNextActionNone                QualificationNextActionKind = "none"
)

const QualificationReadinessSourceGoalSnapshot = "goal_snapshot"

type QualificationFeedFilter struct {
	OrganizationID OrganizationID
	ProjectID      ProjectID
	RepoBindingID  RepoBindingID
	GoalState      GoalState
	Limit          int
}

type QualificationFeedRecord struct {
	IntakeID                 IntakeID
	GoalID                   GoalID
	OrganizationID           OrganizationID
	ProjectID                ProjectID
	RepoBindingID            RepoBindingID
	RepositoryFullName       string
	Title                    string
	IntakeState              IntakeState
	GoalState                GoalState
	ReadinessReasonCodes     []GoalReadinessReasonCode
	OpenClarificationRequest *QualificationOpenClarificationRequest
	LinkedContract           *QualificationLinkedContract
	CreatedAt                time.Time
}

type QualificationFeed struct {
	Items []QualificationFeedItem `json:"items"`
}

type QualificationFeedItem struct {
	IntakeID                 IntakeID                               `json:"intake_id"`
	GoalID                   GoalID                                 `json:"goal_id"`
	OrganizationID           OrganizationID                         `json:"organization_id"`
	ProjectID                ProjectID                              `json:"project_id"`
	RepoBindingID            RepoBindingID                          `json:"repo_binding_id"`
	RepositoryFullName       string                                 `json:"repository_full_name"`
	Title                    string                                 `json:"title"`
	Lane                     QualificationLane                      `json:"lane"`
	IntakeState              IntakeState                            `json:"intake_state"`
	GoalState                GoalState                              `json:"goal_state"`
	Readiness                QualificationReadinessSnapshot         `json:"readiness"`
	OpenClarificationRequest *QualificationOpenClarificationRequest `json:"open_clarification_request,omitempty"`
	LinkedContract           *QualificationLinkedContract           `json:"linked_contract,omitempty"`
	NextAction               QualificationNextAction                `json:"next_action"`
	CreatedAt                time.Time                              `json:"created_at"`
}

type QualificationReadinessSnapshot struct {
	Ready       bool                      `json:"ready"`
	ReasonCodes []GoalReadinessReasonCode `json:"reason_codes"`
	Source      string                    `json:"source"`
}

type QualificationOpenClarificationRequest struct {
	ID        ClarificationRequestID    `json:"id"`
	State     ClarificationRequestState `json:"state"`
	Questions []ClarificationQuestion   `json:"questions"`
}

type QualificationLinkedContract struct {
	ID    ContractID    `json:"id"`
	State ContractState `json:"state"`
}

type QualificationNextAction struct {
	Kind      QualificationNextActionKind `json:"kind"`
	Available bool                        `json:"available"`
	Blocking  bool                        `json:"blocking"`
}
