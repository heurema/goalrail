-- +goose Up
ALTER TABLE execution_command_plans
    DROP CONSTRAINT execution_command_plans_builtin_kind_check,
    DROP CONSTRAINT execution_command_plans_workspace_status_check,
    ADD CONSTRAINT execution_command_plans_kind_action_check CHECK (
        (command_kind = 'builtin_diagnostic' AND action = 'workspace_status')
        OR (command_kind = 'project_probe' AND action = 'detect_declared_test_targets')
    );

ALTER TABLE execution_receipts
    DROP CONSTRAINT execution_receipts_builtin_diagnostic_check,
    DROP CONSTRAINT execution_receipts_no_command_command_fields_check,
    DROP CONSTRAINT execution_receipts_mode_check,
    ADD COLUMN project_probe_metadata JSONB NOT NULL DEFAULT '{
        "detected_manifests": [],
        "package_manager_candidates": [],
        "declared_test_target_candidates": [],
        "unsupported_or_unknowns": [],
        "partiality_reasons": []
    }'::jsonb,
    ADD CONSTRAINT execution_receipts_mode_check CHECK (execution_mode IN ('no_command', 'builtin_diagnostic', 'project_probe')),
    ADD CONSTRAINT execution_receipts_no_command_command_fields_check CHECK (
        execution_mode <> 'no_command'
        OR (
            command_plan_id IS NULL
            AND command_kind = ''
            AND action = ''
            AND runner_started_at IS NULL
            AND runner_finished_at IS NULL
            AND project_probe_metadata = '{
                "detected_manifests": [],
                "package_manager_candidates": [],
                "declared_test_target_candidates": [],
                "unsupported_or_unknowns": [],
                "partiality_reasons": []
            }'::jsonb
        )
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
            AND project_probe_metadata = '{
                "detected_manifests": [],
                "package_manager_candidates": [],
                "declared_test_target_candidates": [],
                "unsupported_or_unknowns": [],
                "partiality_reasons": []
            }'::jsonb
        )
    ),
    ADD CONSTRAINT execution_receipts_project_probe_check CHECK (
        execution_mode <> 'project_probe'
        OR (
            command_plan_id IS NOT NULL
            AND command_kind = 'project_probe'
            AND action = 'detect_declared_test_targets'
            AND process_status = 'metadata_only'
            AND exit_code IS NULL
            AND artifact_refs = '[]'::jsonb
            AND changed_paths_summary = '[]'::jsonb
            AND raw_source_uploaded = FALSE
            AND runner_started_at IS NOT NULL
            AND runner_finished_at IS NOT NULL
            AND runner_finished_at >= runner_started_at
            AND jsonb_typeof(project_probe_metadata) = 'object'
            AND project_probe_metadata ? 'detected_manifests'
            AND project_probe_metadata ? 'package_manager_candidates'
            AND project_probe_metadata ? 'declared_test_target_candidates'
            AND project_probe_metadata ? 'unsupported_or_unknowns'
            AND project_probe_metadata ? 'partiality_reasons'
            AND jsonb_typeof(project_probe_metadata -> 'detected_manifests') = 'array'
            AND jsonb_typeof(project_probe_metadata -> 'package_manager_candidates') = 'array'
            AND jsonb_typeof(project_probe_metadata -> 'declared_test_target_candidates') = 'array'
            AND jsonb_typeof(project_probe_metadata -> 'unsupported_or_unknowns') = 'array'
            AND jsonb_typeof(project_probe_metadata -> 'partiality_reasons') = 'array'
        )
    );

-- +goose Down
DELETE FROM execution_receipts
WHERE execution_mode = 'project_probe';

DELETE FROM execution_command_plans
WHERE command_kind = 'project_probe';

ALTER TABLE execution_receipts
    DROP CONSTRAINT execution_receipts_project_probe_check,
    DROP CONSTRAINT execution_receipts_builtin_diagnostic_check,
    DROP CONSTRAINT execution_receipts_no_command_command_fields_check,
    DROP CONSTRAINT execution_receipts_mode_check,
    DROP COLUMN project_probe_metadata,
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

ALTER TABLE execution_command_plans
    DROP CONSTRAINT execution_command_plans_kind_action_check,
    ADD CONSTRAINT execution_command_plans_builtin_kind_check CHECK (command_kind = 'builtin_diagnostic'),
    ADD CONSTRAINT execution_command_plans_workspace_status_check CHECK (action = 'workspace_status');
