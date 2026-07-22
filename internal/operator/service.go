// Package operator provides the smallest local canary lifecycle surface. It
// composes canonical domain rules and the append-only evidence store; it is not
// a workflow engine or dashboard.
package operator

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/heurema/goalrail/internal/adapters/codex"
	"github.com/heurema/goalrail/internal/adapters/langfuse"
	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/evidence"
)

var (
	ErrRealCanaryNotActivated = evidence.ErrRealCanaryNotActivated
	ErrCanaryStopped          = errors.New("canary assignments are stopped")
	ErrCanaryAlreadyStopped   = errors.New("canary assignments are already stopped")
	ErrChangeAlreadyStarted   = errors.New("canary change already has an admission or assignment")
	ErrChangeNotStarted       = errors.New("canary change is not started")
	ErrChangeAlreadyTerminal  = errors.New("canary change already has a terminal state")
	ErrCheckSetState          = errors.New("check-set action requires an assigned non-terminal change with the expected frozen set")
	ErrAssessmentState        = errors.New("owner assessment requires one delivered unassessed change")
	ErrCorrectionState        = errors.New("assessment correction requires an existing latest assessment")
)

type Service struct {
	store    *evidence.Store
	repoRoot string
	manifest domain.CanaryManifest
}

func NewService(store *evidence.Store, repoRoot string) (*Service, error) {
	return NewServiceForManifest(store, repoRoot, domain.IntentCanaryV0ManifestVersion)
}

// NewServiceForManifest binds every operator action and projection to one
// immutable manifest version and its matching evidence chain.
func NewServiceForManifest(store *evidence.Store, repoRoot string, manifestVersion uint32) (*Service, error) {
	if store == nil {
		return nil, errors.New("operator service requires an evidence store")
	}
	if store.ManifestVersion() != manifestVersion {
		return nil, errors.New("operator manifest version must match the evidence store")
	}
	if _, err := codex.NewRunContext("validation-change", "validation-run", repoRoot); err != nil {
		return nil, fmt.Errorf("validate operator repository root: %w", err)
	}
	manifest, err := domain.NewIntentCanaryV0ManifestForVersion(manifestVersion)
	if err != nil {
		return nil, fmt.Errorf("load frozen manifest: %w", err)
	}
	return &Service{store: store, repoRoot: repoRoot, manifest: manifest}, nil
}

type StartInput struct {
	EventID       domain.EvidenceEventID
	ChangeID      domain.ChangeID
	RunID         domain.RunID
	IntentVersion uint32
	OccurredAt    time.Time
	Actor         domain.ActorID
	SourceRef     domain.EvidenceReference
	Reason        domain.EvidenceReasonCode
	Synthetic     bool
}

type StartReceipt struct {
	CanaryID        domain.CanaryID          `json:"canary_id"`
	ManifestVersion uint32                   `json:"manifest_version"`
	ChangeID        domain.ChangeID          `json:"change_id"`
	RunID           domain.RunID             `json:"run_id"`
	Ordinal         uint32                   `json:"ordinal"`
	Variant         domain.CanaryVariant     `json:"variant"`
	Decision        domain.AdmissionDecision `json:"decision"`
	RunContextEnv   string                   `json:"run_context_env"`
	EvidenceEventID domain.EvidenceEventID   `json:"evidence_event_id"`
}

func (s *Service) Start(input StartInput) (StartReceipt, error) {
	manifest := s.manifest
	events, err := s.store.ReadAll()
	if err != nil {
		return StartReceipt{}, err
	}
	for _, event := range events {
		if event.CanaryID == manifest.ID && event.Kind == domain.EventCanaryStopped {
			return StartReceipt{}, ErrCanaryStopped
		}
		if event.CanaryID == manifest.ID && event.ChangeID == input.ChangeID &&
			(event.Admission != nil || event.Assignment != nil) {
			return StartReceipt{}, ErrChangeAlreadyStarted
		}
	}
	if !input.Synthetic {
		return StartReceipt{}, ErrRealCanaryNotActivated
	}
	ordinal := uint32(1)
	for _, event := range events {
		if event.CanaryID != manifest.ID || event.Assignment == nil {
			continue
		}
		ordinal++
	}
	variant, err := domain.CanaryVariantForOrdinal(ordinal)
	if err != nil {
		return StartReceipt{}, err
	}
	runContext, err := codex.NewRunContext(input.ChangeID, input.RunID, s.repoRoot)
	if err != nil {
		return StartReceipt{}, err
	}
	encodedContext, err := runContext.EnvironmentValue()
	if err != nil {
		return StartReceipt{}, err
	}
	event := domain.EvidenceEvent{
		ID:              input.EventID,
		CanaryID:        manifest.ID,
		ManifestVersion: s.eventManifestVersion(),
		ChangeID:        input.ChangeID,
		Kind:            domain.EventAdmissionDecided,
		OccurredAt:      input.OccurredAt,
		Actor:           input.Actor,
		SourceRef:       input.SourceRef,
		ReasonCode:      input.Reason,
		Admission: &domain.CanaryAdmission{
			Decision:  domain.AdmissionEligible,
			Synthetic: true,
		},
		Assignment: &domain.CanaryAssignment{
			Ordinal:         ordinal,
			Variant:         variant,
			ManifestVersion: manifest.Version,
			IntentVersion:   input.IntentVersion,
			RunID:           input.RunID,
			Synthetic:       true,
		},
	}
	if err := s.store.Append(event); err != nil {
		return StartReceipt{}, err
	}
	return StartReceipt{
		CanaryID:        manifest.ID,
		ManifestVersion: manifest.Version,
		ChangeID:        input.ChangeID,
		RunID:           input.RunID,
		Ordinal:         ordinal,
		Variant:         variant,
		Decision:        domain.AdmissionEligible,
		RunContextEnv:   encodedContext,
		EvidenceEventID: input.EventID,
	}, nil
}

type ExcludeInput struct {
	EventID    domain.EvidenceEventID
	ChangeID   domain.ChangeID
	OccurredAt time.Time
	Actor      domain.ActorID
	SourceRef  domain.EvidenceReference
	Reason     domain.EvidenceReasonCode
	Synthetic  bool
}

type ExcludeReceipt struct {
	CanaryID        domain.CanaryID           `json:"canary_id"`
	ManifestVersion uint32                    `json:"manifest_version"`
	ChangeID        domain.ChangeID           `json:"change_id"`
	Decision        domain.AdmissionDecision  `json:"decision"`
	Reason          domain.EvidenceReasonCode `json:"reason"`
	EvidenceEventID domain.EvidenceEventID    `json:"evidence_event_id"`
}

// Exclude records a candidate decision without calculating or returning the
// next ordinal or variant.
func (s *Service) Exclude(input ExcludeInput) (ExcludeReceipt, error) {
	manifest := s.manifest
	events, err := s.store.ReadAll()
	if err != nil {
		return ExcludeReceipt{}, err
	}
	for _, event := range events {
		if event.CanaryID == manifest.ID && event.Kind == domain.EventCanaryStopped {
			return ExcludeReceipt{}, ErrCanaryStopped
		}
		if event.CanaryID == manifest.ID && event.ChangeID == input.ChangeID &&
			(event.Admission != nil || event.Assignment != nil) {
			return ExcludeReceipt{}, ErrChangeAlreadyStarted
		}
	}
	if !input.Synthetic {
		return ExcludeReceipt{}, ErrRealCanaryNotActivated
	}
	event := domain.EvidenceEvent{
		ID:              input.EventID,
		CanaryID:        manifest.ID,
		ManifestVersion: s.eventManifestVersion(),
		ChangeID:        input.ChangeID,
		Kind:            domain.EventAdmissionDecided,
		OccurredAt:      input.OccurredAt,
		Actor:           input.Actor,
		SourceRef:       input.SourceRef,
		ReasonCode:      input.Reason,
		Admission: &domain.CanaryAdmission{
			Decision:  domain.AdmissionExcluded,
			Synthetic: true,
		},
	}
	if err := s.store.Append(event); err != nil {
		return ExcludeReceipt{}, err
	}
	return ExcludeReceipt{
		CanaryID:        manifest.ID,
		ManifestVersion: manifest.Version,
		ChangeID:        input.ChangeID,
		Decision:        domain.AdmissionExcluded,
		Reason:          input.Reason,
		EvidenceEventID: input.EventID,
	}, nil
}

type DisableInput struct {
	EventID    domain.EvidenceEventID
	OccurredAt time.Time
	Actor      domain.ActorID
	SourceRef  domain.EvidenceReference
	Reason     domain.EvidenceReasonCode
}

type DisableReceipt struct {
	CanaryID           domain.CanaryID           `json:"canary_id"`
	ManifestVersion    uint32                    `json:"manifest_version"`
	AssignmentsStopped bool                      `json:"assignments_stopped"`
	Reason             domain.EvidenceReasonCode `json:"reason"`
	StoppedAt          time.Time                 `json:"stopped_at"`
	EvidenceEventID    domain.EvidenceEventID    `json:"evidence_event_id"`
}

// Disable appends a canary-level stop marker. Existing change evidence remains
// readable and may receive terminal evidence; only new assignments are barred.
func (s *Service) Disable(input DisableInput) (DisableReceipt, error) {
	manifest := s.manifest
	events, err := s.store.ReadAll()
	if err != nil {
		return DisableReceipt{}, err
	}
	for _, event := range events {
		if event.CanaryID == manifest.ID && event.Kind == domain.EventCanaryStopped {
			return DisableReceipt{}, ErrCanaryAlreadyStopped
		}
	}
	event := domain.EvidenceEvent{
		ID:              input.EventID,
		CanaryID:        manifest.ID,
		ManifestVersion: s.eventManifestVersion(),
		Kind:            domain.EventCanaryStopped,
		OccurredAt:      input.OccurredAt,
		Actor:           input.Actor,
		SourceRef:       input.SourceRef,
		ReasonCode:      input.Reason,
	}
	if err := s.store.Append(event); err != nil {
		return DisableReceipt{}, err
	}
	return DisableReceipt{
		CanaryID:           manifest.ID,
		ManifestVersion:    manifest.Version,
		AssignmentsStopped: true,
		Reason:             input.Reason,
		StoppedAt:          input.OccurredAt,
		EvidenceEventID:    input.EventID,
	}, nil
}

type HookInput struct {
	EventID        domain.EvidenceEventID
	OccurredAt     time.Time
	Actor          domain.ActorID
	SourceRef      domain.EvidenceReference
	EncodedContext string
	RawHook        []byte
}

func (s *Service) BindLifecycleHook(input HookInput) (codex.CorrelationResult, error) {
	runContext, err := codex.DecodeRunContext([]byte(input.EncodedContext))
	if err != nil {
		return codex.CorrelationResult{}, err
	}
	view, err := s.Inspect(runContext.ChangeID())
	if err != nil {
		return codex.CorrelationResult{}, err
	}
	if view.Assignment == nil || view.Assignment.RunID != runContext.RunID() {
		return codex.CorrelationResult{}, ErrChangeNotStarted
	}
	previous := previousCorrelation(view)
	result, err := codex.BindLifecycleHook(runContext, input.EncodedContext, input.RawHook, previous)
	if err != nil {
		return codex.CorrelationResult{}, err
	}
	if err := s.appendCorrelation(
		input.EventID,
		input.OccurredAt,
		input.Actor,
		input.SourceRef,
		runContext,
		result,
	); err != nil {
		return codex.CorrelationResult{}, err
	}
	return result, nil
}

type LaunchReceiptInput struct {
	EventID        domain.EvidenceEventID
	OccurredAt     time.Time
	Actor          domain.ActorID
	SourceRef      domain.EvidenceReference
	EncodedContext string
	RawReceipt     []byte
}

// BindLaunchReceipt consumes the provider response directly. Callers cannot
// supply a session ID field separately from that authoritative receipt.
func (s *Service) BindLaunchReceipt(input LaunchReceiptInput) (codex.CorrelationResult, error) {
	runContext, err := codex.DecodeRunContext([]byte(input.EncodedContext))
	if err != nil {
		return codex.CorrelationResult{}, err
	}
	view, err := s.Inspect(runContext.ChangeID())
	if err != nil {
		return codex.CorrelationResult{}, err
	}
	if view.Assignment == nil || view.Assignment.RunID != runContext.RunID() {
		return codex.CorrelationResult{}, ErrChangeNotStarted
	}
	previous := previousCorrelation(view)
	result, err := codex.BindLaunchReceipt(runContext, input.RawReceipt, previous)
	if err != nil {
		return codex.CorrelationResult{}, err
	}
	if err := s.appendCorrelation(
		input.EventID,
		input.OccurredAt,
		input.Actor,
		input.SourceRef,
		runContext,
		result,
	); err != nil {
		return codex.CorrelationResult{}, err
	}
	return result, nil
}

func previousCorrelation(view ChangeView) *codex.CorrelationResult {
	if view.Lineage == nil {
		return nil
	}
	return &codex.CorrelationResult{
		Lineage:            *view.Lineage,
		ResolutionAttempts: view.LineageResolutionAttempts,
	}
}

func (s *Service) appendCorrelation(
	eventID domain.EvidenceEventID,
	occurredAt time.Time,
	actor domain.ActorID,
	sourceRef domain.EvidenceReference,
	runContext codex.RunContext,
	result codex.CorrelationResult,
) error {
	var observationRefs []domain.EvidenceReference
	if result.Lineage.Status == domain.LineageVerified {
		references, err := langfuse.BuildObservationReferences(result.Lineage, nil)
		if err != nil {
			return err
		}
		observationRefs = references.EvidenceReferences()
	}
	lineage := result.Lineage
	return s.store.Append(domain.EvidenceEvent{
		ID:                        eventID,
		CanaryID:                  s.manifest.ID,
		ManifestVersion:           s.eventManifestVersion(),
		ChangeID:                  runContext.ChangeID(),
		Kind:                      domain.EventLineageRecorded,
		OccurredAt:                occurredAt,
		Actor:                     actor,
		SourceRef:                 sourceRef,
		ObservationRefs:           observationRefs,
		Lineage:                   &lineage,
		LineageResolutionAttempts: result.ResolutionAttempts,
	})
}

type ContextBindingInput struct {
	EventID            domain.EvidenceEventID
	ChangeID           domain.ChangeID
	OccurredAt         time.Time
	Actor              domain.ActorID
	SourceRef          domain.EvidenceReference
	ContextPackID      domain.ContextPackID
	ContextPackVersion uint32
}

func (s *Service) RecordContextBinding(input ContextBindingInput) error {
	if _, err := s.requireOpenChange(input.ChangeID); err != nil {
		return err
	}
	return s.store.Append(domain.EvidenceEvent{
		ID:              input.EventID,
		CanaryID:        s.manifest.ID,
		ManifestVersion: s.eventManifestVersion(),
		ChangeID:        input.ChangeID,
		Kind:            domain.EventContextBound,
		OccurredAt:      input.OccurredAt,
		Actor:           input.Actor,
		SourceRef:       input.SourceRef,
		Context: &domain.CanaryContextBinding{
			ContextPackID:      input.ContextPackID,
			ContextPackVersion: input.ContextPackVersion,
		},
	})
}

type AssessmentBasisInput struct {
	EventID    domain.EvidenceEventID
	ChangeID   domain.ChangeID
	OccurredAt time.Time
	Actor      domain.ActorID
	SourceRef  domain.EvidenceReference
	Basis      domain.CanaryAssessmentBasis
}

func (s *Service) RecordAssessmentBasis(input AssessmentBasisInput) error {
	view, err := s.Inspect(input.ChangeID)
	if err != nil {
		return err
	}
	if view.Assignment == nil {
		return ErrChangeNotStarted
	}
	basis := copyAssessmentBasis(input.Basis)
	return s.store.Append(domain.EvidenceEvent{
		ID:              input.EventID,
		CanaryID:        s.manifest.ID,
		ManifestVersion: s.eventManifestVersion(),
		ChangeID:        input.ChangeID,
		Kind:            domain.EventAssessmentBasisRecorded,
		OccurredAt:      input.OccurredAt,
		Actor:           input.Actor,
		SourceRef:       input.SourceRef,
		AssessmentBasis: &basis,
	})
}

type FlowPhaseInput struct {
	EventID     domain.EvidenceEventID
	ChangeID    domain.ChangeID
	OccurredAt  time.Time
	Actor       domain.ActorID
	SourceRef   domain.EvidenceReference
	StartedAt   time.Time
	CompletedAt time.Time
}

func (s *Service) RecordFlowPhase(input FlowPhaseInput) error {
	if _, err := s.requireOpenChange(input.ChangeID); err != nil {
		return err
	}
	return s.store.Append(domain.EvidenceEvent{
		ID:              input.EventID,
		CanaryID:        s.manifest.ID,
		ManifestVersion: s.eventManifestVersion(),
		ChangeID:        input.ChangeID,
		Kind:            domain.EventFlowPhaseRecorded,
		OccurredAt:      input.OccurredAt,
		Actor:           input.Actor,
		SourceRef:       input.SourceRef,
		FlowPhase: &domain.CanaryFlowPhase{
			StartedAt:   input.StartedAt,
			CompletedAt: input.CompletedAt,
		},
	})
}

type TelemetryEvidenceInput struct {
	EventID          domain.EvidenceEventID
	ChangeID         domain.ChangeID
	OccurredAt       time.Time
	Actor            domain.ActorID
	SourceRef        domain.EvidenceReference
	Reason           domain.EvidenceReasonCode
	Telemetry        domain.CanaryTelemetry
	CorrectionReason domain.EvidenceReasonCode
}

func (s *Service) RecordTelemetryEvidence(input TelemetryEvidenceInput) error {
	view, err := s.Inspect(input.ChangeID)
	if err != nil {
		return err
	}
	if view.Assignment == nil {
		return ErrChangeNotStarted
	}
	telemetry := copyTelemetry(input.Telemetry)
	event := domain.EvidenceEvent{
		ID:              input.EventID,
		CanaryID:        s.manifest.ID,
		ManifestVersion: s.eventManifestVersion(),
		ChangeID:        input.ChangeID,
		Kind:            domain.EventTelemetryRecorded,
		OccurredAt:      input.OccurredAt,
		Actor:           input.Actor,
		SourceRef:       input.SourceRef,
		ReasonCode:      input.Reason,
		ObservationRefs: telemetryObservationRefs(telemetry),
		Telemetry:       &telemetry,
	}
	if view.Telemetry != nil {
		if input.CorrectionReason == "" || view.TelemetryEventID == "" {
			return ErrCorrectionState
		}
		event.Kind = domain.EventEvidenceCorrected
		if input.Reason == "" {
			event.ReasonCode = input.CorrectionReason
		}
		event.SupersedesEventID = view.TelemetryEventID
	}
	return s.store.Append(event)
}

type FreezeChecksInput struct {
	EventID    domain.EvidenceEventID
	ChangeID   domain.ChangeID
	OccurredAt time.Time
	Actor      domain.ActorID
	SourceRef  domain.EvidenceReference
	CheckRefs  []domain.EvidenceReference
}

type FreezeChecksReceipt struct {
	ChangeID        domain.ChangeID            `json:"change_id"`
	CheckRefs       []domain.EvidenceReference `json:"check_refs"`
	EvidenceEventID domain.EvidenceEventID     `json:"evidence_event_id"`
}

func (s *Service) FreezeChecks(input FreezeChecksInput) (FreezeChecksReceipt, error) {
	view, err := s.requireOpenChange(input.ChangeID)
	if err != nil {
		return FreezeChecksReceipt{}, err
	}
	if view.CheckSet != nil {
		return FreezeChecksReceipt{}, ErrCheckSetState
	}
	checkRefs := append([]domain.EvidenceReference(nil), input.CheckRefs...)
	if err := s.store.Append(domain.EvidenceEvent{
		ID:              input.EventID,
		CanaryID:        s.manifest.ID,
		ManifestVersion: s.eventManifestVersion(),
		ChangeID:        input.ChangeID,
		Kind:            domain.EventCheckSetFrozen,
		OccurredAt:      input.OccurredAt,
		Actor:           input.Actor,
		SourceRef:       input.SourceRef,
		CheckSet:        &domain.CanaryCheckSet{CheckRefs: checkRefs},
	}); err != nil {
		return FreezeChecksReceipt{}, err
	}
	return FreezeChecksReceipt{
		ChangeID:        input.ChangeID,
		CheckRefs:       append([]domain.EvidenceReference(nil), checkRefs...),
		EvidenceEventID: input.EventID,
	}, nil
}

type CorrectChecksInput struct {
	EventID    domain.EvidenceEventID
	ChangeID   domain.ChangeID
	OccurredAt time.Time
	Actor      domain.ActorID
	SourceRef  domain.EvidenceReference
	Reason     domain.EvidenceReasonCode
	CheckRefs  []domain.EvidenceReference
}

func (s *Service) CorrectChecks(input CorrectChecksInput) (FreezeChecksReceipt, error) {
	view, err := s.requireOpenChange(input.ChangeID)
	if err != nil {
		return FreezeChecksReceipt{}, err
	}
	if view.CheckSet == nil || view.CheckSetEventID == "" {
		return FreezeChecksReceipt{}, ErrCheckSetState
	}
	checkRefs := append([]domain.EvidenceReference(nil), input.CheckRefs...)
	if err := s.store.Append(domain.EvidenceEvent{
		ID:                input.EventID,
		CanaryID:          s.manifest.ID,
		ManifestVersion:   s.eventManifestVersion(),
		ChangeID:          input.ChangeID,
		Kind:              domain.EventEvidenceCorrected,
		OccurredAt:        input.OccurredAt,
		Actor:             input.Actor,
		SourceRef:         input.SourceRef,
		ReasonCode:        input.Reason,
		SupersedesEventID: view.CheckSetEventID,
		CheckSet:          &domain.CanaryCheckSet{CheckRefs: checkRefs},
	}); err != nil {
		return FreezeChecksReceipt{}, err
	}
	return FreezeChecksReceipt{
		ChangeID:        input.ChangeID,
		CheckRefs:       append([]domain.EvidenceReference(nil), checkRefs...),
		EvidenceEventID: input.EventID,
	}, nil
}

type DeliverInput struct {
	EventID             domain.EvidenceEventID
	ChangeID            domain.ChangeID
	OccurredAt          time.Time
	Actor               domain.ActorID
	SourceRef           domain.EvidenceReference
	CheckRefs           []domain.EvidenceReference
	ChecksGreen         bool
	FlowOverheadMinutes *float64
}

func (s *Service) Deliver(input DeliverInput) error {
	view, err := s.requireOpenChange(input.ChangeID)
	if err != nil {
		return err
	}
	if s.manifest.Version == domain.IntentCanaryV0ManifestVersion2 && input.FlowOverheadMinutes != nil {
		return errors.New("manifest v2 derives flow overhead from telemetry evidence")
	}
	if view.Assignment.Variant == domain.VariantBaseline && input.FlowOverheadMinutes != nil {
		return errors.New("baseline change cannot record flow overhead")
	}
	flowOverhead := input.FlowOverheadMinutes
	if s.manifest.Version == domain.IntentCanaryV0ManifestVersion2 && view.Assignment.Variant == domain.VariantFlow {
		flowOverhead = nil
		if view.Telemetry != nil && view.Telemetry.FlowOverhead != nil && view.Telemetry.FlowOverhead.Available {
			flowOverhead = copyFloat(&view.Telemetry.FlowOverhead.TotalMinutes)
		}
	}
	terminal := &domain.CanaryTerminal{
		State:               domain.CanaryStateDelivered,
		CheckRefs:           append([]domain.EvidenceReference(nil), input.CheckRefs...),
		ChecksGreen:         input.ChecksGreen,
		FlowOverheadMinutes: flowOverhead,
	}
	return s.store.Append(domain.EvidenceEvent{
		ID:              input.EventID,
		CanaryID:        s.manifest.ID,
		ManifestVersion: s.eventManifestVersion(),
		ChangeID:        input.ChangeID,
		Kind:            domain.EventTerminalStateChanged,
		OccurredAt:      input.OccurredAt,
		Actor:           input.Actor,
		SourceRef:       input.SourceRef,
		Terminal:        terminal,
	})
}

type AbandonInput struct {
	EventID             domain.EvidenceEventID
	ChangeID            domain.ChangeID
	OccurredAt          time.Time
	Actor               domain.ActorID
	SourceRef           domain.EvidenceReference
	Reason              domain.EvidenceReasonCode
	ProcessCaused       bool
	FlowOverheadMinutes *float64
}

func (s *Service) Abandon(input AbandonInput) error {
	view, err := s.requireOpenChange(input.ChangeID)
	if err != nil {
		return err
	}
	if s.manifest.Version == domain.IntentCanaryV0ManifestVersion2 && input.FlowOverheadMinutes != nil {
		return errors.New("manifest v2 derives flow overhead from telemetry evidence")
	}
	if view.Assignment.Variant == domain.VariantBaseline &&
		(input.ProcessCaused || input.FlowOverheadMinutes != nil) {
		return errors.New("baseline change cannot record flow-only abandonment data")
	}
	flowOverhead := input.FlowOverheadMinutes
	if s.manifest.Version == domain.IntentCanaryV0ManifestVersion2 && view.Assignment.Variant == domain.VariantFlow {
		flowOverhead = nil
		if view.Telemetry != nil && view.Telemetry.FlowOverhead != nil && view.Telemetry.FlowOverhead.Available {
			flowOverhead = copyFloat(&view.Telemetry.FlowOverhead.TotalMinutes)
		}
	}
	return s.store.Append(domain.EvidenceEvent{
		ID:              input.EventID,
		CanaryID:        s.manifest.ID,
		ManifestVersion: s.eventManifestVersion(),
		ChangeID:        input.ChangeID,
		Kind:            domain.EventTerminalStateChanged,
		OccurredAt:      input.OccurredAt,
		Actor:           input.Actor,
		SourceRef:       input.SourceRef,
		Terminal: &domain.CanaryTerminal{
			State:                    domain.CanaryStateAbandoned,
			FlowOverheadMinutes:      flowOverhead,
			AbandonmentReason:        input.Reason,
			ProcessCausedAbandonment: input.ProcessCaused,
		},
	})
}

type AssessInput struct {
	EventID       domain.EvidenceEventID
	ChangeID      domain.ChangeID
	OccurredAt    time.Time
	Owner         domain.ActorID
	SourceRef     domain.EvidenceReference
	Outcome       domain.IntentOutcome
	RepeatOptIn   *bool
	ItemJudgments []domain.IntentItemJudgment
}

func (s *Service) Assess(input AssessInput) error {
	view, err := s.Inspect(input.ChangeID)
	if err != nil {
		return err
	}
	if view.Terminal == nil || view.Terminal.State != domain.CanaryStateDelivered || view.Assessment != nil {
		return ErrAssessmentState
	}
	if view.Assignment == nil ||
		(view.Assignment.Variant == domain.VariantFlow && input.RepeatOptIn == nil) ||
		(view.Assignment.Variant == domain.VariantBaseline && input.RepeatOptIn != nil) {
		return ErrAssessmentState
	}
	assessment := &domain.Assessment{
		Outcome:                          input.Outcome,
		AssessedBy:                       input.Owner,
		AssessedAt:                       input.OccurredAt,
		ChecksGreen:                      view.Terminal.ChecksGreen,
		MaterialCorrectionBeforeDelivery: view.MaterialCorrections > 0,
		RepeatOptIn:                      input.RepeatOptIn,
		ItemJudgments:                    append([]domain.IntentItemJudgment(nil), input.ItemJudgments...),
	}
	return s.store.Append(domain.EvidenceEvent{
		ID:              input.EventID,
		CanaryID:        s.manifest.ID,
		ManifestVersion: s.eventManifestVersion(),
		ChangeID:        input.ChangeID,
		Kind:            domain.EventAssessmentRecorded,
		OccurredAt:      input.OccurredAt,
		Actor:           input.Owner,
		SourceRef:       input.SourceRef,
		Assessment:      assessment,
	})
}

type CorrectAssessmentInput struct {
	EventID       domain.EvidenceEventID
	ChangeID      domain.ChangeID
	OccurredAt    time.Time
	Owner         domain.ActorID
	SourceRef     domain.EvidenceReference
	Reason        domain.EvidenceReasonCode
	Outcome       domain.IntentOutcome
	RepeatOptIn   *bool
	ItemJudgments []domain.IntentItemJudgment
}

func (s *Service) CorrectAssessment(input CorrectAssessmentInput) error {
	view, err := s.Inspect(input.ChangeID)
	if err != nil {
		return err
	}
	if view.Assessment == nil || view.AssessmentEventID == "" || view.Terminal == nil {
		return ErrCorrectionState
	}
	if view.Assignment == nil ||
		(view.Assignment.Variant == domain.VariantFlow && input.RepeatOptIn == nil) ||
		(view.Assignment.Variant == domain.VariantBaseline && input.RepeatOptIn != nil) {
		return ErrCorrectionState
	}
	return s.store.Append(domain.EvidenceEvent{
		ID:                input.EventID,
		CanaryID:          s.manifest.ID,
		ManifestVersion:   s.eventManifestVersion(),
		ChangeID:          input.ChangeID,
		Kind:              domain.EventEvidenceCorrected,
		OccurredAt:        input.OccurredAt,
		Actor:             input.Owner,
		SourceRef:         input.SourceRef,
		ReasonCode:        input.Reason,
		SupersedesEventID: view.AssessmentEventID,
		Assessment: &domain.Assessment{
			Outcome:                          input.Outcome,
			AssessedBy:                       input.Owner,
			AssessedAt:                       input.OccurredAt,
			ChecksGreen:                      view.Terminal.ChecksGreen,
			MaterialCorrectionBeforeDelivery: view.MaterialCorrections > 0,
			RepeatOptIn:                      input.RepeatOptIn,
			ItemJudgments:                    append([]domain.IntentItemJudgment(nil), input.ItemJudgments...),
		},
	})
}

type MaterialCorrectionInput struct {
	EventID    domain.EvidenceEventID
	ChangeID   domain.ChangeID
	OccurredAt time.Time
	Owner      domain.ActorID
	SourceRef  domain.EvidenceReference
	Reason     domain.EvidenceReasonCode
}

// RecordMaterialCorrection records the owner's contemporaneous classification
// while the change is still open. Wording-only edits intentionally emit no
// material-correction event.
func (s *Service) RecordMaterialCorrection(input MaterialCorrectionInput) error {
	if _, err := s.requireOpenChange(input.ChangeID); err != nil {
		return err
	}
	return s.store.Append(domain.EvidenceEvent{
		ID:              input.EventID,
		CanaryID:        s.manifest.ID,
		ManifestVersion: s.eventManifestVersion(),
		ChangeID:        input.ChangeID,
		Kind:            domain.EventMaterialCorrection,
		OccurredAt:      input.OccurredAt,
		Actor:           input.Owner,
		SourceRef:       input.SourceRef,
		ReasonCode:      input.Reason,
	})
}

type ChangeView struct {
	ChangeID                  domain.ChangeID               `json:"change_id"`
	ManifestVersion           uint32                        `json:"manifest_version"`
	Admission                 *domain.CanaryAdmission       `json:"admission,omitempty"`
	AdmissionReason           domain.EvidenceReasonCode     `json:"admission_reason,omitempty"`
	AdmissionEventID          domain.EvidenceEventID        `json:"admission_event_id,omitempty"`
	Assignment                *domain.CanaryAssignment      `json:"assignment,omitempty"`
	AssignedAt                time.Time                     `json:"assigned_at,omitempty"`
	Context                   *domain.CanaryContextBinding  `json:"context,omitempty"`
	ContextEventID            domain.EvidenceEventID        `json:"context_event_id,omitempty"`
	AssessmentBasis           *domain.CanaryAssessmentBasis `json:"assessment_basis,omitempty"`
	AssessmentBasisEventID    domain.EvidenceEventID        `json:"assessment_basis_event_id,omitempty"`
	FlowPhase                 *domain.CanaryFlowPhase       `json:"flow_phase,omitempty"`
	FlowPhaseEventID          domain.EvidenceEventID        `json:"flow_phase_event_id,omitempty"`
	CheckSet                  *domain.CanaryCheckSet        `json:"check_set,omitempty"`
	CheckSetEventID           domain.EvidenceEventID        `json:"check_set_event_id,omitempty"`
	Lineage                   *domain.ExecutionLineage      `json:"lineage,omitempty"`
	LineageResolutionAttempts uint8                         `json:"lineage_resolution_attempts,omitempty"`
	Telemetry                 *domain.CanaryTelemetry       `json:"telemetry,omitempty"`
	TelemetryEventID          domain.EvidenceEventID        `json:"telemetry_event_id,omitempty"`
	Terminal                  *domain.CanaryTerminal        `json:"terminal,omitempty"`
	TerminalAt                time.Time                     `json:"terminal_at,omitempty"`
	Assessment                *domain.Assessment            `json:"assessment,omitempty"`
	AssessmentEventID         domain.EvidenceEventID        `json:"assessment_event_id,omitempty"`
	MaterialCorrections       uint32                        `json:"material_corrections"`
	EventCount                uint32                        `json:"event_count"`
	sessionConflictObserved   bool
}

func (s *Service) Inspect(changeID domain.ChangeID) (ChangeView, error) {
	events, err := s.store.ReadAll()
	if err != nil {
		return ChangeView{}, err
	}
	return projectChange(events, changeID, s.manifest.Version), nil
}

// Report deterministically reduces the verified append-only event record into
// the provider-neutral canary report. It does not query providers or infer
// missing owner data.
func (s *Service) Report() (domain.CanaryReport, error) {
	events, err := s.store.ReadAll()
	if err != nil {
		return domain.CanaryReport{}, err
	}
	changeIDs := make([]domain.ChangeID, 0, domain.CanaryAssignmentCount)
	seen := make(map[domain.ChangeID]struct{}, domain.CanaryAssignmentCount)
	exclusionReasons := make(map[domain.EvidenceReasonCode]uint32)
	var excluded uint32
	assignmentsStopped := false
	for _, event := range events {
		if eventMatchesManifest(event, s.manifest.Version) && event.Kind == domain.EventCanaryStopped {
			assignmentsStopped = true
		}
		if !eventMatchesManifest(event, s.manifest.Version) || event.Assignment == nil {
			if eventMatchesManifest(event, s.manifest.Version) && event.Admission != nil &&
				event.Admission.Decision == domain.AdmissionExcluded {
				excluded++
				exclusionReasons[event.ReasonCode]++
			}
			continue
		}
		if _, duplicate := seen[event.ChangeID]; duplicate {
			continue
		}
		seen[event.ChangeID] = struct{}{}
		changeIDs = append(changeIDs, event.ChangeID)
	}
	observations := make([]domain.CanaryObservation, 0, len(changeIDs))
	for _, changeID := range changeIDs {
		view := projectChange(events, changeID, s.manifest.Version)
		if view.Assignment == nil {
			continue
		}
		observation := domain.CanaryObservation{
			Ordinal:           view.Assignment.Ordinal,
			ChangeID:          changeID,
			Variant:           view.Assignment.Variant,
			TerminalState:     domain.CanaryStatePending,
			LineageOutcome:    lineageOutcome(view),
			TelemetryRequired: view.Assignment.ManifestVersion == domain.IntentCanaryV0ManifestVersion2,
		}
		if view.Telemetry != nil {
			observation.TelemetryStatus = view.Telemetry.Status
		}
		if view.Terminal != nil {
			observation.TerminalState = view.Terminal.State
			observation.FlowOverheadMinutes = copyFloat(view.Terminal.FlowOverheadMinutes)
			observation.ProcessCausedAbandonment = view.Terminal.ProcessCausedAbandonment
		}
		if view.Assessment != nil {
			assessment := copyAssessment(*view.Assessment)
			observation.Assessment = &assessment
		}
		observation.MaterialMisunderstandingPrevented =
			view.Terminal != nil && view.Terminal.State == domain.CanaryStateDelivered &&
				view.MaterialCorrections > 0
		observations = append(observations, observation)
	}
	sort.Slice(observations, func(i, j int) bool {
		return observations[i].Ordinal < observations[j].Ordinal
	})
	return domain.CalculateCanaryReport(domain.CanaryReportInput{
		Observations:       observations,
		Excluded:           excluded,
		ExclusionReasons:   exclusionReasons,
		AssignmentsStopped: assignmentsStopped,
	})
}

func projectChange(events []domain.EvidenceEvent, changeID domain.ChangeID, manifestVersion uint32) ChangeView {
	view := ChangeView{ChangeID: changeID, ManifestVersion: manifestVersion}
	for _, event := range events {
		if !eventMatchesManifest(event, manifestVersion) || event.ChangeID != changeID {
			continue
		}
		view.EventCount++
		if event.Admission != nil {
			admission := *event.Admission
			view.Admission = &admission
			view.AdmissionReason = event.ReasonCode
			view.AdmissionEventID = event.ID
		}
		if event.Assignment != nil {
			assignment := *event.Assignment
			view.Assignment = &assignment
			view.AssignedAt = event.OccurredAt
		}
		if event.Context != nil {
			contextBinding := *event.Context
			view.Context = &contextBinding
			view.ContextEventID = event.ID
		}
		if event.AssessmentBasis != nil {
			basis := copyAssessmentBasis(*event.AssessmentBasis)
			view.AssessmentBasis = &basis
			view.AssessmentBasisEventID = event.ID
		}
		if event.FlowPhase != nil {
			phase := *event.FlowPhase
			view.FlowPhase = &phase
			view.FlowPhaseEventID = event.ID
		}
		if event.Kind == domain.EventCheckSetFrozen && event.CheckSet != nil {
			checkSet := copyCheckSet(*event.CheckSet)
			view.CheckSet = &checkSet
			view.CheckSetEventID = event.ID
		}
		if event.Kind == domain.EventEvidenceCorrected && event.CheckSet != nil &&
			event.SupersedesEventID == view.CheckSetEventID {
			checkSet := copyCheckSet(*event.CheckSet)
			view.CheckSet = &checkSet
			view.CheckSetEventID = event.ID
		}
		if event.Lineage != nil {
			lineage := *event.Lineage
			view.Lineage = &lineage
			view.LineageResolutionAttempts = event.LineageResolutionAttempts
			if lineage.Status == domain.LineageUnlinked &&
				lineage.UnlinkedReasonCode == codex.ReasonSessionConflict &&
				event.LineageResolutionAttempts >= 1 {
				view.sessionConflictObserved = true
			}
		}
		if event.Kind == domain.EventTelemetryRecorded && event.Telemetry != nil {
			telemetry := copyTelemetry(*event.Telemetry)
			view.Telemetry = &telemetry
			view.TelemetryEventID = event.ID
		}
		if event.Kind == domain.EventEvidenceCorrected && event.Telemetry != nil &&
			event.SupersedesEventID == view.TelemetryEventID {
			telemetry := copyTelemetry(*event.Telemetry)
			view.Telemetry = &telemetry
			view.TelemetryEventID = event.ID
		}
		if event.Terminal != nil {
			terminal := *event.Terminal
			terminal.CheckRefs = append([]domain.EvidenceReference(nil), event.Terminal.CheckRefs...)
			view.Terminal = &terminal
			view.TerminalAt = event.OccurredAt
		}
		if event.Kind == domain.EventAssessmentRecorded && event.Assessment != nil {
			assessment := copyAssessment(*event.Assessment)
			view.Assessment = &assessment
			view.AssessmentEventID = event.ID
		}
		if event.Kind == domain.EventEvidenceCorrected && event.Assessment != nil &&
			event.SupersedesEventID == view.AssessmentEventID {
			assessment := copyAssessment(*event.Assessment)
			view.Assessment = &assessment
			view.AssessmentEventID = event.ID
		}
		if event.Kind == domain.EventMaterialCorrection {
			view.MaterialCorrections++
		}
	}
	return view
}

func eventMatchesManifest(event domain.EvidenceEvent, manifestVersion uint32) bool {
	return event.CanaryID == domain.IntentCanaryV0ManifestID &&
		effectiveEventManifestVersion(event) == manifestVersion
}

func effectiveEventManifestVersion(event domain.EvidenceEvent) uint32 {
	if event.ManifestVersion == 0 {
		return domain.IntentCanaryV0ManifestVersion
	}
	return event.ManifestVersion
}

func (s *Service) eventManifestVersion() uint32 {
	if s.manifest.Version == domain.IntentCanaryV0ManifestVersion {
		return 0
	}
	return s.manifest.Version
}

func lineageOutcome(view ChangeView) domain.CanaryLineageOutcome {
	if view.sessionConflictObserved {
		return domain.CanaryLineageWrong
	}
	if view.Lineage == nil {
		return domain.CanaryLineagePending
	}
	if view.Lineage.Status == domain.LineageVerified {
		return domain.CanaryLineageVerified
	}
	if view.Lineage.Status == domain.LineageUnlinked && view.LineageResolutionAttempts >= 1 {
		if view.Lineage.UnlinkedReasonCode == codex.ReasonSessionConflict {
			return domain.CanaryLineageWrong
		}
		return domain.CanaryLineageUnresolvedAfterResolution
	}
	return domain.CanaryLineagePending
}

func copyFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func copyAssessment(assessment domain.Assessment) domain.Assessment {
	if assessment.RepeatOptIn != nil {
		repeat := *assessment.RepeatOptIn
		assessment.RepeatOptIn = &repeat
	}
	assessment.ItemJudgments = append([]domain.IntentItemJudgment(nil), assessment.ItemJudgments...)
	return assessment
}

func copyCheckSet(checkSet domain.CanaryCheckSet) domain.CanaryCheckSet {
	checkSet.CheckRefs = append([]domain.EvidenceReference(nil), checkSet.CheckRefs...)
	return checkSet
}

func copyAssessmentBasis(basis domain.CanaryAssessmentBasis) domain.CanaryAssessmentBasis {
	basis.DesiredOutcomeIDs = append([]domain.IntentItemID(nil), basis.DesiredOutcomeIDs...)
	basis.NonGoalIDs = append([]domain.IntentItemID(nil), basis.NonGoalIDs...)
	basis.SuccessSignalIDs = append([]domain.IntentItemID(nil), basis.SuccessSignalIDs...)
	return basis
}

func copyTelemetry(telemetry domain.CanaryTelemetry) domain.CanaryTelemetry {
	telemetry.TraceIntervals = append([]domain.CanaryTimingInterval(nil), telemetry.TraceIntervals...)
	if telemetry.OwnerReview != nil {
		ownerReview := *telemetry.OwnerReview
		telemetry.OwnerReview = &ownerReview
	}
	if telemetry.FlowOverhead != nil {
		overhead := *telemetry.FlowOverhead
		telemetry.FlowOverhead = &overhead
	}
	return telemetry
}

func telemetryObservationRefs(telemetry domain.CanaryTelemetry) []domain.EvidenceReference {
	refs := make([]domain.EvidenceReference, 0, len(telemetry.TraceIntervals)+1)
	refs = append(refs, telemetry.SessionLookup)
	for _, interval := range telemetry.TraceIntervals {
		refs = append(refs, interval.Reference)
	}
	return refs
}

func (s *Service) requireOpenChange(changeID domain.ChangeID) (ChangeView, error) {
	view, err := s.Inspect(changeID)
	if err != nil {
		return ChangeView{}, err
	}
	if view.Assignment == nil {
		return ChangeView{}, ErrChangeNotStarted
	}
	if view.Terminal != nil {
		return ChangeView{}, ErrChangeAlreadyTerminal
	}
	return view, nil
}
