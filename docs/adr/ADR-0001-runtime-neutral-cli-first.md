# ADR-0001 — Runtime-neutral, CLI-first kernel

Status: accepted
Date: 2026-04-14

## Context

Goalrail must work with different developer runtimes.
The real first integration surface is not vendor APIs.
It is installed and authenticated developer tooling such as:
- Codex CLI
- Claude Code / Cloud Code
- Gemini CLI
- future local or open-source runtimes

We may support API adapters later, but they should not define the kernel.

## Decision

Goalrail adopts a runtime-neutral, CLI-first kernel.

### Rules
- runtime-specific behavior lives behind adapters
- CLI / subscription-backed runtimes are first-class integration targets
- local and open-source runtimes must fit through the same adapter boundary
- raw API adapters are optional later extensions
- init and setup must inspect installed runtimes, auth status, and capabilities rather than assuming API keys

### Minimum kernel concepts
- `ToolRuntimeAdapter`
- `RuntimeCapability`
- `RuntimeBinding`
- `RuntimeRegistry`

### Minimum capability examples
- `execute`
- `review`
- `security_review`
- `performance_review`
- `diff_only`
- `full_repo_context`
- `local_only`

## Consequences

### Positive
- Goalrail avoids vendor lock-in in the kernel
- the product matches real subscription-backed developer workflows
- future local runtimes can join without changing core concepts
- runtime discovery and auth status become inspectable product state

### Negative
- adapter boundaries must be designed carefully from the start
- runtime capabilities will not be fully uniform
- API-first shortcuts should be resisted in MVP design

## Not now

This ADR does not imply:
- a hosted runtime fabric
- API-first orchestration as default
- a marketplace of adapters
- cost or quota management in the kernel
