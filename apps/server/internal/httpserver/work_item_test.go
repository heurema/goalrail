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

	response := doJSON(t, server.router, http.MethodPost, "/v1/proposals/"+string(proposal.ID)+"/acceptance", `{"accepted_by":{"kind":"user","id":"acceptor-1"}}`)
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
	if got := countEventType(server.events.Events(), workitem.EventTypeWorkItemCreated); got != 2 {
		t.Fatalf("work_item.created events = %d, want 2", got)
	}
}

func TestPostProposalAcceptanceRejectsDuplicateAndMissingActor(t *testing.T) {
	t.Run("duplicate acceptance", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		plan := createPlan(t, server, approved.ContractID)
		proposal := submitProposal(t, server, plan.ID, string(approved.ID))
		acceptProposal(t, server, proposal.ID)

		response := doJSON(t, server.router, http.MethodPost, "/v1/proposals/"+string(proposal.ID)+"/acceptance", `{"accepted_by":{"kind":"user","id":"acceptor-1"}}`)
		assertErrorCode(t, response, http.StatusConflict, "already_accepted")
	})

	t.Run("missing accepted_by", func(t *testing.T) {
		server := testServer(t)
		approved := createApprovedContract(t, server)
		plan := createPlan(t, server, approved.ContractID)
		proposal := submitProposal(t, server, plan.ID, string(approved.ID))

		response := doJSON(t, server.router, http.MethodPost, "/v1/proposals/"+string(proposal.ID)+"/acceptance", `{}`)
		assertErrorCode(t, response, http.StatusBadRequest, "validation_failed")
	})
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
	response := doJSON(t, server.router, http.MethodPost, "/v1/proposals/"+string(proposalID)+"/acceptance", `{"accepted_by":{"kind":"user","id":"acceptor-1"}}`)
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
		"run.started":           true,
		"receipt.submitted":     true,
		"gate.decision_written": true,
		"proof.created":         true,
	}
	for _, event := range events {
		if forbidden[event.Type] {
			t.Fatalf("forbidden event type appended: %s", event.Type)
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
