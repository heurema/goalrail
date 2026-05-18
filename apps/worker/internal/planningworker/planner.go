package planningworker

import (
	"errors"
	"fmt"
	"strings"
)

var errUnsupportedPlannerInput = errors.New("unsupported planner input")

type proposalPlanner interface {
	BuildProposal(workerID string, lease planLease, plan workItemPlan) (proposalSubmitRequest, error)
}

type minimalPlanner struct{}

func (minimalPlanner) BuildProposal(workerID string, lease planLease, plan workItemPlan) (proposalSubmitRequest, error) {
	if strings.TrimSpace(workerID) == "" {
		return proposalSubmitRequest{}, fmt.Errorf("%w: worker id is required", errUnsupportedPlannerInput)
	}
	if strings.TrimSpace(plan.ID) == "" || strings.TrimSpace(plan.ContractID) == "" || strings.TrimSpace(plan.ApprovedContractID) == "" || strings.TrimSpace(plan.RepoBindingID) == "" {
		return proposalSubmitRequest{}, fmt.Errorf("%w: plan response is missing required context", errUnsupportedPlannerInput)
	}
	if plan.ID != lease.PlanID || plan.ContractID != lease.ContractID || plan.ApprovedContractID != lease.ApprovedContractID || plan.RepoBindingID != lease.RepoBindingID {
		return proposalSubmitRequest{}, fmt.Errorf("%w: lease and plan context mismatch", errUnsupportedPlannerInput)
	}
	if plan.State != "leased" {
		return proposalSubmitRequest{}, fmt.Errorf("%w: plan state %q is not leased", errUnsupportedPlannerInput, plan.State)
	}
	if err := validateApprovedContractSnapshot(plan); err != nil {
		return proposalSubmitRequest{}, err
	}

	approvedRef := sourceRef{Kind: "approved_contract", ID: plan.ApprovedContractID}
	order := 0
	task := proposedTaskFromApprovedContract(*plan.ApprovedContract, approvedRef, &order)
	return proposalSubmitRequest{
		LeaseID:    lease.ID,
		LeaseToken: lease.LeaseToken,
		SubmittedBy: actorRef{
			Kind: "worker",
			ID:   workerID,
		},
		Planner: map[string]any{
			"kind":    "goalrail_worker",
			"id":      workerID,
			"mode":    "contract_projection",
			"version": plannerVersion,
		},
		SourceSnapshotRefs: []sourceRef{approvedRef},
		Rationale:          "Deterministic planning worker proposal projected from approved Contract title, intent, scope, constraints, non-goals, acceptance criteria, and proof expectations. This worker does not call an LLM, inspect repositories, run code, or verify outcomes.",
		ProposedTasks:      []proposedWorkItem{task},
	}, nil
}

func validateApprovedContractSnapshot(plan workItemPlan) error {
	approved := plan.ApprovedContract
	if approved == nil {
		return fmt.Errorf("%w: plan response is missing approved contract projection", errUnsupportedPlannerInput)
	}
	if strings.TrimSpace(approved.ID) == "" || approved.ID != plan.ApprovedContractID {
		return fmt.Errorf("%w: approved contract projection id mismatch", errUnsupportedPlannerInput)
	}
	if strings.TrimSpace(approved.ContractID) == "" || approved.ContractID != plan.ContractID {
		return fmt.Errorf("%w: approved contract projection contract mismatch", errUnsupportedPlannerInput)
	}
	if strings.TrimSpace(approved.RepoBindingID) == "" || approved.RepoBindingID != plan.RepoBindingID {
		return fmt.Errorf("%w: approved contract projection repo binding mismatch", errUnsupportedPlannerInput)
	}
	if strings.TrimSpace(approved.Title) == "" && strings.TrimSpace(approved.IntentSummary) == "" && len(nonBlankStrings(approved.Scope)) == 0 {
		return fmt.Errorf("%w: approved contract projection is missing title, intent, and scope", errUnsupportedPlannerInput)
	}
	return nil
}

func proposedTaskFromApprovedContract(approved approvedContractSnapshot, approvedRef sourceRef, order *int) proposedWorkItem {
	title := strings.TrimSpace(approved.Title)
	if title == "" {
		title = "Implement approved contract"
	}
	summary := strings.TrimSpace(approved.IntentSummary)
	if summary == "" {
		summary = "Implement the approved Contract according to its server-owned scope and acceptance criteria."
	}

	scope := nonBlankStrings(approved.Scope)
	if len(approved.Constraints) > 0 {
		scope = append(scope, prefixedItems("Constraint: ", approved.Constraints)...)
	}
	if len(approved.NonGoals) > 0 {
		scope = append(scope, prefixedItems("Non-goal: ", approved.NonGoals)...)
	}
	if len(scope) == 0 {
		scope = []string{"Implement the approved Contract without expanding execution scope."}
	}

	return proposedWorkItem{
		Title:                title,
		Summary:              summary,
		Scope:                scope,
		AcceptanceRefs:       indexedRefs("acceptance_criteria", len(nonBlankStrings(approved.AcceptanceCriteria))),
		ProofExpectationRefs: proofRefs(approved),
		OrderIndex:           order,
		SourceRefs:           []sourceRef{approvedRef},
	}
}

func proofRefs(approved approvedContractSnapshot) []string {
	if count := len(nonBlankStrings(approved.ProofExpectations)); count > 0 {
		return indexedRefs("proof_expectations", count)
	}
	if count := len(nonBlankStrings(approved.ExpectedChecks)); count > 0 {
		return indexedRefs("expected_checks", count)
	}
	return []string{"proof_expectations[0]"}
}

func indexedRefs(prefix string, count int) []string {
	if count <= 0 {
		return []string{prefix + "[0]"}
	}
	refs := make([]string, 0, count)
	for i := 0; i < count; i++ {
		refs = append(refs, fmt.Sprintf("%s[%d]", prefix, i))
	}
	return refs
}

func prefixedItems(prefix string, values []string) []string {
	clean := nonBlankStrings(values)
	out := make([]string, 0, len(clean))
	for _, value := range clean {
		out = append(out, prefix+value)
	}
	return out
}

func nonBlankStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
