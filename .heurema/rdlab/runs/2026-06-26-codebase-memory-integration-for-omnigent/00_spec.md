# 00 Spec

## Topic

codebase memory integration for omnigent


## Project context

Omnigent needs codebase-memory-mcp available to local coding agents without
making the Omnigent Python wheel vendor or depend on the codebase-memory-mcp
binary/runtime side effects.

## Decision context

The proposed path is to install codebase-memory-mcp as an explicit companion
tool from the Omnigent installer/setup flow. The decision needs to separate:

- package dependency vs. local developer-machine configuration;
- local installer behavior vs. remote sandbox host-image behavior;
- safe default install steps vs. agent-config/index mutation steps.

## Intended audience

Omnigent maintainers implementing the first integration PR.

## Output type

Decision memo with implementation boundaries, risks, and test plan.

## Scope

- Inspect current Omnigent installer/setup/remote host image surfaces.
- Inspect local codebase-memory-mcp packaging and install/config behavior.
- Run a short proposer/skeptic debate over the integration shape.
- Produce a bounded implementation recommendation.

## Non-scope

- No Omnigent source edits in this run.
- No remote host image rebuild.
- No changes to codebase-memory-mcp upstream behavior.
- No public claims beyond local code evidence.

## Topic registry link

N/A - ad hoc project lab run.

## Source preferences

Prefer local source code and existing docs over external web sources. Use
provider debate only as critique/synthesis, not as primary evidence.

## External source requirement

Internal-only. The decision is about two local repositories and their current
implementation details. External docs are not required because the relevant
package, installer, CLI, and setup behavior is present locally.


## Freshness requirements

Use current checkout state on 2026-06-26. Treat provider debate as advisory and
lower confidence than local code reads.

## Critic / debate requirement

State which independent critic passes should be used. Prefer the project
`providers.toml` router when available. If only one agent/provider is
available, record that limitation explicitly in `09_critic_review.md`.

Run provider router doctor with probes. Use two available routes:

- proposer: `claude-sonnet`
- skeptic: `vibe-default`

Record `agy-default` if it remains unavailable or empty-output.

## Success criteria

- Clear verdict on vendoring/dependency vs. companion install.
- Identify exact Omnigent files/functions for first PR.
- Identify any command in the initial proposal that is unsafe by default.
- Define minimum tests for installer/setup behavior.
- Separate local installer work from remote sandbox image work.

## Constraints

- Do not silently mutate user agent configs from Python package install.
- Omnigent install should remain successful if codebase-memory-mcp install or
  download fails.
- Avoid adding runtime coupling to remote host images accidentally.
- Keep the first PR reviewable and reversible.

## Assumptions

- `uv tool install codebase-memory-mcp` installs the PyPI wrapper console script.
- Calling `codebase-memory-mcp --version` forces the wrapper to fetch/cache the
  platform binary when needed.
- The Omnigent OSS installer may run in interactive and non-interactive modes.

## Open questions

- Should the first PR run `codebase-memory-mcp install -y` by default, given
  that current CBM install can auto-delete existing index DBs when `-y` is used?
- Should codebase-memory-mcp add a non-destructive repair mode before Omnigent
  enables full agent-config repair by default?
- What minimum CBM version should Omnigent require once the exact safe command
  shape is chosen?
