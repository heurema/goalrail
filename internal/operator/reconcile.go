package operator

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/heurema/goalrail/internal/adapters/langfuse"
	"github.com/heurema/goalrail/internal/domain"
)

var ErrTraceSource = errors.New("trace source read failed")

const (
	ReasonTelemetryMissing         = domain.EvidenceReasonCode("telemetry-missing")
	ReasonTelemetryReadFailed      = domain.EvidenceReasonCode("telemetry-read-failed")
	ReasonTelemetrySessionConflict = domain.EvidenceReasonCode("telemetry-session-conflict")
	ReasonTelemetryMalformed       = domain.EvidenceReasonCode("telemetry-malformed")
	ReasonFlowPhaseMissing         = domain.EvidenceReasonCode("flow-phase-missing")
	ReasonFlowTraceMixed           = domain.EvidenceReasonCode("flow-trace-mixed")
)

// TraceSource is the replaceable read-only telemetry boundary used by the
// canary operator.
type TraceSource interface {
	ListSessionObservations(context.Context, domain.TraceObservationQuery) ([]domain.TraceObservation, error)
}

type TelemetryReconciliation struct {
	Telemetry domain.CanaryTelemetry
	Reason    domain.EvidenceReasonCode
}

type ReconcileTelemetryInput struct {
	EventID          domain.EvidenceEventID
	ChangeID         domain.ChangeID
	OccurredAt       time.Time
	Actor            domain.ActorID
	SourceRef        domain.EvidenceReference
	OwnerReview      *domain.CanaryTimingInterval
	CorrectionReason domain.EvidenceReasonCode
}

type ReconcileTelemetryReceipt struct {
	ChangeID        domain.ChangeID           `json:"change_id"`
	Telemetry       domain.CanaryTelemetry    `json:"telemetry"`
	Reason          domain.EvidenceReasonCode `json:"reason,omitempty"`
	EvidenceEventID domain.EvidenceEventID    `json:"evidence_event_id"`
}

// ReconcileTelemetry performs one bounded provider read and records its
// effective result. It never polls or retries; a later explicit invocation can
// append one correction to the evidence chain.
func (s *Service) ReconcileTelemetry(
	ctx context.Context,
	source TraceSource,
	input ReconcileTelemetryInput,
) (ReconcileTelemetryReceipt, error) {
	if s.manifest.Version != domain.IntentCanaryV0ManifestVersion2 {
		return ReconcileTelemetryReceipt{}, errors.New("telemetry reconciliation requires manifest version 2")
	}
	if source == nil {
		return ReconcileTelemetryReceipt{}, ErrTraceSource
	}
	view, err := s.Inspect(input.ChangeID)
	if err != nil {
		return ReconcileTelemetryReceipt{}, err
	}
	if view.Assignment == nil {
		return ReconcileTelemetryReceipt{}, ErrChangeNotStarted
	}
	if view.Lineage == nil || view.Lineage.Status != domain.LineageVerified {
		return ReconcileTelemetryReceipt{}, errors.New("telemetry reconciliation requires verified lineage")
	}

	var result TelemetryReconciliation
	if view.Assignment.Variant == domain.VariantFlow && view.FlowPhase == nil {
		result, err = ReconcileTraceObservations(*view.Lineage, view.Assignment.Variant, nil, nil)
	} else {
		from, to := view.AssignedAt, input.OccurredAt
		if view.Assignment.Variant == domain.VariantBaseline && view.Terminal != nil && !view.TerminalAt.IsZero() {
			to = view.TerminalAt
		}
		if view.Assignment.Variant == domain.VariantFlow {
			to = view.FlowPhase.CompletedAt.Add(time.Nanosecond)
		}
		if from.IsZero() || !from.Before(to) {
			return ReconcileTelemetryReceipt{}, errors.New("telemetry reconciliation window is invalid")
		}
		observations, readErr := source.ListSessionObservations(ctx, domain.TraceObservationQuery{
			SessionID: view.Lineage.RootSessionID, FromStartTime: from.UTC(), ToStartTime: to.UTC(),
		})
		if readErr != nil {
			status, reason := domain.TelemetryUnavailable, ReasonTelemetryReadFailed
			if errors.Is(readErr, langfuse.ErrMalformedResponse) {
				status, reason = domain.TelemetryConflict, ReasonTelemetryMalformed
			}
			result = TelemetryReconciliation{
				Telemetry: domain.CanaryTelemetry{
					Status:        status,
					SessionLookup: domain.EvidenceReference("langfuse-session:" + string(view.Lineage.RootSessionID)),
				},
				Reason: reason,
			}
		} else {
			result, err = ReconcileTraceObservations(*view.Lineage, view.Assignment.Variant, view.FlowPhase, observations)
		}
	}
	if err != nil {
		return ReconcileTelemetryReceipt{}, err
	}

	if view.Assignment.Variant == domain.VariantBaseline {
		if input.OwnerReview != nil {
			return ReconcileTelemetryReceipt{}, errors.New("baseline reconciliation cannot record owner-review overhead")
		}
	} else {
		if input.OwnerReview != nil {
			ownerReview := *input.OwnerReview
			result.Telemetry.OwnerReview = &ownerReview
		}
		measurement, measureErr := domain.CalculateCanaryFlowOverhead(domain.CanaryFlowOverheadInput{
			AgentTurns:          result.Telemetry.TraceIntervals,
			OwnerReview:         result.Telemetry.OwnerReview,
			OwnerReviewRequired: true,
		})
		if measureErr != nil {
			return ReconcileTelemetryReceipt{}, measureErr
		}
		result.Telemetry.FlowOverhead = &measurement
	}

	if err := s.RecordTelemetryEvidence(TelemetryEvidenceInput{
		EventID: input.EventID, ChangeID: input.ChangeID, OccurredAt: input.OccurredAt,
		Actor: input.Actor, SourceRef: input.SourceRef, Reason: result.Reason,
		Telemetry: result.Telemetry, CorrectionReason: input.CorrectionReason,
	}); err != nil {
		return ReconcileTelemetryReceipt{}, err
	}
	return ReconcileTelemetryReceipt{
		ChangeID: input.ChangeID, Telemetry: copyTelemetry(result.Telemetry),
		Reason: result.Reason, EvidenceEventID: input.EventID,
	}, nil
}

// ReconcileTraceObservations groups nested observations into one envelope per
// trace, applies the explicit flow window, and never imputes missing data.
func ReconcileTraceObservations(
	lineage domain.ExecutionLineage,
	variant domain.CanaryVariant,
	phase *domain.CanaryFlowPhase,
	observations []domain.TraceObservation,
) (TelemetryReconciliation, error) {
	references, err := langfuse.BuildObservationReferences(lineage, nil)
	if err != nil {
		return TelemetryReconciliation{}, err
	}
	base := domain.CanaryTelemetry{SessionLookup: references.SessionLookup()}
	if variant == domain.VariantFlow && phase == nil {
		base.Status = domain.TelemetryUnavailable
		return TelemetryReconciliation{Telemetry: base, Reason: ReasonFlowPhaseMissing}, nil
	}

	type envelope struct {
		startedAt time.Time
		endedAt   time.Time
	}
	grouped := make(map[domain.EvidenceReference]envelope)
	for _, observation := range observations {
		if observation.SessionID != lineage.RootSessionID {
			base.Status = domain.TelemetryConflict
			return TelemetryReconciliation{Telemetry: base, Reason: ReasonTelemetrySessionConflict}, nil
		}
		traceID, valid := traceIDFromReference(observation.TraceReference)
		if !valid || observation.StartedAt.IsZero() || observation.StartedAt.Location() != time.UTC ||
			observation.EndedAt.IsZero() || observation.EndedAt.Location() != time.UTC ||
			observation.EndedAt.Before(observation.StartedAt) {
			base.Status = domain.TelemetryConflict
			return TelemetryReconciliation{Telemetry: base, Reason: ReasonTelemetryMalformed}, nil
		}
		if _, err := langfuse.BuildObservationReferences(lineage, []langfuse.ObservationIdentity{{
			TraceID: traceID, SessionID: observation.SessionID,
		}}); err != nil {
			base.Status = domain.TelemetryConflict
			return TelemetryReconciliation{Telemetry: base, Reason: ReasonTelemetryMalformed}, nil
		}
		current, exists := grouped[observation.TraceReference]
		if !exists || observation.StartedAt.Before(current.startedAt) {
			current.startedAt = observation.StartedAt
		}
		if !exists || observation.EndedAt.After(current.endedAt) {
			current.endedAt = observation.EndedAt
		}
		grouped[observation.TraceReference] = current
	}

	traceRefs := make([]domain.EvidenceReference, 0, len(grouped))
	for reference := range grouped {
		traceRefs = append(traceRefs, reference)
	}
	sort.Slice(traceRefs, func(i, j int) bool { return traceRefs[i] < traceRefs[j] })
	for _, reference := range traceRefs {
		envelope := grouped[reference]
		if variant == domain.VariantFlow {
			if envelope.startedAt.Before(phase.StartedAt) {
				if envelope.endedAt.After(phase.StartedAt) {
					base.Status = domain.TelemetryUnavailable
					return TelemetryReconciliation{Telemetry: base, Reason: ReasonFlowTraceMixed}, nil
				}
				continue
			}
			if envelope.startedAt.After(phase.CompletedAt) {
				continue
			}
			if envelope.endedAt.After(phase.CompletedAt) {
				base.Status = domain.TelemetryUnavailable
				return TelemetryReconciliation{Telemetry: base, Reason: ReasonFlowTraceMixed}, nil
			}
		}
		base.TraceIntervals = append(base.TraceIntervals, domain.CanaryTimingInterval{
			Reference: reference,
			StartedAt: envelope.startedAt,
			EndedAt:   envelope.endedAt,
		})
	}
	if len(base.TraceIntervals) == 0 {
		base.Status = domain.TelemetryUnavailable
		return TelemetryReconciliation{Telemetry: base, Reason: ReasonTelemetryMissing}, nil
	}
	base.Status = domain.TelemetryAvailable
	return TelemetryReconciliation{Telemetry: base}, nil
}

func traceIDFromReference(reference domain.EvidenceReference) (string, bool) {
	value := string(reference)
	if !strings.HasPrefix(value, "langfuse-trace:") {
		return "", false
	}
	traceID := strings.TrimPrefix(value, "langfuse-trace:")
	return traceID, traceID != ""
}
