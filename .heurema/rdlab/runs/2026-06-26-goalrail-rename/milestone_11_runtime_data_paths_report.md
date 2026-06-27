# Milestone 11: Host Daemon And Runner Runtime Data Paths

Date: 2026-06-26

## Scope

This milestone moved the next runtime-data slice onto the effective Goalrail data home:

- host daemon pid file
- host daemon per-target registry
- host daemon background log directory
- runner id cache

Covered paths:

- `goalrail/cli.py`
- `goalrail/runner/identity.py`
- `tests/host/test_cli_host.py`
- `tests/runner/test_identity.py`

## Behavior

- `get_stable_runner_id()` now creates the default runner id cache at `<data-home>/runners/runner_id`.
- Background host daemon runtime files now derive from an effective host pid path:
  - `<data-home>/host.pid`
  - `<data-home>/daemons/*.json`
  - `<data-home>/logs/host-daemon/daemon-*.log`
- `GOALRAIL_DATA_DIR` are honored through `data_home_path()`.
- Existing `~/.goalrail` and `~/.goalrail` data homes are still resolved by the shared env compatibility layer.
- Tests that monkeypatch `_HOST_PID_PATH` without a data-home override still keep their isolated path.

## Intentionally Left Out

- Local server pid/log/database paths already have their own helpers and tests; broader cleanup remains separate.
- Update-check cache and auth token storage are not part of this daemon/runner slice.
- Public docs, e2e comments, and user-facing `~/.goalrail` text were not broadly rewritten.
- Fresh default remains `~/.goalrail` through `data_home_path()` until a later explicit migration/default-flip decision.

## Tests

Red tests added first and verified failing before implementation:

```sh
uv run python -m pytest tests/runner/test_identity.py::test_default_runner_id_path_uses_goalrail_data_dir tests/runner/test_identity.py::test_get_stable_runner_id_writes_goalrail_data_dir tests/host/test_cli_host.py::test_ensure_host_daemon_uses_goalrail_data_dir_for_runtime_files -q
```

Initial result: 3 failed because the runner id cache and host daemon runtime files still used legacy paths.

Focused tests after implementation:

```sh
uv run python -m pytest tests/runner/test_identity.py::test_default_runner_id_path_uses_goalrail_data_dir tests/runner/test_identity.py::test_get_stable_runner_id_writes_goalrail_data_dir tests/host/test_cli_host.py::test_ensure_host_daemon_uses_goalrail_data_dir_for_runtime_files -q
```

Result: 3 passed.

Broader targeted suite:

```sh
uv run python -m pytest tests/runner/test_identity.py tests/host/test_cli_host.py tests/cli/test_backend.py tests/cli/test_server_lifecycle.py tests/test_data_home_paths.py tests/test_env_compat.py -q
```

Result: 120 passed, 1 pre-existing `DeprecationWarning` from the zombie-daemon test using `os.fork()`.

Final broader targeted suite:

```sh
uv run python -m pytest tests/test_env_compat.py tests/test_data_home_paths.py tests/runner/test_identity.py tests/host/test_cli_host.py tests/cli/test_backend.py tests/cli/test_server_lifecycle.py tests/cli/test_cli.py tests/host/test_connect.py -q
```

Result: 383 passed, 36 warnings. The warnings were pre-existing for the covered suites: one `os.fork()` `DeprecationWarning` in the zombie-daemon test and non-async tests marked with `@pytest.mark.asyncio` in `tests/host/test_connect.py`.

Pre-commit:

```sh
uv run pre-commit run --files goalrail/cli.py goalrail/runner/identity.py tests/host/test_cli_host.py tests/runner/test_identity.py .heurema/rdlab/runs/2026-06-26-goalrail-rename/milestone_11_runtime_data_paths_report.md
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
nodes: 46658
edges: 230313
```

## Remaining Risk

Risk: narrow.

The main remaining rename risk is broad state not covered by this slice: update-check cache, auth token storage, local-server user-facing path text, and documentation/e2e comments that still mention `~/.goalrail`.
