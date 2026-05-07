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
	SchemaVersion    string           `json:"schema_version"`
	Mode             string           `json:"mode"`
	ServerURL        string           `json:"server_url"`
	OrganizationID   string           `json:"organization_id"`
	ProjectID        string           `json:"project_id"`
	RepoBindingID    RepoBindingID    `json:"repo_binding_id"`
	GoalID           string           `json:"goal_id"`
	ContractID       ContractID       `json:"contract_id"`
	ContractState    ContractState    `json:"contract_state"`
	LocalRepoReceipt LocalRepoReceipt `json:"local_repo_receipt"`
	LocalConfigPath  string           `json:"local_config_path"`
	Display          DisplaySummary   `json:"display"`
	NextAction       NextAction       `json:"next_action"`
}
