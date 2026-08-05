package domain

import (
	"bytes"
	"testing"
	"time"
)

func TestLineageEventDigestIsOrderIndependentAndGolden(t *testing.T) {
	left := validLineageEvent()
	right := validLineageEvent()
	right.Sources[0], right.Sources[1] = right.Sources[1], right.Sources[0]
	right.Targets[0], right.Targets[1] = right.Targets[1], right.Targets[0]

	leftFrozen, err := FreezeLineageEvent(left)
	if err != nil {
		t.Fatal(err)
	}
	rightFrozen, err := FreezeLineageEvent(right)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(leftFrozen.CanonicalJSON(), rightFrozen.CanonicalJSON()) || leftFrozen.Digest() != rightFrozen.Digest() {
		t.Fatal("equivalent event sets must produce byte-identical lineage events")
	}
	const wantSemanticDigest = "sha256:80c4ed20bd873b81cd43e0d6b25b6ed3139b4625fa757c397b56cb5a3be83171"
	semanticDigest, err := LineageEventSemanticDigest(left)
	if err != nil {
		t.Fatal(err)
	}
	if semanticDigest != wantSemanticDigest {
		t.Fatalf("lineage event semantic digest changed: got %s, want %s", semanticDigest, wantSemanticDigest)
	}
}

func TestLineageRejectsConflictingCardinalityAndRawContent(t *testing.T) {
	event := validLineageEvent()
	event.Cardinality = RelationSingle
	if _, err := FreezeLineageEvent(event); err == nil {
		t.Fatal("single-valued event with two targets was accepted")
	}
	event = validLineageEvent()
	event.ActorRef = "raw transcript copied here"
	if _, err := FreezeLineageEvent(event); err == nil {
		t.Fatal("raw content was accepted as lineage actor evidence")
	}
	event = validLineageEvent()
	event.SemanticDigest = digestWith('f')
	if _, err := FreezeLineageEvent(event); err == nil {
		t.Fatal("mismatched semantic digest was accepted")
	}
}

func validWorkUnit() WorkUnit {
	return WorkUnit{
		Schema:            WorkUnitSchemaV1,
		ID:                "wu_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ProjectID:         "prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		DeclarationDigest: digestWith('2'),
		PolicyDigest:      digestWith('a'),
		IntentRef:         evidenceReference("intent", "intent:project-lineage-v1", 'b'),
		ChangeRef:         evidenceReference("change", "change:project-lineage-admission-v0", 'c'),
		CreatedAt:         time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC),
		Lifecycle:         WorkUnitOpen,
		RequiredRelations: []LineageRequirement{
			{Relation: LineageTerminalReceipt, Cardinality: RelationSingle},
			{Relation: LineageCommit, Cardinality: RelationSet},
		},
	}
}

func validLineageEvent() LineageEvent {
	return LineageEvent{
		Schema:      LineageEventSchemaV1,
		WorkUnitID:  "wu_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Relation:    LineageCommit,
		Cardinality: RelationSet,
		Sources: []ContentAddressedEvidenceReference{
			evidenceReference("work_unit", "work-unit:wu_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", '1'),
			evidenceReference("run_receipt", "receipt:run-1", '2'),
		},
		Targets: []ContentAddressedEvidenceReference{
			evidenceReference("commit", "git:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", '3'),
			evidenceReference("commit", "git:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", '4'),
		},
		ActorRef:   "user:owner-a",
		AdapterID:  "git-local",
		ObservedAt: time.Date(2026, 8, 4, 10, 30, 0, 0, time.UTC),
	}
}

func evidenceReference(kind, identity string, digestCharacter byte) ContentAddressedEvidenceReference {
	return ContentAddressedEvidenceReference{
		ArtifactKind: kind,
		Identity:     identity,
		Version:      "1",
		Digest:       digestWith(digestCharacter),
		SourceRef:    "adapter:fixture-v1",
		AdapterID:    "fixture",
	}
}
