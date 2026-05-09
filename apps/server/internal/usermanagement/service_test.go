package usermanagement

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestOwnerCanListUsersInOwnOrganization(t *testing.T) {
	store := newFakeStore()
	service := newTestService(store)

	result, err := service.ListUsers(context.Background(), ListUsersInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      orgID,
	})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(result.Users) != 2 {
		t.Fatalf("users len = %d, want 2", len(result.Users))
	}
	if result.Users[0].User.ID != ownerUserID || result.Users[1].User.ID != memberUserID {
		t.Fatalf("users = %#v, want owner then member", result.Users)
	}
}

func TestNonOwnersCannotManageUsersInV0(t *testing.T) {
	for _, role := range []spine.OrganizationMembershipRole{
		spine.OrganizationMembershipRoleAdmin,
		spine.OrganizationMembershipRoleMember,
		spine.OrganizationMembershipRoleViewer,
	} {
		t.Run(string(role), func(t *testing.T) {
			store := newFakeStore()
			store.memberships[key(orgID, ownerUserID)] = membership(ownerMembershipID, orgID, ownerUserID, role, spine.EntityStateActive)
			service := newTestService(store)

			_, err := service.ListUsers(context.Background(), ListUsersInput{
				AuthenticatedUserID: ownerUserID,
				OrganizationID:      orgID,
			})
			if !errors.Is(err, ErrForbidden) {
				t.Fatalf("ListUsers() error = %v, want ErrForbidden", err)
			}
			_, err = service.CreateUser(context.Background(), CreateUserInput{
				AuthenticatedUserID: ownerUserID,
				OrganizationID:      orgID,
				Email:               "new@example.com",
				DisplayName:         "New User",
				Role:                "member",
			})
			if !errors.Is(err, ErrForbidden) {
				t.Fatalf("CreateUser() error = %v, want ErrForbidden", err)
			}
			nextRole := "viewer"
			_, err = service.PatchUser(context.Background(), PatchUserInput{
				AuthenticatedUserID: ownerUserID,
				OrganizationID:      orgID,
				UserID:              memberUserID,
				Role:                &nextRole,
			})
			if !errors.Is(err, ErrForbidden) {
				t.Fatalf("PatchUser() error = %v, want ErrForbidden", err)
			}
			_, err = service.ResetTemporaryPassword(context.Background(), ResetTemporaryPasswordInput{
				AuthenticatedUserID: ownerUserID,
				OrganizationID:      orgID,
				UserID:              memberUserID,
			})
			if !errors.Is(err, ErrForbidden) {
				t.Fatalf("ResetTemporaryPassword() error = %v, want ErrForbidden", err)
			}
		})
	}
}

func TestCrossOrganizationRequestIsRejected(t *testing.T) {
	store := newFakeStore()
	service := newTestService(store)

	_, err := service.ListUsers(context.Background(), ListUsersInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      otherOrgID,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("ListUsers() error = %v, want ErrForbidden", err)
	}
}

func TestOwnerCanCreateUserWithTemporaryPasswordReturnedOnce(t *testing.T) {
	store := newFakeStore()
	service := newTestService(store)

	result, err := service.CreateUser(context.Background(), CreateUserInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      orgID,
		Email:               " DEV@EXAMPLE.COM ",
		DisplayName:         " Dev Name ",
		Role:                "member",
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if result.TemporaryPassword != "temporary-password" {
		t.Fatalf("TemporaryPassword = %q, want generated password", result.TemporaryPassword)
	}
	if result.User.Email != "dev@example.com" || result.User.DisplayName != "Dev Name" {
		t.Fatalf("user = %#v, want normalized email and display name", result.User)
	}
	if result.OrganizationMembership.Role != spine.OrganizationMembershipRoleMember || result.OrganizationMembership.State != spine.EntityStateActive {
		t.Fatalf("membership = %#v, want active member", result.OrganizationMembership)
	}

	credential := store.credentials[result.User.ID]
	if credential.PasswordHash != "hash:temporary-password" {
		t.Fatalf("stored PasswordHash = %q, want hash", credential.PasswordHash)
	}
	if !credential.MustChangePassword || credential.PasswordChangedAt != nil {
		t.Fatalf("stored credential = %#v, want first-login password change", credential)
	}
	encoded := mustJSON(t, result)
	if strings.Contains(encoded, "password_hash") || strings.Contains(encoded, "hash:temporary-password") {
		t.Fatalf("create response leaked credential material: %s", encoded)
	}
}

func TestCreateUserWithExistingEmailReturnsConflictWithoutMutatingExistingRecords(t *testing.T) {
	store := newFakeStore()
	originalUser := store.users[memberUserID]
	originalMembership := store.memberships[key(orgID, memberUserID)]
	originalCredential := spine.UserPasswordCredential{
		UserID:             memberUserID,
		PasswordHash:       "hash:existing-password",
		MustChangePassword: false,
		PasswordChangedAt:  ptrTime(testNow.Add(-2 * time.Hour)),
		CreatedAt:          testNow.Add(-3 * time.Hour),
		UpdatedAt:          testNow.Add(-2 * time.Hour),
	}
	store.credentials[memberUserID] = originalCredential
	service := newTestService(store)

	result, err := service.CreateUser(context.Background(), CreateUserInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      orgID,
		Email:               " MEMBER@EXAMPLE.COM ",
		DisplayName:         "Changed Name",
		Role:                "owner",
	})
	if !errors.Is(err, ErrUserExists) {
		t.Fatalf("CreateUser() error = %v, want ErrUserExists", err)
	}
	if result.TemporaryPassword != "" {
		t.Fatalf("TemporaryPassword = %q, want empty on conflict", result.TemporaryPassword)
	}
	if got := store.users[memberUserID]; got != originalUser {
		t.Fatalf("user mutated on conflict:\n got: %#v\nwant: %#v", got, originalUser)
	}
	if got := store.memberships[key(orgID, memberUserID)]; got != originalMembership {
		t.Fatalf("membership mutated on conflict:\n got: %#v\nwant: %#v", got, originalMembership)
	}
	if got := store.credentials[memberUserID]; !sameCredential(got, originalCredential) {
		t.Fatalf("credential mutated on conflict:\n got: %#v\nwant: %#v", got, originalCredential)
	}
	if _, ok := store.users[newUserID]; ok {
		t.Fatalf("new user was created despite conflict: %#v", store.users[newUserID])
	}
}

func TestCreateUserWithExistingUserOutsideOrganizationReturnsConflict(t *testing.T) {
	store := newFakeStore()
	originalUser := user(otherExistingUserID, "Existing Other", "other-existing@example.com", spine.EntityStateActive)
	originalCredential := spine.UserPasswordCredential{
		UserID:             otherExistingUserID,
		PasswordHash:       "hash:existing-other-password",
		MustChangePassword: false,
		PasswordChangedAt:  ptrTime(testNow.Add(-2 * time.Hour)),
		CreatedAt:          testNow.Add(-3 * time.Hour),
		UpdatedAt:          testNow.Add(-2 * time.Hour),
	}
	store.users[otherExistingUserID] = originalUser
	store.memberships[key(otherOrgID, otherExistingUserID)] = membership(otherExistingMembershipID, otherOrgID, otherExistingUserID, spine.OrganizationMembershipRoleViewer, spine.EntityStateActive)
	store.credentials[otherExistingUserID] = originalCredential
	service := newTestService(store)

	result, err := service.CreateUser(context.Background(), CreateUserInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      orgID,
		Email:               " OTHER-EXISTING@EXAMPLE.COM ",
		DisplayName:         "Ignored Name",
		Role:                "admin",
	})
	if !errors.Is(err, ErrUserExists) {
		t.Fatalf("CreateUser() error = %v, want ErrUserExists", err)
	}
	if result.TemporaryPassword != "" {
		t.Fatalf("TemporaryPassword = %q, want empty on conflict", result.TemporaryPassword)
	}
	if got := store.users[otherExistingUserID]; got != originalUser {
		t.Fatalf("existing user mutated:\n got: %#v\nwant: %#v", got, originalUser)
	}
	if got := store.credentials[otherExistingUserID]; !sameCredential(got, originalCredential) {
		t.Fatalf("existing credential mutated:\n got: %#v\nwant: %#v", got, originalCredential)
	}
	if _, ok := store.memberships[key(orgID, otherExistingUserID)]; ok {
		t.Fatalf("unexpected membership created for existing user")
	}
	if encoded := mustJSON(t, result); strings.Contains(encoded, "temporary_password") || strings.Contains(encoded, "existing-other-password") || strings.Contains(encoded, "password_hash") {
		t.Fatalf("attach response leaked temporary password or credential material: %s", encoded)
	}
}

func TestCreateUserDuplicateEmailInsertRaceReturnsConflict(t *testing.T) {
	store := newFakeStore()
	originalUser := user(otherExistingUserID, "Race", "race@example.com", spine.EntityStateActive)
	originalCredential := spine.UserPasswordCredential{
		UserID:             otherExistingUserID,
		PasswordHash:       "hash:race-password",
		MustChangePassword: false,
		PasswordChangedAt:  ptrTime(testNow.Add(-time.Hour)),
		CreatedAt:          testNow.Add(-2 * time.Hour),
		UpdatedAt:          testNow.Add(-time.Hour),
	}
	store.createUserConflict = originalUser
	store.credentials[otherExistingUserID] = originalCredential
	service := newTestService(store)

	result, err := service.CreateUser(context.Background(), CreateUserInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      orgID,
		Email:               "race@example.com",
		DisplayName:         "Race",
		Role:                "member",
	})
	if !errors.Is(err, ErrUserExists) {
		t.Fatalf("CreateUser() error = %v, want ErrUserExists", err)
	}
	if result.TemporaryPassword != "" {
		t.Fatalf("TemporaryPassword = %q, want empty on conflict", result.TemporaryPassword)
	}
	if got := store.users[otherExistingUserID]; got != originalUser {
		t.Fatalf("existing user mutated after race:\n got: %#v\nwant: %#v", got, originalUser)
	}
	if got := store.credentials[otherExistingUserID]; !sameCredential(got, originalCredential) {
		t.Fatalf("existing credential mutated after race:\n got: %#v\nwant: %#v", got, originalCredential)
	}
	if _, ok := store.memberships[key(orgID, otherExistingUserID)]; ok {
		t.Fatalf("unexpected membership created for existing user")
	}
}

func TestListUsersDoesNotExposeTemporaryPasswordOrCredentialMaterial(t *testing.T) {
	store := newFakeStore()
	store.credentials[memberUserID] = spine.UserPasswordCredential{
		UserID:             memberUserID,
		PasswordHash:       "hash:member-temporary-password",
		MustChangePassword: true,
		CreatedAt:          testNow,
		UpdatedAt:          testNow,
	}
	service := newTestService(store)

	result, err := service.ListUsers(context.Background(), ListUsersInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      orgID,
	})
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	encoded := mustJSON(t, result)
	for _, forbidden := range []string{
		"temporary_password",
		"member-temporary-password",
		"password_hash",
		"hash:member-temporary-password",
		"refresh_token",
		"access_token",
		"cli_auth",
	} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("list response leaked %q: %s", forbidden, encoded)
		}
	}
}

func TestOwnerCanResetExistingUserTemporaryPassword(t *testing.T) {
	store := newFakeStore()
	originalUser := store.users[memberUserID]
	originalMembership := store.memberships[key(orgID, memberUserID)]
	store.credentials[memberUserID] = spine.UserPasswordCredential{
		UserID:             memberUserID,
		PasswordHash:       "hash:old-password",
		MustChangePassword: false,
		PasswordChangedAt:  ptrTime(testNow.Add(-time.Hour)),
		CreatedAt:          testNow.Add(-2 * time.Hour),
		UpdatedAt:          testNow.Add(-time.Hour),
	}
	service := newTestService(store)

	result, err := service.ResetTemporaryPassword(context.Background(), ResetTemporaryPasswordInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      orgID,
		UserID:              memberUserID,
	})
	if err != nil {
		t.Fatalf("ResetTemporaryPassword() error = %v", err)
	}
	if result.TemporaryPassword != "temporary-password" {
		t.Fatalf("TemporaryPassword = %q, want generated password", result.TemporaryPassword)
	}
	if got := store.users[memberUserID]; got != originalUser {
		t.Fatalf("user mutated on reset:\n got: %#v\nwant: %#v", got, originalUser)
	}
	if got := store.memberships[key(orgID, memberUserID)]; got != originalMembership {
		t.Fatalf("membership mutated on reset:\n got: %#v\nwant: %#v", got, originalMembership)
	}
	credential := store.credentials[memberUserID]
	if credential.PasswordHash != "hash:temporary-password" {
		t.Fatalf("stored PasswordHash = %q, want reset hash", credential.PasswordHash)
	}
	if !credential.MustChangePassword || credential.PasswordChangedAt != nil {
		t.Fatalf("stored credential = %#v, want mandatory password change", credential)
	}
	if !store.revoked[memberUserID] {
		t.Fatalf("sessions for %q were not revoked", memberUserID)
	}
	if !result.Credential.MustChangePassword || result.Credential.PasswordChangedAt != nil {
		t.Fatalf("result credential = %#v, want must_change_password summary", result.Credential)
	}
	encoded := mustJSON(t, result)
	if strings.Contains(encoded, "password_hash") || strings.Contains(encoded, "hash:temporary-password") || strings.Contains(encoded, "old-password") {
		t.Fatalf("reset response leaked credential material: %s", encoded)
	}
}

func TestOwnerCanResetInactiveNonSelfUserTemporaryPassword(t *testing.T) {
	store := newFakeStore()
	store.users[secondOwnerUserID] = user(secondOwnerUserID, "Inactive User", "inactive@example.com", spine.EntityStateInactive)
	store.memberships[key(orgID, secondOwnerUserID)] = membership(secondOwnerMembershipID, orgID, secondOwnerUserID, spine.OrganizationMembershipRoleMember, spine.EntityStateInactive)
	store.credentials[secondOwnerUserID] = spine.UserPasswordCredential{
		UserID:             secondOwnerUserID,
		PasswordHash:       "hash:old-inactive-password",
		MustChangePassword: false,
		PasswordChangedAt:  ptrTime(testNow.Add(-time.Hour)),
		CreatedAt:          testNow.Add(-2 * time.Hour),
		UpdatedAt:          testNow.Add(-time.Hour),
	}
	service := newTestService(store)

	result, err := service.ResetTemporaryPassword(context.Background(), ResetTemporaryPasswordInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      orgID,
		UserID:              secondOwnerUserID,
	})
	if err != nil {
		t.Fatalf("ResetTemporaryPassword() error = %v", err)
	}
	if result.TemporaryPassword != "temporary-password" {
		t.Fatalf("TemporaryPassword = %q, want generated password", result.TemporaryPassword)
	}
	credential := store.credentials[secondOwnerUserID]
	if credential.PasswordHash != "hash:temporary-password" || !credential.MustChangePassword || credential.PasswordChangedAt != nil {
		t.Fatalf("credential = %#v, want reset temporary credential", credential)
	}
	if !store.revoked[secondOwnerUserID] {
		t.Fatalf("sessions for %q were not revoked", secondOwnerUserID)
	}
}

func TestResetTemporaryPasswordRequiresExistingOrganizationUser(t *testing.T) {
	store := newFakeStore()
	service := newTestService(store)

	_, err := service.ResetTemporaryPassword(context.Background(), ResetTemporaryPasswordInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      orgID,
		UserID:              otherExistingUserID,
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("ResetTemporaryPassword() error = %v, want ErrNotFound", err)
	}
	if _, ok := store.credentials[otherExistingUserID]; ok {
		t.Fatalf("credential created for non-member: %#v", store.credentials[otherExistingUserID])
	}
}

func TestSelfOwnerRoleDowngradeIsRejectedWithAnotherActiveOwner(t *testing.T) {
	for _, role := range []string{"admin", "member", "viewer"} {
		t.Run(role, func(t *testing.T) {
			store := newFakeStore()
			store.users[secondOwnerUserID] = user(secondOwnerUserID, "Second Owner", "second@example.com", spine.EntityStateActive)
			store.memberships[key(orgID, secondOwnerUserID)] = membership(secondOwnerMembershipID, orgID, secondOwnerUserID, spine.OrganizationMembershipRoleOwner, spine.EntityStateActive)
			originalMembership := store.memberships[key(orgID, ownerUserID)]
			service := newTestService(store)

			_, err := service.PatchUser(context.Background(), PatchUserInput{
				AuthenticatedUserID: ownerUserID,
				OrganizationID:      orgID,
				UserID:              ownerUserID,
				Role:                &role,
			})
			if !errors.Is(err, ErrSelfActionForbidden) {
				t.Fatalf("PatchUser() error = %v, want ErrSelfActionForbidden", err)
			}
			if got := store.memberships[key(orgID, ownerUserID)]; got != originalMembership {
				t.Fatalf("self membership mutated:\n got: %#v\nwant: %#v", got, originalMembership)
			}
		})
	}
}

func TestSelfOwnerMembershipDeactivationIsRejectedWithAnotherActiveOwner(t *testing.T) {
	store := newFakeStore()
	store.users[secondOwnerUserID] = user(secondOwnerUserID, "Second Owner", "second@example.com", spine.EntityStateActive)
	store.memberships[key(orgID, secondOwnerUserID)] = membership(secondOwnerMembershipID, orgID, secondOwnerUserID, spine.OrganizationMembershipRoleOwner, spine.EntityStateActive)
	originalMembership := store.memberships[key(orgID, ownerUserID)]
	nextState := "inactive"
	service := newTestService(store)

	_, err := service.PatchUser(context.Background(), PatchUserInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      orgID,
		UserID:              ownerUserID,
		State:               &nextState,
	})
	if !errors.Is(err, ErrSelfActionForbidden) {
		t.Fatalf("PatchUser() error = %v, want ErrSelfActionForbidden", err)
	}
	if got := store.memberships[key(orgID, ownerUserID)]; got != originalMembership {
		t.Fatalf("self membership mutated:\n got: %#v\nwant: %#v", got, originalMembership)
	}
}

func TestSelfDisplayNamePatchRemainsAllowed(t *testing.T) {
	store := newFakeStore()
	nextName := "Updated Owner"
	service := newTestService(store)

	result, err := service.PatchUser(context.Background(), PatchUserInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      orgID,
		UserID:              ownerUserID,
		DisplayName:         &nextName,
	})
	if err != nil {
		t.Fatalf("PatchUser() error = %v", err)
	}
	if result.User.DisplayName != nextName {
		t.Fatalf("DisplayName = %q, want %q", result.User.DisplayName, nextName)
	}
	if got := store.memberships[key(orgID, ownerUserID)].Role; got != spine.OrganizationMembershipRoleOwner {
		t.Fatalf("self role = %q, want owner", got)
	}
}

func TestSelfOwnerNoopRoleAndActiveStatePatchRemainAllowed(t *testing.T) {
	store := newFakeStore()
	nextRole := "owner"
	nextState := "active"
	service := newTestService(store)

	result, err := service.PatchUser(context.Background(), PatchUserInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      orgID,
		UserID:              ownerUserID,
		Role:                &nextRole,
		State:               &nextState,
	})
	if err != nil {
		t.Fatalf("PatchUser() error = %v", err)
	}
	if result.OrganizationMembership.Role != spine.OrganizationMembershipRoleOwner || result.OrganizationMembership.State != spine.EntityStateActive {
		t.Fatalf("membership = %#v, want active owner", result.OrganizationMembership)
	}
}

func TestSelfTemporaryPasswordResetIsRejected(t *testing.T) {
	store := newFakeStore()
	originalCredential := store.credentials[ownerUserID]
	service := newTestService(store)

	_, err := service.ResetTemporaryPassword(context.Background(), ResetTemporaryPasswordInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      orgID,
		UserID:              ownerUserID,
	})
	if !errors.Is(err, ErrSelfActionForbidden) {
		t.Fatalf("ResetTemporaryPassword() error = %v, want ErrSelfActionForbidden", err)
	}
	if got := store.credentials[ownerUserID]; !sameCredential(got, originalCredential) {
		t.Fatalf("self credential mutated:\n got: %#v\nwant: %#v", got, originalCredential)
	}
	if store.revoked[ownerUserID] {
		t.Fatalf("self sessions were revoked")
	}
}

func TestInvalidAndObserverRolesAreRejected(t *testing.T) {
	store := newFakeStore()
	service := newTestService(store)

	for _, role := range []string{"manager", "observer"} {
		t.Run(role, func(t *testing.T) {
			_, err := service.CreateUser(context.Background(), CreateUserInput{
				AuthenticatedUserID: ownerUserID,
				OrganizationID:      orgID,
				Email:               "dev@example.com",
				DisplayName:         "Dev",
				Role:                role,
			})
			var validationErr *ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("CreateUser() error = %v, want ValidationError", err)
			}
		})
	}
}

func TestCannotDemoteLastActiveOwner(t *testing.T) {
	store := newFakeStore()
	delete(store.memberships, key(orgID, memberUserID))
	nextRole := "admin"
	service := newTestService(store)

	_, err := service.PatchUser(context.Background(), PatchUserInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      orgID,
		UserID:              ownerUserID,
		Role:                &nextRole,
	})
	if !errors.Is(err, ErrLastActiveOwner) {
		t.Fatalf("PatchUser() error = %v, want ErrLastActiveOwner", err)
	}
}

func TestCannotDisableLastActiveOwner(t *testing.T) {
	store := newFakeStore()
	delete(store.memberships, key(orgID, memberUserID))
	nextState := "inactive"
	service := newTestService(store)

	_, err := service.PatchUser(context.Background(), PatchUserInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      orgID,
		UserID:              ownerUserID,
		State:               &nextState,
	})
	if !errors.Is(err, ErrLastActiveOwner) {
		t.Fatalf("PatchUser() error = %v, want ErrLastActiveOwner", err)
	}
}

func TestLastActiveOwnerGuardLocksOwnersBeforeCounting(t *testing.T) {
	store := newFakeStore()
	store.users[secondOwnerUserID] = user(secondOwnerUserID, "Second Owner", "second@example.com", spine.EntityStateActive)
	store.memberships[key(orgID, secondOwnerUserID)] = membership(secondOwnerMembershipID, orgID, secondOwnerUserID, spine.OrganizationMembershipRoleOwner, spine.EntityStateActive)
	nextRole := "member"
	service := newTestService(store)

	_, err := service.PatchUser(context.Background(), PatchUserInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      orgID,
		UserID:              secondOwnerUserID,
		Role:                &nextRole,
	})
	if err != nil {
		t.Fatalf("PatchUser() error = %v", err)
	}
	if got := strings.Join(store.calls, ","); got != "lock_active_owners,count_active_owners" {
		t.Fatalf("store calls = %q, want lock before count", got)
	}
}

func TestDisablingOrganizationMembershipDoesNotDisableGlobalUserOrRevokeSessions(t *testing.T) {
	store := newFakeStore()
	store.users[secondOwnerUserID] = user(secondOwnerUserID, "Second Owner", "second@example.com", spine.EntityStateActive)
	store.memberships[key(orgID, secondOwnerUserID)] = membership(secondOwnerMembershipID, orgID, secondOwnerUserID, spine.OrganizationMembershipRoleOwner, spine.EntityStateActive)
	nextState := "inactive"
	originalUser := store.users[secondOwnerUserID]
	service := newTestService(store)

	result, err := service.PatchUser(context.Background(), PatchUserInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      orgID,
		UserID:              secondOwnerUserID,
		State:               &nextState,
	})
	if err != nil {
		t.Fatalf("PatchUser() error = %v", err)
	}
	if got := store.users[secondOwnerUserID]; got != originalUser {
		t.Fatalf("global user mutated on membership disable:\n got: %#v\nwant: %#v", got, originalUser)
	}
	if result.User.State != spine.EntityStateActive || result.OrganizationMembership.State != spine.EntityStateInactive {
		t.Fatalf("result = %#v, want active user and inactive membership", result)
	}
	if store.revoked[secondOwnerUserID] {
		t.Fatalf("sessions for %q were revoked for membership-only disable", secondOwnerUserID)
	}
}

func TestOwnerCanReactivateInactiveOrganizationMembership(t *testing.T) {
	store := newFakeStore()
	store.memberships[key(orgID, memberUserID)] = membership(memberMembershipID, orgID, memberUserID, spine.OrganizationMembershipRoleMember, spine.EntityStateInactive)
	nextState := string(spine.EntityStateActive)
	originalUser := store.users[memberUserID]
	service := newTestService(store)

	result, err := service.PatchUser(context.Background(), PatchUserInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      orgID,
		UserID:              memberUserID,
		State:               &nextState,
	})
	if err != nil {
		t.Fatalf("PatchUser() error = %v", err)
	}
	if got := store.users[memberUserID]; got != originalUser {
		t.Fatalf("global user mutated on membership reactivation:\n got: %#v\nwant: %#v", got, originalUser)
	}
	if result.OrganizationMembership.State != spine.EntityStateActive {
		t.Fatalf("membership state = %q, want active", result.OrganizationMembership.State)
	}
	if store.revoked[memberUserID] {
		t.Fatalf("sessions for %q were revoked for membership-only reactivation", memberUserID)
	}
}

func TestInactiveOrganizationMembershipRejectsNonReactivationPatchAndPasswordReset(t *testing.T) {
	store := newFakeStore()
	originalMembership := membership(memberMembershipID, orgID, memberUserID, spine.OrganizationMembershipRoleMember, spine.EntityStateInactive)
	store.memberships[key(orgID, memberUserID)] = originalMembership
	nextRole := "viewer"
	service := newTestService(store)

	_, err := service.PatchUser(context.Background(), PatchUserInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      orgID,
		UserID:              memberUserID,
		Role:                &nextRole,
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("PatchUser() error = %v, want ErrNotFound", err)
	}
	if got := store.memberships[key(orgID, memberUserID)]; got != originalMembership {
		t.Fatalf("inactive membership mutated by role patch:\n got: %#v\nwant: %#v", got, originalMembership)
	}

	_, err = service.ResetTemporaryPassword(context.Background(), ResetTemporaryPasswordInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      orgID,
		UserID:              memberUserID,
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("ResetTemporaryPassword() error = %v, want ErrNotFound", err)
	}
	if store.revoked[memberUserID] {
		t.Fatalf("sessions for %q were revoked despite inactive membership", memberUserID)
	}
}

func TestPathIDsMustBeValidUUIDs(t *testing.T) {
	store := newFakeStore()
	service := newTestService(store)

	_, err := service.ListUsers(context.Background(), ListUsersInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      "not-a-uuid",
	})
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("ListUsers() error = %v, want ValidationError", err)
	}

	role := "member"
	_, err = service.PatchUser(context.Background(), PatchUserInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      orgID,
		UserID:              "not-a-uuid",
		Role:                &role,
	})
	if !errors.As(err, &validationErr) {
		t.Fatalf("PatchUser() error = %v, want ValidationError", err)
	}

	_, err = service.ResetTemporaryPassword(context.Background(), ResetTemporaryPasswordInput{
		AuthenticatedUserID: ownerUserID,
		OrganizationID:      orgID,
		UserID:              "not-a-uuid",
	})
	if !errors.As(err, &validationErr) {
		t.Fatalf("ResetTemporaryPassword() error = %v, want ValidationError", err)
	}
}

func newTestService(store *fakeStore) *Service {
	service := NewService(store, fakeTransactionRunner{})
	service.Clock = fixedClock{now: testNow}
	service.IDs = sequenceIDs{}
	service.Passwords = fixedPasswordGenerator{}
	service.Hasher = fixedHasher{}
	return service
}

type fakeStore struct {
	users              map[spine.UserID]spine.User
	memberships        map[string]spine.OrganizationMembership
	credentials        map[spine.UserID]spine.UserPasswordCredential
	revoked            map[spine.UserID]bool
	createUserConflict spine.User
	calls              []string
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		users: map[spine.UserID]spine.User{
			ownerUserID:  user(ownerUserID, "Owner", "owner@example.com", spine.EntityStateActive),
			memberUserID: user(memberUserID, "Member", "member@example.com", spine.EntityStateActive),
		},
		memberships: map[string]spine.OrganizationMembership{
			key(orgID, ownerUserID):  membership(ownerMembershipID, orgID, ownerUserID, spine.OrganizationMembershipRoleOwner, spine.EntityStateActive),
			key(orgID, memberUserID): membership(memberMembershipID, orgID, memberUserID, spine.OrganizationMembershipRoleMember, spine.EntityStateActive),
		},
		credentials: map[spine.UserID]spine.UserPasswordCredential{
			ownerUserID: {
				UserID:             ownerUserID,
				PasswordHash:       "hash:owner-password",
				MustChangePassword: false,
				PasswordChangedAt:  ptrTime(testNow.Add(-time.Hour)),
				CreatedAt:          testNow,
				UpdatedAt:          testNow,
			},
		},
		revoked: map[spine.UserID]bool{},
	}
}

func (s *fakeStore) ListOrganizationMemberships(_ context.Context, organizationID spine.OrganizationID) ([]spine.OrganizationMembership, error) {
	var result []spine.OrganizationMembership
	for _, userID := range []spine.UserID{ownerUserID, memberUserID, secondOwnerUserID, newUserID} {
		if membership, ok := s.memberships[key(organizationID, userID)]; ok {
			result = append(result, membership)
		}
	}
	return result, nil
}

func (s *fakeStore) GetUser(_ context.Context, userID spine.UserID) (spine.User, bool, error) {
	user, ok := s.users[userID]
	return user, ok, nil
}

func (s *fakeStore) GetUserByEmail(_ context.Context, email string) (spine.User, bool, error) {
	for _, user := range s.users {
		if strings.EqualFold(user.Email, email) {
			return user, true, nil
		}
	}
	return spine.User{}, false, nil
}

func (s *fakeStore) CreateUser(_ context.Context, user spine.User) (bool, error) {
	if s.createUserConflict.ID != "" {
		s.users[s.createUserConflict.ID] = s.createUserConflict
		return false, nil
	}
	s.users[user.ID] = user
	return true, nil
}

func (s *fakeStore) UpsertUser(_ context.Context, user spine.User) error {
	s.users[user.ID] = user
	return nil
}

func (s *fakeStore) GetOrganizationMembership(_ context.Context, organizationID spine.OrganizationID, userID spine.UserID) (spine.OrganizationMembership, bool, error) {
	membership, ok := s.memberships[key(organizationID, userID)]
	return membership, ok, nil
}

func (s *fakeStore) CreateOrganizationMembership(_ context.Context, membership spine.OrganizationMembership) error {
	membershipKey := key(membership.OrganizationID, membership.UserID)
	if _, ok := s.memberships[membershipKey]; ok {
		return errors.New("organization membership already exists")
	}
	s.memberships[membershipKey] = membership
	return nil
}

func (s *fakeStore) UpsertOrganizationMembership(_ context.Context, membership spine.OrganizationMembership) error {
	s.memberships[key(membership.OrganizationID, membership.UserID)] = membership
	return nil
}

func (s *fakeStore) GetPasswordCredential(_ context.Context, userID spine.UserID) (spine.UserPasswordCredential, bool, error) {
	credential, ok := s.credentials[userID]
	return credential, ok, nil
}

func (s *fakeStore) UpsertPasswordCredential(_ context.Context, credential spine.UserPasswordCredential) error {
	s.credentials[credential.UserID] = credential
	return nil
}

func (s *fakeStore) LockActiveOwnerMemberships(_ context.Context, _ spine.OrganizationID) error {
	s.calls = append(s.calls, "lock_active_owners")
	return nil
}

func (s *fakeStore) CountActiveOwners(_ context.Context, organizationID spine.OrganizationID) (int, error) {
	s.calls = append(s.calls, "count_active_owners")
	count := 0
	for _, membership := range s.memberships {
		user := s.users[membership.UserID]
		if membership.OrganizationID == organizationID &&
			membership.Role == spine.OrganizationMembershipRoleOwner &&
			membership.State == spine.EntityStateActive &&
			user.State == spine.EntityStateActive {
			count++
		}
	}
	return count, nil
}

func (s *fakeStore) RevokeActiveSessionsForUser(_ context.Context, userID spine.UserID, _ time.Time) error {
	s.revoked[userID] = true
	return nil
}

type fakeTransactionRunner struct{}

func (fakeTransactionRunner) RunReadCommitted(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type sequenceIDs struct{}

func (sequenceIDs) NewUserID() (spine.UserID, error) {
	return newUserID, nil
}

func (sequenceIDs) NewOrganizationMembershipID() (spine.OrganizationMembershipID, error) {
	return newMembershipID, nil
}

type fixedPasswordGenerator struct{}

func (fixedPasswordGenerator) NewPassword() (string, error) {
	return "temporary-password", nil
}

type fixedHasher struct{}

func (fixedHasher) HashPassword(input string) (string, error) {
	return "hash:" + input, nil
}

func user(id spine.UserID, displayName string, email string, state spine.EntityState) spine.User {
	return spine.User{
		ID:          id,
		DisplayName: displayName,
		Email:       email,
		State:       state,
		CreatedAt:   testNow,
		UpdatedAt:   testNow,
	}
}

func membership(id spine.OrganizationMembershipID, organizationID spine.OrganizationID, userID spine.UserID, role spine.OrganizationMembershipRole, state spine.EntityState) spine.OrganizationMembership {
	return spine.OrganizationMembership{
		ID:             id,
		OrganizationID: organizationID,
		UserID:         userID,
		Role:           role,
		State:          state,
		CreatedAt:      testNow,
		UpdatedAt:      testNow,
	}
}

func key(organizationID spine.OrganizationID, userID spine.UserID) string {
	return string(organizationID) + ":" + string(userID)
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	return string(encoded)
}

func sameCredential(a spine.UserPasswordCredential, b spine.UserPasswordCredential) bool {
	return a.UserID == b.UserID &&
		a.PasswordHash == b.PasswordHash &&
		a.MustChangePassword == b.MustChangePassword &&
		sameTimePtr(a.PasswordChangedAt, b.PasswordChangedAt) &&
		a.CreatedAt.Equal(b.CreatedAt) &&
		a.UpdatedAt.Equal(b.UpdatedAt)
}

func sameTimePtr(a *time.Time, b *time.Time) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return a.Equal(*b)
	}
}

const (
	orgID                     spine.OrganizationID           = "018f0000-0000-7000-8000-000000000002"
	otherOrgID                spine.OrganizationID           = "018f0000-0000-7000-8000-000000000099"
	ownerUserID               spine.UserID                   = "018f0000-0000-7000-8000-000000000001"
	memberUserID              spine.UserID                   = "018f0000-0000-7000-8000-000000000003"
	secondOwnerUserID         spine.UserID                   = "018f0000-0000-7000-8000-000000000004"
	newUserID                 spine.UserID                   = "018f0000-0000-7000-8000-000000000005"
	otherExistingUserID       spine.UserID                   = "018f0000-0000-7000-8000-000000000006"
	ownerMembershipID         spine.OrganizationMembershipID = "018f0000-0000-7000-8000-000000000011"
	memberMembershipID        spine.OrganizationMembershipID = "018f0000-0000-7000-8000-000000000012"
	secondOwnerMembershipID   spine.OrganizationMembershipID = "018f0000-0000-7000-8000-000000000013"
	newMembershipID           spine.OrganizationMembershipID = "018f0000-0000-7000-8000-000000000014"
	otherExistingMembershipID spine.OrganizationMembershipID = "018f0000-0000-7000-8000-000000000015"
)

var testNow = time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
