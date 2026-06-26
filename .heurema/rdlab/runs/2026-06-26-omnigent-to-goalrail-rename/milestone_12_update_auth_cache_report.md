# Milestone 12: Update Cache And Auth Token Data Paths

Date: 2026-06-26

## Scope

This milestone moved two remaining user-level mutable runtime files onto the effective Goalrail data home:

- update-check cache
- CLI auth token storage

Covered paths:

- `omnigent/update_check.py`
- `omnigent/cli_auth.py`
- `tests/cli/test_update_check.py`
- `tests/cli/test_cli_auth.py`

## Behavior

- Update-check cache reads and writes now resolve through effective cache helpers:
  - `<data-home>/.update_check.json`
- CLI auth token storage now uses:
  - `<data-home>/auth_tokens.json`
- `GOALRAIL_DATA_DIR` and `OMNIGENT_DATA_DIR` are honored through `data_home_path()`.
- Existing `~/.goalrail` and `~/.omnigent` data homes are still resolved by the shared env compatibility layer.
- Existing test monkeypatches of `_CACHE_DIR`, `_CACHE_FILE`, and `_token_file_path` remain supported.

## Intentionally Left Out

- Local server user-facing path text and docs/e2e comments.
- Provider-specific credential stores outside Goalrail state, such as Codex, OpenCode, Databricks CLI, or Gemini/Antigravity stores.
- Fresh default remains `~/.omnigent` through `data_home_path()` until a later explicit migration/default-flip decision.

## Tests

Red tests added first and verified failing before implementation:

```sh
uv run python -m pytest tests/cli/test_update_check.py::test_write_cache_uses_goalrail_data_dir tests/cli/test_update_check.py::test_read_cache_uses_goalrail_data_dir tests/cli/test_cli_auth.py::test_store_token_uses_goalrail_data_dir -q
```

Initial result: 3 failed because cache/token files still used legacy paths.

Focused tests after implementation:

```sh
uv run python -m pytest tests/cli/test_update_check.py::test_write_cache_uses_goalrail_data_dir tests/cli/test_update_check.py::test_read_cache_uses_goalrail_data_dir tests/cli/test_cli_auth.py::test_store_token_uses_goalrail_data_dir -q
```

Result: 3 passed.

Broader targeted suite:

```sh
uv run python -m pytest tests/cli/test_update_check.py tests/cli/test_cli_auth.py tests/server/test_accounts.py tests/cli/test_upgrade_command.py tests/test_env_compat.py tests/test_data_home_paths.py -q
```

Result: 234 passed.

Final broader targeted suite:

```sh
uv run python -m pytest tests/cli/test_update_check.py tests/cli/test_cli_auth.py tests/server/test_accounts.py tests/cli/test_upgrade_command.py tests/test_env_compat.py tests/test_data_home_paths.py tests/cli/test_cli.py -q
```

Result: 436 passed.

Pre-commit:

```sh
uv run pre-commit run --files omnigent/update_check.py omnigent/cli_auth.py tests/cli/test_update_check.py tests/cli/test_cli_auth.py .heurema/rdlab/runs/2026-06-26-omnigent-to-goalrail-rename/milestone_12_update_auth_cache_report.md
```

Result: passed.

Whitespace check:

```sh
git diff --check
```

Result: passed.

Codebase-memory:

```text
project: Users-vi-personal-heurema-goalrail
status: ready
nodes: 46676
edges: 230450
```

## Remaining Risk

Risk: narrow.

The remaining rename risk is mostly user-facing path text, docs/e2e comments, and any provider-owned external credential stores that intentionally live outside Goalrail state.
