# Omnigent to Goalrail Rename Plan

## Decision

MODIFY the rename into staged compatibility-preserving blocks.

Do not mechanically replace every `omnigent` token. The current code uses
`omnigent` as package name, Python import root, CLI command, API schema label,
environment prefix, filesystem state directory, web app identity, Electron app
identity, iOS target/project identity, Docker/deploy identity, tests, and docs.
Those surfaces need different migration rules and different verification.

## Current inventory

Fresh codebase-memory full index was run for `/Users/vi/personal/heurema/goalrail`.
Raw exhaustive tracked-file inventory was then generated from `git ls-files`.

- Tracked files scanned: 2638
- Files with matched tokens: 1673
- Matched lines: 24927
- Total literal token occurrences: 26379
- Standalone word `omni` lines: 233

Token counts:

- `omnigent`: 18601
- `Omnigent`: 5386
- `OMNIGENT`: 2190
- `omnigent-ai`: 135
- `omnigents`: 21
- `omnigent.ai`: 16
- `OMNIAGENT`: 6
- `OMNIAGENTS`: 6
- `omniagent`: 5
- `omniagents`: 5
- `Omnigents`: 4
- `OMNIGENTS`: 4

Important compatibility note:

- There is no meaningful product-brand `OmniAgent` surface.
- `OMNIAGENTS_*`, `OMNIGENTS_*`, `.omniagents`, and `.omnigents` are legacy
  compatibility paths in environment/state migration code. Treat them as
  migration inputs, not current brand copy to delete blindly.

## Artifacts

- `inventory_summary.json`: machine-readable totals and top files.
- `inventory_occurrences.csv.gz`: every matched tracked-file line.
- `word_omni_occurrences.csv`: standalone `omni` command/word lines.
- `inventory_report.md`: compact human summary.

## Rename taxonomy

### A. Public brand copy

Examples:

- `README.md`
- docs under `docs/`
- deploy guides under `deploy/**/README.md`
- GitHub issue templates and marketplace-facing descriptions

Rule:

- `Omnigent` -> `Goalrail` where it is visible product prose.
- `omnigent.ai` -> `goalrail.dev`.
- `omnigent-ai/omnigent` links -> `heurema/goalrail` only when links point to
  files now owned by this repo.
- Do not change code examples that must keep current command/package names.

Verification:

- `rg -n "omnigent\\.ai|omnigent-ai/omnigent|Contributors|CONTRIBUTING|Omnigent" README.md docs deploy .github`
- Link sanity for touched docs.
- `git diff --check`
- pre-commit.

### B. Visible UI labels

Examples:

- `ap-web/src/**`
- `ap-web/index.html`
- `ap-web/electron/setup/index.html`
- `ap-web/electron/find/index.html`
- accessibility labels and page titles

Rule:

- Change user-visible `Omnigent` labels to `Goalrail`.
- Do not rename TypeScript APIs like `OmnigentHostConfig` in the same pass.
- Keep URL/base-path logic unchanged unless tests cover it.

Verification:

- `npm --prefix ap-web test -- --run` or the repo's existing web test command.
- Targeted tests from changed files.
- Build smoke if dependency state is available: `npm --prefix ap-web run build`.

### C. Desktop Electron identity

Examples:

- `ap-web/electron/package.json`
- `ap-web/electron/src/main.js`
- `ap-web/electron/icons/**`
- app display name, protocol names, bundle metadata

Rule:

- First pass: display name only.
- Later pass: app id/protocol/data-dir migration, with explicit compatibility
  for existing installed desktop users.

Verification:

- Electron URL tests.
- Package metadata inspection.
- If possible: local package/build smoke.

### D. iOS app identity

Examples:

- `ap-web/ios/Omnigent.xcodeproj`
- `ap-web/ios/Omnigent/**`
- `ap-web/ios/OmnigentTests/**`
- asset catalogs containing `OmnigentLogo`

Rule:

- First pass: visible strings and display name.
- Later pass: Xcode target/project/module rename. That requires project file
  updates, scheme updates, Swift module references, test target rename, asset
  catalog rename, bundle id review, and migration notes.

Verification:

- `ap-web/ios/bin/swift-format.sh format lint --strict --parallel`.
- Xcode project parse/build if local toolchain supports it.
- Swift unit tests if available.

### E. Runtime Python package and CLI

Examples:

- `omnigent/`
- `pyproject.toml`
- `scripts/install_oss.sh`
- `setup.py`
- `uv.lock`
- console scripts: `omnigent`, `omni`

Rule:

- Do not move `omnigent/` to `goalrail/` mechanically.
- Add compatibility first:
  - new `goalrail` console script can point to the same entrypoint;
  - keep `omnigent` and `omni` as aliases during transition;
  - package name rename requires a packaging/release decision and lock update;
  - Python import-root rename requires either import aliases or a larger move.

Verification:

- `python3 -m pytest --confcutdir=tests/scripts tests/scripts/test_install_oss.py -q`
- CLI smoke: `python -m omnigent --help` or installed entrypoint smoke.
- Update-check tests.
- Packaging metadata inspection.

### F. Environment variables and local state

Examples:

- `OMNIGENT_*`
- `OMNIGENTS_*`
- `OMNIAGENTS_*`
- `~/.omnigent`
- `~/.omnigents`
- `~/.omniagents`

Rule:

- Introduce `GOALRAIL_*` and `~/.goalrail`.
- Keep all old prefixes/dirs as compatibility inputs.
- Migration order should become:
  `.omniagents` -> `.omnigents` -> `.omnigent` -> `.goalrail`.
- New env var lookup should prefer `GOALRAIL_*`, then mirror old names.

Verification:

- Existing env compatibility tests.
- Add tests for `GOALRAIL_*` precedence.
- Add tests for state-dir migration into `~/.goalrail`.

### G. API schemas, OpenAPI, DB, server protocol

Examples:

- `openapi.json`
- `omnigent/server/**`
- `omnigent/entities/**`
- API event names and websocket payloads

Rule:

- Do not rename public API fields without compatibility mapping.
- Prefer server-internal display labels first.
- If API names must change, add dual-read / dual-write or explicit versioning.

Verification:

- OpenAPI drift tests.
- Server route tests.
- Backward compatibility tests for old client payloads.

### H. Deployment and infrastructure

Examples:

- `deploy/docker/**`
- `deploy/kubernetes/**`
- `deploy/fly/**`
- `deploy/render/**`
- `deploy/railway/**`
- GitHub release/publish workflows

Rule:

- Separate display/docs from image names, env vars, service names, k8s resource
  names, Helm/Kustomize labels, release artifact names.
- Resource renames can be breaking; treat them as deployment migration work.

Verification:

- YAML parse checks.
- Docker compose config check.
- Kustomize build for touched overlays if tooling is available.
- Workflow syntax/lint if available.

### I. Tests and fixtures

Examples:

- `tests/**`
- `tests/e2e/omnigent/**`
- snapshots and expected output fixtures

Rule:

- Tests should follow each functional block, not be renamed wholesale first.
- Rename test directories only after runtime paths are renamed or explicitly
  aliased.

Verification:

- Run the smallest related test subset after each block.
- End each milestone with broader pytest/web/iOS/deploy checks.

## Proposed execution sequence

### Milestone 1: public/docs + visible UI only

Goal: external reader sees Goalrail, while runtime still works as Omnigent.

Steps:

1. Docs batch:
   - README already started.
   - Update visible brand copy in `docs/`, `deploy/**/README.md`, GitHub issue
     templates.
   - Keep code examples that still require `omnigent` commands.
2. Web UI label batch:
   - Page titles, labels, aria labels, sidebar title, settings back links.
   - No TypeScript interface rename.
3. Electron display batch:
   - App display name only.
   - No app id/protocol/data-dir rename yet.
4. iOS display batch:
   - Display strings only.
   - No Xcode target/module/project rename yet.

Milestone 1 verification:

- `rg -n "Omnigent|omnigent\\.ai|omnigent-ai/omnigent" README.md docs deploy .github ap-web/src ap-web/index.html ap-web/electron ap-web/ios`
- Targeted web tests for changed UI components.
- Electron URL tests.
- Swift format lint if Swift files changed.
- `git diff --check`
- pre-commit.

### Milestone 2: new aliases, no removals

Goal: users can start using `goalrail` names while old `omnigent` names still work.

Steps:

1. Add `goalrail` console script alias to the existing CLI entrypoint.
2. Add `GOALRAIL_*` env prefix support, with old prefixes retained.
3. Add `~/.goalrail` state dir and migration from old dirs.
4. Add tests for alias/env/state precedence.

Milestone 2 verification:

- CLI help smoke for `goalrail`, `omnigent`, `omni`.
- Env compatibility tests.
- State migration tests.
- Installer tests.
- Update check tests.

### Milestone 3: packaging and installer rename

Goal: installer and packaging story become Goalrail-first.

Steps:

1. Decide whether PyPI package remains `omnigent` temporarily or becomes
   `goalrail`.
2. Update installer copy and optional package targets accordingly.
3. Update release workflows, version updater, lockfiles, Homebrew notes.
4. Preserve old install instructions as transitional if still supported.

Milestone 3 verification:

- Installer tests.
- Packaging metadata build smoke.
- Lockfile normalization.
- Release workflow lint/static checks.

### Milestone 4: module/import-root migration

Goal: code import root becomes `goalrail` without breaking old importers.

Steps:

1. Add `goalrail` package facade or move package with `omnigent` compatibility
   shim.
2. Update internal imports in controlled slices.
3. Update policy handler strings and YAML examples with compatibility mapping.
4. Update tests per slice.

Milestone 4 verification:

- Full Python test subset by area.
- Import smoke for both `import goalrail` and `import omnigent`.
- Policy handler resolution tests.
- OpenAPI drift tests.

### Milestone 5: platform identity migration

Goal: desktop/iOS/deploy identities become Goalrail-first.

Steps:

1. Electron app id, protocol, data dir, package metadata migration.
2. iOS project/target/module/test target/asset rename.
3. Docker image/service/k8s resource name rename with deployment migration notes.

Milestone 5 verification:

- Electron package/build smoke.
- iOS project build/test where possible.
- Docker compose config and kustomize overlays.
- Deployment docs updated with rollback notes.

### Milestone 6: cleanup and deprecation

Goal: old Omnigent names remain only in compatibility/deprecation docs and tests.

Steps:

1. Inventory diff: regenerate `inventory_summary.json`.
2. Classify remaining hits as compatibility-allowed or cleanup-required.
3. Add a guard script/check for new forbidden public `Omnigent` occurrences.
4. Decide when/if to remove old aliases.

Milestone 6 verification:

- Regenerated inventory count drops to allowed-list only.
- Guard script passes.
- Full available test matrix or CI.

## Immediate next step

Start Milestone 1 with the safest batch:

1. Update docs/deploy/GitHub visible copy and links.
2. Do not touch runtime package/import names.
3. Commit that batch.
4. Then move to web UI labels as a separate commit.
