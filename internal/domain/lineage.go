package domain

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

type (
	WorkUnitID             string
	LineageRelation        string
	RelationCardinality    string
	WorkUnitLifecycleState string
)

const (
	WorkUnitSchemaV1     = "goalrail.work-unit/v1"
	LineageEventSchemaV1 = "goalrail.lineage-event/v1"
	MaxWorkUnitBytes     = 128 << 10
	MaxLineageEventBytes = 128 << 10
	MaxLineageReferences = 256
	MaxLineageRelations  = 128

	RelationSingle RelationCardinality = "single"
	RelationSet    RelationCardinality = "set"

	WorkUnitOpen           WorkUnitLifecycleState = "open"
	WorkUnitAdmissionReady WorkUnitLifecycleState = "admission_ready"
	WorkUnitClosed         WorkUnitLifecycleState = "closed"

	LineageProjectPolicy   LineageRelation = "project_policy"
	LineageConfirmedIntent LineageRelation = "confirmed_intent"
	LineageChange          LineageRelation = "change"
	LineageWorkSpec        LineageRelation = "work_spec"
	LineageRunSession      LineageRelation = "run_session"
	LineageCommit          LineageRelation = "commit"
	LineagePullRequest     LineageRelation = "pull_request"
	LineageReviewIndex     LineageRelation = "review_index"
	LineageCheckSet        LineageRelation = "check_set"
	LineageTerminalReceipt LineageRelation = "terminal_receipt"
	LineageOwnerDecision   LineageRelation = "owner_decision"
	LineageException       LineageRelation = "exception"
	LineageClosure         LineageRelation = "closure"
)

// ContentAddressedEvidenceReference points at exact bounded evidence without
// copying its semantic body into lineage.
type ContentAddressedEvidenceReference struct {
	ArtifactKind string       `json:"artifact_kind"`
	Identity     string       `json:"identity"`
	Version      string       `json:"version"`
	Digest       SHA256Digest `json:"digest"`
	SourceRef    string       `json:"source_ref"`
	AdapterID    string       `json:"adapter_id"`
}

type LineageRequirement struct {
	Relation    LineageRelation     `json:"relation"`
	Cardinality RelationCardinality `json:"cardinality"`
}

type WorkUnit struct {
	Schema            string                            `json:"schema"`
	ID                WorkUnitID                        `json:"id"`
	ProjectID         ProjectID                         `json:"project_id"`
	DeclarationDigest SHA256Digest                      `json:"declaration_digest"`
	PolicyDigest      SHA256Digest                      `json:"policy_digest"`
	IntentRef         ContentAddressedEvidenceReference `json:"intent_ref"`
	ChangeRef         ContentAddressedEvidenceReference `json:"change_ref"`
	CreatedAt         time.Time                         `json:"created_at"`
	Lifecycle         WorkUnitLifecycleState            `json:"lifecycle"`
	RequiredRelations []LineageRequirement              `json:"required_relations"`
}

// LineageEvent is an immutable backward-reference event. SemanticDigest
// identifies the normalized relation payload without hashing itself; the
// enclosing CanonicalArtifact digest identifies the complete stored bytes.
type LineageEvent struct {
	Schema         string                              `json:"schema"`
	WorkUnitID     WorkUnitID                          `json:"work_unit_id"`
	Relation       LineageRelation                     `json:"relation"`
	Cardinality    RelationCardinality                 `json:"cardinality"`
	Sources        []ContentAddressedEvidenceReference `json:"sources"`
	Targets        []ContentAddressedEvidenceReference `json:"targets"`
	ActorRef       string                              `json:"actor_ref"`
	AdapterID      string                              `json:"adapter_id"`
	ObservedAt     time.Time                           `json:"observed_at"`
	SemanticDigest SHA256Digest                        `json:"semantic_digest"`
}

type lineageEventSemanticPayload struct {
	Schema      string                              `json:"schema"`
	WorkUnitID  WorkUnitID                          `json:"work_unit_id"`
	Relation    LineageRelation                     `json:"relation"`
	Cardinality RelationCardinality                 `json:"cardinality"`
	Sources     []ContentAddressedEvidenceReference `json:"sources"`
	Targets     []ContentAddressedEvidenceReference `json:"targets"`
	ActorRef    string                              `json:"actor_ref"`
	AdapterID   string                              `json:"adapter_id"`
	ObservedAt  time.Time                           `json:"observed_at"`
}

func DecodeWorkUnit(reader io.Reader) (WorkUnit, error) {
	value, err := decodeStrictBoundedJSON[WorkUnit](reader, MaxWorkUnitBytes, "work unit")
	if err != nil {
		return WorkUnit{}, err
	}
	value = normalizeWorkUnit(value)
	if err := ValidateWorkUnit(value); err != nil {
		return WorkUnit{}, err
	}
	return value, nil
}

func FreezeWorkUnit(value WorkUnit) (CanonicalArtifact, error) {
	value = normalizeWorkUnit(value)
	if err := ValidateWorkUnit(value); err != nil {
		return CanonicalArtifact{}, err
	}
	return newCanonicalArtifact(value)
}

func DecodeLineageEvent(reader io.Reader) (LineageEvent, error) {
	value, err := decodeStrictBoundedJSON[LineageEvent](reader, MaxLineageEventBytes, "lineage event")
	if err != nil {
		return LineageEvent{}, err
	}
	value = normalizeLineageEvent(value)
	if err := ValidateLineageEvent(value); err != nil {
		return LineageEvent{}, err
	}
	return value, nil
}

func FreezeLineageEvent(value LineageEvent) (CanonicalArtifact, error) {
	value = normalizeLineageEvent(value)
	semanticDigest, err := LineageEventSemanticDigest(value)
	if err != nil {
		return CanonicalArtifact{}, err
	}
	if value.SemanticDigest != "" && value.SemanticDigest != semanticDigest {
		return CanonicalArtifact{}, fmt.Errorf("lineage event semantic digest mismatch: expected %s, got %s", semanticDigest, value.SemanticDigest)
	}
	value.SemanticDigest = semanticDigest
	if err := ValidateLineageEvent(value); err != nil {
		return CanonicalArtifact{}, err
	}
	return newCanonicalArtifact(value)
}

// LineageEventSemanticDigest identifies the normalized relation payload with
// the digest field excluded. This non-self-referential identity is retained in
// SemanticDigest; the enclosing CanonicalArtifact digest still identifies the
// complete stored JSON bytes.
func LineageEventSemanticDigest(value LineageEvent) (SHA256Digest, error) {
	value = normalizeLineageEvent(value)
	payload := lineageEventSemanticPayload{
		Schema:      value.Schema,
		WorkUnitID:  value.WorkUnitID,
		Relation:    value.Relation,
		Cardinality: value.Cardinality,
		Sources:     value.Sources,
		Targets:     value.Targets,
		ActorRef:    value.ActorRef,
		AdapterID:   value.AdapterID,
		ObservedAt:  value.ObservedAt,
	}
	canonical, err := newCanonicalArtifact(payload)
	if err != nil {
		return "", fmt.Errorf("encode lineage event semantic payload: %w", err)
	}
	return canonical.Digest(), nil
}

func ValidateWorkUnit(value WorkUnit) error {
	v := &validator{}
	if value.Schema != WorkUnitSchemaV1 {
		v.add("work_unit.schema.invalid", "schema", "unsupported work-unit schema")
	}
	if !IsCanonicalID(string(value.ID)) || !strings.HasPrefix(string(value.ID), "wu_") {
		v.add("work_unit.id.invalid", "id", "work-unit ID must be canonical")
	}
	if !IsCanonicalID(string(value.ProjectID)) || !strings.HasPrefix(string(value.ProjectID), "prj_") {
		v.add("work_unit.project_id.invalid", "project_id", "project ID must be canonical")
	}
	validateDigestField(v, "work_unit.declaration_digest_invalid", "declaration_digest", value.DeclarationDigest)
	validateDigestField(v, "work_unit.policy_digest_invalid", "policy_digest", value.PolicyDigest)
	validateContentAddressedReference(v, "intent_ref", value.IntentRef)
	validateContentAddressedReference(v, "change_ref", value.ChangeRef)
	if value.CreatedAt.IsZero() {
		v.add("work_unit.created_at.required", "created_at", "work-unit creation time is required")
	}
	switch value.Lifecycle {
	case WorkUnitOpen, WorkUnitAdmissionReady, WorkUnitClosed:
	default:
		v.add("work_unit.lifecycle.invalid", "lifecycle", "unsupported work-unit lifecycle state")
	}
	if len(value.RequiredRelations) == 0 {
		v.add("work_unit.required_relations.required", "required_relations", "work unit must declare its required relations")
	}
	if len(value.RequiredRelations) > MaxLineageRelations {
		v.add("work_unit.required_relations.too_many", "required_relations", "required relation count exceeds the v1 bound")
	}
	seen := make(map[LineageRelation]struct{}, len(value.RequiredRelations))
	for index, requirement := range value.RequiredRelations {
		path := fmt.Sprintf("required_relations[%d]", index)
		validateLineageRelation(v, path+".relation", requirement.Relation)
		validateRelationCardinality(v, path+".cardinality", requirement.Cardinality)
		if _, exists := seen[requirement.Relation]; exists {
			v.add("work_unit.required_relation.duplicate", path+".relation", "required relations must be unique")
		}
		seen[requirement.Relation] = struct{}{}
	}
	validateNoSecretPayload(v, value)
	return v.result()
}

func ValidateLineageEvent(value LineageEvent) error {
	v := &validator{}
	if value.Schema != LineageEventSchemaV1 {
		v.add("lineage_event.schema.invalid", "schema", "unsupported lineage-event schema")
	}
	if !IsCanonicalID(string(value.WorkUnitID)) || !strings.HasPrefix(string(value.WorkUnitID), "wu_") {
		v.add("lineage_event.work_unit_id.invalid", "work_unit_id", "work-unit ID must be canonical")
	}
	validateLineageRelation(v, "relation", value.Relation)
	validateRelationCardinality(v, "cardinality", value.Cardinality)
	validateContentAddressedReferenceSet(v, "sources", value.Sources, true)
	validateContentAddressedReferenceSet(v, "targets", value.Targets, true)
	if value.Cardinality == RelationSingle && len(value.Targets) != 1 {
		v.add("lineage_event.single_target.invalid", "targets", "single-valued relation requires exactly one target")
	}
	if !IsEvidenceReference(value.ActorRef) {
		v.add("lineage_event.actor_ref.invalid", "actor_ref", "actor reference must be bounded, provider-neutral, and non-secret")
	}
	if !IsCanonicalID(value.AdapterID) {
		v.add("lineage_event.adapter_id.invalid", "adapter_id", "adapter ID must be canonical")
	}
	if value.ObservedAt.IsZero() {
		v.add("lineage_event.observed_at.required", "observed_at", "observation time is required")
	}
	expectedDigest, err := LineageEventSemanticDigest(value)
	if err != nil {
		v.add("lineage_event.semantic_digest.invalid", "semantic_digest", err.Error())
	} else if value.SemanticDigest != expectedDigest {
		v.add("lineage_event.semantic_digest.mismatch", "semantic_digest", "semantic digest must identify the normalized relation payload")
	}
	validateNoSecretPayload(v, value)
	return v.result()
}

func normalizeWorkUnit(value WorkUnit) WorkUnit {
	value.CreatedAt = value.CreatedAt.UTC()
	value.RequiredRelations = append([]LineageRequirement(nil), value.RequiredRelations...)
	sort.Slice(value.RequiredRelations, func(first, second int) bool {
		if value.RequiredRelations[first].Relation != value.RequiredRelations[second].Relation {
			return value.RequiredRelations[first].Relation < value.RequiredRelations[second].Relation
		}
		return value.RequiredRelations[first].Cardinality < value.RequiredRelations[second].Cardinality
	})
	return value
}

func normalizeLineageEvent(value LineageEvent) LineageEvent {
	value.ObservedAt = value.ObservedAt.UTC()
	value.Sources = normalizeContentAddressedReferences(value.Sources)
	value.Targets = normalizeContentAddressedReferences(value.Targets)
	return value
}

func normalizeContentAddressedReferences(values []ContentAddressedEvidenceReference) []ContentAddressedEvidenceReference {
	normalized := append([]ContentAddressedEvidenceReference(nil), values...)
	sort.Slice(normalized, func(first, second int) bool {
		left, right := normalized[first], normalized[second]
		if left.ArtifactKind != right.ArtifactKind {
			return left.ArtifactKind < right.ArtifactKind
		}
		if left.Identity != right.Identity {
			return left.Identity < right.Identity
		}
		if left.Version != right.Version {
			return left.Version < right.Version
		}
		if left.Digest != right.Digest {
			return left.Digest < right.Digest
		}
		if left.SourceRef != right.SourceRef {
			return left.SourceRef < right.SourceRef
		}
		return left.AdapterID < right.AdapterID
	})
	return normalized
}

func validateContentAddressedReferenceSet(v *validator, path string, values []ContentAddressedEvidenceReference, required bool) {
	if required && len(values) == 0 {
		v.add("lineage.references.required", path, "at least one evidence reference is required")
	}
	if len(values) > MaxLineageReferences {
		v.add("lineage.references.too_many", path, "evidence reference count exceeds the v1 bound")
	}
	seen := make(map[ContentAddressedEvidenceReference]struct{}, len(values))
	for index, value := range values {
		validateContentAddressedReference(v, fmt.Sprintf("%s[%d]", path, index), value)
		if _, exists := seen[value]; exists {
			v.add("lineage.reference.duplicate", path, "evidence references must be unique")
		}
		seen[value] = struct{}{}
	}
}

func validateContentAddressedReference(v *validator, path string, value ContentAddressedEvidenceReference) {
	if !IsCanonicalID(value.ArtifactKind) {
		v.add("lineage.reference.kind_invalid", path+".artifact_kind", "artifact kind must be canonical")
	}
	if !IsEvidenceReference(value.Identity) {
		v.add("lineage.reference.identity_invalid", path+".identity", "artifact identity must be a bounded provider-neutral reference")
	}
	if !IsCanonicalID(value.Version) {
		v.add("lineage.reference.version_invalid", path+".version", "artifact version must be a canonical identifier")
	}
	validateDigestField(v, "lineage.reference.digest_invalid", path+".digest", value.Digest)
	if !IsEvidenceReference(value.SourceRef) {
		v.add("lineage.reference.source_invalid", path+".source_ref", "source reference must be bounded and provider-neutral")
	}
	if !IsCanonicalID(value.AdapterID) {
		v.add("lineage.reference.adapter_invalid", path+".adapter_id", "adapter ID must be canonical")
	}
}

func validateLineageRelation(v *validator, path string, value LineageRelation) {
	switch value {
	case LineageProjectPolicy, LineageConfirmedIntent, LineageChange, LineageWorkSpec,
		LineageRunSession, LineageCommit, LineagePullRequest, LineageReviewIndex,
		LineageCheckSet, LineageTerminalReceipt, LineageOwnerDecision,
		LineageException, LineageClosure:
	default:
		v.add("lineage.relation.invalid", path, "unsupported lineage relation")
	}
}

func validateRelationCardinality(v *validator, path string, value RelationCardinality) {
	if value != RelationSingle && value != RelationSet {
		v.add("lineage.cardinality.invalid", path, "relation cardinality must be single or set")
	}
}
