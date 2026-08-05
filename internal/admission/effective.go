package admission

import (
	"errors"
	"sort"
	"time"

	"github.com/heurema/goalrail/internal/domain"
)

// Semantic codec identities pinned by this verifier version. An unknown codec
// identity fails closed and never produces a satisfying projection.
const (
	IntentSnapshotCodecV1  = "goalrail.intent-snapshot/v1"
	TerminalReceiptCodecV1 = "goalrail.terminal-receipt/v1"
)

const (
	ProjectionKindIntent          = "intent"
	ProjectionKindTerminalReceipt = "terminal_receipt"

	ProjectionStateConfirmed = "confirmed"
	ProjectionStatePassed    = "passed"
	ProjectionStateInvalid   = "invalid"
)

var errEphemeralEvidence = errors.New("effective admission evidence exists only for one evaluation and is never serialized")

// SemanticProjection is the bounded, provider-neutral result of validating one
// exact committed artifact against its own codec. It carries no semantic body
// content, exists only for the current call, and is never persisted, cached, or
// emitted as a lineage event.
type SemanticProjection struct {
	Relation domain.LineageRelation
	Kind     string
	Identity string
	Version  string
	Digest   domain.SHA256Digest
	CodecID  string
	State    string
	Valid    bool
}

// MarshalJSON keeps a projection out of every serialized artifact. A packet or
// result that carried one would turn a point-in-time parse into stored lineage.
func (SemanticProjection) MarshalJSON() ([]byte, error) { return nil, errEphemeralEvidence }

// ProviderCandidate is one bounded in-memory observation produced by a
// registered adapter inside the same trusted invocation. It is never
// reconstructed from a packet at rest, never persisted, and never written back
// into the committed graph.
type ProviderCandidate struct {
	Relation      domain.LineageRelation
	Identity      string
	Digest        domain.SHA256Digest
	AdapterID     string
	ProviderRef   string
	BaseRevision  string
	HeadRevision  string
	ActorRef      string
	AuthorityRef  string
	Outcome       domain.OwnerDecisionOutcome
	ObservedAt    time.Time
	Authenticated bool
}

// MarshalJSON keeps a candidate out of every serialized artifact for the same
// reason a projection stays out: a serialized candidate would be a forgeable
// authority claim.
func (ProviderCandidate) MarshalJSON() ([]byte, error) { return nil, errEphemeralEvidence }

// EffectiveView is the immutable per-call composition of committed relations,
// semantic projections, and authenticated provider candidates. Completeness and
// exception evaluation consume it instead of the committed graph alone.
type EffectiveView struct {
	satisfied  map[domain.LineageRelation]bool
	suppressed map[domain.LineageRelation]bool
}

func (view EffectiveView) Satisfied(relation domain.LineageRelation) bool {
	return view.satisfied[relation]
}

// present resolves one relation for the current evaluation. A provider
// candidate can complete a relation the repository never committed; a failed
// semantic projection removes a relation the repository did commit.
func (view EffectiveView) present(relation domain.LineageRelation, committed, unavailable bool) bool {
	if view.satisfied[relation] {
		return true
	}
	if view.suppressed[relation] {
		return false
	}
	return committed && !unavailable
}

// candidateRelations is the closed mapping from provider artifact kind to
// effective relation. An unknown kind fails closed.
var candidateRelations = map[domain.LineageRelation]struct{}{
	domain.LineagePullRequest:   {},
	domain.LineageReviewIndex:   {},
	domain.LineageCheckSet:      {},
	domain.LineageOwnerDecision: {},
	domain.LineageClosure:       {},
}

// semanticRelations names every relation whose committed target must produce a
// satisfying semantic projection before it counts.
var semanticRelations = []domain.LineageRelation{
	domain.LineageConfirmedIntent,
	domain.LineageTerminalReceipt,
}

func projectionContract(kind string) (domain.LineageRelation, string, string, bool) {
	switch kind {
	case ProjectionKindIntent:
		return domain.LineageConfirmedIntent, IntentSnapshotCodecV1, ProjectionStateConfirmed, true
	case ProjectionKindTerminalReceipt:
		return domain.LineageTerminalReceipt, TerminalReceiptCodecV1, ProjectionStatePassed, true
	default:
		return "", "", "", false
	}
}

// projectionStates is the closed admission-relevant state enum for each codec.
// The verifier keeps its own copy so no provider or run package enters the pure
// core.
var projectionStates = map[string][]string{
	ProjectionKindIntent: {ProjectionStateConfirmed, "candidate", ProjectionStateInvalid},
	ProjectionKindTerminalReceipt: {
		ProjectionStatePassed, "failed", "blocked", "unlinked", "launch_failed",
		"launch_attempted_unknown", "verification_incomplete", "prepared",
		"launch_attempted", "awaiting_verification", ProjectionStateInvalid,
	},
}

// buildEffectiveView composes the per-call evidence view. It returns a stable
// reason and evidence references instead of a view when any input is
// unsupported, unauthenticated, out of range, or conflicting.
func buildEffectiveView(input Input, policy domain.ProjectPolicy) (EffectiveView, domain.AdmissionReasonCode, []string) {
	view := EffectiveView{
		satisfied:  make(map[domain.LineageRelation]bool),
		suppressed: make(map[domain.LineageRelation]bool),
	}
	satisfiedProjections, reason, refs := evaluateProjections(input.Range.Graph, input.Range.Projections)
	if reason != "" {
		return EffectiveView{}, reason, refs
	}
	for _, relation := range semanticRelations {
		if !satisfiedProjections[relation] {
			view.suppressed[relation] = true
		}
	}
	// A committed owner-decision event is ordinary repository evidence. Only an
	// authenticated same-invocation observation can satisfy the owner boundary.
	view.suppressed[domain.LineageOwnerDecision] = true

	satisfiedCandidates, reason, refs := evaluateCandidates(input.Candidates, input.Range, policy, input.Packet)
	if reason != "" {
		return EffectiveView{}, reason, refs
	}
	for relation := range satisfiedCandidates {
		view.satisfied[relation] = true
	}
	return view, "", nil
}

func evaluateProjections(graph WorkUnitGraph, projections []SemanticProjection) (map[domain.LineageRelation]bool, domain.AdmissionReasonCode, []string) {
	satisfied := make(map[domain.LineageRelation]bool)
	claimed := make(map[domain.LineageRelation]SemanticProjection)
	sorted := sortedProjections(projections)
	for _, projection := range sorted {
		relation, codec, satisfyingState, known := projectionContract(projection.Kind)
		if !known || projection.Relation != relation || projection.CodecID != codec ||
			!containsValue(projectionStates[projection.Kind], projection.State) ||
			!domain.IsSHA256Digest(projection.Digest) {
			return nil, domain.ReasonPacketInvalid, []string{projectionRef(projection)}
		}
		if !projectionBindsCommittedTarget(graph, projection) {
			return nil, domain.ReasonPacketInvalid, []string{projectionRef(projection)}
		}
		if previous, exists := claimed[relation]; exists && previous != projection {
			return nil, domain.ReasonLineageConflict, []string{projectionRef(previous), projectionRef(projection)}
		}
		claimed[relation] = projection
		if projection.Valid && projection.State == satisfyingState {
			satisfied[relation] = true
		}
	}
	return satisfied, "", nil
}

// projectionBindsCommittedTarget requires the exact committed target of the
// projected relation. A projection can never satisfy another digest, identity,
// version, artifact kind, or relation.
func projectionBindsCommittedTarget(graph WorkUnitGraph, projection SemanticProjection) bool {
	for _, event := range graph.Events {
		if event.Relation != projection.Relation {
			continue
		}
		for _, target := range event.Targets {
			if target.ArtifactKind == projection.Kind &&
				target.Identity == projection.Identity &&
				target.Version == projection.Version &&
				target.Digest == projection.Digest {
				return true
			}
		}
	}
	return false
}

func evaluateCandidates(candidates []ProviderCandidate, frozen FrozenRange, policy domain.ProjectPolicy, packet domain.AdmissionPacket) (map[domain.LineageRelation]bool, domain.AdmissionReasonCode, []string) {
	satisfied := make(map[domain.LineageRelation]bool)
	if len(candidates) == 0 {
		return satisfied, "", nil
	}
	if !hasBoundedEvaluationTime(packet) {
		return nil, domain.ReasonTrustedTimeMissing, []string{"packet:evaluation-time"}
	}
	evaluationTime := packet.EvaluationTime.UTC()
	claimed := make(map[domain.LineageRelation]ProviderCandidate)
	for _, candidate := range sortedCandidates(candidates) {
		if _, known := candidateRelations[candidate.Relation]; !known ||
			!domain.IsEvidenceReference(candidate.Identity) || candidate.AdapterID == "" ||
			!domain.IsSHA256Digest(candidate.Digest) || !domain.IsEvidenceReference(candidate.ProviderRef) {
			return nil, domain.ReasonPacketInvalid, []string{candidateRef(candidate)}
		}
		if candidate.BaseRevision != frozen.BaseRevision || candidate.HeadRevision != frozen.HeadRevision {
			return nil, domain.ReasonRangeMismatch, []string{candidateRef(candidate)}
		}
		if !candidate.Authenticated || candidate.ObservedAt.IsZero() || candidate.ObservedAt.UTC().After(evaluationTime) {
			return nil, domain.ReasonProvenanceUntrusted, []string{candidateRef(candidate)}
		}
		if previous, exists := claimed[candidate.Relation]; exists && previous != candidate {
			return nil, domain.ReasonLineageConflict, []string{candidateRef(previous), candidateRef(candidate)}
		}
		claimed[candidate.Relation] = candidate
		if candidate.Relation != domain.LineageOwnerDecision {
			satisfied[candidate.Relation] = true
			continue
		}
		if !domain.IsEvidenceReference(candidate.ActorRef) || !domain.IsEvidenceReference(candidate.AuthorityRef) ||
			!supportedOwnerOutcome(policy, candidate.Outcome) {
			return nil, domain.ReasonPacketInvalid, []string{candidateRef(candidate)}
		}
		// An unauthorized authority or a rejection is retained as observed
		// negative evidence; neither satisfies the owner boundary.
		if contains(policy.OwnerDecision.AuthorityRefs, candidate.AuthorityRef) && candidate.Outcome == domain.OwnerDecisionAllow {
			satisfied[domain.LineageOwnerDecision] = true
		}
	}
	return satisfied, "", nil
}

func supportedOwnerOutcome(policy domain.ProjectPolicy, outcome domain.OwnerDecisionOutcome) bool {
	for _, supported := range policy.OwnerDecision.Outcomes {
		if supported == outcome {
			return true
		}
	}
	return false
}

// sortedCandidates makes evaluation independent of adapter arrival order.
func sortedCandidates(values []ProviderCandidate) []ProviderCandidate {
	result := append([]ProviderCandidate(nil), values...)
	sort.Slice(result, func(i, j int) bool {
		left, right := result[i], result[j]
		if left.Relation != right.Relation {
			return left.Relation < right.Relation
		}
		if left.Identity != right.Identity {
			return left.Identity < right.Identity
		}
		if left.Digest != right.Digest {
			return left.Digest < right.Digest
		}
		if left.ActorRef != right.ActorRef {
			return left.ActorRef < right.ActorRef
		}
		return left.ObservedAt.UTC().Before(right.ObservedAt.UTC())
	})
	return result
}

func sortedProjections(values []SemanticProjection) []SemanticProjection {
	result := append([]SemanticProjection(nil), values...)
	sort.Slice(result, func(i, j int) bool {
		left, right := result[i], result[j]
		if left.Relation != right.Relation {
			return left.Relation < right.Relation
		}
		if left.Identity != right.Identity {
			return left.Identity < right.Identity
		}
		return left.Digest < right.Digest
	})
	return result
}

func candidateRef(candidate ProviderCandidate) string {
	if candidate.Identity != "" {
		return candidate.Identity
	}
	return "candidate:" + string(candidate.Relation)
}

func projectionRef(projection SemanticProjection) string {
	if projection.Identity != "" {
		return projection.Identity
	}
	return "projection:" + projection.Kind
}

func containsValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
