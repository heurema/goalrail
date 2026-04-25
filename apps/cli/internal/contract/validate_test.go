package contract

import (
	"testing"

	"github.com/heurema/goalrail/apps/cli/internal/spine"
)

func TestValidateContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		contract   spine.Contract
		wantValid  bool
		wantFields []string
	}{
		{
			name: "valid",
			contract: spine.Contract{
				ID:                 "ctr_demo_1",
				RepoBindingID:      "repo_demo_1",
				Goal:               "Improve checkout error copy",
				InScope:            []string{"checkout error state"},
				AcceptanceCriteria: []string{"error copy is updated"},
				ProofExpectations:  []string{"tests pass"},
				State:              spine.ContractStateApproved,
			},
			wantValid: true,
		},
		{
			name: "missing required fields",
			contract: spine.Contract{
				ID:    "ctr_bad_1",
				State: spine.ContractStateApproved,
			},
			wantValid:  false,
			wantFields: []string{"repo_binding_id", "goal", "in_scope", "acceptance_criteria", "proof_expectations"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			report := Validate(tt.contract)
			if report.Valid != tt.wantValid {
				t.Fatalf("Valid = %v, want %v", report.Valid, tt.wantValid)
			}
			for _, field := range tt.wantFields {
				if !hasFinding(report, field) {
					t.Fatalf("missing finding for field %q in %#v", field, report.Findings)
				}
			}
		})
	}
}

func hasFinding(report spine.ContractValidationReport, field string) bool {
	for _, finding := range report.Findings {
		if finding.Field == field {
			return true
		}
	}
	return false
}
