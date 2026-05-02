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

func TestInitMigrationCreatesContractsTableAndLifecycleLinks(t *testing.T) {
	contents, err := FS.ReadFile("00001_init.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	sql := string(contents)
	for _, want := range []string{
		"CREATE TABLE contracts",
		"CONSTRAINT contracts_goal_id_unique UNIQUE (goal_id)",
		"CONSTRAINT contracts_state_check CHECK (state IN ('seeded', 'draft', 'ready_for_approval', 'approved'))",
		"contract_id UUID NOT NULL REFERENCES contracts(id) ON DELETE CASCADE",
		"CONSTRAINT contract_seeds_contract_id_unique UNIQUE (contract_id)",
		"CONSTRAINT contract_drafts_contract_id_unique UNIQUE (contract_id)",
		"CONSTRAINT approved_contracts_contract_id_unique UNIQUE (contract_id)",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("init migration missing %q", want)
		}
	}
	if strings.Index(sql, "CREATE TABLE contracts") > strings.Index(sql, "CREATE TABLE contract_seeds") {
		t.Fatalf("contracts must be created before contract_seeds")
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
	if strings.Index(sql, "DROP TABLE IF EXISTS contracts;") < strings.Index(sql, "DROP TABLE IF EXISTS contract_seeds;") {
		t.Fatalf("contracts must be dropped after contract_seeds")
	}
}

func TestInitMigrationCreatesWorkItemsTable(t *testing.T) {
	contents, err := FS.ReadFile("00001_init.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	sql := string(contents)
	for _, want := range []string{
		"CREATE TABLE work_items",
		"contract_id UUID NOT NULL REFERENCES contracts(id) ON DELETE CASCADE",
		"approved_contract_id UUID NOT NULL REFERENCES approved_contracts(id) ON DELETE CASCADE",
		"plan_id UUID NOT NULL REFERENCES work_item_plans(id) ON DELETE CASCADE",
		"proposal_id UUID NOT NULL REFERENCES work_item_plan_proposals(id) ON DELETE CASCADE",
		"CONSTRAINT work_items_status_check CHECK (status IN ('planned'))",
		"CREATE INDEX work_items_organization_created_at_idx",
		"CREATE INDEX work_items_project_created_at_idx",
		"CREATE INDEX work_items_contract_id_idx",
		"CREATE INDEX work_items_approved_contract_id_idx",
		"CREATE INDEX work_items_plan_id_idx",
		"CREATE INDEX work_items_proposal_id_idx",
		"CREATE INDEX work_items_repo_binding_id_idx",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("init migration missing %q", want)
		}
	}
	if strings.Index(sql, "CREATE TABLE approved_contracts") > strings.Index(sql, "CREATE TABLE work_items") {
		t.Fatalf("work_items must be created after approved_contracts")
	}
	if strings.Index(sql, "CREATE TABLE work_item_plan_proposals") > strings.Index(sql, "CREATE TABLE work_items") {
		t.Fatalf("work_items must be created after work_item_plan_proposals")
	}
	if strings.Index(sql, "CREATE TABLE work_items") > strings.Index(sql, "CREATE TABLE events") {
		t.Fatalf("work_items must be created before events")
	}
	if strings.Contains(sql, "CONSTRAINT work_items_approved_contract_id_unique UNIQUE (approved_contract_id)") {
		t.Fatalf("work_items must not keep one-task-per-approved-contract unique constraint")
	}
}

func TestInitMigrationCreatesWorkItemPlanAndProposalTables(t *testing.T) {
	contents, err := FS.ReadFile("00001_init.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	sql := string(contents)
	for _, want := range []string{
		"CREATE TABLE work_item_plans",
		"CONSTRAINT work_item_plans_contract_id_unique UNIQUE (contract_id)",
		"CONSTRAINT work_item_plans_state_check CHECK (state IN ('queued', 'proposal_submitted', 'accepted'))",
		"CREATE INDEX work_item_plans_organization_created_at_idx",
		"CREATE INDEX work_item_plans_project_created_at_idx",
		"CREATE INDEX work_item_plans_contract_id_idx",
		"CREATE INDEX work_item_plans_approved_contract_id_idx",
		"CREATE INDEX work_item_plans_repo_binding_id_idx",
		"CREATE TABLE work_item_plan_proposals",
		"CONSTRAINT work_item_plan_proposals_plan_id_unique UNIQUE (plan_id)",
		"CONSTRAINT work_item_plan_proposals_state_check CHECK (state IN ('submitted', 'accepted'))",
		"CREATE INDEX work_item_plan_proposals_plan_id_idx",
		"CREATE INDEX work_item_plan_proposals_contract_id_idx",
		"CREATE INDEX work_item_plan_proposals_approved_contract_id_idx",
		"CREATE INDEX work_item_plan_proposals_organization_created_at_idx",
		"CREATE INDEX work_item_plan_proposals_project_created_at_idx",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("init migration missing %q", want)
		}
	}
	if strings.Index(sql, "CREATE TABLE work_item_plans") > strings.Index(sql, "CREATE TABLE work_item_plan_proposals") {
		t.Fatalf("work_item_plans must be created before work_item_plan_proposals")
	}
}

func TestInitMigrationDropsWorkItemsBeforeApprovedContracts(t *testing.T) {
	contents, err := FS.ReadFile("00001_init.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	sql := string(contents)
	for _, want := range []string{
		"DROP INDEX IF EXISTS work_items_repo_binding_id_idx;",
		"DROP INDEX IF EXISTS work_items_proposal_id_idx;",
		"DROP INDEX IF EXISTS work_items_plan_id_idx;",
		"DROP INDEX IF EXISTS work_items_approved_contract_id_idx;",
		"DROP INDEX IF EXISTS work_items_contract_id_idx;",
		"DROP INDEX IF EXISTS work_items_project_created_at_idx;",
		"DROP INDEX IF EXISTS work_items_organization_created_at_idx;",
		"DROP TABLE IF EXISTS work_items;",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("init migration down missing %q", want)
		}
	}
	if strings.Index(sql, "DROP TABLE IF EXISTS events;") > strings.Index(sql, "DROP TABLE IF EXISTS work_items;") {
		t.Fatalf("events must be dropped before work_items")
	}
	if strings.Index(sql, "DROP TABLE IF EXISTS work_items;") > strings.Index(sql, "DROP TABLE IF EXISTS approved_contracts;") {
		t.Fatalf("work_items must be dropped before approved_contracts")
	}
	if strings.Index(sql, "DROP TABLE IF EXISTS work_items;") > strings.Index(sql, "DROP TABLE IF EXISTS work_item_plan_proposals;") {
		t.Fatalf("work_items must be dropped before work_item_plan_proposals")
	}
	if strings.Index(sql, "DROP TABLE IF EXISTS work_item_plan_proposals;") > strings.Index(sql, "DROP TABLE IF EXISTS work_item_plans;") {
		t.Fatalf("work_item_plan_proposals must be dropped before work_item_plans")
	}
}
