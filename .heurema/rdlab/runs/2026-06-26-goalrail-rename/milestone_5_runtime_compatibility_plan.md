# Milestone 5: Runtime Rename Compatibility Plan

Date: 2026-06-26

## Verdict

MODIFY the runtime rename into additive compatibility work.

Do not rename `goalrail/`, SDK packages, Docker images, storage paths, workflow
names, or native project files mechanically. The next implementation work
should introduce `Goalrail` runtime surfaces as aliases first, keep existing
`goalrail` surfaces working, and only remove legacy names after a release and
deprecation plan exists.

## Current Evidence

Sources inspected:

- codebase-memory index for `Users-vi-personal-heurema-goalrail`: ready
  (`46230` nodes, `261585` edges).
- `pyproject.toml`: package name is still `goalrail`; console scripts are
  `goalrail` and `goalrail`; installed package discovery is `goalrail*`.
- `goalrail/_env_compat.py`: current env compatibility mirrors
  `GOALRAIL_*` / `GOALRAIL_*` into `GOALRAIL_*`.
- codebase-memory `search_code` for `GOALRAIL_`: no matches.
- codebase-memory `search_code` for `GOALRAIL_`: broad runtime, deploy, CI,
  web, and test usage.
- SDK surfaces: `goalrail-client`, `goalrail-ui-sdk`, `goalrail_client`,
  `GoalrailClient`.
- Deploy surfaces: `ghcr.io/heurema/goalrail-server`,
  `ghcr.io/heurema/goalrail-host`, Kubernetes service/database examples,
  Fly/Railway/Render/Modal/Cloudflare references.
- Storage/native surfaces: `~/.goalrail`, `.goalrail`, `.goalrail`,
  `ap-web/ios/Goalrail.xcodeproj`, `GoalrailLogo`, Electron/iOS app metadata,
  and web `localStorage` keys such as `goalrail:panel-size-preferences`.

## Runtime Naming Policy

Canonical product/display name:

- Use `Goalrail` for user-visible product labels.

Compatibility names:

- Keep `goalrail` CLI, Python import root, PyPI package, SDK package names,
  deploy image names, existing storage paths, and historical links until each
  surface has a specific alias/migration implementation and tests.
- Keep `goalrail` as a short CLI alias unless a separate product decision removes
  it.

New names:

- Introduce `goalrail` CLI as an additive alias to `goalrail.cli:main`.
- Introduce `GOALRAIL_*` env variables as the new preferred names while keeping
  `GOALRAIL_*`, `GOALRAIL_*`, and `GOALRAIL_*` as compatibility inputs.
- Introduce `~/.goalrail` only with a read-old/write-new migration policy.

## Compatibility Rules

### CLI

Rule:

- Add `goalrail = "goalrail.cli:main"` in package metadata.
- Keep `goalrail` and `goalrail`.
- Do not rename modules, tests, or internal command references in the same PR.

Required tests:

- Installed script metadata includes `goalrail`, `goalrail`, and `goalrail`.
- `goalrail --help`, `goalrail --help`, and `goalrail --help` reach the same CLI.
- Existing install script tests still pass.

### Environment Variables

Rule:

- Update the env compatibility layer so canonical lookups can prefer
  `GOALRAIL_*`.
- Preserve old values by mirroring old prefixes into the new prefix only when
  the new value is absent.
- Conflict precedence should be:
  1. explicit `GOALRAIL_*`
  2. explicit `GOALRAIL_*`
  3. legacy `GOALRAIL_*`
  4. legacy `GOALRAIL_*`

Implementation shape:

- Extend `goalrail/_env_compat.py`; do not scatter one-off fallback checks.
- Keep direct readers safe by calling the shim before modules read env.
- For deploy entrypoints that read env before importing `goalrail`, keep direct
  calls to the shim.

Required tests:

- `GOALRAIL_*` wins over `GOALRAIL_*`.
- `GOALRAIL_*` still works when `GOALRAIL_*` is absent.
- `GOALRAIL_*` and `GOALRAIL_*` still mirror correctly.
- Existing tests using `GOALRAIL_CONFIG_HOME`, `GOALRAIL_SKIP_WEB_UI`,
  `GOALRAIL_DISABLE_KEYRING`, auth, runner, and deploy env continue to pass.

### Config And User State

Rule:

- Do not move or delete user files automatically.
- Add `GOALRAIL_CONFIG_HOME` / `~/.goalrail` support first.
- Read existing `~/.goalrail`, `~/.goalrail`, and `~/.goalrail` as legacy
  inputs.
- Write new files to `~/.goalrail` only after migration behavior is explicit.

Migration policy:

- First release: dual-read, old-write remains acceptable.
- Later release: read-old/write-new with a backup or copy step.
- Final removal: only after a documented major-version window.

Required tests:

- Fresh user writes to the expected configured path.
- Existing `~/.goalrail/config.yaml` is read when `~/.goalrail/config.yaml` is
  absent.
- New config takes precedence over legacy config when both exist.
- Tests do not touch the developer's real home directory.

### Web Storage

Rule:

- Keep existing `goalrail:*` `localStorage` keys until migration code exists.
- Add `goalrail:*` keys with read-old/write-new behavior for user preferences.
- Do not reset panel sizes, remembered usernames, or session UI state during the
  rename.

Required tests:

- Old key values are read.
- New key values take precedence.
- Corrupt old values do not block app boot.

### Python Package And Imports

Rule:

- Do not rename `goalrail/` to `goalrail/` in the next implementation PR.
- Treat Python import-root rename as a major migration.
- If a `goalrail` import package is introduced, it should be a thin compatibility
  re-export first, with tests proving existing `goalrail` imports still work.

Required tests before any package move:

- `import goalrail` still works.
- Any new `import goalrail` shim works.
- build metadata, package data, type-check allowlists, and resource paths remain
  coherent.
- CLI, server, runner, SDK, and onboarding tests pass in the same branch.

### SDK Packages

Rule:

- Do not rename `goalrail-client`, `goalrail-ui-sdk`, `goalrail_client`, or
  `GoalrailClient` until package distribution names are reserved and tested.
- Prefer new SDK aliases (`goalrail-client`, `goalrail_client`,
  `GoalrailClient`) only as additive wrappers first.

Required tests:

- Existing SDK imports and docs examples still run.
- New SDK alias imports map to the same implementation.
- Databricks wheel build and release-version pin checks still pass.

### Deploy Images And Infrastructure

Rule:

- Do not change image references until new GHCR image names are published.
- Keep `ghcr.io/heurema/goalrail-*` deploy docs as legacy runtime truth
  until replacement images exist.
- Introduce `ghcr.io/heurema/goalrail-*` or another chosen namespace as
  mirrored images first, not as a docs-only claim.

Required tests:

- CI publishes or can pull the new image refs.
- Existing deploy templates still work with old images.
- Kubernetes, Fly, Railway, Render, Modal, Cloudflare, and Docker Compose docs
  match real published images.

### Native App Identity

Rule:

- Display strings can say `Goalrail`.
- Do not rename `Goalrail.xcodeproj`, targets, schemes, bundle ids, app group
  ids, app protocols, app data paths, or asset catalog names without a native
  migration PR.

Required tests:

- iOS project parses/builds.
- Swift tests and formatting pass where available.
- Electron app package metadata and user-data path behavior are verified.
- Existing installed app data is not lost.

### Historical And External Names

Rule:

- Keep historical GitHub links, issue/PR refs, bot identities, noreply emails,
  and old org references when they point to immutable history.
- Only update links that now have a real replacement in this repository or a
  live replacement service.

## Implementation Order

1. Add CLI alias `goalrail`.
   - Files likely touched: `pyproject.toml`, install/CLI tests, possibly README
     command examples only after the alias works.
   - Gate: `goalrail --help` and `goalrail --help` both work.

2. Add env alias support.
   - Files likely touched: `goalrail/_env_compat.py`, targeted env tests,
     deploy entrypoint tests.
   - Gate: precedence tests prove `GOALRAIL_*` wins and all old prefixes still
     work.

3. Add config/state path compatibility.
   - Files likely touched: onboarding config/secrets/provider modules and tests.
   - Gate: old config homes are read, new config homes win, no real home writes.

4. Add web storage migration.
   - Files likely touched: `ap-web/src/lib/*`, `ap-web/src/store/*`, tests.
   - Gate: old UI preference keys survive the rename.

5. Decide SDK distribution names.
   - Files likely touched only after names are reserved: `sdks/**`,
     `pyproject.toml`, release workflow, Databricks deploy scripts.
   - Gate: old and new SDK imports both work.

6. Publish/mirror deploy images.
   - Files likely touched: `.github/workflows/oss-publish-images.yml`,
     deploy templates, deploy docs.
   - Gate: new image refs are real and old refs remain usable.

7. Consider package/import-root rename.
   - Only after steps 1-6 are stable.
   - Treat as major-version migration unless the compatibility shim is complete
     and low-risk.

## Stop Conditions

Stop and re-plan if any implementation diff:

- removes `goalrail` CLI support;
- breaks `GOALRAIL_*` env variables;
- moves user data without backup or fallback;
- changes published image names before images exist;
- renames Python modules without import compatibility;
- changes API/schema/event names without a compatibility map;
- updates public docs to claim a package, image, command, or app id exists
  before it is implemented.

## Rollback

The next implementation PRs should be additive. Rollback should be possible by:

- removing the new `goalrail` console script;
- removing `GOALRAIL_*` alias reads;
- keeping all existing `goalrail` files, commands, packages, env names, storage
  paths, and deploy images untouched.

Do not perform destructive migrations or one-way file moves in the first
runtime rename PR.

## Verification For This Plan

Commands:

```sh
codebase-memory-mcp cli index_status '{"project":"Users-vi-personal-heurema-goalrail"}'
codebase-memory-mcp cli search_code '{"project":"Users-vi-personal-heurema-goalrail","pattern":"GOALRAIL_","limit":80}'
codebase-memory-mcp cli search_code '{"project":"Users-vi-personal-heurema-goalrail","pattern":"GOALRAIL_","limit":120}'
rg -n "localStorage|goalrail:|\\.goalrail|\\.goalrail|\\.goalrail|Goalrail\\.xcodeproj|GoalrailLogo|ghcr\\.io/heurema|goalrail_client|GoalrailClient|goalrail-ui-sdk|goalrail-client|release-goalrail" ap-web goalrail sdks deploy .github pyproject.toml setup.py tests
git diff --check
```

Expected result:

- This milestone changes only the plan artifact.
- No runtime files, package metadata, deploy templates, SDK packages, or native
  project files are changed in this milestone.
