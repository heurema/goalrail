package domain

import (
	"bytes"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestAdmissionClassificationAndReasonRegistryAreCompleteAndStable(t *testing.T) {
	classifications := []AdmissionClassification{
		AdmissionValid,
		AdmissionMissing,
		AdmissionAmbiguous,
		AdmissionInvalid,
		AdmissionExempted,
		AdmissionBreakGlass,
		AdmissionBootstrap,
	}
	want := []AdmissionClassification{"VALID", "MISSING", "AMBIGUOUS", "INVALID", "EXEMPTED", "BREAK_GLASS", "BOOTSTRAP"}
	if !reflect.DeepEqual(classifications, want) {
		t.Fatalf("classification registry changed: %#v", classifications)
	}
	reasons := KnownAdmissionReasonCodes()
	if !sort.SliceIsSorted(reasons, func(first, second int) bool { return reasons[first] < reasons[second] }) {
		t.Fatal("reason-code registry must remain sorted for deterministic lookup")
	}
	for _, reason := range reasons {
		if !IsAdmissionReasonCode(reason) {
			t.Fatalf("registered reason is not discoverable: %s", reason)
		}
	}
	if IsAdmissionReasonCode("UNREGISTERED") {
		t.Fatal("unknown reason was accepted")
	}
}

func TestAdmissionResultEnforcesAllowDenyBoundary(t *testing.T) {
	for _, classification := range []AdmissionClassification{AdmissionMissing, AdmissionAmbiguous, AdmissionInvalid} {
		result := validAdmissionResult()
		result.Classification = classification
		result.Outcome = AdmissionAllow
		result.Reasons = []AdmissionReason{{Code: ReasonPacketInvalid}}
		if classification == AdmissionMissing {
			result.MissingRefs = []string{"receipt:missing"}
		}
		if classification == AdmissionAmbiguous {
			result.ConflictRefs = []string{"commit:first", "commit:second"}
		}
		if _, err := FreezeAdmissionResult(result); err == nil {
			t.Fatalf("%s classification was allowed", classification)
		}
	}
}

func TestAdmissionContractsCanonicalizeEvidenceOrder(t *testing.T) {
	left := validAdmissionPacket()
	right := validAdmissionPacket()
	right.Evidence[0], right.Evidence[1] = right.Evidence[1], right.Evidence[0]
	right.Provenance[0], right.Provenance[1] = right.Provenance[1], right.Provenance[0]
	leftFrozen, err := FreezeAdmissionPacket(left)
	if err != nil {
		t.Fatal(err)
	}
	rightFrozen, err := FreezeAdmissionPacket(right)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(leftFrozen.CanonicalJSON(), rightFrozen.CanonicalJSON()) {
		t.Fatal("equivalent admission packets must be byte-identical")
	}
}

func TestAdmissionPacketCanonicalizesEveryProvenanceField(t *testing.T) {
	left := validAdmissionPacket()
	base := left.Provenance[0]
	left.Provenance = []AdmissionProviderProvenance{
		{AdapterID: base.AdapterID, ProviderRef: base.ProviderRef, EvidenceDigest: base.EvidenceDigest, ObservedAt: base.ObservedAt.Add(time.Minute), Authenticated: false},
		{AdapterID: base.AdapterID, ProviderRef: base.ProviderRef, EvidenceDigest: base.EvidenceDigest, ObservedAt: base.ObservedAt, Authenticated: true},
		{AdapterID: base.AdapterID, ProviderRef: base.ProviderRef, EvidenceDigest: base.EvidenceDigest, ObservedAt: base.ObservedAt, Authenticated: false},
	}
	right := left
	right.Provenance = append([]AdmissionProviderProvenance(nil), left.Provenance...)
	right.Provenance[0], right.Provenance[2] = right.Provenance[2], right.Provenance[0]

	leftFrozen, err := FreezeAdmissionPacket(left)
	if err != nil {
		t.Fatal(err)
	}
	rightFrozen, err := FreezeAdmissionPacket(right)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(leftFrozen.CanonicalJSON(), rightFrozen.CanonicalJSON()) {
		t.Fatal("provenance ordering must include observation time and authentication state")
	}
}

func TestCTX9AdmissionPacketRejectsProvenanceWithoutEvaluationTime(t *testing.T) {
	for _, fixture := range []struct {
		name   string
		mutate func(*AdmissionPacket)
	}{
		{name: "nil", mutate: func(packet *AdmissionPacket) {
			packet.EvaluationTime = nil
			packet.TimeAuthorityRef = ""
		}},
		{name: "zero", mutate: func(packet *AdmissionPacket) {
			zero := time.Time{}
			packet.EvaluationTime = &zero
		}},
		{name: "missing-authority", mutate: func(packet *AdmissionPacket) {
			packet.TimeAuthorityRef = ""
		}},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			packet := validAdmissionPacket()
			fixture.mutate(&packet)
			requireViolation(t, ValidateAdmissionPacket(packet), "admission_packet.provenance_time_required")
		})
	}
}

func validAdmissionPacket() AdmissionPacket {
	evaluationTime := time.Date(2026, 8, 4, 11, 0, 0, 0, time.UTC)
	return AdmissionPacket{
		Schema:            AdmissionPacketSchemaV1,
		ProjectID:         "prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		DeclarationDigest: digestWith('2'),
		PolicyDigest:      digestWith('a'),
		BaseRevision:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		HeadRevision:      "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		WorkUnitRef:       evidenceReference("work_unit", "work-unit:wu_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", '1'),
		EvaluationTime:    &evaluationTime,
		TimeAuthorityRef:  "provider:github-event-time",
		Evidence: []ContentAddressedEvidenceReference{
			evidenceReference("terminal_receipt", "receipt:run-1", '2'),
			evidenceReference("owner_decision", "decision:owner-1", '3'),
		},
		Provenance: []AdmissionProviderProvenance{
			{AdapterID: "github", ProviderRef: "github:pull-1", EvidenceDigest: digestWith('4'), ObservedAt: evaluationTime, Authenticated: true},
			{AdapterID: "git-local", ProviderRef: "git:range-1", EvidenceDigest: digestWith('5'), ObservedAt: evaluationTime, Authenticated: true},
		},
	}
}

func validAdmissionResult() AdmissionResult {
	return AdmissionResult{
		Schema:          AdmissionResultSchemaV1,
		ProjectID:       "prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		PolicyDigest:    digestWith('a'),
		BaseRevision:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		HeadRevision:    "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		MaterialPaths:   []string{"internal/domain/lineage.go", "internal/domain/admission_v1.go"},
		WorkUnitRef:     evidenceReference("work_unit", "work-unit:wu_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", '1'),
		Classification:  AdmissionValid,
		Outcome:         AdmissionAllow,
		VerifierVersion: "0.2.0",
	}
}
