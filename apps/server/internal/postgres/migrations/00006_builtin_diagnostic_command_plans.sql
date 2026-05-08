-- +goose Up
CREATE TABLE execution_command_plans (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    repo_binding_id UUID NOT NULL REFERENCES repo_bindings(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES work_items(id) ON DELETE CASCADE,
    checkout_receipt_id UUID NOT NULL REFERENCES checkout_receipts(id) ON DELETE CASCADE,
    execution_job_id UUID NOT NULL REFERENCES execution_jobs(id) ON DELETE CASCADE,
    run_id UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    command_kind TEXT NOT NULL,
    action TEXT NOT NULL,
    shell_allowed BOOLEAN NOT NULL DEFAULT FALSE,
    argv JSONB NOT NULL DEFAULT '[]'::jsonb,
    working_directory TEXT NOT NULL,
    path_scope JSONB NOT NULL DEFAULT '[]'::jsonb,
    timeout_seconds INTEGER NOT NULL,
    max_stdout_bytes INTEGER NOT NULL,
    max_stderr_bytes INTEGER NOT NULL,
    allowed_artifact_kinds JSONB NOT NULL DEFAULT '[]'::jsonb,
    raw_source_upload_allowed BOOLEAN NOT NULL DEFAULT FALSE,
    state TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT execution_command_plans_run_action_unique UNIQUE (run_id, command_kind, action),
    CONSTRAINT execution_command_plans_builtin_kind_check CHECK (command_kind = 'builtin_diagnostic'),
    CONSTRAINT execution_command_plans_workspace_status_check CHECK (action = 'workspace_status'),
    CONSTRAINT execution_command_plans_no_shell_check CHECK (shell_allowed = FALSE),
    CONSTRAINT execution_command_plans_no_argv_check CHECK (argv = '[]'::jsonb),
    CONSTRAINT execution_command_plans_workdir_check CHECK (working_directory = '.'),
    CONSTRAINT execution_command_plans_path_scope_check CHECK (path_scope = '["."]'::jsonb),
    CONSTRAINT execution_command_plans_timeout_check CHECK (timeout_seconds = 30),
    CONSTRAINT execution_command_plans_no_stdout_check CHECK (max_stdout_bytes = 0),
    CONSTRAINT execution_command_plans_no_stderr_check CHECK (max_stderr_bytes = 0),
    CONSTRAINT execution_command_plans_no_artifacts_check CHECK (allowed_artifact_kinds = '[]'::jsonb),
    CONSTRAINT execution_command_plans_no_raw_source_check CHECK (raw_source_upload_allowed = FALSE),
    CONSTRAINT execution_command_plans_state_check CHECK (state = 'planned')
);

CREATE INDEX execution_command_plans_run_id_idx
    ON execution_command_plans(run_id);

CREATE INDEX execution_command_plans_execution_job_id_idx
    ON execution_command_plans(execution_job_id);

CREATE INDEX execution_command_plans_repo_binding_id_idx
    ON execution_command_plans(repo_binding_id);

ALTER TABLE execution_receipts
    DROP CONSTRAINT execution_receipts_mode_check;

ALTER TABLE execution_receipts
    ADD COLUMN command_plan_id UUID NULL REFERENCES execution_command_plans(id) ON DELETE RESTRICT,
    ADD COLUMN command_kind TEXT NOT NULL DEFAULT '',
    ADD COLUMN action TEXT NOT NULL DEFAULT '',
    ADD COLUMN runner_started_at TIMESTAMPTZ NULL,
    ADD COLUMN runner_finished_at TIMESTAMPTZ NULL,
    ADD CONSTRAINT execution_receipts_mode_check CHECK (execution_mode IN ('no_command', 'builtin_diagnostic')),
    ADD CONSTRAINT execution_receipts_no_command_command_fields_check CHECK (
        execution_mode <> 'no_command'
        OR (command_plan_id IS NULL AND command_kind = '' AND action = '' AND runner_started_at IS NULL AND runner_finished_at IS NULL)
    ),
    ADD CONSTRAINT execution_receipts_builtin_diagnostic_check CHECK (
        execution_mode <> 'builtin_diagnostic'
        OR (
            command_plan_id IS NOT NULL
            AND command_kind = 'builtin_diagnostic'
            AND action = 'workspace_status'
            AND process_status = 'metadata_only'
            AND exit_code IS NULL
            AND artifact_refs = '[]'::jsonb
            AND changed_paths_summary = '[]'::jsonb
            AND raw_source_uploaded = FALSE
            AND runner_started_at IS NOT NULL
            AND runner_finished_at IS NOT NULL
            AND runner_finished_at >= runner_started_at
        )
    );

CREATE INDEX execution_receipts_command_plan_id_idx
    ON execution_receipts(command_plan_id);

-- +goose Down
DROP INDEX IF EXISTS execution_receipts_command_plan_id_idx;

DELETE FROM execution_receipts
WHERE execution_mode = 'builtin_diagnostic';

ALTER TABLE execution_receipts
    DROP CONSTRAINT IF EXISTS execution_receipts_builtin_diagnostic_check,
    DROP CONSTRAINT IF EXISTS execution_receipts_no_command_command_fields_check,
    DROP CONSTRAINT execution_receipts_mode_check,
    DROP COLUMN runner_finished_at,
    DROP COLUMN runner_started_at,
    DROP COLUMN action,
    DROP COLUMN command_kind,
    DROP COLUMN command_plan_id,
    ADD CONSTRAINT execution_receipts_mode_check CHECK (execution_mode = 'no_command');

DROP INDEX IF EXISTS execution_command_plans_repo_binding_id_idx;
DROP INDEX IF EXISTS execution_command_plans_execution_job_id_idx;
DROP INDEX IF EXISTS execution_command_plans_run_id_idx;
DROP TABLE IF EXISTS execution_command_plans;
