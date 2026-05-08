-- +goose Up
CREATE TABLE execution_jobs (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES work_items(id) ON DELETE CASCADE,
    contract_id UUID NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    approved_contract_id UUID NOT NULL REFERENCES approved_contracts(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES work_item_plans(id) ON DELETE CASCADE,
    proposal_id UUID NOT NULL REFERENCES work_item_plan_proposals(id) ON DELETE CASCADE,
    repo_binding_id UUID NOT NULL REFERENCES repo_bindings(id) ON DELETE CASCADE,
    checkout_job_id UUID NOT NULL REFERENCES checkout_jobs(id) ON DELETE CASCADE,
    checkout_receipt_id UUID NOT NULL REFERENCES checkout_receipts(id) ON DELETE CASCADE,
    state TEXT NOT NULL,
    requested_by JSONB NOT NULL,
    execution_mode TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT execution_jobs_task_receipt_unique UNIQUE (task_id, checkout_receipt_id),
    CONSTRAINT execution_jobs_state_check CHECK (state IN ('queued')),
    CONSTRAINT execution_jobs_execution_mode_check CHECK (execution_mode <> '')
);

CREATE INDEX execution_jobs_organization_created_at_idx
    ON execution_jobs(organization_id, created_at);

CREATE INDEX execution_jobs_project_created_at_idx
    ON execution_jobs(project_id, created_at);

CREATE INDEX execution_jobs_task_id_idx
    ON execution_jobs(task_id);

CREATE INDEX execution_jobs_checkout_receipt_id_idx
    ON execution_jobs(checkout_receipt_id);

CREATE INDEX execution_jobs_repo_binding_id_idx
    ON execution_jobs(repo_binding_id);

CREATE INDEX execution_jobs_state_created_at_idx
    ON execution_jobs(state, created_at);

-- +goose Down
DROP INDEX IF EXISTS execution_jobs_state_created_at_idx;
DROP INDEX IF EXISTS execution_jobs_repo_binding_id_idx;
DROP INDEX IF EXISTS execution_jobs_checkout_receipt_id_idx;
DROP INDEX IF EXISTS execution_jobs_task_id_idx;
DROP INDEX IF EXISTS execution_jobs_project_created_at_idx;
DROP INDEX IF EXISTS execution_jobs_organization_created_at_idx;
DROP TABLE IF EXISTS execution_jobs;
