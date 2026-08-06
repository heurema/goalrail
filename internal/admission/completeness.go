package admission

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/heurema/goalrail/internal/domain"
)

type Completeness struct {
	Lifecycle         domain.WorkUnitLifecycleState
	Missing           []domain.LineageRelation
	Ambiguous         []domain.LineageRelation
	Unavailable       []domain.LineageRelation
	ExpiredExceptions []domain.SHA256Digest
}

// EvaluateCompleteness resolves every required relation through the ephemeral
// effective view: committed relations, semantic projections of the exact
// committed artifacts, and authenticated same-invocation provider candidates.
func EvaluateCompleteness(graph WorkUnitGraph, evaluatedAt time.Time, view EffectiveView) (Completeness, error) {
	if evaluatedAt.IsZero() {
		return Completeness{}, fmt.Errorf("completeness evaluation time is required")
	}
	evaluatedAt = evaluatedAt.UTC()
	result := Completeness{
		Lifecycle: domain.WorkUnitOpen, Missing: []domain.LineageRelation{},
		Ambiguous: []domain.LineageRelation{}, Unavailable: append([]domain.LineageRelation(nil), graph.Unavailable...),
		ExpiredExceptions: []domain.SHA256Digest{},
	}
	present := make(map[domain.LineageRelation]bool)
	unavailable := make(map[domain.LineageRelation]bool)
	for _, relation := range graph.Unavailable {
		unavailable[relation] = true
	}
	for _, conflict := range graph.Conflicts {
		result.Ambiguous = append(result.Ambiguous, conflict.Relation)
	}
	for _, event := range graph.Events {
		verifiedTargets := 0
		for _, target := range event.Targets {
			if target.ArtifactKind == "provider_unavailable" {
				continue
			}
			verifiedTargets++
			if event.Relation == domain.LineageException {
				expired, err := exceptionExpired(graph.Replicas[target.Digest], evaluatedAt)
				if err != nil {
					return Completeness{}, err
				}
				if expired {
					result.ExpiredExceptions = append(result.ExpiredExceptions, target.Digest)
				}
			}
		}
		if verifiedTargets > 0 && !unavailable[event.Relation] {
			present[event.Relation] = true
		}
	}
	conflicting := make(map[domain.LineageRelation]bool)
	for _, relation := range result.Ambiguous {
		conflicting[relation] = true
	}
	var admissionMissing, closedMissing []domain.LineageRelation
	for _, requirement := range graph.Unit.RequiredRelations {
		effective := view.present(requirement.Relation, present[requirement.Relation], unavailable[requirement.Relation])
		if requirement.Relation != domain.LineageClosure && !effective {
			admissionMissing = append(admissionMissing, requirement.Relation)
		}
		if !effective {
			closedMissing = append(closedMissing, requirement.Relation)
		}
	}
	// A relation an authenticated candidate satisfies is not unavailable, and
	// leaving it in the reported set denies on it later: Verify refuses whenever
	// that slice is non-empty, so live evidence could never repair an explicitly
	// unavailable relation.
	if len(result.Unavailable) > 0 {
		retained := make([]domain.LineageRelation, 0, len(result.Unavailable))
		for _, relation := range result.Unavailable {
			if !view.Satisfied(relation) {
				retained = append(retained, relation)
			}
		}
		result.Unavailable = retained
	}
	sortRelations(admissionMissing)
	sortRelations(closedMissing)
	sortRelations(result.Ambiguous)
	sortRelations(result.Unavailable)
	sort.Slice(result.ExpiredExceptions, func(i, j int) bool { return result.ExpiredExceptions[i] < result.ExpiredExceptions[j] })
	blockedAdmission := len(result.ExpiredExceptions) > 0
	blockedClosed := blockedAdmission
	for relation := range conflicting {
		if requiredRelation(graph.Unit.RequiredRelations, relation) {
			blockedClosed = true
			if relation != domain.LineageClosure {
				blockedAdmission = true
			}
		}
	}
	for _, relation := range result.Unavailable {
		if requiredRelation(graph.Unit.RequiredRelations, relation) {
			blockedClosed = true
			if relation != domain.LineageClosure {
				blockedAdmission = true
			}
		}
	}
	if len(closedMissing) == 0 && !blockedClosed {
		result.Lifecycle = domain.WorkUnitClosed
		result.Missing = []domain.LineageRelation{}
	} else if len(admissionMissing) == 0 && !blockedAdmission {
		result.Lifecycle = domain.WorkUnitAdmissionReady
		result.Missing = closedMissing
	} else {
		result.Missing = admissionMissing
	}
	return result, nil
}

func exceptionExpired(raw []byte, evaluatedAt time.Time) (bool, error) {
	if len(raw) == 0 {
		return false, nil
	}
	var envelope struct {
		Schema    string    `json:"schema"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return false, fmt.Errorf("decode lineage exception replica: %w", err)
	}
	if envelope.Schema != LineageExceptionSchemaV1 || envelope.ExpiresAt.IsZero() {
		return false, fmt.Errorf("lineage exception replica is invalid")
	}
	return !evaluatedAt.Before(envelope.ExpiresAt.UTC()), nil
}

func requiredRelation(requirements []domain.LineageRequirement, relation domain.LineageRelation) bool {
	for _, requirement := range requirements {
		if requirement.Relation == relation {
			return true
		}
	}
	return false
}

func sortRelations(values []domain.LineageRelation) {
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
}
