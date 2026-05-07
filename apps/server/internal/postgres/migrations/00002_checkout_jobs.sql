-- +goose Up
CREATE TABLE checkout_jobs (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES work_items(id) ON DELETE CASCADE,
    contract_id UUID NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    approved_contract_id UUID NOT NULL REFERENCES approved_contracts(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES work_item_plans(id) ON DELETE CASCADE,
    proposal_id UUID NOT NULL REFERENCES work_item_plan_proposals(id) ON DELETE CASCADE,
    repo_binding_id UUID NOT NULL REFERENCES repo_bindings(id) ON DELETE CASCADE,
    state TEXT NOT NULL,
    requested_by JSONB NOT NULL,
    instruction JSONB NOT NULL,
    current_runner_id TEXT NOT NULL DEFAULT '',
    lease_token_hash TEXT NOT NULL DEFAULT '',
    lease_expires_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT checkout_jobs_task_id_unique UNIQUE (task_id),
    CONSTRAINT checkout_jobs_state_check CHECK (state IN ('queued', 'leased', 'receipt_submitted'))
);

CREATE INDEX checkout_jobs_organization_created_at_idx
    ON checkout_jobs(organization_id, created_at);

CREATE INDEX checkout_jobs_project_created_at_idx
    ON checkout_jobs(project_id, created_at);

CREATE INDEX checkout_jobs_task_id_idx
    ON checkout_jobs(task_id);

CREATE INDEX checkout_jobs_state_created_at_idx
    ON checkout_jobs(state, created_at);

CREATE INDEX checkout_jobs_repo_binding_id_idx
    ON checkout_jobs(repo_binding_id);

CREATE INDEX checkout_jobs_lease_expires_at_idx
    ON checkout_jobs(lease_expires_at);

CREATE TABLE checkout_receipts (
    id UUID PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES checkout_jobs(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES work_items(id) ON DELETE CASCADE,
    repo_binding_id UUID NOT NULL REFERENCES repo_bindings(id) ON DELETE CASCADE,
    runner_id TEXT NOT NULL,
    workspace_ref TEXT NOT NULL,
    commit_sha TEXT NOT NULL,
    baseline_id TEXT NOT NULL DEFAULT '',
    overlay_id TEXT NOT NULL DEFAULT '',
    dirty BOOLEAN NOT NULL DEFAULT FALSE,
    partial BOOLEAN NOT NULL DEFAULT FALSE,
    partial_reasons JSONB NOT NULL DEFAULT '[]'::JSONB,
    raw_source_uploaded BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT checkout_receipts_job_id_unique UNIQUE (job_id),
    CONSTRAINT checkout_receipts_runner_id_check CHECK (runner_id <> ''),
    CONSTRAINT checkout_receipts_workspace_ref_check CHECK (workspace_ref <> ''),
    CONSTRAINT checkout_receipts_commit_sha_check CHECK (commit_sha <> ''),
    CONSTRAINT checkout_receipts_no_raw_source_check CHECK (raw_source_uploaded = FALSE)
);

CREATE INDEX checkout_receipts_task_id_idx
    ON checkout_receipts(task_id);

CREATE INDEX checkout_receipts_repo_binding_id_idx
    ON checkout_receipts(repo_binding_id);

-- +goose Down
DROP INDEX IF EXISTS checkout_receipts_repo_binding_id_idx;
DROP INDEX IF EXISTS checkout_receipts_task_id_idx;
DROP TABLE IF EXISTS checkout_receipts;
DROP INDEX IF EXISTS checkout_jobs_lease_expires_at_idx;
DROP INDEX IF EXISTS checkout_jobs_repo_binding_id_idx;
DROP INDEX IF EXISTS checkout_jobs_state_created_at_idx;
DROP INDEX IF EXISTS checkout_jobs_task_id_idx;
DROP INDEX IF EXISTS checkout_jobs_project_created_at_idx;
DROP INDEX IF EXISTS checkout_jobs_organization_created_at_idx;
DROP TABLE IF EXISTS checkout_jobs;
