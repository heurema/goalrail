package httpserver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/approvedcontract"
	"github.com/heurema/goalrail/apps/server/internal/auth"
	"github.com/heurema/goalrail/apps/server/internal/execution"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/workitem"
	"github.com/heurema/goalrail/apps/server/internal/workitemplan"
)

func TestPostContractPlansReturnsQueuedPlan(t *testing.T) {
	server := testServer(t)
	approved := createApprovedContract(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contracts/"+string(approved.ContractID)+"/plans", `{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","requested_by":{"kind":"user","id":"spoofed-requester"}}`)
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	assertNoHiddenContext(t, response.body)
	assertNoForbiddenWorkItemSideEffects(t, server.events.Events())

	var plan spine.WorkItemPlan
	decodeJSON(t, response.body, &plan)
	if plan.State != spine.WorkItemPlanStateQueued {
		t.Fatalf("state = %q, want queued", plan.State)
	}
	if plan.ContractID != approved.ContractID || plan.ApprovedContractID != approved.ID || plan.RepoBindingID != approved.RepoBindingID {
		t.Fatalf("plan ids = %q/%q/%q, want approved contract ids", plan.ContractID, plan.ApprovedContractID, plan.RepoBindingID)
	}
	if plan.RequestedBy.Kind != "user" || plan.RequestedBy.ID != "018f0000-0000-7000-8000-000000000001" {
		t.Fatalf("requested_by = %#v, want authenticated user actor", plan.RequestedBy)
	}
	if _, ok, err := server.workItems.GetByApprovedContractID(context.Background(), approved.ID); err != nil {
		t.Fatalf("workItems.GetByApprovedContractID() error = %v", err)
	} else if ok {
		t.Fatal("plan creation materialized a WorkItem")
	}
	if len(server.workItemLeases.leases) != 0 || len(server.workItemProposals.proposals) != 0 {
		t.Fatalf("leases/proposals = %d/%d, want 0/0", len(server.workItemLeases.leases), len(server.workItemProposals.proposals))
	}
}

func TestPostContractPlansIsAuthenticatedAndIdempotent(t *testing.T) {
	t.Run("invalid bearer token rejects before plan creation", func(t *testing.T) {
		server := testServerWithContinuationAuth(t, fakeHTTPAuthService{meErr: auth.ErrInvalidToken})

		response := doJSON(t, server.router, http.MethodPost, "/v1/contracts/018f0000-0000-7000-8000-000000000009/plans", `{}`)
		assertErrorCode(t, response, http.StatusUnauthorized, "unauthorized")
		if len(server.workItemPlans.plans) != 0 {
			t.Fatalf("plans = %d, want 0 after auth failure", len(server.workItemPlans.plans))
		}
	})

	t.Run("repeated create returns existing plan", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)

		first := doJSON(t, server.router, http.MethodPost, "/v1/contracts/"+string(approved.ContractID)+"/plans", `{}`)
		if first.code != http.StatusCreated {
			t.Fatalf("first status = %d, want %d: %s", first.code, http.StatusCreated, first.body)
		}
		second := doJSON(t, server.router, http.MethodPost, "/v1/contracts/"+string(approved.ContractID)+"/plans", `{}`)
		if second.code != http.StatusOK {
			t.Fatalf("second status = %d, want %d: %s", second.code, http.StatusOK, second.body)
		}
		var firstPlan, secondPlan spine.WorkItemPlan
		decodeJSON(t, first.body, &firstPlan)
		decodeJSON(t, second.body, &secondPlan)
		if firstPlan.ID != secondPlan.ID {
			t.Fatalf("plan ids = %q/%q, want same existing plan", firstPlan.ID, secondPlan.ID)
		}
		if len(server.workItemPlans.plans) != 1 {
			t.Fatalf("plans = %d, want 1 after repeated create", len(server.workItemPlans.plans))
		}
	})
}

func TestPostContractPlansRejectsInvalidInputs(t *testing.T) {
	t.Run("unknown contract", func(t *testing.T) {
		server := testServer(t)
		response := doJSON(t, server.router, http.MethodPost, "/v1/contracts/missing/plans", `{}`)
		assertErrorCode(t, response, http.StatusNotFound, "not_found")
	})

	t.Run("non-approved contract", func(t *testing.T) {
		server := testServer(t)
		contract := createContract(t, server)
		response := doJSON(t, server.router, http.MethodPost, "/v1/contracts/"+string(contract.ID)+"/plans", `{}`)
		assertErrorCode(t, response, http.StatusConflict, "invalid_state")
	})

	t.Run("organization mismatch", func(t *testing.T) {
		server := testServerWithContinuationAuth(t, fakeHTTPAuthService{
			profile: continuationAuthProfile("018f0000-0000-7000-8000-000000000999"),
		})
		approved := storeApprovedPlanningFixture(t, server)
		response := doJSON(t, server.router, http.MethodPost, "/v1/contracts/"+string(approved.ContractID)+"/plans", `{}`)
		assertErrorCode(t, response, http.StatusForbidden, "forbidden")
		if len(server.workItemPlans.plans) != 0 {
			t.Fatalf("plans = %d, want 0 after org mismatch", len(server.workItemPlans.plans))
		}
	})

	t.Run("project expectation mismatch", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		response := doJSON(t, server.router, http.MethodPost, "/v1/contracts/"+string(approved.ContractID)+"/plans", `{"project_id":"project-mismatch"}`)
		assertErrorCode(t, response, http.StatusConflict, "project_context_mismatch")
		if len(server.workItemPlans.plans) != 0 {
			t.Fatalf("plans = %d, want 0 after project mismatch", len(server.workItemPlans.plans))
		}
	})

	t.Run("repo binding expectation mismatch", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		response := doJSON(t, server.router, http.MethodPost, "/v1/contracts/"+string(approved.ContractID)+"/plans", `{"repo_binding_id":"repo-binding-mismatch"}`)
		assertErrorCode(t, response, http.StatusConflict, "project_context_mismatch")
		if len(server.workItemPlans.plans) != 0 {
			t.Fatalf("plans = %d, want 0 after repo mismatch", len(server.workItemPlans.plans))
		}
	})
}

func TestGetPlanReturnsPlanAndUnknownReturnsNotFound(t *testing.T) {
	server := testServer(t)
	approved := createApprovedContract(t, server)
	plan := createPlan(t, server, approved.ContractID)

	response := doJSON(t, server.router, http.MethodGet, "/v1/plans/"+string(plan.ID), "")
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	var got spine.WorkItemPlan
	decodeJSON(t, response.body, &got)
	if got.ID != plan.ID {
		t.Fatalf("id = %q, want %q", got.ID, plan.ID)
	}

	missing := doJSON(t, server.router, http.MethodGet, "/v1/plans/missing", "")
	assertErrorCode(t, missing, http.StatusNotFound, "not_found")
}

func TestPostPlanStatusReturnsProposalAndRequiresContext(t *testing.T) {
	t.Run("submitted proposal included", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		plan := createPlan(t, server, approved.ContractID)
		proposal := submitProposal(t, server, plan.ID, string(approved.ID))

		beforePlan := server.workItemPlans.plans[plan.ID]
		beforeLeaseCount := len(server.workItemLeases.leases)
		beforeProposalCount := len(server.workItemProposals.proposals)
		beforeWorkItemCount := len(server.workItems.items)
		beforeEventCount := len(server.events.Events())
		response := doJSON(t, server.router, http.MethodPost, "/v1/plans/"+string(plan.ID)+"/status", `{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004"}`)
		if response.code != http.StatusOK {
			t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
		}
		assertNoHiddenContext(t, response.body)
		for _, forbidden := range []string{"\"lease_token\"", "\"lease_token_hash\""} {
			if strings.Contains(response.body, forbidden) {
				t.Fatalf("status response exposes worker secret field %s: %s", forbidden, response.body)
			}
		}
		var status spine.WorkItemPlanStatus
		decodeJSON(t, response.body, &status)
		if status.Plan.ID != plan.ID || status.Plan.State != spine.WorkItemPlanStateProposalSubmitted {
			t.Fatalf("plan status = %q/%q, want submitted plan %q", status.Plan.ID, status.Plan.State, plan.ID)
		}
		if status.Proposal == nil || status.Proposal.ID != proposal.ID || len(status.Proposal.ProposedTasks) != 2 {
			t.Fatalf("proposal = %#v, want submitted proposal with tasks", status.Proposal)
		}
		afterPlan := server.workItemPlans.plans[plan.ID]
		if afterPlan.State != beforePlan.State || afterPlan.CurrentLeaseID != beforePlan.CurrentLeaseID || !afterPlan.UpdatedAt.Equal(beforePlan.UpdatedAt) {
			t.Fatalf("status route mutated plan = %#v, want %#v", afterPlan, beforePlan)
		}
		if len(server.workItemLeases.leases) != beforeLeaseCount {
			t.Fatalf("leases = %d, want %d after status read", len(server.workItemLeases.leases), beforeLeaseCount)
		}
		if len(server.workItemProposals.proposals) != beforeProposalCount {
			t.Fatalf("proposals = %d, want %d after status read", len(server.workItemProposals.proposals), beforeProposalCount)
		}
		if len(server.workItems.items) != beforeWorkItemCount {
			t.Fatalf("work items = %d, want %d after status read", len(server.workItems.items), beforeWorkItemCount)
		}
		if len(server.events.Events()) != beforeEventCount {
			t.Fatalf("events = %d, want %d after status read", len(server.events.Events()), beforeEventCount)
		}
	})

	t.Run("unauthenticated rejects before read", func(t *testing.T) {
		server := testServerWithContinuationAuth(t, fakeHTTPAuthService{meErr: auth.ErrInvalidToken})
		response := doJSON(t, server.router, http.MethodPost, "/v1/plans/plan-1/status", `{}`)
		assertErrorCode(t, response, http.StatusUnauthorized, "unauthorized")
	})

	t.Run("project expectation mismatch", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		plan := createPlan(t, server, approved.ContractID)
		response := doJSON(t, server.router, http.MethodPost, "/v1/plans/"+string(plan.ID)+"/status", `{"project_id":"project-mismatch"}`)
		assertErrorCode(t, response, http.StatusConflict, "project_context_mismatch")
	})

	t.Run("repo expectation mismatch", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		plan := createPlan(t, server, approved.ContractID)
		response := doJSON(t, server.router, http.MethodPost, "/v1/plans/"+string(plan.ID)+"/status", `{"repo_binding_id":"repo-binding-mismatch"}`)
		assertErrorCode(t, response, http.StatusConflict, "project_context_mismatch")
	})
}

func TestPlanLeaseLifecycleRoutes(t *testing.T) {
	t.Run("queued plan acquires active lease and token once", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		plan := createPlan(t, server, approved.ContractID)

		response := doJSON(t, server.router, http.MethodPost, "/v1/plans/leases", `{"leased_by":{"kind":"worker","id":"planner-worker-1"}}`)
		if response.code != http.StatusCreated {
			t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
		}
		assertNoHiddenContext(t, response.body)
		var lease spine.WorkItemPlanLeaseCreated
		decodeJSON(t, response.body, &lease)
		if lease.PlanID != plan.ID || lease.State != spine.WorkItemPlanLeaseStateActive || lease.LeaseToken == "" {
			t.Fatalf("lease = %#v, want active lease for plan with token", lease)
		}
		if lease.ExpiresAt.Sub(testTime()) != 15*time.Minute {
			t.Fatalf("default ttl = %s, want 15m", lease.ExpiresAt.Sub(testTime()))
		}

		getResponse := doJSON(t, server.router, http.MethodGet, "/v1/plans/leases/"+string(lease.ID), "")
		if getResponse.code != http.StatusOK {
			t.Fatalf("get status = %d, want %d: %s", getResponse.code, http.StatusOK, getResponse.body)
		}
		if strings.Contains(getResponse.body, "lease_token") {
			t.Fatalf("GET lease response exposes raw token: %s", getResponse.body)
		}

		renew := doJSON(t, server.router, http.MethodPatch, "/v1/plans/leases/"+string(lease.ID), fmt.Sprintf(`{"lease_token":%q,"ttl_seconds":1800}`, lease.LeaseToken))
		if renew.code != http.StatusOK {
			t.Fatalf("renew status = %d, want %d: %s", renew.code, http.StatusOK, renew.body)
		}
		if strings.Contains(renew.body, "lease_token") {
			t.Fatalf("renew response exposes raw token: %s", renew.body)
		}
	})

	t.Run("no queued plans returns 204", func(t *testing.T) {
		server := testServer(t)
		response := doJSON(t, server.router, http.MethodPost, "/v1/plans/leases", `{"leased_by":{"kind":"worker","id":"planner-worker-1"}}`)
		if response.code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d: %s", response.code, http.StatusNoContent, response.body)
		}
	})

	t.Run("invalid lease create input", func(t *testing.T) {
		server := testServer(t)
		createApprovedContract(t, server)
		response := doJSON(t, server.router, http.MethodPost, "/v1/plans/leases", `{}`)
		assertErrorCode(t, response, http.StatusBadRequest, "validation_failed")
		response = doJSON(t, server.router, http.MethodPost, "/v1/plans/leases", `{"leased_by":{"kind":"worker","id":"planner-worker-1"},"ttl_seconds":29}`)
		assertErrorCode(t, response, http.StatusBadRequest, "validation_failed")
		response = doJSON(t, server.router, http.MethodPost, "/v1/plans/leases", `{"leased_by":{"kind":"worker","id":"planner-worker-1"},"ttl_seconds":3601}`)
		assertErrorCode(t, response, http.StatusBadRequest, "validation_failed")
	})

	t.Run("renew rejects bad expired and completed leases", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		plan := createPlan(t, server, approved.ContractID)
		lease := acquireLease(t, server)
		bad := doJSON(t, server.router, http.MethodPatch, "/v1/plans/leases/"+string(lease.ID), `{"lease_token":"bad-token"}`)
		assertErrorCode(t, bad, http.StatusConflict, "invalid_lease")

		stored := server.workItemLeases.leases[lease.ID]
		stored.ExpiresAt = testTime().Add(-time.Minute)
		server.workItemLeases.leases[lease.ID] = stored
		expired := doJSON(t, server.router, http.MethodPatch, "/v1/plans/leases/"+string(lease.ID), fmt.Sprintf(`{"lease_token":%q}`, lease.LeaseToken))
		assertErrorCode(t, expired, http.StatusConflict, "lease_expired")

		server = testServer(t)
		approved = createApprovedContract(t, server)
		plan = createPlan(t, server, approved.ContractID)
		lease = acquireLease(t, server)
		response := doJSON(t, server.router, http.MethodPost, "/v1/plans/"+string(plan.ID)+"/proposals", validProposalJSON(string(approved.ID), lease))
		if response.code != http.StatusCreated {
			t.Fatalf("submit proposal status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
		}
		completed := doJSON(t, server.router, http.MethodPatch, "/v1/plans/leases/"+string(lease.ID), fmt.Sprintf(`{"lease_token":%q}`, lease.LeaseToken))
		assertErrorCode(t, completed, http.StatusConflict, "lease_completed")
	})
}

func TestPlanLeaseQueueSelection(t *testing.T) {
	server := testServer(t)
	first := createPlan(t, server, createApprovedContract(t, server).ContractID)
	second := createPlan(t, server, createApprovedContract(t, server).ContractID)
	third := createPlan(t, server, createApprovedContract(t, server).ContractID)
	fourth := createPlan(t, server, createApprovedContract(t, server).ContractID)

	lease := acquireLease(t, server)
	if lease.PlanID != first.ID {
		t.Fatalf("first leased plan = %q, want oldest %q", lease.PlanID, first.ID)
	}
	next := acquireLease(t, server)
	if next.PlanID != second.ID {
		t.Fatalf("second leased plan = %q, want %q", next.PlanID, second.ID)
	}
	expiredPlan := server.workItemPlans.plans[first.ID]
	expiredPlan.LeaseExpiresAt = ptrTime(testTime())
	server.workItemPlans.plans[first.ID] = expiredPlan
	proposalSubmitted := server.workItemPlans.plans[third.ID]
	proposalSubmitted.State = spine.WorkItemPlanStateProposalSubmitted
	server.workItemPlans.plans[third.ID] = proposalSubmitted
	accepted := server.workItemPlans.plans[fourth.ID]
	accepted.State = spine.WorkItemPlanStateAccepted
	server.workItemPlans.plans[fourth.ID] = accepted

	releasedAgain := acquireLease(t, server)
	if releasedAgain.PlanID != first.ID {
		t.Fatalf("expired leased plan = %q, want released %q", releasedAgain.PlanID, first.ID)
	}
	empty := doJSON(t, server.router, http.MethodPost, "/v1/plans/leases", `{"leased_by":{"kind":"worker","id":"planner-worker-1"}}`)
	if empty.code != http.StatusNoContent {
		t.Fatalf("after skipped states status = %d, want 204: %s", empty.code, empty.body)
	}
}

func TestPostPlanProposalsStoresProposalAndDoesNotCreateTasks(t *testing.T) {
	server := testServer(t)
	approved := createApprovedContract(t, server)
	plan := createPlan(t, server, approved.ContractID)
	lease := acquireLease(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/plans/"+string(plan.ID)+"/proposals", validProposalJSON(string(approved.ID), lease))
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	assertNoHiddenContext(t, response.body)

	var proposal spine.WorkItemPlanProposal
	decodeJSON(t, response.body, &proposal)
	if proposal.State != spine.WorkItemProposalStateSubmitted {
		t.Fatalf("proposal state = %q, want submitted", proposal.State)
	}
	if proposal.PlanID != plan.ID || proposal.ContractID != approved.ContractID || proposal.ApprovedContractID != approved.ID {
		t.Fatalf("proposal ids = %q/%q/%q, want plan/contract/approved ids", proposal.PlanID, proposal.ContractID, proposal.ApprovedContractID)
	}
	if got := len(proposal.ProposedTasks); got != 2 {
		t.Fatalf("proposed tasks = %d, want 2", got)
	}
	if proposal.ProposedTasks[1].OrderIndex == nil || *proposal.ProposedTasks[1].OrderIndex != 1 {
		t.Fatalf("second order_index = %#v, want server-filled 1", proposal.ProposedTasks[1].OrderIndex)
	}

	storedPlan, ok, err := server.workItemPlans.Get(context.Background(), plan.ID)
	if err != nil {
		t.Fatalf("plans.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("plan missing")
	}
	if storedPlan.State != spine.WorkItemPlanStateProposalSubmitted {
		t.Fatalf("plan state = %q, want proposal_submitted", storedPlan.State)
	}
	storedLease, ok, err := server.workItemLeases.Get(context.Background(), lease.ID)
	if err != nil {
		t.Fatalf("leases.Get() error = %v", err)
	}
	if !ok || storedLease.State != spine.WorkItemPlanLeaseStateCompleted {
		t.Fatalf("lease state = %q ok=%v, want completed true", storedLease.State, ok)
	}
	if _, ok, err := server.workItems.GetByApprovedContractID(context.Background(), approved.ID); err != nil {
		t.Fatalf("workItems.GetByApprovedContractID() error = %v", err)
	} else if ok {
		t.Fatal("proposal submission materialized a WorkItem")
	}
}

func TestPostPlanProposalsRejectsInvalidAndDuplicateProposal(t *testing.T) {
	t.Run("missing lease proof", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		plan := createPlan(t, server, approved.ContractID)
		response := doJSON(t, server.router, http.MethodPost, "/v1/plans/"+string(plan.ID)+"/proposals", `{"submitted_by":{"kind":"worker","id":"planner-1"},"proposed_tasks":[{"title":"t","summary":"s","scope":["x"],"acceptance_refs":["a"],"proof_expectation_refs":["p"]}]}`)
		assertErrorCode(t, response, http.StatusBadRequest, "validation_failed")
		lease := acquireLease(t, server)
		response = doJSON(t, server.router, http.MethodPost, "/v1/plans/"+string(plan.ID)+"/proposals", proposalJSONWithLeaseValues(string(approved.ID), string(lease.ID), "bad-token"))
		assertErrorCode(t, response, http.StatusConflict, "invalid_lease")
	})

	t.Run("lease for wrong plan", func(t *testing.T) {
		server := testServer(t)
		first := createApprovedContract(t, server)
		second := createApprovedContract(t, server)
		planOne := createPlan(t, server, first.ContractID)
		planTwo := createPlan(t, server, second.ContractID)
		lease := acquireLease(t, server)
		if lease.PlanID != planOne.ID {
			t.Fatalf("lease plan = %q, want %q", lease.PlanID, planOne.ID)
		}
		_ = acquireLease(t, server)
		response := doJSON(t, server.router, http.MethodPost, "/v1/plans/"+string(planTwo.ID)+"/proposals", validProposalJSON(string(second.ID), lease))
		assertErrorCode(t, response, http.StatusConflict, "invalid_lease")
	})

	t.Run("expired lease proof", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		plan := createPlan(t, server, approved.ContractID)
		lease := acquireLease(t, server)
		stored := server.workItemLeases.leases[lease.ID]
		stored.ExpiresAt = testTime().Add(-time.Minute)
		server.workItemLeases.leases[lease.ID] = stored
		response := doJSON(t, server.router, http.MethodPost, "/v1/plans/"+string(plan.ID)+"/proposals", validProposalJSON(string(approved.ID), lease))
		assertErrorCode(t, response, http.StatusConflict, "lease_expired")
	})

	t.Run("missing proposed tasks", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		plan := createPlan(t, server, approved.ContractID)
		lease := acquireLease(t, server)
		response := doJSON(t, server.router, http.MethodPost, "/v1/plans/"+string(plan.ID)+"/proposals", proposalJSONWithTasks(string(approved.ID), lease, `[]`))
		assertErrorCode(t, response, http.StatusBadRequest, "validation_failed")
	})

	t.Run("invalid proposed task fields", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		plan := createPlan(t, server, approved.ContractID)
		lease := acquireLease(t, server)
		response := doJSON(t, server.router, http.MethodPost, "/v1/plans/"+string(plan.ID)+"/proposals", proposalJSONWithTasks(string(approved.ID), lease, `[{"title":"","summary":"s","scope":["x"],"acceptance_refs":["a"],"proof_expectation_refs":["p"]}]`))
		assertErrorCode(t, response, http.StatusBadRequest, "validation_failed")
	})

	t.Run("duplicate proposal", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		plan := createPlan(t, server, approved.ContractID)
		submitProposal(t, server, plan.ID, string(approved.ID))
		response := doJSON(t, server.router, http.MethodPost, "/v1/plans/"+string(plan.ID)+"/proposals", proposalJSONWithLeaseValues(string(approved.ID), "lease-unknown", "token"))
		assertErrorCode(t, response, http.StatusConflict, "already_proposed")
	})
}

func TestGetProposalReturnsProposalAndUnknownReturnsNotFound(t *testing.T) {
	server := testServer(t)
	approved := createApprovedContract(t, server)
	plan := createPlan(t, server, approved.ContractID)
	proposal := submitProposal(t, server, plan.ID, string(approved.ID))

	response := doJSON(t, server.router, http.MethodGet, "/v1/proposals/"+string(proposal.ID), "")
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	var got spine.WorkItemPlanProposal
	decodeJSON(t, response.body, &got)
	if got.ID != proposal.ID {
		t.Fatalf("id = %q, want %q", got.ID, proposal.ID)
	}

	missing := doJSON(t, server.router, http.MethodGet, "/v1/proposals/missing", "")
	assertErrorCode(t, missing, http.StatusNotFound, "not_found")
}

func TestPostProposalAcceptanceCreatesDurableWorkItems(t *testing.T) {
	server := testServer(t)
	approved := createApprovedContract(t, server)
	plan := createPlan(t, server, approved.ContractID)
	proposal := submitProposal(t, server, plan.ID, string(approved.ID))

	response := doJSON(t, server.router, http.MethodPost, "/v1/proposals/"+string(proposal.ID)+"/acceptance", `{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","accepted_by":{"kind":"user","id":"spoofed-acceptor"}}`)
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	assertNoForbiddenWorkItemSideEffects(t, server.events.Events())

	var accepted spine.WorkItemPlanAcceptanceResult
	decodeJSON(t, response.body, &accepted)
	if accepted.State != spine.WorkItemProposalStateAccepted {
		t.Fatalf("acceptance state = %q, want accepted", accepted.State)
	}
	if got := len(accepted.CreatedTaskIDs); got != 2 {
		t.Fatalf("created_task_ids = %d, want 2", got)
	}
	if accepted.AcceptedBy.Kind != "user" || accepted.AcceptedBy.ID != "018f0000-0000-7000-8000-000000000001" {
		t.Fatalf("accepted_by = %#v, want authenticated user actor", accepted.AcceptedBy)
	}

	for _, taskID := range accepted.CreatedTaskIDs {
		stored, ok, err := server.workItems.Get(context.Background(), taskID)
		if err != nil {
			t.Fatalf("workItems.Get(%s) error = %v", taskID, err)
		}
		if !ok {
			t.Fatalf("task %s not stored", taskID)
		}
		if stored.Status != spine.WorkItemStatusPlanned || stored.PlanID != plan.ID || stored.ProposalID != proposal.ID {
			t.Fatalf("stored task state/trace = %q/%q/%q, want planned/%q/%q", stored.Status, stored.PlanID, stored.ProposalID, plan.ID, proposal.ID)
		}
		if !hasSourceRef(stored.SourceRefs, workitem.SourceRefKindApprovedContract, string(approved.ID)) {
			t.Fatalf("source_refs = %#v, want approved_contract ref", stored.SourceRefs)
		}
		if !hasSourceRef(stored.SourceRefs, workitemplan.SourceRefKindProposal, string(proposal.ID)) {
			t.Fatalf("source_refs = %#v, want proposal ref", stored.SourceRefs)
		}
	}

	getResponse := doJSON(t, server.router, http.MethodGet, "/v1/tasks/"+string(accepted.CreatedTaskIDs[0]), "")
	if getResponse.code != http.StatusOK {
		t.Fatalf("GET task status = %d, want %d: %s", getResponse.code, http.StatusOK, getResponse.body)
	}
	var task spine.WorkItem
	decodeJSON(t, getResponse.body, &task)
	if task.ID != accepted.CreatedTaskIDs[0] {
		t.Fatalf("task id = %q, want %q", task.ID, accepted.CreatedTaskIDs[0])
	}

	storedPlan, _, _ := server.workItemPlans.Get(context.Background(), plan.ID)
	if storedPlan.State != spine.WorkItemPlanStateAccepted {
		t.Fatalf("plan state = %q, want accepted", storedPlan.State)
	}
	storedProposal, _, _ := server.workItemProposals.Get(context.Background(), proposal.ID)
	if storedProposal.State != spine.WorkItemProposalStateAccepted || storedProposal.AcceptedBy == nil {
		t.Fatalf("proposal accepted state = %q/%#v, want accepted actor", storedProposal.State, storedProposal.AcceptedBy)
	}
	if storedProposal.AcceptedBy.ID != "018f0000-0000-7000-8000-000000000001" {
		t.Fatalf("stored accepted_by = %#v, want authenticated user actor", storedProposal.AcceptedBy)
	}
	if got := countEventType(server.events.Events(), workitem.EventTypeWorkItemCreated); got != 2 {
		t.Fatalf("work_item.created events = %d, want 2", got)
	}
}

func TestPostProposalAcceptanceRejectsDuplicateAndInvalidContext(t *testing.T) {
	t.Run("duplicate acceptance", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		plan := createPlan(t, server, approved.ContractID)
		proposal := submitProposal(t, server, plan.ID, string(approved.ID))
		acceptProposal(t, server, proposal.ID)
		beforeWorkItemCount := len(server.workItems.items)
		beforeWorkItemEvents := countEventType(server.events.Events(), workitem.EventTypeWorkItemCreated)

		response := doJSON(t, server.router, http.MethodPost, "/v1/proposals/"+string(proposal.ID)+"/acceptance", `{}`)
		assertErrorCode(t, response, http.StatusConflict, "already_accepted")
		if len(server.workItems.items) != beforeWorkItemCount {
			t.Fatalf("work items = %d, want %d after duplicate acceptance", len(server.workItems.items), beforeWorkItemCount)
		}
		if got := countEventType(server.events.Events(), workitem.EventTypeWorkItemCreated); got != beforeWorkItemEvents {
			t.Fatalf("work_item.created events = %d, want %d after duplicate acceptance", got, beforeWorkItemEvents)
		}
	})

	t.Run("unauthenticated rejects before mutation", func(t *testing.T) {
		server := testServerWithContinuationAuth(t, fakeHTTPAuthService{meErr: auth.ErrInvalidToken})
		response := doJSON(t, server.router, http.MethodPost, "/v1/proposals/proposal-1/acceptance", `{}`)
		assertErrorCode(t, response, http.StatusUnauthorized, "unauthorized")
		if len(server.workItems.items) != 0 {
			t.Fatalf("work items = %d, want 0 after auth failure", len(server.workItems.items))
		}
	})

	t.Run("project expectation mismatch", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		plan := createPlan(t, server, approved.ContractID)
		proposal := submitProposal(t, server, plan.ID, string(approved.ID))

		response := doJSON(t, server.router, http.MethodPost, "/v1/proposals/"+string(proposal.ID)+"/acceptance", `{"project_id":"project-mismatch"}`)
		assertErrorCode(t, response, http.StatusConflict, "project_context_mismatch")
		if len(server.workItems.items) != 0 {
			t.Fatalf("work items = %d, want 0 after project mismatch", len(server.workItems.items))
		}
	})

	t.Run("repo expectation mismatch", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		plan := createPlan(t, server, approved.ContractID)
		proposal := submitProposal(t, server, plan.ID, string(approved.ID))

		response := doJSON(t, server.router, http.MethodPost, "/v1/proposals/"+string(proposal.ID)+"/acceptance", `{"repo_binding_id":"repo-binding-mismatch"}`)
		assertErrorCode(t, response, http.StatusConflict, "project_context_mismatch")
		if len(server.workItems.items) != 0 {
			t.Fatalf("work items = %d, want 0 after repo mismatch", len(server.workItems.items))
		}
	})
}

func TestPostTaskCheckoutJobsCreatesInstructionWithoutRuntimeSideEffects(t *testing.T) {
	server := testServer(t)
	approved := createApprovedContract(t, server)
	plan := createPlan(t, server, approved.ContractID)
	proposal := submitProposal(t, server, plan.ID, string(approved.ID))
	accepted := acceptProposal(t, server, proposal.ID)
	taskID := accepted.CreatedTaskIDs[0]

	response := doJSON(t, server.router, http.MethodPost, "/v1/tasks/"+string(taskID)+"/checkout-jobs", `{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","requested_by":{"kind":"user","id":"spoofed-requester"}}`)
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	assertNoHiddenContext(t, response.body)
	for _, forbidden := range []string{"\"lease_token\"", "\"lease_token_hash\"", "\"run_id\"", "\"proof_id\""} {
		if strings.Contains(response.body, forbidden) {
			t.Fatalf("checkout job response exposes forbidden field %s: %s", forbidden, response.body)
		}
	}
	var job spine.CheckoutJob
	decodeJSON(t, response.body, &job)
	if job.TaskID != taskID || job.State != spine.CheckoutJobStateQueued {
		t.Fatalf("job = %#v, want queued job for task %s", job, taskID)
	}
	if job.RequestedBy.ID != "018f0000-0000-7000-8000-000000000001" {
		t.Fatalf("requested_by = %#v, want authenticated user actor", job.RequestedBy)
	}
	if job.Instruction.TaskID != taskID || job.Instruction.RepositoryFullName == "" || job.Instruction.RawSourceUploaded {
		t.Fatalf("instruction = %#v, want task-bound no-raw-source instruction", job.Instruction)
	}
	storedTask, ok, err := server.workItems.Get(context.Background(), taskID)
	if err != nil || !ok {
		t.Fatalf("workItems.Get() = %#v/%v/%v", storedTask, ok, err)
	}
	if storedTask.Status != spine.WorkItemStatusPlanned {
		t.Fatalf("task status = %q, want planned after checkout job creation", storedTask.Status)
	}
	assertNoForbiddenRuntimeSideEffects(t, server.events.Events())

	second := doJSON(t, server.router, http.MethodPost, "/v1/tasks/"+string(taskID)+"/checkout-jobs", `{}`)
	if second.code != http.StatusOK {
		t.Fatalf("second status = %d, want %d: %s", second.code, http.StatusOK, second.body)
	}
	var existing spine.CheckoutJob
	decodeJSON(t, second.body, &existing)
	if existing.ID != job.ID || len(server.checkoutJobs.jobs) != 1 {
		t.Fatalf("existing job id/count = %q/%d, want %q/1", existing.ID, len(server.checkoutJobs.jobs), job.ID)
	}
}

func TestCheckoutJobLeaseAndReceiptRecordWorkspaceOnly(t *testing.T) {
	server := testServer(t)
	approved := createApprovedContract(t, server)
	plan := createPlan(t, server, approved.ContractID)
	proposal := submitProposal(t, server, plan.ID, string(approved.ID))
	accepted := acceptProposal(t, server, proposal.ID)
	taskID := accepted.CreatedTaskIDs[0]
	jobResponse := doJSON(t, server.router, http.MethodPost, "/v1/tasks/"+string(taskID)+"/checkout-jobs", `{}`)
	if jobResponse.code != http.StatusCreated {
		t.Fatalf("create checkout job status = %d, want %d: %s", jobResponse.code, http.StatusCreated, jobResponse.body)
	}
	var job spine.CheckoutJob
	decodeJSON(t, jobResponse.body, &job)

	leaseBody := fmt.Sprintf(`{"project_id":%q,"repo_binding_id":%q,"runner_id":"runner-1"}`, "018f0000-0000-7000-8000-000000000003", job.RepoBindingID)
	leaseResponse := doJSON(t, server.router, http.MethodPost, "/v1/checkout-jobs/leases", leaseBody)
	if leaseResponse.code != http.StatusCreated {
		t.Fatalf("lease status = %d, want %d: %s", leaseResponse.code, http.StatusCreated, leaseResponse.body)
	}
	if strings.Contains(leaseResponse.body, "lease_token_hash") {
		t.Fatalf("lease response exposes token hash: %s", leaseResponse.body)
	}
	var lease spine.CheckoutJobLeaseCreated
	decodeJSON(t, leaseResponse.body, &lease)
	if lease.JobID != job.ID || lease.TaskID != taskID || lease.LeaseToken == "" {
		t.Fatalf("lease = %#v, want lease for checkout job %s with one-time token", lease, job.ID)
	}

	receiptBody := fmt.Sprintf(`{"lease_token":%q,"runner_id":"runner-1","workspace_ref":"mounted:/workspace/goalrail","commit_sha":"abc123","baseline_id":"baseline-1","overlay_id":"overlay-1","dirty":false,"partial":false,"raw_source_uploaded":false}`, lease.LeaseToken)
	receiptResponse := doJSON(t, server.router, http.MethodPost, "/v1/checkout-jobs/"+string(job.ID)+"/receipts", receiptBody)
	if receiptResponse.code != http.StatusCreated {
		t.Fatalf("receipt status = %d, want %d: %s", receiptResponse.code, http.StatusCreated, receiptResponse.body)
	}
	assertNoHiddenContext(t, receiptResponse.body)
	var receipt spine.CheckoutReceipt
	decodeJSON(t, receiptResponse.body, &receipt)
	if receipt.JobID != job.ID || receipt.TaskID != taskID || receipt.RawSourceUploaded {
		t.Fatalf("receipt = %#v, want job/task receipt without raw source", receipt)
	}
	if len(server.checkoutReceipts.receipts) != 1 {
		t.Fatalf("checkout receipts = %d, want 1", len(server.checkoutReceipts.receipts))
	}
	storedJob := server.checkoutJobs.jobs[job.ID]
	if storedJob.State != spine.CheckoutJobStateReceiptSubmitted {
		t.Fatalf("checkout job state = %q, want receipt_submitted", storedJob.State)
	}
	storedTask, _, _ := server.workItems.Get(context.Background(), taskID)
	if storedTask.Status != spine.WorkItemStatusPlanned {
		t.Fatalf("task status = %q, want planned after checkout receipt", storedTask.Status)
	}
	assertNoForbiddenRuntimeSideEffects(t, server.events.Events())

	duplicate := doJSON(t, server.router, http.MethodPost, "/v1/checkout-jobs/"+string(job.ID)+"/receipts", receiptBody)
	assertErrorCode(t, duplicate, http.StatusConflict, "already_receipted")
	if len(server.checkoutReceipts.receipts) != 1 {
		t.Fatalf("checkout receipts = %d, want no duplicate receipt", len(server.checkoutReceipts.receipts))
	}
}

func TestPostTaskExecutionJobsCreatesQueuedJobWithoutRuntimeSideEffects(t *testing.T) {
	server := testServer(t)
	approved := createApprovedContract(t, server)
	plan := createPlan(t, server, approved.ContractID)
	proposal := submitProposal(t, server, plan.ID, string(approved.ID))
	accepted := acceptProposal(t, server, proposal.ID)
	taskID := accepted.CreatedTaskIDs[0]
	_, receipt := createCheckoutReceipt(t, server, taskID)

	body := fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","checkout_receipt_id":%q,"requested_by":{"kind":"user","id":"spoofed-requester"}}`, receipt.ID)
	response := doJSON(t, server.router, http.MethodPost, "/v1/tasks/"+string(taskID)+"/execution-jobs", body)
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	assertNoHiddenContext(t, response.body)
	for _, forbidden := range []string{"\"lease_token\"", "\"lease_token_hash\"", "\"run_id\"", "\"execution_receipt_id\"", "\"gate_decision_id\"", "\"proof_id\""} {
		if strings.Contains(response.body, forbidden) {
			t.Fatalf("execution job response exposes forbidden field %s: %s", forbidden, response.body)
		}
	}
	var job spine.ExecutionJob
	decodeJSON(t, response.body, &job)
	if job.TaskID != taskID || job.CheckoutReceiptID != receipt.ID || job.State != spine.ExecutionJobStateQueued {
		t.Fatalf("execution job = %#v, want queued job for task/receipt", job)
	}
	if job.RequestedBy.ID != "018f0000-0000-7000-8000-000000000001" {
		t.Fatalf("requested_by = %#v, want authenticated user actor", job.RequestedBy)
	}
	storedTask, ok, err := server.workItems.Get(context.Background(), taskID)
	if err != nil || !ok {
		t.Fatalf("workItems.Get() = %#v/%v/%v", storedTask, ok, err)
	}
	if storedTask.Status != spine.WorkItemStatusPlanned {
		t.Fatalf("task status = %q, want planned after execution job preparation", storedTask.Status)
	}
	assertNoForbiddenRuntimeSideEffects(t, server.events.Events())
	if got := countEventType(server.events.Events(), execution.EventTypeExecutionJobCreated); got != 1 {
		t.Fatalf("execution_job.created events = %d, want 1", got)
	}

	second := doJSON(t, server.router, http.MethodPost, "/v1/tasks/"+string(taskID)+"/execution-jobs", body)
	if second.code != http.StatusOK {
		t.Fatalf("second status = %d, want %d: %s", second.code, http.StatusOK, second.body)
	}
	var existing spine.ExecutionJob
	decodeJSON(t, second.body, &existing)
	if existing.ID != job.ID || len(server.executionJobs.jobs) != 1 {
		t.Fatalf("existing execution job id/count = %q/%d, want %q/1", existing.ID, len(server.executionJobs.jobs), job.ID)
	}
	if got := countEventType(server.events.Events(), execution.EventTypeExecutionJobCreated); got != 1 {
		t.Fatalf("execution_job.created events = %d, want no duplicate event", got)
	}
}

func TestPostTaskExecutionJobsRejectsBoundaryFailures(t *testing.T) {
	t.Run("unauthenticated rejects before mutation", func(t *testing.T) {
		server := testServerWithContinuationAuth(t, fakeHTTPAuthService{meErr: auth.ErrInvalidToken})
		response := doJSON(t, server.router, http.MethodPost, "/v1/tasks/work-item-1/execution-jobs", `{"checkout_receipt_id":"checkout-receipt-1"}`)
		assertErrorCode(t, response, http.StatusUnauthorized, "unauthorized")
		if len(server.executionJobs.jobs) != 0 {
			t.Fatalf("execution jobs = %d, want 0 after auth failure", len(server.executionJobs.jobs))
		}
	})

	t.Run("project expectation mismatch rejects before mutation", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		plan := createPlan(t, server, approved.ContractID)
		proposal := submitProposal(t, server, plan.ID, string(approved.ID))
		accepted := acceptProposal(t, server, proposal.ID)
		taskID := accepted.CreatedTaskIDs[0]
		_, receipt := createCheckoutReceipt(t, server, taskID)

		body := fmt.Sprintf(`{"project_id":"project-mismatch","checkout_receipt_id":%q}`, receipt.ID)
		response := doJSON(t, server.router, http.MethodPost, "/v1/tasks/"+string(taskID)+"/execution-jobs", body)
		assertErrorCode(t, response, http.StatusConflict, "project_context_mismatch")
		if len(server.executionJobs.jobs) != 0 {
			t.Fatalf("execution jobs = %d, want 0 after project mismatch", len(server.executionJobs.jobs))
		}
	})

	t.Run("missing checkout receipt rejects before mutation", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		plan := createPlan(t, server, approved.ContractID)
		proposal := submitProposal(t, server, plan.ID, string(approved.ID))
		accepted := acceptProposal(t, server, proposal.ID)
		taskID := accepted.CreatedTaskIDs[0]

		response := doJSON(t, server.router, http.MethodPost, "/v1/tasks/"+string(taskID)+"/execution-jobs", `{"checkout_receipt_id":"checkout-receipt-missing"}`)
		assertErrorCode(t, response, http.StatusNotFound, "not_found")
		if len(server.executionJobs.jobs) != 0 {
			t.Fatalf("execution jobs = %d, want 0 after missing receipt", len(server.executionJobs.jobs))
		}
	})

	t.Run("receipt for different task rejects before mutation", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		plan := createPlan(t, server, approved.ContractID)
		proposal := submitProposal(t, server, plan.ID, string(approved.ID))
		accepted := acceptProposal(t, server, proposal.ID)
		taskID := accepted.CreatedTaskIDs[0]
		_, receipt := createCheckoutReceipt(t, server, taskID)
		otherTask := server.workItems.items[taskID]
		otherTask.ID = "work-item-other"
		server.workItems.items[otherTask.ID] = otherTask

		body := fmt.Sprintf(`{"checkout_receipt_id":%q}`, receipt.ID)
		response := doJSON(t, server.router, http.MethodPost, "/v1/tasks/"+string(otherTask.ID)+"/execution-jobs", body)
		assertErrorCode(t, response, http.StatusConflict, "checkout_receipt_mismatch")
		if len(server.executionJobs.jobs) != 0 {
			t.Fatalf("execution jobs = %d, want 0 after task mismatch", len(server.executionJobs.jobs))
		}
	})

	t.Run("raw source receipt rejects before mutation", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		plan := createPlan(t, server, approved.ContractID)
		proposal := submitProposal(t, server, plan.ID, string(approved.ID))
		accepted := acceptProposal(t, server, proposal.ID)
		taskID := accepted.CreatedTaskIDs[0]
		_, receipt := createCheckoutReceipt(t, server, taskID)
		receipt.RawSourceUploaded = true
		server.checkoutReceipts.receipts[receipt.ID] = receipt

		body := fmt.Sprintf(`{"checkout_receipt_id":%q}`, receipt.ID)
		response := doJSON(t, server.router, http.MethodPost, "/v1/tasks/"+string(taskID)+"/execution-jobs", body)
		assertErrorCode(t, response, http.StatusBadRequest, "validation_failed")
		if len(server.executionJobs.jobs) != 0 {
			t.Fatalf("execution jobs = %d, want 0 after raw source receipt", len(server.executionJobs.jobs))
		}
	})

	t.Run("checkout job not receipted rejects before mutation", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		plan := createPlan(t, server, approved.ContractID)
		proposal := submitProposal(t, server, plan.ID, string(approved.ID))
		accepted := acceptProposal(t, server, proposal.ID)
		taskID := accepted.CreatedTaskIDs[0]
		job := spine.CheckoutJob{
			ID:                 "checkout-job-unreceipted",
			OrganizationID:     approved.OrganizationID,
			ProjectID:          approved.ProjectID,
			TaskID:             taskID,
			ContractID:         approved.ContractID,
			ApprovedContractID: approved.ID,
			PlanID:             plan.ID,
			ProposalID:         proposal.ID,
			RepoBindingID:      approved.RepoBindingID,
			State:              spine.CheckoutJobStateQueued,
			CreatedAt:          testTime(),
			UpdatedAt:          testTime(),
		}
		server.checkoutJobs.jobs[job.ID] = job
		receipt := spine.CheckoutReceipt{
			ID:                "checkout-receipt-unreceipted",
			JobID:             job.ID,
			TaskID:            taskID,
			RepoBindingID:     approved.RepoBindingID,
			RunnerID:          "runner-1",
			WorkspaceRef:      "mounted:/workspace/goalrail",
			CommitSHA:         "abc123",
			RawSourceUploaded: false,
			CreatedAt:         testTime(),
		}
		server.checkoutReceipts.receipts[receipt.ID] = receipt

		body := fmt.Sprintf(`{"checkout_receipt_id":%q}`, receipt.ID)
		response := doJSON(t, server.router, http.MethodPost, "/v1/tasks/"+string(taskID)+"/execution-jobs", body)
		assertErrorCode(t, response, http.StatusConflict, "invalid_state")
		if len(server.executionJobs.jobs) != 0 {
			t.Fatalf("execution jobs = %d, want 0 after invalid checkout job state", len(server.executionJobs.jobs))
		}
	})
}

func TestExecutionRunnerRoutesLeaseAndStartRun(t *testing.T) {
	server := testServer(t)
	approved := createApprovedContract(t, server)
	plan := createPlan(t, server, approved.ContractID)
	proposal := submitProposal(t, server, plan.ID, string(approved.ID))
	accepted := acceptProposal(t, server, proposal.ID)
	taskID := accepted.CreatedTaskIDs[0]
	_, receipt := createCheckoutReceipt(t, server, taskID)
	job := createExecutionJob(t, server, taskID, receipt.ID)

	leaseResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/leases", fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q,"runner_id":"runner-1"}`, job.RepoBindingID))
	if leaseResponse.code != http.StatusCreated {
		t.Fatalf("execution lease status = %d, want %d: %s", leaseResponse.code, http.StatusCreated, leaseResponse.body)
	}
	for _, forbidden := range []string{"\"lease_token_hash\"", "\"execution_receipt_id\"", "\"gate_decision_id\"", "\"proof_id\""} {
		if strings.Contains(leaseResponse.body, forbidden) {
			t.Fatalf("execution lease response exposes forbidden field %s: %s", forbidden, leaseResponse.body)
		}
	}
	var lease spine.ExecutionJobLeaseCreated
	decodeJSON(t, leaseResponse.body, &lease)
	if lease.ID == "" || lease.ExecutionJobID != job.ID || lease.TaskID != taskID || lease.CheckoutReceiptID != receipt.ID || lease.RunnerID != "runner-1" || lease.LeaseToken == "" {
		t.Fatalf("execution lease = %#v, want scoped lease with one-time token", lease)
	}
	if lease.ExecutionJob.State != spine.ExecutionJobStateLeased {
		t.Fatalf("lease execution job state = %q, want leased", lease.ExecutionJob.State)
	}
	if len(server.runs.runs) != 0 {
		t.Fatalf("runs = %d, want no Run after lease acquisition", len(server.runs.runs))
	}
	storedJob := server.executionJobs.jobs[job.ID]
	if storedJob.State != spine.ExecutionJobStateLeased || storedJob.CurrentLeaseID == nil || *storedJob.CurrentLeaseID != lease.ID || storedJob.LeaseTokenHash == "" || storedJob.LeaseTokenHash == lease.LeaseToken {
		t.Fatalf("stored execution job lease fields = %#v, want hashed lease proof only", storedJob)
	}

	runResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(job.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-1"}`, lease.ID, lease.LeaseToken))
	if runResponse.code != http.StatusCreated {
		t.Fatalf("run start status = %d, want %d: %s", runResponse.code, http.StatusCreated, runResponse.body)
	}
	for _, forbidden := range []string{"\"lease_token\"", "\"lease_token_hash\"", "\"execution_receipt_id\"", "\"gate_decision_id\"", "\"proof_id\""} {
		if strings.Contains(runResponse.body, forbidden) {
			t.Fatalf("run response exposes forbidden field %s: %s", forbidden, runResponse.body)
		}
	}
	var run spine.Run
	decodeJSON(t, runResponse.body, &run)
	if run.ID == "" || run.ExecutionJobID != job.ID || run.ExecutionLeaseID != lease.ID || run.TaskID != taskID || run.CheckoutReceiptID != receipt.ID || run.RunnerID != "runner-1" || run.State != spine.RunStateStarted {
		t.Fatalf("run = %#v, want started run bound to execution lease", run)
	}
	if got := server.executionJobs.jobs[job.ID].State; got != spine.ExecutionJobStateRunStarted {
		t.Fatalf("execution job state = %q, want run_started", got)
	}
	storedTask, ok, err := server.workItems.Get(context.Background(), taskID)
	if err != nil || !ok {
		t.Fatalf("workItems.Get() = %#v/%v/%v", storedTask, ok, err)
	}
	if storedTask.Status != spine.WorkItemStatusPlanned {
		t.Fatalf("task status = %q, want planned after run start", storedTask.Status)
	}
	assertNoForbiddenPostRunSideEffects(t, server.events.Events())
	if got := countEventType(server.events.Events(), execution.EventTypeRunStarted); got != 1 {
		t.Fatalf("run.started events = %d, want 1", got)
	}

	secondRunResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(job.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-1"}`, lease.ID, lease.LeaseToken))
	if secondRunResponse.code != http.StatusOK {
		t.Fatalf("second run start status = %d, want %d: %s", secondRunResponse.code, http.StatusOK, secondRunResponse.body)
	}
	var existing spine.Run
	decodeJSON(t, secondRunResponse.body, &existing)
	if existing.ID != run.ID || len(server.runs.runs) != 1 {
		t.Fatalf("existing run id/count = %q/%d, want %q/1", existing.ID, len(server.runs.runs), run.ID)
	}

	wrongProofResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(job.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":"wrong-token","runner_id":"runner-1"}`, lease.ID))
	assertErrorCode(t, wrongProofResponse, http.StatusConflict, "invalid_lease")
	if len(server.runs.runs) != 1 {
		t.Fatalf("runs = %d, want no duplicate run after wrong repeated proof", len(server.runs.runs))
	}

	receiptResponse := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/receipts", executionReceiptBody(job.ID, lease.ID, lease.LeaseToken, "runner-1", "mounted:/workspace/goalrail", "abc123", false))
	if receiptResponse.code != http.StatusCreated {
		t.Fatalf("execution receipt status = %d, want %d: %s", receiptResponse.code, http.StatusCreated, receiptResponse.body)
	}
	for _, forbidden := range []string{"\"lease_token\"", "\"lease_token_hash\"", "\"gate_decision_id\"", "\"proof_id\""} {
		if strings.Contains(receiptResponse.body, forbidden) {
			t.Fatalf("execution receipt response exposes forbidden field %s: %s", forbidden, receiptResponse.body)
		}
	}
	var receiptResult spine.ExecutionReceipt
	decodeJSON(t, receiptResponse.body, &receiptResult)
	if receiptResult.ID == "" || receiptResult.RunID != run.ID || receiptResult.ExecutionJobID != job.ID || receiptResult.ExecutionLeaseID != lease.ID || receiptResult.TaskID != taskID || receiptResult.CheckoutReceiptID != receipt.ID || receiptResult.RunnerID != "runner-1" {
		t.Fatalf("execution receipt = %#v, want receipt bound to run/job/lease/task", receiptResult)
	}
	if receiptResult.ExecutionMode != spine.ExecutionReceiptModeNoCommand || receiptResult.ProcessStatus != spine.ExecutionReceiptStatusNotExecuted || receiptResult.ExitCode != nil || receiptResult.RawSourceUploaded {
		t.Fatalf("execution receipt mode/status = %#v, want no-command metadata-only receipt", receiptResult)
	}
	if len(receiptResult.ArtifactRefs) != 0 || len(receiptResult.ChangedPathsSummary) != 0 || strings.Contains(receiptResponse.body, `"artifact_refs":null`) || strings.Contains(receiptResponse.body, `"changed_paths_summary":null`) {
		t.Fatalf("execution receipt artifact/path claims = %#v/%#v body=%s, want empty JSON arrays", receiptResult.ArtifactRefs, receiptResult.ChangedPathsSummary, receiptResponse.body)
	}
	if receiptResult.NextAction.Kind != spine.ExecutionReceiptNextActionGateReview || receiptResult.NextAction.Available {
		t.Fatalf("execution receipt next_action = %#v, want unavailable gate_review", receiptResult.NextAction)
	}
	if got := server.runs.runs[run.ID].State; got != spine.RunStateReceiptSubmitted {
		t.Fatalf("run state = %q, want receipt_submitted", got)
	}
	if got := server.executionJobs.jobs[job.ID].State; got != spine.ExecutionJobStateReceiptSubmitted {
		t.Fatalf("execution job state = %q, want receipt_submitted", got)
	}
	if len(server.executionReceipts.receipts) != 1 {
		t.Fatalf("execution receipts = %d, want 1", len(server.executionReceipts.receipts))
	}
	if got := countEventType(server.events.Events(), execution.EventTypeExecutionReceiptSubmitted); got != 1 {
		t.Fatalf("execution_receipt.submitted events = %d, want 1", got)
	}
	storedTaskAfterReceipt, ok, err := server.workItems.Get(context.Background(), taskID)
	if err != nil || !ok {
		t.Fatalf("workItems.Get() after receipt = %#v/%v/%v", storedTaskAfterReceipt, ok, err)
	}
	if storedTaskAfterReceipt.Status != spine.WorkItemStatusPlanned {
		t.Fatalf("task status = %q, want planned after execution receipt", storedTaskAfterReceipt.Status)
	}
	assertNoForbiddenPostReceiptSideEffects(t, server.events.Events())

	secondReceiptResponse := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/receipts", executionReceiptBody(job.ID, lease.ID, lease.LeaseToken, "runner-1", "mounted:/workspace/goalrail", "abc123", false))
	if secondReceiptResponse.code != http.StatusOK {
		t.Fatalf("second execution receipt status = %d, want %d: %s", secondReceiptResponse.code, http.StatusOK, secondReceiptResponse.body)
	}
	var existingReceipt spine.ExecutionReceipt
	decodeJSON(t, secondReceiptResponse.body, &existingReceipt)
	if existingReceipt.ID != receiptResult.ID || len(server.executionReceipts.receipts) != 1 {
		t.Fatalf("existing execution receipt id/count = %q/%d, want %q/1", existingReceipt.ID, len(server.executionReceipts.receipts), receiptResult.ID)
	}
}

func TestExecutionCommandPlanAndBuiltinDiagnosticReceipt(t *testing.T) {
	server := testServer(t)
	job, lease := createLeasedExecutionJob(t, server)
	runResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(job.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-1"}`, lease.ID, lease.LeaseToken))
	if runResponse.code != http.StatusCreated {
		t.Fatalf("run start status = %d, want %d: %s", runResponse.code, http.StatusCreated, runResponse.body)
	}
	var run spine.Run
	decodeJSON(t, runResponse.body, &run)

	planResponse := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/command-plans", fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q}`, job.RepoBindingID))
	if planResponse.code != http.StatusCreated {
		t.Fatalf("command plan status = %d, want %d: %s", planResponse.code, http.StatusCreated, planResponse.body)
	}
	for _, forbidden := range []string{"\"lease_token\"", "\"lease_token_hash\"", "\"gate_decision_id\"", "\"proof_id\"", "\"bash -lc\""} {
		if strings.Contains(planResponse.body, forbidden) {
			t.Fatalf("command plan response exposes forbidden field %s: %s", forbidden, planResponse.body)
		}
	}
	var plan spine.ExecutionCommandPlan
	decodeJSON(t, planResponse.body, &plan)
	if plan.ID == "" || plan.RunID != run.ID || plan.ExecutionJobID != job.ID || plan.TaskID != run.TaskID || plan.CheckoutReceiptID != run.CheckoutReceiptID || plan.RepoBindingID != job.RepoBindingID {
		t.Fatalf("command plan = %#v, want run/job/task/receipt scoped plan", plan)
	}
	if plan.CommandKind != spine.ExecutionCommandKindBuiltinDiagnostic || plan.Action != spine.ExecutionCommandActionWorkspaceStatus || plan.ShellAllowed || len(plan.Argv) != 0 || plan.WorkingDirectory != "." || len(plan.PathScope) != 1 || plan.PathScope[0] != "." || plan.RawSourceUploadAllowed {
		t.Fatalf("command plan policy = %#v, want fixed builtin diagnostic without shell/argv/raw source", plan)
	}
	if plan.TimeoutSeconds != 30 || plan.MaxStdoutBytes != 0 || plan.MaxStderrBytes != 0 || len(plan.AllowedArtifactKinds) != 0 || plan.State != spine.ExecutionCommandPlanStatePlanned {
		t.Fatalf("command plan limits/state = %#v, want diagnostic metadata-only policy", plan)
	}
	if len(server.commandPlans.plans) != 1 {
		t.Fatalf("command plans = %d, want 1", len(server.commandPlans.plans))
	}
	if len(server.executionReceipts.receipts) != 0 {
		t.Fatalf("execution receipts = %d, want 0 after command plan creation", len(server.executionReceipts.receipts))
	}
	if got := server.runs.runs[run.ID].State; got != spine.RunStateStarted {
		t.Fatalf("run state = %q, want started after command plan creation", got)
	}
	if got := countEventType(server.events.Events(), execution.EventTypeExecutionCommandPlanCreated); got != 1 {
		t.Fatalf("execution_command_plan.created events = %d, want 1", got)
	}

	secondPlanResponse := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/command-plans", fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q,"command_kind":"builtin_diagnostic","action":"workspace_status"}`, job.RepoBindingID))
	if secondPlanResponse.code != http.StatusOK {
		t.Fatalf("second command plan status = %d, want %d: %s", secondPlanResponse.code, http.StatusOK, secondPlanResponse.body)
	}
	var existingPlan spine.ExecutionCommandPlan
	decodeJSON(t, secondPlanResponse.body, &existingPlan)
	if existingPlan.ID != plan.ID || len(server.commandPlans.plans) != 1 {
		t.Fatalf("existing command plan id/count = %q/%d, want %q/1", existingPlan.ID, len(server.commandPlans.plans), plan.ID)
	}

	receiptResponse := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/receipts", builtinDiagnosticReceiptBody(job.ID, lease.ID, lease.LeaseToken, "runner-1", plan.ID))
	if receiptResponse.code != http.StatusCreated {
		t.Fatalf("builtin diagnostic receipt status = %d, want %d: %s", receiptResponse.code, http.StatusCreated, receiptResponse.body)
	}
	for _, forbidden := range []string{"\"lease_token\"", "\"lease_token_hash\"", "\"gate_decision_id\"", "\"proof_id\""} {
		if strings.Contains(receiptResponse.body, forbidden) {
			t.Fatalf("builtin diagnostic receipt response exposes forbidden field %s: %s", forbidden, receiptResponse.body)
		}
	}
	var receipt spine.ExecutionReceipt
	decodeJSON(t, receiptResponse.body, &receipt)
	if receipt.CommandPlanID == nil || *receipt.CommandPlanID != plan.ID || receipt.ExecutionMode != spine.ExecutionReceiptModeBuiltinDiagnostic || receipt.CommandKind != spine.ExecutionCommandKindBuiltinDiagnostic || receipt.Action != spine.ExecutionCommandActionWorkspaceStatus {
		t.Fatalf("builtin diagnostic receipt command metadata = %#v, want fixed command plan metadata", receipt)
	}
	if receipt.ProcessStatus != spine.ExecutionReceiptStatusMetadataOnly || receipt.ExitCode != nil || receipt.RawSourceUploaded {
		t.Fatalf("builtin diagnostic receipt status = %#v, want metadata_only without exit/raw source", receipt)
	}
	if len(receipt.ArtifactRefs) != 0 || len(receipt.ChangedPathsSummary) != 0 || strings.Contains(receiptResponse.body, `"artifact_refs":null`) || strings.Contains(receiptResponse.body, `"changed_paths_summary":null`) {
		t.Fatalf("builtin diagnostic artifact/path claims = %#v/%#v body=%s, want empty arrays", receipt.ArtifactRefs, receipt.ChangedPathsSummary, receiptResponse.body)
	}
	if receipt.RunnerStartedAt == nil || receipt.RunnerFinishedAt == nil {
		t.Fatalf("builtin diagnostic runner timing = %#v/%#v, want runner timestamps", receipt.RunnerStartedAt, receipt.RunnerFinishedAt)
	}
	if got := server.runs.runs[run.ID].State; got != spine.RunStateReceiptSubmitted {
		t.Fatalf("run state = %q, want receipt_submitted", got)
	}
	if got := server.executionJobs.jobs[job.ID].State; got != spine.ExecutionJobStateReceiptSubmitted {
		t.Fatalf("execution job state = %q, want receipt_submitted", got)
	}
	storedTask, ok, err := server.workItems.Get(context.Background(), run.TaskID)
	if err != nil || !ok {
		t.Fatalf("workItems.Get() after builtin diagnostic receipt = %#v/%v/%v", storedTask, ok, err)
	}
	if storedTask.Status != spine.WorkItemStatusPlanned {
		t.Fatalf("task status = %q, want planned after builtin diagnostic receipt", storedTask.Status)
	}
	assertNoForbiddenPostReceiptSideEffects(t, server.events.Events())
}

func TestProjectProbeCommandPlanAndReceipt(t *testing.T) {
	server := testServer(t)
	job, lease := createLeasedExecutionJob(t, server)
	runResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(job.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-1"}`, lease.ID, lease.LeaseToken))
	if runResponse.code != http.StatusCreated {
		t.Fatalf("run start status = %d, want %d: %s", runResponse.code, http.StatusCreated, runResponse.body)
	}
	var run spine.Run
	decodeJSON(t, runResponse.body, &run)

	planResponse := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/command-plans", fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q,"command_kind":"project_probe","action":"detect_declared_test_targets"}`, job.RepoBindingID))
	if planResponse.code != http.StatusCreated {
		t.Fatalf("project probe command plan status = %d, want %d: %s", planResponse.code, http.StatusCreated, planResponse.body)
	}
	for _, forbidden := range []string{"\"lease_token\"", "\"lease_token_hash\"", "\"gate_decision_id\"", "\"proof_id\"", "\"bash -lc\"", "\"sh -c\""} {
		if strings.Contains(planResponse.body, forbidden) {
			t.Fatalf("project probe command plan response exposes forbidden field %s: %s", forbidden, planResponse.body)
		}
	}
	var plan spine.ExecutionCommandPlan
	decodeJSON(t, planResponse.body, &plan)
	if plan.ID == "" || plan.RunID != run.ID || plan.ExecutionJobID != job.ID || plan.TaskID != run.TaskID || plan.CheckoutReceiptID != run.CheckoutReceiptID || plan.RepoBindingID != job.RepoBindingID {
		t.Fatalf("project probe command plan = %#v, want run/job/task/receipt scoped plan", plan)
	}
	if plan.CommandKind != spine.ExecutionCommandKindProjectProbe || plan.Action != spine.ExecutionCommandActionDetectTestTargets || plan.ShellAllowed || len(plan.Argv) != 0 || plan.WorkingDirectory != "." || len(plan.PathScope) != 1 || plan.PathScope[0] != "." || plan.RawSourceUploadAllowed {
		t.Fatalf("project probe command plan policy = %#v, want typed probe without shell/argv/raw source", plan)
	}
	if plan.TimeoutSeconds != 30 || plan.MaxStdoutBytes != 0 || plan.MaxStderrBytes != 0 || len(plan.AllowedArtifactKinds) != 0 || plan.State != spine.ExecutionCommandPlanStatePlanned {
		t.Fatalf("project probe command plan limits/state = %#v, want metadata-only probe policy", plan)
	}
	secondPlanResponse := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/command-plans", fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q,"command_kind":"project_probe","action":"detect_declared_test_targets"}`, job.RepoBindingID))
	if secondPlanResponse.code != http.StatusOK {
		t.Fatalf("second project probe command plan status = %d, want %d: %s", secondPlanResponse.code, http.StatusOK, secondPlanResponse.body)
	}
	var existingPlan spine.ExecutionCommandPlan
	decodeJSON(t, secondPlanResponse.body, &existingPlan)
	if existingPlan.ID != plan.ID || len(server.commandPlans.plans) != 1 {
		t.Fatalf("existing project probe command plan id/count = %q/%d, want %q/1", existingPlan.ID, len(server.commandPlans.plans), plan.ID)
	}

	receiptResponse := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/receipts", projectProbeReceiptBody(job.ID, lease.ID, lease.LeaseToken, "runner-1", plan.ID))
	if receiptResponse.code != http.StatusCreated {
		t.Fatalf("project probe receipt status = %d, want %d: %s", receiptResponse.code, http.StatusCreated, receiptResponse.body)
	}
	for _, forbidden := range []string{"\"lease_token\"", "\"lease_token_hash\"", "\"gate_decision_id\"", "\"proof_id\"", "\"bash -lc\"", "\"sh -c\"", "\"artifact_refs\":null", "\"changed_paths_summary\":null"} {
		if strings.Contains(receiptResponse.body, forbidden) {
			t.Fatalf("project probe receipt response exposes forbidden field %s: %s", forbidden, receiptResponse.body)
		}
	}
	var receipt spine.ExecutionReceipt
	decodeJSON(t, receiptResponse.body, &receipt)
	if receipt.CommandPlanID == nil || *receipt.CommandPlanID != plan.ID || receipt.ExecutionMode != spine.ExecutionReceiptModeProjectProbe || receipt.CommandKind != spine.ExecutionCommandKindProjectProbe || receipt.Action != spine.ExecutionCommandActionDetectTestTargets {
		t.Fatalf("project probe receipt command metadata = %#v, want fixed project probe metadata", receipt)
	}
	if receipt.ProcessStatus != spine.ExecutionReceiptStatusMetadataOnly || receipt.ExitCode != nil || receipt.RawSourceUploaded {
		t.Fatalf("project probe receipt status = %#v, want metadata_only without exit/raw source", receipt)
	}
	if len(receipt.ArtifactRefs) != 0 || len(receipt.ChangedPathsSummary) != 0 {
		t.Fatalf("project probe artifact/path claims = %#v/%#v, want empty arrays", receipt.ArtifactRefs, receipt.ChangedPathsSummary)
	}
	if receipt.ProjectProbeMetadata == nil || len(receipt.ProjectProbeMetadata.DetectedManifests) != 1 || len(receipt.ProjectProbeMetadata.PackageManagerCandidates) != 1 || len(receipt.ProjectProbeMetadata.DeclaredTestTargetCandidates) != 1 || len(receipt.ProjectProbeMetadata.PartialityReasons) != 1 {
		t.Fatalf("project probe metadata = %#v, want structured metadata evidence", receipt.ProjectProbeMetadata)
	}
	if receipt.NextAction.Kind != spine.ExecutionReceiptNextActionGateReview || receipt.NextAction.Available || receipt.NextAction.PlannedSlice != spine.ExecutionReceiptNextActionPlannedSlice {
		t.Fatalf("project probe next_action = %#v, want unavailable gate_review", receipt.NextAction)
	}
	if got := server.runs.runs[run.ID].State; got != spine.RunStateReceiptSubmitted {
		t.Fatalf("run state = %q, want receipt_submitted", got)
	}
	if got := server.executionJobs.jobs[job.ID].State; got != spine.ExecutionJobStateReceiptSubmitted {
		t.Fatalf("execution job state = %q, want receipt_submitted", got)
	}
	storedTask, ok, err := server.workItems.Get(context.Background(), run.TaskID)
	if err != nil || !ok {
		t.Fatalf("workItems.Get() after project probe receipt = %#v/%v/%v", storedTask, ok, err)
	}
	if storedTask.Status != spine.WorkItemStatusPlanned {
		t.Fatalf("task status = %q, want planned after project probe receipt", storedTask.Status)
	}
	assertNoForbiddenPostReceiptSideEffects(t, server.events.Events())
}

func TestProjectTestCommandPlanPreparation(t *testing.T) {
	const selectedTargetID = "package.json#package_json_script:test"

	t.Run("smoke pins planning-only boundary", func(t *testing.T) {
		server := testServer(t)
		probeJob, _, _, _, probeReceipt := createProjectProbeReceipt(t, server)
		testJob, testRun := createStartedExecutionRunForProjectTestPlan(t, server, probeJob)

		body := fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q,"command_kind":"project_test","action":"run_declared_test_target","project_probe_receipt_id":%q,"selected_target_id":%q}`, testJob.RepoBindingID, probeReceipt.ID, selectedTargetID)
		response := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(testRun.ID)+"/command-plans", body)
		if response.code != http.StatusCreated {
			t.Fatalf("project test command plan smoke status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
		}
		for _, forbidden := range []string{"\"lease_token\"", "\"lease_token_hash\"", "\"execution_receipt_id\"", "\"gate_decision_id\"", "\"proof_id\"", "\"bash -lc\"", "\"sh -c\"", "\"stdout_capture\"", "\"stderr_capture\"", "\"artifact_refs\"", "\"changed_paths_summary\"", "\"execution_mode\"", "\"process_status\"", "\"exit_code\""} {
			if strings.Contains(response.body, forbidden) {
				t.Fatalf("project test command plan smoke response exposes forbidden field %s: %s", forbidden, response.body)
			}
		}
		var plan spine.ExecutionCommandPlan
		decodeJSON(t, response.body, &plan)
		if plan.CommandKind != spine.ExecutionCommandKindProjectTest || plan.Action != spine.ExecutionCommandActionRunTestTarget {
			t.Fatalf("project test command plan kind/action = %s/%s, want typed project_test", plan.CommandKind, plan.Action)
		}
		if plan.SourceProjectProbeReceiptID == nil || *plan.SourceProjectProbeReceiptID != probeReceipt.ID || plan.SelectedTargetID != selectedTargetID {
			t.Fatalf("project test source receipt/target = %#v/%q, want %q/%q", plan.SourceProjectProbeReceiptID, plan.SelectedTargetID, probeReceipt.ID, selectedTargetID)
		}
		if plan.DeclaredTestTarget == nil || plan.DeclaredTestTarget.Name != "test" || plan.DeclaredTestTarget.SourcePath != "package.json" || plan.DeclaredTestTarget.SourceKind != "package_json_script" {
			t.Fatalf("project test declared target = %#v, want selected package.json test metadata", plan.DeclaredTestTarget)
		}
		storedPlan := server.commandPlans.plans[plan.ID]
		if storedPlan.DeclaredTestTarget == nil || *storedPlan.DeclaredTestTarget != *plan.DeclaredTestTarget || storedPlan.SourceProjectProbeReceiptID == nil || *storedPlan.SourceProjectProbeReceiptID != probeReceipt.ID {
			t.Fatalf("stored project test plan = %#v, want persisted source receipt and target metadata", storedPlan)
		}
		if plan.ShellAllowed || len(plan.Argv) != 0 || plan.WorkingDirectory != "." || len(plan.PathScope) != 1 || plan.PathScope[0] != "." || plan.RawSourceUploadAllowed {
			t.Fatalf("project test command plan smoke policy = %#v, want no shell/argv/raw source", plan)
		}
		if plan.TimeoutSeconds != 120 || plan.NetworkAllowed || plan.WorkspaceWriteAllowed || plan.ScratchWriteAllowed || plan.MaxStdoutBytes != 0 || plan.MaxStderrBytes != 0 || len(plan.AllowedArtifactKinds) != 0 || plan.ChangedPathsAllowed || plan.State != spine.ExecutionCommandPlanStatePlanned {
			t.Fatalf("project test command plan smoke limits/state = %#v, want planning-only fail-closed policy", plan)
		}
		if plan.NextAction == nil || plan.NextAction.Kind != spine.ExecutionCommandPlanNextActionRunnerProjectTestRequired || plan.NextAction.Available || plan.NextAction.PlannedSlice != spine.ExecutionCommandPlanNextActionProjectTestPlannedSlice {
			t.Fatalf("project test smoke next_action = %#v, want unavailable runner_project_test_required", plan.NextAction)
		}
		if len(server.executionReceipts.receipts) != 1 {
			t.Fatalf("execution receipts = %d, want only prior project_probe receipt after project_test plan smoke", len(server.executionReceipts.receipts))
		}
		for id, receipt := range server.executionReceipts.receipts {
			if receipt.ExecutionMode == spine.ExecutionCommandKindProjectTest || receipt.CommandKind == spine.ExecutionCommandKindProjectTest || receipt.Action == spine.ExecutionCommandActionRunTestTarget {
				t.Fatalf("execution receipt %s = %#v, want no project_test receipt in H2.6.1+ smoke", id, receipt)
			}
		}
		if got := server.runs.runs[testRun.ID].State; got != spine.RunStateStarted {
			t.Fatalf("project test run state = %q, want started after plan-only smoke", got)
		}
		if got := server.executionJobs.jobs[testJob.ID].State; got != spine.ExecutionJobStateRunStarted {
			t.Fatalf("project test job state = %q, want run_started after plan-only smoke", got)
		}
		storedTask, ok, err := server.workItems.Get(context.Background(), testRun.TaskID)
		if err != nil || !ok {
			t.Fatalf("workItems.Get() after project test plan smoke = %#v/%v/%v", storedTask, ok, err)
		}
		if storedTask.Status != spine.WorkItemStatusPlanned {
			t.Fatalf("task status = %q, want planned after project test plan smoke", storedTask.Status)
		}
		if got := countEventType(server.events.Events(), execution.EventTypeExecutionCommandPlanCreated); got != 2 {
			t.Fatalf("execution_command_plan.created events = %d, want project_probe + project_test plans only", got)
		}
		if got := countEventType(server.events.Events(), execution.EventTypeExecutionReceiptSubmitted); got != 1 {
			t.Fatalf("execution_receipt.submitted events = %d, want only prior project_probe receipt", got)
		}
		assertNoForbiddenPostReceiptSideEffects(t, server.events.Events())
	})

	t.Run("creates typed plan from project probe receipt without execution side effects", func(t *testing.T) {
		server := testServer(t)
		probeJob, _, _, _, probeReceipt := createProjectProbeReceipt(t, server)
		testJob, testRun := createStartedExecutionRunForProjectTestPlan(t, server, probeJob)

		body := fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q,"command_kind":"project_test","action":"run_declared_test_target","project_probe_receipt_id":%q,"selected_target_id":%q}`, testJob.RepoBindingID, probeReceipt.ID, selectedTargetID)
		response := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(testRun.ID)+"/command-plans", body)
		if response.code != http.StatusCreated {
			t.Fatalf("project test command plan status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
		}
		for _, forbidden := range []string{"\"lease_token\"", "\"lease_token_hash\"", "\"gate_decision_id\"", "\"proof_id\"", "\"bash -lc\"", "\"sh -c\"", "\"stdout_capture\"", "\"stderr_capture\"", "\"artifact_refs\""} {
			if strings.Contains(response.body, forbidden) {
				t.Fatalf("project test command plan response exposes forbidden field %s: %s", forbidden, response.body)
			}
		}
		var plan spine.ExecutionCommandPlan
		decodeJSON(t, response.body, &plan)
		if plan.ID == "" || plan.RunID != testRun.ID || plan.ExecutionJobID != testJob.ID || plan.TaskID != testRun.TaskID || plan.CheckoutReceiptID != testRun.CheckoutReceiptID || plan.RepoBindingID != testJob.RepoBindingID {
			t.Fatalf("project test command plan = %#v, want run/job/task/receipt scoped plan", plan)
		}
		if plan.CommandKind != spine.ExecutionCommandKindProjectTest || plan.Action != spine.ExecutionCommandActionRunTestTarget {
			t.Fatalf("project test command plan kind/action = %s/%s, want typed project_test", plan.CommandKind, plan.Action)
		}
		if plan.SourceProjectProbeReceiptID == nil || *plan.SourceProjectProbeReceiptID != probeReceipt.ID || plan.SelectedTargetID != selectedTargetID {
			t.Fatalf("project test source receipt/target = %#v/%q, want %q/%q", plan.SourceProjectProbeReceiptID, plan.SelectedTargetID, probeReceipt.ID, selectedTargetID)
		}
		if plan.DeclaredTestTarget == nil || plan.DeclaredTestTarget.Name != "test" || plan.DeclaredTestTarget.SourcePath != "package.json" || plan.DeclaredTestTarget.SourceKind != "package_json_script" {
			t.Fatalf("project test declared target = %#v, want selected package.json test metadata", plan.DeclaredTestTarget)
		}
		if plan.ShellAllowed || len(plan.Argv) != 0 || plan.WorkingDirectory != "." || len(plan.PathScope) != 1 || plan.PathScope[0] != "." || plan.RawSourceUploadAllowed {
			t.Fatalf("project test command plan policy = %#v, want no shell/argv/raw source", plan)
		}
		if plan.TimeoutSeconds != 120 || plan.NetworkAllowed || plan.WorkspaceWriteAllowed || plan.ScratchWriteAllowed || plan.MaxStdoutBytes != 0 || plan.MaxStderrBytes != 0 || len(plan.AllowedArtifactKinds) != 0 || plan.ChangedPathsAllowed || plan.State != spine.ExecutionCommandPlanStatePlanned {
			t.Fatalf("project test command plan limits/state = %#v, want planning-only policy", plan)
		}
		if plan.NextAction == nil || plan.NextAction.Kind != spine.ExecutionCommandPlanNextActionRunnerProjectTestRequired || plan.NextAction.Available || plan.NextAction.PlannedSlice != spine.ExecutionCommandPlanNextActionProjectTestPlannedSlice {
			t.Fatalf("project test next_action = %#v, want unavailable runner_project_test_required", plan.NextAction)
		}
		if len(server.executionReceipts.receipts) != 1 {
			t.Fatalf("execution receipts = %d, want only prior project_probe receipt after plan creation", len(server.executionReceipts.receipts))
		}
		if got := server.runs.runs[testRun.ID].State; got != spine.RunStateStarted {
			t.Fatalf("project test run state = %q, want started after plan preparation", got)
		}
		if got := server.executionJobs.jobs[testJob.ID].State; got != spine.ExecutionJobStateRunStarted {
			t.Fatalf("project test job state = %q, want run_started after plan preparation", got)
		}
		storedTask, ok, err := server.workItems.Get(context.Background(), testRun.TaskID)
		if err != nil || !ok {
			t.Fatalf("workItems.Get() after project test plan = %#v/%v/%v", storedTask, ok, err)
		}
		if storedTask.Status != spine.WorkItemStatusPlanned {
			t.Fatalf("task status = %q, want planned after project test plan", storedTask.Status)
		}

		secondResponse := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(testRun.ID)+"/command-plans", body)
		if secondResponse.code != http.StatusOK {
			t.Fatalf("second project test command plan status = %d, want %d: %s", secondResponse.code, http.StatusOK, secondResponse.body)
		}
		var existing spine.ExecutionCommandPlan
		decodeJSON(t, secondResponse.body, &existing)
		if existing.ID != plan.ID || len(server.commandPlans.plans) != 2 {
			t.Fatalf("existing project test command plan id/count = %q/%d, want %q/2", existing.ID, len(server.commandPlans.plans), plan.ID)
		}
		assertNoForbiddenPostReceiptSideEffects(t, server.events.Events())
	})

	t.Run("rejects omitted project probe receipt id", func(t *testing.T) {
		server := testServer(t)
		probeJob, _, _, _, _ := createProjectProbeReceipt(t, server)
		testJob, testRun := createStartedExecutionRunForProjectTestPlan(t, server, probeJob)

		body := fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q,"command_kind":"project_test","action":"run_declared_test_target","selected_target_id":%q}`, testJob.RepoBindingID, selectedTargetID)
		response := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(testRun.ID)+"/command-plans", body)
		assertErrorCode(t, response, http.StatusBadRequest, "validation_failed")
		if len(server.commandPlans.plans) != 1 {
			t.Fatalf("command plans = %d, want only prior project_probe plan after missing project_probe_receipt_id", len(server.commandPlans.plans))
		}
	})

	t.Run("rejects missing project probe receipt", func(t *testing.T) {
		server := testServer(t)
		job, lease := createLeasedExecutionJob(t, server)
		runResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(job.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-1"}`, lease.ID, lease.LeaseToken))
		if runResponse.code != http.StatusCreated {
			t.Fatalf("run start status = %d, want %d: %s", runResponse.code, http.StatusCreated, runResponse.body)
		}
		var run spine.Run
		decodeJSON(t, runResponse.body, &run)

		body := fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q,"command_kind":"project_test","action":"run_declared_test_target","project_probe_receipt_id":"018f0000-0000-7000-8004-999999999999","selected_target_id":%q}`, job.RepoBindingID, selectedTargetID)
		response := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/command-plans", body)
		assertErrorCode(t, response, http.StatusNotFound, "not_found")
		if len(server.commandPlans.plans) != 0 {
			t.Fatalf("command plans = %d, want 0 after missing project_probe receipt", len(server.commandPlans.plans))
		}
	})

	t.Run("rejects malformed project probe receipt id", func(t *testing.T) {
		server := testServer(t)
		job, lease := createLeasedExecutionJob(t, server)
		runResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(job.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-1"}`, lease.ID, lease.LeaseToken))
		if runResponse.code != http.StatusCreated {
			t.Fatalf("run start status = %d, want %d: %s", runResponse.code, http.StatusCreated, runResponse.body)
		}
		var run spine.Run
		decodeJSON(t, runResponse.body, &run)

		body := fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q,"command_kind":"project_test","action":"run_declared_test_target","project_probe_receipt_id":"not-a-uuid","selected_target_id":%q}`, job.RepoBindingID, selectedTargetID)
		response := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/command-plans", body)
		assertErrorCode(t, response, http.StatusBadRequest, "validation_failed")
		if len(server.commandPlans.plans) != 0 {
			t.Fatalf("command plans = %d, want 0 after malformed project_probe receipt id", len(server.commandPlans.plans))
		}
	})

	t.Run("rejects project probe receipt from different lineage", func(t *testing.T) {
		server := testServer(t)
		_, _, _, _, probeReceipt := createProjectProbeReceipt(t, server)
		otherJob, otherLease := createLeasedExecutionJob(t, server)
		runResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(otherJob.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-1"}`, otherLease.ID, otherLease.LeaseToken))
		if runResponse.code != http.StatusCreated {
			t.Fatalf("run start status = %d, want %d: %s", runResponse.code, http.StatusCreated, runResponse.body)
		}
		var run spine.Run
		decodeJSON(t, runResponse.body, &run)

		body := fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q,"command_kind":"project_test","action":"run_declared_test_target","project_probe_receipt_id":%q,"selected_target_id":%q}`, otherJob.RepoBindingID, probeReceipt.ID, selectedTargetID)
		response := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/command-plans", body)
		assertErrorCode(t, response, http.StatusConflict, "invalid_command_plan")
		if len(server.commandPlans.plans) != 1 {
			t.Fatalf("command plans = %d, want only prior project_probe plan after lineage mismatch", len(server.commandPlans.plans))
		}
	})

	t.Run("rejects unknown selected target", func(t *testing.T) {
		server := testServer(t)
		probeJob, _, _, _, probeReceipt := createProjectProbeReceipt(t, server)
		testJob, testRun := createStartedExecutionRunForProjectTestPlan(t, server, probeJob)

		body := fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q,"command_kind":"project_test","action":"run_declared_test_target","project_probe_receipt_id":%q,"selected_target_id":"package.json#package_json_script:test:integration"}`, testJob.RepoBindingID, probeReceipt.ID)
		response := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(testRun.ID)+"/command-plans", body)
		assertErrorCode(t, response, http.StatusBadRequest, "validation_failed")
		if len(server.commandPlans.plans) != 1 {
			t.Fatalf("command plans = %d, want only prior project_probe plan after unknown target", len(server.commandPlans.plans))
		}
	})

	t.Run("rejects unsupported target family", func(t *testing.T) {
		server := testServer(t)
		job, lease := createLeasedExecutionJob(t, server)
		runResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(job.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-1"}`, lease.ID, lease.LeaseToken))
		if runResponse.code != http.StatusCreated {
			t.Fatalf("run start status = %d, want %d: %s", runResponse.code, http.StatusCreated, runResponse.body)
		}
		var probeRun spine.Run
		decodeJSON(t, runResponse.body, &probeRun)
		planResponse := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(probeRun.ID)+"/command-plans", fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q,"command_kind":"project_probe","action":"detect_declared_test_targets"}`, job.RepoBindingID))
		if planResponse.code != http.StatusCreated {
			t.Fatalf("project probe command plan status = %d, want %d: %s", planResponse.code, http.StatusCreated, planResponse.body)
		}
		var probePlan spine.ExecutionCommandPlan
		decodeJSON(t, planResponse.body, &probePlan)
		receiptResponse := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(probeRun.ID)+"/receipts", projectProbeReceiptBodyWithTarget(job.ID, lease.ID, lease.LeaseToken, "runner-1", probePlan.ID, "go_package_tests", "go.mod", "go_module_manifest"))
		if receiptResponse.code != http.StatusCreated {
			t.Fatalf("project probe receipt status = %d, want %d: %s", receiptResponse.code, http.StatusCreated, receiptResponse.body)
		}
		var probeReceipt spine.ExecutionReceipt
		decodeJSON(t, receiptResponse.body, &probeReceipt)
		testJob, testRun := createStartedExecutionRunForProjectTestPlan(t, server, job)

		body := fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q,"command_kind":"project_test","action":"run_declared_test_target","project_probe_receipt_id":%q,"selected_target_id":"go.mod#go_module_manifest:go_package_tests"}`, testJob.RepoBindingID, probeReceipt.ID)
		response := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(testRun.ID)+"/command-plans", body)
		assertErrorCode(t, response, http.StatusBadRequest, "validation_failed")
		if len(server.commandPlans.plans) != 1 {
			t.Fatalf("command plans = %d, want only prior project_probe plan after unsupported target", len(server.commandPlans.plans))
		}
	})

	t.Run("rejects unsafe project test plan fields", func(t *testing.T) {
		for _, tt := range []struct {
			name string
			body string
		}{
			{name: "shell", body: `,"shell":true`},
			{name: "shell_allowed", body: `,"shell_allowed":true`},
			{name: "argv", body: `,"argv":["npm","test"]`},
			{name: "command", body: `,"command":"npm test"`},
			{name: "command_string", body: `,"command_string":"go test ./..."`},
			{name: "user_command", body: `,"user_command":"pytest"`},
			{name: "run_all_tests", body: `,"run_all_tests":true`},
			{name: "stdout_capture", body: `,"stdout_capture":{"mode":"inline"}`},
			{name: "stderr_capture", body: `,"stderr_capture":{"mode":"inline"}`},
			{name: "artifacts_allowed", body: `,"artifacts_allowed":true`},
			{name: "artifact_refs", body: `,"artifact_refs":["junit.xml"]`},
			{name: "allowed_artifact_kinds", body: `,"allowed_artifact_kinds":["junit"]`},
			{name: "changed_paths_allowed", body: `,"changed_paths_allowed":true`},
			{name: "changed_paths_summary", body: `,"changed_paths_summary":["package.json"]`},
			{name: "raw_source_upload", body: `,"raw_source_upload":true`},
			{name: "raw_source_uploaded", body: `,"raw_source_uploaded":true`},
			{name: "raw_source_upload_allowed", body: `,"raw_source_upload_allowed":true`},
			{name: "network_allowed", body: `,"network_allowed":true`},
			{name: "write_allowed", body: `,"write_allowed":true`},
			{name: "workspace_write_allowed", body: `,"workspace_write_allowed":true`},
			{name: "scratch_write_allowed", body: `,"scratch_write_allowed":true`},
		} {
			t.Run(tt.name, func(t *testing.T) {
				server := testServer(t)
				probeJob, _, _, _, probeReceipt := createProjectProbeReceipt(t, server)
				testJob, testRun := createStartedExecutionRunForProjectTestPlan(t, server, probeJob)

				body := fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q,"command_kind":"project_test","action":"run_declared_test_target","project_probe_receipt_id":%q,"selected_target_id":%q%s}`, testJob.RepoBindingID, probeReceipt.ID, selectedTargetID, tt.body)
				response := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(testRun.ID)+"/command-plans", body)
				assertErrorCode(t, response, http.StatusBadRequest, "validation_failed")
				if len(server.commandPlans.plans) != 1 {
					t.Fatalf("command plans = %d, want only prior project_probe plan after unsafe request", len(server.commandPlans.plans))
				}
			})
		}
	})
}

func TestProjectTestCommandPlanAndReceipt(t *testing.T) {
	t.Run("submits project test policy rejection receipt as evidence only", func(t *testing.T) {
		server := testServer(t)
		job, lease, run, plan := createProjectTestPlanFromSeededProbe(t, server)

		getResponse := doJSON(t, server.router, http.MethodGet, "/v1/runs/"+string(run.ID)+"/command-plans/project_test/run_declared_test_target", "")
		if getResponse.code != http.StatusOK {
			t.Fatalf("get project test command plan status = %d, want %d: %s", getResponse.code, http.StatusOK, getResponse.body)
		}
		var fetchedPlan spine.ExecutionCommandPlan
		decodeJSON(t, getResponse.body, &fetchedPlan)
		if fetchedPlan.ID != plan.ID || fetchedPlan.CommandKind != spine.ExecutionCommandKindProjectTest || fetchedPlan.Action != spine.ExecutionCommandActionRunTestTarget {
			t.Fatalf("fetched project test command plan = %#v, want existing project_test plan", fetchedPlan)
		}

		receiptResponse := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/receipts", projectTestReceiptBody(job.ID, lease.ID, lease.LeaseToken, "runner-1", plan.ID, spine.ExecutionReceiptStatusPolicyRejected, nil, false))
		if receiptResponse.code != http.StatusCreated {
			t.Fatalf("project test receipt status = %d, want %d: %s", receiptResponse.code, http.StatusCreated, receiptResponse.body)
		}
		for _, forbidden := range []string{"\"lease_token\"", "\"lease_token_hash\"", "\"gate_decision_id\"", "\"proof_id\"", "\"project_probe_metadata\"", "\"stdout\"", "\"stderr\""} {
			if strings.Contains(receiptResponse.body, forbidden) {
				t.Fatalf("project test receipt response exposes forbidden field %s: %s", forbidden, receiptResponse.body)
			}
		}
		var receipt spine.ExecutionReceipt
		decodeJSON(t, receiptResponse.body, &receipt)
		if receipt.CommandPlanID == nil || *receipt.CommandPlanID != plan.ID || receipt.ExecutionMode != spine.ExecutionReceiptModeProjectTest || receipt.CommandKind != spine.ExecutionCommandKindProjectTest || receipt.Action != spine.ExecutionCommandActionRunTestTarget {
			t.Fatalf("project test receipt command metadata = %#v, want fixed project_test metadata", receipt)
		}
		if receipt.ProcessStatus != spine.ExecutionReceiptStatusPolicyRejected || receipt.ExitCode != nil || receipt.RawSourceUploaded {
			t.Fatalf("project test receipt process evidence = %#v, want policy_rejected without exit_code/raw source", receipt)
		}
		if len(receipt.ArtifactRefs) != 0 || len(receipt.ChangedPathsSummary) != 0 || receipt.ProjectProbeMetadata != nil {
			t.Fatalf("project test evidence claims = artifacts=%#v changed=%#v probe=%#v, want process-only receipt", receipt.ArtifactRefs, receipt.ChangedPathsSummary, receipt.ProjectProbeMetadata)
		}
		if got := server.runs.runs[run.ID].State; got != spine.RunStateReceiptSubmitted {
			t.Fatalf("project test run state = %q, want receipt_submitted", got)
		}
		if got := server.executionJobs.jobs[job.ID].State; got != spine.ExecutionJobStateReceiptSubmitted {
			t.Fatalf("project test execution job state = %q, want receipt_submitted", got)
		}
		storedTask, ok, err := server.workItems.Get(context.Background(), run.TaskID)
		if err != nil || !ok {
			t.Fatalf("workItems.Get() after project test receipt = %#v/%v/%v", storedTask, ok, err)
		}
		if storedTask.Status != spine.WorkItemStatusPlanned {
			t.Fatalf("task status = %q, want planned after project test receipt", storedTask.Status)
		}
		assertNoForbiddenPostReceiptSideEffects(t, server.events.Events())
	})

	t.Run("rejects unsafe project test receipt claims", func(t *testing.T) {
		for _, tt := range []struct {
			name string
			body func(spine.ExecutionJob, spine.ExecutionJobLeaseCreated, spine.ExecutionCommandPlan) string
		}{
			{
				name: "exited status",
				body: func(job spine.ExecutionJob, lease spine.ExecutionJobLeaseCreated, plan spine.ExecutionCommandPlan) string {
					return projectTestReceiptBody(job.ID, lease.ID, lease.LeaseToken, "runner-1", plan.ID, spine.ExecutionReceiptStatusExited, intPtr(0), false)
				},
			},
			{
				name: "exit code for policy rejected",
				body: func(job spine.ExecutionJob, lease spine.ExecutionJobLeaseCreated, plan spine.ExecutionCommandPlan) string {
					return projectTestReceiptBody(job.ID, lease.ID, lease.LeaseToken, "runner-1", plan.ID, spine.ExecutionReceiptStatusPolicyRejected, intPtr(124), false)
				},
			},
			{
				name: "artifacts",
				body: func(job spine.ExecutionJob, lease spine.ExecutionJobLeaseCreated, plan spine.ExecutionCommandPlan) string {
					body := projectTestReceiptBody(job.ID, lease.ID, lease.LeaseToken, "runner-1", plan.ID, spine.ExecutionReceiptStatusExited, intPtr(1), false)
					return strings.Replace(body, `"artifact_refs":[],"changed_paths_summary":[]`, `"artifact_refs":["junit.xml"],"changed_paths_summary":[]`, 1)
				},
			},
			{
				name: "changed paths",
				body: func(job spine.ExecutionJob, lease spine.ExecutionJobLeaseCreated, plan spine.ExecutionCommandPlan) string {
					body := projectTestReceiptBody(job.ID, lease.ID, lease.LeaseToken, "runner-1", plan.ID, spine.ExecutionReceiptStatusExited, intPtr(1), false)
					return strings.Replace(body, `"artifact_refs":[],"changed_paths_summary":[]`, `"artifact_refs":[],"changed_paths_summary":["package.json"]`, 1)
				},
			},
			{
				name: "raw source upload",
				body: func(job spine.ExecutionJob, lease spine.ExecutionJobLeaseCreated, plan spine.ExecutionCommandPlan) string {
					return projectTestReceiptBody(job.ID, lease.ID, lease.LeaseToken, "runner-1", plan.ID, spine.ExecutionReceiptStatusExited, intPtr(1), true)
				},
			},
			{
				name: "project probe metadata",
				body: func(job spine.ExecutionJob, lease spine.ExecutionJobLeaseCreated, plan spine.ExecutionCommandPlan) string {
					body := projectTestReceiptBody(job.ID, lease.ID, lease.LeaseToken, "runner-1", plan.ID, spine.ExecutionReceiptStatusExited, intPtr(0), false)
					return strings.Replace(body, `"runner_finished_at":"2026-04-25T12:00:00Z"`, `"runner_finished_at":"2026-04-25T12:00:00Z","project_probe_metadata":{"partiality_reasons":["not allowed"]}`, 1)
				},
			},
			{
				name: "wrong command plan",
				body: func(job spine.ExecutionJob, lease spine.ExecutionJobLeaseCreated, _ spine.ExecutionCommandPlan) string {
					return projectTestReceiptBody(job.ID, lease.ID, lease.LeaseToken, "runner-1", "execution-command-plan-missing", spine.ExecutionReceiptStatusExited, intPtr(0), false)
				},
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				server := testServer(t)
				job, lease, run, plan := createProjectTestPlanFromSeededProbe(t, server)
				response := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/receipts", tt.body(job, lease, plan))
				if response.code != http.StatusBadRequest && response.code != http.StatusNotFound && response.code != http.StatusConflict {
					t.Fatalf("project test unsafe receipt status = %d, want reject: %s", response.code, response.body)
				}
				if len(server.executionReceipts.byRun) != 1 {
					t.Fatalf("execution receipt byRun count = %d, want only seeded project_probe receipt after unsafe project_test receipt", len(server.executionReceipts.byRun))
				}
			})
		}
	})
}

func TestExecutionRunnerRoutesCanRecoverReceiptAfterExpiredRunStartedLease(t *testing.T) {
	server := testServer(t)
	job, lease := createLeasedExecutionJob(t, server)
	runResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(job.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-1"}`, lease.ID, lease.LeaseToken))
	if runResponse.code != http.StatusCreated {
		t.Fatalf("run start status = %d, want %d: %s", runResponse.code, http.StatusCreated, runResponse.body)
	}
	var run spine.Run
	decodeJSON(t, runResponse.body, &run)
	expiredAt := testTime().Add(-time.Minute)
	storedJob := server.executionJobs.jobs[job.ID]
	storedJob.LeaseExpiresAt = &expiredAt
	server.executionJobs.jobs[job.ID] = storedJob

	recoverLeaseResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/leases", fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q,"runner_id":"runner-1"}`, job.RepoBindingID))
	if recoverLeaseResponse.code != http.StatusCreated {
		t.Fatalf("recover lease status = %d, want %d: %s", recoverLeaseResponse.code, http.StatusCreated, recoverLeaseResponse.body)
	}
	var recoverLease spine.ExecutionJobLeaseCreated
	decodeJSON(t, recoverLeaseResponse.body, &recoverLease)
	if recoverLease.ID == lease.ID || recoverLease.ExecutionJobID != job.ID || recoverLease.ExecutionJob.State != spine.ExecutionJobStateRunStarted {
		t.Fatalf("recovery lease = %#v, want fresh lease for existing run_started job", recoverLease)
	}
	recoveredRunResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(job.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-1"}`, recoverLease.ID, recoverLease.LeaseToken))
	if recoveredRunResponse.code != http.StatusOK {
		t.Fatalf("recovered run start status = %d, want %d: %s", recoveredRunResponse.code, http.StatusOK, recoveredRunResponse.body)
	}
	var recoveredRun spine.Run
	decodeJSON(t, recoveredRunResponse.body, &recoveredRun)
	if recoveredRun.ID != run.ID || len(server.runs.runs) != 1 {
		t.Fatalf("recovered run id/count = %q/%d, want existing %q/1", recoveredRun.ID, len(server.runs.runs), run.ID)
	}
	receiptResponse := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/receipts", executionReceiptBody(job.ID, recoverLease.ID, recoverLease.LeaseToken, "runner-1", "mounted:/workspace/goalrail", "abc123", false))
	if receiptResponse.code != http.StatusCreated {
		t.Fatalf("recovered execution receipt status = %d, want %d: %s", receiptResponse.code, http.StatusCreated, receiptResponse.body)
	}
	var receipt spine.ExecutionReceipt
	decodeJSON(t, receiptResponse.body, &receipt)
	if receipt.ExecutionLeaseID != recoverLease.ID || receipt.RunID != run.ID || receipt.ExecutionJobID != job.ID {
		t.Fatalf("recovered receipt = %#v, want fresh lease proof on existing run", receipt)
	}
	if server.executionJobs.jobs[job.ID].State != spine.ExecutionJobStateReceiptSubmitted || server.runs.runs[run.ID].State != spine.RunStateReceiptSubmitted {
		t.Fatalf("job/run state = %q/%q, want receipt_submitted", server.executionJobs.jobs[job.ID].State, server.runs.runs[run.ID].State)
	}
}

func TestExecutionRunnerRoutesRejectBoundaryFailures(t *testing.T) {
	t.Run("unauthenticated rejects before mutation", func(t *testing.T) {
		server := testServerWithContinuationAuth(t, fakeHTTPAuthService{meErr: auth.ErrInvalidToken})
		lease := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/leases", `{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","runner_id":"runner-1"}`)
		assertErrorCode(t, lease, http.StatusUnauthorized, "unauthorized")
		run := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/execution-job-1/runs", `{"lease_id":"execution-lease-1","lease_token":"secret","runner_id":"runner-1"}`)
		assertErrorCode(t, run, http.StatusUnauthorized, "unauthorized")
		commandPlan := doJSON(t, server.router, http.MethodPost, "/v1/runs/run-1/command-plans", `{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004"}`)
		assertErrorCode(t, commandPlan, http.StatusUnauthorized, "unauthorized")
		receipt := doJSON(t, server.router, http.MethodPost, "/v1/runs/run-1/receipts", executionReceiptBody("execution-job-1", "execution-lease-1", "secret", "runner-1", "mounted:/workspace/goalrail", "abc123", false))
		assertErrorCode(t, receipt, http.StatusUnauthorized, "unauthorized")
		if len(server.runs.runs) != 0 {
			t.Fatalf("runs = %d, want 0 after auth failure", len(server.runs.runs))
		}
		if len(server.executionReceipts.receipts) != 0 {
			t.Fatalf("execution receipts = %d, want 0 after auth failure", len(server.executionReceipts.receipts))
		}
		if len(server.commandPlans.plans) != 0 {
			t.Fatalf("command plans = %d, want 0 after auth failure", len(server.commandPlans.plans))
		}
	})

	t.Run("no queued execution job returns no work", func(t *testing.T) {
		server := testServer(t)
		response := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/leases", `{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","runner_id":"runner-1"}`)
		if response.code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d: %s", response.code, http.StatusNoContent, response.body)
		}
		if len(server.runs.runs) != 0 {
			t.Fatalf("runs = %d, want no Run after no-work lease", len(server.runs.runs))
		}
	})

	t.Run("lease rejects foreign organization scope", func(t *testing.T) {
		server := testServerWithContinuationAuth(t, fakeHTTPAuthService{
			profile: continuationAuthProfile("018f0000-0000-7000-8000-000000000099"),
		})
		response := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/leases", `{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","runner_id":"runner-1"}`)
		assertErrorCode(t, response, http.StatusForbidden, "forbidden")
	})

	t.Run("lease rejects project expectation mismatch", func(t *testing.T) {
		server := testServer(t)
		response := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/leases", `{"project_id":"project-mismatch","repo_binding_id":"018f0000-0000-7000-8000-000000000004","runner_id":"runner-1"}`)
		assertErrorCode(t, response, http.StatusConflict, "project_context_mismatch")
	})

	t.Run("run start rejects invalid lease proof", func(t *testing.T) {
		server := testServer(t)
		job, lease := createLeasedExecutionJob(t, server)

		response := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(job.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":"wrong-token","runner_id":"runner-1"}`, lease.ID))
		assertErrorCode(t, response, http.StatusConflict, "invalid_lease")
		if len(server.runs.runs) != 0 {
			t.Fatalf("runs = %d, want 0 after invalid lease token", len(server.runs.runs))
		}
	})

	t.Run("run start rejects wrong runner", func(t *testing.T) {
		server := testServer(t)
		job, lease := createLeasedExecutionJob(t, server)

		response := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(job.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-other"}`, lease.ID, lease.LeaseToken))
		assertErrorCode(t, response, http.StatusConflict, "invalid_lease")
		if len(server.runs.runs) != 0 {
			t.Fatalf("runs = %d, want 0 after wrong runner", len(server.runs.runs))
		}
	})

	t.Run("run start rejects expired lease", func(t *testing.T) {
		server := testServer(t)
		job, lease := createLeasedExecutionJob(t, server)
		storedJob := server.executionJobs.jobs[job.ID]
		expiredAt := testTime().Add(-time.Minute)
		storedJob.LeaseExpiresAt = &expiredAt
		server.executionJobs.jobs[job.ID] = storedJob

		response := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(job.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-1"}`, lease.ID, lease.LeaseToken))
		assertErrorCode(t, response, http.StatusConflict, "lease_expired")
		if len(server.runs.runs) != 0 {
			t.Fatalf("runs = %d, want 0 after expired lease", len(server.runs.runs))
		}
	})

	t.Run("receipt submit rejects unknown run", func(t *testing.T) {
		server := testServer(t)
		response := doJSON(t, server.router, http.MethodPost, "/v1/runs/run-missing/receipts", executionReceiptBody("execution-job-1", "execution-lease-1", "secret", "runner-1", "mounted:/workspace/goalrail", "abc123", false))
		assertErrorCode(t, response, http.StatusNotFound, "not_found")
		if len(server.executionReceipts.receipts) != 0 {
			t.Fatalf("execution receipts = %d, want 0 after unknown run", len(server.executionReceipts.receipts))
		}
	})

	t.Run("receipt submit rejects non-started run", func(t *testing.T) {
		server := testServer(t)
		job, lease := createLeasedExecutionJob(t, server)
		run := spine.Run{
			ID:                "run-invalid-state",
			ExecutionJobID:    job.ID,
			ExecutionLeaseID:  lease.ID,
			TaskID:            lease.TaskID,
			CheckoutReceiptID: lease.CheckoutReceiptID,
			RunnerID:          "runner-1",
			State:             spine.RunState("failed"),
			StartedAt:         testTime(),
			CreatedAt:         testTime(),
			UpdatedAt:         testTime(),
		}
		if err := server.runs.Create(context.Background(), run); err != nil {
			t.Fatalf("runs.Create() error = %v", err)
		}
		response := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/receipts", executionReceiptBody(job.ID, lease.ID, lease.LeaseToken, "runner-1", "mounted:/workspace/goalrail", "abc123", false))
		assertErrorCode(t, response, http.StatusConflict, "invalid_state")
		if len(server.executionReceipts.receipts) != 0 {
			t.Fatalf("execution receipts = %d, want 0 after invalid run state", len(server.executionReceipts.receipts))
		}
	})

	t.Run("receipt submit rejects raw source upload", func(t *testing.T) {
		server := testServer(t)
		job, lease := createLeasedExecutionJob(t, server)
		runResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(job.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-1"}`, lease.ID, lease.LeaseToken))
		if runResponse.code != http.StatusCreated {
			t.Fatalf("run start status = %d, want %d: %s", runResponse.code, http.StatusCreated, runResponse.body)
		}
		var run spine.Run
		decodeJSON(t, runResponse.body, &run)
		response := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/receipts", executionReceiptBody(job.ID, lease.ID, lease.LeaseToken, "runner-1", "mounted:/workspace/goalrail", "abc123", true))
		assertErrorCode(t, response, http.StatusBadRequest, "validation_failed")
		if len(server.executionReceipts.receipts) != 0 {
			t.Fatalf("execution receipts = %d, want 0 after raw source upload", len(server.executionReceipts.receipts))
		}
	})

	t.Run("receipt submit rejects artifact and changed path claims", func(t *testing.T) {
		server := testServer(t)
		job, lease := createLeasedExecutionJob(t, server)
		runResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(job.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-1"}`, lease.ID, lease.LeaseToken))
		if runResponse.code != http.StatusCreated {
			t.Fatalf("run start status = %d, want %d: %s", runResponse.code, http.StatusCreated, runResponse.body)
		}
		var run spine.Run
		decodeJSON(t, runResponse.body, &run)

		body := fmt.Sprintf(`{"execution_job_id":%q,"lease_id":%q,"lease_token":%q,"runner_id":"runner-1","workspace_ref":"mounted:/workspace/goalrail","commit_sha":"abc123","baseline_id":"baseline-1","overlay_id":"overlay-1","execution_mode":"no_command","process_status":"not_executed","artifact_refs":["artifact-1"],"changed_paths_summary":["file.go"],"raw_source_uploaded":false}`, job.ID, lease.ID, lease.LeaseToken)
		response := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/receipts", body)
		assertErrorCode(t, response, http.StatusBadRequest, "validation_failed")
		if len(server.executionReceipts.receipts) != 0 {
			t.Fatalf("execution receipts = %d, want 0 after artifact/path claim", len(server.executionReceipts.receipts))
		}
	})

	t.Run("command plan rejects non-started run", func(t *testing.T) {
		server := testServer(t)
		job, lease := createLeasedExecutionJob(t, server)
		run := spine.Run{
			ID:                "run-invalid-command-plan-state",
			ExecutionJobID:    job.ID,
			ExecutionLeaseID:  lease.ID,
			TaskID:            lease.TaskID,
			CheckoutReceiptID: lease.CheckoutReceiptID,
			RunnerID:          "runner-1",
			State:             spine.RunStateReceiptSubmitted,
			StartedAt:         testTime(),
			CreatedAt:         testTime(),
			UpdatedAt:         testTime(),
		}
		if err := server.runs.Create(context.Background(), run); err != nil {
			t.Fatalf("runs.Create() error = %v", err)
		}
		response := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/command-plans", fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q}`, job.RepoBindingID))
		assertErrorCode(t, response, http.StatusConflict, "invalid_state")
		if len(server.commandPlans.plans) != 0 {
			t.Fatalf("command plans = %d, want 0 after invalid run state", len(server.commandPlans.plans))
		}
	})

	t.Run("command plan rejects non-allowlisted kind and action", func(t *testing.T) {
		for _, tt := range []struct {
			name   string
			kind   string
			action string
		}{
			{name: "project command", kind: "project_command", action: "npm_test"},
			{name: "project probe action", kind: "project_probe", action: "run_tests"},
			{name: "builtin diagnostic action", kind: "builtin_diagnostic", action: "detect_declared_test_targets"},
		} {
			t.Run(tt.name, func(t *testing.T) {
				server := testServer(t)
				job, lease := createLeasedExecutionJob(t, server)
				runResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(job.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-1"}`, lease.ID, lease.LeaseToken))
				if runResponse.code != http.StatusCreated {
					t.Fatalf("run start status = %d, want %d: %s", runResponse.code, http.StatusCreated, runResponse.body)
				}
				var run spine.Run
				decodeJSON(t, runResponse.body, &run)

				body := fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q,"command_kind":%q,"action":%q}`, job.RepoBindingID, tt.kind, tt.action)
				response := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/command-plans", body)
				assertErrorCode(t, response, http.StatusBadRequest, "validation_failed")
				if len(server.commandPlans.plans) != 0 {
					t.Fatalf("command plans = %d, want 0 after non-allowlisted command", len(server.commandPlans.plans))
				}
			})
		}
	})

	t.Run("command plan rejects shell argv and user command strings", func(t *testing.T) {
		for _, tt := range []struct {
			name string
			body string
		}{
			{name: "shell", body: `,"shell":true`},
			{name: "shell_allowed", body: `,"shell_allowed":true`},
			{name: "argv", body: `,"argv":["npm","test"]`},
			{name: "command", body: `,"command":"npm test"`},
			{name: "command_string", body: `,"command_string":"go test ./..."`},
			{name: "user_command", body: `,"user_command":"pytest"`},
		} {
			t.Run(tt.name, func(t *testing.T) {
				server := testServer(t)
				job, lease := createLeasedExecutionJob(t, server)
				runResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(job.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-1"}`, lease.ID, lease.LeaseToken))
				if runResponse.code != http.StatusCreated {
					t.Fatalf("run start status = %d, want %d: %s", runResponse.code, http.StatusCreated, runResponse.body)
				}
				var run spine.Run
				decodeJSON(t, runResponse.body, &run)

				body := fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q,"command_kind":"project_probe","action":"detect_declared_test_targets"%s}`, job.RepoBindingID, tt.body)
				response := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/command-plans", body)
				assertErrorCode(t, response, http.StatusBadRequest, "validation_failed")
				if len(server.commandPlans.plans) != 0 {
					t.Fatalf("command plans = %d, want 0 after unsafe command request", len(server.commandPlans.plans))
				}
			})
		}
	})

	t.Run("builtin diagnostic receipt rejects missing command plan", func(t *testing.T) {
		server := testServer(t)
		job, lease := createLeasedExecutionJob(t, server)
		runResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(job.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-1"}`, lease.ID, lease.LeaseToken))
		if runResponse.code != http.StatusCreated {
			t.Fatalf("run start status = %d, want %d: %s", runResponse.code, http.StatusCreated, runResponse.body)
		}
		var run spine.Run
		decodeJSON(t, runResponse.body, &run)
		response := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/receipts", builtinDiagnosticReceiptBody(job.ID, lease.ID, lease.LeaseToken, "runner-1", "execution-command-plan-missing"))
		assertErrorCode(t, response, http.StatusNotFound, "not_found")
		if len(server.executionReceipts.receipts) != 0 {
			t.Fatalf("execution receipts = %d, want 0 after missing command plan", len(server.executionReceipts.receipts))
		}
	})

	t.Run("project probe receipt rejects missing metadata", func(t *testing.T) {
		server := testServer(t)
		job, lease := createLeasedExecutionJob(t, server)
		runResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(job.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-1"}`, lease.ID, lease.LeaseToken))
		if runResponse.code != http.StatusCreated {
			t.Fatalf("run start status = %d, want %d: %s", runResponse.code, http.StatusCreated, runResponse.body)
		}
		var run spine.Run
		decodeJSON(t, runResponse.body, &run)
		planResponse := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/command-plans", fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q,"command_kind":"project_probe","action":"detect_declared_test_targets"}`, job.RepoBindingID))
		if planResponse.code != http.StatusCreated {
			t.Fatalf("project probe command plan status = %d, want %d: %s", planResponse.code, http.StatusCreated, planResponse.body)
		}
		var plan spine.ExecutionCommandPlan
		decodeJSON(t, planResponse.body, &plan)

		body := fmt.Sprintf(`{"execution_job_id":%q,"lease_id":%q,"lease_token":%q,"runner_id":"runner-1","workspace_ref":"mounted:/workspace/goalrail","commit_sha":"abc123","execution_mode":"project_probe","command_plan_id":%q,"command_kind":"project_probe","action":"detect_declared_test_targets","process_status":"metadata_only","artifact_refs":[],"changed_paths_summary":[],"raw_source_uploaded":false,"runner_started_at":"2026-04-25T12:00:00Z","runner_finished_at":"2026-04-25T12:00:00Z"}`, job.ID, lease.ID, lease.LeaseToken, plan.ID)
		response := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/receipts", body)
		assertErrorCode(t, response, http.StatusBadRequest, "validation_failed")
		if len(server.executionReceipts.receipts) != 0 {
			t.Fatalf("execution receipts = %d, want 0 after missing project probe metadata", len(server.executionReceipts.receipts))
		}
	})

	t.Run("project probe receipt rejects artifact changed path and raw source claims", func(t *testing.T) {
		for _, tt := range []struct {
			name   string
			mutate func(string) string
		}{
			{
				name: "artifacts and changed paths",
				mutate: func(body string) string {
					return strings.Replace(body, `"artifact_refs":[],"changed_paths_summary":[]`, `"artifact_refs":["artifact-1"],"changed_paths_summary":["file.go"]`, 1)
				},
			},
			{
				name: "raw source upload",
				mutate: func(body string) string {
					return strings.Replace(body, `"raw_source_uploaded":false`, `"raw_source_uploaded":true`, 1)
				},
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				server := testServer(t)
				job, lease := createLeasedExecutionJob(t, server)
				runResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(job.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-1"}`, lease.ID, lease.LeaseToken))
				if runResponse.code != http.StatusCreated {
					t.Fatalf("run start status = %d, want %d: %s", runResponse.code, http.StatusCreated, runResponse.body)
				}
				var run spine.Run
				decodeJSON(t, runResponse.body, &run)
				planResponse := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/command-plans", fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q,"command_kind":"project_probe","action":"detect_declared_test_targets"}`, job.RepoBindingID))
				if planResponse.code != http.StatusCreated {
					t.Fatalf("project probe command plan status = %d, want %d: %s", planResponse.code, http.StatusCreated, planResponse.body)
				}
				var plan spine.ExecutionCommandPlan
				decodeJSON(t, planResponse.body, &plan)

				body := tt.mutate(projectProbeReceiptBody(job.ID, lease.ID, lease.LeaseToken, "runner-1", plan.ID))
				response := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/receipts", body)
				assertErrorCode(t, response, http.StatusBadRequest, "validation_failed")
				if len(server.executionReceipts.receipts) != 0 {
					t.Fatalf("execution receipts = %d, want 0 after unsafe project probe receipt", len(server.executionReceipts.receipts))
				}
			})
		}
	})
}

func TestPostTaskCheckoutJobsRejectsAuthAndContextMismatch(t *testing.T) {
	t.Run("unauthenticated rejects before job creation", func(t *testing.T) {
		server := testServerWithContinuationAuth(t, fakeHTTPAuthService{meErr: auth.ErrInvalidToken})
		response := doJSON(t, server.router, http.MethodPost, "/v1/tasks/work-item-1/checkout-jobs", `{}`)
		assertErrorCode(t, response, http.StatusUnauthorized, "unauthorized")
		if len(server.checkoutJobs.jobs) != 0 {
			t.Fatalf("checkout jobs = %d, want 0 after auth failure", len(server.checkoutJobs.jobs))
		}
	})

	t.Run("project expectation mismatch rejects before job creation", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		plan := createPlan(t, server, approved.ContractID)
		proposal := submitProposal(t, server, plan.ID, string(approved.ID))
		accepted := acceptProposal(t, server, proposal.ID)

		response := doJSON(t, server.router, http.MethodPost, "/v1/tasks/"+string(accepted.CreatedTaskIDs[0])+"/checkout-jobs", `{"project_id":"project-mismatch"}`)
		assertErrorCode(t, response, http.StatusConflict, "project_context_mismatch")
		if len(server.checkoutJobs.jobs) != 0 {
			t.Fatalf("checkout jobs = %d, want 0 after project mismatch", len(server.checkoutJobs.jobs))
		}
	})
}

func createCheckoutReceipt(t *testing.T, server testServerDeps, taskID spine.WorkItemID) (spine.CheckoutJob, spine.CheckoutReceipt) {
	t.Helper()

	jobResponse := doJSON(t, server.router, http.MethodPost, "/v1/tasks/"+string(taskID)+"/checkout-jobs", `{}`)
	if jobResponse.code != http.StatusCreated && jobResponse.code != http.StatusOK {
		t.Fatalf("create checkout job status = %d, want success: %s", jobResponse.code, jobResponse.body)
	}
	var job spine.CheckoutJob
	decodeJSON(t, jobResponse.body, &job)

	leaseBody := fmt.Sprintf(`{"project_id":%q,"repo_binding_id":%q,"runner_id":"runner-1"}`, "018f0000-0000-7000-8000-000000000003", job.RepoBindingID)
	leaseResponse := doJSON(t, server.router, http.MethodPost, "/v1/checkout-jobs/leases", leaseBody)
	if leaseResponse.code != http.StatusCreated {
		t.Fatalf("lease status = %d, want %d: %s", leaseResponse.code, http.StatusCreated, leaseResponse.body)
	}
	var lease spine.CheckoutJobLeaseCreated
	decodeJSON(t, leaseResponse.body, &lease)

	receiptBody := fmt.Sprintf(`{"lease_token":%q,"runner_id":"runner-1","workspace_ref":"mounted:/workspace/goalrail","commit_sha":"abc123","baseline_id":"baseline-1","overlay_id":"overlay-1","dirty":false,"partial":false,"raw_source_uploaded":false}`, lease.LeaseToken)
	receiptResponse := doJSON(t, server.router, http.MethodPost, "/v1/checkout-jobs/"+string(job.ID)+"/receipts", receiptBody)
	if receiptResponse.code != http.StatusCreated {
		t.Fatalf("receipt status = %d, want %d: %s", receiptResponse.code, http.StatusCreated, receiptResponse.body)
	}
	var receipt spine.CheckoutReceipt
	decodeJSON(t, receiptResponse.body, &receipt)
	return job, receipt
}

func createExecutionJob(t *testing.T, server testServerDeps, taskID spine.WorkItemID, receiptID spine.CheckoutReceiptID) spine.ExecutionJob {
	t.Helper()

	response := doJSON(t, server.router, http.MethodPost, "/v1/tasks/"+string(taskID)+"/execution-jobs", fmt.Sprintf(`{"checkout_receipt_id":%q}`, receiptID))
	if response.code != http.StatusCreated && response.code != http.StatusOK {
		t.Fatalf("create execution job status = %d, want success: %s", response.code, response.body)
	}
	var job spine.ExecutionJob
	decodeJSON(t, response.body, &job)
	return job
}

func createLeasedExecutionJob(t *testing.T, server testServerDeps) (spine.ExecutionJob, spine.ExecutionJobLeaseCreated) {
	t.Helper()

	approved := createApprovedContract(t, server)
	plan := createPlan(t, server, approved.ContractID)
	proposal := submitProposal(t, server, plan.ID, string(approved.ID))
	accepted := acceptProposal(t, server, proposal.ID)
	taskID := accepted.CreatedTaskIDs[0]
	_, receipt := createCheckoutReceipt(t, server, taskID)
	job := createExecutionJob(t, server, taskID, receipt.ID)
	response := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/leases", fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q,"runner_id":"runner-1"}`, job.RepoBindingID))
	if response.code != http.StatusCreated {
		t.Fatalf("execution lease status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	var lease spine.ExecutionJobLeaseCreated
	decodeJSON(t, response.body, &lease)
	return job, lease
}

func executionReceiptBody(jobID spine.ExecutionJobID, leaseID spine.ExecutionLeaseID, leaseToken string, runnerID string, workspaceRef string, commitSHA string, rawSourceUploaded bool) string {
	return fmt.Sprintf(`{"execution_job_id":%q,"lease_id":%q,"lease_token":%q,"runner_id":%q,"workspace_ref":%q,"commit_sha":%q,"baseline_id":"baseline-1","overlay_id":"overlay-1","execution_mode":"no_command","process_status":"not_executed","artifact_refs":[],"changed_paths_summary":[],"raw_source_uploaded":%t}`, jobID, leaseID, leaseToken, runnerID, workspaceRef, commitSHA, rawSourceUploaded)
}

func builtinDiagnosticReceiptBody(jobID spine.ExecutionJobID, leaseID spine.ExecutionLeaseID, leaseToken string, runnerID string, planID spine.ExecutionCommandPlanID) string {
	return fmt.Sprintf(`{"execution_job_id":%q,"lease_id":%q,"lease_token":%q,"runner_id":%q,"workspace_ref":"mounted:/workspace/goalrail","commit_sha":"abc123","baseline_id":"baseline-1","overlay_id":"overlay-1","execution_mode":"builtin_diagnostic","command_plan_id":%q,"command_kind":"builtin_diagnostic","action":"workspace_status","process_status":"metadata_only","artifact_refs":[],"changed_paths_summary":[],"raw_source_uploaded":false,"runner_started_at":"2026-04-25T12:00:00Z","runner_finished_at":"2026-04-25T12:00:00Z"}`, jobID, leaseID, leaseToken, runnerID, planID)
}

func projectProbeReceiptBody(jobID spine.ExecutionJobID, leaseID spine.ExecutionLeaseID, leaseToken string, runnerID string, planID spine.ExecutionCommandPlanID) string {
	return projectProbeReceiptBodyWithTarget(jobID, leaseID, leaseToken, runnerID, planID, "test", "package.json", "package_json_script")
}

func projectProbeReceiptBodyWithTarget(jobID spine.ExecutionJobID, leaseID spine.ExecutionLeaseID, leaseToken string, runnerID string, planID spine.ExecutionCommandPlanID, targetName string, targetSourcePath string, targetSourceKind string) string {
	return fmt.Sprintf(`{"execution_job_id":%q,"lease_id":%q,"lease_token":%q,"runner_id":%q,"workspace_ref":"mounted:/workspace/goalrail","commit_sha":"abc123","baseline_id":"baseline-1","overlay_id":"overlay-1","execution_mode":"project_probe","command_plan_id":%q,"command_kind":"project_probe","action":"detect_declared_test_targets","process_status":"metadata_only","artifact_refs":[],"changed_paths_summary":[],"raw_source_uploaded":false,"runner_started_at":"2026-04-25T12:00:00Z","runner_finished_at":"2026-04-25T12:00:00Z","project_probe_metadata":{"detected_manifests":[{"path":"package.json","kind":"node_package_manifest"}],"package_manager_candidates":[{"name":"npm","source_path":"package.json"}],"declared_test_target_candidates":[{"name":%q,"source_path":%q,"source_kind":%q}],"unsupported_or_unknowns":[],"partiality_reasons":["probe reads only allowlisted manifest files under path_scope"]}}`, jobID, leaseID, leaseToken, runnerID, planID, targetName, targetSourcePath, targetSourceKind)
}

func projectTestReceiptBody(jobID spine.ExecutionJobID, leaseID spine.ExecutionLeaseID, leaseToken string, runnerID string, planID spine.ExecutionCommandPlanID, status string, exitCode *int, rawSourceUploaded bool) string {
	exitCodeField := ""
	if exitCode != nil {
		exitCodeField = fmt.Sprintf(`,"exit_code":%d`, *exitCode)
	}
	return fmt.Sprintf(`{"execution_job_id":%q,"lease_id":%q,"lease_token":%q,"runner_id":%q,"workspace_ref":"mounted:/workspace/goalrail","commit_sha":"abc123","baseline_id":"baseline-1","overlay_id":"overlay-1","execution_mode":"project_test","command_plan_id":%q,"command_kind":"project_test","action":"run_declared_test_target","process_status":%q%s,"artifact_refs":[],"changed_paths_summary":[],"raw_source_uploaded":%t,"runner_started_at":"2026-04-25T12:00:00Z","runner_finished_at":"2026-04-25T12:00:00Z"}`, jobID, leaseID, leaseToken, runnerID, planID, status, exitCodeField, rawSourceUploaded)
}

func intPtr(value int) *int {
	return &value
}

func createProjectProbeReceipt(t *testing.T, server testServerDeps) (spine.ExecutionJob, spine.ExecutionJobLeaseCreated, spine.Run, spine.ExecutionCommandPlan, spine.ExecutionReceipt) {
	t.Helper()

	job, lease := createLeasedExecutionJob(t, server)
	runResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(job.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-1"}`, lease.ID, lease.LeaseToken))
	if runResponse.code != http.StatusCreated {
		t.Fatalf("run start status = %d, want %d: %s", runResponse.code, http.StatusCreated, runResponse.body)
	}
	var run spine.Run
	decodeJSON(t, runResponse.body, &run)

	planResponse := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/command-plans", fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q,"command_kind":"project_probe","action":"detect_declared_test_targets"}`, job.RepoBindingID))
	if planResponse.code != http.StatusCreated {
		t.Fatalf("project probe command plan status = %d, want %d: %s", planResponse.code, http.StatusCreated, planResponse.body)
	}
	var plan spine.ExecutionCommandPlan
	decodeJSON(t, planResponse.body, &plan)

	receiptResponse := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/receipts", projectProbeReceiptBody(job.ID, lease.ID, lease.LeaseToken, "runner-1", plan.ID))
	if receiptResponse.code != http.StatusCreated {
		t.Fatalf("project probe receipt status = %d, want %d: %s", receiptResponse.code, http.StatusCreated, receiptResponse.body)
	}
	var receipt spine.ExecutionReceipt
	decodeJSON(t, receiptResponse.body, &receipt)
	return job, lease, run, plan, receipt
}

func createProjectTestPlanFromSeededProbe(t *testing.T, server testServerDeps) (spine.ExecutionJob, spine.ExecutionJobLeaseCreated, spine.Run, spine.ExecutionCommandPlan) {
	t.Helper()

	job, lease := createLeasedExecutionJob(t, server)
	runResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(job.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-1"}`, lease.ID, lease.LeaseToken))
	if runResponse.code != http.StatusCreated {
		t.Fatalf("run start status = %d, want %d: %s", runResponse.code, http.StatusCreated, runResponse.body)
	}
	var run spine.Run
	decodeJSON(t, runResponse.body, &run)
	probeReceipt := seedProjectProbeReceiptForJob(t, server, job)

	body := fmt.Sprintf(`{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":%q,"command_kind":"project_test","action":"run_declared_test_target","project_probe_receipt_id":%q,"selected_target_id":"package.json#package_json_script:test"}`, job.RepoBindingID, probeReceipt.ID)
	planResponse := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/command-plans", body)
	if planResponse.code != http.StatusCreated {
		t.Fatalf("project test command plan status = %d, want %d: %s", planResponse.code, http.StatusCreated, planResponse.body)
	}
	var plan spine.ExecutionCommandPlan
	decodeJSON(t, planResponse.body, &plan)
	return job, lease, run, plan
}

func seedProjectProbeReceiptForJob(t *testing.T, server testServerDeps, job spine.ExecutionJob) spine.ExecutionReceipt {
	t.Helper()

	now := testTime()
	receipt := spine.ExecutionReceipt{
		ID:                  "018f0000-0000-7000-8004-000000000123",
		RunID:               "run-seeded-project-probe",
		ExecutionJobID:      "execution-job-seeded-project-probe",
		ExecutionLeaseID:    "execution-lease-seeded-project-probe",
		TaskID:              job.TaskID,
		CheckoutReceiptID:   job.CheckoutReceiptID,
		RepoBindingID:       job.RepoBindingID,
		RunnerID:            "runner-1",
		WorkspaceRef:        "mounted:/workspace/goalrail",
		CommitSHA:           "abc123",
		ExecutionMode:       spine.ExecutionReceiptModeProjectProbe,
		CommandKind:         spine.ExecutionCommandKindProjectProbe,
		Action:              spine.ExecutionCommandActionDetectTestTargets,
		ProcessStatus:       spine.ExecutionReceiptStatusMetadataOnly,
		ArtifactRefs:        []string{},
		ChangedPathsSummary: []string{},
		RawSourceUploaded:   false,
		ProjectProbeMetadata: &spine.ProjectProbeMetadata{
			DetectedManifests: []spine.ProjectProbeManifest{
				{Path: "package.json", Kind: "node_package_manifest"},
			},
			PackageManagerCandidates: []spine.ProjectProbePackageManagerCandidate{
				{Name: "npm", SourcePath: "package.json"},
			},
			DeclaredTestTargetCandidates: []spine.ProjectProbeTestTargetCandidate{
				{Name: "test", SourcePath: "package.json", SourceKind: "package_json_script"},
			},
			UnsupportedOrUnknowns: []string{},
			PartialityReasons:     []string{"seeded project_probe metadata for project_test receipt regression"},
		},
		StartedAt:  now,
		FinishedAt: now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	server.executionReceipts.receipts[receipt.ID] = receipt
	server.executionReceipts.byRun[receipt.RunID] = receipt.ID
	return receipt
}

func createStartedExecutionRunForProjectTestPlan(t *testing.T, server testServerDeps, sourceJob spine.ExecutionJob) (spine.ExecutionJob, spine.Run) {
	t.Helper()

	storedSourceJob, ok := server.executionJobs.jobs[sourceJob.ID]
	if !ok {
		t.Fatalf("source execution job %q not stored", sourceJob.ID)
	}
	now := testTime().Add(time.Minute)
	expiresAt := now.Add(15 * time.Minute)
	leaseID := spine.ExecutionLeaseID("execution-lease-project-test")
	job := storedSourceJob
	job.ID = "execution-job-project-test"
	job.State = spine.ExecutionJobStateRunStarted
	job.CurrentLeaseID = &leaseID
	job.CurrentRunnerID = "runner-1"
	job.LeaseTokenHash = "test-lease-token-hash"
	job.LeaseExpiresAt = &expiresAt
	job.CreatedAt = now
	job.UpdatedAt = now
	server.executionJobs.jobs[job.ID] = job

	run := spine.Run{
		ID:                "run-project-test-plan",
		ExecutionJobID:    job.ID,
		ExecutionLeaseID:  leaseID,
		TaskID:            storedSourceJob.TaskID,
		CheckoutReceiptID: storedSourceJob.CheckoutReceiptID,
		RunnerID:          "runner-1",
		State:             spine.RunStateStarted,
		StartedAt:         now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	server.runs.runs[run.ID] = run
	server.runs.byJob[job.ID] = run.ID
	return job, run
}

func TestCheckoutRunnerRoutesRejectUnauthenticatedRequests(t *testing.T) {
	server := testServerWithContinuationAuth(t, fakeHTTPAuthService{meErr: auth.ErrInvalidToken})

	lease := doJSON(t, server.router, http.MethodPost, "/v1/checkout-jobs/leases", `{"runner_id":"runner-1"}`)
	assertErrorCode(t, lease, http.StatusUnauthorized, "unauthorized")

	receipt := doJSON(t, server.router, http.MethodPost, "/v1/checkout-jobs/job-1/receipts", `{"lease_token":"secret","runner_id":"runner-1","workspace_ref":"mounted:/workspace/goalrail","commit_sha":"abc123","raw_source_uploaded":false}`)
	assertErrorCode(t, receipt, http.StatusUnauthorized, "unauthorized")
	if len(server.checkoutReceipts.receipts) != 0 {
		t.Fatalf("checkout receipts = %d, want 0 after auth failure", len(server.checkoutReceipts.receipts))
	}
}

func TestCheckoutRunnerRoutesRespectOrganizationBoundary(t *testing.T) {
	t.Run("lease only sees jobs from authenticated organization", func(t *testing.T) {
		server := testServerWithContinuationAuth(t, fakeHTTPAuthService{
			profile: continuationAuthProfile("018f0000-0000-7000-8000-000000000099"),
		})
		job := spine.CheckoutJob{
			ID:             "checkout-job-foreign",
			OrganizationID: "018f0000-0000-7000-8000-000000000002",
			ProjectID:      "018f0000-0000-7000-8000-000000000003",
			TaskID:         "work-item-foreign",
			RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
			State:          spine.CheckoutJobStateQueued,
			CreatedAt:      testTime(),
			UpdatedAt:      testTime(),
		}
		server.checkoutJobs.jobs[job.ID] = job

		response := doJSON(t, server.router, http.MethodPost, "/v1/checkout-jobs/leases", `{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","runner_id":"runner-1"}`)
		assertErrorCode(t, response, http.StatusForbidden, "forbidden")
		if got := server.checkoutJobs.jobs[job.ID].State; got != spine.CheckoutJobStateQueued {
			t.Fatalf("foreign checkout job state = %q, want queued", got)
		}
	})

	t.Run("lease only sees requested project and repo binding scope", func(t *testing.T) {
		server := testServer(t)
		otherBinding := server.repoBindings.bindings["018f0000-0000-7000-8000-000000000004"]
		otherBinding.ID = "018f0000-0000-7000-8000-000000000044"
		otherBinding.RepositoryFullName = "heurema/other"
		otherBinding.RepositoryURL = "https://github.com/heurema/other"
		server.repoBindings.bindings[otherBinding.ID] = otherBinding

		wantedJob := spine.CheckoutJob{
			ID:             "checkout-job-wanted",
			OrganizationID: "018f0000-0000-7000-8000-000000000002",
			ProjectID:      "018f0000-0000-7000-8000-000000000003",
			TaskID:         "work-item-wanted",
			RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
			State:          spine.CheckoutJobStateQueued,
			CreatedAt:      testTime().Add(time.Minute),
			UpdatedAt:      testTime(),
		}
		otherJob := spine.CheckoutJob{
			ID:             "checkout-job-other",
			OrganizationID: "018f0000-0000-7000-8000-000000000002",
			ProjectID:      "018f0000-0000-7000-8000-000000000003",
			TaskID:         "work-item-other",
			RepoBindingID:  otherBinding.ID,
			State:          spine.CheckoutJobStateQueued,
			CreatedAt:      testTime(),
			UpdatedAt:      testTime(),
		}
		server.checkoutJobs.jobs[wantedJob.ID] = wantedJob
		server.checkoutJobs.jobs[otherJob.ID] = otherJob

		response := doJSON(t, server.router, http.MethodPost, "/v1/checkout-jobs/leases", `{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","runner_id":"runner-1"}`)
		if response.code != http.StatusCreated {
			t.Fatalf("lease status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
		}
		var lease spine.CheckoutJobLeaseCreated
		decodeJSON(t, response.body, &lease)
		if lease.JobID != wantedJob.ID {
			t.Fatalf("lease job = %q, want scoped job %q", lease.JobID, wantedJob.ID)
		}
		if got := server.checkoutJobs.jobs[otherJob.ID].State; got != spine.CheckoutJobStateQueued {
			t.Fatalf("other repo checkout job state = %q, want queued", got)
		}
	})

	t.Run("receipt rejects job from another organization", func(t *testing.T) {
		server := testServerWithContinuationAuth(t, fakeHTTPAuthService{
			profile: continuationAuthProfile("018f0000-0000-7000-8000-000000000099"),
		})
		expiresAt := testTime().Add(time.Hour)
		job := spine.CheckoutJob{
			ID:              "checkout-job-foreign",
			OrganizationID:  "018f0000-0000-7000-8000-000000000002",
			ProjectID:       "018f0000-0000-7000-8000-000000000003",
			TaskID:          "work-item-foreign",
			RepoBindingID:   "018f0000-0000-7000-8000-000000000004",
			State:           spine.CheckoutJobStateLeased,
			CurrentRunnerID: "runner-1",
			LeaseTokenHash:  "unused",
			LeaseExpiresAt:  &expiresAt,
			CreatedAt:       testTime(),
			UpdatedAt:       testTime(),
		}
		server.checkoutJobs.jobs[job.ID] = job

		response := doJSON(t, server.router, http.MethodPost, "/v1/checkout-jobs/"+string(job.ID)+"/receipts", `{"lease_token":"secret","runner_id":"runner-1","workspace_ref":"mounted:/workspace/goalrail","commit_sha":"abc123","raw_source_uploaded":false}`)
		assertErrorCode(t, response, http.StatusForbidden, "forbidden")
		if len(server.checkoutReceipts.receipts) != 0 {
			t.Fatalf("checkout receipts = %d, want 0 after org mismatch", len(server.checkoutReceipts.receipts))
		}
	})
}

func TestCheckoutRunnerRoutesReturnClientErrorForMalformedIDs(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodPost, "/v1/checkout-jobs/malformed-id/receipts", `{"lease_token":"secret","runner_id":"runner-1","workspace_ref":"mounted:/workspace/goalrail","commit_sha":"abc123","raw_source_uploaded":false}`)
	assertErrorCode(t, response, http.StatusBadRequest, "validation_failed")
}

func TestRemovedDirectTaskRouteAndListRoutesReturnNotFound(t *testing.T) {
	server := testServer(t)
	for _, tt := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/v1/contracts/contract-1/tasks"},
		{method: http.MethodGet, path: "/v1/plans"},
		{method: http.MethodGet, path: "/v1/proposals"},
		{method: http.MethodGet, path: "/v1/tasks"},
		{method: http.MethodGet, path: "/v1/plans/leases"},
		{method: http.MethodGet, path: "/v1/queue/jobs"},
	} {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			response := doJSON(t, server.router, tt.method, tt.path, "")
			assertErrorCode(t, response, http.StatusNotFound, "not_found")
		})
	}
}

func createPlan(t *testing.T, server testServerDeps, contractID spine.ContractID) spine.WorkItemPlan {
	t.Helper()
	response := doJSON(t, server.router, http.MethodPost, "/v1/contracts/"+string(contractID)+"/plans", `{}`)
	if response.code != http.StatusCreated {
		t.Fatalf("create plan status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	var plan spine.WorkItemPlan
	decodeJSON(t, response.body, &plan)
	return plan
}

func storeApprovedPlanningFixture(t *testing.T, server testServerDeps) spine.ApprovedContract {
	t.Helper()

	approvedID := spine.ApprovedContractID("approved-contract-fixture")
	contractID := spine.ContractID("contract-fixture")
	seedID := spine.ContractSeedID("contract-seed-fixture")
	draftID := spine.ContractDraftID("contract-draft-fixture")
	approved := spine.ApprovedContract{
		ID:                 approvedID,
		OrganizationID:     "018f0000-0000-7000-8000-000000000002",
		ProjectID:          "018f0000-0000-7000-8000-000000000003",
		ContractID:         contractID,
		ContractDraftID:    draftID,
		ContractSeedID:     seedID,
		GoalID:             "goal-fixture",
		RepoBindingID:      "018f0000-0000-7000-8000-000000000004",
		Title:              "Fixture approved contract",
		IntentSummary:      "Fixture",
		Scope:              []string{"Fixture scope"},
		AcceptanceCriteria: []string{"Fixture acceptance"},
		ProofExpectations:  []string{"Fixture proof expectation"},
		ApprovedBy:         spine.ActorRef{Kind: "user", ID: "approver-fixture"},
		ApprovedAt:         testTime(),
		State:              spine.ApprovedContractStateApproved,
	}
	if err := server.approvedContracts.Create(context.Background(), approved); err != nil {
		t.Fatalf("approvedContracts.Create() error = %v", err)
	}
	if err := server.contracts.Create(context.Background(), spine.Contract{
		ID:                 contractID,
		OrganizationID:     approved.OrganizationID,
		ProjectID:          approved.ProjectID,
		RepoBindingID:      approved.RepoBindingID,
		GoalID:             approved.GoalID,
		State:              spine.ContractStateApproved,
		CurrentSeedID:      &seedID,
		CurrentDraftID:     &draftID,
		ApprovedSnapshotID: &approvedID,
		CreatedAt:          testTime(),
		UpdatedAt:          testTime(),
	}); err != nil {
		t.Fatalf("contracts.Create() error = %v", err)
	}
	return approved
}

func submitProposal(t *testing.T, server testServerDeps, planID spine.WorkItemPlanID, approvedContractID string) spine.WorkItemPlanProposal {
	t.Helper()
	lease := acquireLease(t, server)
	response := doJSON(t, server.router, http.MethodPost, "/v1/plans/"+string(planID)+"/proposals", validProposalJSON(approvedContractID, lease))
	if response.code != http.StatusCreated {
		t.Fatalf("submit proposal status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	var proposal spine.WorkItemPlanProposal
	decodeJSON(t, response.body, &proposal)
	return proposal
}

func acquireLease(t *testing.T, server testServerDeps) spine.WorkItemPlanLeaseCreated {
	t.Helper()
	response := doJSON(t, server.router, http.MethodPost, "/v1/plans/leases", `{"leased_by":{"kind":"worker","id":"planner-worker-1"}}`)
	if response.code != http.StatusCreated {
		t.Fatalf("acquire lease status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	var lease spine.WorkItemPlanLeaseCreated
	decodeJSON(t, response.body, &lease)
	if lease.LeaseToken == "" {
		t.Fatal("lease_token is empty")
	}
	return lease
}

func acceptProposal(t *testing.T, server testServerDeps, proposalID spine.WorkItemPlanProposalID) spine.WorkItemPlanAcceptanceResult {
	t.Helper()
	response := doJSON(t, server.router, http.MethodPost, "/v1/proposals/"+string(proposalID)+"/acceptance", `{}`)
	if response.code != http.StatusCreated {
		t.Fatalf("accept proposal status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	var result spine.WorkItemPlanAcceptanceResult
	decodeJSON(t, response.body, &result)
	return result
}

func validProposalJSON(approvedContractID string, lease spine.WorkItemPlanLeaseCreated) string {
	return proposalJSONWithLeaseValues(approvedContractID, string(lease.ID), lease.LeaseToken)
}

func proposalJSONWithTasks(approvedContractID string, lease spine.WorkItemPlanLeaseCreated, tasks string) string {
	return fmt.Sprintf(`{
  "lease_id": %q,
  "lease_token": %q,
  "submitted_by": {"kind": "worker", "id": "planner-worker-1"},
  "planner": {"kind": "goalrail_worker", "id": "planner-worker-1", "version": "0.1.0"},
  "source_snapshot_refs": [{"kind": "approved_contract", "id": %q}],
  "rationale": "Split independent refactor and coverage tasks.",
  "proposed_tasks": %s
}`, string(lease.ID), lease.LeaseToken, approvedContractID, tasks)
}

func proposalJSONWithLeaseValues(approvedContractID string, leaseID string, leaseToken string) string {
	return fmt.Sprintf(`{
  "lease_id": %q,
  "lease_token": %q,
  "submitted_by": {"kind": "worker", "id": "planner-worker-1"},
  "planner": {"kind": "goalrail_worker", "id": "planner-worker-1", "version": "0.1.0"},
  "source_snapshot_refs": [{"kind": "approved_contract", "id": %q}],
  "rationale": "Split independent refactor and coverage tasks.",
  "proposed_tasks": [
    {
      "title": "Refactor CSV export filter builder",
      "summary": "Extract duplicated filter construction logic.",
      "scope": ["Update export filter construction code"],
      "acceptance_refs": ["acceptance_criteria[0]"],
      "proof_expectation_refs": ["proof_expectations[0]"],
      "owner_hint": "",
      "order_index": 0,
      "source_refs": [{"kind": "approved_contract", "id": %q}]
    },
    {
      "title": "Cover CSV export filter behavior",
      "summary": "Add coverage for preserved filter behavior.",
      "scope": ["Add focused tests for CSV export filters"],
      "acceptance_refs": ["acceptance_criteria[0]"],
      "proof_expectation_refs": ["proof_expectations[0]"],
      "source_refs": [{"kind": "approved_contract", "id": %q}]
    }
  ]
}`, leaseID, leaseToken, approvedContractID, approvedContractID, approvedContractID)
}

func createApprovedContract(t *testing.T, server testServerDeps) spine.ApprovedContract {
	t.Helper()

	contract := createContract(t, server)
	ready := submitContractForApproval(t, server, contract.ID)
	approvedContract := approvePublicContract(t, server, ready.ID)
	if approvedContract.ApprovedSnapshotID == nil {
		t.Fatal("approved_snapshot_id is nil")
	}
	stored, ok, err := server.approvedContracts.Get(context.Background(), *approvedContract.ApprovedSnapshotID)
	if err != nil {
		t.Fatalf("approvedContracts.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("approved contract not stored")
	}
	return stored
}

func validHTTPApprovedContract() spine.ApprovedContract {
	return spine.ApprovedContract{
		ID:                 "approved-contract-1",
		OrganizationID:     "018f0000-0000-7000-8000-000000000002",
		ProjectID:          "018f0000-0000-7000-8000-000000000003",
		ContractID:         "contract-1",
		ContractDraftID:    "contract-draft-1",
		ContractSeedID:     "contract-seed-1",
		GoalID:             "goal-1",
		RepoBindingID:      "018f0000-0000-7000-8000-000000000004",
		Title:              "Refactor CSV export filters",
		IntentSummary:      "Current code duplicates filter logic.",
		Scope:              []string{"Refactor duplicate CSV export filter logic"},
		AcceptanceCriteria: []string{"Existing CSV export behavior is preserved"},
		ProofExpectations:  []string{"Provide evidence that acceptance criteria were checked."},
		ApprovedBy:         spine.ActorRef{Kind: "user", ID: "dev_approver"},
		ApprovedAt:         testTime(),
		SourceRefs: []spine.SourceRef{
			{Kind: approvedcontract.SourceRefKindContractDraft, ID: "contract-draft-1"},
			{Kind: "contract_seed", ID: "contract-seed-1"},
			{Kind: "goal", ID: "goal-1"},
		},
		State: spine.ApprovedContractStateApproved,
	}
}

func storeHTTPContractForApproved(t *testing.T, server testServerDeps, approved spine.ApprovedContract) {
	t.Helper()
	currentSeedID := approved.ContractSeedID
	currentDraftID := approved.ContractDraftID
	approvedSnapshotID := approved.ID
	contract := spine.Contract{
		ID:                 approved.ContractID,
		OrganizationID:     approved.OrganizationID,
		ProjectID:          approved.ProjectID,
		RepoBindingID:      approved.RepoBindingID,
		GoalID:             approved.GoalID,
		State:              spine.ContractStateApproved,
		CurrentSeedID:      &currentSeedID,
		CurrentDraftID:     &currentDraftID,
		ApprovedSnapshotID: &approvedSnapshotID,
		CreatedAt:          testTime(),
		UpdatedAt:          testTime(),
	}
	if err := server.contracts.Create(context.Background(), contract); err != nil {
		t.Fatalf("contracts.Create() error = %v", err)
	}
}

func assertNoForbiddenWorkItemSideEffects(t *testing.T, events []spine.Event) {
	t.Helper()

	forbidden := map[string]bool{
		"work_item.assigned":          true,
		"work_item.claimed":           true,
		"checkout.created":            true,
		"run.created":                 true,
		"run.started":                 true,
		"receipt.created":             true,
		"receipt.submitted":           true,
		"execution_receipt.submitted": true,
		"decision.created":            true,
		"gate_decision.created":       true,
		"gate.decision_written":       true,
		"proof.created":               true,
	}
	for _, event := range events {
		if forbidden[event.Type] {
			t.Fatalf("forbidden event type appended: %s", event.Type)
		}
	}
}

func assertNoForbiddenRuntimeSideEffects(t *testing.T, events []spine.Event) {
	t.Helper()

	forbidden := map[string]bool{
		"work_item.assigned":          true,
		"work_item.claimed":           true,
		"run.created":                 true,
		"run.started":                 true,
		"receipt.created":             true,
		"receipt.submitted":           true,
		"execution_receipt.submitted": true,
		"decision.created":            true,
		"gate_decision.created":       true,
		"gate.decision_written":       true,
		"proof.created":               true,
	}
	for _, event := range events {
		if forbidden[event.Type] {
			t.Fatalf("forbidden runtime event type appended: %s", event.Type)
		}
	}
}

func assertNoForbiddenPostRunSideEffects(t *testing.T, events []spine.Event) {
	t.Helper()

	forbidden := map[string]bool{
		"work_item.assigned":          true,
		"work_item.claimed":           true,
		"receipt.created":             true,
		"receipt.submitted":           true,
		"execution_receipt.submitted": true,
		"decision.created":            true,
		"gate_decision.created":       true,
		"gate.decision_written":       true,
		"proof.created":               true,
	}
	for _, event := range events {
		if forbidden[event.Type] {
			t.Fatalf("forbidden post-run event type appended: %s", event.Type)
		}
	}
}

func assertNoForbiddenPostReceiptSideEffects(t *testing.T, events []spine.Event) {
	t.Helper()

	forbidden := map[string]bool{
		"work_item.assigned":    true,
		"work_item.claimed":     true,
		"decision.created":      true,
		"gate_decision.created": true,
		"gate.decision_written": true,
		"proof.created":         true,
	}
	for _, event := range events {
		if forbidden[event.Type] {
			t.Fatalf("forbidden post-receipt event type appended: %s", event.Type)
		}
	}
}

func assertNoHiddenContext(t *testing.T, body string) {
	t.Helper()
	for _, hiddenField := range []string{"\"organization_id\"", "\"project_id\""} {
		if strings.Contains(body, hiddenField) {
			t.Fatalf("response includes hidden field %s", hiddenField)
		}
	}
	for _, forbiddenField := range []string{"\"run_id\"", "\"receipt_id\"", "\"gate_decision_id\"", "\"proof_id\""} {
		if strings.Contains(body, forbiddenField) {
			t.Fatalf("response includes forbidden field %s", forbiddenField)
		}
	}
}

func assertErrorCode(t *testing.T, response routeResponse, status int, code string) {
	t.Helper()
	if response.code != status {
		t.Fatalf("status = %d, want %d: %s", response.code, status, response.body)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != code {
		t.Fatalf("error code = %q, want %q", body.Error.Code, code)
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

func TestWorkItemResponseJSONDoesNotExposeContext(t *testing.T) {
	item := spine.WorkItem{
		ID:             "work-item-1",
		OrganizationID: "organization-1",
		ProjectID:      "project-1",
	}
	encoded, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if strings.Contains(string(encoded), "organization_id") || strings.Contains(string(encoded), "project_id") {
		t.Fatalf("encoded work item exposes internal context: %s", encoded)
	}
}
