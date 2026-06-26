# 10 Final Memo

## Executive summary

MODIFY the initial plan. The strategic direction is right: codebase-memory-mcp
should be a companion tool installed/configured explicitly by Omnigent
installer/setup, not vendored into the Omnigent wheel and not hidden as a
`pyproject.toml` dependency.

The command sequence needs one safety change: do not run
`codebase-memory-mcp install -y` blindly as a default step. Current CBM install
can delete existing index DBs when `-y` auto-confirms prompts. Make binary
install and `auto_index=true` warning-not-fatal by default; make full agent
config repair explicit, prompted, or dependent on a safer upstream repair mode.

## Decision context

Omnigent wants codebase-memory available for local coding agents. CBM install
mutates external agent configs, instructions, skills, and hooks. That is a
setup action, not a Python dependency side effect.

## What is known

- Omnigent installer has a clean insertion point after `verify_omnigent`.
- Installer tests already source `install_oss.sh` and can cover new shell
  functions without network.
- `omnigent setup` is an explicit user-facing configuration flow and can host
  check/repair behavior.
- The CBM PyPI package is a wrapper that downloads a GitHub Releases binary on
  first run and then execs it.
- CBM documents `config set auto_index true`.
- CBM `install` writes external agent configs and hooks.
- CBM `install -y` can auto-confirm deletion of existing indexes.
- Omnigent remote sandboxes use a prebaked host image and wheel overlay with
  `pip --no-deps`; local installer work does not affect that image.

## What is uncertain

- Whether CBM will add a non-destructive repair/config-refresh mode.
- Exact minimum CBM version Omnigent should require after command shape is
  finalized.
- Desired UX copy for prompting users about full agent-config repair.

## What changed

The initial plan treated `install -y` as a simple setup command. Local code
inspection showed it can remove existing CBM indexes under auto-yes. The
recommendation now splits safe default install/config from guarded full repair.

## Taxonomy / landscape map

- Package install: installs Python/wrapper artifacts only; should not mutate
  external agent configs.
- Companion install: explicit installer/setup step; allowed to install/check a
  separate tool.
- Agent config repair: writes Codex/Claude/etc configs; must be visible.
- Index state: local CBM cache DBs; should not be deleted silently.
- Remote host image: separate runtime layer for sandboxes.

## Key patterns

- Use `uv tool install` for companion binary separation.
- Verify wrapper/binary immediately with `--version` so download failure appears
  during installer run.
- Treat every CBM step as warning-not-fatal.
- Keep setup repair explicit and testable.

## Anti-patterns

- Vendoring `../codebase-memory-mcp` into Omnigent.
- Adding CBM as an Omnigent runtime dependency and expecting package install to
  mutate agent configs.
- Running full `codebase-memory-mcp install -y` silently in non-interactive
  installer mode.
- Claiming local installer support covers remote sandboxes.

## Opportunity map

- First PR can make most local installs better with low risk.
- `omnigent setup` can become the durable repair/check point.
- CBM `install --plan` can provide transparent planned writes if surfaced.
- Host image work can later give remote agents the same tool.

## Risk map

- Narrow: CBM download fails; Omnigent should still install.
- Narrow: `auto_index` config fails; Omnigent should warn only.
- Moderate: full CBM repair mutates multiple agent configs.
- Moderate: `install -y` deletes/rebuilds existing CBM indexes.
- Moderate: users assume remote sandboxes have CBM when only local installer ran.

## Recommended experiments

- Add installer tests for skip/default and warning-not-fatal failures.
- Add setup helper tests with `shutil.which`/subprocess stubs.
- Manually inspect `install --plan` output on a machine with detected agents.
- Add a follow-up test once a safer CBM repair command exists.

## Decisions / implications

Decision:

- First PR should install/check CBM as a companion tool and set `auto_index=true`.
- First PR should include `--skip-codebase-memory`.
- First PR should not make full `install -y` an unconditional default.
- Remote sandbox support should be separate host image work.

## Next research questions

- Can CBM expose `install --repair-configs --keep-indexes` or equivalent?
- Should Omnigent pin `codebase-memory-mcp>=0.8.1` or wait for a safer command?
- What should the prompt copy say when existing indexes may be rebuilt?

## Evidence notes

Primary evidence is local code in Omnigent and codebase-memory-mcp. Provider
debate was used for critique only. `agy-default` was unavailable due to empty
stdout with exit 0.

## Decision log entry

```text
Date: 2026-06-26
Project: Omnigent
Decision: Integrate codebase-memory-mcp as an explicit companion tool through installer/setup, not as a vendored binary or hidden Python dependency.
Reason: codebase-memory-mcp install mutates external agent configs and its PyPI wrapper downloads a release binary on first run; those side effects belong in visible setup flows. Current install -y can also delete existing indexes, so full repair needs guardrails.
What this prevents: hidden wheel side effects, vendored binary drift, broken remote sandbox assumptions, and silent deletion/rebuild of existing CBM indexes.
Review date: Before first Omnigent implementation PR or after CBM provides a non-destructive repair command.
```
