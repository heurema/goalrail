package lineage

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/heurema/goalrail/internal/domain"
)

const LineageExceptionSchemaV1 = "goalrail.lineage-exception/v1"

type Completeness struct {
	Lifecycle         domain.WorkUnitLifecycleState `json:"lifecycle"`
	Missing           []domain.LineageRelation      `json:"missing"`
	Ambiguous         []domain.LineageRelation      `json:"ambiguous"`
	Unavailable       []domain.LineageRelation      `json:"unavailable"`
	ExplicitEmpty     []domain.LineageRelation      `json:"explicit_empty"`
	ExpiredExceptions []domain.SHA256Digest         `json:"expired_exceptions"`
}

type lineageExceptionEnvelope struct {
	Schema    string    `json:"schema"`
	ExpiresAt time.Time `json:"expires_at"`
}

func EvaluateCompleteness(graph WorkUnitGraph, evaluatedAt time.Time) (Completeness, error) {
	if evaluatedAt.IsZero() {
		return Completeness{}, fmt.Errorf("completeness evaluation time is required")
	}
	evaluatedAt = evaluatedAt.UTC()
	result := Completeness{
		Lifecycle: domain.WorkUnitOpen,
		Missing:   []domain.LineageRelation{}, Ambiguous: []domain.LineageRelation{},
		Unavailable:   append([]domain.LineageRelation(nil), graph.Unavailable...),
		ExplicitEmpty: []domain.LineageRelation{}, ExpiredExceptions: []domain.SHA256Digest{},
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
			switch target.ArtifactKind {
			case "provider_unavailable":
				continue
			case "empty_set":
				result.ExplicitEmpty = appendUniqueRelation(result.ExplicitEmpty, event.Relation)
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
		if requirement.Relation != domain.LineageClosure && (!present[requirement.Relation] || unavailable[requirement.Relation]) {
			admissionMissing = append(admissionMissing, requirement.Relation)
		}
		if !present[requirement.Relation] || unavailable[requirement.Relation] {
			closedMissing = append(closedMissing, requirement.Relation)
		}
	}
	sortRelations(admissionMissing)
	sortRelations(closedMissing)
	sortRelations(result.Ambiguous)
	sortRelations(result.Unavailable)
	sortRelations(result.ExplicitEmpty)
	sort.Slice(result.ExpiredExceptions, func(first, second int) bool {
		return result.ExpiredExceptions[first] < result.ExpiredExceptions[second]
	})
	hasAdmissionConflict := false
	hasClosedConflict := false
	for relation := range conflicting {
		for _, requirement := range graph.Unit.RequiredRelations {
			if requirement.Relation == relation {
				hasClosedConflict = true
				if relation != domain.LineageClosure {
					hasAdmissionConflict = true
				}
			}
		}
	}
	hasAdmissionUnavailable := false
	hasClosedUnavailable := false
	for _, relation := range result.Unavailable {
		for _, requirement := range graph.Unit.RequiredRelations {
			if requirement.Relation == relation {
				hasClosedUnavailable = true
				if relation != domain.LineageClosure {
					hasAdmissionUnavailable = true
				}
			}
		}
	}
	blockedAdmission := hasAdmissionConflict || hasAdmissionUnavailable || len(result.ExpiredExceptions) > 0
	blockedClosed := hasClosedConflict || hasClosedUnavailable || len(result.ExpiredExceptions) > 0
	if len(closedMissing) == 0 && !blockedClosed {
		result.Lifecycle = domain.WorkUnitClosed
		result.Missing = []domain.LineageRelation{}
	} else if len(admissionMissing) == 0 && !blockedAdmission {
		result.Lifecycle = domain.WorkUnitAdmissionReady
		result.Missing = closedMissing
	} else {
		result.Lifecycle = domain.WorkUnitOpen
		result.Missing = admissionMissing
	}
	return result, nil
}

func exceptionExpired(raw []byte, evaluatedAt time.Time) (bool, error) {
	if len(raw) == 0 {
		return false, nil
	}
	var envelope lineageExceptionEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return false, fmt.Errorf("decode lineage exception replica: %w", err)
	}
	if envelope.Schema != LineageExceptionSchemaV1 {
		return false, fmt.Errorf("lineage exception replica has schema %q", envelope.Schema)
	}
	if envelope.ExpiresAt.IsZero() {
		return false, fmt.Errorf("lineage exception replica omits expires_at")
	}
	return !evaluatedAt.Before(envelope.ExpiresAt.UTC()), nil
}

func appendUniqueRelation(values []domain.LineageRelation, value domain.LineageRelation) []domain.LineageRelation {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func sortRelations(values []domain.LineageRelation) {
	sort.Slice(values, func(first, second int) bool { return values[first] < values[second] })
}
