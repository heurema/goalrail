package domain

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"sort"
	"strings"
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

// Every declared reason code must be registered.
//
// The registry is searched with a binary search, so an unregistered code is not
// merely undiscoverable — it is invisible. A new code compiles, is emitted by
// the verifier, and reaches a caller that asks whether it is a known reason and
// is told no. This was not hypothetical: `RESTORATION_NOT_ANCHORED` and
// `RESTORATION_UNBOUND` were declared, used and tested before anyone noticed
// they were absent here, because nothing failed. The sortedness check above
// cannot catch it — a registry missing an entry is still sorted.
//
// Reading the declarations rather than restating them is deliberate: a second
// hand-maintained list would need the same discipline it exists to enforce.
func TestEveryDeclaredReasonCodeIsRegistered(t *testing.T) {
	fileSet := token.NewFileSet()
	packages, err := parser.ParseDir(fileSet, ".", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	declared := 0
	for _, parsed := range packages {
		for _, file := range parsed.Files {
			for _, declaration := range file.Decls {
				group, ok := declaration.(*ast.GenDecl)
				if !ok || group.Tok != token.CONST {
					continue
				}
				for _, specification := range group.Specs {
					value, ok := specification.(*ast.ValueSpec)
					if !ok {
						continue
					}
					name, ok := value.Type.(*ast.Ident)
					if !ok || name.Name != "AdmissionReasonCode" {
						continue
					}
					for _, expression := range value.Values {
						literal, ok := expression.(*ast.BasicLit)
						if !ok || literal.Kind != token.STRING {
							continue
						}
						declared++
						code := AdmissionReasonCode(strings.Trim(literal.Value, `"`))
						if !IsAdmissionReasonCode(code) {
							t.Errorf("reason code %s is declared but not registered", code)
						}
					}
				}
			}
		}
	}
	if declared != len(KnownAdmissionReasonCodes()) {
		t.Fatalf("%d reason codes declared, %d registered", declared, len(KnownAdmissionReasonCodes()))
	}
}
