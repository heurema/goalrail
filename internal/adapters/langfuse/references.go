// Package langfuse translates verified provider observation identities into
// provider-neutral evidence references. It performs no network or credential
// access and does not own execution lineage.
package langfuse

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/heurema/goalrail/internal/domain"
)

const (
	sessionReferenceScheme = "langfuse-session:"
	traceReferenceScheme   = "langfuse-trace:"
)

var (
	ErrUnverifiedLineage          = errors.New("Langfuse references require verified execution lineage")
	ErrInvalidObservationIdentity = errors.New("invalid Langfuse observation identity")
	ErrSessionMismatch            = errors.New("Langfuse observation session does not match execution lineage")

	sha256Pattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	traceIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)
)

// ObservationIdentity is the bounded identity subset returned by a Langfuse
// observation lookup. Raw observation input, output, metadata, and credentials
// do not cross this adapter boundary.
type ObservationIdentity struct {
	TraceID   string
	SessionID domain.SessionID
}

// ObservationReferences separates a verified session lookup key from trace
// evidence that was actually observed. A session lookup alone never asserts
// that Langfuse contains a trace for the run.
type ObservationReferences struct {
	sessionLookup domain.EvidenceReference
	traceRefs     []domain.EvidenceReference
}

// BuildObservationReferences constructs deterministic evidence references for
// one verified root session. An empty observations slice is valid so missing or
// delayed Langfuse evidence never blocks the Goalrail harness.
func BuildObservationReferences(
	lineage domain.ExecutionLineage,
	observations []ObservationIdentity,
) (ObservationReferences, error) {
	if err := validateVerifiedLineage(lineage); err != nil {
		return ObservationReferences{}, err
	}

	result := ObservationReferences{
		sessionLookup: domain.EvidenceReference(sessionReferenceScheme + string(lineage.RootSessionID)),
	}
	seenTraceIDs := make(map[string]struct{}, len(observations))
	for index, observation := range observations {
		if observation.SessionID != lineage.RootSessionID {
			return ObservationReferences{}, fmt.Errorf(
				"%w: observation %d has session %q, want %q",
				ErrSessionMismatch,
				index,
				observation.SessionID,
				lineage.RootSessionID,
			)
		}
		if !traceIDPattern.MatchString(observation.TraceID) {
			return ObservationReferences{}, fmt.Errorf(
				"%w: observation %d trace ID must be 32 hexadecimal characters",
				ErrInvalidObservationIdentity,
				index,
			)
		}

		traceID := strings.ToLower(observation.TraceID)
		if _, duplicate := seenTraceIDs[traceID]; duplicate {
			continue
		}
		seenTraceIDs[traceID] = struct{}{}
		result.traceRefs = append(
			result.traceRefs,
			domain.EvidenceReference(traceReferenceScheme+traceID),
		)
	}
	sort.Slice(result.traceRefs, func(i, j int) bool {
		return result.traceRefs[i] < result.traceRefs[j]
	})
	return result, nil
}

// SessionLookup returns the verified join key used to find traces for the root
// Codex session. It is not proof that matching observations exist.
func (references ObservationReferences) SessionLookup() domain.EvidenceReference {
	return references.sessionLookup
}

// TraceReferences returns a copy of the verified trace references.
func (references ObservationReferences) TraceReferences() []domain.EvidenceReference {
	return append([]domain.EvidenceReference(nil), references.traceRefs...)
}

// EvidenceReferences returns the session lookup followed by sorted trace
// references, ready to attach to a provider-neutral evidence event.
func (references ObservationReferences) EvidenceReferences() []domain.EvidenceReference {
	if references.sessionLookup == "" {
		return nil
	}
	result := make([]domain.EvidenceReference, 0, len(references.traceRefs)+1)
	result = append(result, references.sessionLookup)
	result = append(result, references.traceRefs...)
	return result
}

// HasTraceEvidence reports whether at least one exact trace identity was
// observed. The session lookup remains usable when this is false.
func (references ObservationReferences) HasTraceEvidence() bool {
	return len(references.traceRefs) > 0
}

func validateVerifiedLineage(lineage domain.ExecutionLineage) error {
	if lineage.Status != domain.LineageVerified ||
		!domain.IsCanonicalID(string(lineage.ChangeID)) ||
		!domain.IsCanonicalID(string(lineage.RunID)) ||
		!domain.IsCanonicalID(string(lineage.RootSessionID)) ||
		!sha256Pattern.MatchString(lineage.ContextDigest) ||
		(lineage.IdentitySource != domain.SessionIdentityLifecycleHook &&
			lineage.IdentitySource != domain.SessionIdentityLaunchReceipt) ||
		lineage.UnlinkedReasonCode != "" {
		return ErrUnverifiedLineage
	}
	return nil
}
