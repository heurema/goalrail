-- +goose Up
ALTER TABLE execution_command_plans
    DROP CONSTRAINT execution_command_plans_kind_action_check,
    DROP CONSTRAINT execution_command_plans_timeout_check,
    ADD COLUMN source_project_probe_receipt_id UUID NULL REFERENCES execution_receipts(id) ON DELETE RESTRICT,
    ADD COLUMN selected_target_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN declared_test_target JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN network_allowed BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN workspace_write_allowed BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN scratch_write_allowed BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN changed_paths_allowed BOOLEAN NOT NULL DEFAULT FALSE,
    ADD CONSTRAINT execution_command_plans_kind_action_check CHECK (
        (command_kind = 'builtin_diagnostic' AND action = 'workspace_status')
        OR (command_kind = 'project_probe' AND action = 'detect_declared_test_targets')
        OR (command_kind = 'project_test' AND action = 'run_declared_test_target')
    ),
    ADD CONSTRAINT execution_command_plans_timeout_check CHECK (
        (
            command_kind IN ('builtin_diagnostic', 'project_probe')
            AND timeout_seconds = 30
        )
        OR (
            command_kind = 'project_test'
            AND timeout_seconds = 120
        )
    ),
    ADD CONSTRAINT execution_command_plans_non_test_source_check CHECK (
        command_kind = 'project_test'
        OR (
            source_project_probe_receipt_id IS NULL
            AND selected_target_id = ''
            AND declared_test_target = '{}'::jsonb
        )
    ),
    ADD CONSTRAINT execution_command_plans_project_test_check CHECK (
        command_kind <> 'project_test'
        OR (
            source_project_probe_receipt_id IS NOT NULL
            AND selected_target_id <> ''
            AND jsonb_typeof(declared_test_target) = 'object'
            AND declared_test_target ? 'name'
            AND declared_test_target ? 'source_path'
            AND declared_test_target ? 'source_kind'
            AND declared_test_target ->> 'source_kind' = 'package_json_script'
            AND shell_allowed = FALSE
            AND argv = '[]'::jsonb
            AND working_directory = '.'
            AND path_scope = '["."]'::jsonb
            AND network_allowed = FALSE
            AND workspace_write_allowed = FALSE
            AND scratch_write_allowed = FALSE
            AND max_stdout_bytes = 0
            AND max_stderr_bytes = 0
            AND allowed_artifact_kinds = '[]'::jsonb
            AND changed_paths_allowed = FALSE
            AND raw_source_upload_allowed = FALSE
        )
    );

CREATE INDEX execution_command_plans_source_project_probe_receipt_id_idx
    ON execution_command_plans(source_project_probe_receipt_id)
    WHERE source_project_probe_receipt_id IS NOT NULL;

-- +goose Down
DELETE FROM execution_command_plans
WHERE command_kind = 'project_test';

DROP INDEX IF EXISTS execution_command_plans_source_project_probe_receipt_id_idx;

ALTER TABLE execution_command_plans
    DROP CONSTRAINT execution_command_plans_project_test_check,
    DROP CONSTRAINT execution_command_plans_non_test_source_check,
    DROP CONSTRAINT execution_command_plans_timeout_check,
    DROP CONSTRAINT execution_command_plans_kind_action_check,
    DROP COLUMN changed_paths_allowed,
    DROP COLUMN scratch_write_allowed,
    DROP COLUMN workspace_write_allowed,
    DROP COLUMN network_allowed,
    DROP COLUMN declared_test_target,
    DROP COLUMN selected_target_id,
    DROP COLUMN source_project_probe_receipt_id,
    ADD CONSTRAINT execution_command_plans_kind_action_check CHECK (
        (command_kind = 'builtin_diagnostic' AND action = 'workspace_status')
        OR (command_kind = 'project_probe' AND action = 'detect_declared_test_targets')
    ),
    ADD CONSTRAINT execution_command_plans_timeout_check CHECK (timeout_seconds = 30);
