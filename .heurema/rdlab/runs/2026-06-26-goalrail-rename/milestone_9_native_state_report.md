# Milestone 9: Goalrail Native State Compatibility

Date: 2026-06-26

## Scope

Implemented the next bounded runtime-data compatibility cut for native harness
state and bridge directories. This does not migrate existing files and does not
switch the fresh default away from `~/.goalrail`.

Changed runtime surfaces:

- `goalrail/claude_native_state.py`
- `goalrail/codex_native_state.py`
- `goalrail/opencode_native_state.py`
- `goalrail/codex_native_bridge.py`
- `goalrail/opencode_native_bridge.py`
- `goalrail/antigravity_native_bridge.py`
- `goalrail/pi_native_bridge.py`
- `goalrail/claude_native_bridge.py`

Changed tests:

- `tests/test_claude_native_state.py`
- `tests/test_codex_native_state.py`
- `tests/test_opencode_native_state.py`
- `tests/test_codex_native_bridge.py`
- `tests/test_opencode_native_bridge.py`
- `tests/test_antigravity_native_bridge.py`
- `tests/test_pi_native_bridge.py`
- `tests/test_claude_native_bridge.py`

## Compatibility Policy

Native launch-state and durable bridge defaults now resolve through
`data_home_path()`:

1. `GOALRAIL_DATA_DIR`
2. `GOALRAIL_DATA_DIR`
3. existing `~/.goalrail`
4. existing `~/.goalrail`
5. existing `~/.goalrail`
6. existing `~/.goalrail`
7. fresh default `~/.goalrail`

Existing per-harness state overrides still win:

- `GOALRAIL_CLAUDE_NATIVE_STATE_DIR`
- `GOALRAIL_CODEX_NATIVE_STATE_DIR`
- `GOALRAIL_OPENCODE_NATIVE_STATE_DIR`

Existing test and advanced-user bridge-root overrides still work because
`_BRIDGE_ROOT` remains monkeypatch-able. The production default is now computed
lazily only when `_BRIDGE_ROOT` is unset.

## Covered In This Milestone

- `claude-native`, `codex-native`, and `opencode-native` launch-state roots now
  honor `GOALRAIL_DATA_DIR`.
- `codex-native`, `opencode-native`, `antigravity-native`, and `pi-native`
  durable bridge roots now honor `GOALRAIL_DATA_DIR`.
- Shared Claude MCP relay bridge validation now treats data-home bridge roots
  as owned runtime data. For `<data-home>/<native>/<hash>`, the trusted anchor
  is `<data-home>.parent`, so `_ensure_secure_dir` validates and chmods the
  data-home directory itself instead of trusting a user-supplied data dir.
- Claude bridge subprocess tests now use `tempfile.gettempdir()` instead of
  hardcoded `/tmp`, matching the subprocess's production temp root on macOS.
- Relay error-text expectations were updated from `Goalrail` to `Goalrail`
  where runtime output had already changed.

## Intentionally Left For Later

- `claude-native`, `hermes-native`, `kiro-native`, `qwen-native`, and
  cursor-family tmp bridge roots remain temp IPC roots. They are not durable
  data-home state.
- Antigravity's own `~/.gemini/antigravity-cli` app-data root is a third-party
  CLI concern, not Goalrail runtime data.
- Host identity and global agent registry are still separate:
  `goalrail/host/identity.py`, `goalrail/cli.py` agent dirs.
- UI SDK/TUI config state still needs a config-vs-data decision:
  `sdks/ui/goalrail_ui_sdk/terminal/_config.py`.
- Update-check cache and event/debug logs remain for later:
  `goalrail/update_check.py`, `goalrail/repl/_event_tape.py`.
- Frontend/e2e fixtures and docs that still describe `~/.goalrail` remain for a
  separate documentation/fixture pass.

## Verification

RED checks:

```sh
uv run python -m pytest tests/test_claude_native_state.py::test_default_state_root_honors_goalrail_data_dir tests/test_codex_native_state.py::test_default_state_root_honors_goalrail_data_dir tests/test_opencode_native_state.py::test_default_state_root_honors_goalrail_data_dir tests/test_codex_native_bridge.py::test_bridge_root_honors_goalrail_data_dir tests/test_opencode_native_bridge.py::test_bridge_root_honors_goalrail_data_dir tests/test_antigravity_native_bridge.py::test_bridge_root_honors_goalrail_data_dir tests/test_pi_native_bridge.py::test_bridge_root_honors_goalrail_data_dir -q
uv run python -m pytest tests/test_claude_native_bridge.py::test_trusted_parent_validates_goalrail_data_dir_for_codex_native -q
```

Results:

- New native-root tests initially failed against the old
  `Path.home() / ".goalrail"` behavior.
- The custom-data-dir trusted-parent guard initially failed after isolating it
  from the Claude bridge autouse root.

GREEN checks:

```sh
uv run python -m pytest tests/test_claude_native_state.py::test_default_state_root_honors_goalrail_data_dir tests/test_codex_native_state.py::test_default_state_root_honors_goalrail_data_dir tests/test_opencode_native_state.py::test_default_state_root_honors_goalrail_data_dir tests/test_codex_native_bridge.py::test_bridge_root_honors_goalrail_data_dir tests/test_opencode_native_bridge.py::test_bridge_root_honors_goalrail_data_dir tests/test_antigravity_native_bridge.py::test_bridge_root_honors_goalrail_data_dir tests/test_pi_native_bridge.py::test_bridge_root_honors_goalrail_data_dir tests/test_claude_native_bridge.py::test_trusted_parent_validates_goalrail_data_dir_for_codex_native -q
uv run python -m pytest tests/test_claude_native_bridge.py::test_mcp_server_initialize_omits_blocked_channel_capability tests/test_claude_native_bridge.py::test_channel_server_relays_active_goalrail_tools tests/test_claude_native_bridge.py::test_call_relay_tool_returns_mcp_error_on_read_timeout tests/test_claude_native_bridge.py::test_serve_mcp_survives_handler_exception_and_keeps_serving -q
uv run python -m pytest tests/test_claude_native_state.py tests/test_codex_native_state.py tests/test_opencode_native_state.py tests/test_codex_native_bridge.py tests/test_opencode_native_bridge.py tests/test_antigravity_native_bridge.py tests/test_pi_native_bridge.py tests/test_claude_native_bridge.py -q
uv run python -m pytest tests/test_env_compat.py tests/test_data_home_paths.py tests/test_claude_native_state.py tests/test_codex_native_state.py tests/test_opencode_native_state.py tests/test_codex_native_bridge.py tests/test_opencode_native_bridge.py tests/test_antigravity_native_bridge.py tests/test_pi_native_bridge.py tests/test_claude_native_bridge.py -q
pre-commit run --files goalrail/antigravity_native_bridge.py goalrail/claude_native_bridge.py goalrail/claude_native_state.py goalrail/codex_native_bridge.py goalrail/codex_native_state.py goalrail/opencode_native_bridge.py goalrail/opencode_native_state.py goalrail/pi_native_bridge.py tests/test_antigravity_native_bridge.py tests/test_claude_native_bridge.py tests/test_claude_native_state.py tests/test_codex_native_bridge.py tests/test_codex_native_state.py tests/test_opencode_native_bridge.py tests/test_opencode_native_state.py tests/test_pi_native_bridge.py .heurema/rdlab/runs/2026-06-26-goalrail-rename/milestone_9_native_state_report.md
git diff --check
codebase-memory-mcp cli index_repository '{"repo_path":"/Users/vi/personal/heurema/goalrail","mode":"full","persistence":false}'
```

Results:

- New native-root/security tests: 8 passed.
- Previously failing Claude bridge subprocess/relay tests: 4 passed after test
  fixture correction.
- Related native state/bridge suite: 264 passed.
- Final env/data/native suite: 281 passed.
- `pre-commit run --files ...`: passed after an initial `ruff format` hook
  rewrote one test file.
- `git diff --check`: passed.
- `codebase-memory-mcp` index status remained ready after a full CLI index call.

## Next Milestone

Move to host identity and global agent registry roots. Keep that separate from
native harness state because it affects user identity/config semantics rather
than per-session runtime data.
