# Publishing Migration Plan

## Status
- **Status:** repo_cleanup_complete
- Repo-local legacy directory `.punk/publishing/` has been removed.
- First-pass external copy preserved the legacy tree shape.
- `.punk/publishing.toml` is the committed binding manifest.
- **Manual Bootstrap:** Optional `.punk/publishing.local.toml` is supported; runtime support is pending.

## Goal
- Move runtime publishing artifacts (drafts, assets, prompts, receipts) out of the project repository.
- Preserve only the repo-local binding manifest (`.punk/publishing.toml`) in the repository.
- Avoid committing OS-specific physical paths.
- Avoid using symlinks as a workspace solution.
- Keep the migration process reversible, auditable, and verifiable.

## Current legacy tree
Top-level directories under `.punk/publishing/`:
- `assets/` — image assets (e.g., LinkedIn banners, avatars).
- `posts/` — draft and ready-to-post content.
- `prompts/` — image generation and text narrative prompts.
- `published/` — receipts and records of published content.
- `stories/` — multi-post narrative series.
- `styles/` — narrative and visual style definitions.

## First-pass migration rule
- **Preserve tree shape:** The first migration must preserve the legacy directory structure exactly as it is in the repository.
- **No early renaming:** Do not rename `published/` to `receipts/` at this stage.
- **No semantic splitting:** Do not split `styles/` or `prompts/` yet.
- Semantic cleanup and normalization happen only after a successful copy and verification phase.

## First-pass classification
| Directory | Action | Notes |
| :--- | :--- | :--- |
| `assets/` | `move_to_workspace` | Binary and source image assets. |
| `posts/` | `move_to_workspace` | Markdown drafts and final posts. |
| `prompts/` | `move_to_workspace` | Narrative and image prompts. |
| `published/` | `move_to_workspace` | Historical receipts and published markdown. |
| `stories/` | `move_to_workspace` | Narrative series drafts. |
| `styles/` | `requires_review` | Some styles may be candidates for `docs/brand/` promotion. |

## Proposed external workspace logical shape
The resolved workspace should follow a project-isolated logical structure:
```text
publishing/projects/goalrail/
  workspace.toml  # local workspace metadata
  assets/
  posts/
  prompts/
  published/
  stories/
  styles/
  metrics/
  migration/      # migration logs and receipts
```

## Resolver contract
Agents and tools must use the machine contract specified in [PUBLISHING_RESOLVER_CONTRACT.md](PUBLISHING_RESOLVER_CONTRACT.md):
```bash
punk publishing locate --project-root . --json
```

The `locate` command is side-effect free and read-only by default.

**Note:** Physical paths are platform-native and resolver-owned. They must never be committed into repository documentation or manifests.

## Migration command model
Proposed migration workflow:
1. `punk publishing inventory --project-root . --json` — verify what will be moved.
2. `punk publishing migrate --project-root . --dry-run` — simulate the migration.
3. `punk publishing migrate --project-root . --apply` — perform the copy operation.
4. `punk publishing migrate --project-root . --verify` — compare repo vs workspace content.

**Rules:**
- Always run `dry-run` before `apply`.
- Copy artifacts before any removal.
- Verify integrity before repository cleanup.
- Repository cleanup (removal of `.punk/publishing/`) is a separate, reviewed commit.

## Safety gates
- **No secrets:** No API keys, session tokens, or account passwords in the repository.
- **No credentials:** No `credentials.json` or platform-specific login artifacts in the repository.
- **No browser sessions:** No puppeteer/playwright session data or cookies in the repository.
- **No platform cache:** No transient runtime cache files in the repository.
- **No symlinks:** Symlinks are intentionally not used as a workspace bridge.
- **No OS-specific paths:** No absolute or home-relative paths in repository files.
- **Integrity first:** Do not delete legacy content until copy verification passes.

## Post-migration review
A second-pass review will address semantic organization:
- `styles/` may be split into product/public canon (moved to `docs/brand/`), operator-specific profiles, and generated host wrappers.
- `prompts/` may be split into reusable system profiles, generated host wrappers, and an execution archive.
- `published/` may be normalized into a strict `receipts/published` model.

## Open questions
- Final naming convention: `Punk` vs `PubPunk` for the CLI.
- Current status of the resolver implementation in the runtime.
- Exact `workspace.toml` schema for external storage.
- Selection of specific style files for promotion into `docs/product/` or `docs/brand/`.
