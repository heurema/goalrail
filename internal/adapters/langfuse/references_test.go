package langfuse

import (
	"errors"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/domain"
)

func TestBuildObservationReferencesKeepsSessionLookupWhenTracesAreMissing(t *testing.T) {
	references, err := BuildObservationReferences(validLineage(), nil)
	if err != nil {
		t.Fatalf("build references without observations: %v", err)
	}
	if got, want := references.SessionLookup(), domain.EvidenceReference("langfuse-session:session-1"); got != want {
		t.Fatalf("session lookup = %q, want %q", got, want)
	}
	if references.HasTraceEvidence() {
		t.Fatal("missing observations reported as trace evidence")
	}
	if traces := references.TraceReferences(); len(traces) != 0 {
		t.Fatalf("trace references = %#v, want none", traces)
	}
	all := references.EvidenceReferences()
	if len(all) != 1 || all[0] != references.SessionLookup() {
		t.Fatalf("evidence references = %#v, want only session lookup", all)
	}
}

func TestBuildObservationReferencesValidatesSortsAndDeduplicatesTraces(t *testing.T) {
	traceA := strings.Repeat("a", 32)
	traceB := strings.Repeat("b", 32)
	references, err := BuildObservationReferences(validLineage(), []ObservationIdentity{
		{TraceID: traceB, SessionID: "session-1"},
		{TraceID: strings.ToUpper(traceA), SessionID: "session-1"},
		{TraceID: traceA, SessionID: "session-1"},
	})
	if err != nil {
		t.Fatalf("build references: %v", err)
	}
	if !references.HasTraceEvidence() {
		t.Fatal("trace evidence was not reported")
	}
	want := []domain.EvidenceReference{
		domain.EvidenceReference("langfuse-trace:" + traceA),
		domain.EvidenceReference("langfuse-trace:" + traceB),
	}
	got := references.TraceReferences()
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("trace references = %#v, want %#v", got, want)
	}

	got[0] = "changed:outside"
	if references.TraceReferences()[0] != want[0] {
		t.Fatal("caller mutation changed stored trace references")
	}
}

func TestBuildObservationReferencesRejectsWrongJoin(t *testing.T) {
	_, err := BuildObservationReferences(validLineage(), []ObservationIdentity{
		{TraceID: strings.Repeat("a", 32), SessionID: "different-session"},
	})
	if !errors.Is(err, ErrSessionMismatch) {
		t.Fatalf("build references error = %v, want ErrSessionMismatch", err)
	}
}

func TestBuildObservationReferencesRejectsUnverifiedOrMalformedIdentity(t *testing.T) {
	tests := []struct {
		name         string
		lineage      domain.ExecutionLineage
		observations []ObservationIdentity
		want         error
	}{
		{
			name: "unlinked lineage",
			lineage: domain.ExecutionLineage{
				Status:             domain.LineageUnlinked,
				ChangeID:           "change-1",
				RunID:              "run-1",
				ContextDigest:      strings.Repeat("a", 64),
				UnlinkedReasonCode: "missing-context",
			},
			want: ErrUnverifiedLineage,
		},
		{
			name:    "malformed trace ID",
			lineage: validLineage(),
			observations: []ObservationIdentity{
				{TraceID: "not-a-trace", SessionID: "session-1"},
			},
			want: ErrInvalidObservationIdentity,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := BuildObservationReferences(test.lineage, test.observations)
			if !errors.Is(err, test.want) {
				t.Fatalf("build references error = %v, want %v", err, test.want)
			}
		})
	}
}

func validLineage() domain.ExecutionLineage {
	return domain.ExecutionLineage{
		Status:         domain.LineageVerified,
		ChangeID:       "change-1",
		RunID:          "run-1",
		RootSessionID:  "session-1",
		IdentitySource: domain.SessionIdentityLifecycleHook,
		ContextDigest:  strings.Repeat("a", 64),
	}
}
