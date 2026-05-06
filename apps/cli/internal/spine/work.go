package spine

type DisplaySummary struct {
	Summary string `json:"summary"`
}

type NextAction struct {
	Kind         string                     `json:"kind"`
	Blocking     bool                       `json:"blocking"`
	Command      string                     `json:"command,omitempty"`
	Available    bool                       `json:"available"`
	PlannedSlice string                     `json:"planned_slice,omitempty"`
	RequestID    string                     `json:"request_id,omitempty"`
	Questions    []ClarificationQuestionRef `json:"questions,omitempty"`
}

type ClarificationQuestionRef struct {
	ID         string `json:"id"`
	Text       string `json:"text"`
	WhyNeeded  string `json:"why_needed,omitempty"`
	AnswerType string `json:"answer_type"`
	MapsTo     string `json:"maps_to"`
}

type WorkStartOutput struct {
	SchemaVersion        string         `json:"schema_version"`
	Mode                 string         `json:"mode"`
	ServerURL            string         `json:"server_url"`
	OrganizationID       string         `json:"organization_id"`
	ProjectID            string         `json:"project_id"`
	RepoBindingID        string         `json:"repo_binding_id"`
	IntakeID             string         `json:"intake_id"`
	IntakeState          string         `json:"intake_state"`
	GoalID               string         `json:"goal_id"`
	GoalState            string         `json:"goal_state"`
	Title                string         `json:"title"`
	LocalConfigPath      string         `json:"local_config_path"`
	Display              DisplaySummary `json:"display"`
	NextAction           NextAction     `json:"next_action"`
	Message              string         `json:"message"`
	NextSuggestedCommand string         `json:"next_suggested_command"`
}

type WorkContinueOutput struct {
	SchemaVersion   string         `json:"schema_version"`
	Mode            string         `json:"mode"`
	ServerURL       string         `json:"server_url"`
	OrganizationID  string         `json:"organization_id"`
	ProjectID       string         `json:"project_id"`
	RepoBindingID   string         `json:"repo_binding_id"`
	GoalID          string         `json:"goal_id"`
	State           string         `json:"state"`
	LocalConfigPath string         `json:"local_config_path"`
	Display         DisplaySummary `json:"display"`
	NextAction      NextAction     `json:"next_action"`
}
