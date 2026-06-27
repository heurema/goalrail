# Milestone 14: Pytest Collection And Local Ambient Isolation

Date: 2026-06-27

## Scope

This milestone fixed two test-infra blockers found while trying to run the full
unit suite after the runtime path-text milestone:

- full-suite collection failed importing `tests/scripts/test_update_versions.py`
- `tests/cli/test_configure_models.py::test_readd_same_source_key_updates_in_place`
  could hang on macOS when the developer's real Claude Keychain login leaked
  into ambient detection

Covered paths:

- `tests/scripts/test_update_versions.py`
- `tests/cli/test_configure_models.py`

## Behavior

- `test_update_versions.py` now imports `scripts/update_versions.py` directly
  via `importlib.util.spec_from_file_location`, matching the existing pattern
  used by script tests that must be robust to `tests/scripts` package shadowing.
- `test_configure_models.py` now stubs `harness_cli_logged_in()` to `False` in
  its autouse harness fixture. This keeps setup menu tests independent of a
  developer's real macOS Claude Keychain login while preserving explicit
  `harness_login()` / `harness_logout()` stubs used by subscription-flow tests.

## Tests

Red evidence before fixes:

```sh
uv run pytest --collect-only
```

Initial result: collection failed with:

```text
ImportError: cannot import name 'update_versions' from 'scripts' (/Users/vi/personal/heurema/goalrail/tests/scripts/__init__.py)
```

After the import fix:

```sh
uv run pytest --collect-only
```

Result: collected 13043 tests.

```sh
uv run pytest tests/scripts/test_update_versions.py
```

Result: 10 passed.

Red evidence for the second blocker:

```sh
uv run pytest tests/cli/test_configure_models.py::test_readd_same_source_key_updates_in_place
```

Initial result: hung in the Claude subscription credential menu because ambient
Claude CLI login detection leaked from the developer machine.

After the fixture isolation fix:

```sh
uv run pytest tests/cli/test_configure_models.py::test_readd_same_source_key_updates_in_place
```

Result: 1 passed.

```sh
uv run pytest tests/cli/test_configure_models.py
```

Result: 88 passed.

Pre-commit:

```sh
uv run pre-commit run --files tests/scripts/test_update_versions.py tests/cli/test_configure_models.py .heurema/rdlab/runs/2026-06-26-goalrail-rename/milestone_14_pytest_unblock_report.md
```

Result: passed.

Whitespace check:

```sh
git diff --check
```

Result: passed.

Full-suite attempt after both fixes:

```sh
uv run pytest
```

Result: no longer failed collection and no longer hung on
`test_readd_same_source_key_updates_in_place`, but still failed broadly outside
this milestone's scope. The run was interrupted after 19m55s with:

```text
39 failed, 7566 passed, 64 skipped, 6 xfailed
```

Observed remaining failure areas included missing optional `psycopg`,
`deploy.docker` import layout, frontend SDK Rich rendering expectations, sandbox
security tests, server integration policy/session tests, and other broad suite
health failures.

## Codebase-Memory

Before implementation:

```text
project: Users-vi-personal-heurema-goalrail
status: ready
nodes: 46691
edges: 230322
```

After implementation:

```text
project: Users-vi-personal-heurema-goalrail
status: ready
nodes: 46702
edges: 230538
```

## Remaining Risk

Risk: narrow for this patch.

The fixed behavior is limited to test import robustness and local ambient
credential isolation. Full-suite health still needs separate triage; this patch
only removes the two blockers that prevented reaching those broader failures.
