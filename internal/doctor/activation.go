package doctor

import (
	"context"
	"strings"
	"time"

	"github.com/heurema/goalrail/internal/domain"
)

type ActivationState string

const (
	ActivationActive       ActivationState = "active"
	ActivationInactive     ActivationState = "inactive"
	ActivationUnknown      ActivationState = "unknown"
	ActivationStale        ActivationState = "stale"
	ActivationAccessDenied ActivationState = "access_denied"
)

type ActivationFreshness string

const (
	ActivationFreshCurrent    ActivationFreshness = "current"
	ActivationFreshStale      ActivationFreshness = "stale"
	ActivationFreshUnverified ActivationFreshness = "unverified"
)

type ActivationRequest struct {
	ProjectID     domain.ProjectID `json:"project_id"`
	AdapterID     string           `json:"adapter_id"`
	RequiredCheck string           `json:"required_check"`
	PreparedPath  string           `json:"prepared_path"`
}

// ActivationEvidence is deliberately provider-neutral. An adapter may obtain
// it from GitHub, another forge, or a signed local receipt, but active requires
// current evidence about the protected target rather than repository files.
type ActivationEvidence struct {
	State          ActivationState     `json:"state"`
	AdapterID      string              `json:"adapter_id,omitempty"`
	Provider       string              `json:"provider,omitempty"`
	Boundary       string              `json:"boundary,omitempty"`
	RequiredCheck  string              `json:"required_check,omitempty"`
	ObservedTarget string              `json:"observed_target,omitempty"`
	EvidenceSource string              `json:"evidence_source,omitempty"`
	ObservedAt     time.Time           `json:"observed_at,omitempty"`
	Freshness      ActivationFreshness `json:"freshness"`
	Detail         string              `json:"detail,omitempty"`
}

type ActivationObserver interface {
	ObserveActivation(context.Context, ActivationRequest) ActivationEvidence
}

type ActivationObserverFunc func(context.Context, ActivationRequest) ActivationEvidence

func (function ActivationObserverFunc) ObserveActivation(ctx context.Context, request ActivationRequest) ActivationEvidence {
	return function(ctx, request)
}

func activationEvidence(ctx context.Context, observer ActivationObserver, request ActivationRequest) ActivationEvidence {
	if observer == nil {
		return ActivationEvidence{
			State: ActivationUnknown, Freshness: ActivationFreshUnverified,
			RequiredCheck: request.RequiredCheck,
			Detail:        "no read-only protected-admission evidence observer was supplied",
		}
	}
	evidence := observer.ObserveActivation(ctx, request)
	switch evidence.State {
	case ActivationActive, ActivationInactive, ActivationUnknown, ActivationStale, ActivationAccessDenied:
	default:
		evidence.State = ActivationUnknown
		evidence.Detail = "activation observer returned an unsupported state"
	}
	if evidence.RequiredCheck == "" {
		evidence.RequiredCheck = request.RequiredCheck
	}
	if evidence.AdapterID == "" {
		evidence.AdapterID = request.AdapterID
	}
	if evidence.AdapterID != request.AdapterID {
		evidence.State = ActivationUnknown
		evidence.Freshness = ActivationFreshUnverified
		evidence.Detail = "activation evidence names a different configured adapter"
	}
	if evidence.RequiredCheck != request.RequiredCheck {
		evidence.State = ActivationUnknown
		evidence.Freshness = ActivationFreshUnverified
		evidence.Detail = "activation evidence names a different required check"
	}
	if evidence.State == ActivationActive {
		if evidence.ObservedAt.IsZero() || evidence.Freshness != ActivationFreshCurrent ||
			strings.TrimSpace(evidence.Provider) == "" || strings.TrimSpace(evidence.Boundary) == "" ||
			strings.TrimSpace(evidence.ObservedTarget) == "" || strings.TrimSpace(evidence.EvidenceSource) == "" {
			evidence.State = ActivationStale
			if evidence.Freshness == "" {
				evidence.Freshness = ActivationFreshUnverified
			}
			evidence.Detail = "active claim lacks current independent target evidence"
		}
	}
	return evidence
}
