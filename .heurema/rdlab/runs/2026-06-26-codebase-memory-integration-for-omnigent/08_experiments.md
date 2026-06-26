# 08 Experiments

## Installer tests

1. `parse_args --skip-codebase-memory`
   - Expected: sets `INSTALL_CODEBASE_MEMORY=false`.

2. `parse_args` default
   - Expected: `INSTALL_CODEBASE_MEMORY=true`.

3. `install_codebase_memory` when `uv tool install` fails
   - Stub `uv` to exit non-zero.
   - Expected: function returns 0 and emits a warning.

4. `install_codebase_memory` when wrapper download/version check fails
   - Stub `$bin_dir/codebase-memory-mcp --version` to exit non-zero.
   - Expected: function returns 0 and emits a warning.

5. `install_codebase_memory` when `config set auto_index true` fails
   - Expected: warning-not-fatal; Omnigent install flow continues.

6. `--skip-codebase-memory`
   - Expected: no `uv tool install codebase-memory-mcp` command is invoked.

7. Interactive full repair branch, if implemented
   - Stub `install --plan`, prompt yes/no, and `install -y`.
   - Expected: `install -y` only runs after affirmative prompt.

8. Non-interactive full repair branch, if implemented
   - Expected: does not run `install -y`; prints manual retry command.

## Setup tests

1. CBM missing
   - Stub `shutil.which`/helper to return none.
   - Expected: setup emits advisory; no exception.

2. CBM present and auto_index succeeds
   - Stub subprocess success.
   - Expected: `config set auto_index true` invoked.

3. CBM present and auto_index fails
   - Stub subprocess non-zero.
   - Expected: warning/advisory only; setup continues.

4. Plan/repair prompt, if implemented
   - Expected: no full repair subprocess without explicit confirmation.

## Manual probes before PR merge

- Run installer tests under POSIX `sh`.
- Run `install_oss.sh --skip-codebase-memory --non-interactive` in a stubbed PATH.
- Run `codebase-memory-mcp install --plan` manually on a development machine to inspect planned writes.
- Confirm remote sandbox behavior remains unchanged.

## Candidate experiments

| ID | Hypothesis | Method | Evidence rows | Success criteria | Risk |
|---|---|---|---|---|---|

## Experiment backlog


## Experiments not worth running
