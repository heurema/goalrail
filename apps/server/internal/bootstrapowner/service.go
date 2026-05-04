package bootstrapowner

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/auth/password"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

var (
	ErrInvalidInput               = errors.New("invalid bootstrap owner input")
	ErrExistingPasswordCredential = errors.New("password credential already exists")
	ErrBootstrappedOwnerExists    = errors.New("bootstrapped owner already exists")
)

type Store interface {
	GetSelfHostedInstallation(ctx context.Context) (spine.Installation, bool, error)
	UpsertInstallation(ctx context.Context, installation spine.Installation) error
	GetPrimaryOrganization(ctx context.Context, installationID spine.InstallationID) (spine.Organization, bool, error)
	UpsertOrganization(ctx context.Context, org spine.Organization) error
	GetBootstrappedOwner(ctx context.Context, organizationID spine.OrganizationID) (spine.User, spine.UserPasswordCredential, bool, error)
	GetUserByEmail(ctx context.Context, email string) (spine.User, bool, error)
	UpsertUser(ctx context.Context, user spine.User) error
	GetOrganizationMembership(ctx context.Context, organizationID spine.OrganizationID, userID spine.UserID) (spine.OrganizationMembership, bool, error)
	UpsertOrganizationMembership(ctx context.Context, membership spine.OrganizationMembership) error
	GetPasswordCredential(ctx context.Context, userID spine.UserID) (spine.UserPasswordCredential, bool, error)
	CreatePasswordCredential(ctx context.Context, credential spine.UserPasswordCredential) error
}

type IDGenerator interface {
	NewInstallationID() (spine.InstallationID, error)
	NewOrganizationID() (spine.OrganizationID, error)
	NewUserID() (spine.UserID, error)
	NewOrganizationMembershipID() (spine.OrganizationMembershipID, error)
}

type PasswordGenerator interface {
	NewPassword() (string, error)
}

type PasswordHasher interface {
	HashPassword(password string) (string, error)
}

type Clock interface {
	Now() time.Time
}

type Service struct {
	Store     Store
	IDs       IDGenerator
	Passwords PasswordGenerator
	Hasher    PasswordHasher
	Clock     Clock
}

type Input struct {
	Email            string
	DisplayName      string
	OrganizationSlug string
	OrganizationName string
	PublicBaseURL    string
}

type Result struct {
	Installation              spine.Installation
	Organization              spine.Organization
	User                      spine.User
	Membership                spine.OrganizationMembership
	TemporaryPassword         string
	PasswordCredentialCreated bool
}

func NewService(store Store) *Service {
	if store == nil {
		panic("bootstrapowner: store is required")
	}
	return &Service{
		Store:     store,
		IDs:       SpineIDGenerator{},
		Passwords: CryptoPasswordGenerator{Bytes: 18},
		Hasher:    Argon2idHasher{},
		Clock:     SystemClock{},
	}
}

func (s *Service) BootstrapOwner(ctx context.Context, input Input) (Result, error) {
	normalized, err := NormalizeInput(input)
	if err != nil {
		return Result{}, err
	}

	now := s.Clock.Now().UTC()
	installation, err := s.upsertInstallation(ctx, normalized.PublicBaseURL, now)
	if err != nil {
		return Result{}, err
	}
	org, err := s.upsertPrimaryOrganization(ctx, installation.ID, normalized.OrganizationSlug, normalized.OrganizationName, now)
	if err != nil {
		return Result{}, err
	}
	bootstrappedOwner, _, ok, err := s.Store.GetBootstrappedOwner(ctx, org.ID)
	if err != nil {
		return Result{}, fmt.Errorf("get bootstrapped owner: %w", err)
	}
	if ok && !strings.EqualFold(bootstrappedOwner.Email, normalized.Email) {
		return Result{}, fmt.Errorf("%w: organization %s already has credentialed owner %s", ErrBootstrappedOwnerExists, org.ID, bootstrappedOwner.Email)
	}
	user, err := s.upsertUser(ctx, normalized.Email, normalized.DisplayName, now)
	if err != nil {
		return Result{}, err
	}
	membership, err := s.upsertOwnerMembership(ctx, org.ID, user.ID, now)
	if err != nil {
		return Result{}, err
	}

	_, ok, err = s.Store.GetPasswordCredential(ctx, user.ID)
	if err != nil {
		return Result{}, fmt.Errorf("get password credential: %w", err)
	}
	if ok {
		return Result{
			Installation:              installation,
			Organization:              org,
			User:                      user,
			Membership:                membership,
			PasswordCredentialCreated: false,
			TemporaryPassword:         "",
		}, nil
	}
	temporaryPassword, err := s.Passwords.NewPassword()
	if err != nil {
		return Result{}, fmt.Errorf("generate temporary password: %w", err)
	}
	passwordHash, err := s.Hasher.HashPassword(temporaryPassword)
	if err != nil {
		return Result{}, fmt.Errorf("hash temporary password: %w", err)
	}

	err = s.Store.CreatePasswordCredential(ctx, spine.UserPasswordCredential{
		UserID:             user.ID,
		PasswordHash:       passwordHash,
		MustChangePassword: true,
		PasswordChangedAt:  nil,
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	if err != nil {
		if errors.Is(err, ErrExistingPasswordCredential) {
			return Result{
				Installation:              installation,
				Organization:              org,
				User:                      user,
				Membership:                membership,
				PasswordCredentialCreated: false,
				TemporaryPassword:         "",
			}, nil
		}
		return Result{}, fmt.Errorf("create password credential: %w", err)
	}

	return Result{
		Installation:              installation,
		Organization:              org,
		User:                      user,
		Membership:                membership,
		PasswordCredentialCreated: true,
		TemporaryPassword:         temporaryPassword,
	}, nil
}

func (s *Service) upsertInstallation(ctx context.Context, publicBaseURL string, now time.Time) (spine.Installation, error) {
	installation, ok, err := s.Store.GetSelfHostedInstallation(ctx)
	if err != nil {
		return spine.Installation{}, fmt.Errorf("get self-hosted installation: %w", err)
	}
	if !ok {
		id, err := s.IDs.NewInstallationID()
		if err != nil {
			return spine.Installation{}, fmt.Errorf("new installation id: %w", err)
		}
		installation = spine.Installation{
			ID:            id,
			Mode:          spine.InstallationModeSelfHosted,
			PublicBaseURL: publicBaseURL,
			State:         spine.EntityStateActive,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
	} else {
		installation.Mode = spine.InstallationModeSelfHosted
		installation.PublicBaseURL = publicBaseURL
		installation.State = spine.EntityStateActive
		installation.UpdatedAt = now
	}
	if err := s.Store.UpsertInstallation(ctx, installation); err != nil {
		return spine.Installation{}, fmt.Errorf("upsert installation: %w", err)
	}
	return installation, nil
}

func (s *Service) upsertPrimaryOrganization(ctx context.Context, installationID spine.InstallationID, slug string, name string, now time.Time) (spine.Organization, error) {
	org, ok, err := s.Store.GetPrimaryOrganization(ctx, installationID)
	if err != nil {
		return spine.Organization{}, fmt.Errorf("get primary organization: %w", err)
	}
	if !ok {
		id, err := s.IDs.NewOrganizationID()
		if err != nil {
			return spine.Organization{}, fmt.Errorf("new organization id: %w", err)
		}
		org = spine.Organization{
			ID:             id,
			InstallationID: installationID,
			Slug:           slug,
			DisplayName:    name,
			State:          spine.EntityStateActive,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
	} else {
		org.InstallationID = installationID
		org.Slug = slug
		org.DisplayName = name
		org.State = spine.EntityStateActive
		org.UpdatedAt = now
	}
	if err := s.Store.UpsertOrganization(ctx, org); err != nil {
		return spine.Organization{}, fmt.Errorf("upsert organization: %w", err)
	}
	return org, nil
}

func (s *Service) upsertUser(ctx context.Context, email string, displayName string, now time.Time) (spine.User, error) {
	user, ok, err := s.Store.GetUserByEmail(ctx, email)
	if err != nil {
		return spine.User{}, fmt.Errorf("get user by email: %w", err)
	}
	if !ok {
		id, err := s.IDs.NewUserID()
		if err != nil {
			return spine.User{}, fmt.Errorf("new user id: %w", err)
		}
		user = spine.User{
			ID:          id,
			DisplayName: displayName,
			Email:       email,
			State:       spine.EntityStateActive,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	} else {
		user.DisplayName = displayName
		user.Email = email
		user.State = spine.EntityStateActive
		user.UpdatedAt = now
	}
	if err := s.Store.UpsertUser(ctx, user); err != nil {
		return spine.User{}, fmt.Errorf("upsert user: %w", err)
	}
	return user, nil
}

func (s *Service) upsertOwnerMembership(ctx context.Context, orgID spine.OrganizationID, userID spine.UserID, now time.Time) (spine.OrganizationMembership, error) {
	membership, ok, err := s.Store.GetOrganizationMembership(ctx, orgID, userID)
	if err != nil {
		return spine.OrganizationMembership{}, fmt.Errorf("get organization membership: %w", err)
	}
	if !ok {
		id, err := s.IDs.NewOrganizationMembershipID()
		if err != nil {
			return spine.OrganizationMembership{}, fmt.Errorf("new organization membership id: %w", err)
		}
		membership = spine.OrganizationMembership{
			ID:             id,
			OrganizationID: orgID,
			UserID:         userID,
			Role:           spine.OrganizationMembershipRoleOwner,
			State:          spine.EntityStateActive,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
	} else {
		membership.Role = spine.OrganizationMembershipRoleOwner
		membership.State = spine.EntityStateActive
		membership.UpdatedAt = now
	}
	if err := s.Store.UpsertOrganizationMembership(ctx, membership); err != nil {
		return spine.OrganizationMembership{}, fmt.Errorf("upsert organization membership: %w", err)
	}
	return membership, nil
}

func NormalizeInput(input Input) (Input, error) {
	normalized := Input{
		Email:            strings.ToLower(strings.TrimSpace(input.Email)),
		DisplayName:      strings.TrimSpace(input.DisplayName),
		OrganizationSlug: strings.ToLower(strings.TrimSpace(input.OrganizationSlug)),
		OrganizationName: strings.TrimSpace(input.OrganizationName),
		PublicBaseURL:    strings.TrimSpace(input.PublicBaseURL),
	}
	if normalized.Email == "" || !strings.Contains(normalized.Email, "@") {
		return Input{}, fmt.Errorf("%w: --email is required and must look like an email address", ErrInvalidInput)
	}
	if normalized.DisplayName == "" {
		return Input{}, fmt.Errorf("%w: --display-name is required", ErrInvalidInput)
	}
	if normalized.OrganizationName == "" {
		return Input{}, fmt.Errorf("%w: --organization-name is required", ErrInvalidInput)
	}
	if !validOrganizationSlug(normalized.OrganizationSlug) {
		return Input{}, fmt.Errorf("%w: --organization-slug must contain lowercase letters, numbers, and hyphens", ErrInvalidInput)
	}
	publicBaseURL, err := normalizePublicBaseURL(normalized.PublicBaseURL)
	if err != nil {
		return Input{}, err
	}
	normalized.PublicBaseURL = publicBaseURL
	return normalized, nil
}

var organizationSlugPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

func validOrganizationSlug(slug string) bool {
	return organizationSlugPattern.MatchString(slug)
}

func normalizePublicBaseURL(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("%w: --public-base-url is required", ErrInvalidInput)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("%w: --public-base-url is invalid: %w", ErrInvalidInput, err)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return "", fmt.Errorf("%w: --public-base-url must use http or https", ErrInvalidInput)
	}
	if parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("%w: --public-base-url must be an absolute URL without userinfo, query, or fragment", ErrInvalidInput)
	}
	if parsed.Scheme == "http" && !isLocalhost(parsed.Hostname()) {
		return "", fmt.Errorf("%w: --public-base-url must use https except for localhost or loopback hosts", ErrInvalidInput)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	if parsed.Path == "" {
		parsed.Path = ""
	}
	return parsed.String(), nil
}

func isLocalhost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

type SpineIDGenerator struct{}

func (SpineIDGenerator) NewInstallationID() (spine.InstallationID, error) {
	return spine.NewInstallationID()
}

func (SpineIDGenerator) NewOrganizationID() (spine.OrganizationID, error) {
	return spine.NewOrganizationID()
}

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
	return time.Now()
}
