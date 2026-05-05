-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY,
    display_name TEXT NOT NULL,
    email TEXT NOT NULL DEFAULT '',
    state TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE user_password_credentials (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    password_hash TEXT NOT NULL,
    must_change_password BOOLEAN NOT NULL DEFAULT TRUE,
    password_changed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT user_password_credentials_password_hash_check CHECK (password_hash <> '')
);

CREATE INDEX user_password_credentials_must_change_password_idx
    ON user_password_credentials(must_change_password);

CREATE TABLE user_sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash TEXT NOT NULL,
    state TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ NULL,
    last_used_at TIMESTAMPTZ NULL,
    CONSTRAINT user_sessions_refresh_token_hash_unique UNIQUE (refresh_token_hash),
    CONSTRAINT user_sessions_refresh_token_hash_check CHECK (refresh_token_hash <> ''),
    CONSTRAINT user_sessions_state_check CHECK (state IN ('active', 'revoked', 'expired'))
);

CREATE INDEX user_sessions_user_state_idx
    ON user_sessions(user_id, state);

CREATE INDEX user_sessions_expires_at_idx
    ON user_sessions(expires_at);

CREATE TABLE cli_auth_codes (
    code_hash TEXT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    redirect_uri TEXT NOT NULL,
    state TEXT NOT NULL,
    code_challenge TEXT NOT NULL,
    code_challenge_method TEXT NOT NULL DEFAULT 'S256',
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ NULL,
    CONSTRAINT cli_auth_codes_code_hash_check CHECK (code_hash <> ''),
    CONSTRAINT cli_auth_codes_redirect_uri_check CHECK (redirect_uri <> ''),
    CONSTRAINT cli_auth_codes_state_check CHECK (state <> ''),
    CONSTRAINT cli_auth_codes_code_challenge_check CHECK (code_challenge <> ''),
    CONSTRAINT cli_auth_codes_code_challenge_method_check CHECK (code_challenge_method IN ('S256'))
);

CREATE INDEX cli_auth_codes_user_id_idx
    ON cli_auth_codes(user_id);

CREATE INDEX cli_auth_codes_expires_at_idx
    ON cli_auth_codes(expires_at);

CREATE TABLE installations (
    id UUID PRIMARY KEY,
    mode TEXT NOT NULL,
    public_base_url TEXT NOT NULL,
    state TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT installations_mode_check CHECK (mode IN ('self_hosted', 'saas')),
    CONSTRAINT installations_public_base_url_check
        CHECK (public_base_url <> '' AND public_base_url !~ '/$')
);

CREATE TABLE organizations (
    id UUID PRIMARY KEY,
    installation_id UUID NOT NULL REFERENCES installations(id) ON DELETE CASCADE,
    slug TEXT NOT NULL,
    display_name TEXT NOT NULL,
    state TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT organizations_installation_slug_unique
        UNIQUE (installation_id, slug)
);

CREATE TABLE organization_memberships (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    state TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT organization_memberships_role_check
        CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    CONSTRAINT organization_memberships_org_user_unique
        UNIQUE (organization_id, user_id)
);

CREATE TABLE projects (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_by_user_id UUID NOT NULL REFERENCES users(id),
    slug TEXT NOT NULL,
    display_name TEXT NOT NULL,
    state TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT projects_org_slug_unique UNIQUE (organization_id, slug)
);

CREATE TABLE repo_bindings (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    created_by_user_id UUID NOT NULL REFERENCES users(id),
    vcs_connection_id TEXT NOT NULL DEFAULT '',
    provider TEXT NOT NULL,
    repository_external_id TEXT NOT NULL DEFAULT '',
    repository_full_name TEXT NOT NULL,
    repository_url TEXT NOT NULL,
    default_branch TEXT NOT NULL,
    workflow_base_branch TEXT NOT NULL,
    path_scope TEXT NOT NULL,
    access_mode TEXT NOT NULL,
    state TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT repo_bindings_access_mode_check
        CHECK (access_mode IN (
            'provider_token_checkout',
            'customer_runner_checkout',
            'customer_mounted_workspace',
            'metadata_only'
        ))
);

CREATE INDEX repo_bindings_project_id_idx ON repo_bindings(project_id);

CREATE UNIQUE INDEX repo_bindings_one_active_per_project_idx
    ON repo_bindings(project_id)
    WHERE state = 'active';

CREATE TABLE intake_records (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    repo_binding_id UUID NOT NULL REFERENCES repo_bindings(id) ON DELETE CASCADE,
    source JSONB NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    request_author JSONB NOT NULL,
    intent_owner JSONB NOT NULL,
    state TEXT NOT NULL,
    canonical_contract_created BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX intake_records_organization_created_at_idx
    ON intake_records(organization_id, created_at);

CREATE INDEX intake_records_project_created_at_idx
    ON intake_records(project_id, created_at);

CREATE INDEX intake_records_repo_binding_created_at_idx
    ON intake_records(repo_binding_id, created_at);

CREATE TABLE goals (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    repo_binding_id UUID NOT NULL REFERENCES repo_bindings(id) ON DELETE CASCADE,
    intake_id UUID NOT NULL REFERENCES intake_records(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    summary TEXT NOT NULL,
    scope_hint TEXT NOT NULL DEFAULT '',
    acceptance_hint TEXT NOT NULL DEFAULT '',
    source_refs JSONB NOT NULL DEFAULT '[]'::JSONB,
    request_author JSONB NOT NULL,
    intent_owner JSONB NOT NULL,
    state TEXT NOT NULL,
    last_readiness_reason_codes JSONB NOT NULL DEFAULT '[]'::JSONB,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT goals_intake_id_unique UNIQUE (intake_id)
);

CREATE INDEX goals_organization_created_at_idx
    ON goals(organization_id, created_at);

CREATE INDEX goals_project_created_at_idx
    ON goals(project_id, created_at);

CREATE INDEX goals_repo_binding_created_at_idx
    ON goals(repo_binding_id, created_at);

CREATE INDEX goals_intake_id_idx
    ON goals(intake_id);

CREATE TABLE clarification_requests (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    repo_binding_id UUID NOT NULL REFERENCES repo_bindings(id) ON DELETE CASCADE,
    goal_id UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    state TEXT NOT NULL,
    reason_codes JSONB NOT NULL DEFAULT '[]'::JSONB,
    questions JSONB NOT NULL DEFAULT '[]'::JSONB,
    target JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT clarification_requests_state_check CHECK (state IN ('open', 'answered'))
);

CREATE UNIQUE INDEX clarification_requests_one_open_per_goal_idx
    ON clarification_requests(goal_id)
    WHERE state = 'open';

CREATE INDEX clarification_requests_organization_created_at_idx
    ON clarification_requests(organization_id, created_at);

CREATE INDEX clarification_requests_project_created_at_idx
    ON clarification_requests(project_id, created_at);

CREATE INDEX clarification_requests_repo_binding_id_idx
    ON clarification_requests(repo_binding_id);

CREATE INDEX clarification_requests_goal_id_idx
    ON clarification_requests(goal_id);

CREATE INDEX clarification_requests_state_idx
    ON clarification_requests(state);

CREATE TABLE clarification_answers (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    repo_binding_id UUID NOT NULL REFERENCES repo_bindings(id) ON DELETE CASCADE,
    goal_id UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    clarification_request_id UUID NOT NULL REFERENCES clarification_requests(id) ON DELETE CASCADE,
    submitted_by JSONB NOT NULL,
    answers JSONB NOT NULL DEFAULT '[]'::JSONB,
    applied BOOLEAN NOT NULL DEFAULT FALSE,
    applied_by JSONB NULL,
    applied_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT clarification_answers_request_id_unique UNIQUE (clarification_request_id)
);

CREATE INDEX clarification_answers_organization_created_at_idx
    ON clarification_answers(organization_id, created_at);

CREATE INDEX clarification_answers_project_created_at_idx
    ON clarification_answers(project_id, created_at);

CREATE INDEX clarification_answers_repo_binding_id_idx
    ON clarification_answers(repo_binding_id);

CREATE INDEX clarification_answers_goal_id_idx
    ON clarification_answers(goal_id);

CREATE INDEX clarification_answers_request_id_idx
    ON clarification_answers(clarification_request_id);

CREATE INDEX clarification_answers_applied_idx
    ON clarification_answers(applied);

CREATE TABLE contracts (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    repo_binding_id UUID NOT NULL REFERENCES repo_bindings(id) ON DELETE CASCADE,
    goal_id UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    state TEXT NOT NULL,
    current_seed_id UUID NULL,
    current_draft_id UUID NULL,
    approved_snapshot_id UUID NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT contracts_goal_id_unique UNIQUE (goal_id),
    CONSTRAINT contracts_state_check CHECK (state IN ('seeded', 'draft', 'ready_for_approval', 'approved'))
);

CREATE INDEX contracts_organization_created_at_idx
    ON contracts(organization_id, created_at);

CREATE INDEX contracts_project_created_at_idx
    ON contracts(project_id, created_at);

CREATE INDEX contracts_repo_binding_created_at_idx
    ON contracts(repo_binding_id, created_at);

CREATE TABLE contract_seeds (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    repo_binding_id UUID NOT NULL REFERENCES repo_bindings(id) ON DELETE CASCADE,
    contract_id UUID NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    goal_id UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    intent_summary TEXT NOT NULL,
    intent_owner JSONB NOT NULL,
    scope_hint TEXT NOT NULL,
    acceptance_hint TEXT NOT NULL,
    source_refs JSONB NOT NULL DEFAULT '[]'::JSONB,
    state TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT contract_seeds_contract_id_unique UNIQUE (contract_id),
    CONSTRAINT contract_seeds_goal_id_unique UNIQUE (goal_id),
    CONSTRAINT contract_seeds_state_check CHECK (state IN ('created'))
);

CREATE INDEX contract_seeds_organization_created_at_idx
    ON contract_seeds(organization_id, created_at);

CREATE INDEX contract_seeds_project_created_at_idx
    ON contract_seeds(project_id, created_at);

CREATE INDEX contract_seeds_repo_binding_created_at_idx
    ON contract_seeds(repo_binding_id, created_at);

CREATE INDEX contract_seeds_contract_id_idx
    ON contract_seeds(contract_id);

CREATE TABLE contract_drafts (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    repo_binding_id UUID NOT NULL REFERENCES repo_bindings(id) ON DELETE CASCADE,
    contract_id UUID NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    contract_seed_id UUID NOT NULL REFERENCES contract_seeds(id) ON DELETE CASCADE,
    goal_id UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    intent_summary TEXT NOT NULL,
    proposed_scope JSONB NOT NULL DEFAULT '[]'::JSONB,
    proposed_non_goals JSONB NOT NULL DEFAULT '[]'::JSONB,
    proposed_constraints JSONB NOT NULL DEFAULT '[]'::JSONB,
    proposed_acceptance_criteria JSONB NOT NULL DEFAULT '[]'::JSONB,
    proposed_expected_checks JSONB NOT NULL DEFAULT '[]'::JSONB,
    proposed_proof_expectations JSONB NOT NULL DEFAULT '[]'::JSONB,
    risk_hints JSONB NOT NULL DEFAULT '[]'::JSONB,
    source_refs JSONB NOT NULL DEFAULT '[]'::JSONB,
    state TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT contract_drafts_contract_id_unique UNIQUE (contract_id),
    CONSTRAINT contract_drafts_contract_seed_id_unique UNIQUE (contract_seed_id),
    CONSTRAINT contract_drafts_state_check CHECK (state IN ('draft', 'ready_for_approval'))
);

CREATE INDEX contract_drafts_organization_created_at_idx
    ON contract_drafts(organization_id, created_at);

CREATE INDEX contract_drafts_project_created_at_idx
    ON contract_drafts(project_id, created_at);

CREATE INDEX contract_drafts_repo_binding_created_at_idx
    ON contract_drafts(repo_binding_id, created_at);

CREATE INDEX contract_drafts_contract_id_idx
    ON contract_drafts(contract_id);

CREATE TABLE approved_contracts (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    repo_binding_id UUID NOT NULL REFERENCES repo_bindings(id) ON DELETE CASCADE,
    contract_id UUID NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    contract_draft_id UUID NOT NULL REFERENCES contract_drafts(id) ON DELETE CASCADE,
    contract_seed_id UUID NOT NULL REFERENCES contract_seeds(id) ON DELETE CASCADE,
    goal_id UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    intent_summary TEXT NOT NULL,
    scope JSONB NOT NULL DEFAULT '[]'::JSONB,
    non_goals JSONB NOT NULL DEFAULT '[]'::JSONB,
    constraints JSONB NOT NULL DEFAULT '[]'::JSONB,
    acceptance_criteria JSONB NOT NULL DEFAULT '[]'::JSONB,
    expected_checks JSONB NOT NULL DEFAULT '[]'::JSONB,
    proof_expectations JSONB NOT NULL DEFAULT '[]'::JSONB,
    risk_hints JSONB NOT NULL DEFAULT '[]'::JSONB,
    approved_by JSONB NOT NULL,
    approved_at TIMESTAMPTZ NOT NULL,
    source_refs JSONB NOT NULL DEFAULT '[]'::JSONB,
    state TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT approved_contracts_contract_id_unique UNIQUE (contract_id),
    CONSTRAINT approved_contracts_contract_draft_id_unique UNIQUE (contract_draft_id),
    CONSTRAINT approved_contracts_state_check CHECK (state IN ('approved'))
);

CREATE INDEX approved_contracts_organization_approved_at_idx
    ON approved_contracts(organization_id, approved_at);

CREATE INDEX approved_contracts_project_approved_at_idx
    ON approved_contracts(project_id, approved_at);

CREATE INDEX approved_contracts_repo_binding_approved_at_idx
    ON approved_contracts(repo_binding_id, approved_at);

CREATE INDEX approved_contracts_contract_seed_id_idx
    ON approved_contracts(contract_seed_id);

CREATE INDEX approved_contracts_goal_id_idx
    ON approved_contracts(goal_id);

CREATE INDEX approved_contracts_contract_id_idx
    ON approved_contracts(contract_id);

CREATE TABLE work_item_plans (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    contract_id UUID NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    approved_contract_id UUID NOT NULL REFERENCES approved_contracts(id) ON DELETE CASCADE,
    repo_binding_id UUID NOT NULL REFERENCES repo_bindings(id) ON DELETE CASCADE,
    state TEXT NOT NULL,
    requested_by JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT work_item_plans_contract_id_unique UNIQUE (contract_id),
    CONSTRAINT work_item_plans_state_check CHECK (state IN ('queued', 'proposal_submitted', 'accepted'))
);

CREATE INDEX work_item_plans_organization_created_at_idx
    ON work_item_plans(organization_id, created_at);

CREATE INDEX work_item_plans_project_created_at_idx
    ON work_item_plans(project_id, created_at);

CREATE INDEX work_item_plans_contract_id_idx
    ON work_item_plans(contract_id);

CREATE INDEX work_item_plans_approved_contract_id_idx
    ON work_item_plans(approved_contract_id);

CREATE INDEX work_item_plans_repo_binding_id_idx
    ON work_item_plans(repo_binding_id);

CREATE TABLE work_item_plan_proposals (
    id UUID PRIMARY KEY,
    plan_id UUID NOT NULL REFERENCES work_item_plans(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    contract_id UUID NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    approved_contract_id UUID NOT NULL REFERENCES approved_contracts(id) ON DELETE CASCADE,
    repo_binding_id UUID NOT NULL REFERENCES repo_bindings(id) ON DELETE CASCADE,
    state TEXT NOT NULL,
    submitted_by JSONB NOT NULL,
    planner JSONB NOT NULL DEFAULT '{}'::JSONB,
    source_snapshot_refs JSONB NOT NULL DEFAULT '[]'::JSONB,
    rationale TEXT NOT NULL DEFAULT '',
    proposed_tasks JSONB NOT NULL,
    accepted_by JSONB NULL,
    accepted_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT work_item_plan_proposals_plan_id_unique UNIQUE (plan_id),
    CONSTRAINT work_item_plan_proposals_state_check CHECK (state IN ('submitted', 'accepted'))
);

CREATE INDEX work_item_plan_proposals_plan_id_idx
    ON work_item_plan_proposals(plan_id);

CREATE INDEX work_item_plan_proposals_contract_id_idx
    ON work_item_plan_proposals(contract_id);

CREATE INDEX work_item_plan_proposals_approved_contract_id_idx
    ON work_item_plan_proposals(approved_contract_id);

CREATE INDEX work_item_plan_proposals_organization_created_at_idx
    ON work_item_plan_proposals(organization_id, created_at);

CREATE INDEX work_item_plan_proposals_project_created_at_idx
    ON work_item_plan_proposals(project_id, created_at);

CREATE TABLE work_items (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    contract_id UUID NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    approved_contract_id UUID NOT NULL REFERENCES approved_contracts(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES work_item_plans(id) ON DELETE CASCADE,
    proposal_id UUID NOT NULL REFERENCES work_item_plan_proposals(id) ON DELETE CASCADE,
    repo_binding_id UUID NOT NULL REFERENCES repo_bindings(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    summary TEXT NOT NULL,
    scope JSONB NOT NULL DEFAULT '[]'::JSONB,
    acceptance_refs JSONB NOT NULL DEFAULT '[]'::JSONB,
    proof_expectation_refs JSONB NOT NULL DEFAULT '[]'::JSONB,
    status TEXT NOT NULL,
    owner_hint TEXT NOT NULL DEFAULT '',
    order_index INTEGER NULL,
    source_refs JSONB NOT NULL DEFAULT '[]'::JSONB,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT work_items_status_check CHECK (status IN ('planned'))
);

CREATE INDEX work_items_organization_created_at_idx
    ON work_items(organization_id, created_at);

CREATE INDEX work_items_project_created_at_idx
    ON work_items(project_id, created_at);

CREATE INDEX work_items_contract_id_idx
    ON work_items(contract_id);

CREATE INDEX work_items_approved_contract_id_idx
    ON work_items(approved_contract_id);

CREATE INDEX work_items_plan_id_idx
    ON work_items(plan_id);

CREATE INDEX work_items_proposal_id_idx
    ON work_items(proposal_id);

CREATE INDEX work_items_repo_binding_id_idx
    ON work_items(repo_binding_id);

CREATE TABLE events (
    id UUID PRIMARY KEY,
    event_sequence BIGINT GENERATED ALWAYS AS IDENTITY UNIQUE,
    type TEXT NOT NULL,
    organization_id UUID NULL REFERENCES organizations(id) ON DELETE SET NULL,
    project_id UUID NULL REFERENCES projects(id) ON DELETE SET NULL,
    repo_binding_id UUID NULL REFERENCES repo_bindings(id) ON DELETE SET NULL,
    entity_type TEXT NOT NULL,
    entity_id UUID NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    artifact_refs JSONB NOT NULL DEFAULT '[]'::JSONB,
    causation_id UUID NULL,
    correlation_id UUID NULL
);

CREATE INDEX events_organization_sequence_idx
    ON events(organization_id, event_sequence);

CREATE INDEX events_project_sequence_idx
    ON events(project_id, event_sequence);

CREATE INDEX events_entity_sequence_idx
    ON events(entity_type, entity_id, event_sequence);

CREATE INDEX events_type_sequence_idx
    ON events(type, event_sequence);

-- +goose Down
DROP INDEX IF EXISTS events_type_sequence_idx;
DROP INDEX IF EXISTS events_entity_sequence_idx;
DROP INDEX IF EXISTS events_project_sequence_idx;
DROP INDEX IF EXISTS events_organization_sequence_idx;
DROP TABLE IF EXISTS events;
DROP INDEX IF EXISTS work_items_repo_binding_id_idx;
DROP INDEX IF EXISTS work_items_proposal_id_idx;
DROP INDEX IF EXISTS work_items_plan_id_idx;
DROP INDEX IF EXISTS work_items_approved_contract_id_idx;
DROP INDEX IF EXISTS work_items_contract_id_idx;
DROP INDEX IF EXISTS work_items_project_created_at_idx;
DROP INDEX IF EXISTS work_items_organization_created_at_idx;
DROP TABLE IF EXISTS work_items;
DROP INDEX IF EXISTS work_item_plan_proposals_project_created_at_idx;
DROP INDEX IF EXISTS work_item_plan_proposals_organization_created_at_idx;
DROP INDEX IF EXISTS work_item_plan_proposals_approved_contract_id_idx;
DROP INDEX IF EXISTS work_item_plan_proposals_contract_id_idx;
DROP INDEX IF EXISTS work_item_plan_proposals_plan_id_idx;
DROP TABLE IF EXISTS work_item_plan_proposals;
DROP INDEX IF EXISTS work_item_plans_repo_binding_id_idx;
DROP INDEX IF EXISTS work_item_plans_approved_contract_id_idx;
DROP INDEX IF EXISTS work_item_plans_contract_id_idx;
DROP INDEX IF EXISTS work_item_plans_project_created_at_idx;
DROP INDEX IF EXISTS work_item_plans_organization_created_at_idx;
DROP TABLE IF EXISTS work_item_plans;
DROP INDEX IF EXISTS approved_contracts_contract_id_idx;
DROP INDEX IF EXISTS approved_contracts_goal_id_idx;
DROP INDEX IF EXISTS approved_contracts_contract_seed_id_idx;
DROP INDEX IF EXISTS approved_contracts_repo_binding_approved_at_idx;
DROP INDEX IF EXISTS approved_contracts_project_approved_at_idx;
DROP INDEX IF EXISTS approved_contracts_organization_approved_at_idx;
DROP TABLE IF EXISTS approved_contracts;
DROP INDEX IF EXISTS contract_drafts_repo_binding_created_at_idx;
DROP INDEX IF EXISTS contract_drafts_contract_id_idx;
DROP INDEX IF EXISTS contract_drafts_project_created_at_idx;
DROP INDEX IF EXISTS contract_drafts_organization_created_at_idx;
DROP TABLE IF EXISTS contract_drafts;
DROP INDEX IF EXISTS contract_seeds_repo_binding_created_at_idx;
DROP INDEX IF EXISTS contract_seeds_contract_id_idx;
DROP INDEX IF EXISTS contract_seeds_project_created_at_idx;
DROP INDEX IF EXISTS contract_seeds_organization_created_at_idx;
DROP TABLE IF EXISTS contract_seeds;
DROP INDEX IF EXISTS contracts_repo_binding_created_at_idx;
DROP INDEX IF EXISTS contracts_project_created_at_idx;
DROP INDEX IF EXISTS contracts_organization_created_at_idx;
DROP TABLE IF EXISTS contracts;
DROP INDEX IF EXISTS clarification_answers_applied_idx;
DROP INDEX IF EXISTS clarification_answers_request_id_idx;
DROP INDEX IF EXISTS clarification_answers_goal_id_idx;
DROP INDEX IF EXISTS clarification_answers_repo_binding_id_idx;
DROP INDEX IF EXISTS clarification_answers_project_created_at_idx;
DROP INDEX IF EXISTS clarification_answers_organization_created_at_idx;
DROP TABLE IF EXISTS clarification_answers;
DROP INDEX IF EXISTS clarification_requests_state_idx;
DROP INDEX IF EXISTS clarification_requests_goal_id_idx;
DROP INDEX IF EXISTS clarification_requests_repo_binding_id_idx;
DROP INDEX IF EXISTS clarification_requests_project_created_at_idx;
DROP INDEX IF EXISTS clarification_requests_organization_created_at_idx;
DROP INDEX IF EXISTS clarification_requests_one_open_per_goal_idx;
DROP TABLE IF EXISTS clarification_requests;
DROP INDEX IF EXISTS goals_intake_id_idx;
DROP INDEX IF EXISTS goals_repo_binding_created_at_idx;
DROP INDEX IF EXISTS goals_project_created_at_idx;
DROP INDEX IF EXISTS goals_organization_created_at_idx;
DROP TABLE IF EXISTS goals;
DROP INDEX IF EXISTS intake_records_repo_binding_created_at_idx;
DROP INDEX IF EXISTS intake_records_project_created_at_idx;
DROP INDEX IF EXISTS intake_records_organization_created_at_idx;
DROP TABLE IF EXISTS intake_records;
DROP INDEX IF EXISTS repo_bindings_one_active_per_project_idx;
DROP INDEX IF EXISTS repo_bindings_project_id_idx;
DROP TABLE IF EXISTS repo_bindings;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS organization_memberships;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS installations;
DROP INDEX IF EXISTS cli_auth_codes_expires_at_idx;
DROP INDEX IF EXISTS cli_auth_codes_user_id_idx;
DROP TABLE IF EXISTS cli_auth_codes;
DROP INDEX IF EXISTS user_sessions_expires_at_idx;
DROP INDEX IF EXISTS user_sessions_user_state_idx;
DROP TABLE IF EXISTS user_sessions;
DROP INDEX IF EXISTS user_password_credentials_must_change_password_idx;
DROP TABLE IF EXISTS user_password_credentials;
DROP TABLE IF EXISTS users;
