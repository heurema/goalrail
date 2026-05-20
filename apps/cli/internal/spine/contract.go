package spine

// ContractState enumerates the local/demo CLI Contract DTO states.
//
// These are compatibility states for the offline `goalrail contract validate`
// path; they do not mirror the canonical server-owned lifecycle, where Goal,
// ContractSeed, ContractDraft(draft), ContractDraft(ready_for_approval), and
// ApprovedContract(approved) are separate objects with their own state types.
// CLI integration with that server-owned lifecycle is a separate planned slice.
type ContractState string

// Local/demo CLI Contract DTO states used by `goalrail contract validate`.
// These are compatibility values for the local validator only; they are not
// the canonical server-owned ContractDraft / ApprovedContract states.
const (
	ContractStateDraft              ContractState = "draft"
	ContractStateNeedsClarification ContractState = "needs_clarification"
	ContractStateReadyForApproval   ContractState = "ready_for_approval"
	ContractStateApproved           ContractState = "approved"
	ContractStateRejected           ContractState = "rejected"
)

// Contract is the local/demo JSON DTO consumed by the CLI
// `goalrail contract validate` path. It is not a mirror of the canonical
// server-owned contract lifecycle: server canon splits scope shaping across
// ContractSeed, ContractDraft(draft), ContractDraft(ready_for_approval), and
// ApprovedContract(approved), while this CLI type keeps a single object with
// a single State field for the offline validator only. Future CLI integration
// with the server lifecycle is a separate planned slice.
type Contract struct {
	ID                 ContractID    `json:"id"`
	RepoBindingID      RepoBindingID `json:"repo_binding_id"`
	Goal               string        `json:"goal"`
	BusinessContext    string        `json:"business_context"`
	InScope            []string      `json:"in_scope"`
	OutOfScope         []string      `json:"out_of_scope"`
	Constraints        []string      `json:"constraints"`
	AcceptanceCriteria []string      `json:"acceptance_criteria"`
	ExpectedChecks     []string      `json:"expected_checks"`
	ProofExpectations  []string      `json:"proof_expectations"`
	State              ContractState `json:"state"`
}

type ContractValidationReport struct {
	Valid      bool                        `json:"valid"`
	ContractID ContractID                  `json:"contract_id,omitempty"`
	Findings   []ContractValidationFinding `json:"findings"`
}

type ContractValidationFinding struct {
	Field    string `json:"field"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type LocalRepoReceipt struct {
	RepoBindingID            RepoBindingID `json:"repo_binding_id"`
	HeadSHA                  string        `json:"head_sha"`
	BaselineID               string        `json:"baseline_id,omitempty"`
	OverlayID                string        `json:"overlay_id"`
	OverlayState             string        `json:"overlay_state"`
	Freshness                string        `json:"freshness"`
	Dirty                    bool          `json:"dirty"`
	Partial                  bool          `json:"partial"`
	RawSourceUploaded        bool          `json:"raw_source_uploaded"`
	BaselineRebuilt          bool          `json:"baseline_rebuilt"`
	PartialReasons           []string      `json:"partial_reasons,omitempty"`
	ScanCriticalChangedPaths []string      `json:"scan_critical_changed_paths,omitempty"`
	UnmergedPaths            []string      `json:"unmerged_paths,omitempty"`
}

type ContractDraftOutput struct {
	SchemaVersion    string               `json:"schema_version"`
	Mode             string               `json:"mode"`
	ServerURL        string               `json:"server_url"`
	AuthSession      *AuthSessionMetadata `json:"auth_session,omitempty"`
	OrganizationID   string               `json:"organization_id"`
	ProjectID        string               `json:"project_id"`
	RepoBindingID    RepoBindingID        `json:"repo_binding_id"`
	GoalID           string               `json:"goal_id"`
	ContractID       ContractID           `json:"contract_id"`
	ContractState    ContractState        `json:"contract_state"`
	LocalRepoReceipt LocalRepoReceipt     `json:"local_repo_receipt"`
	LocalConfigPath  string               `json:"local_config_path"`
	Display          DisplaySummary       `json:"display"`
	NextAction       NextAction           `json:"next_action"`
}

type ContractUpdateOutput struct {
	SchemaVersion   string               `json:"schema_version"`
	Mode            string               `json:"mode"`
	ServerURL       string               `json:"server_url"`
	AuthSession     *AuthSessionMetadata `json:"auth_session,omitempty"`
	OrganizationID  string               `json:"organization_id"`
	ProjectID       string               `json:"project_id"`
	RepoBindingID   RepoBindingID        `json:"repo_binding_id"`
	ContractID      ContractID           `json:"contract_id"`
	ContractState   ContractState        `json:"contract_state"`
	ChangedFields   []string             `json:"changed_fields"`
	LocalConfigPath string               `json:"local_config_path"`
	Display         DisplaySummary       `json:"display"`
	NextAction      NextAction           `json:"next_action"`
}

type ContractTransitionOutput struct {
	SchemaVersion   string               `json:"schema_version"`
	Mode            string               `json:"mode"`
	ServerURL       string               `json:"server_url"`
	AuthSession     *AuthSessionMetadata `json:"auth_session,omitempty"`
	OrganizationID  string               `json:"organization_id"`
	ProjectID       string               `json:"project_id"`
	RepoBindingID   RepoBindingID        `json:"repo_binding_id"`
	ContractID      ContractID           `json:"contract_id"`
	ContractState   ContractState        `json:"contract_state"`
	LocalConfigPath string               `json:"local_config_path"`
	Display         DisplaySummary       `json:"display"`
	NextAction      NextAction           `json:"next_action"`
}

type ContractShowDraft struct {
	ID                         string   `json:"id"`
	State                      string   `json:"state"`
	Title                      string   `json:"title,omitempty"`
	IntentSummary              string   `json:"intent_summary,omitempty"`
	ProposedScope              []string `json:"proposed_scope,omitempty"`
	ProposedNonGoals           []string `json:"proposed_non_goals,omitempty"`
	ProposedConstraints        []string `json:"proposed_constraints,omitempty"`
	ProposedAcceptanceCriteria []string `json:"proposed_acceptance_criteria,omitempty"`
	ProposedExpectedChecks     []string `json:"proposed_expected_checks,omitempty"`
	ProposedProofExpectations  []string `json:"proposed_proof_expectations,omitempty"`
	RiskHints                  []string `json:"risk_hints,omitempty"`
}

type ContractShowOutput struct {
	SchemaVersion   string               `json:"schema_version"`
	Mode            string               `json:"mode"`
	ServerURL       string               `json:"server_url"`
	AuthSession     *AuthSessionMetadata `json:"auth_session,omitempty"`
	OrganizationID  string               `json:"organization_id"`
	ProjectID       string               `json:"project_id"`
	RepoBindingID   RepoBindingID        `json:"repo_binding_id"`
	GoalID          string               `json:"goal_id"`
	ContractID      ContractID           `json:"contract_id"`
	ContractState   ContractState        `json:"contract_state"`
	CurrentSeedID   string               `json:"current_seed_id,omitempty"`
	CurrentDraftID  string               `json:"current_draft_id,omitempty"`
	CurrentDraft    *ContractShowDraft   `json:"current_draft,omitempty"`
	LocalConfigPath string               `json:"local_config_path"`
	Display         DisplaySummary       `json:"display"`
	NextAction      NextAction           `json:"next_action"`
}
