# Milestone 7: Goalrail Config Home Compatibility

Date: 2026-06-26

## Scope

Implemented config-home compatibility for the rebrand without moving the full
state directory yet.

Changed runtime surfaces:

- `omnigent/_env_compat.py`
- `omnigent/cli.py`
- `omnigent/onboarding/provider_config.py`
- `omnigent/onboarding/secrets.py`
- `omnigent/runner/_entry.py`

Changed tests:

- `tests/test_env_compat.py`
- `tests/cli/test_cli.py`
- `tests/onboarding/test_provider_config.py`
- `tests/onboarding/test_secrets.py`
- `tests/runner/test_runner_entry.py`

## Compatibility Policy

Resolution order for config home is:

1. `GOALRAIL_CONFIG_HOME`
2. `OMNIGENT_CONFIG_HOME`
3. existing `~/.goalrail`
4. existing `~/.omnigent`
5. existing `~/.omnigents`
6. existing `~/.omniagents`
7. fresh default `~/.omnigent`

The fresh default intentionally remains `~/.omnigent` for now. This avoids a
hidden one-way state migration before daemon data, logs, native session state,
web storage, and deployment templates have their own compatibility gate.

## Additional Test Hygiene

`tests/test_env_compat.py` now restores prefixed env vars after every test.
`mirror_legacy_env()` mutates `os.environ` directly, so plain pytest
`monkeypatch` did not roll back generated mirror variables. Without this, later
tests in the same process could inherit stale `GOALRAIL_CONFIG_HOME` values.

`tests/runner/test_runner_entry.py` now imports the runner app before replacing
`httpx.AsyncClient` with a test factory. This avoids a test-order failure where
the lazy MCP import evaluated `httpx.AsyncClient | None` after the test had
already replaced `httpx.AsyncClient` with a function.

## Intentionally Left For Later

- Full state directory default migration from `~/.omnigent` to `~/.goalrail`.
- Data directory and daemon-state compatibility (`OMNIGENT_DATA_DIR`,
  pid/socket/log paths, local DB).
- Web storage key migration from `omnigent:*` to `goalrail:*`.
- Deployment image/template updates for remote/sandbox agents.
- Python package/import-root rename.

## Verification

Commands:

```sh
uv run python -m pytest tests/test_env_compat.py -q
uv run python -m pytest tests/cli/test_cli.py::test_load_global_config_uses_goalrail_env_override tests/cli/test_cli.py::test_load_global_config_reads_existing_goalrail_home_by_default tests/runner/test_runner_entry.py::test_load_runner_idle_timeout_reads_goalrail_config_home tests/onboarding/test_provider_config.py::test_config_path_uses_goalrail_config_home_env tests/onboarding/test_secrets.py::test_config_home_uses_goalrail_config_home_env -q
uv run python -m pytest tests/test_env_compat.py tests/cli/test_cli.py::test_load_global_config_uses_env_override tests/cli/test_cli.py::test_load_global_config_uses_goalrail_env_override tests/cli/test_cli.py::test_load_global_config_reads_existing_goalrail_home_by_default tests/cli/test_cli.py::test_load_global_config_returns_empty_when_missing tests/cli/test_cli.py::test_save_and_load_global_config_round_trips tests/cli/test_cli.py::test_save_global_config_merges_with_existing tests/cli/test_cli.py::test_save_global_config_unset_removes_key tests/cli/test_cli.py::test_config_set_global_reports_effective_config_home tests/runner/test_runner_entry.py::test_load_runner_idle_timeout_defaults_when_config_missing tests/runner/test_runner_entry.py::test_load_runner_idle_timeout_reads_nested_runner_config tests/runner/test_runner_entry.py::test_load_runner_idle_timeout_reads_goalrail_config_home tests/runner/test_runner_entry.py::test_load_runner_idle_timeout_zero_disables_watchdog tests/onboarding/test_provider_config.py::test_config_path_uses_goalrail_config_home_env tests/onboarding/test_secrets.py::test_config_home_uses_goalrail_config_home_env -q
uv run python -m pytest tests/onboarding/test_provider_config.py tests/onboarding/test_secrets.py -q
uv run python -m pytest tests/runner/test_runner_entry.py -q
uv run python -m pytest tests/test_model_catalog.py tests/runtime/test_provider_spawn_env.py -q
uv run python -m pytest tests/cli/test_cli.py -q
```

Results:

- Config-home targeted suite: 5 passed.
- Compatibility suite: 24 passed.
- Provider config/secrets suite: 53 passed.
- Runner entry suite: 61 passed.
- Model catalog/provider spawn env suite: 78 passed.
- CLI suite: 199 passed.

## Next Milestone

Add data/state path compatibility deliberately. The next step should inventory
runtime state homes separately from config home, then decide whether fresh
Goalrail installs can write to `~/.goalrail` or whether a longer dual-read /
old-write window is needed for daemon state and native app storage.
