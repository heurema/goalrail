package httpserver_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/execution"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/workitem"
)

func TestAgentPullLoopServerSmokeThroughWorkItemPlanned(t *testing.T) {
	server := testServer(t)

	intakeID := createIntake(t, server, validIntakeJSON)
	created := promoteIntake(t, server, intakeID)

	continuation := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/continuation", "")
	if continuation.code != http.StatusOK {
		t.Fatalf("continuation status = %d, want %d: %s", continuation.code, http.StatusOK, continuation.body)
	}
	var continued struct {
		GoalID               spine.GoalID               `json:"goal_id"`
		State                spine.GoalState            `json:"state"`
		ClarificationRequest spine.ClarificationRequest `json:"clarification_request"`
	}
	decodeJSON(t, continuation.body, &continued)
	if continued.GoalID != created.ID || continued.State != spine.GoalStateNeedsClarification {
		t.Fatalf("continuation = %#v, want needs_clarification for goal", continued)
	}
	if continued.ClarificationRequest.ID == "" || len(continued.ClarificationRequest.Questions) == 0 {
		t.Fatalf("clarification request = %#v, want open request with questions", continued.ClarificationRequest)
	}

	answer := doJSON(
		t,
		server.router,
		http.MethodPost,
		"/v1/clarifications/"+string(continued.ClarificationRequest.ID)+"/answers/continuation",
		workAnswerSubmissionJSONWithValues(continued.ClarificationRequest, map[spine.ClarificationMapsTo]string{
			spine.ClarificationMapsToGoalScopeHint:      "Refactor duplicate CSV export filter logic",
			spine.ClarificationMapsToGoalAcceptanceHint: "Existing CSV export behavior is preserved",
		}),
	)
	if answer.code != http.StatusOK {
		t.Fatalf("answer continuation status = %d, want %d: %s", answer.code, http.StatusOK, answer.body)
	}
	var answered struct {
		GoalID spine.GoalID    `json:"goal_id"`
		State  spine.GoalState `json:"state"`
	}
	decodeJSON(t, answer.body, &answered)
	if answered.GoalID != created.ID || answered.State != spine.GoalStateReadyForContractSeed {
		t.Fatalf("answer continuation = %#v, want ready_for_contract_seed", answered)
	}

	contractCreate := doJSON(t, server.router, http.MethodPost, "/v1/contracts", contractCreateJSONWithContext(created.ID, created.ProjectID, created.RepoBindingID))
	if contractCreate.code != http.StatusCreated {
		t.Fatalf("contract create status = %d, want %d: %s", contractCreate.code, http.StatusCreated, contractCreate.body)
	}
	var contract spine.Contract
	decodeJSON(t, contractCreate.body, &contract)
	if contract.State != spine.ContractStateDraft {
		t.Fatalf("contract state = %q, want draft", contract.State)
	}

	contractUpdate := doJSON(
		t,
		server.router,
		http.MethodPatch,
		"/v1/contracts/"+string(contract.ID),
		updateDraftJSONWithContext(string(contract.ProjectID), string(contract.RepoBindingID), `{"proposed_scope":["Refactor duplicate CSV export filter logic"],"proposed_acceptance_criteria":["Existing CSV export behavior is preserved"],"proposed_proof_expectations":["CLI and server tests pass"]}`),
	)
	if contractUpdate.code != http.StatusOK {
		t.Fatalf("contract update status = %d, want %d: %s", contractUpdate.code, http.StatusOK, contractUpdate.body)
	}
	decodeJSON(t, contractUpdate.body, &contract)
	if contract.State != spine.ContractStateDraft {
		t.Fatalf("updated contract state = %q, want draft", contract.State)
	}

	submit := doJSON(t, server.router, http.MethodPost, "/v1/contracts/"+string(contract.ID)+"/submissions", readyForApprovalJSONWithContext(string(contract.ProjectID), string(contract.RepoBindingID)))
	if submit.code != http.StatusOK {
		t.Fatalf("contract submit status = %d, want %d: %s", submit.code, http.StatusOK, submit.body)
	}
	decodeJSON(t, submit.body, &contract)
	if contract.State != spine.ContractStateReadyForApproval {
		t.Fatalf("submitted contract state = %q, want ready_for_approval", contract.State)
	}

	approve := doJSON(t, server.router, http.MethodPost, "/v1/contracts/"+string(contract.ID)+"/approvals", approveContractJSONWithContext(string(contract.ProjectID), string(contract.RepoBindingID)))
	if approve.code != http.StatusCreated {
		t.Fatalf("contract approve status = %d, want %d: %s", approve.code, http.StatusCreated, approve.body)
	}
	decodeJSON(t, approve.body, &contract)
	if contract.State != spine.ContractStateApproved || contract.ApprovedSnapshotID == nil {
		t.Fatalf("approved contract = %#v, want approved state with snapshot", contract)
	}
	if len(server.approvedContracts.approved) != 1 {
		t.Fatalf("approved contracts = %d, want 1", len(server.approvedContracts.approved))
	}
	if len(server.workItems.items) != 0 {
		t.Fatalf("work items = %d, want 0 after approval", len(server.workItems.items))
	}
	assertNoPlanningStores(t, server)
	assertNoForbiddenApprovalSideEffects(t, server.events.Events())

	planCreate := doJSON(t, server.router, http.MethodPost, "/v1/contracts/"+string(contract.ID)+"/plans", planCreateJSONWithContext(string(contract.ProjectID), string(contract.RepoBindingID)))
	if planCreate.code != http.StatusCreated {
		t.Fatalf("work plan status = %d, want %d: %s", planCreate.code, http.StatusCreated, planCreate.body)
	}
	var plan spine.WorkItemPlan
	decodeJSON(t, planCreate.body, &plan)
	if plan.ContractID != contract.ID || plan.State != spine.WorkItemPlanStateQueued {
		t.Fatalf("work plan = %#v, want queued plan for contract %q", plan, contract.ID)
	}

	proposal := submitProposal(t, server, plan.ID, string(*contract.ApprovedSnapshotID))
	if proposal.PlanID != plan.ID || proposal.State != spine.WorkItemProposalStateSubmitted || len(proposal.ProposedTasks) == 0 {
		t.Fatalf("proposal = %#v, want submitted proposal with tasks", proposal)
	}
	if len(server.workItems.items) != 0 {
		t.Fatalf("work items = %d, want 0 before proposal acceptance", len(server.workItems.items))
	}

	statusResponse := doJSON(t, server.router, http.MethodPost, "/v1/plans/"+string(plan.ID)+"/status", planCreateJSONWithContext(string(contract.ProjectID), string(contract.RepoBindingID)))
	if statusResponse.code != http.StatusOK {
		t.Fatalf("plan status code = %d, want %d: %s", statusResponse.code, http.StatusOK, statusResponse.body)
	}
	for _, forbidden := range []string{"\"lease_token\"", "\"lease_token_hash\""} {
		if strings.Contains(statusResponse.body, forbidden) {
			t.Fatalf("plan status exposes worker secret field %s: %s", forbidden, statusResponse.body)
		}
	}
	var status spine.WorkItemPlanStatus
	decodeJSON(t, statusResponse.body, &status)
	if status.Plan.ID != plan.ID || status.Plan.State != spine.WorkItemPlanStateProposalSubmitted {
		t.Fatalf("plan status = %#v, want proposal_submitted", status.Plan)
	}
	if status.Proposal == nil || status.Proposal.ID != proposal.ID || len(status.Proposal.ProposedTasks) == 0 {
		t.Fatalf("status proposal = %#v, want submitted proposal details", status.Proposal)
	}

	accept := doJSON(t, server.router, http.MethodPost, "/v1/proposals/"+string(proposal.ID)+"/acceptance", planCreateJSONWithContext(string(contract.ProjectID), string(contract.RepoBindingID)))
	if accept.code != http.StatusCreated {
		t.Fatalf("proposal accept status = %d, want %d: %s", accept.code, http.StatusCreated, accept.body)
	}
	var accepted spine.WorkItemPlanAcceptanceResult
	decodeJSON(t, accept.body, &accepted)
	if accepted.State != spine.WorkItemProposalStateAccepted || accepted.PlanID != plan.ID || accepted.ProposalID != proposal.ID || len(accepted.CreatedTaskIDs) != 2 {
		t.Fatalf("accepted proposal = %#v, want accepted state and planned tasks", accepted)
	}
	for _, taskID := range accepted.CreatedTaskIDs {
		task, ok, err := server.workItems.Get(context.Background(), taskID)
		if err != nil {
			t.Fatalf("workItems.Get(%s) error = %v", taskID, err)
		}
		if !ok {
			t.Fatalf("work item %s not stored", taskID)
		}
		if task.Status != spine.WorkItemStatusPlanned || task.PlanID != plan.ID || task.ProposalID != proposal.ID {
			t.Fatalf("work item trace = %#v, want planned task from plan/proposal", task)
		}
	}
	if got := countEventType(server.events.Events(), workitem.EventTypeWorkItemCreated); got != len(accepted.CreatedTaskIDs) {
		t.Fatalf("work_item.created events = %d, want %d", got, len(accepted.CreatedTaskIDs))
	}
	assertNoForbiddenWorkItemSideEffects(t, server.events.Events())

	taskID := accepted.CreatedTaskIDs[0]
	contextProjectID := created.ProjectID
	contextRepoBindingID := created.RepoBindingID
	checkoutJobResponse := doJSON(t, server.router, http.MethodPost, "/v1/tasks/"+string(taskID)+"/checkout-jobs", planCreateJSONWithContext(string(contextProjectID), string(contextRepoBindingID)))
	if checkoutJobResponse.code != http.StatusCreated {
		t.Fatalf("checkout job status = %d, want %d: %s", checkoutJobResponse.code, http.StatusCreated, checkoutJobResponse.body)
	}
	for _, forbidden := range []string{"\"lease_token\"", "\"lease_token_hash\"", "\"run_id\"", "\"proof_id\""} {
		if strings.Contains(checkoutJobResponse.body, forbidden) {
			t.Fatalf("checkout job response exposes forbidden field %s: %s", forbidden, checkoutJobResponse.body)
		}
	}
	var checkoutJob spine.CheckoutJob
	decodeJSON(t, checkoutJobResponse.body, &checkoutJob)
	if checkoutJob.TaskID != taskID || checkoutJob.State != spine.CheckoutJobStateQueued {
		t.Fatalf("checkout job = %#v, want queued job for task %q", checkoutJob, taskID)
	}
	if checkoutJob.ContractID != contract.ID || checkoutJob.PlanID != plan.ID || checkoutJob.ProposalID != proposal.ID || checkoutJob.RepoBindingID != contextRepoBindingID {
		t.Fatalf("checkout job trace = %#v, want contract/plan/proposal/repo trace", checkoutJob)
	}
	if checkoutJob.Instruction.JobID != checkoutJob.ID || checkoutJob.Instruction.TaskID != taskID || checkoutJob.Instruction.RepoBindingID != contextRepoBindingID {
		t.Fatalf("checkout instruction identity = %#v, want job/task/repo-bound instruction", checkoutJob.Instruction)
	}
	if checkoutJob.Instruction.RepositoryFullName == "" || checkoutJob.Instruction.WorkflowBaseBranch == "" || checkoutJob.Instruction.RawSourceUploaded {
		t.Fatalf("checkout instruction repository/raw-source = %#v, want metadata-only instruction", checkoutJob.Instruction)
	}

	secondCheckoutJobResponse := doJSON(t, server.router, http.MethodPost, "/v1/tasks/"+string(taskID)+"/checkout-jobs", planCreateJSONWithContext(string(contextProjectID), string(contextRepoBindingID)))
	if secondCheckoutJobResponse.code != http.StatusOK {
		t.Fatalf("second checkout job status = %d, want %d: %s", secondCheckoutJobResponse.code, http.StatusOK, secondCheckoutJobResponse.body)
	}
	var existingCheckoutJob spine.CheckoutJob
	decodeJSON(t, secondCheckoutJobResponse.body, &existingCheckoutJob)
	if existingCheckoutJob.ID != checkoutJob.ID || len(server.checkoutJobs.jobs) != 1 {
		t.Fatalf("existing checkout job id/count = %q/%d, want %q/1", existingCheckoutJob.ID, len(server.checkoutJobs.jobs), checkoutJob.ID)
	}

	leaseResponse := doJSON(t, server.router, http.MethodPost, "/v1/checkout-jobs/leases", fmt.Sprintf(`{"project_id":%q,"repo_binding_id":%q,"runner_id":"runner-smoke"}`, contextProjectID, contextRepoBindingID))
	if leaseResponse.code != http.StatusCreated {
		t.Fatalf("checkout lease status = %d, want %d: %s", leaseResponse.code, http.StatusCreated, leaseResponse.body)
	}
	if strings.Contains(leaseResponse.body, "lease_token_hash") {
		t.Fatalf("checkout lease response exposes token hash: %s", leaseResponse.body)
	}
	var checkoutLease spine.CheckoutJobLeaseCreated
	decodeJSON(t, leaseResponse.body, &checkoutLease)
	if checkoutLease.JobID != checkoutJob.ID || checkoutLease.TaskID != taskID || checkoutLease.RunnerID != "runner-smoke" || checkoutLease.LeaseToken == "" {
		t.Fatalf("checkout lease = %#v, want scoped lease with raw token only in lease response", checkoutLease)
	}
	if checkoutLease.Instruction.RawSourceUploaded || checkoutLease.Instruction.RepoBindingID != contextRepoBindingID {
		t.Fatalf("checkout lease instruction = %#v, want no raw source and matching repo", checkoutLease.Instruction)
	}

	receiptResponse := doJSON(t, server.router, http.MethodPost, "/v1/checkout-jobs/"+string(checkoutJob.ID)+"/receipts", fmt.Sprintf(`{"lease_token":%q,"runner_id":"runner-smoke","workspace_ref":"mounted:/workspace/goalrail#checkout_job=%s;task=%s;repo_binding=%s","commit_sha":"abc123","baseline_id":"baseline-smoke","overlay_id":"overlay-smoke","dirty":false,"partial":false,"raw_source_uploaded":false}`, checkoutLease.LeaseToken, checkoutJob.ID, taskID, contextRepoBindingID))
	if receiptResponse.code != http.StatusCreated {
		t.Fatalf("checkout receipt status = %d, want %d: %s", receiptResponse.code, http.StatusCreated, receiptResponse.body)
	}
	for _, forbidden := range []string{"\"lease_token\"", "\"lease_token_hash\"", "\"run_id\"", "\"proof_id\""} {
		if strings.Contains(receiptResponse.body, forbidden) {
			t.Fatalf("checkout receipt response exposes forbidden field %s: %s", forbidden, receiptResponse.body)
		}
	}
	var receipt spine.CheckoutReceipt
	decodeJSON(t, receiptResponse.body, &receipt)
	if receipt.JobID != checkoutJob.ID || receipt.TaskID != taskID || receipt.RunnerID != "runner-smoke" || receipt.RawSourceUploaded {
		t.Fatalf("checkout receipt = %#v, want runner workspace metadata without raw source", receipt)
	}
	if receipt.BaselineID != "baseline-smoke" || receipt.OverlayID != "overlay-smoke" || receipt.CommitSHA != "abc123" {
		t.Fatalf("checkout receipt evidence fields = %#v, want smoke metadata", receipt)
	}
	storedCheckoutJob := server.checkoutJobs.jobs[checkoutJob.ID]
	if storedCheckoutJob.State != spine.CheckoutJobStateReceiptSubmitted {
		t.Fatalf("checkout job state = %q, want receipt_submitted", storedCheckoutJob.State)
	}
	storedTask, ok, err := server.workItems.Get(context.Background(), taskID)
	if err != nil || !ok {
		t.Fatalf("workItems.Get(%s) = %#v/%v/%v", taskID, storedTask, ok, err)
	}
	if storedTask.Status != spine.WorkItemStatusPlanned {
		t.Fatalf("work item status = %q, want planned after checkout receipt", storedTask.Status)
	}
	if len(server.checkoutReceipts.receipts) != 1 {
		t.Fatalf("checkout receipts = %d, want 1", len(server.checkoutReceipts.receipts))
	}
	assertNoForbiddenRuntimeSideEffects(t, server.events.Events())

	executionJobResponse := doJSON(t, server.router, http.MethodPost, "/v1/tasks/"+string(taskID)+"/execution-jobs", fmt.Sprintf(`{"project_id":%q,"repo_binding_id":%q,"checkout_receipt_id":%q}`, contextProjectID, contextRepoBindingID, receipt.ID))
	if executionJobResponse.code != http.StatusCreated {
		t.Fatalf("execution job status = %d, want %d: %s", executionJobResponse.code, http.StatusCreated, executionJobResponse.body)
	}
	for _, forbidden := range []string{"\"lease_token\"", "\"lease_token_hash\"", "\"run_id\"", "\"execution_receipt_id\"", "\"gate_decision_id\"", "\"proof_id\""} {
		if strings.Contains(executionJobResponse.body, forbidden) {
			t.Fatalf("execution job response exposes forbidden field %s: %s", forbidden, executionJobResponse.body)
		}
	}
	var executionJob spine.ExecutionJob
	decodeJSON(t, executionJobResponse.body, &executionJob)
	if executionJob.TaskID != taskID || executionJob.CheckoutReceiptID != receipt.ID || executionJob.CheckoutJobID != checkoutJob.ID || executionJob.State != spine.ExecutionJobStateQueued {
		t.Fatalf("execution job = %#v, want queued job for task/checkout receipt", executionJob)
	}
	if executionJob.ContractID != contract.ID || executionJob.PlanID != plan.ID || executionJob.ProposalID != proposal.ID || executionJob.RepoBindingID != contextRepoBindingID {
		t.Fatalf("execution job trace = %#v, want contract/plan/proposal/repo trace", executionJob)
	}
	storedTaskAfterExecutionPrepare, ok, err := server.workItems.Get(context.Background(), taskID)
	if err != nil || !ok {
		t.Fatalf("workItems.Get(%s) after execution prepare = %#v/%v/%v", taskID, storedTaskAfterExecutionPrepare, ok, err)
	}
	if storedTaskAfterExecutionPrepare.Status != spine.WorkItemStatusPlanned {
		t.Fatalf("work item status = %q, want planned after execution job prepare", storedTaskAfterExecutionPrepare.Status)
	}
	if len(server.executionJobs.jobs) != 1 {
		t.Fatalf("execution jobs = %d, want 1", len(server.executionJobs.jobs))
	}
	if got := countEventType(server.events.Events(), execution.EventTypeExecutionJobCreated); got != 1 {
		t.Fatalf("execution_job.created events = %d, want 1", got)
	}

	secondExecutionJobResponse := doJSON(t, server.router, http.MethodPost, "/v1/tasks/"+string(taskID)+"/execution-jobs", fmt.Sprintf(`{"project_id":%q,"repo_binding_id":%q,"checkout_receipt_id":%q}`, contextProjectID, contextRepoBindingID, receipt.ID))
	if secondExecutionJobResponse.code != http.StatusOK {
		t.Fatalf("second execution job status = %d, want %d: %s", secondExecutionJobResponse.code, http.StatusOK, secondExecutionJobResponse.body)
	}
	var existingExecutionJob spine.ExecutionJob
	decodeJSON(t, secondExecutionJobResponse.body, &existingExecutionJob)
	if existingExecutionJob.ID != executionJob.ID || len(server.executionJobs.jobs) != 1 {
		t.Fatalf("existing execution job id/count = %q/%d, want %q/1", existingExecutionJob.ID, len(server.executionJobs.jobs), executionJob.ID)
	}
	if got := countEventType(server.events.Events(), execution.EventTypeExecutionJobCreated); got != 1 {
		t.Fatalf("execution_job.created events = %d, want no duplicate event", got)
	}
	assertNoForbiddenRuntimeSideEffects(t, server.events.Events())

	executionLeaseResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/leases", fmt.Sprintf(`{"project_id":%q,"repo_binding_id":%q,"runner_id":"runner-smoke"}`, contextProjectID, contextRepoBindingID))
	if executionLeaseResponse.code != http.StatusCreated {
		t.Fatalf("execution lease status = %d, want %d: %s", executionLeaseResponse.code, http.StatusCreated, executionLeaseResponse.body)
	}
	for _, forbidden := range []string{"\"lease_token_hash\"", "\"run_id\"", "\"execution_receipt_id\"", "\"gate_decision_id\"", "\"proof_id\""} {
		if strings.Contains(executionLeaseResponse.body, forbidden) {
			t.Fatalf("execution lease response exposes forbidden field %s: %s", forbidden, executionLeaseResponse.body)
		}
	}
	var executionLease spine.ExecutionJobLeaseCreated
	decodeJSON(t, executionLeaseResponse.body, &executionLease)
	if executionLease.ID == "" || executionLease.ExecutionJobID != executionJob.ID || executionLease.TaskID != taskID || executionLease.CheckoutReceiptID != receipt.ID || executionLease.RunnerID != "runner-smoke" || executionLease.LeaseToken == "" {
		t.Fatalf("execution lease = %#v, want scoped execution lease with one-time token", executionLease)
	}
	if executionLease.ExecutionJob.State != spine.ExecutionJobStateLeased {
		t.Fatalf("leased execution job state = %q, want leased", executionLease.ExecutionJob.State)
	}
	if len(server.runs.runs) != 0 {
		t.Fatalf("runs = %d, want no Run after execution lease acquisition", len(server.runs.runs))
	}
	if got := countEventType(server.events.Events(), execution.EventTypeExecutionJobLeased); got != 1 {
		t.Fatalf("execution_job.leased events = %d, want 1", got)
	}

	runResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(executionJob.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-smoke"}`, executionLease.ID, executionLease.LeaseToken))
	if runResponse.code != http.StatusCreated {
		t.Fatalf("run start status = %d, want %d: %s", runResponse.code, http.StatusCreated, runResponse.body)
	}
	for _, forbidden := range []string{"\"lease_token\"", "\"lease_token_hash\"", "\"execution_receipt_id\"", "\"gate_decision_id\"", "\"proof_id\""} {
		if strings.Contains(runResponse.body, forbidden) {
			t.Fatalf("run start response exposes forbidden field %s: %s", forbidden, runResponse.body)
		}
	}
	var run spine.Run
	decodeJSON(t, runResponse.body, &run)
	if run.ID == "" || run.ExecutionJobID != executionJob.ID || run.ExecutionLeaseID != executionLease.ID || run.TaskID != taskID || run.CheckoutReceiptID != receipt.ID || run.RunnerID != "runner-smoke" || run.State != spine.RunStateStarted {
		t.Fatalf("run = %#v, want started Run bound to execution lease", run)
	}
	if got := server.executionJobs.jobs[executionJob.ID].State; got != spine.ExecutionJobStateRunStarted {
		t.Fatalf("execution job state = %q, want run_started", got)
	}
	if got := countEventType(server.events.Events(), execution.EventTypeRunStarted); got != 1 {
		t.Fatalf("run.started events = %d, want 1", got)
	}
	storedTaskAfterRunStart, ok, err := server.workItems.Get(context.Background(), taskID)
	if err != nil || !ok {
		t.Fatalf("workItems.Get(%s) after run start = %#v/%v/%v", taskID, storedTaskAfterRunStart, ok, err)
	}
	if storedTaskAfterRunStart.Status != spine.WorkItemStatusPlanned {
		t.Fatalf("work item status = %q, want planned after run start", storedTaskAfterRunStart.Status)
	}
	if len(server.runs.runs) != 1 {
		t.Fatalf("runs = %d, want exactly one started Run", len(server.runs.runs))
	}

	secondRunResponse := doJSON(t, server.router, http.MethodPost, "/v1/execution-jobs/"+string(executionJob.ID)+"/runs", fmt.Sprintf(`{"lease_id":%q,"lease_token":%q,"runner_id":"runner-smoke"}`, executionLease.ID, executionLease.LeaseToken))
	if secondRunResponse.code != http.StatusOK {
		t.Fatalf("second run start status = %d, want %d: %s", secondRunResponse.code, http.StatusOK, secondRunResponse.body)
	}
	var existingRun spine.Run
	decodeJSON(t, secondRunResponse.body, &existingRun)
	if existingRun.ID != run.ID || len(server.runs.runs) != 1 {
		t.Fatalf("existing run id/count = %q/%d, want %q/1", existingRun.ID, len(server.runs.runs), run.ID)
	}
	assertNoForbiddenPostRunSideEffects(t, server.events.Events())

	executionReceiptResponse := doJSON(t, server.router, http.MethodPost, "/v1/runs/"+string(run.ID)+"/receipts", executionReceiptBody(executionJob.ID, executionLease.ID, executionLease.LeaseToken, "runner-smoke", "mounted:/workspace/goalrail#run="+string(run.ID), "abc123", false))
	if executionReceiptResponse.code != http.StatusCreated {
		t.Fatalf("execution receipt status = %d, want %d: %s", executionReceiptResponse.code, http.StatusCreated, executionReceiptResponse.body)
	}
	for _, forbidden := range []string{"\"lease_token\"", "\"lease_token_hash\"", "\"gate_decision_id\"", "\"proof_id\""} {
		if strings.Contains(executionReceiptResponse.body, forbidden) {
			t.Fatalf("execution receipt response exposes forbidden field %s: %s", forbidden, executionReceiptResponse.body)
		}
	}
	var executionReceipt spine.ExecutionReceipt
	decodeJSON(t, executionReceiptResponse.body, &executionReceipt)
	if executionReceipt.ID == "" || executionReceipt.RunID != run.ID || executionReceipt.ExecutionJobID != executionJob.ID || executionReceipt.ExecutionLeaseID != executionLease.ID || executionReceipt.TaskID != taskID || executionReceipt.CheckoutReceiptID != receipt.ID || executionReceipt.RunnerID != "runner-smoke" {
		t.Fatalf("execution receipt = %#v, want receipt bound to run/job/lease/task", executionReceipt)
	}
	if executionReceipt.ExecutionMode != spine.ExecutionReceiptModeNoCommand || executionReceipt.ProcessStatus != spine.ExecutionReceiptStatusNotExecuted || executionReceipt.ExitCode != nil || executionReceipt.RawSourceUploaded {
		t.Fatalf("execution receipt mode/status = %#v, want no-command metadata-only receipt", executionReceipt)
	}
	if executionReceipt.NextAction.Kind != spine.ExecutionReceiptNextActionGateReview || executionReceipt.NextAction.Available || executionReceipt.NextAction.PlannedSlice != spine.ExecutionReceiptNextActionPlannedSlice {
		t.Fatalf("execution receipt next_action = %#v, want unavailable gate_review", executionReceipt.NextAction)
	}
	if got := server.runs.runs[run.ID].State; got != spine.RunStateReceiptSubmitted {
		t.Fatalf("run state = %q, want receipt_submitted", got)
	}
	if got := server.executionJobs.jobs[executionJob.ID].State; got != spine.ExecutionJobStateReceiptSubmitted {
		t.Fatalf("execution job state = %q, want receipt_submitted", got)
	}
	if len(server.executionReceipts.receipts) != 1 {
		t.Fatalf("execution receipts = %d, want 1", len(server.executionReceipts.receipts))
	}
	storedTaskAfterExecutionReceipt, ok, err := server.workItems.Get(context.Background(), taskID)
	if err != nil || !ok {
		t.Fatalf("workItems.Get(%s) after execution receipt = %#v/%v/%v", taskID, storedTaskAfterExecutionReceipt, ok, err)
	}
	if storedTaskAfterExecutionReceipt.Status != spine.WorkItemStatusPlanned {
		t.Fatalf("work item status = %q, want planned after execution receipt", storedTaskAfterExecutionReceipt.Status)
	}
	if got := countEventType(server.events.Events(), execution.EventTypeExecutionReceiptSubmitted); got != 1 {
		t.Fatalf("execution_receipt.submitted events = %d, want 1", got)
	}
	assertNoForbiddenPostReceiptSideEffects(t, server.events.Events())
}

func contractCreateJSONWithContext(goalID spine.GoalID, projectID spine.ProjectID, repoBindingID spine.RepoBindingID) string {
	return fmt.Sprintf(`{"goal_id":%q,"project_id":%q,"repo_binding_id":%q}`, goalID, projectID, repoBindingID)
}

func planCreateJSONWithContext(projectID string, repoBindingID string) string {
	return fmt.Sprintf(`{"project_id":%q,"repo_binding_id":%q}`, projectID, repoBindingID)
}
