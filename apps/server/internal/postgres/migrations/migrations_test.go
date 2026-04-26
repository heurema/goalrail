package migrations

import (
	"strings"
	"testing"
)

func TestInitMigrationAllowsContractDraftReadyForApprovalState(t *testing.T) {
	contents, err := FS.ReadFile("00001_init.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	sql := string(contents)
	if !strings.Contains(sql, "CONSTRAINT contract_drafts_state_check CHECK (state IN ('draft', 'ready_for_approval'))") {
		t.Fatalf("contract_drafts_state_check does not allow draft and ready_for_approval states")
	}
}

func TestInitMigrationCreatesApprovedContractsTable(t *testing.T) {
	contents, err := FS.ReadFile("00001_init.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	sql := string(contents)
	for _, want := range []string{
		"CREATE TABLE approved_contracts",
		"CONSTRAINT approved_contracts_contract_draft_id_unique UNIQUE (contract_draft_id)",
		"CONSTRAINT approved_contracts_state_check CHECK (state IN ('approved'))",
		"CREATE INDEX approved_contracts_organization_approved_at_idx",
		"CREATE INDEX approved_contracts_project_approved_at_idx",
		"CREATE INDEX approved_contracts_repo_binding_approved_at_idx",
		"CREATE INDEX approved_contracts_contract_seed_id_idx",
		"CREATE INDEX approved_contracts_goal_id_idx",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("init migration missing %q", want)
		}
	}
}

func TestInitMigrationDropsApprovedContractsTable(t *testing.T) {
	contents, err := FS.ReadFile("00001_init.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	sql := string(contents)
	for _, want := range []string{
		"DROP INDEX IF EXISTS approved_contracts_goal_id_idx;",
		"DROP INDEX IF EXISTS approved_contracts_contract_seed_id_idx;",
		"DROP INDEX IF EXISTS approved_contracts_repo_binding_approved_at_idx;",
		"DROP INDEX IF EXISTS approved_contracts_project_approved_at_idx;",
		"DROP INDEX IF EXISTS approved_contracts_organization_approved_at_idx;",
		"DROP TABLE IF EXISTS approved_contracts;",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("init migration down missing %q", want)
		}
	}
	if strings.Index(sql, "DROP TABLE IF EXISTS approved_contracts;") > strings.Index(sql, "DROP TABLE IF EXISTS contract_drafts;") {
		t.Fatalf("approved_contracts must be dropped before contract_drafts")
	}
}
