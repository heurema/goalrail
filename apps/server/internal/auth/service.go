package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/auth/password"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

const (
	defaultAccessTokenTTL  = 15 * time.Minute
	defaultRefreshTokenTTL = 30 * 24 * time.Hour
	refreshTokenBytes      = 32
	cliCodeChallengeMethod = "S256"
)

var (
	ErrInvalidCredentials     = errors.New("invalid email or password")
	ErrInactiveUser           = errors.New("user is inactive")
	ErrMembershipRequired     = errors.New("active organization membership is required")
	ErrSessionInvalid         = errors.New("session is invalid")
	ErrCurrentPassword        = errors.New("current password is invalid")
	ErrNewPasswordRequired    = errors.New("new password is required")
	ErrInvalidRedirectURI     = errors.New("redirect URI must be a localhost loopback URL")
	ErrPasswordChangeRequired = errors.New("password change required before CLI login")
	ErrCLIAuthCodeInvalid     = errors.New("CLI auth code is invalid")
	ErrCLIAuthCodeExpired     = errors.New("CLI auth code is expired")
	ErrCLIAuthCodeUsed        = errors.New("CLI auth code was already used")
	ErrStateInvalid           = errors.New("state is invalid")
)

type Store interface {
	GetUserByEmail(context.Context, string) (spine.User, bool, error)
	GetUser(context.Context, spine.UserID) (spine.User, bool, error)
	GetPasswordCredential(context.Context, spine.UserID) (spine.UserPasswordCredential, bool, error)
	UpsertPasswordCredential(context.Context, spine.UserPasswordCredential) error
	UpsertSession(context.Context, spine.UserSession) error
	GetSession(context.Context, spine.UserSessionID) (spine.UserSession, bool, error)
	GetSessionByRefreshTokenHash(context.Context, string) (spine.UserSession, bool, error)
	GetPrimaryOrganizationMembership(context.Context, spine.UserID) (spine.OrganizationMembership, bool, error)
	CreateCLIAuthCode(context.Context, spine.CLIAuthCode) error
	GetCLIAuthCodeByHash(context.Context, string) (spine.CLIAuthCode, bool, error)
	MarkCLIAuthCodeConsumed(context.Context, string, time.Time) (bool, error)
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
	newCLIAuthCode  func() (string, error)
	hashPassword    func(string) (string, error)
	verifyPassword  func(string, string) (bool, error)
}

type Option func(*Service)

func NewService(store Store, jwtSecret string, options ...Option) *Service {
	if store == nil {
		panic("auth: store is required")
	}
	service := &Service{
		store:           store,
		accessTokens:    NewAccessTokenManager(jwtSecret),
		clock:           SystemClock{},
		accessTokenTTL:  defaultAccessTokenTTL,
		refreshTokenTTL: defaultRefreshTokenTTL,
		newSessionID:    spine.NewUserSessionID,
		newRefreshToken: newOpaqueToken,
		newCLIAuthCode:  newOpaqueToken,
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

func WithCLIAuthCodeGenerator(generator func() (string, error)) Option {
	return func(service *Service) {
		if generator != nil {
			service.newCLIAuthCode = generator
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

type CLILoginInput struct {
	Email         string
	Password      string
	RedirectURI   string
	State         string
	CodeChallenge string
}

type CLILoginResult struct {
	RedirectURI string
}

type CLIExchangeInput struct {
	Code         string `json:"code"`
	State        string `json:"state"`
	CodeVerifier string `json:"code_verifier"`
}

type CLIExchangeResult struct {
	UserID               spine.UserID `json:"user_id"`
	AccessToken          string       `json:"access_token"`
	AccessTokenExpiresAt time.Time    `json:"access_token_expires_at"`
	TokenType            string       `json:"token_type"`
	RefreshToken         string       `json:"refresh_token"`
}

type RefreshInput struct {
	RefreshToken string `json:"refresh_token"`
}

type RefreshResult struct {
	UserID               spine.UserID `json:"user_id"`
	AccessToken          string       `json:"access_token"`
	AccessTokenExpiresAt time.Time    `json:"access_token_expires_at"`
	TokenType            string       `json:"token_type"`
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

type LogoutResult struct {
	Revoked bool `json:"revoked"`
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
	now := s.clock.Now().UTC()
	user, credential, membership, err := s.verifyLoginCredentials(ctx, input.Email, input.Password)
	if err != nil {
		return LoginResult{}, err
	}

	tokenResult, err := s.createTokenSession(ctx, user.ID, now)
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		UserID:                 user.ID,
		AccessToken:            tokenResult.AccessToken,
		AccessTokenExpiresAt:   tokenResult.AccessTokenExpiresAt,
		TokenType:              tokenResult.TokenType,
		RefreshToken:           tokenResult.RefreshToken,
		RefreshTokenExpiresAt:  tokenResult.RefreshTokenExpiresAt,
		MustChangePassword:     credential.MustChangePassword,
		OrganizationMembership: membership,
	}, nil
}

func (s *Service) StartCLILogin(ctx context.Context, input CLILoginInput) (CLILoginResult, error) {
	if err := ValidateLoopbackRedirectURI(input.RedirectURI); err != nil {
		return CLILoginResult{}, err
	}
	if strings.TrimSpace(input.State) == "" {
		return CLILoginResult{}, ErrStateInvalid
	}
	codeChallenge := strings.TrimSpace(input.CodeChallenge)
	if !validCodeChallenge(codeChallenge) {
		return CLILoginResult{}, ErrCLIAuthCodeInvalid
	}
	user, credential, _, err := s.verifyLoginCredentials(ctx, input.Email, input.Password)
	if err != nil {
		return CLILoginResult{}, err
	}
	if credential.MustChangePassword {
		return CLILoginResult{}, ErrPasswordChangeRequired
	}

	now := s.clock.Now().UTC()
	code, err := s.newCLIAuthCode()
	if err != nil {
		return CLILoginResult{}, fmt.Errorf("new CLI auth code: %w", err)
	}
	authCode := spine.CLIAuthCode{
		CodeHash:            hashOpaqueToken(code),
		UserID:              user.ID,
		RedirectURI:         strings.TrimSpace(input.RedirectURI),
		State:               strings.TrimSpace(input.State),
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: cliCodeChallengeMethod,
		CreatedAt:           now,
		ExpiresAt:           now.Add(5 * time.Minute),
	}
	if err := s.store.CreateCLIAuthCode(ctx, authCode); err != nil {
		return CLILoginResult{}, fmt.Errorf("create CLI auth code: %w", err)
	}

	redirectURI, err := appendCodeAndState(authCode.RedirectURI, code, authCode.State)
	if err != nil {
		return CLILoginResult{}, err
	}
	return CLILoginResult{RedirectURI: redirectURI}, nil
}

func (s *Service) ExchangeCLIAuthCode(ctx context.Context, input CLIExchangeInput) (CLIExchangeResult, error) {
	if strings.TrimSpace(input.Code) == "" || strings.TrimSpace(input.State) == "" {
		return CLIExchangeResult{}, ErrCLIAuthCodeInvalid
	}
	codeVerifier := strings.TrimSpace(input.CodeVerifier)
	if codeVerifier == "" {
		return CLIExchangeResult{}, ErrCLIAuthCodeInvalid
	}

	now := s.clock.Now().UTC()
	codeHash := hashOpaqueToken(strings.TrimSpace(input.Code))
	authCode, ok, err := s.store.GetCLIAuthCodeByHash(ctx, codeHash)
	if err != nil {
		return CLIExchangeResult{}, fmt.Errorf("get CLI auth code: %w", err)
	}
	if !ok {
		return CLIExchangeResult{}, ErrCLIAuthCodeInvalid
	}
	if authCode.ConsumedAt != nil {
		return CLIExchangeResult{}, ErrCLIAuthCodeUsed
	}
	if !now.Before(authCode.ExpiresAt.UTC()) {
		return CLIExchangeResult{}, ErrCLIAuthCodeExpired
	}
	if authCode.State != strings.TrimSpace(input.State) {
		return CLIExchangeResult{}, ErrStateInvalid
	}
	if authCode.CodeChallengeMethod != cliCodeChallengeMethod || authCode.CodeChallenge == "" {
		return CLIExchangeResult{}, ErrCLIAuthCodeInvalid
	}
	if subtle.ConstantTimeCompare([]byte(codeChallengeS256(codeVerifier)), []byte(authCode.CodeChallenge)) != 1 {
		return CLIExchangeResult{}, ErrCLIAuthCodeInvalid
	}
	consumed, err := s.store.MarkCLIAuthCodeConsumed(ctx, codeHash, now)
	if err != nil {
		return CLIExchangeResult{}, fmt.Errorf("consume CLI auth code: %w", err)
	}
	if !consumed {
		return CLIExchangeResult{}, ErrCLIAuthCodeUsed
	}

	user, ok, err := s.store.GetUser(ctx, authCode.UserID)
	if err != nil {
		return CLIExchangeResult{}, fmt.Errorf("get user: %w", err)
	}
	if !ok {
		return CLIExchangeResult{}, ErrCLIAuthCodeInvalid
	}
	if user.State != spine.EntityStateActive {
		return CLIExchangeResult{}, ErrInactiveUser
	}
	membership, ok, err := s.store.GetPrimaryOrganizationMembership(ctx, user.ID)
	if err != nil {
		return CLIExchangeResult{}, fmt.Errorf("get organization membership: %w", err)
	}
	if !ok || membership.State != spine.EntityStateActive {
		return CLIExchangeResult{}, ErrMembershipRequired
	}

	tokenResult, err := s.createTokenSession(ctx, user.ID, now)
	if err != nil {
		return CLIExchangeResult{}, err
	}
	return CLIExchangeResult{
		UserID:               user.ID,
		AccessToken:          tokenResult.AccessToken,
		AccessTokenExpiresAt: tokenResult.AccessTokenExpiresAt,
		TokenType:            tokenResult.TokenType,
		RefreshToken:         tokenResult.RefreshToken,
	}, nil
}

func (s *Service) Refresh(ctx context.Context, input RefreshInput) (RefreshResult, error) {
	if strings.TrimSpace(input.RefreshToken) == "" {
		return RefreshResult{}, ErrSessionInvalid
	}

	now := s.clock.Now().UTC()
	session, ok, err := s.store.GetSessionByRefreshTokenHash(ctx, hashOpaqueToken(input.RefreshToken))
	if err != nil {
		return RefreshResult{}, fmt.Errorf("get user session by refresh token: %w", err)
	}
	if !ok || session.State != spine.UserSessionStateActive || !now.Before(session.ExpiresAt.UTC()) {
		return RefreshResult{}, ErrSessionInvalid
	}

	user, ok, err := s.store.GetUser(ctx, session.UserID)
	if err != nil {
		return RefreshResult{}, fmt.Errorf("get user: %w", err)
	}
	if !ok {
		return RefreshResult{}, ErrSessionInvalid
	}
	if user.State != spine.EntityStateActive {
		return RefreshResult{}, ErrInactiveUser
	}
	membership, ok, err := s.store.GetPrimaryOrganizationMembership(ctx, user.ID)
	if err != nil {
		return RefreshResult{}, fmt.Errorf("get organization membership: %w", err)
	}
	if !ok || membership.State != spine.EntityStateActive {
		return RefreshResult{}, ErrMembershipRequired
	}

	accessTokenExpiresAt := now.Add(s.accessTokenTTL)
	accessToken, err := s.accessTokens.Sign(AccessTokenClaims{
		UserID:    user.ID,
		SessionID: session.ID,
		IssuedAt:  now,
		ExpiresAt: accessTokenExpiresAt,
	})
	if err != nil {
		return RefreshResult{}, err
	}

	session.UpdatedAt = now
	session.LastUsedAt = &now
	if err := s.store.UpsertSession(ctx, session); err != nil {
		return RefreshResult{}, fmt.Errorf("update user session last use: %w", err)
	}

	return RefreshResult{
		UserID:               user.ID,
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessTokenExpiresAt,
		TokenType:            "Bearer",
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

func (s *Service) Logout(ctx context.Context, accessToken string) (LogoutResult, error) {
	authenticated, err := s.AuthenticateAccessToken(ctx, accessToken)
	if err != nil {
		return LogoutResult{}, err
	}

	now := s.clock.Now().UTC()
	session := authenticated.Session
	session.State = spine.UserSessionStateRevoked
	session.RevokedAt = &now
	session.UpdatedAt = now
	if err := s.store.UpsertSession(ctx, session); err != nil {
		return LogoutResult{}, fmt.Errorf("revoke user session: %w", err)
	}
	return LogoutResult{Revoked: true}, nil
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

type tokenSessionResult struct {
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	TokenType             string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

func (s *Service) verifyLoginCredentials(ctx context.Context, email string, inputPassword string) (spine.User, spine.UserPasswordCredential, spine.OrganizationMembership, error) {
	if strings.TrimSpace(email) == "" || inputPassword == "" {
		return spine.User{}, spine.UserPasswordCredential{}, spine.OrganizationMembership{}, ErrInvalidCredentials
	}

	user, ok, err := s.store.GetUserByEmail(ctx, normalizeEmail(email))
	if err != nil {
		return spine.User{}, spine.UserPasswordCredential{}, spine.OrganizationMembership{}, fmt.Errorf("get user by email: %w", err)
	}
	if !ok {
		return spine.User{}, spine.UserPasswordCredential{}, spine.OrganizationMembership{}, ErrInvalidCredentials
	}
	if user.State != spine.EntityStateActive {
		return spine.User{}, spine.UserPasswordCredential{}, spine.OrganizationMembership{}, ErrInactiveUser
	}

	credential, ok, err := s.store.GetPasswordCredential(ctx, user.ID)
	if err != nil {
		return spine.User{}, spine.UserPasswordCredential{}, spine.OrganizationMembership{}, fmt.Errorf("get password credential: %w", err)
	}
	if !ok {
		return spine.User{}, spine.UserPasswordCredential{}, spine.OrganizationMembership{}, ErrInvalidCredentials
	}
	match, err := s.verifyPassword(inputPassword, credential.PasswordHash)
	if err != nil {
		return spine.User{}, spine.UserPasswordCredential{}, spine.OrganizationMembership{}, fmt.Errorf("verify password: %w", err)
	}
	if !match {
		return spine.User{}, spine.UserPasswordCredential{}, spine.OrganizationMembership{}, ErrInvalidCredentials
	}

	membership, ok, err := s.store.GetPrimaryOrganizationMembership(ctx, user.ID)
	if err != nil {
		return spine.User{}, spine.UserPasswordCredential{}, spine.OrganizationMembership{}, fmt.Errorf("get organization membership: %w", err)
	}
	if !ok || membership.State != spine.EntityStateActive {
		return spine.User{}, spine.UserPasswordCredential{}, spine.OrganizationMembership{}, ErrMembershipRequired
	}
	return user, credential, membership, nil
}

func (s *Service) createTokenSession(ctx context.Context, userID spine.UserID, now time.Time) (tokenSessionResult, error) {
	sessionID, err := s.newSessionID()
	if err != nil {
		return tokenSessionResult{}, fmt.Errorf("new session id: %w", err)
	}
	refreshToken, err := s.newRefreshToken()
	if err != nil {
		return tokenSessionResult{}, fmt.Errorf("new refresh token: %w", err)
	}
	refreshTokenExpiresAt := now.Add(s.refreshTokenTTL)
	session := spine.UserSession{
		ID:               sessionID,
		UserID:           userID,
		RefreshTokenHash: hashOpaqueToken(refreshToken),
		State:            spine.UserSessionStateActive,
		CreatedAt:        now,
		UpdatedAt:        now,
		ExpiresAt:        refreshTokenExpiresAt,
	}

	accessTokenExpiresAt := now.Add(s.accessTokenTTL)
	accessToken, err := s.accessTokens.Sign(AccessTokenClaims{
		UserID:    userID,
		SessionID: sessionID,
		IssuedAt:  now,
		ExpiresAt: accessTokenExpiresAt,
	})
	if err != nil {
		return tokenSessionResult{}, err
	}
	if err := s.store.UpsertSession(ctx, session); err != nil {
		return tokenSessionResult{}, fmt.Errorf("create user session: %w", err)
	}

	return tokenSessionResult{
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessTokenExpiresAt,
		TokenType:             "Bearer",
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshTokenExpiresAt,
	}, nil
}

func ValidateLoopbackRedirectURI(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !parsed.IsAbs() {
		return ErrInvalidRedirectURI
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidRedirectURI
	}
	if parsed.User != nil || parsed.Fragment != "" || parsed.Port() == "" {
		return ErrInvalidRedirectURI
	}
	host := strings.Trim(parsed.Hostname(), "[]")
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return nil
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return nil
	}
	return ErrInvalidRedirectURI
}

func appendCodeAndState(rawRedirectURI string, code string, state string) (string, error) {
	parsed, err := url.Parse(rawRedirectURI)
	if err != nil {
		return "", ErrInvalidRedirectURI
	}
	query := parsed.Query()
	query.Set("code", code)
	query.Set("state", state)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
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

func codeChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func validCodeChallenge(challenge string) bool {
	if strings.TrimSpace(challenge) == "" {
		return false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(challenge)
	return err == nil && len(decoded) == sha256.Size
}

func hashOpaqueToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
