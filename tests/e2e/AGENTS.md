# `tests/e2e/` - prerequisites & how to run

These tests start a real `goalrail` server subprocess, upload real agent
bundles, and call real LLM APIs. They are **excluded from the default `pytest`
run** via `addopts = --ignore=tests/e2e` in `pyproject.toml`. To exercise them
against a real provider you must opt in with `--llm-api-key`.

## Always Run Integration + Unit Tests In The Background

The e2e suite takes 5-10 minutes even fully parallel; the unit suite takes 5-7
minutes parallel. Do not block an interactive agent terminal on a foreground
run. Start it backgrounded and monitor a known log path:

```bash
export OPENAI_API_KEY=sk-...

(env -u ANTHROPIC_API_KEY \
  uv run --no-sync pytest tests/e2e/ \
    --llm-api-key="$OPENAI_API_KEY" \
    -n 8 --dist=loadscope \
    --tb=line -q -rfs 2>&1 | tee /tmp/e2e.log) &

until grep -qE "passed in [0-9]|failed in [0-9]|short test summary" /tmp/e2e.log 2>/dev/null; do
  sleep 10
done
grep -E "passed in [0-9]|failed in [0-9]" /tmp/e2e.log | tail -1
```

This applies to both the unit suite (`uv run pytest -n 8 --dist=loadfile`) and
the e2e suite.

## LLM Credentials

### Direct Provider Key

Use a real OpenAI-compatible key directly:

```bash
export OPENAI_API_KEY=sk-...
uv run pytest tests/e2e/ \
  --llm-api-key="$OPENAI_API_KEY" \
  -n 8 --dist=loadscope
```

### OpenAI-Compatible Gateway

Set `GOALRAIL_E2E_OPENAI_BASE_URL` when the key belongs to an
OpenAI-compatible gateway rather than `api.openai.com`:

```bash
export GOALRAIL_E2E_OPENAI_BASE_URL=https://gateway.example.com/v1
export GOALRAIL_E2E_LLM_API_KEY=...

uv run pytest tests/e2e/ \
  --llm-api-key="$GOALRAIL_E2E_LLM_API_KEY" \
  -n 8 --dist=loadscope
```

When the gateway URL is set, the spawned server receives `OPENAI_BASE_URL` and
agent bundle model names are rewritten via `_GATEWAY_MODEL_MAP` in
`tests/e2e/conftest.py`.

## Binaries On `PATH`

Some tests gate on local binaries. Install whichever you need; tests for
missing binaries skip individually rather than failing the suite.

| Binary | Required by | Install |
| --- | --- | --- |
| `tmux` | terminal and REPL e2e tests | `brew install tmux` (macOS) / `apt install tmux` (Debian) |
| `claude` | `claude-sdk` and `claude-native` rows | Anthropic Claude CLI |
| `codex` | `codex` and `codex-native` rows | OpenAI Codex CLI |
| `pi` | `pi` harness rows | Internal CLI, see project docs |
| `goalrail` | CLI-backed e2e tests | `uv sync` makes it available via `uv run goalrail ...` |

## Python Environment

Run this from the repo root:

```bash
uv sync --extra dev --extra claude-sdk --extra openai-agents
```

## Recommended Invocation

```bash
export OPENAI_API_KEY=sk-...
uv run pytest tests/e2e/ \
  --llm-api-key="$OPENAI_API_KEY" \
  -n 8 --dist=loadscope
```

- `-n 8` is the empirical sweet spot on a 12-core laptop. `-n 4` is more stable
  but slower.
- `--dist=loadscope` keeps tests within one file on a single worker so the
  session-scoped `live_server` and agent-upload fixtures spawn once per worker
  per file.
- `--profile` is no longer supported. Use `GOALRAIL_E2E_OPENAI_BASE_URL` for
  gateway routing.

## Skip-Reason Cheat Sheet

| Reason | Fix |
| --- | --- |
| `tmux not installed; ...` | Install `tmux` |
| `tmux, pi, and provider API key required` | Install `tmux` + `pi` + set `OPENAI_API_KEY` |
| `OPENAI_API_KEY not set` | `export OPENAI_API_KEY=...` or pass `--llm-api-key` |
| `Integration tests require --integration flag` | Add `--integration` |
| `<harness>'s CLI not on PATH` | Install the named binary (`claude` / `codex` / `pi`) |
| `test uses an LLM judge that hits api.openai.com directly` | Export a real `OPENAI_API_KEY=sk-...` when gateway mode is enabled |

## Parallel-Safety Notes

- A few tests still write to fixed `/tmp/...` paths
  (`test_harness_wrap_e2e.py`, `test_example_agent_with_os_env_fork.py`). They
  serialize naturally because each is a single test, but coordinate any new
  tests that reuse those paths.
