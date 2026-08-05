package admission

import "github.com/heurema/goalrail/internal/domain"

const LineageExceptionSchemaV1 = "goalrail.lineage-exception/v1"

type RelationConflict struct {
	Relation domain.LineageRelation `json:"relation"`
	Digests  []domain.SHA256Digest  `json:"event_digests"`
}

// WorkUnitGraph is frozen verifier input. Collection and mutable storage live
// outside the core package.
type WorkUnitGraph struct {
	Unit        domain.WorkUnit                `json:"unit"`
	Events      []domain.LineageEvent          `json:"events"`
	Conflicts   []RelationConflict             `json:"conflicts"`
	Unavailable []domain.LineageRelation       `json:"unavailable"`
	Replicas    map[domain.SHA256Digest][]byte `json:"-"`
}

func RequiredSpine() []domain.LineageRequirement {
	return []domain.LineageRequirement{
		{Relation: domain.LineageProjectPolicy, Cardinality: domain.RelationSet},
		{Relation: domain.LineageConfirmedIntent, Cardinality: domain.RelationSingle},
		{Relation: domain.LineageChange, Cardinality: domain.RelationSingle},
		{Relation: domain.LineageWorkSpec, Cardinality: domain.RelationSingle},
		{Relation: domain.LineageRunSession, Cardinality: domain.RelationSingle},
		{Relation: domain.LineageCommit, Cardinality: domain.RelationSet},
		{Relation: domain.LineagePullRequest, Cardinality: domain.RelationSingle},
		{Relation: domain.LineageReviewIndex, Cardinality: domain.RelationSingle},
		{Relation: domain.LineageCheckSet, Cardinality: domain.RelationSingle},
		{Relation: domain.LineageTerminalReceipt, Cardinality: domain.RelationSingle},
		{Relation: domain.LineageOwnerDecision, Cardinality: domain.RelationSingle},
		{Relation: domain.LineageClosure, Cardinality: domain.RelationSingle},
	}
}
