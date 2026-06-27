# Milestone 13: User-Facing Runtime Log Paths

Date: 2026-06-27

## Scope

This milestone updated the remaining targeted runtime error messages that still
printed hardcoded legacy log locations:

- local daemon exited before its local Goalrail server became ready
- local Goalrail server discovery timed out
- daemon-spawned runner did not connect before timeout

Covered paths:

- `omnigent/cli.py`
- `omnigent/host/daemon_launch.py`
- `tests/cli/test_backend.py`
- `tests/host/test_daemon_launch.py`

## Behavior

- Local server discovery errors now render log directories from the effective
  data home:
  - `<data-home>/logs/host-daemon/`
  - `<data-home>/logs/server/`
- Daemon runner timeout guidance now renders:
  - `<data-home>/logs/host-runner/`
- `GOALRAIL_DATA_DIR` and `OMNIGENT_DATA_DIR` are honored through
  `data_home_path()`.
- Paths under the user's home directory still collapse to `~/...`; explicit
  data dirs outside `$HOME` render as absolute paths.

## Intentionally Left Out

- Broad documentation, e2e comments, and static web bundle text.
- Fresh default remains `~/.omnigent` through `data_home_path()` until a later
  explicit migration/default-flip decision.
- Full-suite collection import repair for `tests/scripts/test_update_versions.py`;
  that is a separate test-infra issue and was not mixed into this milestone.

## Tests

Red tests added first and verified failing before implementation:

```sh
uv run pytest tests/cli/test_backend.py::test_discover_local_server_url_dead_daemon_reports_effective_log_dirs tests/cli/test_backend.py::test_discover_local_server_url_timeout_reports_effective_log_dir tests/host/test_daemon_launch.py::test_wait_for_runner_online_timeout_reports_effective_log_dir
```

Initial result: 3 failed because user-facing messages still printed hardcoded
`~/.omnigent/logs/...` paths.

Focused tests after implementation:

```sh
uv run pytest tests/cli/test_backend.py::test_discover_local_server_url_dead_daemon_reports_effective_log_dirs tests/cli/test_backend.py::test_discover_local_server_url_timeout_reports_effective_log_dir tests/host/test_daemon_launch.py::test_wait_for_runner_online_timeout_reports_effective_log_dir
```

Result: 3 passed.

Broader targeted suite:

```sh
uv run pytest tests/cli/test_backend.py tests/host/test_daemon_launch.py tests/test_data_home_paths.py tests/test_env_compat.py tests/host/test_connect.py
```

Result: 155 passed, 35 existing pytest warnings about sync tests marked async.

Additional CLI path/status surface:

```sh
uv run pytest tests/cli/test_server_lifecycle.py tests/cli/test_runner_startup.py
```

Result: 38 passed.

Wide CLI suite:

```sh
uv run pytest tests/cli/test_cli.py
```

Result: 202 passed.

Pre-commit on changed code/test files:

```sh
uv run pre-commit run --files omnigent/cli.py omnigent/host/daemon_launch.py tests/cli/test_backend.py tests/host/test_daemon_launch.py
```

Result: passed.

Whitespace check:

```sh
git diff --check
```

Result: passed.

Full-suite attempt:

```sh
uv run pytest
```

Result: blocked during collection before executing tests:

```text
ImportError: cannot import name 'update_versions' from 'scripts' (/Users/vi/personal/heurema/goalrail/tests/scripts/__init__.py)
```

The failing collection module is `tests/scripts/test_update_versions.py`, whose
`from scripts import update_versions` import resolves to `tests/scripts` during
full collection.

## Codebase-Memory

Before implementation:

```text
project: Users-vi-personal-heurema-goalrail
status: ready
nodes: 46676
edges: 230450
```

After implementation:

```text
project: Users-vi-personal-heurema-goalrail
status: ready
nodes: 46691
edges: 230322
```

## Remaining Risk

Risk: narrow.

The changed behavior is limited to user-facing guidance strings. The main
remaining rename risk is broad docs/e2e/static bundle text and the later
decision about whether fresh installs should default to `~/.goalrail`.
