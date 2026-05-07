package repositorycontext

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

const (
	EventTypeSnapshotRecorded = "repository_context_snapshot.recorded"
	EntityTypeSnapshot        = "RepositoryContextSnapshot"
	recordedMessage           = "Repository context snapshot recorded."
	existingMessage           = "Repository context snapshot already recorded."
	supportedSchemaVersion    = 1
)

var (
	ErrForbidden            = errors.New("user is not allowed to record repository context snapshot")
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrRepoBindingNotFound  = errors.New("repo binding not found")
	ErrRepoBindingInactive  = errors.New("repo binding is not active")
	ErrSnapshotMismatch     = errors.New("repository context snapshot does not match repo binding")
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
	GetRepoBinding(context.Context, spine.RepoBindingID) (spine.RepoBinding, bool, error)
	ListActiveProjectRepoBindingContexts(context.Context, spine.OrganizationID) ([]spine.ProjectRepoBindingContext, error)
	GetRepositoryContextSnapshotByFingerprint(context.Context, spine.RepoBindingID, string) (spine.RepositoryContextSnapshotRecord, bool, error)
	CreateRepositoryContextSnapshot(context.Context, spine.RepositoryContextSnapshotRecord) error
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
	NewRepositoryContextSnapshotID() (spine.RepositoryContextSnapshotID, error)
	NewEventID() (spine.EventID, error)
}

type RecordInput struct {
	AuthenticatedUserID spine.UserID
	Membership          spine.OrganizationMembership
	RepoBindingID       spine.RepoBindingID
	Snapshot            spine.RepositoryContextSnapshotRequest
}

type ReadOrganizationContextInput struct {
	AuthenticatedUserID spine.UserID
	Membership          spine.OrganizationMembership
	OrganizationID      spine.OrganizationID
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

func (s *Service) RecordSnapshot(ctx context.Context, input RecordInput) (spine.RepositoryContextSnapshotResult, error) {
	normalized, err := normalizeInput(input)
	if err != nil {
		return spine.RepositoryContextSnapshotResult{}, err
	}
	if err := authorize(normalized.Membership); err != nil {
		return spine.RepositoryContextSnapshotResult{}, err
	}

	binding, ok, err := s.Store.GetRepoBinding(ctx, normalized.RepoBindingID)
	if err != nil {
		return spine.RepositoryContextSnapshotResult{}, fmt.Errorf("get repo binding: %w", err)
	}
	if !ok {
		return spine.RepositoryContextSnapshotResult{}, ErrRepoBindingNotFound
	}
	if binding.State != spine.EntityStateActive {
		return spine.RepositoryContextSnapshotResult{}, ErrRepoBindingInactive
	}
	if binding.OrganizationID != normalized.Membership.OrganizationID {
		return spine.RepositoryContextSnapshotResult{}, ErrForbidden
	}
	if !snapshotMatchesBinding(normalized.Snapshot, binding) {
		return spine.RepositoryContextSnapshotResult{}, ErrSnapshotMismatch
	}

	snapshotJSON, err := json.Marshal(normalized.Snapshot)
	if err != nil {
		return spine.RepositoryContextSnapshotResult{}, fmt.Errorf("marshal repository context snapshot: %w", err)
	}
	fingerprint := fingerprintSnapshot(snapshotJSON)

	existing, ok, err := s.Store.GetRepositoryContextSnapshotByFingerprint(ctx, binding.ID, fingerprint)
	if err != nil {
		return spine.RepositoryContextSnapshotResult{}, fmt.Errorf("get repository context snapshot by fingerprint: %w", err)
	}
	if ok {
		return snapshotResult(existing, false, existingMessage), nil
	}

	now := s.Clock.Now().UTC()
	snapshotID, err := s.IDs.NewRepositoryContextSnapshotID()
	if err != nil {
		return spine.RepositoryContextSnapshotResult{}, fmt.Errorf("new repository context snapshot id: %w", err)
	}
	record := spine.RepositoryContextSnapshotRecord{
		ID:             snapshotID,
		OrganizationID: binding.OrganizationID,
		ProjectID:      binding.ProjectID,
		RepoBindingID:  binding.ID,
		Source:         normalized.Snapshot.Source,
		SchemaVersion:  normalized.Snapshot.SchemaVersion,
		Fingerprint:    fingerprint,
		Snapshot:       snapshotJSON,
		CreatedAt:      now,
	}
	event, err := s.snapshotRecordedEvent(record)
	if err != nil {
		return spine.RepositoryContextSnapshotResult{}, err
	}

	if err := s.TxRunner.RunReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.Store.CreateRepositoryContextSnapshot(txCtx, record); err != nil {
			return fmt.Errorf("create repository context snapshot: %w", err)
		}
		if err := s.Events.Append(txCtx, event); err != nil {
			return fmt.Errorf("append repository context snapshot event: %w", err)
		}
		return nil
	}); err != nil {
		if isUniqueConstraintError(err) {
			existing, ok, getErr := s.Store.GetRepositoryContextSnapshotByFingerprint(ctx, binding.ID, fingerprint)
			if getErr != nil {
				return spine.RepositoryContextSnapshotResult{}, fmt.Errorf("get repository context snapshot after create conflict: %w", getErr)
			}
			if ok {
				return snapshotResult(existing, false, existingMessage), nil
			}
		}
		return spine.RepositoryContextSnapshotResult{}, err
	}

	return snapshotResult(record, true, recordedMessage), nil
}

func (s *Service) GetOrganizationRepositoryContext(ctx context.Context, input ReadOrganizationContextInput) (spine.OrganizationRepositoryContextResult, error) {
	normalized, err := normalizeReadInput(input)
	if err != nil {
		return spine.OrganizationRepositoryContextResult{}, err
	}
	if err := authorizeRead(normalized.Membership, normalized.OrganizationID); err != nil {
		return spine.OrganizationRepositoryContextResult{}, err
	}

	organization, ok, err := s.Store.GetOrganization(ctx, normalized.OrganizationID)
	if err != nil {
		return spine.OrganizationRepositoryContextResult{}, fmt.Errorf("get organization: %w", err)
	}
	if !ok || organization.State != spine.EntityStateActive {
		return spine.OrganizationRepositoryContextResult{}, ErrOrganizationNotFound
	}

	contexts, err := s.Store.ListActiveProjectRepoBindingContexts(ctx, normalized.OrganizationID)
	if err != nil {
		return spine.OrganizationRepositoryContextResult{}, fmt.Errorf("list active project repo binding contexts: %w", err)
	}
	if contexts == nil {
		contexts = []spine.ProjectRepoBindingContext{}
	}
	return spine.OrganizationRepositoryContextResult{
		Organization: publicOrganization(organization),
		Contexts:     publicContexts(contexts),
	}, nil
}

func publicOrganization(organization spine.Organization) spine.OrganizationRepositoryContextOrganization {
	return spine.OrganizationRepositoryContextOrganization{
		ID:          organization.ID,
		Slug:        organization.Slug,
		DisplayName: organization.DisplayName,
		State:       organization.State,
	}
}

func publicContexts(contexts []spine.ProjectRepoBindingContext) []spine.OrganizationRepositoryContext {
	public := make([]spine.OrganizationRepositoryContext, 0, len(contexts))
	for _, context := range contexts {
		public = append(public, spine.OrganizationRepositoryContext{
			Project: spine.OrganizationRepositoryContextProject{
				ID:          context.Project.ID,
				Slug:        context.Project.Slug,
				DisplayName: context.Project.DisplayName,
				State:       context.Project.State,
				CreatedAt:   context.Project.CreatedAt,
				UpdatedAt:   context.Project.UpdatedAt,
			},
			RepoBinding: spine.OrganizationRepositoryContextRepoBinding{
				ID:                 context.RepoBinding.ID,
				Provider:           context.RepoBinding.Provider,
				RepositoryFullName: context.RepoBinding.RepositoryFullName,
				RepositoryURL:      sanitizeRepositoryURL(context.RepoBinding.RepositoryURL),
				DefaultBranch:      context.RepoBinding.DefaultBranch,
				WorkflowBaseBranch: context.RepoBinding.WorkflowBaseBranch,
				PathScope:          context.RepoBinding.PathScope,
				AccessMode:         context.RepoBinding.AccessMode,
				State:              context.RepoBinding.State,
				CreatedAt:          context.RepoBinding.CreatedAt,
				UpdatedAt:          context.RepoBinding.UpdatedAt,
			},
		})
	}
	return public
}

func sanitizeRepositoryURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User == nil {
		return raw
	}
	parsed.User = nil
	return parsed.String()
}

func normalizeReadInput(input ReadOrganizationContextInput) (ReadOrganizationContextInput, error) {
	if strings.TrimSpace(string(input.AuthenticatedUserID)) == "" {
		return ReadOrganizationContextInput{}, &ValidationError{Field: "user_id", Message: "authenticated user is required"}
	}
	if strings.TrimSpace(string(input.OrganizationID)) == "" {
		return ReadOrganizationContextInput{}, &ValidationError{Field: "organization_id", Message: "is required"}
	}
	if err := validateUUIDv7("organization_id", string(input.OrganizationID)); err != nil {
		return ReadOrganizationContextInput{}, err
	}
	return input, nil
}

func normalizeInput(input RecordInput) (RecordInput, error) {
	if strings.TrimSpace(string(input.AuthenticatedUserID)) == "" {
		return RecordInput{}, &ValidationError{Field: "user_id", Message: "authenticated user is required"}
	}
	if strings.TrimSpace(string(input.RepoBindingID)) == "" {
		return RecordInput{}, &ValidationError{Field: "repo_binding_id", Message: "is required"}
	}
	if err := validateUUIDv7("repo_binding_id", string(input.RepoBindingID)); err != nil {
		return RecordInput{}, err
	}
	snapshot := input.Snapshot
	snapshot.Source = strings.TrimSpace(snapshot.Source)
	if snapshot.Source == "" {
		return RecordInput{}, &ValidationError{Field: "source", Message: "is required"}
	}
	if snapshot.SchemaVersion != supportedSchemaVersion {
		return RecordInput{}, &ValidationError{Field: "schema_version", Message: "must be 1"}
	}
	snapshot.Repository.Provider = strings.ToLower(strings.TrimSpace(snapshot.Repository.Provider))
	snapshot.Repository.FullName = normalizeRepositoryFullName(snapshot.Repository.FullName)
	snapshot.Repository.URL = strings.TrimSpace(snapshot.Repository.URL)
	snapshot.Repository.ProviderDefaultBranch = strings.TrimSpace(snapshot.Repository.ProviderDefaultBranch)
	snapshot.Repository.WorkflowBaseBranch = strings.TrimSpace(snapshot.Repository.WorkflowBaseBranch)
	snapshot.Repository.RemoteName = strings.TrimSpace(snapshot.Repository.RemoteName)
	snapshot.Repository.HeadSHA = strings.TrimSpace(snapshot.Repository.HeadSHA)
	if snapshot.Repository.Provider == "" {
		return RecordInput{}, &ValidationError{Field: "repository.provider", Message: "is required"}
	}
	if snapshot.Repository.FullName == "" {
		return RecordInput{}, &ValidationError{Field: "repository.full_name", Message: "is required"}
	}
	if snapshot.Repository.URL == "" {
		return RecordInput{}, &ValidationError{Field: "repository.url", Message: "is required"}
	}
	if snapshot.Repository.WorkflowBaseBranch == "" {
		return RecordInput{}, &ValidationError{Field: "repository.workflow_base_branch", Message: "is required"}
	}
	snapshot.DetectedPaths = normalizeStringList(snapshot.DetectedPaths)
	snapshot.DetectedToolchains = normalizeStringList(snapshot.DetectedToolchains)
	snapshot.DetectedPackageManagers = normalizeStringList(snapshot.DetectedPackageManagers)
	snapshot.WorkspaceCandidates = normalizeStringList(snapshot.WorkspaceCandidates)
	input.Snapshot = snapshot
	return input, nil
}

func authorize(membership spine.OrganizationMembership) error {
	if membership.State != spine.EntityStateActive || strings.TrimSpace(string(membership.OrganizationID)) == "" {
		return ErrForbidden
	}
	switch membership.Role {
	case spine.OrganizationMembershipRoleOwner, spine.OrganizationMembershipRoleAdmin, spine.OrganizationMembershipRoleMember:
		return nil
	default:
		return ErrForbidden
	}
}

func authorizeRead(membership spine.OrganizationMembership, organizationID spine.OrganizationID) error {
	if membership.State != spine.EntityStateActive || !sameUUID(membership.OrganizationID, organizationID) {
		return ErrForbidden
	}
	switch membership.Role {
	case spine.OrganizationMembershipRoleOwner, spine.OrganizationMembershipRoleAdmin, spine.OrganizationMembershipRoleMember, spine.OrganizationMembershipRoleViewer:
		return nil
	default:
		return ErrForbidden
	}
}

func sameUUID(left spine.OrganizationID, right spine.OrganizationID) bool {
	leftID, leftErr := uuid.Parse(string(left))
	rightID, rightErr := uuid.Parse(string(right))
	return leftErr == nil && rightErr == nil && leftID == rightID
}

func snapshotMatchesBinding(snapshot spine.RepositoryContextSnapshotRequest, binding spine.RepoBinding) bool {
	return strings.EqualFold(snapshot.Repository.Provider, binding.Provider) &&
		strings.EqualFold(normalizeRepositoryFullName(snapshot.Repository.FullName), normalizeRepositoryFullName(binding.RepositoryFullName)) &&
		snapshot.Repository.URL == strings.TrimSpace(binding.RepositoryURL) &&
		snapshot.Repository.WorkflowBaseBranch == strings.TrimSpace(binding.WorkflowBaseBranch)
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

func normalizeStringList(values []string) []string {
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		seen[value] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func fingerprintSnapshot(snapshotJSON []byte) string {
	sum := sha256.Sum256(snapshotJSON)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func (s *Service) snapshotRecordedEvent(record spine.RepositoryContextSnapshotRecord) (spine.Event, error) {
	eventID, err := s.IDs.NewEventID()
	if err != nil {
		return spine.Event{}, fmt.Errorf("new event id: %w", err)
	}
	payload, err := json.Marshal(map[string]any{
		"source":         record.Source,
		"schema_version": record.SchemaVersion,
		"fingerprint":    record.Fingerprint,
	})
	if err != nil {
		return spine.Event{}, fmt.Errorf("marshal repository context snapshot event payload: %w", err)
	}
	return spine.Event{
		ID:             eventID,
		Type:           EventTypeSnapshotRecorded,
		EntityType:     EntityTypeSnapshot,
		EntityID:       string(record.ID),
		OrganizationID: record.OrganizationID,
		ProjectID:      record.ProjectID,
		RepoBindingID:  record.RepoBindingID,
		Timestamp:      record.CreatedAt,
		Payload:        payload,
	}, nil
}

func snapshotResult(record spine.RepositoryContextSnapshotRecord, created bool, message string) spine.RepositoryContextSnapshotResult {
	return spine.RepositoryContextSnapshotResult{
		ContextSnapshotID: record.ID,
		OrganizationID:    record.OrganizationID,
		ProjectID:         record.ProjectID,
		RepoBindingID:     record.RepoBindingID,
		Source:            record.Source,
		SchemaVersion:     record.SchemaVersion,
		Fingerprint:       record.Fingerprint,
		Created:           created,
		Message:           message,
	}
}

type uniqueConstraintError interface {
	ConstraintName() string
}

func isUniqueConstraintError(err error) bool {
	var uniqueErr uniqueConstraintError
	return errors.As(err, &uniqueErr)
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}

type UUIDGenerator struct{}

func (UUIDGenerator) NewRepositoryContextSnapshotID() (spine.RepositoryContextSnapshotID, error) {
	return spine.NewRepositoryContextSnapshotID()
}

func (UUIDGenerator) NewEventID() (spine.EventID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.EventID(id.String()), nil
}
