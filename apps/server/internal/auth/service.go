package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/auth/password"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

const (
	defaultAccessTokenTTL  = 15 * time.Minute
	defaultRefreshTokenTTL = 30 * 24 * time.Hour
	refreshTokenBytes      = 32
)

var (
	ErrStoreUnavailable    = errors.New("auth store is not configured")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInactiveUser        = errors.New("user is inactive")
	ErrMembershipRequired  = errors.New("active organization membership is required")
	ErrSessionInvalid      = errors.New("session is invalid")
	ErrCurrentPassword     = errors.New("current password is invalid")
	ErrNewPasswordRequired = errors.New("new password is required")
)

type Store interface {
	GetUserByEmail(context.Context, string) (spine.User, bool, error)
	GetUser(context.Context, spine.UserID) (spine.User, bool, error)
	GetPasswordCredential(context.Context, spine.UserID) (spine.UserPasswordCredential, bool, error)
	UpsertPasswordCredential(context.Context, spine.UserPasswordCredential) error
	UpsertSession(context.Context, spine.UserSession) error
	GetSession(context.Context, spine.UserSessionID) (spine.UserSession, bool, error)
	GetPrimaryOrganizationMembership(context.Context, spine.UserID) (spine.OrganizationMembership, bool, error)
}

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}

type Service struct {
	store           Store
	accessTokens    AccessTokenManager
	clock           Clock
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	newSessionID    func() (spine.UserSessionID, error)
	newRefreshToken func() (string, error)
	hashPassword    func(string) (string, error)
	verifyPassword  func(string, string) (bool, error)
}

type Option func(*Service)

func NewService(store Store, jwtSecret string, options ...Option) *Service {
	service := &Service{
		store:           store,
		accessTokens:    NewAccessTokenManager(jwtSecret),
		clock:           SystemClock{},
		accessTokenTTL:  defaultAccessTokenTTL,
		refreshTokenTTL: defaultRefreshTokenTTL,
		newSessionID:    spine.NewUserSessionID,
		newRefreshToken: newOpaqueToken,
		hashPassword:    password.HashPassword,
		verifyPassword:  password.VerifyPassword,
	}
	for _, option := range options {
		option(service)
	}
	return service
}

func WithClock(clock Clock) Option {
	return func(service *Service) {
		if clock != nil {
			service.clock = clock
		}
	}
}

func WithSessionIDGenerator(generator func() (spine.UserSessionID, error)) Option {
	return func(service *Service) {
		if generator != nil {
			service.newSessionID = generator
		}
	}
}

func WithRefreshTokenGenerator(generator func() (string, error)) Option {
	return func(service *Service) {
		if generator != nil {
			service.newRefreshToken = generator
		}
	}
}

func WithPasswordHasher(hash func(string) (string, error), verify func(string, string) (bool, error)) Option {
	return func(service *Service) {
		if hash != nil {
			service.hashPassword = hash
		}
		if verify != nil {
			service.verifyPassword = verify
		}
	}
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResult struct {
	UserID                 spine.UserID `json:"user_id"`
	AccessToken            string       `json:"access_token"`
	AccessTokenExpiresAt   time.Time    `json:"access_token_expires_at"`
	TokenType              string       `json:"token_type"`
	RefreshToken           string       `json:"refresh_token"`
	RefreshTokenExpiresAt  time.Time    `json:"refresh_token_expires_at"`
	MustChangePassword     bool         `json:"must_change_password"`
	OrganizationMembership spine.OrganizationMembership
}

type ChangePasswordInput struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type ChangePasswordResult struct {
	UserID             spine.UserID `json:"user_id"`
	MustChangePassword bool         `json:"must_change_password"`
	PasswordChangedAt  time.Time    `json:"password_changed_at"`
}

type Profile struct {
	User                   spine.User                   `json:"user"`
	OrganizationMembership spine.OrganizationMembership `json:"organization_membership"`
}

type AuthenticatedUser struct {
	User    spine.User
	Session spine.UserSession
	Claims  AccessTokenClaims
}

func (s *Service) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	if s.store == nil {
		return LoginResult{}, ErrStoreUnavailable
	}
	if strings.TrimSpace(input.Email) == "" || input.Password == "" {
		return LoginResult{}, ErrInvalidCredentials
	}

	now := s.clock.Now().UTC()
	user, ok, err := s.store.GetUserByEmail(ctx, normalizeEmail(input.Email))
	if err != nil {
		return LoginResult{}, fmt.Errorf("get user by email: %w", err)
	}
	if !ok {
		return LoginResult{}, ErrInvalidCredentials
	}
	if user.State != spine.EntityStateActive {
		return LoginResult{}, ErrInactiveUser
	}

	credential, ok, err := s.store.GetPasswordCredential(ctx, user.ID)
	if err != nil {
		return LoginResult{}, fmt.Errorf("get password credential: %w", err)
	}
	if !ok {
		return LoginResult{}, ErrInvalidCredentials
	}
	match, err := s.verifyPassword(input.Password, credential.PasswordHash)
	if err != nil {
		return LoginResult{}, fmt.Errorf("verify password: %w", err)
	}
	if !match {
		return LoginResult{}, ErrInvalidCredentials
	}

	membership, ok, err := s.store.GetPrimaryOrganizationMembership(ctx, user.ID)
	if err != nil {
		return LoginResult{}, fmt.Errorf("get organization membership: %w", err)
	}
	if !ok || membership.State != spine.EntityStateActive {
		return LoginResult{}, ErrMembershipRequired
	}

	sessionID, err := s.newSessionID()
	if err != nil {
		return LoginResult{}, fmt.Errorf("new session id: %w", err)
	}
	refreshToken, err := s.newRefreshToken()
	if err != nil {
		return LoginResult{}, fmt.Errorf("new refresh token: %w", err)
	}
	refreshTokenExpiresAt := now.Add(s.refreshTokenTTL)
	session := spine.UserSession{
		ID:               sessionID,
		UserID:           user.ID,
		RefreshTokenHash: hashOpaqueToken(refreshToken),
		State:            spine.UserSessionStateActive,
		CreatedAt:        now,
		UpdatedAt:        now,
		ExpiresAt:        refreshTokenExpiresAt,
	}

	accessTokenExpiresAt := now.Add(s.accessTokenTTL)
	accessToken, err := s.accessTokens.Sign(AccessTokenClaims{
		UserID:    user.ID,
		SessionID: sessionID,
		IssuedAt:  now,
		ExpiresAt: accessTokenExpiresAt,
	})
	if err != nil {
		return LoginResult{}, err
	}
	if err := s.store.UpsertSession(ctx, session); err != nil {
		return LoginResult{}, fmt.Errorf("create user session: %w", err)
	}

	return LoginResult{
		UserID:                 user.ID,
		AccessToken:            accessToken,
		AccessTokenExpiresAt:   accessTokenExpiresAt,
		TokenType:              "Bearer",
		RefreshToken:           refreshToken,
		RefreshTokenExpiresAt:  refreshTokenExpiresAt,
		MustChangePassword:     credential.MustChangePassword,
		OrganizationMembership: membership,
	}, nil
}

func (s *Service) ChangePassword(ctx context.Context, accessToken string, input ChangePasswordInput) (ChangePasswordResult, error) {
	if input.CurrentPassword == "" {
		return ChangePasswordResult{}, ErrCurrentPassword
	}
	if strings.TrimSpace(input.NewPassword) == "" {
		return ChangePasswordResult{}, ErrNewPasswordRequired
	}
	authenticated, err := s.AuthenticateAccessToken(ctx, accessToken)
	if err != nil {
		return ChangePasswordResult{}, err
	}
	credential, ok, err := s.store.GetPasswordCredential(ctx, authenticated.User.ID)
	if err != nil {
		return ChangePasswordResult{}, fmt.Errorf("get password credential: %w", err)
	}
	if !ok {
		return ChangePasswordResult{}, ErrInvalidCredentials
	}
	match, err := s.verifyPassword(input.CurrentPassword, credential.PasswordHash)
	if err != nil {
		return ChangePasswordResult{}, fmt.Errorf("verify current password: %w", err)
	}
	if !match {
		return ChangePasswordResult{}, ErrCurrentPassword
	}

	now := s.clock.Now().UTC()
	newHash, err := s.hashPassword(input.NewPassword)
	if err != nil {
		return ChangePasswordResult{}, fmt.Errorf("hash new password: %w", err)
	}
	credential.PasswordHash = newHash
	credential.MustChangePassword = false
	credential.PasswordChangedAt = &now
	credential.UpdatedAt = now
	if credential.CreatedAt.IsZero() {
		credential.CreatedAt = now
	}
	if err := s.store.UpsertPasswordCredential(ctx, credential); err != nil {
		return ChangePasswordResult{}, fmt.Errorf("update password credential: %w", err)
	}

	return ChangePasswordResult{
		UserID:             authenticated.User.ID,
		MustChangePassword: false,
		PasswordChangedAt:  now,
	}, nil
}

func (s *Service) Me(ctx context.Context, accessToken string) (Profile, error) {
	authenticated, err := s.AuthenticateAccessToken(ctx, accessToken)
	if err != nil {
		return Profile{}, err
	}
	membership, ok, err := s.store.GetPrimaryOrganizationMembership(ctx, authenticated.User.ID)
	if err != nil {
		return Profile{}, fmt.Errorf("get organization membership: %w", err)
	}
	if !ok || membership.State != spine.EntityStateActive {
		return Profile{}, ErrMembershipRequired
	}
	return Profile{
		User:                   authenticated.User,
		OrganizationMembership: membership,
	}, nil
}

func (s *Service) AuthenticateAccessToken(ctx context.Context, accessToken string) (AuthenticatedUser, error) {
	if s.store == nil {
		return AuthenticatedUser{}, ErrStoreUnavailable
	}
	claims, err := s.accessTokens.Validate(strings.TrimSpace(accessToken), s.clock.Now().UTC())
	if err != nil {
		return AuthenticatedUser{}, err
	}
	session, ok, err := s.store.GetSession(ctx, claims.SessionID)
	if err != nil {
		return AuthenticatedUser{}, fmt.Errorf("get user session: %w", err)
	}
	if !ok || session.UserID != claims.UserID || session.State != spine.UserSessionStateActive || !s.clock.Now().UTC().Before(session.ExpiresAt.UTC()) {
		return AuthenticatedUser{}, ErrSessionInvalid
	}
	user, ok, err := s.store.GetUser(ctx, claims.UserID)
	if err != nil {
		return AuthenticatedUser{}, fmt.Errorf("get user: %w", err)
	}
	if !ok {
		return AuthenticatedUser{}, ErrSessionInvalid
	}
	if user.State != spine.EntityStateActive {
		return AuthenticatedUser{}, ErrInactiveUser
	}
	return AuthenticatedUser{
		User:    user,
		Session: session,
		Claims:  claims,
	}, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func newOpaqueToken() (string, error) {
	var raw [refreshTokenBytes]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func hashOpaqueToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
