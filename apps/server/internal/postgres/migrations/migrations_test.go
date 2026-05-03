package migrations

import (
	"strings"
	"testing"
)

func TestInitMigrationCreatesInstallationBoundary(t *testing.T) {
	contents, err := FS.ReadFile("00001_init.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	sql := string(contents)
	for _, want := range []string{
		"CREATE TABLE installations",
		"public_base_url TEXT NOT NULL",
		"CONSTRAINT installations_mode_check CHECK (mode IN ('self_hosted', 'saas'))",
		"installation_id UUID NOT NULL REFERENCES installations(id) ON DELETE CASCADE",
		"CONSTRAINT organizations_installation_slug_unique",
		"UNIQUE (installation_id, slug)",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("init migration missing %q", want)
		}
	}
	if strings.Contains(sql, "slug TEXT NOT NULL UNIQUE") {
		t.Fatalf("organizations slug must not be globally unique")
	}
	if strings.Index(sql, "CREATE TABLE installations") > strings.Index(sql, "CREATE TABLE organizations") {
		t.Fatalf("installations must be created before organizations")
	}
	if strings.Index(sql, "DROP TABLE IF EXISTS organizations;") > strings.Index(sql, "DROP TABLE IF EXISTS installations;") {
		t.Fatalf("installations must be dropped after organizations")
	}
}

func TestInitMigrationCreatesAuthCredentialTables(t *testing.T) {
	contents, err := FS.ReadFile("00001_init.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	sql := string(contents)
	for _, want := range []string{
		"CREATE TABLE user_password_credentials",
		"user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE",
		"password_hash TEXT NOT NULL",
		"must_change_password BOOLEAN NOT NULL DEFAULT TRUE",
		"password_changed_at TIMESTAMPTZ NULL",
		"CONSTRAINT user_password_credentials_password_hash_check CHECK (password_hash <> '')",
		"CREATE INDEX user_password_credentials_must_change_password_idx",
		"CREATE TABLE user_sessions",
		"id UUID PRIMARY KEY",
		"user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE",
		"refresh_token_hash TEXT NOT NULL",
		"expires_at TIMESTAMPTZ NOT NULL",
		"revoked_at TIMESTAMPTZ NULL",
		"last_used_at TIMESTAMPTZ NULL",
		"CONSTRAINT user_sessions_refresh_token_hash_unique UNIQUE (refresh_token_hash)",
		"CONSTRAINT user_sessions_refresh_token_hash_check CHECK (refresh_token_hash <> '')",
		"CONSTRAINT user_sessions_state_check CHECK (state IN ('active', 'revoked', 'expired'))",
		"CREATE INDEX user_sessions_user_state_idx",
		"CREATE INDEX user_sessions_expires_at_idx",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("init migration missing %q", want)
		}
	}
	if strings.Contains(sql, "password_hash TEXT") && strings.Contains(sql, "CREATE TABLE users") {
		usersTable := sql[strings.Index(sql, "CREATE TABLE users"):strings.Index(sql, "CREATE TABLE user_password_credentials")]
		if strings.Contains(usersTable, "password_hash") {
			t.Fatalf("users table must not store password_hash")
		}
	}
	if strings.Index(sql, "CREATE TABLE users") > strings.Index(sql, "CREATE TABLE user_password_credentials") {
		t.Fatalf("user_password_credentials must be created after users")
	}
	if strings.Index(sql, "CREATE TABLE users") > strings.Index(sql, "CREATE TABLE user_sessions") {
		t.Fatalf("user_sessions must be created after users")
	}
	if strings.Index(sql, "CREATE TABLE user_sessions") > strings.Index(sql, "CREATE TABLE installations") {
		t.Fatalf("auth session table should stay before installation/project context tables")
	}
}

func TestInitMigrationDropsAuthCredentialTablesBeforeUsers(t *testing.T) {
	contents, err := FS.ReadFile("00001_init.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	sql := string(contents)
	for _, want := range []string{
		"DROP INDEX IF EXISTS user_sessions_expires_at_idx;",
		"DROP INDEX IF EXISTS user_sessions_user_state_idx;",
		"DROP TABLE IF EXISTS user_sessions;",
		"DROP INDEX IF EXISTS user_password_credentials_must_change_password_idx;",
		"DROP TABLE IF EXISTS user_password_credentials;",
		"DROP TABLE IF EXISTS users;",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("init migration down missing %q", want)
		}
	}
	if strings.Index(sql, "DROP TABLE IF EXISTS user_sessions;") > strings.Index(sql, "DROP TABLE IF EXISTS users;") {
		t.Fatalf("user_sessions must be dropped before users")
	}
	if strings.Index(sql, "DROP TABLE IF EXISTS user_password_credentials;") > strings.Index(sql, "DROP TABLE IF EXISTS users;") {
		t.Fatalf("user_password_credentials must be dropped before users")
	}
	if strings.Index(sql, "DROP TABLE IF EXISTS organizations;") > strings.Index(sql, "DROP TABLE IF EXISTS user_sessions;") {
		t.Fatalf("organization-owned tables must be dropped before auth tables")
	}
}

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

func TestInitMigrationCreatesClarificationTables(t *testing.T) {
	contents, err := FS.ReadFile("00001_init.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	sql := string(contents)
	for _, want := range []string{
		"CREATE TABLE clarification_requests",
		"organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE",
		"project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE",
		"repo_binding_id UUID NOT NULL REFERENCES repo_bindings(id) ON DELETE CASCADE",
		"goal_id UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE",
		"CONSTRAINT clarification_requests_state_check CHECK (state IN ('open', 'answered'))",
		"CREATE UNIQUE INDEX clarification_requests_one_open_per_goal_idx",
		"WHERE state = 'open'",
		"CREATE INDEX clarification_requests_organization_created_at_idx",
		"CREATE INDEX clarification_requests_project_created_at_idx",
		"CREATE INDEX clarification_requests_repo_binding_id_idx",
		"CREATE INDEX clarification_requests_goal_id_idx",
		"CREATE INDEX clarification_requests_state_idx",
		"CREATE TABLE clarification_answers",
		"clarification_request_id UUID NOT NULL REFERENCES clarification_requests(id) ON DELETE CASCADE",
		"CONSTRAINT clarification_answers_request_id_unique UNIQUE (clarification_request_id)",
		"CREATE INDEX clarification_answers_organization_created_at_idx",
		"CREATE INDEX clarification_answers_project_created_at_idx",
		"CREATE INDEX clarification_answers_repo_binding_id_idx",
		"CREATE INDEX clarification_answers_goal_id_idx",
		"CREATE INDEX clarification_answers_request_id_idx",
		"CREATE INDEX clarification_answers_applied_idx",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("init migration missing %q", want)
		}
	}
	if strings.Index(sql, "CREATE TABLE goals") > strings.Index(sql, "CREATE TABLE clarification_requests") {
		t.Fatalf("clarification_requests must be created after goals")
	}
	if strings.Index(sql, "CREATE TABLE clarification_requests") > strings.Index(sql, "CREATE TABLE clarification_answers") {
		t.Fatalf("clarification_answers must be created after clarification_requests")
	}
	if strings.Index(sql, "CREATE TABLE clarification_answers") > strings.Index(sql, "CREATE TABLE contracts") {
		t.Fatalf("clarification tables must be created before contracts")
	}
}

func TestInitMigrationDropsClarificationTablesBeforeGoals(t *testing.T) {
	contents, err := FS.ReadFile("00001_init.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	sql := string(contents)
	for _, want := range []string{
		"DROP INDEX IF EXISTS clarification_answers_applied_idx;",
		"DROP INDEX IF EXISTS clarification_answers_request_id_idx;",
		"DROP INDEX IF EXISTS clarification_answers_goal_id_idx;",
		"DROP INDEX IF EXISTS clarification_answers_repo_binding_id_idx;",
		"DROP INDEX IF EXISTS clarification_answers_project_created_at_idx;",
		"DROP INDEX IF EXISTS clarification_answers_organization_created_at_idx;",
		"DROP TABLE IF EXISTS clarification_answers;",
		"DROP INDEX IF EXISTS clarification_requests_state_idx;",
		"DROP INDEX IF EXISTS clarification_requests_goal_id_idx;",
		"DROP INDEX IF EXISTS clarification_requests_repo_binding_id_idx;",
		"DROP INDEX IF EXISTS clarification_requests_project_created_at_idx;",
		"DROP INDEX IF EXISTS clarification_requests_organization_created_at_idx;",
		"DROP INDEX IF EXISTS clarification_requests_one_open_per_goal_idx;",
		"DROP TABLE IF EXISTS clarification_requests;",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("init migration down missing %q", want)
		}
	}
	if strings.Index(sql, "DROP TABLE IF EXISTS clarification_answers;") > strings.Index(sql, "DROP TABLE IF EXISTS clarification_requests;") {
		t.Fatalf("clarification_answers must be dropped before clarification_requests")
	}
	if strings.Index(sql, "DROP TABLE IF EXISTS clarification_requests;") > strings.Index(sql, "DROP TABLE IF EXISTS goals;") {
		t.Fatalf("clarification_requests must be dropped before goals")
	}
	if strings.Index(sql, "DROP TABLE IF EXISTS contracts;") > strings.Index(sql, "DROP TABLE IF EXISTS clarification_answers;") {
		t.Fatalf("contracts must be dropped before clarification tables")
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
