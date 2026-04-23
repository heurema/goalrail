---
name: goalrail-web-stack
description: >
  Build Goalrail frontend resources on the standard React + Vite + Mantine stack. Use this skill
  when creating or updating any app under `apps/web/*`, when standardizing frontend structure, or
  when wiring Mantine, Vite, PostCSS, and Vitest together for repeatable local delivery.
---

# Goalrail Web Stack

## Purpose

Use one repeatable frontend stack across Goalrail web resources.

Current standard:
- React
- Vite
- Mantine
- PostCSS with `postcss-preset-mantine`
- Vitest + React Testing Library

## Canonical layout

- Each frontend lives under `apps/web/<resource>`
- Do not place a runnable app directly under `apps/web/`
- `apps/web/` is the shared namespace for multiple frontend resources

## Default workflow

1. Put the app in `apps/web/<resource>`
2. Use the existing root npm workspace pattern `apps/web/*`
3. Keep package versions explicit and reproducible
4. Import Mantine styles at the app root
5. Wrap the app in `MantineProvider`
6. Add Vitest setup for `matchMedia`, `ResizeObserver`, `scrollIntoView`, and `document.fonts`
7. Add at least one smoke-level component test before trusting the app scaffold

## Mantine AI tooling

Prefer official Mantine tooling first:
- `mantine-form`
- `mantine-combobox`
- `mantine-custom-components`
- Mantine MCP via `@mantine/mcp-server`

Use the Mantine-specific skills when the task matches them. Do not reinvent their patterns locally.

## Official docs sources

- React: `https://react.dev/llms.txt`
- Vite: `https://vite.dev/llms.txt`
- Mantine: `https://mantine.dev/llms.txt`

React rule:
- use official React docs only
- do not assume an official React MCP or official React skill package exists here

Mantine rule:
- use the project-local Mantine MCP config
- use the vendored official Mantine skills in `.codex/skills/mantine-*`

## References

- [`references/stack-rules.md`](references/stack-rules.md)
- [`references/official-ai-sources.md`](references/official-ai-sources.md)
