package usermanagement

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/heurema/goalrail/apps/server/internal/auth/password"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

var (
	ErrForbidden           = errors.New("user is not allowed to manage organization users")
	ErrNotFound            = errors.New("organization user not found")
	ErrUserExists          = errors.New("organization user already exists")
	ErrLastActiveOwner     = errors.New("last active owner cannot be disabled or demoted")
	ErrSelfActionForbidden = errors.New("self action forbidden")
)

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

type Store interface {
	ListOrganizationMemberships(context.Context, spine.OrganizationID) ([]spine.OrganizationMembership, error)
	GetUser(context.Context, spine.UserID) (spine.User, bool, error)
	GetUserByEmail(context.Context, string) (spine.User, bool, error)
	CreateUser(context.Context, spine.User) (bool, error)
	UpsertUser(context.Context, spine.User) error
	GetOrganizationMembership(context.Context, spine.OrganizationID, spine.UserID) (spine.OrganizationMembership, bool, error)
	CreateOrganizationMembership(context.Context, spine.OrganizationMembership) error
	UpsertOrganizationMembership(context.Context, spine.OrganizationMembership) error
	GetPasswordCredential(context.Context, spine.UserID) (spine.UserPasswordCredential, bool, error)
	UpsertPasswordCredential(context.Context, spine.UserPasswordCredential) error
	RevokeActiveSessionsForUser(context.Context, spine.UserID, time.Time) error
	LockActiveOwnerMemberships(context.Context, spine.OrganizationID) error
	CountActiveOwners(context.Context, spine.OrganizationID) (int, error)
}

type TransactionRunner interface {
	RunReadCommitted(context.Context, func(context.Context) error) error
}

type IDGenerator interface {
	NewUserID() (spine.UserID, error)
	NewOrganizationMembershipID() (spine.OrganizationMembershipID, error)
}

type PasswordGenerator interface {
	NewPassword() (string, error)
}

type PasswordHasher interface {
	HashPassword(string) (string, error)
}

type Clock interface {
	Now() time.Time
}

type Service struct {
	Store     Store
	TxRunner  TransactionRunner
	IDs       IDGenerator
	Passwords PasswordGenerator
	Hasher    PasswordHasher
	Clock     Clock
}

type UserRecord struct {
	User                   spine.User                   `json:"user"`
	OrganizationMembership spine.OrganizationMembership `json:"organization_membership"`
	Credential             CredentialSummary            `json:"credential"`
}

type CredentialSummary struct {
	MustChangePassword bool       `json:"must_change_password"`
	PasswordChangedAt  *time.Time `json:"password_changed_at"`
}

type ListUsersInput struct {
	AuthenticatedUserID spine.UserID
	OrganizationID      spine.OrganizationID
}

type ListUsersResult struct {
	Users []UserRecord `json:"users"`
}

type CreateUserInput struct {
	AuthenticatedUserID spine.UserID
	OrganizationID      spine.OrganizationID
	Email               string
	DisplayName         string
	Role                string
}

type CreateUserResult struct {
	User                   spine.User                   `json:"user"`
	OrganizationMembership spine.OrganizationMembership `json:"organization_membership"`
	Credential             CredentialSummary            `json:"credential"`
	TemporaryPassword      string                       `json:"temporary_password,omitempty"`
}

type PatchUserInput struct {
	AuthenticatedUserID spine.UserID
	OrganizationID      spine.OrganizationID
	UserID              spine.UserID
	DisplayName         *string
	Role                *string
	State               *string
}

type PatchUserResult struct {
	User                   spine.User                   `json:"user"`
	OrganizationMembership spine.OrganizationMembership `json:"organization_membership"`
	Credential             CredentialSummary            `json:"credential"`
}

type ResetTemporaryPasswordInput struct {
	AuthenticatedUserID spine.UserID
	OrganizationID      spine.OrganizationID
	UserID              spine.UserID
}

type ResetTemporaryPasswordResult struct {
	User                   spine.User                   `json:"user"`
	OrganizationMembership spine.OrganizationMembership `json:"organization_membership"`
	Credential             CredentialSummary            `json:"credential"`
	TemporaryPassword      string                       `json:"temporary_password"`
}

func NewService(store Store, txRunner TransactionRunner) *Service {
	if store == nil {
		panic("usermanagement: store is required")
	}
	if txRunner == nil {
		panic("usermanagement: transaction runner is required")
	}
	return &Service{
		Store:     store,
		TxRunner:  txRunner,
		IDs:       SpineIDGenerator{},
		Passwords: CryptoPasswordGenerator{Bytes: 18},
		Hasher:    Argon2idHasher{},
		Clock:     SystemClock{},
	}
}

func (s *Service) ListUsers(ctx context.Context, input ListUsersInput) (ListUsersResult, error) {
	if err := validateRequiredUUID(input.OrganizationID, "organization_id"); err != nil {
		return ListUsersResult{}, err
	}
	if err := validateRequiredUUID(input.AuthenticatedUserID, "authenticated_user_id"); err != nil {
		return ListUsersResult{}, err
	}

	var result ListUsersResult
	err := s.TxRunner.RunReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.requireOwner(txCtx, input.OrganizationID, input.AuthenticatedUserID); err != nil {
			return err
		}
		memberships, err := s.Store.ListOrganizationMemberships(txCtx, input.OrganizationID)
		if err != nil {
			return fmt.Errorf("list organization memberships: %w", err)
		}
		result.Users = make([]UserRecord, 0, len(memberships))
		for _, membership := range memberships {
			user, ok, err := s.Store.GetUser(txCtx, membership.UserID)
			if err != nil {
				return fmt.Errorf("get organization user: %w", err)
			}
			if !ok {
				continue
			}
			credential, ok, err := s.Store.GetPasswordCredential(txCtx, user.ID)
			if err != nil {
				return fmt.Errorf("get password credential summary: %w", err)
			}
			result.Users = append(result.Users, UserRecord{
				User:                   user,
				OrganizationMembership: membership,
				Credential:             credentialSummary(credential, ok),
			})
		}
		return nil
	})
	if err != nil {
		return ListUsersResult{}, err
	}
	return result, nil
}

func (s *Service) CreateUser(ctx context.Context, input CreateUserInput) (CreateUserResult, error) {
	normalized, err := normalizeCreateInput(input)
	if err != nil {
		return CreateUserResult{}, err
	}

	var result CreateUserResult
	err = s.TxRunner.RunReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.requireOwner(txCtx, normalized.OrganizationID, normalized.AuthenticatedUserID); err != nil {
			return err
		}

		now := s.Clock.Now().UTC()
		user, ok, err := s.Store.GetUserByEmail(txCtx, normalized.Email)
		if err != nil {
			return fmt.Errorf("get user by email: %w", err)
		}
		if ok {
			return ErrUserExists
		}

		id, err := s.IDs.NewUserID()
		if err != nil {
			return fmt.Errorf("new user id: %w", err)
		}
		user = spine.User{
			ID:          id,
			Email:       normalized.Email,
			DisplayName: normalized.DisplayName,
			State:       spine.EntityStateActive,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		created, err := s.Store.CreateUser(txCtx, user)
		if err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		if !created {
			existingUser, ok, err := s.Store.GetUserByEmail(txCtx, normalized.Email)
			if err != nil {
				return fmt.Errorf("get user by email after create conflict: %w", err)
			}
			if !ok || existingUser.State != spine.EntityStateActive {
				return ErrUserExists
			}
			return ErrUserExists
		}
		membershipID, err := s.IDs.NewOrganizationMembershipID()
		if err != nil {
			return fmt.Errorf("new organization membership id: %w", err)
		}
		membership := spine.OrganizationMembership{
			ID:             membershipID,
			OrganizationID: normalized.OrganizationID,
			UserID:         user.ID,
			Role:           spine.OrganizationMembershipRole(normalized.Role),
			State:          spine.EntityStateActive,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := s.Store.CreateOrganizationMembership(txCtx, membership); err != nil {
			if _, exists, lookupErr := s.Store.GetOrganizationMembership(txCtx, normalized.OrganizationID, user.ID); lookupErr == nil && exists {
				return ErrUserExists
			}
			return fmt.Errorf("create organization membership: %w", err)
		}

		temporaryPassword, credential, err := s.newTemporaryCredential(user.ID, now)
		if err != nil {
			return err
		}
		if err := s.Store.UpsertPasswordCredential(txCtx, credential); err != nil {
			return fmt.Errorf("upsert temporary password credential: %w", err)
		}

		result = CreateUserResult{
			User:                   user,
			OrganizationMembership: membership,
			Credential:             credentialSummary(credential, true),
			TemporaryPassword:      temporaryPassword,
		}
		return nil
	})
	if err != nil {
		return CreateUserResult{}, err
	}
	return result, nil
}

func (s *Service) PatchUser(ctx context.Context, input PatchUserInput) (PatchUserResult, error) {
	normalized, err := normalizePatchInput(input)
	if err != nil {
		return PatchUserResult{}, err
	}

	var result PatchUserResult
	err = s.TxRunner.RunReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.requireOwner(txCtx, normalized.OrganizationID, normalized.AuthenticatedUserID); err != nil {
			return err
		}
		user, ok, err := s.Store.GetUser(txCtx, normalized.UserID)
		if err != nil {
			return fmt.Errorf("get user: %w", err)
		}
		if !ok {
			return ErrNotFound
		}
		membership, ok, err := s.Store.GetOrganizationMembership(txCtx, normalized.OrganizationID, normalized.UserID)
		if err != nil {
			return fmt.Errorf("get organization membership: %w", err)
		}
		if !ok {
			return ErrNotFound
		}
		if membership.State != spine.EntityStateActive && !patchReactivatesMembership(normalized) {
			return ErrNotFound
		}

		nextUser := user
		nextMembership := membership
		now := s.Clock.Now().UTC()
		if normalized.DisplayName != nil {
			nextUser.DisplayName = *normalized.DisplayName
			nextUser.UpdatedAt = now
		}
		if normalized.Role != nil {
			nextMembership.Role = spine.OrganizationMembershipRole(*normalized.Role)
			nextMembership.UpdatedAt = now
		}
		if normalized.State != nil {
			state := spine.EntityState(*normalized.State)
			nextMembership.State = state
			nextMembership.UpdatedAt = now
		}

		if err := s.guardLastActiveOwner(txCtx, normalized.OrganizationID, user, membership, nextUser, nextMembership); err != nil {
			return err
		}
		if err := guardSelfPatch(normalized.AuthenticatedUserID, user, membership, nextMembership); err != nil {
			return err
		}
		if nextUser != user {
			if err := s.Store.UpsertUser(txCtx, nextUser); err != nil {
				return fmt.Errorf("upsert user: %w", err)
			}
		}
		if err := s.Store.UpsertOrganizationMembership(txCtx, nextMembership); err != nil {
			return fmt.Errorf("upsert organization membership: %w", err)
		}
		credential, ok, err := s.Store.GetPasswordCredential(txCtx, nextUser.ID)
		if err != nil {
			return fmt.Errorf("get password credential summary: %w", err)
		}
		result = PatchUserResult{
			User:                   nextUser,
			OrganizationMembership: nextMembership,
			Credential:             credentialSummary(credential, ok),
		}
		return nil
	})
	if err != nil {
		return PatchUserResult{}, err
	}
	return result, nil
}

func (s *Service) ResetTemporaryPassword(ctx context.Context, input ResetTemporaryPasswordInput) (ResetTemporaryPasswordResult, error) {
	normalized, err := normalizeResetTemporaryPasswordInput(input)
	if err != nil {
		return ResetTemporaryPasswordResult{}, err
	}

	var result ResetTemporaryPasswordResult
	err = s.TxRunner.RunReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.requireOwner(txCtx, normalized.OrganizationID, normalized.AuthenticatedUserID); err != nil {
			return err
		}
		if normalized.AuthenticatedUserID == normalized.UserID {
			return ErrSelfActionForbidden
		}
		user, ok, err := s.Store.GetUser(txCtx, normalized.UserID)
		if err != nil {
			return fmt.Errorf("get user: %w", err)
		}
		if !ok {
			return ErrNotFound
		}
		membership, ok, err := s.Store.GetOrganizationMembership(txCtx, normalized.OrganizationID, normalized.UserID)
		if err != nil {
			return fmt.Errorf("get organization membership: %w", err)
		}
		if !ok {
			return ErrNotFound
		}
		if membership.State != spine.EntityStateActive {
			return ErrNotFound
		}

		now := s.Clock.Now().UTC()
		temporaryPassword, credential, err := s.newTemporaryCredential(user.ID, now)
		if err != nil {
			return err
		}
		if err := s.Store.UpsertPasswordCredential(txCtx, credential); err != nil {
			return fmt.Errorf("upsert reset temporary password credential: %w", err)
		}
		if err := s.Store.RevokeActiveSessionsForUser(txCtx, user.ID, now); err != nil {
			return fmt.Errorf("revoke user sessions after temporary password reset: %w", err)
		}

		result = ResetTemporaryPasswordResult{
			User:                   user,
			OrganizationMembership: membership,
			Credential:             credentialSummary(credential, true),
			TemporaryPassword:      temporaryPassword,
		}
		return nil
	})
	if err != nil {
		return ResetTemporaryPasswordResult{}, err
	}
	return result, nil
}

func patchReactivatesMembership(input PatchUserInput) bool {
	return input.State != nil && *input.State == string(spine.EntityStateActive)
}

func (s *Service) createMembershipForExistingUser(ctx context.Context, user spine.User, input CreateUserInput, now time.Time) (spine.OrganizationMembership, bool, error) {
	if _, ok, err := s.Store.GetOrganizationMembership(ctx, input.OrganizationID, user.ID); err != nil {
		return spine.OrganizationMembership{}, false, fmt.Errorf("get existing organization membership: %w", err)
	} else if ok {
		return spine.OrganizationMembership{}, false, nil
	}

	membershipID, err := s.IDs.NewOrganizationMembershipID()
	if err != nil {
		return spine.OrganizationMembership{}, false, fmt.Errorf("new organization membership id: %w", err)
	}
	membership := spine.OrganizationMembership{
		ID:             membershipID,
		OrganizationID: input.OrganizationID,
		UserID:         user.ID,
		Role:           spine.OrganizationMembershipRole(input.Role),
		State:          spine.EntityStateActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.Store.CreateOrganizationMembership(ctx, membership); err != nil {
		if _, exists, lookupErr := s.Store.GetOrganizationMembership(ctx, input.OrganizationID, user.ID); lookupErr == nil && exists {
			return spine.OrganizationMembership{}, false, ErrUserExists
		}
		return spine.OrganizationMembership{}, false, fmt.Errorf("create organization membership: %w", err)
	}
	return membership, true, nil
}

func (s *Service) requireOwner(ctx context.Context, organizationID spine.OrganizationID, userID spine.UserID) error {
	membership, ok, err := s.Store.GetOrganizationMembership(ctx, organizationID, userID)
	if err != nil {
		return fmt.Errorf("get caller organization membership: %w", err)
	}
	if !ok || membership.OrganizationID != organizationID || membership.State != spine.EntityStateActive || membership.Role != spine.OrganizationMembershipRoleOwner {
		return ErrForbidden
	}
	return nil
}

func guardSelfPatch(authenticatedUserID spine.UserID, currentUser spine.User, currentMembership spine.OrganizationMembership, nextMembership spine.OrganizationMembership) error {
	if authenticatedUserID != currentUser.ID {
		return nil
	}
	if currentMembership.Role == spine.OrganizationMembershipRoleOwner && nextMembership.Role != spine.OrganizationMembershipRoleOwner {
		return ErrSelfActionForbidden
	}
	if currentMembership.Role == spine.OrganizationMembershipRoleOwner && nextMembership.State == spine.EntityStateInactive {
		return ErrSelfActionForbidden
	}
	return nil
}

func (s *Service) guardLastActiveOwner(ctx context.Context, organizationID spine.OrganizationID, currentUser spine.User, currentMembership spine.OrganizationMembership, nextUser spine.User, nextMembership spine.OrganizationMembership) error {
	if currentUser.State != spine.EntityStateActive || currentMembership.State != spine.EntityStateActive || currentMembership.Role != spine.OrganizationMembershipRoleOwner {
		return nil
	}
	stillActiveOwner := nextUser.State == spine.EntityStateActive &&
		nextMembership.State == spine.EntityStateActive &&
		nextMembership.Role == spine.OrganizationMembershipRoleOwner
	if stillActiveOwner {
		return nil
	}
	if err := s.Store.LockActiveOwnerMemberships(ctx, organizationID); err != nil {
		return fmt.Errorf("lock active owners: %w", err)
	}
	count, err := s.Store.CountActiveOwners(ctx, organizationID)
	if err != nil {
		return fmt.Errorf("count active owners: %w", err)
	}
	if count <= 1 {
		return ErrLastActiveOwner
	}
	return nil
}

func (s *Service) newTemporaryCredential(userID spine.UserID, now time.Time) (string, spine.UserPasswordCredential, error) {
	temporaryPassword, err := s.Passwords.NewPassword()
	if err != nil {
		return "", spine.UserPasswordCredential{}, fmt.Errorf("generate temporary password: %w", err)
	}
	passwordHash, err := s.Hasher.HashPassword(temporaryPassword)
	if err != nil {
		return "", spine.UserPasswordCredential{}, fmt.Errorf("hash temporary password: %w", err)
	}
	credential := spine.UserPasswordCredential{
		UserID:             userID,
		PasswordHash:       passwordHash,
		MustChangePassword: true,
		PasswordChangedAt:  nil,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	return temporaryPassword, credential, nil
}

func normalizeCreateInput(input CreateUserInput) (CreateUserInput, error) {
	normalized := CreateUserInput{
		AuthenticatedUserID: input.AuthenticatedUserID,
		OrganizationID:      input.OrganizationID,
		Email:               normalizeEmail(input.Email),
		DisplayName:         strings.TrimSpace(input.DisplayName),
		Role:                strings.TrimSpace(input.Role),
	}
	if err := validateRequiredUUID(normalized.OrganizationID, "organization_id"); err != nil {
		return CreateUserInput{}, err
	}
	if err := validateRequiredUUID(normalized.AuthenticatedUserID, "authenticated_user_id"); err != nil {
		return CreateUserInput{}, err
	}
	if normalized.Email == "" || !strings.Contains(normalized.Email, "@") {
		return CreateUserInput{}, &ValidationError{Message: "email is required and must look like an email address"}
	}
	if normalized.DisplayName == "" {
		return CreateUserInput{}, &ValidationError{Message: "display_name is required"}
	}
	if !validRole(normalized.Role) {
		return CreateUserInput{}, &ValidationError{Message: "role must be one of owner, admin, member, or viewer"}
	}
	return normalized, nil
}

func normalizePatchInput(input PatchUserInput) (PatchUserInput, error) {
	normalized := input
	if err := validateRequiredUUID(normalized.OrganizationID, "organization_id"); err != nil {
		return PatchUserInput{}, err
	}
	if err := validateRequiredUUID(normalized.AuthenticatedUserID, "authenticated_user_id"); err != nil {
		return PatchUserInput{}, err
	}
	if err := validateRequiredUUID(normalized.UserID, "user_id"); err != nil {
		return PatchUserInput{}, err
	}
	if normalized.DisplayName != nil {
		value := strings.TrimSpace(*normalized.DisplayName)
		if value == "" {
			return PatchUserInput{}, &ValidationError{Message: "display_name cannot be empty"}
		}
		normalized.DisplayName = &value
	}
	if normalized.Role != nil {
		value := strings.TrimSpace(*normalized.Role)
		if !validRole(value) {
			return PatchUserInput{}, &ValidationError{Message: "role must be one of owner, admin, member, or viewer"}
		}
		normalized.Role = &value
	}
	if normalized.State != nil {
		value := strings.TrimSpace(*normalized.State)
		if value != string(spine.EntityStateActive) && value != string(spine.EntityStateInactive) {
			return PatchUserInput{}, &ValidationError{Message: "state must be active or inactive"}
		}
		normalized.State = &value
	}
	if normalized.DisplayName == nil && normalized.Role == nil && normalized.State == nil {
		return PatchUserInput{}, &ValidationError{Message: "at least one of display_name, role, or state is required"}
	}
	return normalized, nil
}

func normalizeResetTemporaryPasswordInput(input ResetTemporaryPasswordInput) (ResetTemporaryPasswordInput, error) {
	normalized := input
	if err := validateRequiredUUID(normalized.OrganizationID, "organization_id"); err != nil {
		return ResetTemporaryPasswordInput{}, err
	}
	if err := validateRequiredUUID(normalized.AuthenticatedUserID, "authenticated_user_id"); err != nil {
		return ResetTemporaryPasswordInput{}, err
	}
	if err := validateRequiredUUID(normalized.UserID, "user_id"); err != nil {
		return ResetTemporaryPasswordInput{}, err
	}
	return normalized, nil
}

func validateRequiredUUID(value any, field string) error {
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" {
		return &ValidationError{Message: field + " is required"}
	}
	if _, err := uuid.Parse(text); err != nil {
		return &ValidationError{Message: field + " must be a valid UUID"}
	}
	return nil
}

func validRole(role string) bool {
	switch spine.OrganizationMembershipRole(role) {
	case spine.OrganizationMembershipRoleOwner, spine.OrganizationMembershipRoleAdmin, spine.OrganizationMembershipRoleMember, spine.OrganizationMembershipRoleViewer:
		return true
	default:
		return false
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func credentialSummary(credential spine.UserPasswordCredential, ok bool) CredentialSummary {
	if !ok {
		return CredentialSummary{}
	}
	return CredentialSummary{
		MustChangePassword: credential.MustChangePassword,
		PasswordChangedAt:  credential.PasswordChangedAt,
	}
}

type SpineIDGenerator struct{}

func (SpineIDGenerator) NewUserID() (spine.UserID, error) {
	return spine.NewUserID()
}

func (SpineIDGenerator) NewOrganizationMembershipID() (spine.OrganizationMembershipID, error) {
	return spine.NewOrganizationMembershipID()
}

type CryptoPasswordGenerator struct {
	Bytes int
}

func (g CryptoPasswordGenerator) NewPassword() (string, error) {
	length := g.Bytes
	if length <= 0 {
		length = 18
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

type Argon2idHasher struct{}

func (Argon2idHasher) HashPassword(value string) (string, error) {
	return password.HashPassword(value)
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}
