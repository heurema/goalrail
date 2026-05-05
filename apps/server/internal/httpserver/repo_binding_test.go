package httpserver_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/auth"
	"github.com/heurema/goalrail/apps/server/internal/httpserver"
	"github.com/heurema/goalrail/apps/server/internal/repobinding"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestRepoBindingInitRequiresBearerAuth(t *testing.T) {
	handler := httpserver.NewRepoBindingHandler(fakeHTTPAuthService{meErr: auth.ErrInvalidToken}, fakeRepoBindingInitService{})

	response := doAuthRequest(t, http.HandlerFunc(handler.Init), http.MethodPost, "/v1/projects/018f0000-0000-7000-8000-000000000003/repo-bindings/init", repoBindingInitJSON(), "")
	if response.code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusUnauthorized, response.body)
	}
	if !strings.Contains(response.body, "unauthorized") {
		t.Fatalf("body = %q, want unauthorized", response.body)
	}
}

func TestRepoBindingInitReturnsCreatedResponse(t *testing.T) {
	handler := httpserver.NewRepoBindingHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, fakeRepoBindingInitService{
		result: spine.RepoBindingInitResult{
			RepoBindingID:         "018f0000-0000-7000-8000-000000000004",
			ProjectID:             "018f0000-0000-7000-8000-000000000003",
			OrganizationID:        "018f0000-0000-7000-8000-000000000002",
			Provider:              "github",
			RepositoryFullName:    "heurema/goalrail",
			RepositoryURL:         "git@github.com:heurema/goalrail.git",
			ProviderDefaultBranch: "main",
			WorkflowBaseBranch:    "main",
			State:                 spine.EntityStateActive,
			Created:               true,
			Message:               "Repository binding initialized.",
		},
	})

	response := doAuthRequest(t, http.HandlerFunc(handler.Init), http.MethodPost, "/v1/projects/018f0000-0000-7000-8000-000000000003/repo-bindings/init", repoBindingInitJSON(), "Bearer access-token")
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	var body struct {
		RepoBindingID      string `json:"repo_binding_id"`
		WorkflowBaseBranch string `json:"workflow_base_branch"`
		Created            bool   `json:"created"`
	}
	decodeJSON(t, response.body, &body)
	if body.RepoBindingID != "018f0000-0000-7000-8000-000000000004" || body.WorkflowBaseBranch != "main" || !body.Created {
		t.Fatalf("response = %#v, want repo binding init result", body)
	}
}

func TestRepoBindingInitMapsConflict(t *testing.T) {
	handler := httpserver.NewRepoBindingHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, fakeRepoBindingInitService{err: repobinding.ErrDifferentRepoBinding})

	response := doAuthRequest(t, http.HandlerFunc(handler.Init), http.MethodPost, "/v1/projects/018f0000-0000-7000-8000-000000000003/repo-bindings/init", repoBindingInitJSON(), "Bearer access-token")
	if response.code != http.StatusConflict {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusConflict, response.body)
	}
	if !strings.Contains(response.body, "repo_binding_conflict") {
		t.Fatalf("body = %q, want repo_binding_conflict", response.body)
	}
}

func TestRepoBindingInitMapsValidationError(t *testing.T) {
	handler := httpserver.NewRepoBindingHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, fakeRepoBindingInitService{
		err: &repobinding.ValidationError{Field: "workflow_base_branch", Message: "is required"},
	})

	response := doAuthRequest(t, http.HandlerFunc(handler.Init), http.MethodPost, "/v1/projects/018f0000-0000-7000-8000-000000000003/repo-bindings/init", repoBindingInitJSON(), "Bearer access-token")
	if response.code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusBadRequest, response.body)
	}
	if !strings.Contains(response.body, "validation_failed") {
		t.Fatalf("body = %q, want validation_failed", response.body)
	}
}

func repoBindingInitJSON() string {
	return `{"provider":"github","repository_full_name":"heurema/goalrail","repository_url":"git@github.com:heurema/goalrail.git","provider_default_branch":"main","workflow_base_branch":"main","local_remote_name":"origin","local_head_sha":"abc123"}`
}

func repoBindingProfile() auth.Profile {
	return auth.Profile{
		User: spine.User{ID: "018f0000-0000-7000-8000-000000000001", State: spine.EntityStateActive},
		OrganizationMembership: spine.OrganizationMembership{
			ID:             "018f0000-0000-7000-8000-000000000005",
			OrganizationID: "018f0000-0000-7000-8000-000000000002",
			UserID:         "018f0000-0000-7000-8000-000000000001",
			Role:           spine.OrganizationMembershipRoleMember,
			State:          spine.EntityStateActive,
		},
	}
}

type fakeRepoBindingInitService struct {
	result spine.RepoBindingInitResult
	err    error
}

func (s fakeRepoBindingInitService) Init(context.Context, repobinding.InitInput) (spine.RepoBindingInitResult, error) {
	if s.err != nil {
		return spine.RepoBindingInitResult{}, s.err
	}
	if s.result.RepoBindingID == "" {
		return spine.RepoBindingInitResult{}, errors.New("missing test result")
	}
	return s.result, nil
}
