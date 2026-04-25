package spine

type ProofVerdict string

const (
	ProofVerdictAccept   ProofVerdict = "accept"
	ProofVerdictBlock    ProofVerdict = "block"
	ProofVerdictEscalate ProofVerdict = "escalate"
)

type Proof struct {
	ID              ProofID         `json:"id"`
	ContractID      ContractID      `json:"contract_id"`
	RunID           RunID           `json:"run_id"`
	Verdict         ProofVerdict    `json:"verdict"`
	ChangedScope    []string        `json:"changed_scope"`
	UnchangedScope  []string        `json:"unchanged_scope"`
	Coverage        []ProofCoverage `json:"coverage"`
	Checks          []ProofCheck    `json:"checks"`
	ResidualRisks   []string        `json:"residual_risks"`
	DecisionSummary string          `json:"decision_summary"`
}

type ProofCoverage struct {
	Criterion string   `json:"criterion"`
	Evidence  []string `json:"evidence"`
	Status    string   `json:"status"`
}

type ProofCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Note   string `json:"note"`
}
