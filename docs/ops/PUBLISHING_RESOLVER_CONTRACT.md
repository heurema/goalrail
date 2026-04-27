# Publishing Resolver Contract

## Status

- Status: planned
- Implementation may be pending.
- This document specifies the machine contract for agents and tools.

## Command

Primary command:

```bash
punk publishing locate --project-root . --json
```

## Exit Codes

- `0`: Success. Manifest found, physical paths computed.
- `1`: Binding manifest (`.punk/publishing.toml`) not found in project root.
- `2`: Invalid manifest schema or corrupted data.
- `3`: Project root path is invalid or inaccessible.

## JSON Output Schema

The command must return a single JSON object:

```json
{
  "schema_version": "punk.publishing.resolver.v1",
  "project_id": "goalrail",
  "workspace_ref": "punk-publishing://project/goalrail",
  "binding": {
    "manifest_path": ".punk/publishing.toml",
    "found": true
  },
  "workspace": {
    "exists": false,
    "migration_required": true,
    "logical_paths": {
      "root": "<physical-path>",
      "config": "<physical-path>",
      "data": "<physical-path>",
      "state": "<physical-path>",
      "cache": "<physical-path>"
    }
  },
  "legacy": {
    "repo_local_path": ".punk/publishing/",
    "exists": true
  }
}
```

## Storage Classes (Logical)

- `root`: The base directory of the project-isolated external workspace.
- `config`: User-level tool settings and operator profiles.
- `data`: Primary content (posts, assets, prompts, receipts, metrics).
- `state`: Transient runtime metadata. Not for browser or account sessions.
- `cache`: Volatile platform-specific cache.

## Platform Resolution Rules

- Physical paths are platform-native and computed at runtime.
- Use native application data and cache locations for the host OS.
- Never commit expanded physical paths into repository files.
- Never assume physical paths in binding manifests.


## Local Overrides (Manual Bootstrap)

In environments where the system-wide resolver is not yet implemented or active, agents/tools may use an optional, ignored local file:

- **Path:** `.punk/publishing.local.toml`
- **Purpose:** Per-machine pointer to an external workspace.
- **Rule:** This file is not part of the project truth and must be ignored by Git.

### Validation and Precedence

- **Project Identity:** `.punk/publishing.local.toml` must not override the `project_id` from the committed `.punk/publishing.toml`.
- **Reference Check:** Implementations should validate `workspace_ref` against the committed binding when present in the local pointer.
- **Initialization:** A future `init` command may claim or import a manual workspace only after explicit user confirmation.
- **Conflict Resolution:** When both a resolver-managed workspace and a local pointer exist, the precedence rules must be established before performing any migration operations.

Suggested logical shape for manual bootstrap:
```toml
schema_version = "punk.publishing.local.v1"

workspace_ref = "punk-publishing://project/goalrail"
workspace_root = "<external-workspace-path-on-this-machine>"

mode = "manual-bootstrap"
resolver_status = "pending"
```

## Related Commands

Planned commands for publishing lifecycle:
```bash
punk publishing init --project-root .
punk publishing inventory --project-root . --json
punk publishing migrate --project-root . --dry-run
punk publishing migrate --project-root . --apply
punk publishing migrate --project-root . --verify
```

## Non-Goals

- Does not create directories or files (read-only).
- Does not perform migration (side-effect free).
- Does not store or manage credentials, secrets, or browser sessions.
- Does not use or require symlinks.
