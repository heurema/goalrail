package proof

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/heurema/goalrail/apps/cli/internal/spine"
)

func TestRenderJSONProducesValidJSON(t *testing.T) {
	t.Parallel()

	packet := spine.Proof{
		ID:              "pf_demo_1",
		ContractID:      "ctr_demo_1",
		RunID:           "run_demo_1",
		Verdict:         spine.ProofVerdictAccept,
		ChangedScope:    []string{"README.md"},
		UnchangedScope:  []string{"runtime"},
		Coverage:        []spine.ProofCoverage{{Criterion: "copy updated", Evidence: []string{"diff"}, Status: "covered"}},
		Checks:          []spine.ProofCheck{{Name: "go test ./...", Status: "pass", Note: "demo"}},
		ResidualRisks:   []string{"demo proof only"},
		DecisionSummary: "Sample proof packet accepted for demo rendering.",
	}

	var buf bytes.Buffer
	if err := RenderJSON(&buf, packet); err != nil {
		t.Fatalf("RenderJSON() error = %v", err)
	}

	var got spine.Proof
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("rendered JSON is invalid: %v\n%s", err, buf.String())
	}
	if got.ID != packet.ID {
		t.Fatalf("ID = %q, want %q", got.ID, packet.ID)
	}
}
