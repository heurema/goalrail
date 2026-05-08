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

func TestInitMigrationStoresRepoBindingWorkflowBaseBranch(t *testing.T) {
	contents, err := FS.ReadFile("00001_init.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	sql := string(contents)
	for _, want := range []string{
		"CREATE TABLE repo_bindings",
		"default_branch TEXT NOT NULL",
		"workflow_base_branch TEXT NOT NULL",
		"CONSTRAINT repo_bindings_access_mode_check",
		"'metadata_only'",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("init migration missing %q", want)
		}
	}
	if strings.Index(sql, "default_branch TEXT NOT NULL") > strings.Index(sql, "workflow_base_branch TEXT NOT NULL") {
		t.Fatalf("workflow_base_branch should be stored next to default_branch")
	}
}

func TestInitMigrationEnforcesOneActiveRepoBindingPerOrganizationRepository(t *testing.T) {
	contents, err := FS.ReadFile("00001_init.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	sql := string(contents)
	for _, want := range []string{
		"CREATE UNIQUE INDEX repo_bindings_one_active_per_org_repository_idx",
		"ON repo_bindings(organization_id, lower(provider), lower(repository_full_name))",
		"WHERE state = 'active'",
		"DROP INDEX IF EXISTS repo_bindings_one_active_per_org_repository_idx;",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("init migration missing %q", want)
		}
	}
}

func TestInitMigrationCreatesRepositoryContextSnapshots(t *testing.T) {
	contents, err := FS.ReadFile("00001_init.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	sql := string(contents)
	for _, want := range []string{
		"CREATE TABLE repository_context_snapshots",
		"repo_binding_id UUID NOT NULL REFERENCES repo_bindings(id) ON DELETE CASCADE",
		"snapshot JSONB NOT NULL",
		"CONSTRAINT repository_context_snapshots_schema_version_check CHECK (schema_version = 1)",
		"CREATE UNIQUE INDEX repository_context_snapshots_repo_binding_fingerprint_idx",
		"ON repository_context_snapshots(repo_binding_id, fingerprint)",
		"CREATE INDEX repository_context_snapshots_organization_created_at_idx",
		"CREATE INDEX repository_context_snapshots_project_created_at_idx",
		"CREATE INDEX repository_context_snapshots_repo_binding_created_at_idx",
		"DROP TABLE IF EXISTS repository_context_snapshots;",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("init migration missing %q", want)
		}
	}
	if strings.Index(sql, "CREATE TABLE repo_bindings") > strings.Index(sql, "CREATE TABLE repository_context_snapshots") {
		t.Fatalf("repository_context_snapshots must be created after repo_bindings")
	}
	if strings.Index(sql, "CREATE TABLE repository_context_snapshots") > strings.Index(sql, "CREATE TABLE intake_records") {
		t.Fatalf("repository_context_snapshots should stay in project-context foundation before intake")
	}
	if strings.Index(sql, "DROP TABLE IF EXISTS intake_records;") > strings.Index(sql, "DROP TABLE IF EXISTS repository_context_snapshots;") {
		t.Fatalf("repository_context_snapshots must be dropped after intake_records")
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
		"CREATE UNIQUE INDEX users_email_lower_unique",
		"ON users (lower(email))",
		"WHERE email <> ''",
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
		"CREATE TABLE cli_auth_codes",
		"code_hash TEXT PRIMARY KEY",
		"redirect_uri TEXT NOT NULL",
		"state TEXT NOT NULL",
		"code_challenge TEXT NOT NULL",
		"code_challenge_method TEXT NOT NULL DEFAULT 'S256'",
		"CONSTRAINT cli_auth_codes_code_hash_check CHECK (code_hash <> '')",
		"CONSTRAINT cli_auth_codes_redirect_uri_check CHECK (redirect_uri <> '')",
		"CONSTRAINT cli_auth_codes_state_check CHECK (state <> '')",
		"CONSTRAINT cli_auth_codes_code_challenge_check CHECK (code_challenge <> '')",
		"CONSTRAINT cli_auth_codes_code_challenge_method_check CHECK (code_challenge_method IN ('S256'))",
		"CREATE INDEX cli_auth_codes_user_id_idx",
		"CREATE INDEX cli_auth_codes_expires_at_idx",
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
	if strings.Index(sql, "CREATE TABLE user_sessions") > strings.Index(sql, "CREATE TABLE cli_auth_codes") {
		t.Fatalf("cli_auth_codes must be created after user_sessions")
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
		"DROP INDEX IF EXISTS cli_auth_codes_expires_at_idx;",
		"DROP INDEX IF EXISTS cli_auth_codes_user_id_idx;",
		"DROP TABLE IF EXISTS cli_auth_codes;",
		"DROP INDEX IF EXISTS user_password_credentials_must_change_password_idx;",
		"DROP TABLE IF EXISTS user_password_credentials;",
		"DROP INDEX IF EXISTS users_email_lower_unique;",
		"DROP TABLE IF EXISTS users;",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("init migration down missing %q", want)
		}
	}
	if strings.Index(sql, "DROP TABLE IF EXISTS user_sessions;") > strings.Index(sql, "DROP TABLE IF EXISTS users;") {
		t.Fatalf("user_sessions must be dropped before users")
	}
	if strings.Index(sql, "DROP TABLE IF EXISTS cli_auth_codes;") > strings.Index(sql, "DROP TABLE IF EXISTS users;") {
		t.Fatalf("cli_auth_codes must be dropped before users")
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
	if strings.Contains(sql, "CREATE TABLE checkout_jobs") || strings.Contains(sql, "CREATE TABLE checkout_receipts") {
		t.Fatalf("checkout tables must live in a follow-up migration, not 00001_init.sql")
	}
}

func TestCheckoutMigrationCreatesCheckoutTables(t *testing.T) {
	contents, err := FS.ReadFile("00002_checkout_jobs.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	sql := string(contents)
	for _, want := range []string{
		"CREATE TABLE checkout_jobs",
		"task_id UUID NOT NULL REFERENCES work_items(id) ON DELETE CASCADE",
		"current_runner_id TEXT NOT NULL DEFAULT ''",
		"lease_token_hash TEXT NOT NULL DEFAULT ''",
		"CONSTRAINT checkout_jobs_task_id_unique UNIQUE (task_id)",
		"CONSTRAINT checkout_jobs_state_check CHECK (state IN ('queued', 'leased', 'receipt_submitted'))",
		"CREATE INDEX checkout_jobs_organization_created_at_idx",
		"CREATE INDEX checkout_jobs_project_created_at_idx",
		"CREATE INDEX checkout_jobs_task_id_idx",
		"CREATE INDEX checkout_jobs_state_created_at_idx",
		"CREATE INDEX checkout_jobs_repo_binding_id_idx",
		"CREATE INDEX checkout_jobs_lease_expires_at_idx",
		"CREATE TABLE checkout_receipts",
		"job_id UUID NOT NULL REFERENCES checkout_jobs(id) ON DELETE CASCADE",
		"raw_source_uploaded BOOLEAN NOT NULL DEFAULT FALSE",
		"CONSTRAINT checkout_receipts_job_id_unique UNIQUE (job_id)",
		"CONSTRAINT checkout_receipts_no_raw_source_check CHECK (raw_source_uploaded = FALSE)",
		"CREATE INDEX checkout_receipts_task_id_idx",
		"CREATE INDEX checkout_receipts_repo_binding_id_idx",
		"DROP TABLE IF EXISTS checkout_receipts;",
		"DROP TABLE IF EXISTS checkout_jobs;",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("checkout migration missing %q", want)
		}
	}
	if strings.Index(sql, "CREATE TABLE checkout_jobs") > strings.Index(sql, "CREATE TABLE checkout_receipts") {
		t.Fatalf("checkout_jobs must be created before checkout_receipts")
	}
	if strings.Index(sql, "DROP TABLE IF EXISTS checkout_receipts;") > strings.Index(sql, "DROP TABLE IF EXISTS checkout_jobs;") {
		t.Fatalf("checkout_receipts must be dropped before checkout_jobs")
	}
}

func TestExecutionMigrationCreatesExecutionJobTable(t *testing.T) {
	contents, err := FS.ReadFile("00003_execution_jobs.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	sql := string(contents)
	for _, want := range []string{
		"CREATE TABLE execution_jobs",
		"task_id UUID NOT NULL REFERENCES work_items(id) ON DELETE CASCADE",
		"checkout_job_id UUID NOT NULL REFERENCES checkout_jobs(id) ON DELETE CASCADE",
		"checkout_receipt_id UUID NOT NULL REFERENCES checkout_receipts(id) ON DELETE CASCADE",
		"CONSTRAINT execution_jobs_task_receipt_unique UNIQUE (task_id, checkout_receipt_id)",
		"CONSTRAINT execution_jobs_state_check CHECK (state IN ('queued'))",
		"CONSTRAINT execution_jobs_execution_mode_check CHECK (execution_mode <> '')",
		"CREATE INDEX execution_jobs_organization_created_at_idx",
		"CREATE INDEX execution_jobs_project_created_at_idx",
		"CREATE INDEX execution_jobs_task_id_idx",
		"CREATE INDEX execution_jobs_checkout_receipt_id_idx",
		"CREATE INDEX execution_jobs_repo_binding_id_idx",
		"CREATE INDEX execution_jobs_state_created_at_idx",
		"DROP TABLE IF EXISTS execution_jobs;",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("execution migration missing %q", want)
		}
	}
	if strings.Contains(sql, "CREATE TABLE runs") || strings.Contains(sql, "CREATE TABLE execution_receipts") {
		t.Fatalf("execution job migration must not create Run or execution receipt tables")
	}
}

func TestExecutionLeaseRunMigrationAddsRunStartBoundary(t *testing.T) {
	contents, err := FS.ReadFile("00004_execution_leases_runs.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	sql := string(contents)
	for _, want := range []string{
		"ADD COLUMN current_lease_id UUID NULL",
		"CONSTRAINT execution_jobs_state_check CHECK (state IN ('queued', 'leased', 'run_started'))",
		"CREATE TABLE execution_leases",
		"lease_token_hash TEXT NOT NULL",
		"CONSTRAINT execution_leases_state_check CHECK (state IN ('active', 'expired', 'run_started'))",
		"CREATE TABLE runs",
		"execution_job_id UUID NOT NULL REFERENCES execution_jobs(id) ON DELETE CASCADE",
		"execution_lease_id UUID NOT NULL REFERENCES execution_leases(id) ON DELETE CASCADE",
		"CONSTRAINT runs_execution_job_id_unique UNIQUE (execution_job_id)",
		"CONSTRAINT runs_execution_lease_id_unique UNIQUE (execution_lease_id)",
		"CONSTRAINT runs_state_check CHECK (state IN ('started'))",
		"DROP TABLE IF EXISTS runs;",
		"DROP TABLE IF EXISTS execution_leases;",
		"WHERE state IN ('leased', 'run_started')",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("execution lease/run migration missing %q", want)
		}
	}
	if strings.Contains(sql, "CREATE TABLE execution_receipts") || strings.Contains(sql, "CREATE TABLE gate_decisions") || strings.Contains(sql, "CREATE TABLE proofs") {
		t.Fatalf("execution lease/run migration must not create receipt, gate, or proof tables")
	}
	if strings.Contains(sql, "lease_token TEXT") {
		t.Fatalf("execution lease migration must not store raw lease token")
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
		"current_lease_id UUID NULL",
		"leased_by JSONB NULL",
		"lease_expires_at TIMESTAMPTZ NULL",
		"CONSTRAINT work_item_plans_state_check CHECK (state IN ('queued', 'leased', 'proposal_submitted', 'accepted'))",
		"CREATE INDEX work_item_plans_organization_created_at_idx",
		"CREATE INDEX work_item_plans_project_created_at_idx",
		"CREATE INDEX work_item_plans_contract_id_idx",
		"CREATE INDEX work_item_plans_approved_contract_id_idx",
		"CREATE INDEX work_item_plans_repo_binding_id_idx",
		"CREATE INDEX work_item_plans_state_created_at_idx",
		"CREATE INDEX work_item_plans_lease_expires_at_idx",
		"CREATE TABLE work_item_plan_proposals",
		"CONSTRAINT work_item_plan_proposals_plan_id_unique UNIQUE (plan_id)",
		"CONSTRAINT work_item_plan_proposals_state_check CHECK (state IN ('submitted', 'accepted'))",
		"CREATE INDEX work_item_plan_proposals_plan_id_idx",
		"CREATE INDEX work_item_plan_proposals_contract_id_idx",
		"CREATE INDEX work_item_plan_proposals_approved_contract_id_idx",
		"CREATE INDEX work_item_plan_proposals_organization_created_at_idx",
		"CREATE INDEX work_item_plan_proposals_project_created_at_idx",
		"CREATE TABLE work_item_plan_leases",
		"lease_token_hash TEXT NOT NULL",
		"CONSTRAINT work_item_plan_leases_state_check CHECK (state IN ('active', 'completed', 'expired'))",
		"CREATE INDEX work_item_plan_leases_plan_id_idx",
		"CREATE INDEX work_item_plan_leases_state_expires_at_idx",
		"CREATE INDEX work_item_plan_leases_contract_id_idx",
		"CREATE INDEX work_item_plan_leases_repo_binding_id_idx",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("init migration missing %q", want)
		}
	}
	if strings.Index(sql, "CREATE TABLE work_item_plans") > strings.Index(sql, "CREATE TABLE work_item_plan_proposals") {
		t.Fatalf("work_item_plans must be created before work_item_plan_proposals")
	}
	if strings.Index(sql, "CREATE TABLE work_item_plan_proposals") > strings.Index(sql, "CREATE TABLE work_item_plan_leases") {
		t.Fatalf("work_item_plan_leases must be created after work_item_plan_proposals")
	}
	if strings.Index(sql, "CREATE TABLE work_item_plan_leases") > strings.Index(sql, "CREATE TABLE work_items") {
		t.Fatalf("work_items must be created after work_item_plan_leases")
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
		"DROP INDEX IF EXISTS work_item_plan_leases_repo_binding_id_idx;",
		"DROP INDEX IF EXISTS work_item_plan_leases_contract_id_idx;",
		"DROP INDEX IF EXISTS work_item_plan_leases_state_expires_at_idx;",
		"DROP INDEX IF EXISTS work_item_plan_leases_plan_id_idx;",
		"DROP TABLE IF EXISTS work_item_plan_leases;",
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
	if strings.Index(sql, "DROP TABLE IF EXISTS work_items;") > strings.Index(sql, "DROP TABLE IF EXISTS work_item_plan_leases;") {
		t.Fatalf("work_items must be dropped before work_item_plan_leases")
	}
	if strings.Index(sql, "DROP TABLE IF EXISTS work_item_plan_leases;") > strings.Index(sql, "DROP TABLE IF EXISTS work_item_plans;") {
		t.Fatalf("work_item_plan_leases must be dropped before work_item_plans")
	}
	if strings.Index(sql, "DROP TABLE IF EXISTS work_item_plan_proposals;") > strings.Index(sql, "DROP TABLE IF EXISTS work_item_plans;") {
		t.Fatalf("work_item_plan_proposals must be dropped before work_item_plans")
	}
}
