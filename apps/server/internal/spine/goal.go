package spine

import "time"

type GoalID string

type GoalState string

const GoalStateCreated GoalState = "created"

type SourceRef struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type Goal struct {
	ID            GoalID        `json:"id"`
	IntakeID      IntakeID      `json:"intake_id"`
	RepoBindingID RepoBindingID `json:"repo_binding_id"`
	Title         string        `json:"title"`
	Summary       string        `json:"summary"`
	SourceRefs    []SourceRef   `json:"source_refs"`
	RequestAuthor ActorRef      `json:"request_author"`
	IntentOwner   ActorRef      `json:"intent_owner"`
	State         GoalState     `json:"state"`
	CreatedAt     time.Time     `json:"created_at"`
}
