package spine

import (
	"encoding/json"
	"time"
)

type IntakeID string

type EventID string

type RepoBindingID string

type IntakeState string

const IntakeStateReceived IntakeState = "received"

type IntakeSource struct {
	Kind       string `json:"kind"`
	ExternalID string `json:"external_id,omitempty"`
	URL        string `json:"url,omitempty"`
}

type ActorRef struct {
	Kind        string `json:"kind"`
	ID          string `json:"id"`
	DisplayName string `json:"display_name,omitempty"`
}

type IntakeSubmission struct {
	RepoBindingID RepoBindingID `json:"repo_binding_id"`
	Source        IntakeSource  `json:"source"`
	Title         string        `json:"title"`
	Body          string        `json:"body"`
	RequestAuthor ActorRef      `json:"request_author"`
	IntentOwner   *ActorRef     `json:"intent_owner,omitempty"`
}

type IntakeRecord struct {
	ID                       IntakeID      `json:"id"`
	RepoBindingID            RepoBindingID `json:"repo_binding_id"`
	Source                   IntakeSource  `json:"source"`
	Title                    string        `json:"title"`
	Body                     string        `json:"body"`
	RequestAuthor            ActorRef      `json:"request_author"`
	IntentOwner              ActorRef      `json:"intent_owner"`
	State                    IntakeState   `json:"state"`
	CanonicalContractCreated bool          `json:"canonical_contract_created"`
	CreatedAt                time.Time     `json:"created_at"`
}

type Event struct {
	ID         EventID         `json:"id"`
	Type       string          `json:"type"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	Timestamp  time.Time       `json:"timestamp"`
	Payload    json.RawMessage `json:"payload"`
}
