# ADR-0026 - Agent Pack boundary

Status: accepted
Date: 2026-05-06

## Context

Goalrail is runtime-neutral and CLI-first. Users should be able to talk to
their local coding agent, while the agent calls the Goalrail CLI as the machine
interface.

This creates pressure to ship useful repo-local agent guidance, but Goalrail
must not become a provider-specific skill, plugin, slash-command system, IDE
adapter, runner, or local agent owner. Server state remains canonical for
Intake, Goal, readiness, clarification, Contract, events, gate, proof, and
verification.

## Decision

Goalrail will provide a provider-neutral Agent Pack v0.

The installed repo-local files are:

- `.goalrail/agent/GOALRAIL.md`
- `.goalrail/agent/commands.json`

The Agent Pack is not a Codex, Claude, Gemini, Cursor, Windsurf, Gravity, or
other provider-specific skill, plugin, slash command, adapter, or setting.

Provider-specific shims are deferred. They may be added later only through an
explicit bounded slice that preserves the provider-neutral core pack.

The Agent Pack may instruct a local coding agent to call Goalrail CLI commands,
preferably with `--format json`. The agent must not invent Goalrail state.
When the user asks to start Goalrail work or pastes task text, the agent should
call:

```bash
goalrail work start --title <title> --body-file - --format json
```

The pasted task body should be passed through stdin.

The server remains the canonical source of truth for Intake, Goal, readiness,
clarification, Contract, events, gate, proof, and verification state.

This slice does not implement:

- provider-specific shims
- runner, checkout, execution, or local LLM ownership
- gate or proof automation
- readiness automation
- Contract creation automation
- Jira, Linear, or tracker sync

`goalrail init` must not install the Agent Pack by default in v0.

Installation is explicit:

```bash
goalrail agent install
```

## Consequences

Goalrail can become agent-friendly without owning the coding agent runtime.

The repo-local pack gives agents a stable machine-readable command map while
keeping server state canonical and keeping provider-specific files out of the
default install path.

Existing initialized repositories opt in explicitly. `goalrail init` remains a
repository binding command, not an agent setup command.

## Rejected alternatives

### Install provider-specific files by default

Rejected. Default Codex, Claude, Gemini, Cursor, Windsurf, Gravity, or other
provider-specific files would make Goalrail appear to own or prefer a provider
runtime.

### Install Agent Pack during `goalrail init`

Rejected for v0. Init binds repository metadata to the server context. Agent
guidance is useful but separate and should be explicitly installed.

### Let agents infer Goalrail state from local files

Rejected. Local files may guide command usage, but the server owns canonical
Goalrail state.
