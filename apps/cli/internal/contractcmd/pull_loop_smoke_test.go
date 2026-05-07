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
)

func TestAgentPullLoopCLISmokeThroughApprovedContract(t *testing.T) {
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

	var started spine.WorkStartOutput
	if err := runSmokeWorkCommand(t, repoDir, store, "", &started, "start", "--title", "Refactor CSV export filters", "--body", "Preserve current behavior.", "--format", "json"); err != nil {
		t.Fatalf("work start smoke error = %v", err)
	}
	assertNextAction(t, started.NextAction, "continue_goal", true, false, "")
	if started.GoalID != smokeGoalID {
		t.Fatalf("work start goal_id = %q, want %q", started.GoalID, smokeGoalID)
	}

	var continued spine.WorkContinueOutput
	if err := runSmokeWorkCommand(t, repoDir, store, "", &continued, "continue", "--goal-id", started.GoalID, "--format", "json"); err != nil {
		t.Fatalf("work continue smoke error = %v", err)
	}
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
	assertNextAction(t, answered.NextAction, "draft_contract", true, false, "")
	wantDraftCommand := "goalrail contract draft --goal-id " + smokeGoalID + " --format json"
	if answered.NextAction.Command != wantDraftCommand {
		t.Fatalf("work answer next command = %q, want %q", answered.NextAction.Command, wantDraftCommand)
	}

	var drafted spine.ContractDraftOutput
	if err := runSmokeContractCommand(t, repoDir, store, "", &drafted, "draft", "--goal-id", answered.GoalID, "--format", "json"); err != nil {
		t.Fatalf("contract draft smoke error = %v", err)
	}
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
	assertNextAction(t, updated.NextAction, "review_contract", true, true, "")
	if !sameStrings(updated.ChangedFields, []string{"proposed_acceptance_criteria", "proposed_proof_expectations", "proposed_scope"}) {
		t.Fatalf("changed_fields = %#v, want expected proposed fields", updated.ChangedFields)
	}

	var submitted spine.ContractTransitionOutput
	if err := runSmokeContractCommand(t, repoDir, store, "", &submitted, "submit", "--contract-id", string(updated.ContractID), "--format", "json"); err != nil {
		t.Fatalf("contract submit smoke error = %v", err)
	}
	assertNextAction(t, submitted.NextAction, "approve_contract", true, true, "")
	if !strings.Contains(submitted.NextAction.Command, "--confirm-user-approval") {
		t.Fatalf("approve_contract command = %q, want explicit confirmation flag", submitted.NextAction.Command)
	}

	var approved spine.ContractTransitionOutput
	if err := runSmokeContractCommand(t, repoDir, store, "", &approved, "approve", "--contract-id", string(submitted.ContractID), "--confirm-user-approval", "--format", "json"); err != nil {
		t.Fatalf("contract approve smoke error = %v", err)
	}
	assertNextAction(t, approved.NextAction, "plan_work", true, false, "")
	wantPlanCommand := "goalrail work plan --contract-id " + smokeContractID + " --format json"
	if approved.NextAction.Command != wantPlanCommand {
		t.Fatalf("approved next command = %q, want %q", approved.NextAction.Command, wantPlanCommand)
	}

	server.AssertNoForbiddenCalls(t)
	server.AssertCalled(t, http.MethodPost, "/v1/contracts/"+smokeContractID+"/approvals", 1)
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

	if isForbiddenSmokePath(r.URL.Path) {
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

func isForbiddenSmokePath(path string) bool {
	for _, fragment := range []string{"/plans", "/work-items", "/runs", "/decisions", "/proof"} {
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
