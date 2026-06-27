# Milestone 8: Goalrail Runtime Data Home Compatibility

Date: 2026-06-26

## Scope

Implemented the next bounded data/state compatibility layer for the rebrand.
This does not physically migrate user data and does not change the fresh-write
default yet.

Changed runtime surfaces:

- `goalrail/_env_compat.py`
- `goalrail/host/local_server.py`
- `goalrail/chat.py`
- `goalrail/host/connect.py`
- `goalrail/server/admin_list.py`
- `goalrail/cli.py`

Changed tests:

- `tests/test_env_compat.py`
- `tests/test_data_home_paths.py`
- `tests/host/test_local_server.py`
- `tests/host/test_connect.py`
- `tests/server/test_admin_list.py`
- `tests/cli/test_cli.py`

## Compatibility Policy

Resolution order for runtime data home is:

1. `GOALRAIL_DATA_DIR`
2. `GOALRAIL_DATA_DIR`
3. existing `~/.goalrail`
4. existing `~/.goalrail`
5. existing `~/.goalrail`
6. existing `~/.goalrail`
7. fresh default `~/.goalrail`

The old-write default is intentional. Runtime data includes SQLite DBs,
pidfiles, log sidecars, artifacts, and native harness state, so switching fresh
writes to `~/.goalrail` should happen only after the remaining roots have
explicit migration/compatibility coverage.

## Covered In This Milestone

- Local server data root now honors `GOALRAIL_DATA_DIR`.
- Chat persistent data root and process log root now honor `GOALRAIL_DATA_DIR`.
- Host runner log root now honors `GOALRAIL_DATA_DIR`.
- Server-side admin/allowed-domains/config data root now honors
  `GOALRAIL_DATA_DIR` unless `GOALRAIL_ADMIN_CREDENTIALS_PATH` pins a Docker
  volume parent.
- Host runner env propagation now allows both Goalrail and Goalrail path envs.
- Legacy state migration skips when `GOALRAIL_DATA_DIR` is set, matching the
  existing `GOALRAIL_DATA_DIR` safety behavior.

## Intentionally Left For Later

Remaining `~/.goalrail` roots need separate compatibility cuts:

- Native bridge/state roots:
  `claude_native_state.py`, `codex_native_state.py`,
  `opencode_native_state.py`, `*_native_bridge.py`.
- Host identity and global agent registry:
  `goalrail/host/identity.py`, `goalrail/cli.py` agent dirs.
- UI SDK/TUI config state:
  `sdks/ui/goalrail_ui_sdk/terminal/_config.py`.
- Update-check cache and event/debug logs:
  `goalrail/update_check.py`, `goalrail/repl/_event_tape.py`.
- Frontend/e2e fixtures and docs that still describe `~/.goalrail`.

These are deliberately not mass-renamed here because some are config, some are
runtime data, and some are per-native-harness private homes.

## Verification

Commands:

```sh
uv run python -m pytest tests/test_env_compat.py -q
uv run python -m pytest tests/host/test_local_server.py::test_local_data_dir_honors_data_dir_not_config_home tests/test_data_home_paths.py tests/server/test_admin_list.py::test_resolve_data_dir_uses_goalrail_data_dir -q
uv run python -m pytest tests/cli/test_cli.py::test_migrate_legacy_state_dir_skips_goalrail_data_dir -q
uv run python -m pytest tests/host/test_connect.py::test_build_runner_env_propagates_data_dir_paths_not_db_uri tests/cli/test_cli.py::test_migrate_legacy_state_dir_skips_goalrail_data_dir tests/host/test_local_server.py::test_local_data_dir_honors_data_dir_not_config_home tests/test_data_home_paths.py tests/server/test_admin_list.py::test_resolve_data_dir_uses_goalrail_data_dir tests/test_env_compat.py -q
uv run python -m pytest tests/test_env_compat.py tests/test_data_home_paths.py tests/server/test_admin_list.py -q
uv run python -m pytest tests/host/test_local_server.py tests/host/test_connect.py -q
uv run python -m pytest tests/cli/test_cli.py -q
uv run python -m pytest tests/test_env_compat.py tests/test_data_home_paths.py tests/server/test_admin_list.py tests/host/test_local_server.py tests/host/test_connect.py tests/cli/test_cli.py -q
git diff --check
```

Results:

- Resolver suite: 14 passed.
- Core consumer suite: 5 passed.
- Migration safety test: 1 passed.
- Combined targeted suite: 21 passed, with the pre-existing async-mark warning
  on `tests/host/test_connect.py::test_build_runner_env_propagates_data_dir_paths_not_db_uri`.
- Env/data/admin broader suite: 40 passed.
- Host local/connect broader suite: 89 passed, with pre-existing async-mark
  warnings in `tests/host/test_connect.py`.
- Full CLI suite: 200 passed.
- Final combined suite: 329 passed, with the same pre-existing async-mark
  warnings in `tests/host/test_connect.py`.
- `git diff --check`: passed.

## Next Milestone

Split the remaining roots by ownership before editing:

1. Native harness state roots.
2. Host identity / agent registry.
3. UI SDK config-vs-data decision.
4. Cache/debug/log directories.

Do not switch fresh writes to `~/.goalrail` until those roots have dual-read or
explicit migration behavior.
