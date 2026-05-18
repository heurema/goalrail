-- +goose Up
CREATE TABLE runner_capability_reports (
    id UUID PRIMARY KEY,
    runner_id TEXT NOT NULL,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    repo_binding_id UUID NOT NULL REFERENCES repo_bindings(id) ON DELETE CASCADE,
    network_isolation_declared BOOLEAN NOT NULL DEFAULT FALSE,
    workspace_write_isolation_declared BOOLEAN NOT NULL DEFAULT FALSE,
    process_tree_control_declared BOOLEAN NOT NULL DEFAULT FALSE,
    stdout_stderr_policy_declared BOOLEAN NOT NULL DEFAULT FALSE,
    artifact_policy_declared BOOLEAN NOT NULL DEFAULT FALSE,
    trust_state TEXT NOT NULL,
    reported_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT runner_capability_reports_runner_id_check CHECK (runner_id <> ''),
    CONSTRAINT runner_capability_reports_trust_state_check CHECK (trust_state = 'self_declared_untrusted')
);

CREATE INDEX runner_capability_reports_scope_idx
    ON runner_capability_reports(organization_id, project_id, repo_binding_id, runner_id, reported_at DESC);

-- +goose Down
DROP INDEX IF EXISTS runner_capability_reports_scope_idx;
DROP TABLE IF EXISTS runner_capability_reports;
