package spine

type ContractState string

const (
	ContractStateDraft              ContractState = "draft"
	ContractStateNeedsClarification ContractState = "needs_clarification"
	ContractStateApproved           ContractState = "approved"
	ContractStateRejected           ContractState = "rejected"
)

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
