-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS users_email_lower_unique
    ON users (lower(email))
    WHERE email <> '';

ALTER TABLE repo_bindings
    ADD COLUMN IF NOT EXISTS workflow_base_branch TEXT;

UPDATE repo_bindings
SET workflow_base_branch = COALESCE(NULLIF(default_branch, ''), 'main')
WHERE workflow_base_branch IS NULL OR workflow_base_branch = '';

ALTER TABLE repo_bindings
    ALTER COLUMN workflow_base_branch SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS repo_bindings_one_active_per_org_repository_idx
    ON repo_bindings(organization_id, lower(provider), lower(repository_full_name))
    WHERE state = 'active';

CREATE TABLE IF NOT EXISTS repository_context_snapshots (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    repo_binding_id UUID NOT NULL REFERENCES repo_bindings(id) ON DELETE CASCADE,
    source TEXT NOT NULL,
    schema_version INTEGER NOT NULL,
    fingerprint TEXT NOT NULL,
    snapshot JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT repository_context_snapshots_source_check CHECK (source <> ''),
    CONSTRAINT repository_context_snapshots_schema_version_check CHECK (schema_version = 1),
    CONSTRAINT repository_context_snapshots_fingerprint_check CHECK (fingerprint <> '')
);

CREATE UNIQUE INDEX IF NOT EXISTS repository_context_snapshots_repo_binding_fingerprint_idx
    ON repository_context_snapshots(repo_binding_id, fingerprint);

CREATE INDEX IF NOT EXISTS repository_context_snapshots_organization_created_at_idx
    ON repository_context_snapshots(organization_id, created_at);

CREATE INDEX IF NOT EXISTS repository_context_snapshots_project_created_at_idx
    ON repository_context_snapshots(project_id, created_at);

CREATE INDEX IF NOT EXISTS repository_context_snapshots_repo_binding_created_at_idx
    ON repository_context_snapshots(repo_binding_id, created_at);

ALTER TABLE work_item_plans
    ADD COLUMN IF NOT EXISTS current_lease_id UUID NULL,
    ADD COLUMN IF NOT EXISTS leased_by JSONB NULL,
    ADD COLUMN IF NOT EXISTS lease_expires_at TIMESTAMPTZ NULL;

ALTER TABLE work_item_plans
    DROP CONSTRAINT IF EXISTS work_item_plans_state_check;

ALTER TABLE work_item_plans
    ADD CONSTRAINT work_item_plans_state_check CHECK (state IN ('queued', 'leased', 'proposal_submitted', 'accepted'));

CREATE INDEX IF NOT EXISTS work_item_plans_state_created_at_idx
    ON work_item_plans(state, created_at);

CREATE INDEX IF NOT EXISTS work_item_plans_lease_expires_at_idx
    ON work_item_plans(lease_expires_at);

CREATE TABLE IF NOT EXISTS work_item_plan_leases (
    id UUID PRIMARY KEY,
    plan_id UUID NOT NULL REFERENCES work_item_plans(id) ON DELETE CASCADE,
    contract_id UUID NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    approved_contract_id UUID NOT NULL REFERENCES approved_contracts(id) ON DELETE CASCADE,
    repo_binding_id UUID NOT NULL REFERENCES repo_bindings(id) ON DELETE CASCADE,
    leased_by JSONB NOT NULL,
    state TEXT NOT NULL,
    lease_token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT work_item_plan_leases_state_check CHECK (state IN ('active', 'completed', 'expired')),
    CONSTRAINT work_item_plan_leases_token_hash_check CHECK (lease_token_hash <> '')
);

CREATE INDEX IF NOT EXISTS work_item_plan_leases_plan_id_idx
    ON work_item_plan_leases(plan_id);

CREATE INDEX IF NOT EXISTS work_item_plan_leases_state_expires_at_idx
    ON work_item_plan_leases(state, expires_at);

CREATE INDEX IF NOT EXISTS work_item_plan_leases_contract_id_idx
    ON work_item_plan_leases(contract_id);

CREATE INDEX IF NOT EXISTS work_item_plan_leases_repo_binding_id_idx
    ON work_item_plan_leases(repo_binding_id);

-- +goose Down
DROP INDEX IF EXISTS work_item_plan_leases_repo_binding_id_idx;
DROP INDEX IF EXISTS work_item_plan_leases_contract_id_idx;
DROP INDEX IF EXISTS work_item_plan_leases_state_expires_at_idx;
DROP INDEX IF EXISTS work_item_plan_leases_plan_id_idx;
DROP TABLE IF EXISTS work_item_plan_leases;

DROP INDEX IF EXISTS work_item_plans_lease_expires_at_idx;
DROP INDEX IF EXISTS work_item_plans_state_created_at_idx;

UPDATE work_item_plans
SET state = 'queued'
WHERE state = 'leased';

ALTER TABLE work_item_plans
    DROP CONSTRAINT IF EXISTS work_item_plans_state_check;

ALTER TABLE work_item_plans
    ADD CONSTRAINT work_item_plans_state_check CHECK (state IN ('queued', 'proposal_submitted', 'accepted'));

ALTER TABLE work_item_plans
    DROP COLUMN IF EXISTS lease_expires_at,
    DROP COLUMN IF EXISTS leased_by,
    DROP COLUMN IF EXISTS current_lease_id;

DROP INDEX IF EXISTS repository_context_snapshots_repo_binding_created_at_idx;
DROP INDEX IF EXISTS repository_context_snapshots_project_created_at_idx;
DROP INDEX IF EXISTS repository_context_snapshots_organization_created_at_idx;
DROP INDEX IF EXISTS repository_context_snapshots_repo_binding_fingerprint_idx;
DROP TABLE IF EXISTS repository_context_snapshots;

DROP INDEX IF EXISTS repo_bindings_one_active_per_org_repository_idx;

ALTER TABLE repo_bindings
    DROP COLUMN IF EXISTS workflow_base_branch;

DROP INDEX IF EXISTS users_email_lower_unique;
