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
