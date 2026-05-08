-- +goose Up
ALTER TABLE execution_jobs
    DROP CONSTRAINT execution_jobs_state_check;

ALTER TABLE execution_jobs
    ADD CONSTRAINT execution_jobs_state_check CHECK (state IN ('queued', 'leased', 'run_started', 'receipt_submitted'));

ALTER TABLE runs
    DROP CONSTRAINT runs_state_check;

ALTER TABLE runs
    ADD COLUMN finished_at TIMESTAMPTZ NULL,
    ADD CONSTRAINT runs_state_check CHECK (state IN ('started', 'receipt_submitted'));

CREATE TABLE execution_receipts (
    id UUID PRIMARY KEY,
    run_id UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    execution_job_id UUID NOT NULL REFERENCES execution_jobs(id) ON DELETE CASCADE,
    execution_lease_id UUID NOT NULL REFERENCES execution_leases(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES work_items(id) ON DELETE CASCADE,
    checkout_receipt_id UUID NOT NULL REFERENCES checkout_receipts(id) ON DELETE CASCADE,
    repo_binding_id UUID NOT NULL REFERENCES repo_bindings(id) ON DELETE CASCADE,
    runner_id TEXT NOT NULL,
    workspace_ref TEXT NOT NULL,
    commit_sha TEXT NOT NULL,
    baseline_id TEXT NOT NULL DEFAULT '',
    overlay_id TEXT NOT NULL DEFAULT '',
    execution_mode TEXT NOT NULL,
    process_status TEXT NOT NULL,
    exit_code INTEGER NULL,
    artifact_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
    changed_paths_summary JSONB NOT NULL DEFAULT '[]'::jsonb,
    raw_source_uploaded BOOLEAN NOT NULL DEFAULT FALSE,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT execution_receipts_run_id_unique UNIQUE (run_id),
    CONSTRAINT execution_receipts_runner_id_check CHECK (runner_id <> ''),
    CONSTRAINT execution_receipts_workspace_ref_check CHECK (workspace_ref <> ''),
    CONSTRAINT execution_receipts_commit_sha_check CHECK (commit_sha <> ''),
    CONSTRAINT execution_receipts_mode_check CHECK (execution_mode = 'no_command'),
    CONSTRAINT execution_receipts_process_status_check CHECK (process_status IN ('not_executed', 'metadata_only')),
    CONSTRAINT execution_receipts_no_exit_code_check CHECK (exit_code IS NULL),
    CONSTRAINT execution_receipts_no_artifacts_check CHECK (artifact_refs = '[]'::jsonb),
    CONSTRAINT execution_receipts_no_changed_paths_check CHECK (changed_paths_summary = '[]'::jsonb),
    CONSTRAINT execution_receipts_no_raw_source_check CHECK (raw_source_uploaded = FALSE)
);

CREATE INDEX execution_receipts_job_id_idx
    ON execution_receipts(execution_job_id);

CREATE INDEX execution_receipts_task_id_idx
    ON execution_receipts(task_id);

CREATE INDEX execution_receipts_repo_binding_id_idx
    ON execution_receipts(repo_binding_id);

CREATE INDEX execution_receipts_finished_at_idx
    ON execution_receipts(finished_at);

-- +goose Down
DROP INDEX IF EXISTS execution_receipts_finished_at_idx;
DROP INDEX IF EXISTS execution_receipts_repo_binding_id_idx;
DROP INDEX IF EXISTS execution_receipts_task_id_idx;
DROP INDEX IF EXISTS execution_receipts_job_id_idx;
DROP TABLE IF EXISTS execution_receipts;

UPDATE runs
SET state = 'started'
WHERE state = 'receipt_submitted';

ALTER TABLE runs
    DROP CONSTRAINT runs_state_check,
    DROP COLUMN finished_at,
    ADD CONSTRAINT runs_state_check CHECK (state IN ('started'));

UPDATE execution_jobs
SET state = 'run_started'
WHERE state = 'receipt_submitted';

ALTER TABLE execution_jobs
    DROP CONSTRAINT execution_jobs_state_check;

ALTER TABLE execution_jobs
    ADD CONSTRAINT execution_jobs_state_check CHECK (state IN ('queued', 'leased', 'run_started'));
