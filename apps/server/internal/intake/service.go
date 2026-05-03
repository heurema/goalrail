package intake

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
	EventTypeReceived = "intake.received"
	EntityTypeIntake  = "IntakeRecord"
)

var ErrNotFound = errors.New("intake record not found")

var ErrProjectContextUnavailable = errors.New("project context resolver unavailable")

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
	Create(context.Context, spine.IntakeRecord) error
	Get(context.Context, spine.IntakeID) (spine.IntakeRecord, bool, error)
}

type ProjectContextResolver interface {
	ResolveRepoBinding(context.Context, spine.RepoBindingID) (spine.ResolvedRepoBindingContext, bool, error)
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
	NewIntakeID() (spine.IntakeID, error)
	NewEventID() (spine.EventID, error)
}

type Service struct {
	Store          Store
	ProjectContext ProjectContextResolver
	Events         EventLog
	TxRunner       TransactionRunner
	Clock          Clock
	IDs            IDGenerator
}

type Option func(*Service)

func WithTransactionRunner(runner TransactionRunner) Option {
	return func(s *Service) {
		s.TxRunner = runner
	}
}

func NewService(store Store, projectContext ProjectContextResolver, events EventLog, clock Clock, ids IDGenerator, opts ...Option) *Service {
	service := &Service{
		Store:          store,
		ProjectContext: projectContext,
		Events:         events,
		Clock:          clock,
		IDs:            ids,
	}
	for _, opt := range opts {
		opt(service)
	}
	return service
}

func (s *Service) Submit(ctx context.Context, submission spine.IntakeSubmission) (spine.IntakeRecord, error) {
	if err := ValidateSubmission(submission); err != nil {
		return spine.IntakeRecord{}, err
	}
	if err := s.validateDependencies(); err != nil {
		return spine.IntakeRecord{}, err
	}
	resolved, err := s.resolveProjectContext(ctx, submission)
	if err != nil {
		return spine.IntakeRecord{}, err
	}

	now := s.Clock.Now().UTC()
	intakeID, err := s.IDs.NewIntakeID()
	if err != nil {
		return spine.IntakeRecord{}, fmt.Errorf("new intake id: %w", err)
	}

	intentOwner := submission.RequestAuthor
	if submission.IntentOwner != nil {
		intentOwner = *submission.IntentOwner
	}

	record := spine.IntakeRecord{
		ID:                       intakeID,
		OrganizationID:           resolved.OrganizationID,
		ProjectID:                resolved.ProjectID,
		RepoBindingID:            submission.RepoBindingID,
		Source:                   submission.Source,
		Title:                    submission.Title,
		Body:                     submission.Body,
		RequestAuthor:            submission.RequestAuthor,
		IntentOwner:              intentOwner,
		State:                    spine.IntakeStateReceived,
		CanonicalContractCreated: false,
		CreatedAt:                now,
	}

	event, err := s.receivedEvent(record, now)
	if err != nil {
		return spine.IntakeRecord{}, err
	}
	if s.TxRunner != nil {
		if err := s.TxRunner.RunReadCommitted(ctx, func(txCtx context.Context) error {
			if err := s.Store.Create(txCtx, record); err != nil {
				return err
			}
			if err := s.Events.Append(txCtx, event); err != nil {
				return err
			}
			return nil
		}); err != nil {
			return spine.IntakeRecord{}, fmt.Errorf("create intake record with event: %w", err)
		}
		return record, nil
	}

	if err := s.Store.Create(ctx, record); err != nil {
		return spine.IntakeRecord{}, fmt.Errorf("create intake record: %w", err)
	}
	if err := s.Events.Append(ctx, event); err != nil {
		return spine.IntakeRecord{}, fmt.Errorf("append intake event: %w", err)
	}

	return record, nil
}

func (s *Service) Get(ctx context.Context, id spine.IntakeID) (spine.IntakeRecord, error) {
	if err := s.validateDependencies(); err != nil {
		return spine.IntakeRecord{}, err
	}

	record, ok, err := s.Store.Get(ctx, id)
	if err != nil {
		return spine.IntakeRecord{}, fmt.Errorf("get intake record: %w", err)
	}
	if !ok {
		return spine.IntakeRecord{}, ErrNotFound
	}
	return record, nil
}

func (s *Service) resolveProjectContext(ctx context.Context, submission spine.IntakeSubmission) (spine.ResolvedRepoBindingContext, error) {
	if s.ProjectContext == nil {
		return spine.ResolvedRepoBindingContext{}, ErrProjectContextUnavailable
	}

	resolved, ok, err := s.ProjectContext.ResolveRepoBinding(ctx, submission.RepoBindingID)
	if err != nil {
		return spine.ResolvedRepoBindingContext{}, fmt.Errorf("resolve repo binding context: %w", err)
	}
	if !ok {
		return spine.ResolvedRepoBindingContext{}, &ValidationError{Field: "repo_binding_id", Message: "does not exist"}
	}
	if resolved.ProjectID != submission.ProjectID {
		return spine.ResolvedRepoBindingContext{}, &ValidationError{Field: "repo_binding_id", Message: "does not belong to project_id"}
	}
	if strings.TrimSpace(string(resolved.OrganizationID)) == "" {
		return spine.ResolvedRepoBindingContext{}, &ValidationError{Field: "organization_id", Message: "resolved context is required"}
	}
	if strings.TrimSpace(string(resolved.RepoBindingID)) == "" {
		return spine.ResolvedRepoBindingContext{}, &ValidationError{Field: "repo_binding_id", Message: "resolved context is required"}
	}
	if resolved.RepoBindingID != submission.RepoBindingID {
		return spine.ResolvedRepoBindingContext{}, &ValidationError{Field: "repo_binding_id", Message: "resolved context does not match request"}
	}
	return resolved, nil
}

func (s *Service) receivedEvent(record spine.IntakeRecord, timestamp time.Time) (spine.Event, error) {
	eventID, err := s.IDs.NewEventID()
	if err != nil {
		return spine.Event{}, fmt.Errorf("new event id: %w", err)
	}

	payload, err := json.Marshal(record)
	if err != nil {
		return spine.Event{}, fmt.Errorf("marshal intake event payload: %w", err)
	}

	return spine.Event{
		ID:             eventID,
		Type:           EventTypeReceived,
		EntityType:     EntityTypeIntake,
		EntityID:       string(record.ID),
		OrganizationID: record.OrganizationID,
		ProjectID:      record.ProjectID,
		RepoBindingID:  record.RepoBindingID,
		Timestamp:      timestamp,
		Payload:        payload,
	}, nil
}

func (s *Service) validateDependencies() error {
	if s.Store == nil {
		return errors.New("intake service store is nil")
	}
	if s.Events == nil {
		return errors.New("intake service event log is nil")
	}
	if s.Clock == nil {
		return errors.New("intake service clock is nil")
	}
	if s.IDs == nil {
		return errors.New("intake service id generator is nil")
	}
	return nil
}

func ValidateSubmission(submission spine.IntakeSubmission) error {
	if strings.TrimSpace(string(submission.ProjectID)) == "" {
		return &ValidationError{Field: "project_id", Message: "is required"}
	}
	if err := validateUUIDv7("project_id", string(submission.ProjectID)); err != nil {
		return err
	}
	if strings.TrimSpace(string(submission.RepoBindingID)) == "" {
		return &ValidationError{Field: "repo_binding_id", Message: "is required"}
	}
	if err := validateUUIDv7("repo_binding_id", string(submission.RepoBindingID)); err != nil {
		return err
	}
	if strings.TrimSpace(submission.Source.Kind) == "" {
		return &ValidationError{Field: "source.kind", Message: "is required"}
	}
	if strings.TrimSpace(submission.Title) == "" && strings.TrimSpace(submission.Body) == "" {
		return &ValidationError{Field: "title", Message: "title or body is required"}
	}
	if strings.TrimSpace(submission.RequestAuthor.Kind) == "" {
		return &ValidationError{Field: "request_author.kind", Message: "is required"}
	}
	if strings.TrimSpace(submission.RequestAuthor.ID) == "" {
		return &ValidationError{Field: "request_author.id", Message: "is required"}
	}
	return nil
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

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}

type UUIDGenerator struct{}

func (UUIDGenerator) NewIntakeID() (spine.IntakeID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.IntakeID(id.String()), nil
}

func (UUIDGenerator) NewEventID() (spine.EventID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.EventID(id.String()), nil
}
