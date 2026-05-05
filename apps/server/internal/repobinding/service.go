package repobinding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

const (
	EventTypeInitialized = "repo_binding.initialized"
	EntityType           = "RepoBinding"
	initMessage          = "Repository binding initialized."
	existingMessage      = "Repository binding already initialized."
)

var (
	ErrProjectNotFound        = errors.New("project not found")
	ErrForbidden              = errors.New("user is not allowed to initialize repo binding for this project")
	ErrDifferentRepoBinding   = errors.New("project already has active repo binding for a different repository")
	ErrRepositoryAlreadyBound = errors.New("organization already has active repo binding for this repository")
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
	GetProject(context.Context, spine.ProjectID) (spine.Project, bool, error)
	GetActiveRepoBindingForProject(context.Context, spine.ProjectID) (spine.RepoBinding, bool, error)
	GetActiveRepoBindingByOrganizationAndRepository(context.Context, spine.OrganizationID, string, string) (spine.RepoBinding, bool, error)
	CreateRepoBinding(context.Context, spine.RepoBinding) error
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
	NewRepoBindingID() (spine.RepoBindingID, error)
	NewEventID() (spine.EventID, error)
}

type InitInput struct {
	ProjectID             spine.ProjectID
	AuthenticatedUserID   spine.UserID
	Membership            spine.OrganizationMembership
	Provider              string
	RepositoryFullName    string
	RepositoryURL         string
	ProviderDefaultBranch string
	WorkflowBaseBranch    string
	LocalRemoteName       string
	LocalHeadSHA          string
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

func (s *Service) Init(ctx context.Context, input InitInput) (spine.RepoBindingInitResult, error) {
	normalized, err := normalizeInput(input)
	if err != nil {
		return spine.RepoBindingInitResult{}, err
	}

	project, ok, err := s.Store.GetProject(ctx, normalized.ProjectID)
	if err != nil {
		return spine.RepoBindingInitResult{}, fmt.Errorf("get project: %w", err)
	}
	if !ok || project.State != spine.EntityStateActive {
		return spine.RepoBindingInitResult{}, ErrProjectNotFound
	}
	if err := authorize(project, normalized); err != nil {
		return spine.RepoBindingInitResult{}, err
	}

	existing, ok, err := s.Store.GetActiveRepoBindingForProject(ctx, project.ID)
	if err != nil {
		return spine.RepoBindingInitResult{}, fmt.Errorf("get active repo binding for project: %w", err)
	}
	if ok {
		if !sameRepository(existing, normalized) {
			return spine.RepoBindingInitResult{}, ErrDifferentRepoBinding
		}
		return initResult(existing, false, existingMessage), nil
	}

	existing, ok, err = s.Store.GetActiveRepoBindingByOrganizationAndRepository(ctx, project.OrganizationID, normalized.Provider, normalized.RepositoryFullName)
	if err != nil {
		return spine.RepoBindingInitResult{}, fmt.Errorf("get active repo binding by organization repository: %w", err)
	}
	if ok {
		if existing.ProjectID == project.ID {
			return initResult(existing, false, existingMessage), nil
		}
		return spine.RepoBindingInitResult{}, ErrRepositoryAlreadyBound
	}

	now := s.Clock.Now().UTC()
	bindingID, err := s.IDs.NewRepoBindingID()
	if err != nil {
		return spine.RepoBindingInitResult{}, fmt.Errorf("new repo binding id: %w", err)
	}
	binding := spine.RepoBinding{
		ID:                 bindingID,
		OrganizationID:     project.OrganizationID,
		ProjectID:          project.ID,
		CreatedByUserID:    normalized.AuthenticatedUserID,
		Provider:           normalized.Provider,
		RepositoryFullName: normalized.RepositoryFullName,
		RepositoryURL:      normalized.RepositoryURL,
		DefaultBranch:      normalized.ProviderDefaultBranch,
		WorkflowBaseBranch: normalized.WorkflowBaseBranch,
		PathScope:          ".",
		AccessMode:         spine.RepoBindingAccessModeMetadataOnly,
		State:              spine.EntityStateActive,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	event, err := s.initializedEvent(binding, normalized, now)
	if err != nil {
		return spine.RepoBindingInitResult{}, err
	}

	if err := s.TxRunner.RunReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.Store.CreateRepoBinding(txCtx, binding); err != nil {
			return fmt.Errorf("create repo binding: %w", err)
		}
		if err := s.Events.Append(txCtx, event); err != nil {
			return fmt.Errorf("append repo binding initialized event: %w", err)
		}
		return nil
	}); err != nil {
		if isUniqueConstraintError(err) {
			return s.resolveCreateRepoBindingRace(ctx, project, normalized, err)
		}
		return spine.RepoBindingInitResult{}, err
	}

	return initResult(binding, true, initMessage), nil
}

func (s *Service) resolveCreateRepoBindingRace(ctx context.Context, project spine.Project, input InitInput, cause error) (spine.RepoBindingInitResult, error) {
	existing, ok, err := s.Store.GetActiveRepoBindingForProject(ctx, project.ID)
	if err != nil {
		return spine.RepoBindingInitResult{}, fmt.Errorf("get active repo binding for project after create conflict: %w", err)
	}
	if ok {
		if !sameRepository(existing, input) {
			return spine.RepoBindingInitResult{}, ErrDifferentRepoBinding
		}
		return initResult(existing, false, existingMessage), nil
	}

	existing, ok, err = s.Store.GetActiveRepoBindingByOrganizationAndRepository(ctx, project.OrganizationID, input.Provider, input.RepositoryFullName)
	if err != nil {
		return spine.RepoBindingInitResult{}, fmt.Errorf("get active repo binding by organization repository after create conflict: %w", err)
	}
	if ok {
		if existing.ProjectID == project.ID {
			return initResult(existing, false, existingMessage), nil
		}
		return spine.RepoBindingInitResult{}, ErrRepositoryAlreadyBound
	}

	return spine.RepoBindingInitResult{}, cause
}

func normalizeInput(input InitInput) (InitInput, error) {
	input.Provider = strings.ToLower(strings.TrimSpace(input.Provider))
	input.RepositoryFullName = normalizeRepositoryFullName(input.RepositoryFullName)
	input.RepositoryURL = strings.TrimSpace(input.RepositoryURL)
	input.ProviderDefaultBranch = strings.TrimSpace(input.ProviderDefaultBranch)
	input.WorkflowBaseBranch = strings.TrimSpace(input.WorkflowBaseBranch)
	input.LocalRemoteName = strings.TrimSpace(input.LocalRemoteName)
	input.LocalHeadSHA = strings.TrimSpace(input.LocalHeadSHA)

	if strings.TrimSpace(string(input.ProjectID)) == "" {
		return InitInput{}, &ValidationError{Field: "project_id", Message: "is required"}
	}
	if err := validateUUIDv7("project_id", string(input.ProjectID)); err != nil {
		return InitInput{}, err
	}
	if strings.TrimSpace(string(input.AuthenticatedUserID)) == "" {
		return InitInput{}, &ValidationError{Field: "user_id", Message: "authenticated user is required"}
	}
	if input.Provider == "" {
		return InitInput{}, &ValidationError{Field: "provider", Message: "is required"}
	}
	if input.RepositoryFullName == "" {
		return InitInput{}, &ValidationError{Field: "repository_full_name", Message: "is required"}
	}
	if input.RepositoryURL == "" {
		return InitInput{}, &ValidationError{Field: "repository_url", Message: "is required"}
	}
	if input.WorkflowBaseBranch == "" {
		input.WorkflowBaseBranch = input.ProviderDefaultBranch
	}
	if input.WorkflowBaseBranch == "" {
		return InitInput{}, &ValidationError{Field: "workflow_base_branch", Message: "is required when provider_default_branch is empty"}
	}
	if input.ProviderDefaultBranch == "" {
		input.ProviderDefaultBranch = input.WorkflowBaseBranch
	}
	return input, nil
}

func normalizeRepositoryFullName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "/")
	value = strings.TrimSuffix(value, ".git")
	value = strings.Trim(value, "/")
	return value
}

func validateUUIDv7(field string, value string) error {
	id, err := uuid.Parse(value)
	if err != nil {
		return &ValidationError{Field: field, Message: "must be a UUID"}
	}
	if id.Version() != 7 {
		return &ValidationError{Field: field, Message: "must be a UUIDv7"}
	}
	return nil
}

func authorize(project spine.Project, input InitInput) error {
	membership := input.Membership
	if membership.State != spine.EntityStateActive || membership.OrganizationID != project.OrganizationID {
		return ErrForbidden
	}
	switch membership.Role {
	case spine.OrganizationMembershipRoleOwner, spine.OrganizationMembershipRoleAdmin, spine.OrganizationMembershipRoleMember:
		return nil
	default:
		return ErrForbidden
	}
}

func sameRepository(existing spine.RepoBinding, input InitInput) bool {
	return strings.EqualFold(strings.TrimSpace(existing.Provider), input.Provider) &&
		strings.EqualFold(normalizeRepositoryFullName(existing.RepositoryFullName), input.RepositoryFullName)
}

type uniqueConstraintError interface {
	ConstraintName() string
}

func isUniqueConstraintError(err error) bool {
	var constraintErr uniqueConstraintError
	return errors.As(err, &constraintErr)
}

func initResult(binding spine.RepoBinding, created bool, message string) spine.RepoBindingInitResult {
	return spine.RepoBindingInitResult{
		RepoBindingID:         binding.ID,
		ProjectID:             binding.ProjectID,
		OrganizationID:        binding.OrganizationID,
		Provider:              binding.Provider,
		RepositoryFullName:    binding.RepositoryFullName,
		RepositoryURL:         binding.RepositoryURL,
		ProviderDefaultBranch: binding.DefaultBranch,
		WorkflowBaseBranch:    binding.WorkflowBaseBranch,
		State:                 binding.State,
		Created:               created,
		Message:               message,
	}
}

func (s *Service) initializedEvent(binding spine.RepoBinding, input InitInput, timestamp time.Time) (spine.Event, error) {
	eventID, err := s.IDs.NewEventID()
	if err != nil {
		return spine.Event{}, fmt.Errorf("new event id: %w", err)
	}
	payload, err := json.Marshal(map[string]any{
		"repo_binding_id":         binding.ID,
		"project_id":              binding.ProjectID,
		"organization_id":         binding.OrganizationID,
		"provider":                binding.Provider,
		"repository_full_name":    binding.RepositoryFullName,
		"repository_url":          binding.RepositoryURL,
		"provider_default_branch": binding.DefaultBranch,
		"workflow_base_branch":    binding.WorkflowBaseBranch,
		"access_mode":             binding.AccessMode,
		"state":                   binding.State,
		"local_remote_name":       input.LocalRemoteName,
		"local_head_sha":          input.LocalHeadSHA,
	})
	if err != nil {
		return spine.Event{}, fmt.Errorf("marshal repo binding initialized event payload: %w", err)
	}
	return spine.Event{
		ID:             eventID,
		Type:           EventTypeInitialized,
		EntityType:     EntityType,
		EntityID:       string(binding.ID),
		OrganizationID: binding.OrganizationID,
		ProjectID:      binding.ProjectID,
		RepoBindingID:  binding.ID,
		Timestamp:      timestamp,
		Payload:        payload,
	}, nil
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}

type UUIDGenerator struct{}

func (UUIDGenerator) NewRepoBindingID() (spine.RepoBindingID, error) {
	return spine.NewRepoBindingID()
}

func (UUIDGenerator) NewEventID() (spine.EventID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.EventID(id.String()), nil
}
