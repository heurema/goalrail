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
	ErrRealCanaryNotActivated = errors.New("real-change canary is not activated")
	ErrCanaryStopped          = errors.New("canary assignments are stopped")
	ErrCanaryAlreadyStopped   = errors.New("canary assignments are already stopped")
	ErrChangeAlreadyStarted   = errors.New("canary change already started")
	ErrChangeNotStarted       = errors.New("canary change is not started")
	ErrChangeAlreadyTerminal  = errors.New("canary change already has a terminal state")
	ErrAssessmentState        = errors.New("owner assessment requires one delivered unassessed change")
	ErrCorrectionState        = errors.New("assessment correction requires an existing latest assessment")
)

type Service struct {
	store    *evidence.Store
	repoRoot string
}

func NewService(store *evidence.Store, repoRoot string) (*Service, error) {
	if store == nil {
		return nil, errors.New("operator service requires an evidence store")
	}
	if _, err := codex.NewRunContext("validation-change", "validation-run", repoRoot); err != nil {
		return nil, fmt.Errorf("validate operator repository root: %w", err)
	}
	return &Service{store: store, repoRoot: repoRoot}, nil
}

type StartInput struct {
	EventID       domain.EvidenceEventID
	ChangeID      domain.ChangeID
	RunID         domain.RunID
	IntentVersion uint32
	OccurredAt    time.Time
	Actor         domain.ActorID
	SourceRef     domain.EvidenceReference
	Synthetic     bool
}

type StartReceipt struct {
	CanaryID        domain.CanaryID        `json:"canary_id"`
	ChangeID        domain.ChangeID        `json:"change_id"`
	RunID           domain.RunID           `json:"run_id"`
	Ordinal         uint32                 `json:"ordinal"`
	Variant         domain.CanaryVariant   `json:"variant"`
	RunContextEnv   string                 `json:"run_context_env"`
	EvidenceEventID domain.EvidenceEventID `json:"evidence_event_id"`
}

func (s *Service) Start(input StartInput) (StartReceipt, error) {
	manifest, err := domain.NewIntentCanaryV0Manifest()
	if err != nil {
		return StartReceipt{}, fmt.Errorf("load frozen manifest: %w", err)
	}
	events, err := s.store.ReadAll()
	if err != nil {
		return StartReceipt{}, err
	}
	for _, event := range events {
		if event.CanaryID == manifest.ID && event.Kind == domain.EventCanaryStopped {
			return StartReceipt{}, ErrCanaryStopped
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
		if event.ChangeID == input.ChangeID {
			return StartReceipt{}, ErrChangeAlreadyStarted
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
		ID:         input.EventID,
		CanaryID:   manifest.ID,
		ChangeID:   input.ChangeID,
		Kind:       domain.EventChangeStarted,
		OccurredAt: input.OccurredAt,
		Actor:      input.Actor,
		SourceRef:  input.SourceRef,
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
		ChangeID:        input.ChangeID,
		RunID:           input.RunID,
		Ordinal:         ordinal,
		Variant:         variant,
		RunContextEnv:   encodedContext,
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
	manifest, err := domain.NewIntentCanaryV0Manifest()
	if err != nil {
		return DisableReceipt{}, fmt.Errorf("load frozen manifest: %w", err)
	}
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
		ID:         input.EventID,
		CanaryID:   manifest.ID,
		Kind:       domain.EventCanaryStopped,
		OccurredAt: input.OccurredAt,
		Actor:      input.Actor,
		SourceRef:  input.SourceRef,
		ReasonCode: input.Reason,
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
		CanaryID:                  domain.IntentCanaryV0ManifestID,
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
	if view.Assignment.Variant == domain.VariantBaseline && input.FlowOverheadMinutes != nil {
		return errors.New("baseline change cannot record flow overhead")
	}
	terminal := &domain.CanaryTerminal{
		State:               domain.CanaryStateDelivered,
		CheckRefs:           append([]domain.EvidenceReference(nil), input.CheckRefs...),
		ChecksGreen:         input.ChecksGreen,
		FlowOverheadMinutes: input.FlowOverheadMinutes,
	}
	return s.store.Append(domain.EvidenceEvent{
		ID:         input.EventID,
		CanaryID:   domain.IntentCanaryV0ManifestID,
		ChangeID:   input.ChangeID,
		Kind:       domain.EventTerminalStateChanged,
		OccurredAt: input.OccurredAt,
		Actor:      input.Actor,
		SourceRef:  input.SourceRef,
		Terminal:   terminal,
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
	if view.Assignment.Variant == domain.VariantBaseline &&
		(input.ProcessCaused || input.FlowOverheadMinutes != nil) {
		return errors.New("baseline change cannot record flow-only abandonment data")
	}
	return s.store.Append(domain.EvidenceEvent{
		ID:         input.EventID,
		CanaryID:   domain.IntentCanaryV0ManifestID,
		ChangeID:   input.ChangeID,
		Kind:       domain.EventTerminalStateChanged,
		OccurredAt: input.OccurredAt,
		Actor:      input.Actor,
		SourceRef:  input.SourceRef,
		Terminal: &domain.CanaryTerminal{
			State:                    domain.CanaryStateAbandoned,
			FlowOverheadMinutes:      input.FlowOverheadMinutes,
			AbandonmentReason:        input.Reason,
			ProcessCausedAbandonment: input.ProcessCaused,
		},
	})
}

type AssessInput struct {
	EventID     domain.EvidenceEventID
	ChangeID    domain.ChangeID
	OccurredAt  time.Time
	Owner       domain.ActorID
	SourceRef   domain.EvidenceReference
	Outcome     domain.IntentOutcome
	RepeatOptIn *bool
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
	}
	return s.store.Append(domain.EvidenceEvent{
		ID:         input.EventID,
		CanaryID:   domain.IntentCanaryV0ManifestID,
		ChangeID:   input.ChangeID,
		Kind:       domain.EventAssessmentRecorded,
		OccurredAt: input.OccurredAt,
		Actor:      input.Owner,
		SourceRef:  input.SourceRef,
		Assessment: assessment,
	})
}

type CorrectAssessmentInput struct {
	EventID     domain.EvidenceEventID
	ChangeID    domain.ChangeID
	OccurredAt  time.Time
	Owner       domain.ActorID
	SourceRef   domain.EvidenceReference
	Reason      domain.EvidenceReasonCode
	Outcome     domain.IntentOutcome
	RepeatOptIn *bool
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
		CanaryID:          domain.IntentCanaryV0ManifestID,
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
		ID:         input.EventID,
		CanaryID:   domain.IntentCanaryV0ManifestID,
		ChangeID:   input.ChangeID,
		Kind:       domain.EventMaterialCorrection,
		OccurredAt: input.OccurredAt,
		Actor:      input.Owner,
		SourceRef:  input.SourceRef,
		ReasonCode: input.Reason,
	})
}

type ChangeView struct {
	ChangeID                  domain.ChangeID          `json:"change_id"`
	Assignment                *domain.CanaryAssignment `json:"assignment,omitempty"`
	Lineage                   *domain.ExecutionLineage `json:"lineage,omitempty"`
	LineageResolutionAttempts uint8                    `json:"lineage_resolution_attempts,omitempty"`
	Terminal                  *domain.CanaryTerminal   `json:"terminal,omitempty"`
	Assessment                *domain.Assessment       `json:"assessment,omitempty"`
	AssessmentEventID         domain.EvidenceEventID   `json:"assessment_event_id,omitempty"`
	MaterialCorrections       uint32                   `json:"material_corrections"`
	EventCount                uint32                   `json:"event_count"`
	sessionConflictObserved   bool
}

func (s *Service) Inspect(changeID domain.ChangeID) (ChangeView, error) {
	events, err := s.store.ReadAll()
	if err != nil {
		return ChangeView{}, err
	}
	return projectChange(events, changeID), nil
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
	assignmentsStopped := false
	for _, event := range events {
		if event.CanaryID == domain.IntentCanaryV0ManifestID && event.Kind == domain.EventCanaryStopped {
			assignmentsStopped = true
		}
		if event.CanaryID != domain.IntentCanaryV0ManifestID || event.Assignment == nil {
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
		view := projectChange(events, changeID)
		if view.Assignment == nil {
			continue
		}
		observation := domain.CanaryObservation{
			Ordinal:        view.Assignment.Ordinal,
			ChangeID:       changeID,
			Variant:        view.Assignment.Variant,
			TerminalState:  domain.CanaryStatePending,
			LineageOutcome: lineageOutcome(view),
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
		AssignmentsStopped: assignmentsStopped,
	})
}

func projectChange(events []domain.EvidenceEvent, changeID domain.ChangeID) ChangeView {
	view := ChangeView{ChangeID: changeID}
	for _, event := range events {
		if event.CanaryID != domain.IntentCanaryV0ManifestID || event.ChangeID != changeID {
			continue
		}
		view.EventCount++
		if event.Assignment != nil {
			assignment := *event.Assignment
			view.Assignment = &assignment
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
		if event.Terminal != nil {
			terminal := *event.Terminal
			terminal.CheckRefs = append([]domain.EvidenceReference(nil), event.Terminal.CheckRefs...)
			view.Terminal = &terminal
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
	return assessment
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
