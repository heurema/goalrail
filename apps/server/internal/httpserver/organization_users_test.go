package httpserver_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/auth"
	"github.com/heurema/goalrail/apps/server/internal/httpserver"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/usermanagement"
)

func TestOrganizationUsersRequiresBearerAuth(t *testing.T) {
	handler := httpserver.NewOrganizationUsersHandler(fakeHTTPAuthService{meErr: auth.ErrInvalidToken}, fakeOrganizationUsersService{})

	response := doAuthRequest(t, http.HandlerFunc(handler.List), http.MethodGet, "/v1/organizations/018f0000-0000-7000-8000-000000000002/users", "", "")
	if response.code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusUnauthorized, response.body)
	}
	if !strings.Contains(response.body, "unauthorized") {
		t.Fatalf("body = %q, want unauthorized", response.body)
	}
}

func TestOrganizationUsersCreateReturnsTemporaryPasswordOnce(t *testing.T) {
	handler := httpserver.NewOrganizationUsersHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, fakeOrganizationUsersService{
		createResult: usermanagement.CreateUserResult{
			User: spine.User{
				ID:          "018f0000-0000-7000-8000-000000000003",
				Email:       "dev@example.com",
				DisplayName: "Dev",
				State:       spine.EntityStateActive,
			},
			OrganizationMembership: spine.OrganizationMembership{
				ID:             "018f0000-0000-7000-8000-000000000004",
				OrganizationID: "018f0000-0000-7000-8000-000000000002",
				UserID:         "018f0000-0000-7000-8000-000000000003",
				Role:           spine.OrganizationMembershipRoleMember,
				State:          spine.EntityStateActive,
			},
			Credential:        usermanagement.CredentialSummary{MustChangePassword: true},
			TemporaryPassword: "shown-once-secret",
		},
	})

	response := doAuthRequest(t, http.HandlerFunc(handler.Create), http.MethodPost, "/v1/organizations/018f0000-0000-7000-8000-000000000002/users", `{"email":"dev@example.com","display_name":"Dev","role":"member"}`, "Bearer access-token")
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	if !strings.Contains(response.body, `"temporary_password":"shown-once-secret"`) {
		t.Fatalf("body = %q, want shown-once temporary password", response.body)
	}
	if strings.Contains(response.body, "password_hash") || strings.Contains(response.body, "refresh_token") {
		t.Fatalf("body leaked credential or token material: %s", response.body)
	}
}

func TestOrganizationUsersResetTemporaryPasswordReturnsTemporaryPasswordOnce(t *testing.T) {
	handler := httpserver.NewOrganizationUsersHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, fakeOrganizationUsersService{
		resetResult: usermanagement.ResetTemporaryPasswordResult{
			User: spine.User{
				ID:          "018f0000-0000-7000-8000-000000000003",
				Email:       "dev@example.com",
				DisplayName: "Dev",
				State:       spine.EntityStateActive,
			},
			OrganizationMembership: spine.OrganizationMembership{
				ID:             "018f0000-0000-7000-8000-000000000004",
				OrganizationID: "018f0000-0000-7000-8000-000000000002",
				UserID:         "018f0000-0000-7000-8000-000000000003",
				Role:           spine.OrganizationMembershipRoleMember,
				State:          spine.EntityStateActive,
			},
			Credential:        usermanagement.CredentialSummary{MustChangePassword: true},
			TemporaryPassword: "rotated-once-secret",
		},
	})

	response := doAuthRequest(t, http.HandlerFunc(handler.ResetTemporaryPassword), http.MethodPost, "/v1/organizations/018f0000-0000-7000-8000-000000000002/users/018f0000-0000-7000-8000-000000000003/temporary-password-resets", "", "Bearer access-token")
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	if !strings.Contains(response.body, `"temporary_password":"rotated-once-secret"`) {
		t.Fatalf("body = %q, want shown-once rotated temporary password", response.body)
	}
	if strings.Contains(response.body, "password_hash") || strings.Contains(response.body, "refresh_token") {
		t.Fatalf("body leaked credential or token material: %s", response.body)
	}
}

func TestOrganizationUsersMapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "forbidden", err: usermanagement.ErrForbidden, wantStatus: http.StatusForbidden, wantCode: "forbidden"},
		{name: "validation", err: &usermanagement.ValidationError{Message: "role must be one of owner, admin, member, or viewer"}, wantStatus: http.StatusBadRequest, wantCode: "validation_failed"},
		{name: "user exists", err: usermanagement.ErrUserExists, wantStatus: http.StatusConflict, wantCode: "organization_user_exists"},
		{name: "last owner", err: usermanagement.ErrLastActiveOwner, wantStatus: http.StatusConflict, wantCode: "last_active_owner"},
		{name: "not found", err: usermanagement.ErrNotFound, wantStatus: http.StatusNotFound, wantCode: "not_found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := httpserver.NewOrganizationUsersHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, fakeOrganizationUsersService{listErr: tt.err})

			response := doAuthRequest(t, http.HandlerFunc(handler.List), http.MethodGet, "/v1/organizations/018f0000-0000-7000-8000-000000000002/users", "", "Bearer access-token")
			if response.code != tt.wantStatus {
				t.Fatalf("status = %d, want %d: %s", response.code, tt.wantStatus, response.body)
			}
			if !strings.Contains(response.body, tt.wantCode) {
				t.Fatalf("body = %q, want error code %q", response.body, tt.wantCode)
			}
		})
	}
}

func TestOrganizationUsersCreateConflictDoesNotReturnTemporaryPassword(t *testing.T) {
	handler := httpserver.NewOrganizationUsersHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, fakeOrganizationUsersService{createErr: usermanagement.ErrUserExists})

	response := doAuthRequest(t, http.HandlerFunc(handler.Create), http.MethodPost, "/v1/organizations/018f0000-0000-7000-8000-000000000002/users", `{"email":"dev@example.com","display_name":"Dev","role":"member"}`, "Bearer access-token")
	if response.code != http.StatusConflict {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusConflict, response.body)
	}
	if !strings.Contains(response.body, "organization_user_exists") {
		t.Fatalf("body = %q, want organization_user_exists", response.body)
	}
	if strings.Contains(response.body, "temporary_password") {
		t.Fatalf("body leaked temporary_password on conflict: %s", response.body)
	}
}

type fakeOrganizationUsersService struct {
	listResult   usermanagement.ListUsersResult
	listErr      error
	createResult usermanagement.CreateUserResult
	createErr    error
	patchResult  usermanagement.PatchUserResult
	patchErr     error
	resetResult  usermanagement.ResetTemporaryPasswordResult
	resetErr     error
}

func (s fakeOrganizationUsersService) ListUsers(context.Context, usermanagement.ListUsersInput) (usermanagement.ListUsersResult, error) {
	if s.listErr != nil {
		return usermanagement.ListUsersResult{}, s.listErr
	}
	return s.listResult, nil
}

func (s fakeOrganizationUsersService) CreateUser(context.Context, usermanagement.CreateUserInput) (usermanagement.CreateUserResult, error) {
	if s.createErr != nil {
		return usermanagement.CreateUserResult{}, s.createErr
	}
	if s.createResult.User.ID == "" {
		return usermanagement.CreateUserResult{}, errors.New("missing test create result")
	}
	return s.createResult, nil
}

func (s fakeOrganizationUsersService) PatchUser(context.Context, usermanagement.PatchUserInput) (usermanagement.PatchUserResult, error) {
	if s.patchErr != nil {
		return usermanagement.PatchUserResult{}, s.patchErr
	}
	return s.patchResult, nil
}

func (s fakeOrganizationUsersService) ResetTemporaryPassword(context.Context, usermanagement.ResetTemporaryPasswordInput) (usermanagement.ResetTemporaryPasswordResult, error) {
	if s.resetErr != nil {
		return usermanagement.ResetTemporaryPasswordResult{}, s.resetErr
	}
	if s.resetResult.User.ID == "" {
		return usermanagement.ResetTemporaryPasswordResult{}, errors.New("missing test reset result")
	}
	return s.resetResult, nil
}
