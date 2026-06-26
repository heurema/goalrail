# Milestone 6: Goalrail Environment Alias Compatibility

Date: 2026-06-26

## Scope

Implemented the first runtime compatibility step for environment variables:
`GOALRAIL_*` is now the canonical public prefix, while existing `OMNIGENT_*`,
`OMNIGENTS_*`, and `OMNIAGENTS_*` values remain supported.

Changed surfaces:

- `omnigent/_env_compat.py`
- `tests/test_env_compat.py`

## Compatibility Policy

Precedence is:

1. `GOALRAIL_*`
2. `OMNIGENT_*`
3. `OMNIGENTS_*`
4. `OMNIAGENTS_*`

The shim still populates `OMNIGENT_*` because most existing runtime readers
continue to consume that prefix during the migration. That keeps the change
additive and avoids a broad mechanical rename of environment reads.

## Intentionally Left For Later

- Mass documentation replacement of `OMNIGENT_*` env names.
- Deployment template env-var renames.
- Config home migration from `~/.omnigent` to `~/.goalrail`.
- Web storage key migration from `omnigent:*` to `goalrail:*`.
- Python package/import-root rename.

These need separate compatibility gates.

## Verification

Commands:

```sh
uv run python -m pytest tests/test_env_compat.py -q
uv run python -m pytest tests/test_native_codex_provider.py tests/runtime/test_provider_spawn_env.py tests/test_model_catalog.py -q
```

Results:

- `tests/test_env_compat.py`: 5 passed.
- Existing env-heavy tests: 93 passed.

## Next Milestone

Add config/state path compatibility for `GOALRAIL_CONFIG_HOME` and eventual
`~/.goalrail` support, while preserving existing `OMNIGENT_CONFIG_HOME` and
legacy state directories.
