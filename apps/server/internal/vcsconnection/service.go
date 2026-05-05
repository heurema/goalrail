package vcsconnection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

const (
	EventTypeSetupStarted = "provider_connection.setup_started"
	EntityType            = "VcsConnection"
	SetupTTL              = 30 * time.Minute
)

var (
	ErrForbidden = errors.New("user is not allowed to manage VCS connections for this organization")
	ErrNotFound  = errors.New("VCS connection not found")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return e.Field + ": " + e.Message
}

type Store interface {
	GetOrganization(context.Context, spine.OrganizationID) (spine.Organization, bool, error)
	CreatePendingSetup(context.Context, spine.VcsConnection) error
	GetVcsConnection(context.Context, spine.VcsConnectionID) (spine.VcsConnection, bool, error)
}

type EventLog interface {
	Append(context.Context, spine.Event) error
}

type TransactionRunner interface {
	RunReadCommitted(context.Context, func(context.Context) error) error
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewVcsConnectionID() (spine.VcsConnectionID, error)
	NewEventID() (spine.EventID, error)
}

type CreateInput struct {
	AuthenticatedUserID spine.UserID
	Membership          spine.OrganizationMembership
	ProviderKind        string
	ProviderInstanceURL string
}

type GetInput struct {
	VcsConnectionID spine.VcsConnectionID
	Membership      spine.OrganizationMembership
}

type Service struct {
	Store    Store
	Events   EventLog
	TxRunner TransactionRunner
	Clock    Clock
	IDs      IDGenerator
}

func NewService(store Store, events EventLog, txRunner TransactionRunner, clock Clock, ids IDGenerator) *Service {
	return &Service{
		Store:    store,
		Events:   events,
		TxRunner: txRunner,
		Clock:    clock,
		IDs:      ids,
	}
}

func (s *Service) CreatePendingSetup(ctx context.Context, input CreateInput) (spine.VcsConnection, error) {
	normalized, err := normalizeCreateInput(input)
	if err != nil {
		return spine.VcsConnection{}, err
	}
	if err := authorizeCreate(normalized.Membership); err != nil {
		return spine.VcsConnection{}, err
	}

	organization, ok, err := s.Store.GetOrganization(ctx, normalized.Membership.OrganizationID)
	if err != nil {
		return spine.VcsConnection{}, fmt.Errorf("get organization: %w", err)
	}
	if !ok || organization.State != spine.EntityStateActive {
		return spine.VcsConnection{}, ErrForbidden
	}

	now := s.Clock.Now().UTC()
	connectionID, err := s.IDs.NewVcsConnectionID()
	if err != nil {
		return spine.VcsConnection{}, fmt.Errorf("new VCS connection id: %w", err)
	}
	connection := spine.VcsConnection{
		ID:                  connectionID,
		InstallationID:      organization.InstallationID,
		OrganizationID:      organization.ID,
		CreatedByUserID:     normalized.AuthenticatedUserID,
		ProviderKind:        normalized.ProviderKind,
		ProviderInstanceURL: normalized.ProviderInstanceURL,
		State:               spine.VcsConnectionStatePendingSetup,
		SetupExpiresAt:      now.Add(SetupTTL),
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	event, err := s.setupStartedEvent(connection, now)
	if err != nil {
		return spine.VcsConnection{}, err
	}

	if err := s.TxRunner.RunReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.Store.CreatePendingSetup(txCtx, connection); err != nil {
			return fmt.Errorf("create pending setup VCS connection: %w", err)
		}
		if err := s.Events.Append(txCtx, event); err != nil {
			return fmt.Errorf("append VCS connection setup event: %w", err)
		}
		return nil
	}); err != nil {
		return spine.VcsConnection{}, err
	}

	return connection, nil
}

func (s *Service) Get(ctx context.Context, input GetInput) (spine.VcsConnection, error) {
	if err := validateUUIDv7("id", string(input.VcsConnectionID)); err != nil {
		return spine.VcsConnection{}, err
	}
	if input.Membership.State != spine.EntityStateActive || strings.TrimSpace(string(input.Membership.OrganizationID)) == "" {
		return spine.VcsConnection{}, ErrForbidden
	}

	connection, ok, err := s.Store.GetVcsConnection(ctx, input.VcsConnectionID)
	if err != nil {
		return spine.VcsConnection{}, fmt.Errorf("get VCS connection: %w", err)
	}
	if !ok || connection.OrganizationID != input.Membership.OrganizationID {
		return spine.VcsConnection{}, ErrNotFound
	}
	return connection, nil
}

func normalizeCreateInput(input CreateInput) (CreateInput, error) {
	input.ProviderKind = strings.ToLower(strings.TrimSpace(input.ProviderKind))
	input.ProviderInstanceURL = strings.TrimSpace(input.ProviderInstanceURL)

	if strings.TrimSpace(string(input.AuthenticatedUserID)) == "" {
		return CreateInput{}, &ValidationError{Field: "user_id", Message: "authenticated user is required"}
	}
	if err := validateUUIDv7("user_id", string(input.AuthenticatedUserID)); err != nil {
		return CreateInput{}, err
	}
	if err := validateProviderKind(input.ProviderKind); err != nil {
		return CreateInput{}, err
	}
	normalizedURL, err := normalizeProviderInstanceURL(input.ProviderInstanceURL)
	if err != nil {
		return CreateInput{}, err
	}
	input.ProviderInstanceURL = normalizedURL
	return input, nil
}

func validateProviderKind(value string) error {
	if value == "" {
		return &ValidationError{Field: "provider_kind", Message: "is required"}
	}
	if len(value) > 64 {
		return &ValidationError{Field: "provider_kind", Message: "must be 64 characters or fewer"}
	}
	for i, r := range value {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '_' && i > 0 {
			continue
		}
		return &ValidationError{Field: "provider_kind", Message: "must be a lowercase provider-neutral slug"}
	}
	if !unicode.IsLetter(rune(value[0])) {
		return &ValidationError{Field: "provider_kind", Message: "must start with a letter"}
	}
	return nil
}

func normalizeProviderInstanceURL(raw string) (string, error) {
	if raw == "" {
		return "", &ValidationError{Field: "provider_instance_url", Message: "is required"}
	}
	parsed, err := url.Parse(raw)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" || parsed.Opaque != "" {
		return "", &ValidationError{Field: "provider_instance_url", Message: "must be an absolute http(s) URL"}
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", &ValidationError{Field: "provider_instance_url", Message: "must not include credentials, query, or fragment"}
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", &ValidationError{Field: "provider_instance_url", Message: "must use http or https"}
	}
	if parsed.Scheme == "http" && !isLocalhost(parsed.Hostname()) {
		return "", &ValidationError{Field: "provider_instance_url", Message: "must use https except for localhost URLs"}
	}
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed.String(), nil
}

func isLocalhost(host string) bool {
	host = strings.Trim(strings.ToLower(host), "[]")
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func validateUUIDv7(field string, value string) error {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return &ValidationError{Field: field, Message: "must be a UUID"}
	}
	if id.Version() != 7 {
		return &ValidationError{Field: field, Message: "must be a UUIDv7"}
	}
	return nil
}

func authorizeCreate(membership spine.OrganizationMembership) error {
	if membership.State != spine.EntityStateActive || strings.TrimSpace(string(membership.OrganizationID)) == "" {
		return ErrForbidden
	}
	switch membership.Role {
	case spine.OrganizationMembershipRoleOwner, spine.OrganizationMembershipRoleAdmin:
		return nil
	default:
		return ErrForbidden
	}
}

func (s *Service) setupStartedEvent(connection spine.VcsConnection, timestamp time.Time) (spine.Event, error) {
	eventID, err := s.IDs.NewEventID()
	if err != nil {
		return spine.Event{}, fmt.Errorf("new event id: %w", err)
	}
	payload, err := json.Marshal(map[string]any{
		"vcs_connection_id":     connection.ID,
		"installation_id":       connection.InstallationID,
		"organization_id":       connection.OrganizationID,
		"provider_kind":         connection.ProviderKind,
		"provider_instance_url": connection.ProviderInstanceURL,
		"state":                 connection.State,
		"setup_expires_at":      connection.SetupExpiresAt,
	})
	if err != nil {
		return spine.Event{}, fmt.Errorf("marshal VCS connection setup event payload: %w", err)
	}
	return spine.Event{
		ID:             eventID,
		Type:           EventTypeSetupStarted,
		EntityType:     EntityType,
		EntityID:       string(connection.ID),
		OrganizationID: connection.OrganizationID,
		Timestamp:      timestamp,
		Payload:        payload,
	}, nil
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}

type UUIDGenerator struct{}

func (UUIDGenerator) NewVcsConnectionID() (spine.VcsConnectionID, error) {
	return spine.NewVcsConnectionID()
}

func (UUIDGenerator) NewEventID() (spine.EventID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.EventID(id.String()), nil
}
