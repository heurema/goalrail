-- +goose Up
ALTER TABLE execution_receipts
    DROP CONSTRAINT execution_receipts_project_probe_check,
    DROP CONSTRAINT execution_receipts_builtin_diagnostic_check,
    DROP CONSTRAINT execution_receipts_no_command_command_fields_check,
    DROP CONSTRAINT execution_receipts_mode_check,
    DROP CONSTRAINT execution_receipts_process_status_check,
    ADD COLUMN enforcement_report JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE execution_receipts
SET enforcement_report = '{
    "network_policy": "disabled_required",
    "network_enforcement": "unavailable",
    "workspace_write_policy": "disabled_required",
    "workspace_write_enforcement": "unavailable",
    "process_tree_enforcement": "unavailable",
    "scratch_write_policy": "allowed_runner_local",
    "decision": "policy_rejected",
    "reason": "enforcement_unavailable"
}'::jsonb
WHERE execution_mode = 'project_test'
  AND process_status = 'policy_rejected'
  AND enforcement_report = '{}'::jsonb;

ALTER TABLE execution_receipts
    ADD CONSTRAINT execution_receipts_mode_check CHECK (execution_mode IN ('no_command', 'builtin_diagnostic', 'project_probe', 'project_test')),
    ADD CONSTRAINT execution_receipts_process_status_check CHECK (process_status IN ('not_executed', 'metadata_only', 'policy_rejected')),
    ADD CONSTRAINT execution_receipts_no_command_command_fields_check CHECK (
        execution_mode <> 'no_command'
        OR (
            command_plan_id IS NULL
            AND command_kind = ''
            AND action = ''
            AND process_status IN ('not_executed', 'metadata_only')
            AND runner_started_at IS NULL
            AND runner_finished_at IS NULL
            AND project_probe_metadata = '{
                "detected_manifests": [],
                "package_manager_candidates": [],
                "declared_test_target_candidates": [],
                "unsupported_or_unknowns": [],
                "partiality_reasons": []
            }'::jsonb
            AND enforcement_report = '{}'::jsonb
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
            AND enforcement_report = '{}'::jsonb
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
            AND enforcement_report = '{}'::jsonb
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
    ),
    ADD CONSTRAINT execution_receipts_project_test_check CHECK (
        execution_mode <> 'project_test'
        OR (
            command_plan_id IS NOT NULL
            AND command_kind = 'project_test'
            AND action = 'run_declared_test_target'
            AND process_status = 'policy_rejected'
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
            AND (
                enforcement_report = '{
                    "network_policy": "disabled_required",
                    "network_enforcement": "unavailable",
                    "workspace_write_policy": "disabled_required",
                    "workspace_write_enforcement": "unavailable",
                    "process_tree_enforcement": "unavailable",
                    "decision": "policy_rejected",
                    "reason": "enforcement_unavailable"
                }'::jsonb
                OR enforcement_report = '{
                    "network_policy": "disabled_required",
                    "network_enforcement": "unavailable",
                    "workspace_write_policy": "disabled_required",
                    "workspace_write_enforcement": "unavailable",
                    "process_tree_enforcement": "unavailable",
                    "scratch_write_policy": "allowed_runner_local",
                    "decision": "policy_rejected",
                    "reason": "enforcement_unavailable"
                }'::jsonb
            )
        )
    );

-- +goose Down
DELETE FROM execution_receipts
WHERE execution_mode = 'project_test';

ALTER TABLE execution_receipts
    DROP CONSTRAINT execution_receipts_project_test_check,
    DROP CONSTRAINT execution_receipts_project_probe_check,
    DROP CONSTRAINT execution_receipts_builtin_diagnostic_check,
    DROP CONSTRAINT execution_receipts_no_command_command_fields_check,
    DROP CONSTRAINT execution_receipts_mode_check,
    DROP CONSTRAINT execution_receipts_process_status_check,
    DROP COLUMN enforcement_report,
    ADD CONSTRAINT execution_receipts_mode_check CHECK (execution_mode IN ('no_command', 'builtin_diagnostic', 'project_probe')),
    ADD CONSTRAINT execution_receipts_process_status_check CHECK (process_status IN ('not_executed', 'metadata_only')),
    ADD CONSTRAINT execution_receipts_no_command_command_fields_check CHECK (
        execution_mode <> 'no_command'
        OR (
            command_plan_id IS NULL
            AND command_kind = ''
            AND action = ''
            AND process_status IN ('not_executed', 'metadata_only')
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
