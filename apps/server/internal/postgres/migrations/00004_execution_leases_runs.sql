-- +goose Up
ALTER TABLE execution_jobs
    DROP CONSTRAINT execution_jobs_state_check;

ALTER TABLE execution_jobs
    ADD COLUMN current_lease_id UUID NULL,
    ADD COLUMN current_runner_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN lease_token_hash TEXT NOT NULL DEFAULT '',
    ADD COLUMN lease_expires_at TIMESTAMPTZ NULL,
    ADD CONSTRAINT execution_jobs_state_check CHECK (state IN ('queued', 'leased', 'run_started'));

CREATE TABLE execution_leases (
    id UUID PRIMARY KEY,
    execution_job_id UUID NOT NULL REFERENCES execution_jobs(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES work_items(id) ON DELETE CASCADE,
    checkout_receipt_id UUID NOT NULL REFERENCES checkout_receipts(id) ON DELETE CASCADE,
    repo_binding_id UUID NOT NULL REFERENCES repo_bindings(id) ON DELETE CASCADE,
    runner_id TEXT NOT NULL,
    state TEXT NOT NULL,
    lease_token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT execution_leases_state_check CHECK (state IN ('active', 'expired', 'run_started')),
    CONSTRAINT execution_leases_runner_id_check CHECK (runner_id <> ''),
    CONSTRAINT execution_leases_token_hash_check CHECK (lease_token_hash <> '')
);

ALTER TABLE execution_jobs
    ADD CONSTRAINT execution_jobs_current_lease_id_fkey
    FOREIGN KEY (current_lease_id) REFERENCES execution_leases(id) ON DELETE SET NULL;

CREATE INDEX execution_leases_job_state_idx
    ON execution_leases(execution_job_id, state);

CREATE INDEX execution_leases_expires_at_idx
    ON execution_leases(expires_at);

CREATE INDEX execution_leases_repo_binding_state_idx
    ON execution_leases(repo_binding_id, state);

CREATE TABLE runs (
    id UUID PRIMARY KEY,
    execution_job_id UUID NOT NULL REFERENCES execution_jobs(id) ON DELETE CASCADE,
    execution_lease_id UUID NOT NULL REFERENCES execution_leases(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES work_items(id) ON DELETE CASCADE,
    checkout_receipt_id UUID NOT NULL REFERENCES checkout_receipts(id) ON DELETE CASCADE,
    runner_id TEXT NOT NULL,
    state TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT runs_execution_job_id_unique UNIQUE (execution_job_id),
    CONSTRAINT runs_execution_lease_id_unique UNIQUE (execution_lease_id),
    CONSTRAINT runs_state_check CHECK (state IN ('started')),
    CONSTRAINT runs_runner_id_check CHECK (runner_id <> '')
);

CREATE INDEX runs_task_id_idx
    ON runs(task_id);

CREATE INDEX runs_checkout_receipt_id_idx
    ON runs(checkout_receipt_id);

CREATE INDEX runs_state_started_at_idx
    ON runs(state, started_at);

CREATE INDEX execution_jobs_current_lease_id_idx
    ON execution_jobs(current_lease_id);

-- +goose Down
DROP INDEX IF EXISTS execution_jobs_current_lease_id_idx;
DROP INDEX IF EXISTS runs_state_started_at_idx;
DROP INDEX IF EXISTS runs_checkout_receipt_id_idx;
DROP INDEX IF EXISTS runs_task_id_idx;
DROP TABLE IF EXISTS runs;
DROP INDEX IF EXISTS execution_leases_repo_binding_state_idx;
DROP INDEX IF EXISTS execution_leases_expires_at_idx;
DROP INDEX IF EXISTS execution_leases_job_state_idx;
ALTER TABLE execution_jobs
    DROP CONSTRAINT IF EXISTS execution_jobs_current_lease_id_fkey;
DROP TABLE IF EXISTS execution_leases;

UPDATE execution_jobs
SET state = 'queued'
WHERE state IN ('leased', 'run_started');

ALTER TABLE execution_jobs
    DROP CONSTRAINT execution_jobs_state_check,
    DROP COLUMN lease_expires_at,
    DROP COLUMN lease_token_hash,
    DROP COLUMN current_runner_id,
    DROP COLUMN current_lease_id,
    ADD CONSTRAINT execution_jobs_state_check CHECK (state IN ('queued'));
