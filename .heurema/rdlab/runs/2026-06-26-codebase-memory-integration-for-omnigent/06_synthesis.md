# 06 Synthesis

## Verdict

Use codebase-memory-mcp as an explicit companion tool, not as an Omnigent wheel
dependency and not as vendored source/binary.

## Refined decision

The initial direction is correct, but the exact `install -y` step needs a
guardrail. Current codebase-memory-mcp install can delete existing index DBs
when `-y` is supplied. That is acceptable only with explicit user awareness or
with a safer upstream repair mode.

## Recommended first PR shape

### Installer

- Add `INSTALL_CODEBASE_MEMORY=true`.
- Add `--skip-codebase-memory`.
- Add `install_codebase_memory "$bin_dir"`.
- Call it after `verify_omnigent "$bin_dir"` and before PATH/next-step output.
- Use warning-not-fatal handling for every CBM command.

Recommended safe default commands:

```sh
uv tool install --force -q --python "$PYTHON_VERSION" codebase-memory-mcp
"$bin_dir/codebase-memory-mcp" --version
"$bin_dir/codebase-memory-mcp" config set auto_index true
```

Do not blindly run this as an unconditional non-interactive default:

```sh
"$bin_dir/codebase-memory-mcp" install -y
```

Instead, either:

- interactive: show a clear prompt before full agent-config repair, noting that
  existing CBM indexes may be rebuilt/deleted by current CBM install behavior;
- non-interactive: skip full repair and print the command to run manually;
- future-safe: use a new CBM non-destructive repair mode once available.

### Setup repair/check

Add a small Omnigent-side helper, e.g. `omnigent/onboarding/codebase_memory.py`,
with:

- `find_binary()`
- `version()`
- `configure_auto_index()`
- `install_plan()`
- `repair_agent_configs()` only if explicitly confirmed

Then wire `omnigent setup` to:

- if CBM is missing: advise install or offer explicit install;
- if CBM exists: ensure `auto_index=true`;
- optionally show `install --plan` and ask before full repair.

### Remote sandbox

Do not treat local installer work as remote support. Add codebase-memory-mcp to
the host image layer separately if sandbox agents need it.

## Why this beats dependency/vendoring

- It keeps package install side-effect-free.
- It makes external agent config mutation explicit.
- It avoids bundling a platform binary into Omnigent's wheel lifecycle.
- It lets Omnigent remain usable when CBM download/config fails.
- It keeps remote host-image work separate from local developer setup.

## Main risk

The main risk is not installing the companion binary; it is performing full CBM
agent-config repair with `-y` in a way that silently deletes/rebuilds existing
CBM indexes. Solve that before making full repair automatic.

## What is known


## What is uncertain


## What changed recently


## Patterns


## Anti-patterns


## Open problems


## Strategic implications


## Opportunity map


## Risk map


## Recommended experiments
