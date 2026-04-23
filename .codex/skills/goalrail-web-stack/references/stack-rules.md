# Goalrail Web Stack Rules

## Scope

This is a tooling and structure rule, not a product-surface claim.
A frontend scaffold or demo prototype does not mean the Goalrail web product loop exists.

## Structure

- Root workspaces use `apps/web/*`
- Shared conventions live in `apps/web/README.md`
- Resource-specific code lives in `apps/web/<resource>`

## Stack

- React for UI
- Vite for build/dev server
- Mantine for UI system
- PostCSS with `postcss-preset-mantine` and `postcss-simple-vars`
- Vitest + Testing Library for the first reliability layer

## Reliability rules

- Pin versions explicitly
- Prefer npm registry packages for Mantine consumption
- Do not file-link `~/contrib/mantine` as the default app dependency source
- Keep one small smoke test in every frontend app
- Keep app naming explicit: `@goalrail/<resource>-web`

## AI assistance rules

- Use official Mantine skills from `.codex/skills/mantine-*`
- Use official Mantine MCP when available in the client
- For React, prefer official `react.dev/llms.txt` because no official MCP/skill package is established here
- For Vite, prefer official `vite.dev/llms.txt` because no official MCP/skill package is established here
- Project-local MCP config lives in `.codex/config.toml` and `.mcp.json`
