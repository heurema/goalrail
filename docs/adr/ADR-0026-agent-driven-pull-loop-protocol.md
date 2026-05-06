# ADR-0026 - Agent-driven pull-loop protocol through Goalrail CLI

Status: accepted
Date: 2026-05-06

## Context

Goalrail must support users working through local coding agents such as Codex,
Cursor, Claude Code, Gemini CLI, and future local runtimes.

These agents can read repo-local instructions and call local tools, but there
is no provider-neutral server push channel that reliably injects a Goalrail
question back into an arbitrary live agent session.

Goalrail also keeps repository access local by default. The API server remains
the canonical state machine, but it must not clone repositories, store
repository secrets, require raw source uploads, or run repository checks
in-process for the MVP.

Current implemented state:

- `goalrail init` binds local Git metadata to server-side Project and
  RepoBinding context and writes `.goalrail/project.yml`.
- `goalrail agent install` installs provider-neutral Agent Pack guidance.
- `goalrail work start` creates an IntakeRecord and promotes it to a Goal.
- Server-side Goal readiness, ClarificationRequest, ClarificationAnswer,
  Contract lifecycle, approval, and WorkItem planning primitives exist.
- Runner, checkout, execution, gate, proof, and provider-specific agent
  adapters are not implemented.

## Decision

Goalrail will use an agent-driven pull-loop protocol:

```text
User -> local agent -> Goalrail CLI -> Goalrail server
                       <- JSON next_action <-
User <- local agent renders result/question
```

Responsibilities:

- Server owns canonical state, state transitions, clarification requests,
  clarification answers, contracts, approvals, planning records, and event log.
- CLI owns local repository detection, local auth/session use, local repository
  receipts, and transport between local agent and server.
- Agent owns conversational UX only: rendering questions, collecting answers,
  reading local files when needed, and calling Goalrail CLI.
- Agent instructions are guidance only, not authority.
- Goalrail server remains the authority for canonical workflow truth.

The server must not try to push questions into arbitrary live Codex, Cursor,
Claude, Gemini, or other agent sessions. Goalrail integrates through local
agent pull: the agent calls CLI commands and renders returned `next_action`
instructions.

## Protocol rules

### Agent Pack bootstrap

The CLI may install a provider-neutral Goalrail Agent Pack:

```text
.goalrail/agent/GOALRAIL.md
.goalrail/agent/commands.json
```

The Agent Pack is bootstrap guidance for local agents. It is not the main
protocol, not a provider adapter, and not an authority over Goalrail state.

Root `AGENTS.md` may be created only as a tiny shim when no root `AGENTS.md`
already exists. Existing provider or agent instruction files must not be
overwritten by default or by `--force`; the CLI should report that a manual
patch is needed instead.

Provider-specific shims for Claude, Cursor, Gemini, Windsurf, Gravity, or other
tools are out of scope for Slice A.

### `next_action`

Agent-facing JSON responses should include a stable protocol envelope:

```json
{
  "schema_version": "goalrail.cli.v1",
  "goal_id": "...",
  "goal_state": "...",
  "display": {
    "summary": "Human-safe summary for the agent to show."
  },
  "next_action": {
    "kind": "...",
    "blocking": true
  }
}
```

Agents must treat `next_action.available=false` as a planned or unavailable
command and must not call `next_action.command` in that case.

### `goalrail work start`

`work start` accepts pasted tracker/plain-text tasks through stdin:

```bash
goalrail work start --title "<title>" --body-file - --format json
```

In Slice A, it creates IntakeRecord and Goal through the existing server
endpoints and returns an agent-facing JSON envelope with a planned Slice B
continuation action. It does not run readiness reconciliation and does not
implement `work continue`.

Target direction after Slice A:

- `work start` returns the first real `next_action` after initial readiness
  reconciliation.
- `work answer` records structured answers and returns the next `next_action`.
- `work continue` is the universal resume/reconcile command.

### Clarification and contracts

Clarification remains server-owned. The server creates ClarificationRequest and
records ClarificationAnswer as canonical state. CLI and agents only transport
questions and answers.

No standalone `work context prepare` command is introduced in the MVP. Future
local code context belongs inside a bounded `contract draft` helper, but no
contract draft CLI is implemented in Slice A.

## Non-goals

This ADR does not implement:

- provider-specific Codex, Claude, Gemini, Cursor, Windsurf, Gravity, or other
  adapters
- server push into agent sessions
- Jira or Linear sync
- local LLM ownership of canonical truth
- server-side repository clone
- raw source upload by default
- standalone `work context prepare`
- `work continue` implementation in Slice A
- `work answer`
- contract draft CLI in Slice A
- runner checkout
- execution
- gate
- proof
- Problem Details migration
- idempotency or optimistic concurrency hardening
- broad queue platform
- generic agent framework

## Consequences

Goalrail can support heterogeneous local agents without owning their runtime or
pretending a universal server push channel exists.

The server remains canonical and auditable. The CLI becomes the local bridge for
repository context, auth, and transport. The agent remains a UX layer.

Slice A prepares the JSON protocol shape without implying unimplemented
continuation commands are executable.

## Rejected alternatives

### Server push into local agents

Rejected. There is no provider-neutral channel that reliably returns a question
into an arbitrary live agent session.

### Agent instructions as enforcement

Rejected. Repo-local instructions are prompt/context guidance, not a canonical
state or enforcement layer.

### Provider-specific shims by default

Rejected for Slice A. The provider-neutral pack must stay canonical and small;
provider-specific files can only be added later through bounded explicit work.
