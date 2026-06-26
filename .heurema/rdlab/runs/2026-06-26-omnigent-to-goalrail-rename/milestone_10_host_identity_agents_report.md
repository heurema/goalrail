# Milestone 10: Host Identity And Agent Registry Config Roots

Date: 2026-06-26

## Scope

This milestone moved host identity and user-level agent registry defaults onto the effective Goalrail config home, while preserving legacy Omnigent compatibility.

Covered paths:

- `omnigent/host/identity.py`
- `omnigent/host/connect.py`
- `omnigent/cli.py`
- `omnigent/onboarding/wizard.py`

## Behavior

- `load_or_create_host_identity()` now resolves its default config path through `default_config_path()`.
- `run_host_process()` uses the same effective host identity config path unless an explicit `config_path` is passed.
- CLI bundled-agent materialization writes under the effective user config home agent registry.
- Onboarding wizard agent discovery, naming, and YAML writes use the effective user config home agent registry.
- CLI and wizard display messages now show the effective config path instead of the legacy constant.

Compatibility preserved:

- `GOALRAIL_CONFIG_HOME` takes precedence through `config_home_path()`.
- `OMNIGENT_CONFIG_HOME` remains supported.
- Existing `~/.goalrail` and `~/.omnigent` config homes are still detected by the shared env compatibility layer.
- Existing tests that monkeypatch legacy module constants still keep their explicit override behavior.

## Intentionally Left Out

- Host daemon pid/log/registry runtime paths such as `_HOST_PID_PATH`.
- Runner identity caches and other runtime data paths.
- Project-local `.omnigent/config.yaml` behavior.
- Broad docs, frontend copy, package/distribution names, or e2e string cleanup.

Those are separate milestones because they touch data/runtime state or broader branding surfaces.

## Tests

Red tests added first and verified failing before implementation:

```sh
uv run python -m pytest tests/host/test_identity.py::test_default_config_path_honors_goalrail_config_home tests/cli/test_cli.py::test_global_agents_dir_uses_goalrail_config_home tests/onboarding/test_wizard.py::test_agents_dir_uses_goalrail_config_home -q
```

Focused compatibility tests after implementation:

```sh
uv run python -m pytest tests/host/test_identity.py::test_default_config_path_honors_goalrail_config_home tests/host/test_identity.py::test_default_load_or_create_writes_goalrail_config_home tests/cli/test_cli.py::test_global_agents_dir_uses_goalrail_config_home tests/cli/test_cli.py::test_materialize_bundled_example_uses_goalrail_config_home tests/onboarding/test_wizard.py::test_agents_dir_uses_goalrail_config_home tests/onboarding/test_wizard.py::test_save_yaml_uses_goalrail_config_home -q
```

Result: 6 passed.

Broader targeted suite:

```sh
uv run python -m pytest tests/host/test_identity.py tests/host/test_connect.py tests/cli/test_cli.py tests/onboarding/test_wizard.py -q
```

Result: 274 passed, 35 pre-existing `PytestWarning` warnings about non-async tests marked with `@pytest.mark.asyncio` in `tests/host/test_connect.py`.

Final targeted suite after pre-commit formatting:

```sh
uv run python -m pytest tests/test_env_compat.py tests/host/test_identity.py tests/host/test_connect.py tests/cli/test_cli.py tests/onboarding/test_wizard.py -q
```

Result: 288 passed, 35 pre-existing `PytestWarning` warnings in `tests/host/test_connect.py`.

Pre-commit:

```sh
uv run pre-commit run --files omnigent/cli.py omnigent/host/connect.py omnigent/host/identity.py omnigent/onboarding/wizard.py tests/cli/test_cli.py tests/host/test_identity.py tests/onboarding/test_wizard.py .heurema/rdlab/runs/2026-06-26-omnigent-to-goalrail-rename/milestone_10_host_identity_agents_report.md
```

Result: passed after ruff format/check auto-fixes were applied and the hook was rerun.

Whitespace check:

```sh
git diff --check
```

Result: passed.

Codebase-memory:

```text
project: Users-vi-personal-heurema-goalrail
status: ready
nodes: 46644
edges: 230462
```

## Remaining Risk

Risk: narrow.

The remaining risk is accidental divergence between config-path helpers and runtime data-path helpers. Keep the next milestone focused on runtime data paths so config compatibility remains reviewable.
