-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY,
    display_name TEXT NOT NULL,
    email TEXT NOT NULL DEFAULT '',
    state TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE organizations (
    id UUID PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    state TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE organization_memberships (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    state TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT organization_memberships_role_check
        CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    CONSTRAINT organization_memberships_org_user_unique
        UNIQUE (organization_id, user_id)
);

CREATE TABLE projects (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_by_user_id UUID NOT NULL REFERENCES users(id),
    slug TEXT NOT NULL,
    display_name TEXT NOT NULL,
    state TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT projects_org_slug_unique UNIQUE (organization_id, slug)
);

CREATE TABLE repo_bindings (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    created_by_user_id UUID NOT NULL REFERENCES users(id),
    vcs_connection_id TEXT NOT NULL DEFAULT '',
    provider TEXT NOT NULL,
    repository_external_id TEXT NOT NULL DEFAULT '',
    repository_full_name TEXT NOT NULL,
    repository_url TEXT NOT NULL,
    default_branch TEXT NOT NULL,
    path_scope TEXT NOT NULL,
    access_mode TEXT NOT NULL,
    state TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT repo_bindings_access_mode_check
        CHECK (access_mode IN (
            'provider_token_checkout',
            'customer_runner_checkout',
            'customer_mounted_workspace',
            'metadata_only'
        ))
);

CREATE INDEX repo_bindings_project_id_idx ON repo_bindings(project_id);

CREATE UNIQUE INDEX repo_bindings_one_active_per_project_idx
    ON repo_bindings(project_id)
    WHERE state = 'active';

-- +goose Down
DROP INDEX IF EXISTS repo_bindings_one_active_per_project_idx;
DROP INDEX IF EXISTS repo_bindings_project_id_idx;
DROP TABLE IF EXISTS repo_bindings;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS organization_memberships;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS users;
