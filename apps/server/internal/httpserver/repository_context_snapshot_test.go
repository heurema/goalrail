package httpserver_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/auth"
	"github.com/heurema/goalrail/apps/server/internal/httpserver"
	"github.com/heurema/goalrail/apps/server/internal/repositorycontext"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestRepositoryContextSnapshotRequiresBearerAuth(t *testing.T) {
	handler := httpserver.NewRepositoryContextSnapshotHandler(fakeHTTPAuthService{meErr: auth.ErrInvalidToken}, fakeRepositoryContextSnapshotService{})

	response := doAuthRequest(t, http.HandlerFunc(handler.Record), http.MethodPost, "/v1/repo-bindings/018f0000-0000-7000-8000-000000000004/context-snapshots", repositoryContextSnapshotJSON(), "")
	if response.code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusUnauthorized, response.body)
	}
	if !strings.Contains(response.body, "unauthorized") {
		t.Fatalf("body = %q, want unauthorized", response.body)
	}
}

func TestRepositoryContextSnapshotReturnsCreatedResponse(t *testing.T) {
	handler := httpserver.NewRepositoryContextSnapshotHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, fakeRepositoryContextSnapshotService{
		result: spine.RepositoryContextSnapshotResult{
			ContextSnapshotID: "018f0000-0000-7000-8000-000000000301",
			OrganizationID:    "018f0000-0000-7000-8000-000000000002",
			ProjectID:         "018f0000-0000-7000-8000-000000000003",
			RepoBindingID:     "018f0000-0000-7000-8000-000000000004",
			Source:            "goalrail_cli_init",
			SchemaVersion:     1,
			Fingerprint:       "sha256:abc123",
			Created:           true,
			Message:           "Repository context snapshot recorded.",
		},
	})

	response := doAuthRequest(t, http.HandlerFunc(handler.Record), http.MethodPost, "/v1/repo-bindings/018f0000-0000-7000-8000-000000000004/context-snapshots", repositoryContextSnapshotJSON(), "Bearer access-token")
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	var body struct {
		ContextSnapshotID string `json:"context_snapshot_id"`
		Fingerprint       string `json:"fingerprint"`
		Created           bool   `json:"created"`
	}
	decodeJSON(t, response.body, &body)
	if body.ContextSnapshotID != "018f0000-0000-7000-8000-000000000301" || body.Fingerprint != "sha256:abc123" || !body.Created {
		t.Fatalf("response = %#v, want snapshot result", body)
	}
}

func TestRepositoryContextSnapshotMapsMismatchConflict(t *testing.T) {
	handler := httpserver.NewRepositoryContextSnapshotHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, fakeRepositoryContextSnapshotService{err: repositorycontext.ErrSnapshotMismatch})

	response := doAuthRequest(t, http.HandlerFunc(handler.Record), http.MethodPost, "/v1/repo-bindings/018f0000-0000-7000-8000-000000000004/context-snapshots", repositoryContextSnapshotJSON(), "Bearer access-token")
	if response.code != http.StatusConflict {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusConflict, response.body)
	}
	if !strings.Contains(response.body, "repository_context_snapshot_conflict") {
		t.Fatalf("body = %q, want snapshot conflict", response.body)
	}
}

func TestRepositoryContextSnapshotRejectsUnknownJSONField(t *testing.T) {
	handler := httpserver.NewRepositoryContextSnapshotHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, fakeRepositoryContextSnapshotService{
		result: spine.RepositoryContextSnapshotResult{ContextSnapshotID: "018f0000-0000-7000-8000-000000000301"},
	})
	body := strings.TrimSuffix(repositoryContextSnapshotJSON(), "}") + `,"access_token":"secret"}`

	response := doAuthRequest(t, http.HandlerFunc(handler.Record), http.MethodPost, "/v1/repo-bindings/018f0000-0000-7000-8000-000000000004/context-snapshots", body, "Bearer access-token")
	if response.code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusBadRequest, response.body)
	}
	if !strings.Contains(response.body, "invalid_json") {
		t.Fatalf("body = %q, want invalid_json", response.body)
	}
}

func repositoryContextSnapshotJSON() string {
	return `{"source":"goalrail_cli_init","schema_version":1,"repository":{"provider":"github","full_name":"heurema/goalrail","url":"git@github.com:heurema/goalrail.git","provider_default_branch":"main","workflow_base_branch":"main","remote_name":"origin","head_sha":"abc123"},"detected_paths":["go.mod","package.json"],"detected_toolchains":["go","node"],"detected_package_managers":["pnpm"],"workspace_candidates":["apps/cli"]}`
}

type fakeRepositoryContextSnapshotService struct {
	result spine.RepositoryContextSnapshotResult
	err    error
}

func (s fakeRepositoryContextSnapshotService) RecordSnapshot(context.Context, repositorycontext.RecordInput) (spine.RepositoryContextSnapshotResult, error) {
	if s.err != nil {
		return spine.RepositoryContextSnapshotResult{}, s.err
	}
	if s.result.ContextSnapshotID == "" {
		return spine.RepositoryContextSnapshotResult{}, errors.New("missing test result")
	}
	return s.result, nil
}
