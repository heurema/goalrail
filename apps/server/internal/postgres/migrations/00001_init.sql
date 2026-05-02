-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY,
    display_name TEXT NOT NULL,
    email TEXT NOT NULL DEFAULT '',
    state TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE organizations (
    id UUID PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    state TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
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
DROP TABLE IF EXISTS users;
