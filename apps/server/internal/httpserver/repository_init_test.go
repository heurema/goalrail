package httpserver_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/auth"
	"github.com/heurema/goalrail/apps/server/internal/httpserver"
	"github.com/heurema/goalrail/apps/server/internal/repositoryinit"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestRepositoryContextInitRequiresBearerAuth(t *testing.T) {
	handler := httpserver.NewRepositoryInitHandler(fakeHTTPAuthService{meErr: auth.ErrInvalidToken}, fakeRepositoryContextInitService{})

	response := doAuthRequest(t, http.HandlerFunc(handler.Init), http.MethodPost, "/v1/init/repository-context", repositoryContextInitJSON(), "")
	if response.code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusUnauthorized, response.body)
	}
	if !strings.Contains(response.body, "unauthorized") {
		t.Fatalf("body = %q, want unauthorized", response.body)
	}
}

func TestRepositoryContextInitReturnsCreatedResponse(t *testing.T) {
	handler := httpserver.NewRepositoryInitHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, fakeRepositoryContextInitService{
		result: spine.RepositoryContextInitResult{
			OrganizationID:        "018f0000-0000-7000-8000-000000000002",
			ProjectID:             "018f0000-0000-7000-8000-000000000003",
			ProjectSlug:           "github-acme-frontend",
			ProjectDisplayName:    "acme/frontend",
			ProjectCreated:        true,
			RepoBindingID:         "018f0000-0000-7000-8000-000000000004",
			RepoBindingCreated:    true,
			Provider:              "github",
			RepositoryFullName:    "acme/frontend",
			RepositoryURL:         "git@github.com:acme/frontend.git",
			ProviderDefaultBranch: "main",
			WorkflowBaseBranch:    "main",
			State:                 spine.EntityStateActive,
			Message:               "Repository context initialized.",
		},
	})

	response := doAuthRequest(t, http.HandlerFunc(handler.Init), http.MethodPost, "/v1/init/repository-context", repositoryContextInitJSON(), "Bearer access-token")
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	var body struct {
		ProjectSlug        string `json:"project_slug"`
		ProjectCreated     bool   `json:"project_created"`
		RepoBindingCreated bool   `json:"repo_binding_created"`
	}
	decodeJSON(t, response.body, &body)
	if body.ProjectSlug != "github-acme-frontend" || !body.ProjectCreated || !body.RepoBindingCreated {
		t.Fatalf("response = %#v, want repository context init result", body)
	}
}

func TestRepositoryContextInitRejectsOrganizationIDBodyField(t *testing.T) {
	service := fakeRepositoryContextInitService{
		result: spine.RepositoryContextInitResult{ProjectID: "018f0000-0000-7000-8000-000000000003"},
	}
	handler := httpserver.NewRepositoryInitHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, service)
	body := `{"organization_id":"018f0000-0000-7000-8000-000000000999","provider":"github","repository_full_name":"acme/frontend","repository_url":"git@github.com:acme/frontend.git","provider_default_branch":"main","workflow_base_branch":"main","local_remote_name":"origin","local_head_sha":"abc123","suggested_project_slug":"github-acme-frontend","suggested_project_display_name":"acme/frontend"}`

	response := doAuthRequest(t, http.HandlerFunc(handler.Init), http.MethodPost, "/v1/init/repository-context", body, "Bearer access-token")
	if response.code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusBadRequest, response.body)
	}
	if !strings.Contains(response.body, "invalid_json") {
		t.Fatalf("body = %q, want invalid_json", response.body)
	}
}

func TestRepositoryContextInitMapsViewerForbidden(t *testing.T) {
	handler := httpserver.NewRepositoryInitHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, fakeRepositoryContextInitService{err: repositoryinit.ErrForbidden})

	response := doAuthRequest(t, http.HandlerFunc(handler.Init), http.MethodPost, "/v1/init/repository-context", repositoryContextInitJSON(), "Bearer access-token")
	if response.code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusForbidden, response.body)
	}
	if !strings.Contains(response.body, "forbidden") {
		t.Fatalf("body = %q, want forbidden", response.body)
	}
}

func TestRepositoryContextInitMapsProjectSlugConflict(t *testing.T) {
	handler := httpserver.NewRepositoryInitHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, fakeRepositoryContextInitService{err: repositoryinit.ErrProjectSlugConflict})

	response := doAuthRequest(t, http.HandlerFunc(handler.Init), http.MethodPost, "/v1/init/repository-context", repositoryContextInitJSON(), "Bearer access-token")
	if response.code != http.StatusConflict {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusConflict, response.body)
	}
	if !strings.Contains(response.body, "project slug is already bound to a different repository") {
		t.Fatalf("body = %q, want project slug conflict message", response.body)
	}
}

func TestRepositoryContextInitMapsProjectSlugUnavailable(t *testing.T) {
	handler := httpserver.NewRepositoryInitHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, fakeRepositoryContextInitService{err: repositoryinit.ErrProjectSlugUnavailable})

	response := doAuthRequest(t, http.HandlerFunc(handler.Init), http.MethodPost, "/v1/init/repository-context", repositoryContextInitJSON(), "Bearer access-token")
	if response.code != http.StatusConflict {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusConflict, response.body)
	}
	if !strings.Contains(response.body, "project slug is already used by an inactive project") {
		t.Fatalf("body = %q, want inactive project slug conflict message", response.body)
	}
}

func repositoryContextInitJSON() string {
	return `{"provider":"github","repository_full_name":"acme/frontend","repository_url":"git@github.com:acme/frontend.git","provider_default_branch":"main","workflow_base_branch":"main","local_remote_name":"origin","local_head_sha":"abc123","suggested_project_slug":"github-acme-frontend","suggested_project_display_name":"acme/frontend"}`
}

type fakeRepositoryContextInitService struct {
	result spine.RepositoryContextInitResult
	err    error
}

func (s fakeRepositoryContextInitService) Init(context.Context, repositoryinit.InitInput) (spine.RepositoryContextInitResult, error) {
	if s.err != nil {
		return spine.RepositoryContextInitResult{}, s.err
	}
	if s.result.ProjectID == "" {
		return spine.RepositoryContextInitResult{}, errors.New("missing test result")
	}
	return s.result, nil
}
