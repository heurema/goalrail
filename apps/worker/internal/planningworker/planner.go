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

	approvedRef := sourceRef{Kind: "approved_contract", ID: plan.ApprovedContractID}
	order := 0
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
			"mode":    "minimal_dev",
			"version": plannerVersion,
		},
		SourceSnapshotRefs: []sourceRef{approvedRef},
		Rationale:          "Minimal planning worker proposal from server-owned plan and approved contract metadata. This worker does not inspect repositories, run code, or verify outcomes.",
		ProposedTasks: []proposedWorkItem{
			{
				Title:                "Implement approved contract",
				Summary:              "Implement the approved Contract according to its server-owned scope and acceptance criteria.",
				Scope:                []string{"Implement the approved Contract without expanding execution scope."},
				AcceptanceRefs:       []string{"acceptance_criteria[0]"},
				ProofExpectationRefs: []string{"proof_expectations[0]"},
				OrderIndex:           &order,
				SourceRefs:           []sourceRef{approvedRef},
			},
		},
	}, nil
}
