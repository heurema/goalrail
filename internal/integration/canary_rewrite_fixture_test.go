package integration

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/evidence"
)

func TestEvidenceRewriteFixtureBecomesStopVerdict(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	store, err := evidence.NewStore(path)
	if err != nil {
		t.Fatalf("create evidence store: %v", err)
	}
	event := domain.EvidenceEvent{
		ID:         "event-rewrite-fixture-1",
		CanaryID:   domain.IntentCanaryV0ManifestID,
		ChangeID:   "change-rewrite-fixture-1",
		Kind:       domain.EventLineageRecorded,
		OccurredAt: integrationTime,
		Actor:      "goalrail-adapter",
		SourceRef:  "hook:session-start",
		Lineage: &domain.ExecutionLineage{
			Status:         domain.LineageVerified,
			ChangeID:       "change-rewrite-fixture-1",
			RunID:          "run-rewrite-fixture-1",
			RootSessionID:  "session-rewrite-fixture-1",
			IdentitySource: domain.SessionIdentityLifecycleHook,
			ContextDigest:  strings.Repeat("a", 64),
		},
	}
	if err := store.Append(event); err != nil {
		t.Fatalf("append fixture event: %v", err)
	}

	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture evidence: %v", err)
	}
	rewritten := bytes.Replace(original, []byte(`"actor":"goalrail-adapter"`), []byte(`"actor":"other-adapter"`), 1)
	if bytes.Equal(original, rewritten) {
		t.Fatal("fixture did not rewrite the targeted prior event")
	}
	if err := os.WriteFile(path, rewritten, 0o600); err != nil {
		t.Fatalf("write tampered fixture: %v", err)
	}
	if err := store.Verify(); !errors.Is(err, evidence.ErrIntegrity) {
		t.Fatalf("rewrite verification error = %v, want ErrIntegrity", err)
	}

	report, err := domain.CalculateCanaryReport(domain.CanaryReportInput{
		EvidenceIntegrityViolations: 1,
	})
	if err != nil {
		t.Fatalf("calculate rewrite verdict: %v", err)
	}
	if report.Verdict != domain.CanaryVerdictStop ||
		!report.HardStopSignals.EvidenceIntegrityViolation {
		t.Fatalf("rewrite fixture verdict = %q signals=%#v, want STOP", report.Verdict, report.HardStopSignals)
	}
}
