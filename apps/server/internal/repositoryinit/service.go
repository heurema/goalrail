package repositoryinit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/heurema/goalrail/apps/server/internal/repobinding"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

const (
	EventTypeProjectCreated = "project.created"
	EntityTypeProject       = "Project"
	initMessage             = "Repository context initialized."
	existingMessage         = "Repository context already initialized."
)

var (
	ErrForbidden                 = errors.New("user is not allowed to initialize repository context")
	ErrMembershipRequired        = errors.New("active organization membership is required; self-hosted servers must be bootstrapped with goalrail-server bootstrap owner")
	ErrProjectSlugConflict       = errors.New("project slug is already bound to a different repository")
	ErrProjectSlugUnavailable    = errors.New("project slug is already used by an inactive project")
	ErrProjectForBindingNotFound = errors.New("project for existing repo binding not found")
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
	GetProjectByOrganizationAndSlug(context.Context, spine.OrganizationID, string) (spine.Project, bool, error)
	GetActiveRepoBindingForProject(context.Context, spine.ProjectID) (spine.RepoBinding, bool, error)
	GetActiveRepoBindingByOrganizationAndRepository(context.Context, spine.OrganizationID, string, string) (spine.RepoBinding, bool, error)
	CreateProject(context.Context, spine.Project) error
}

type RepoBindingInitializer interface {
	Init(context.Context, repobinding.InitInput) (spine.RepoBindingInitResult, error)
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
	NewProjectID() (spine.ProjectID, error)
	NewEventID() (spine.EventID, error)
}

type InitInput struct {
	AuthenticatedUserID         spine.UserID
	Membership                  spine.OrganizationMembership
	Provider                    string
	RepositoryFullName          string
	RepositoryURL               string
	ProviderDefaultBranch       string
	WorkflowBaseBranch          string
	LocalRemoteName             string
	LocalHeadSHA                string
	SuggestedProjectSlug        string
	SuggestedProjectDisplayName string
}

type Service struct {
	Store        Store
	RepoBindings RepoBindingInitializer
	Events       EventLog
	TxRunner     TransactionRunner
	Clock        Clock
	IDs          IDGenerator
}

func NewService(store Store, repoBindings RepoBindingInitializer, events EventLog, txRunner TransactionRunner, clock Clock, ids IDGenerator) *Service {
	return &Service{
		Store:        store,
		RepoBindings: repoBindings,
		Events:       events,
		TxRunner:     txRunner,
		Clock:        clock,
		IDs:          ids,
	}
}

func (s *Service) Init(ctx context.Context, input InitInput) (spine.RepositoryContextInitResult, error) {
	normalized, err := normalizeInput(input)
	if err != nil {
		return spine.RepositoryContextInitResult{}, err
	}
	if err := authorize(normalized.Membership); err != nil {
		return spine.RepositoryContextInitResult{}, err
	}

	organizationID := normalized.Membership.OrganizationID
	existingBinding, ok, err := s.Store.GetActiveRepoBindingByOrganizationAndRepository(ctx, organizationID, normalized.Provider, normalized.RepositoryFullName)
	if err != nil {
		return spine.RepositoryContextInitResult{}, fmt.Errorf("get active repo binding by organization repository: %w", err)
	}
	if ok {
		project, ok, err := s.Store.GetProject(ctx, existingBinding.ProjectID)
		if err != nil {
			return spine.RepositoryContextInitResult{}, fmt.Errorf("get project for repo binding: %w", err)
		}
		if !ok || project.State != spine.EntityStateActive {
			return spine.RepositoryContextInitResult{}, ErrProjectForBindingNotFound
		}
		return contextResult(project, existingBinding, false, false, existingMessage), nil
	}

	projectSlug := DeriveProjectSlug(normalized.Provider, normalized.RepositoryFullName)
	projectDisplayName := normalized.RepositoryFullName

	project, projectOK, err := s.Store.GetProjectByOrganizationAndSlug(ctx, organizationID, projectSlug)
	if err != nil {
		return spine.RepositoryContextInitResult{}, fmt.Errorf("get project by organization slug: %w", err)
	}
	if projectOK {
		if project.State != spine.EntityStateActive {
			return spine.RepositoryContextInitResult{}, ErrProjectSlugUnavailable
		} else if err := s.ensureProjectReusable(ctx, project, normalized); err != nil {
			return spine.RepositoryContextInitResult{}, err
		}
	}

	projectCreated := false
	if !projectOK {
		projectCreated = true
		now := s.Clock.Now().UTC()
		projectID, err := s.IDs.NewProjectID()
		if err != nil {
			return spine.RepositoryContextInitResult{}, fmt.Errorf("new project id: %w", err)
		}
		project = spine.Project{
			ID:              projectID,
			OrganizationID:  organizationID,
			CreatedByUserID: normalized.AuthenticatedUserID,
			Slug:            projectSlug,
			DisplayName:     projectDisplayName,
			State:           spine.EntityStateActive,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
	}

	repoInput := repobinding.InitInput{
		ProjectID:             project.ID,
		AuthenticatedUserID:   normalized.AuthenticatedUserID,
		Membership:            normalized.Membership,
		Provider:              normalized.Provider,
		RepositoryFullName:    normalized.RepositoryFullName,
		RepositoryURL:         normalized.RepositoryURL,
		ProviderDefaultBranch: normalized.ProviderDefaultBranch,
		WorkflowBaseBranch:    normalized.WorkflowBaseBranch,
		LocalRemoteName:       normalized.LocalRemoteName,
		LocalHeadSHA:          normalized.LocalHeadSHA,
	}

	var bindingResult spine.RepoBindingInitResult
	if projectCreated {
		event, err := s.projectCreatedEvent(project, normalized)
		if err != nil {
			return spine.RepositoryContextInitResult{}, err
		}
		if err := s.TxRunner.RunReadCommitted(ctx, func(txCtx context.Context) error {
			if err := s.Store.CreateProject(txCtx, project); err != nil {
				return fmt.Errorf("create project: %w", err)
			}
			if err := s.Events.Append(txCtx, event); err != nil {
				return fmt.Errorf("append project created event: %w", err)
			}
			result, err := s.RepoBindings.Init(txCtx, repoInput)
			if err != nil {
				return err
			}
			bindingResult = result
			return nil
		}); err != nil {
			return spine.RepositoryContextInitResult{}, err
		}
	} else {
		bindingResult, err = s.RepoBindings.Init(ctx, repoInput)
		if err != nil {
			return spine.RepositoryContextInitResult{}, err
		}
	}

	binding := spine.RepoBinding{
		ID:                 bindingResult.RepoBindingID,
		OrganizationID:     bindingResult.OrganizationID,
		ProjectID:          bindingResult.ProjectID,
		Provider:           bindingResult.Provider,
		RepositoryFullName: bindingResult.RepositoryFullName,
		RepositoryURL:      bindingResult.RepositoryURL,
		DefaultBranch:      bindingResult.ProviderDefaultBranch,
		WorkflowBaseBranch: bindingResult.WorkflowBaseBranch,
		State:              bindingResult.State,
	}
	return contextResult(project, binding, projectCreated, bindingResult.Created, initMessage), nil
}

func (s *Service) ensureProjectReusable(ctx context.Context, project spine.Project, input InitInput) error {
	existing, ok, err := s.Store.GetActiveRepoBindingForProject(ctx, project.ID)
	if err != nil {
		return fmt.Errorf("get active repo binding for project slug: %w", err)
	}
	if !ok || sameRepository(existing, input.Provider, input.RepositoryFullName) {
		return nil
	}
	return ErrProjectSlugConflict
}

func normalizeInput(input InitInput) (InitInput, error) {
	input.Provider = strings.ToLower(strings.TrimSpace(input.Provider))
	input.RepositoryFullName = NormalizeRepositoryFullName(input.RepositoryFullName)
	input.RepositoryURL = strings.TrimSpace(input.RepositoryURL)
	input.ProviderDefaultBranch = strings.TrimSpace(input.ProviderDefaultBranch)
	input.WorkflowBaseBranch = strings.TrimSpace(input.WorkflowBaseBranch)
	input.LocalRemoteName = strings.TrimSpace(input.LocalRemoteName)
	input.LocalHeadSHA = strings.TrimSpace(input.LocalHeadSHA)
	input.SuggestedProjectSlug = strings.TrimSpace(input.SuggestedProjectSlug)
	input.SuggestedProjectDisplayName = strings.TrimSpace(input.SuggestedProjectDisplayName)

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
	if DeriveProjectSlug(input.Provider, input.RepositoryFullName) == "" {
		return InitInput{}, &ValidationError{Field: "repository_full_name", Message: "cannot derive project slug"}
	}
	return input, nil
}

func authorize(membership spine.OrganizationMembership) error {
	if membership.State != spine.EntityStateActive || strings.TrimSpace(string(membership.OrganizationID)) == "" {
		return ErrMembershipRequired
	}
	switch membership.Role {
	case spine.OrganizationMembershipRoleOwner, spine.OrganizationMembershipRoleAdmin, spine.OrganizationMembershipRoleMember:
		return nil
	default:
		return ErrForbidden
	}
}

func NormalizeRepositoryFullName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "/")
	value = strings.TrimSuffix(value, ".git")
	value = strings.Trim(value, "/")
	return value
}

func DeriveProjectSlug(provider string, repositoryFullName string) string {
	source := strings.ToLower(strings.TrimSpace(provider) + "-" + NormalizeRepositoryFullName(repositoryFullName))
	var b strings.Builder
	lastDash := false
	for _, r := range source {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func sameRepository(binding spine.RepoBinding, provider string, repositoryFullName string) bool {
	return strings.EqualFold(strings.TrimSpace(binding.Provider), strings.TrimSpace(provider)) &&
		strings.EqualFold(NormalizeRepositoryFullName(binding.RepositoryFullName), NormalizeRepositoryFullName(repositoryFullName))
}

func contextResult(project spine.Project, binding spine.RepoBinding, projectCreated bool, repoBindingCreated bool, message string) spine.RepositoryContextInitResult {
	return spine.RepositoryContextInitResult{
		OrganizationID:        project.OrganizationID,
		ProjectID:             project.ID,
		ProjectSlug:           project.Slug,
		ProjectDisplayName:    project.DisplayName,
		ProjectCreated:        projectCreated,
		RepoBindingID:         binding.ID,
		RepoBindingCreated:    repoBindingCreated,
		Provider:              binding.Provider,
		RepositoryFullName:    binding.RepositoryFullName,
		RepositoryURL:         binding.RepositoryURL,
		ProviderDefaultBranch: binding.DefaultBranch,
		WorkflowBaseBranch:    binding.WorkflowBaseBranch,
		State:                 binding.State,
		Message:               message,
	}
}

func (s *Service) projectCreatedEvent(project spine.Project, input InitInput) (spine.Event, error) {
	eventID, err := s.IDs.NewEventID()
	if err != nil {
		return spine.Event{}, fmt.Errorf("new event id: %w", err)
	}
	payload, err := json.Marshal(map[string]any{
		"project_id":             project.ID,
		"organization_id":        project.OrganizationID,
		"project_slug":           project.Slug,
		"project_display_name":   project.DisplayName,
		"provider":               input.Provider,
		"repository_full_name":   input.RepositoryFullName,
		"repository_url":         input.RepositoryURL,
		"workflow_base_branch":   input.WorkflowBaseBranch,
		"suggested_project_slug": input.SuggestedProjectSlug,
		"suggested_display_name": input.SuggestedProjectDisplayName,
	})
	if err != nil {
		return spine.Event{}, fmt.Errorf("marshal project created event payload: %w", err)
	}
	return spine.Event{
		ID:             eventID,
		Type:           EventTypeProjectCreated,
		EntityType:     EntityTypeProject,
		EntityID:       string(project.ID),
		OrganizationID: project.OrganizationID,
		ProjectID:      project.ID,
		Timestamp:      project.CreatedAt,
		Payload:        payload,
	}, nil
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}

type UUIDGenerator struct{}

func (UUIDGenerator) NewProjectID() (spine.ProjectID, error) {
	return spine.NewProjectID()
}

func (UUIDGenerator) NewEventID() (spine.EventID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.EventID(id.String()), nil
}
