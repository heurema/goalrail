package httpserver_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/auth"
	"github.com/heurema/goalrail/apps/server/internal/httpserver"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/vcsconnection"
)

func TestVcsConnectionCreateRequiresBearerAuth(t *testing.T) {
	handler := httpserver.NewVcsConnectionHandler(fakeHTTPAuthService{meErr: auth.ErrInvalidToken}, &fakeVcsConnectionService{})

	response := doAuthRequest(t, http.HandlerFunc(handler.Create), http.MethodPost, "/v1/vcs-connections", vcsConnectionCreateJSON(), "")
	if response.code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusUnauthorized, response.body)
	}
	if !strings.Contains(response.body, "unauthorized") {
		t.Fatalf("body = %q, want unauthorized", response.body)
	}
}

func TestVcsConnectionCreateReturnsPendingSetupWithoutCredentialFields(t *testing.T) {
	result := testVcsConnection()
	service := &fakeVcsConnectionService{createResult: result}
	handler := httpserver.NewVcsConnectionHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, service)

	response := doAuthRequest(t, http.HandlerFunc(handler.Create), http.MethodPost, "/v1/vcs-connections", vcsConnectionCreateJSON(), "Bearer access-token")
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	var body struct {
		ID                  string `json:"id"`
		State               string `json:"state"`
		ProviderKind        string `json:"provider_kind"`
		ProviderInstanceURL string `json:"provider_instance_url"`
		SetupExpiresAt      string `json:"setup_expires_at"`
	}
	decodeJSON(t, response.body, &body)
	if body.State != "pending_setup" || body.ProviderKind != "gitlab" || body.ProviderInstanceURL != "https://gitlab.example.com" || body.SetupExpiresAt == "" {
		t.Fatalf("response = %#v, want pending setup connection", body)
	}
	denyResponseFields(t, response.body)
	if service.createInput.AuthenticatedUserID != repoBindingProfile().User.ID {
		t.Fatalf("authenticated user id = %q, want profile user", service.createInput.AuthenticatedUserID)
	}
	if service.createInput.Membership.OrganizationID != repoBindingProfile().OrganizationMembership.OrganizationID {
		t.Fatalf("membership organization id = %q, want profile organization", service.createInput.Membership.OrganizationID)
	}
}

func TestVcsConnectionCreateRejectsSecretLikeUnknownFields(t *testing.T) {
	handler := httpserver.NewVcsConnectionHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, &fakeVcsConnectionService{createResult: testVcsConnection()})
	body := `{"provider_kind":"gitlab","provider_instance_url":"https://gitlab.example.com","access_token":"secret"}`

	response := doAuthRequest(t, http.HandlerFunc(handler.Create), http.MethodPost, "/v1/vcs-connections", body, "Bearer access-token")
	if response.code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusBadRequest, response.body)
	}
	if !strings.Contains(response.body, "invalid_json") {
		t.Fatalf("body = %q, want invalid_json", response.body)
	}
}

func TestVcsConnectionCreateRejectsOrganizationIDBodyField(t *testing.T) {
	handler := httpserver.NewVcsConnectionHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, &fakeVcsConnectionService{createResult: testVcsConnection()})
	body := `{"provider_kind":"gitlab","provider_instance_url":"https://gitlab.example.com","organization_id":"018f0000-0000-7000-8000-000000000999"}`

	response := doAuthRequest(t, http.HandlerFunc(handler.Create), http.MethodPost, "/v1/vcs-connections", body, "Bearer access-token")
	if response.code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusBadRequest, response.body)
	}
	if !strings.Contains(response.body, "invalid_json") {
		t.Fatalf("body = %q, want invalid_json", response.body)
	}
}

func TestVcsConnectionCreateMapsValidationAndForbidden(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		want int
		code string
	}{
		{name: "validation", err: &vcsconnection.ValidationError{Field: "provider_kind", Message: "is invalid"}, want: http.StatusBadRequest, code: "validation_failed"},
		{name: "forbidden", err: vcsconnection.ErrForbidden, want: http.StatusForbidden, code: "forbidden"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			handler := httpserver.NewVcsConnectionHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, &fakeVcsConnectionService{createErr: tt.err})

			response := doAuthRequest(t, http.HandlerFunc(handler.Create), http.MethodPost, "/v1/vcs-connections", vcsConnectionCreateJSON(), "Bearer access-token")
			if response.code != tt.want {
				t.Fatalf("status = %d, want %d: %s", response.code, tt.want, response.body)
			}
			if !strings.Contains(response.body, tt.code) {
				t.Fatalf("body = %q, want %s", response.body, tt.code)
			}
		})
	}
}

func TestVcsConnectionGetReturnsPendingSetupWithoutCredentialFields(t *testing.T) {
	handler := httpserver.NewVcsConnectionHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, &fakeVcsConnectionService{getResult: testVcsConnection()})

	response := doAuthRequest(t, http.HandlerFunc(handler.Get), http.MethodGet, "/v1/vcs-connections/018f0000-0000-7000-8000-000000000010", "", "Bearer access-token")
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	if !strings.Contains(response.body, `"state":"pending_setup"`) {
		t.Fatalf("body = %q, want pending_setup", response.body)
	}
	denyResponseFields(t, response.body)
}

func TestVcsConnectionGetMapsNotFound(t *testing.T) {
	handler := httpserver.NewVcsConnectionHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, &fakeVcsConnectionService{getErr: vcsconnection.ErrNotFound})

	response := doAuthRequest(t, http.HandlerFunc(handler.Get), http.MethodGet, "/v1/vcs-connections/018f0000-0000-7000-8000-000000000099", "", "Bearer access-token")
	if response.code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusNotFound, response.body)
	}
	if !strings.Contains(response.body, "not_found") {
		t.Fatalf("body = %q, want not_found", response.body)
	}
}

func vcsConnectionCreateJSON() string {
	return `{"provider_kind":"gitlab","provider_instance_url":"https://gitlab.example.com/"}`
}

func testVcsConnection() spine.VcsConnection {
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	return spine.VcsConnection{
		ID:                  "018f0000-0000-7000-8000-000000000010",
		InstallationID:      "018f0000-0000-7000-8000-000000000006",
		OrganizationID:      "018f0000-0000-7000-8000-000000000002",
		CreatedByUserID:     "018f0000-0000-7000-8000-000000000001",
		ProviderKind:        "gitlab",
		ProviderInstanceURL: "https://gitlab.example.com",
		State:               spine.VcsConnectionStatePendingSetup,
		SetupExpiresAt:      now.Add(30 * time.Minute),
		CreatedAt:           now,
		UpdatedAt:           now,
	}
}

func denyResponseFields(t *testing.T, body string) {
	t.Helper()
	lower := strings.ToLower(body)
	for _, denied := range []string{
		"access_token",
		"refresh_token",
		"token",
		"credential",
		"client_secret",
		"authorization_code",
		"code_verifier",
		"private_key",
		"deploy_key",
		"checkout",
		"repo_binding_id",
	} {
		if strings.Contains(lower, denied) {
			t.Fatalf("response body contains denied field marker %q: %s", denied, body)
		}
	}
}

type fakeVcsConnectionService struct {
	createInput  vcsconnection.CreateInput
	createResult spine.VcsConnection
	createErr    error
	getInput     vcsconnection.GetInput
	getResult    spine.VcsConnection
	getErr       error
}

func (s *fakeVcsConnectionService) CreatePendingSetup(_ context.Context, input vcsconnection.CreateInput) (spine.VcsConnection, error) {
	s.createInput = input
	if s.createErr != nil {
		return spine.VcsConnection{}, s.createErr
	}
	if s.createResult.ID == "" {
		return spine.VcsConnection{}, errors.New("missing test create result")
	}
	return s.createResult, nil
}

func (s *fakeVcsConnectionService) Get(_ context.Context, input vcsconnection.GetInput) (spine.VcsConnection, error) {
	s.getInput = input
	if s.getErr != nil {
		return spine.VcsConnection{}, s.getErr
	}
	if s.getResult.ID == "" {
		return spine.VcsConnection{}, errors.New("missing test get result")
	}
	return s.getResult, nil
}
