package contract

import (
	"errors"
	"strings"

	"github.com/heurema/goalrail/apps/cli/internal/spine"
)

var ErrValidationFailed = errors.New("contract validation failed")

func Validate(contract spine.Contract) spine.ContractValidationReport {
	findings := []spine.ContractValidationFinding{}

	addRequired := func(field, message string) {
		findings = append(findings, spine.ContractValidationFinding{Field: field, Severity: "error", Message: message})
	}

	if strings.TrimSpace(string(contract.RepoBindingID)) == "" {
		addRequired("repo_binding_id", "repo_binding_id is required before a contract can be approved or executed")
	}
	if strings.TrimSpace(contract.Goal) == "" {
		addRequired("goal", "goal is required before a contract can be approved or executed")
	}
	if len(nonEmpty(contract.InScope)) == 0 {
		addRequired("in_scope", "in_scope must include at least one item")
	}
	if len(nonEmpty(contract.AcceptanceCriteria)) == 0 {
		addRequired("acceptance_criteria", "acceptance_criteria must include at least one item")
	}
	if len(nonEmpty(contract.ProofExpectations)) == 0 {
		addRequired("proof_expectations", "proof_expectations must include at least one item")
	}
	if !validState(contract.State) {
		addRequired("state", "state must be draft, needs_clarification, approved, or rejected")
	}

	return spine.ContractValidationReport{
		Valid:      len(findings) == 0,
		ContractID: contract.ID,
		Findings:   findings,
	}
}

func validState(state spine.ContractState) bool {
	switch state {
	case spine.ContractStateDraft, spine.ContractStateNeedsClarification, spine.ContractStateApproved, spine.ContractStateRejected:
		return true
	default:
		return false
	}
}

func nonEmpty(values []string) []string {
	out := []string{}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}
