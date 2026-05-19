package contractcmd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/cli/internal/authstore"
	contractcmd "github.com/heurema/goalrail/apps/cli/internal/contractcmd"
	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
	"github.com/heurema/goalrail/apps/cli/internal/projectconfig"
	"github.com/heurema/goalrail/apps/cli/internal/spine"
	"github.com/heurema/goalrail/apps/cli/internal/term"
	"github.com/heurema/goalrail/apps/cli/internal/workcmd"
)

const (
	smokeOrganizationID         = "018f0000-0000-7000-8000-000000000002"
	smokeProjectID              = "018f0000-0000-7000-8000-000000000003"
	smokeRepoBindingID          = "018f0000-0000-7000-8000-000000000004"
	smokeIntakeID               = "018f0000-0000-7000-8000-000000000005"
	smokeGoalID                 = "018f0000-0000-7000-8000-000000000006"
	smokeClarificationRequestID = "018f0000-0000-7000-8000-000000000007"
	smokeContractID             = "018f0000-0000-7000-8000-000000000009"
	smokeApprovedSnapshotID     = "018f0000-0000-7000-8000-000000000010"
	smokePlanID                 = "018f0000-0000-7000-8000-000000000301"
	smokeProposalID             = "018f0000-0000-7000-8000-000000000302"
	smokeWorkItemID             = "018f0000-0000-7000-8000-000000000401"
	smokeCheckoutJobID          = "018f0000-0000-7000-8000-000000000501"
	smokeCheckoutReceiptID      = "018f0000-0000-7000-8000-000000000502"
	smokeExecutionJobID         = "018f0000-0000-7000-8000-000000000601"
)

func TestAgentPullLoopCLISmokeThroughWorkItemPlanned(t *testing.T) {
	requireSmokeGit(t)

	server := newPullLoopSmokeServer(t)
	defer server.Close()

	repoDir := setupSmokeGitRepo(t)
	writeSmokeProjectConfig(t, repoDir, server.URL)
	store := smokeSessionStore{session: validSmokeSession(server.URL)}

	requestsBeforeApprovalWithoutFlag := server.TotalRequests()
	err := runSmokeContractCommand(t, repoDir, store, "", nil, "approve", "--contract-id", smokeContractID, "--format", "json")
	if err == nil {
		t.Fatal("contract approve without --confirm-user-approval error = nil, want usage error")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("contract approve without confirmation exit code = %d, want usage", got)
	}
	if !strings.Contains(err.Error(), "--confirm-user-approval") {
		t.Fatalf("contract approve without confirmation error = %q, want confirmation flag hint", err.Error())
	}
	if got := server.TotalRequests(); got != requestsBeforeApprovalWithoutFlag {
		t.Fatalf("server requests after approve without confirmation = %d, want %d", got, requestsBeforeApprovalWithoutFlag)
	}

	requestsBeforeProposalAcceptWithoutFlag := server.TotalRequests()
	err = runSmokeWorkCommand(t, repoDir, store, "", nil, "proposal", "accept", "--proposal-id", smokeProposalID, "--format", "json")
	if err == nil {
		t.Fatal("work proposal accept without --confirm-user-acceptance error = nil, want usage error")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("proposal accept without confirmation exit code = %d, want usage", got)
	}
	if !strings.Contains(err.Error(), "--confirm-user-acceptance") {
		t.Fatalf("proposal accept without confirmation error = %q, want confirmation flag hint", err.Error())
	}
	if got := server.TotalRequests(); got != requestsBeforeProposalAcceptWithoutFlag {
		t.Fatalf("server requests after proposal accept without confirmation = %d, want %d", got, requestsBeforeProposalAcceptWithoutFlag)
	}

	var started spine.WorkStartOutput
	if err := runSmokeWorkCommand(t, repoDir, store, "", &started, "start", "--title", "Refactor CSV export filters", "--body", "Preserve current behavior.", "--format", "json"); err != nil {
		t.Fatalf("work start smoke error = %v", err)
	}
	assertSmokeSchema(t, started.SchemaVersion)
	assertNextAction(t, started.NextAction, "continue_goal", true, false, "")
	if started.GoalID != smokeGoalID {
		t.Fatalf("work start goal_id = %q, want %q", started.GoalID, smokeGoalID)
	}

	var continued spine.WorkContinueOutput
	if err := runSmokeWorkCommand(t, repoDir, store, "", &continued, "continue", "--goal-id", started.GoalID, "--format", "json"); err != nil {
		t.Fatalf("work continue smoke error = %v", err)
	}
	assertSmokeSchema(t, continued.SchemaVersion)
	assertNextAction(t, continued.NextAction, "ask_user", true, true, "")
	if continued.NextAction.RequestID != smokeClarificationRequestID {
		t.Fatalf("ask_user request_id = %q, want %q", continued.NextAction.RequestID, smokeClarificationRequestID)
	}
	if len(continued.NextAction.Questions) != 2 {
		t.Fatalf("ask_user questions = %d, want 2", len(continued.NextAction.Questions))
	}

	answerJSON := `{"answers":[{"question_id":"q_scope","value":"Refactor duplicate CSV export filter logic"},{"question_id":"q_acceptance","value":"Existing CSV export behavior is preserved"}]}`
	var answered spine.WorkAnswerOutput
	if err := runSmokeWorkCommand(t, repoDir, store, answerJSON, &answered, "answer", "--clarification-request-id", continued.NextAction.RequestID, "--answers-file", "-", "--format", "json"); err != nil {
		t.Fatalf("work answer smoke error = %v", err)
	}
	assertSmokeSchema(t, answered.SchemaVersion)
	assertNextAction(t, answered.NextAction, "draft_contract", true, false, "")
	wantDraftCommand := "goalrail contract draft --goal-id " + smokeGoalID + " --format json"
	if answered.NextAction.Command != wantDraftCommand {
		t.Fatalf("work answer next command = %q, want %q", answered.NextAction.Command, wantDraftCommand)
	}

	var drafted spine.ContractDraftOutput
	if err := runSmokeContractCommand(t, repoDir, store, "", &drafted, "draft", "--goal-id", answered.GoalID, "--format", "json"); err != nil {
		t.Fatalf("contract draft smoke error = %v", err)
	}
	assertSmokeSchema(t, drafted.SchemaVersion)
	assertNextAction(t, drafted.NextAction, "update_contract", true, false, "")
	if drafted.ContractID != smokeContractID || drafted.ContractState != spine.ContractStateDraft {
		t.Fatalf("contract draft id/state = %q/%q, want %q/draft", drafted.ContractID, drafted.ContractState, smokeContractID)
	}
	if drafted.LocalRepoReceipt.HeadSHA == "" || drafted.LocalRepoReceipt.OverlayID == "" {
		t.Fatalf("local repo receipt missing head/overlay: %#v", drafted.LocalRepoReceipt)
	}
	if drafted.LocalRepoReceipt.RawSourceUploaded {
		t.Fatal("local repo receipt raw_source_uploaded = true, want false")
	}

	fieldsJSON := `{"proposed_scope":["Refactor CSV export filter duplication"],"proposed_acceptance_criteria":["Existing CSV export behavior is preserved"],"proposed_proof_expectations":["CLI and server tests pass"]}`
	var updated spine.ContractUpdateOutput
	if err := runSmokeContractCommand(t, repoDir, store, fieldsJSON, &updated, "update", "--contract-id", string(drafted.ContractID), "--fields-file", "-", "--format", "json"); err != nil {
		t.Fatalf("contract update smoke error = %v", err)
	}
	assertSmokeSchema(t, updated.SchemaVersion)
	assertNextAction(t, updated.NextAction, "review_contract", true, true, "")
	if !sameStrings(updated.ChangedFields, []string{"proposed_acceptance_criteria", "proposed_proof_expectations", "proposed_scope"}) {
		t.Fatalf("changed_fields = %#v, want expected proposed fields", updated.ChangedFields)
	}

	var submitted spine.ContractTransitionOutput
	if err := runSmokeContractCommand(t, repoDir, store, "", &submitted, "submit", "--contract-id", string(updated.ContractID), "--format", "json"); err != nil {
		t.Fatalf("contract submit smoke error = %v", err)
	}
	assertSmokeSchema(t, submitted.SchemaVersion)
	assertNextAction(t, submitted.NextAction, "approve_contract", true, true, "")
	if !strings.Contains(submitted.NextAction.Command, "--confirm-user-approval") {
		t.Fatalf("approve_contract command = %q, want explicit confirmation flag", submitted.NextAction.Command)
	}
	if submitted.NextAction.RequiresHumanApproval == nil || !*submitted.NextAction.RequiresHumanApproval {
		t.Fatalf("approve_contract requires_human_approval = %#v, want true", submitted.NextAction.RequiresHumanApproval)
	}

	var approved spine.ContractTransitionOutput
	if err := runSmokeContractCommand(t, repoDir, store, "", &approved, "approve", "--contract-id", string(submitted.ContractID), "--confirm-user-approval", "--format", "json"); err != nil {
		t.Fatalf("contract approve smoke error = %v", err)
	}
	assertSmokeSchema(t, approved.SchemaVersion)
	assertNextAction(t, approved.NextAction, "plan_work", true, false, "")
	wantPlanCommand := "goalrail work plan --contract-id " + smokeContractID + " --format json"
	if approved.NextAction.Command != wantPlanCommand {
		t.Fatalf("approved next command = %q, want %q", approved.NextAction.Command, wantPlanCommand)
	}

	var planned spine.WorkPlanOutput
	if err := runSmokeWorkCommand(t, repoDir, store, "", &planned, "plan", "--contract-id", string(approved.ContractID), "--format", "json"); err != nil {
		t.Fatalf("work plan smoke error = %v", err)
	}
	assertSmokeSchema(t, planned.SchemaVersion)
	assertNextAction(t, planned.NextAction, "planning_worker_required", true, true, "")
	if planned.NextAction.CommandPacket == nil || !strings.Contains(planned.NextAction.CommandPacket.SafetyNote, "does not accept proposals") {
		t.Fatalf("planning worker command_packet = %#v, want bounded safety note", planned.NextAction.CommandPacket)
	}
	if planned.PlanID != smokePlanID || planned.PlanState != "queued" {
		t.Fatalf("work plan id/state = %q/%q, want %q/queued", planned.PlanID, planned.PlanState, smokePlanID)
	}

	var status spine.WorkPlanStatusOutput
	if err := runSmokeWorkCommand(t, repoDir, store, "", &status, "plan", "status", "--plan-id", planned.PlanID, "--format", "json"); err != nil {
		t.Fatalf("work plan status smoke error = %v", err)
	}
	assertSmokeSchema(t, status.SchemaVersion)
	assertNextAction(t, status.NextAction, "accept_proposal", true, true, "")
	if status.NextAction.RequiresHumanApproval == nil || !*status.NextAction.RequiresHumanApproval {
		t.Fatalf("accept_proposal requires_human_approval = %#v, want true", status.NextAction.RequiresHumanApproval)
	}
	if status.ProposalID != smokeProposalID || status.ProposalState != "submitted" || len(status.ProposedTasks) != 1 {
		t.Fatalf("plan status proposal = %q/%q/%d, want submitted proposal", status.ProposalID, status.ProposalState, len(status.ProposedTasks))
	}
	if !strings.Contains(status.NextAction.Command, "--confirm-user-acceptance") {
		t.Fatalf("accept_proposal command = %q, want explicit confirmation flag", status.NextAction.Command)
	}

	var accepted spine.WorkProposalAcceptOutput
	if err := runSmokeWorkCommand(t, repoDir, store, "", &accepted, "proposal", "accept", "--proposal-id", status.ProposalID, "--confirm-user-acceptance", "--format", "json"); err != nil {
		t.Fatalf("work proposal accept smoke error = %v", err)
	}
	assertSmokeSchema(t, accepted.SchemaVersion)
	assertNextAction(t, accepted.NextAction, "show_work_item", true, false, "")
	if accepted.ProposalID != smokeProposalID || accepted.PlanID != smokePlanID || len(accepted.CreatedTaskIDs) != 1 || accepted.CreatedTaskIDs[0] != smokeWorkItemID {
		t.Fatalf("proposal accept output = %#v, want one planned WorkItem trace", accepted)
	}
	if !strings.Contains(accepted.NextAction.Command, "goalrail work item show --task-id "+smokeWorkItemID) {
		t.Fatalf("proposal accept next command = %q, want WorkItem detail command", accepted.NextAction.Command)
	}
	if accepted.NextAction.MutatesState == nil || *accepted.NextAction.MutatesState {
		t.Fatalf("proposal accept next mutates_state = %#v, want false for WorkItem show", accepted.NextAction.MutatesState)
	}

	var checkoutPrepared spine.WorkCheckoutPrepareOutput
	if err := runSmokeWorkCommand(t, repoDir, store, "", &checkoutPrepared, "checkout", "prepare", "--task-id", smokeWorkItemID, "--format", "json"); err != nil {
		t.Fatalf("work checkout prepare smoke error = %v", err)
	}
	assertSmokeSchema(t, checkoutPrepared.SchemaVersion)
	assertNextAction(t, checkoutPrepared.NextAction, "runner_checkout_required", false, true, "H2")
	if checkoutPrepared.TaskID != smokeWorkItemID || checkoutPrepared.CheckoutJobID != smokeCheckoutJobID || checkoutPrepared.CheckoutJobState != "queued" {
		t.Fatalf("checkout prepare output = %#v, want queued checkout job for planned WorkItem", checkoutPrepared)
	}
	if checkoutPrepared.Instruction.JobID != smokeCheckoutJobID || checkoutPrepared.Instruction.TaskID != smokeWorkItemID || checkoutPrepared.Instruction.RepoBindingID != spine.RepoBindingID(smokeRepoBindingID) {
		t.Fatalf("checkout instruction identity = %#v, want job/task/repo-bound instruction", checkoutPrepared.Instruction)
	}
	if checkoutPrepared.Instruction.RepositoryFullName != "heurema/goalrail" || checkoutPrepared.Instruction.WorkflowBaseBranch != "main" || checkoutPrepared.Instruction.RawSourceUploaded {
		t.Fatalf("checkout instruction repository/raw-source = %#v, want metadata-only instruction", checkoutPrepared.Instruction)
	}

	var executionPrepared spine.WorkExecutionPrepareOutput
	if err := runSmokeWorkCommand(t, repoDir, store, "", &executionPrepared, "execution", "prepare", "--task-id", smokeWorkItemID, "--checkout-receipt-id", smokeCheckoutReceiptID, "--format", "json"); err != nil {
		t.Fatalf("work execution prepare smoke error = %v", err)
	}
	assertSmokeSchema(t, executionPrepared.SchemaVersion)
	assertNextAction(t, executionPrepared.NextAction, "runner_execution_required", false, true, "H2.3")
	if executionPrepared.TaskID != smokeWorkItemID || executionPrepared.CheckoutReceiptID != smokeCheckoutReceiptID || executionPrepared.ExecutionJobID != smokeExecutionJobID || executionPrepared.ExecutionJobState != "queued" {
		t.Fatalf("execution prepare output = %#v, want queued execution job for task/receipt", executionPrepared)
	}
	if !strings.Contains(executionPrepared.Display.Summary, "No Run was created") || !strings.Contains(executionPrepared.Display.Summary, "no command was executed") {
		t.Fatalf("execution prepare summary = %q, want honest no-Run/no-execution language", executionPrepared.Display.Summary)
	}

	server.AssertNoForbiddenCalls(t)
	server.AssertCalled(t, http.MethodPost, "/v1/contracts/"+smokeContractID+"/approvals", 1)
	server.AssertCalled(t, http.MethodPost, "/v1/contracts/"+smokeContractID+"/plans", 1)
	server.AssertCalled(t, http.MethodPost, "/v1/plans/"+smokePlanID+"/status", 1)
	server.AssertCalled(t, http.MethodPost, "/v1/proposals/"+smokeProposalID+"/acceptance", 1)
	server.AssertCalled(t, http.MethodPost, "/v1/tasks/"+smokeWorkItemID+"/checkout-jobs", 1)
	server.AssertCalled(t, http.MethodPost, "/v1/tasks/"+smokeWorkItemID+"/execution-jobs", 1)
}

func runSmokeWorkCommand(t *testing.T, repoDir string, store smokeSessionStore, stdin string, target any, args ...string) error {
	t.Helper()

	var stdout, stderr bytes.Buffer
	options := workcmd.Options{
		Store: store,
		Now:   smokeNow,
		Stdin: strings.NewReader(stdin),
	}
	err := workcmd.RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, args, options)
	if err != nil {
		return err
	}
	if target != nil {
		decodeSmokeJSON(t, stdout.Bytes(), target)
	}
	return nil
}

func runSmokeContractCommand(t *testing.T, repoDir string, store smokeSessionStore, stdin string, target any, args ...string) error {
	t.Helper()

	var stdout, stderr bytes.Buffer
	options := contractcmd.Options{
		Store:     store,
		CacheRoot: t.TempDir(),
		Now:       smokeNow,
		Stdin:     strings.NewReader(stdin),
	}
	err := contractcmd.RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, args, options)
	if err != nil {
		return err
	}
	if target != nil {
		decodeSmokeJSON(t, stdout.Bytes(), target)
	}
	return nil
}

func decodeSmokeJSON(t *testing.T, raw []byte, target any) {
	t.Helper()

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		t.Fatalf("decode smoke JSON %q: %v", string(raw), err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		t.Fatalf("decode smoke JSON %q: extra JSON value", string(raw))
	}
}

func assertNextAction(t *testing.T, action spine.NextAction, kind string, available bool, blocking bool, plannedSlice string) {
	t.Helper()

	if action.Kind != kind || action.Available != available || action.Blocking != blocking || action.PlannedSlice != plannedSlice {
		t.Fatalf("next_action = %#v, want kind=%q available=%t blocking=%t planned_slice=%q", action, kind, available, blocking, plannedSlice)
	}
}

func assertSmokeSchema(t *testing.T, schemaVersion string) {
	t.Helper()

	if schemaVersion != "goalrail.cli.v1" {
		t.Fatalf("schema_version = %q, want goalrail.cli.v1", schemaVersion)
	}
}

type pullLoopSmokeServer struct {
	*httptest.Server

	mu             sync.Mutex
	calls          map[string]int
	forbiddenCalls []string
}

func newPullLoopSmokeServer(t *testing.T) *pullLoopSmokeServer {
	t.Helper()

	smoke := &pullLoopSmokeServer{
		calls: map[string]int{},
	}
	smoke.Server = httptest.NewServer(http.HandlerFunc(smoke.handle))
	return smoke
}

func (s *pullLoopSmokeServer) TotalRequests() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	total := 0
	for _, count := range s.calls {
		total += count
	}
	return total
}

func (s *pullLoopSmokeServer) AssertCalled(t *testing.T, method string, path string, want int) {
	t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()

	key := method + " " + path
	if got := s.calls[key]; got != want {
		t.Fatalf("%s calls = %d, want %d", key, got, want)
	}
}

func (s *pullLoopSmokeServer) AssertNoForbiddenCalls(t *testing.T) {
	t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.forbiddenCalls) != 0 {
		t.Fatalf("forbidden planning/execution/proof calls = %#v, want none", s.forbiddenCalls)
	}
}

func (s *pullLoopSmokeServer) handle(w http.ResponseWriter, r *http.Request) {
	s.record(r)

	if isForbiddenSmokePath(r.URL.Path) && r.URL.Path != "/v1/tasks/"+smokeWorkItemID+"/checkout-jobs" && r.URL.Path != "/v1/tasks/"+smokeWorkItemID+"/execution-jobs" {
		s.recordForbidden(r)
		http.Error(w, "forbidden smoke path", http.StatusInternalServerError)
		return
	}
	if r.Header.Get("Authorization") != "Bearer access-token" {
		http.Error(w, "bad auth", http.StatusUnauthorized)
		return
	}

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/v1/me":
		writeSmokeJSON(w, http.StatusOK, `{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"`+smokeOrganizationID+`","role":"member","state":"active"}}`)
	case r.Method == http.MethodPost && r.URL.Path == "/v1/intakes":
		var body struct {
			ProjectID     string `json:"project_id"`
			RepoBindingID string `json:"repo_binding_id"`
			Source        struct {
				Kind       string `json:"kind"`
				ExternalID string `json:"external_id,omitempty"`
				URL        string `json:"url,omitempty"`
			} `json:"source"`
			Title         string `json:"title"`
			Body          string `json:"body"`
			RequestAuthor struct {
				Kind        string `json:"kind"`
				ID          string `json:"id"`
				DisplayName string `json:"display_name,omitempty"`
			} `json:"request_author"`
		}
		if !decodeSmokeRequest(w, r, &body) {
			return
		}
		if body.ProjectID != smokeProjectID || body.RepoBindingID != smokeRepoBindingID || body.Source.Kind == "" || body.Title == "" || body.Body == "" || body.RequestAuthor.ID == "" {
			http.Error(w, "bad intake request", http.StatusBadRequest)
			return
		}
		writeSmokeJSON(w, http.StatusAccepted, `{"intake_id":"`+smokeIntakeID+`","organization_id":"`+smokeOrganizationID+`","project_id":"`+smokeProjectID+`","repo_binding_id":"`+smokeRepoBindingID+`","state":"received","canonical_contract_created":false,"next":"server will validate and may promote intake to goal"}`)
	case r.Method == http.MethodPost && r.URL.Path == "/v1/intakes/"+smokeIntakeID+"/goals":
		writeSmokeJSON(w, http.StatusCreated, smokeGoalJSON("created"))
	case r.Method == http.MethodPost && r.URL.Path == "/v1/goals/"+smokeGoalID+"/continuation":
		writeSmokeJSON(w, http.StatusOK, `{"goal_id":"`+smokeGoalID+`","state":"needs_clarification","goal":`+smokeGoalJSON("needs_clarification")+`,"clarification_request":{"id":"`+smokeClarificationRequestID+`","goal_id":"`+smokeGoalID+`","reason_codes":["missing_scope_hint","missing_acceptance_hint"],"state":"open","questions":[{"id":"q_scope","text":"What is in scope?","why_needed":"Scope bounds the Contract.","answer_type":"text","maps_to":"goal.scope_hint"},{"id":"q_acceptance","text":"What proves success?","why_needed":"Acceptance criteria gate the Contract.","answer_type":"text","maps_to":"goal.acceptance_hint"}]}}`)
	case r.Method == http.MethodPost && r.URL.Path == "/v1/clarifications/"+smokeClarificationRequestID+"/answers/continuation":
		var body struct {
			Answers []struct {
				QuestionID string `json:"question_id"`
				Value      string `json:"value"`
			} `json:"answers"`
		}
		if !decodeSmokeRequest(w, r, &body) {
			return
		}
		if len(body.Answers) != 2 {
			http.Error(w, "bad answer request", http.StatusBadRequest)
			return
		}
		writeSmokeJSON(w, http.StatusOK, `{"goal_id":"`+smokeGoalID+`","state":"ready_for_contract_seed","goal":`+smokeGoalJSON("ready_for_contract_seed")+`,"readiness":{"goal_id":"`+smokeGoalID+`","state":"ready_for_contract_seed","ready":true,"reason_codes":[],"message":"ready"}}`)
	case r.Method == http.MethodPost && r.URL.Path == "/v1/contracts":
		var body struct {
			GoalID        string `json:"goal_id"`
			ProjectID     string `json:"project_id"`
			RepoBindingID string `json:"repo_binding_id"`
		}
		if !decodeSmokeRequest(w, r, &body) {
			return
		}
		if body.GoalID != smokeGoalID || body.ProjectID != smokeProjectID || body.RepoBindingID != smokeRepoBindingID {
			http.Error(w, "bad contract create request", http.StatusBadRequest)
			return
		}
		writeSmokeJSON(w, http.StatusCreated, smokeContractJSON("draft"))
	case r.Method == http.MethodPatch && r.URL.Path == "/v1/contracts/"+smokeContractID:
		var body struct {
			ProjectID     string                     `json:"project_id"`
			RepoBindingID string                     `json:"repo_binding_id"`
			Changes       map[string]json.RawMessage `json:"changes"`
			UpdatedBy     struct {
				Kind        string `json:"kind"`
				ID          string `json:"id"`
				DisplayName string `json:"display_name,omitempty"`
			} `json:"updated_by"`
		}
		if !decodeSmokeRequest(w, r, &body) {
			return
		}
		if body.ProjectID != smokeProjectID || body.RepoBindingID != smokeRepoBindingID || body.UpdatedBy.ID == "" || len(body.Changes) != 3 {
			http.Error(w, "bad contract update request", http.StatusBadRequest)
			return
		}
		writeSmokeJSON(w, http.StatusOK, smokeContractJSON("draft"))
	case r.Method == http.MethodPost && r.URL.Path == "/v1/contracts/"+smokeContractID+"/submissions":
		if !decodeSmokeTransition(w, r) {
			return
		}
		writeSmokeJSON(w, http.StatusOK, smokeContractJSON("ready_for_approval"))
	case r.Method == http.MethodPost && r.URL.Path == "/v1/contracts/"+smokeContractID+"/approvals":
		if !decodeSmokeTransition(w, r) {
			return
		}
		writeSmokeJSON(w, http.StatusCreated, smokeContractJSON("approved"))
	case r.Method == http.MethodPost && r.URL.Path == "/v1/contracts/"+smokeContractID+"/plans":
		if !decodeSmokeTransition(w, r) {
			return
		}
		writeSmokeJSON(w, http.StatusCreated, smokePlanJSON("queued"))
	case r.Method == http.MethodPost && r.URL.Path == "/v1/plans/"+smokePlanID+"/status":
		if !decodeSmokeTransition(w, r) {
			return
		}
		writeSmokeJSON(w, http.StatusOK, `{"plan":`+smokePlanJSON("proposal_submitted")+`,"proposal":`+smokeProposalJSON()+`}`)
	case r.Method == http.MethodPost && r.URL.Path == "/v1/proposals/"+smokeProposalID+"/acceptance":
		if !decodeSmokeTransition(w, r) {
			return
		}
		writeSmokeJSON(w, http.StatusCreated, `{"proposal_id":"`+smokeProposalID+`","plan_id":"`+smokePlanID+`","contract_id":"`+smokeContractID+`","state":"accepted","created_task_ids":["`+smokeWorkItemID+`"]}`)
	case r.Method == http.MethodPost && r.URL.Path == "/v1/tasks/"+smokeWorkItemID+"/checkout-jobs":
		if !decodeSmokeTransition(w, r) {
			return
		}
		writeSmokeJSON(w, http.StatusCreated, smokeCheckoutJobJSON("queued"))
	case r.Method == http.MethodPost && r.URL.Path == "/v1/tasks/"+smokeWorkItemID+"/execution-jobs":
		var body struct {
			ProjectID         string `json:"project_id"`
			RepoBindingID     string `json:"repo_binding_id"`
			CheckoutReceiptID string `json:"checkout_receipt_id"`
		}
		if !decodeSmokeRequest(w, r, &body) {
			return
		}
		if body.ProjectID != smokeProjectID || body.RepoBindingID != smokeRepoBindingID || body.CheckoutReceiptID != smokeCheckoutReceiptID {
			http.Error(w, "bad execution job request", http.StatusBadRequest)
			return
		}
		writeSmokeJSON(w, http.StatusCreated, smokeExecutionJobJSON("queued"))
	default:
		http.NotFound(w, r)
	}
}

func (s *pullLoopSmokeServer) record(r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls[r.Method+" "+r.URL.Path]++
}

func (s *pullLoopSmokeServer) recordForbidden(r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.forbiddenCalls = append(s.forbiddenCalls, r.Method+" "+r.URL.Path)
}

func decodeSmokeTransition(w http.ResponseWriter, r *http.Request) bool {
	var body struct {
		ProjectID     string `json:"project_id"`
		RepoBindingID string `json:"repo_binding_id"`
	}
	if !decodeSmokeRequest(w, r, &body) {
		return false
	}
	if body.ProjectID != smokeProjectID || body.RepoBindingID != smokeRepoBindingID {
		http.Error(w, "bad transition request", http.StatusBadRequest)
		return false
	}
	return true
}

func decodeSmokeRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return false
	}
	return true
}

func writeSmokeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func smokeGoalJSON(state string) string {
	return `{"id":"` + smokeGoalID + `","intake_id":"` + smokeIntakeID + `","organization_id":"` + smokeOrganizationID + `","project_id":"` + smokeProjectID + `","repo_binding_id":"` + smokeRepoBindingID + `","title":"Refactor CSV export filters","summary":"Refactor CSV export filters","state":"` + state + `"}`
}

func smokeContractJSON(state string) string {
	approvedSnapshot := ""
	if state == string(spine.ContractStateApproved) {
		approvedSnapshot = `,"approved_snapshot_id":"` + smokeApprovedSnapshotID + `"`
	}
	return `{"id":"` + smokeContractID + `","repo_binding_id":"` + smokeRepoBindingID + `","goal_id":"` + smokeGoalID + `","state":"` + state + `","current_seed_id":"018f0000-0000-7000-8000-000000000011","current_draft_id":"018f0000-0000-7000-8000-000000000012"` + approvedSnapshot + `}`
}

func smokePlanJSON(state string) string {
	return `{"id":"` + smokePlanID + `","contract_id":"` + smokeContractID + `","approved_contract_id":"` + smokeApprovedSnapshotID + `","repo_binding_id":"` + smokeRepoBindingID + `","state":"` + state + `"}`
}

func smokeProposalJSON() string {
	return `{"id":"` + smokeProposalID + `","plan_id":"` + smokePlanID + `","contract_id":"` + smokeContractID + `","approved_contract_id":"` + smokeApprovedSnapshotID + `","repo_binding_id":"` + smokeRepoBindingID + `","state":"submitted","proposed_tasks":[{"title":"Refactor CSV export filters","summary":"Refactor duplicate filter construction while preserving current behavior.","scope":["Update export filter construction"],"acceptance_refs":["acceptance_criteria[0]"],"proof_expectation_refs":["proof_expectations[0]"],"order_index":0}]}`
}

func smokeCheckoutJobJSON(state string) string {
	return `{"id":"` + smokeCheckoutJobID + `","task_id":"` + smokeWorkItemID + `","contract_id":"` + smokeContractID + `","approved_contract_id":"` + smokeApprovedSnapshotID + `","plan_id":"` + smokePlanID + `","proposal_id":"` + smokeProposalID + `","repo_binding_id":"` + smokeRepoBindingID + `","state":"` + state + `","instruction":{"job_id":"` + smokeCheckoutJobID + `","task_id":"` + smokeWorkItemID + `","repo_binding_id":"` + smokeRepoBindingID + `","access_mode":"customer_mounted_workspace","provider":"github","repository_full_name":"heurema/goalrail","repository_url":"https://github.com/heurema/goalrail","workflow_base_branch":"main","path_scope":".","source_ref":{"kind":"work_item","id":"` + smokeWorkItemID + `"},"raw_source_uploaded":false}}`
}

func smokeExecutionJobJSON(state string) string {
	return `{"id":"` + smokeExecutionJobID + `","task_id":"` + smokeWorkItemID + `","contract_id":"` + smokeContractID + `","approved_contract_id":"` + smokeApprovedSnapshotID + `","plan_id":"` + smokePlanID + `","proposal_id":"` + smokeProposalID + `","repo_binding_id":"` + smokeRepoBindingID + `","checkout_job_id":"` + smokeCheckoutJobID + `","checkout_receipt_id":"` + smokeCheckoutReceiptID + `","state":"` + state + `","execution_mode":"prepare_v0"}`
}

func isForbiddenSmokePath(path string) bool {
	for _, fragment := range []string{"/plans/leases", "/tasks", "/work-items", "/assignments", "/claims", "/checkout", "/runs", "/decisions", "/proof"} {
		if strings.Contains(path, fragment) {
			return true
		}
	}
	return false
}

type smokeSessionStore struct {
	session authstore.Session
}

func (s smokeSessionStore) Load() (authstore.Session, error) {
	return s.session, nil
}

func validSmokeSession(serverURL string) authstore.Session {
	return authstore.Session{
		ServerURL:            serverURL,
		AccessToken:          "access-token",
		RefreshToken:         "refresh-token",
		AccessTokenExpiresAt: time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC),
		TokenType:            "Bearer",
	}
}

func setupSmokeGitRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	runSmokeGit(t, repoDir, "init")
	runSmokeGit(t, repoDir, "-c", "user.name=Goalrail Test", "-c", "user.email=goalrail@example.test", "commit", "--allow-empty", "-m", "initial")
	return repoDir
}

func writeSmokeProjectConfig(t *testing.T, repoDir string, serverURL string) {
	t.Helper()

	if _, err := projectconfig.Write(repoDir, projectconfig.Config{
		Version:        projectconfig.Version,
		ServerURL:      serverURL,
		OrganizationID: smokeOrganizationID,
		ProjectID:      smokeProjectID,
		RepoBindingID:  smokeRepoBindingID,
		Repository: projectconfig.Repository{
			Provider:           "github",
			FullName:           "heurema/goalrail",
			URL:                "git@github.com:heurema/goalrail.git",
			WorkflowBaseBranch: "main",
		},
	}); err != nil {
		t.Fatalf("write project config: %v", err)
	}
}

func smokeNow() time.Time {
	return time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
}

func sameStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func requireSmokeGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
}

func runSmokeGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
}
