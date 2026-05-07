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

type SourceRef struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
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

type WorkAnswerOutput struct {
	SchemaVersion          string         `json:"schema_version"`
	Mode                   string         `json:"mode"`
	ServerURL              string         `json:"server_url"`
	OrganizationID         string         `json:"organization_id"`
	ProjectID              string         `json:"project_id"`
	RepoBindingID          string         `json:"repo_binding_id"`
	GoalID                 string         `json:"goal_id"`
	State                  string         `json:"state"`
	ClarificationRequestID string         `json:"clarification_request_id"`
	LocalConfigPath        string         `json:"local_config_path"`
	Display                DisplaySummary `json:"display"`
	NextAction             NextAction     `json:"next_action"`
}

type WorkPlanOutput struct {
	SchemaVersion   string         `json:"schema_version"`
	Mode            string         `json:"mode"`
	ServerURL       string         `json:"server_url"`
	OrganizationID  string         `json:"organization_id"`
	ProjectID       string         `json:"project_id"`
	RepoBindingID   RepoBindingID  `json:"repo_binding_id"`
	ContractID      ContractID     `json:"contract_id"`
	PlanID          string         `json:"plan_id"`
	PlanState       string         `json:"plan_state"`
	LocalConfigPath string         `json:"local_config_path"`
	Display         DisplaySummary `json:"display"`
	NextAction      NextAction     `json:"next_action"`
}

type ProposedWorkItem struct {
	Title                string      `json:"title"`
	Summary              string      `json:"summary"`
	Scope                []string    `json:"scope"`
	AcceptanceRefs       []string    `json:"acceptance_refs"`
	ProofExpectationRefs []string    `json:"proof_expectation_refs"`
	OwnerHint            string      `json:"owner_hint,omitempty"`
	OrderIndex           *int        `json:"order_index,omitempty"`
	SourceRefs           []SourceRef `json:"source_refs,omitempty"`
}

type WorkPlanStatusOutput struct {
	SchemaVersion   string             `json:"schema_version"`
	Mode            string             `json:"mode"`
	ServerURL       string             `json:"server_url"`
	OrganizationID  string             `json:"organization_id"`
	ProjectID       string             `json:"project_id"`
	RepoBindingID   RepoBindingID      `json:"repo_binding_id"`
	ContractID      ContractID         `json:"contract_id"`
	PlanID          string             `json:"plan_id"`
	PlanState       string             `json:"plan_state"`
	ProposalID      string             `json:"proposal_id,omitempty"`
	ProposalState   string             `json:"proposal_state,omitempty"`
	ProposedTasks   []ProposedWorkItem `json:"proposed_tasks,omitempty"`
	LocalConfigPath string             `json:"local_config_path"`
	Display         DisplaySummary     `json:"display"`
	NextAction      NextAction         `json:"next_action"`
}

type WorkProposalAcceptOutput struct {
	SchemaVersion   string         `json:"schema_version"`
	Mode            string         `json:"mode"`
	ServerURL       string         `json:"server_url"`
	OrganizationID  string         `json:"organization_id"`
	ProjectID       string         `json:"project_id"`
	RepoBindingID   RepoBindingID  `json:"repo_binding_id"`
	ContractID      ContractID     `json:"contract_id"`
	PlanID          string         `json:"plan_id"`
	ProposalID      string         `json:"proposal_id"`
	ProposalState   string         `json:"proposal_state"`
	CreatedTaskIDs  []string       `json:"created_task_ids"`
	LocalConfigPath string         `json:"local_config_path"`
	Display         DisplaySummary `json:"display"`
	NextAction      NextAction     `json:"next_action"`
}

type CheckoutInstruction struct {
	JobID              string        `json:"job_id"`
	TaskID             string        `json:"task_id"`
	RepoBindingID      RepoBindingID `json:"repo_binding_id"`
	AccessMode         string        `json:"access_mode"`
	Provider           string        `json:"provider"`
	RepositoryFullName string        `json:"repository_full_name"`
	RepositoryURL      string        `json:"repository_url"`
	WorkflowBaseBranch string        `json:"workflow_base_branch"`
	PathScope          string        `json:"path_scope"`
	SourceRef          SourceRef     `json:"source_ref"`
	RawSourceUploaded  bool          `json:"raw_source_uploaded"`
}

type WorkCheckoutPrepareOutput struct {
	SchemaVersion    string              `json:"schema_version"`
	Mode             string              `json:"mode"`
	ServerURL        string              `json:"server_url"`
	OrganizationID   string              `json:"organization_id"`
	ProjectID        string              `json:"project_id"`
	RepoBindingID    RepoBindingID       `json:"repo_binding_id"`
	TaskID           string              `json:"task_id"`
	CheckoutJobID    string              `json:"checkout_job_id"`
	CheckoutJobState string              `json:"checkout_job_state"`
	Instruction      CheckoutInstruction `json:"instruction"`
	LocalConfigPath  string              `json:"local_config_path"`
	Display          DisplaySummary      `json:"display"`
	NextAction       NextAction          `json:"next_action"`
}
